/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package linearreconciler

import (
	"testing"
	"time"
)

func TestIssue_HasLabel(t *testing.T) {
	issue := &Issue{}
	issue.Labels.Nodes = []struct {
		Name string `json:"name"`
	}{
		{Name: "game"},
		{Name: "Bug"},
	}

	if !issue.HasLabel("game") {
		t.Error("expected HasLabel(game) = true")
	}
	if !issue.HasLabel("GAME") {
		t.Error("expected HasLabel(GAME) = true (case-insensitive)")
	}
	if !issue.HasLabel("bug") {
		t.Error("expected HasLabel(bug) = true (case-insensitive)")
	}
	if issue.HasLabel("feature") {
		t.Error("expected HasLabel(feature) = false")
	}
}

func TestIssue_FindAttachment(t *testing.T) {
	issue := &Issue{}
	issue.Attachments.Nodes = []Attachment{
		{ID: "a1", Title: "game_state", URL: "https://example.com/state.json"},
		{ID: "a2", Title: "other", URL: "https://example.com/other.json"},
	}

	att := issue.FindAttachment("game_state")
	if att == nil {
		t.Fatal("expected to find game_state attachment")
	}
	if att.ID != "a1" {
		t.Errorf("got ID %s, want a1", att.ID)
	}

	if issue.FindAttachment("missing") != nil {
		t.Error("expected FindAttachment(missing) = nil")
	}
}

func TestIssue_UnprocessedComments(t *testing.T) {
	botID := "bot-123"

	tests := []struct {
		name     string
		comments []Comment
		wantLen  int
	}{
		{
			name:     "no comments",
			comments: nil,
			wantLen:  0,
		},
		{
			name: "all player comments",
			comments: []Comment{
				{ID: "c1", Body: "hello", User: User{ID: "player-1"}},
				{ID: "c2", Body: "world", User: User{ID: "player-2"}},
			},
			wantLen: 2,
		},
		{
			name: "bot responded last",
			comments: []Comment{
				{ID: "c1", Body: "hello", User: User{ID: "player-1"}, CreatedAt: time.Now().Add(-2 * time.Minute)},
				{ID: "c2", Body: "response", User: User{ID: botID}, CreatedAt: time.Now().Add(-1 * time.Minute)},
			},
			wantLen: 0,
		},
		{
			name: "player after bot",
			comments: []Comment{
				{ID: "c1", Body: "hello", User: User{ID: "player-1"}, CreatedAt: time.Now().Add(-3 * time.Minute)},
				{ID: "c2", Body: "response", User: User{ID: botID}, CreatedAt: time.Now().Add(-2 * time.Minute)},
				{ID: "c3", Body: "next move", User: User{ID: "player-1"}, CreatedAt: time.Now().Add(-1 * time.Minute)},
			},
			wantLen: 1,
		},
		{
			name: "multiple players after bot",
			comments: []Comment{
				{ID: "c1", Body: "response", User: User{ID: botID}, CreatedAt: time.Now().Add(-3 * time.Minute)},
				{ID: "c2", Body: "move 1", User: User{ID: "player-1"}, CreatedAt: time.Now().Add(-2 * time.Minute)},
				{ID: "c3", Body: "move 2", User: User{ID: "player-2"}, CreatedAt: time.Now().Add(-1 * time.Minute)},
			},
			wantLen: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issue := &Issue{}
			issue.Comments.Nodes = tc.comments

			got := issue.UnprocessedComments(botID)
			if len(got) != tc.wantLen {
				t.Errorf("UnprocessedComments() returned %d comments, want %d", len(got), tc.wantLen)
			}

			hasUnprocessed := issue.HasUnprocessedComments(botID)
			wantHas := tc.wantLen > 0
			if hasUnprocessed != wantHas {
				t.Errorf("HasUnprocessedComments() = %v, want %v", hasUnprocessed, wantHas)
			}
		})
	}
}
