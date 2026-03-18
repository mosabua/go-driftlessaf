/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package linearreconciler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"
)

const linearGraphQLEndpoint = "https://api.linear.app/graphql"

// RateLimitError is returned when the Linear API returns HTTP 429.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("linear rate limited, retry after %v", e.RetryAfter)
}

// Client is a Linear API client that uses GraphQL.
type Client struct {
	token       string
	httpClient  *http.Client
	statePrefix string

	// BotUserID is the authenticated user's ID, resolved during reconciler construction.
	BotUserID string
}

// NewClient creates a new Linear API client with the given API token.
func NewClient(token string) *Client {
	return &Client{
		token:       token,
		httpClient:  http.DefaultClient,
		statePrefix: "reconciler",
	}
}

func (c *Client) graphql(ctx context.Context, query string, variables map[string]any, result any) error {
	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return fmt.Errorf("marshaling query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, linearGraphQLEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := time.Minute
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if seconds, err := strconv.Atoi(ra); err == nil {
				retryAfter = time.Duration(seconds) * time.Second
			}
		}
		return &RateLimitError{RetryAfter: retryAfter}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, respBody)
	}

	var gqlResp struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	return json.Unmarshal(gqlResp.Data, result)
}

// GetViewer returns the authenticated user.
func (c *Client) GetViewer(ctx context.Context) (*User, error) {
	var result struct {
		Viewer User `json:"viewer"`
	}
	err := c.graphql(ctx, `query { viewer { id name } }`, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result.Viewer, nil
}

// GetIssue fetches an issue by ID, including comments, labels, and attachments.
func (c *Client) GetIssue(ctx context.Context, issueID string) (*Issue, error) {
	const query = `query($id: String!) {
		issue(id: $id) {
			id identifier title description updatedAt
			state { name type }
			team { key name }
			assignee { id name }
			labels { nodes { name } }
			attachments { nodes { id title subtitle url } }
			comments(first: 100, orderBy: createdAt) {
				nodes {
					id body createdAt
					user { id name }
				}
			}
		}
	}`

	var result struct {
		Issue *Issue `json:"issue"`
	}
	err := c.graphql(ctx, query, map[string]any{"id": issueID}, &result)
	if err != nil {
		return nil, err
	}
	if result.Issue == nil {
		return nil, fmt.Errorf("issue %s not found", issueID)
	}

	sort.Slice(result.Issue.Comments.Nodes, func(i, j int) bool {
		return result.Issue.Comments.Nodes[i].CreatedAt.Before(result.Issue.Comments.Nodes[j].CreatedAt)
	})

	return result.Issue, nil
}

// CreateComment posts a comment on an issue.
func (c *Client) CreateComment(ctx context.Context, issueID, body string) error {
	const mutation = `mutation($issueId: String!, $body: String!) {
		commentCreate(input: { issueId: $issueId, body: $body }) {
			success
		}
	}`

	var result struct {
		CommentCreate struct {
			Success bool `json:"success"`
		} `json:"commentCreate"`
	}
	err := c.graphql(ctx, mutation, map[string]any{
		"issueId": issueID,
		"body":    body,
	}, &result)
	if err != nil {
		return err
	}
	if !result.CommentCreate.Success {
		return fmt.Errorf("comment creation failed")
	}
	return nil
}

// UpdateIssueDescription updates the issue's description.
func (c *Client) UpdateIssueDescription(ctx context.Context, issueID, description string) error {
	const mutation = `mutation($id: String!, $description: String!) {
		issueUpdate(id: $id, input: { description: $description }) {
			success
		}
	}`

	var result struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}
	return c.graphql(ctx, mutation, map[string]any{
		"id":          issueID,
		"description": description,
	}, &result)
}

// UploadFileAttachment uploads content as a file attachment on an issue.
// If existingID is non-empty, the old attachment is deleted first (best-effort).
func (c *Client) UploadFileAttachment(ctx context.Context, issueID, title string, content []byte, existingID string) error {
	// Delete old attachment if updating.
	if existingID != "" {
		const deleteMutation = `mutation($id: String!) {
			attachmentDelete(id: $id) { success }
		}`
		var deleteResult struct {
			AttachmentDelete struct {
				Success bool `json:"success"`
			} `json:"attachmentDelete"`
		}
		_ = c.graphql(ctx, deleteMutation, map[string]any{"id": existingID}, &deleteResult)
	}

	// Step 1: Request a presigned upload URL.
	const uploadMutation = `mutation($contentType: String!, $filename: String!, $size: Int!) {
		fileUpload(contentType: $contentType, filename: $filename, size: $size) {
			uploadFile {
				uploadUrl
				assetUrl
				headers { key value }
			}
		}
	}`
	var uploadResult struct {
		FileUpload struct {
			UploadFile struct {
				UploadURL string `json:"uploadUrl"`
				AssetURL  string `json:"assetUrl"`
				Headers   []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"headers"`
			} `json:"uploadFile"`
		} `json:"fileUpload"`
	}
	if err := c.graphql(ctx, uploadMutation, map[string]any{
		"contentType": "application/json",
		"filename":    title + ".json",
		"size":        len(content),
	}, &uploadResult); err != nil {
		return fmt.Errorf("requesting upload URL: %w", err)
	}

	// Step 2: PUT the content to the presigned URL.
	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadResult.FileUpload.UploadFile.UploadURL, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("creating PUT request: %w", err)
	}
	putReq.Header.Set("Content-Type", "application/json")
	for _, h := range uploadResult.FileUpload.UploadFile.Headers {
		putReq.Header.Set(h.Key, h.Value)
	}
	putResp, err := c.httpClient.Do(putReq)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(putResp.Body)
		return fmt.Errorf("upload failed with status %d: %s", putResp.StatusCode, respBody)
	}

	// Step 3: Create an attachment linking to the uploaded file.
	const attachMutation = `mutation($issueId: String!, $title: String!, $url: String!) {
		attachmentCreate(input: { issueId: $issueId, title: $title, url: $url }) {
			success
		}
	}`
	var attachResult struct {
		AttachmentCreate struct {
			Success bool `json:"success"`
		} `json:"attachmentCreate"`
	}
	if err := c.graphql(ctx, attachMutation, map[string]any{
		"issueId": issueID,
		"title":   title,
		"url":     uploadResult.FileUpload.UploadFile.AssetURL,
	}, &attachResult); err != nil {
		return fmt.Errorf("creating attachment: %w", err)
	}

	return nil
}

// FetchAttachmentContent downloads the content of a file attachment by URL.
func (c *Client) FetchAttachmentContent(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching attachment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
