/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package changemanager

import (
	"fmt"
	"strings"
)

// formatCheckRunDetails builds a human-readable details string for a check run.
func formatCheckRunDetails(name, status, conclusion, title, summary, text, detailsURL string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Check Run: %s\n", name))
	sb.WriteString(fmt.Sprintf("Status: %s\n", status))
	sb.WriteString(fmt.Sprintf("Conclusion: %s\n", conclusion))
	if title != "" {
		sb.WriteString(fmt.Sprintf("Title: %s\n", title))
	}
	if summary != "" {
		sb.WriteString(fmt.Sprintf("Summary: %s\n", summary))
	}
	if text != "" {
		sb.WriteString(fmt.Sprintf("Details:\n%s\n", text))
	}
	if detailsURL != "" {
		sb.WriteString(fmt.Sprintf("Details URL: %s\n", detailsURL))
	}
	return sb.String()
}

// formatThreadDetails builds a human-readable details string for a review thread.
// Includes commit SHA and outdated status so the agent can contextualize via history tools.
func formatThreadDetails(path string, line int, isOutdated bool, comments []gqlThreadComment) string {
	var sb strings.Builder

	first := comments[0]

	sb.WriteString(fmt.Sprintf("Review thread by @%s (%s)\n", first.Author.Login, first.AuthorAssociation))
	sb.WriteString(fmt.Sprintf("Path: %s:%d\n", path, line))

	commitAnnotation := first.Commit.Oid
	if isOutdated {
		commitAnnotation += " (outdated)"
	}
	sb.WriteString(fmt.Sprintf("Commit: %s\n", commitAnnotation))

	for _, c := range comments {
		sb.WriteString(fmt.Sprintf("\n[Comment by @%s]\n%s\n", c.Author.Login, c.Body))
	}

	return sb.String()
}

// formatReviewBodyDetails builds a human-readable details string for a review body.
func formatReviewBodyDetails(review gqlReviewBodyNode) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Review by @%s (%s) - %s\n", review.Author.Login, review.AuthorAssociation, review.State))
	sb.WriteString(fmt.Sprintf("Submitted: %s\n", review.SubmittedAt))
	sb.WriteString(fmt.Sprintf("Commit: %s\n", review.Commit.Oid))
	sb.WriteString(fmt.Sprintf("\n%s\n", review.Body))

	return sb.String()
}
