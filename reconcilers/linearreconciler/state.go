/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package linearreconciler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chainguard-dev/clog"
)

// stateAttachmentTitle returns the attachment title used for state storage.
func (c *Client) stateAttachmentTitle() string {
	return c.statePrefix + "_state"
}

// LoadState fetches the state attachment from the issue, deserializes it into v.
// Returns true if state was found and loaded, false if no state attachment exists.
func (c *Client) LoadState(ctx context.Context, issue *Issue, v any) (bool, error) {
	att := issue.FindAttachment(c.stateAttachmentTitle())
	if att == nil || att.URL == "" {
		return false, nil
	}

	data, err := c.FetchAttachmentContent(ctx, att.URL)
	if err != nil {
		return false, fmt.Errorf("fetching state attachment: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return false, fmt.Errorf("unmarshaling state: %w", err)
	}

	clog.FromContext(ctx).Infof("Loaded state from attachment %q", c.stateAttachmentTitle())
	return true, nil
}

// SaveState serializes v as JSON and uploads it as a file attachment on the issue.
// If a state attachment already exists, it is replaced.
func (c *Client) SaveState(ctx context.Context, issue *Issue, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	existingID := ""
	if att := issue.FindAttachment(c.stateAttachmentTitle()); att != nil {
		existingID = att.ID
	}

	if err := c.UploadFileAttachment(ctx, issue.ID, c.stateAttachmentTitle(), data, existingID); err != nil {
		return fmt.Errorf("uploading state attachment: %w", err)
	}

	clog.FromContext(ctx).Infof("Saved state to attachment %q", c.stateAttachmentTitle())
	return nil
}
