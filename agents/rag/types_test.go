/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package rag

import (
	"testing"
)

func TestSearchOptionsDefaults(t *testing.T) {
	tests := []struct {
		name  string
		input SearchOptions
		want  SearchOptions
	}{
		{
			name:  "zero values: TopK defaults, no threshold filtering",
			input: SearchOptions{},
			want:  SearchOptions{TopK: DefaultTopK, DistanceThreshold: 0},
		},
		{
			name:  "explicit values preserved",
			input: SearchOptions{TopK: 10, DistanceThreshold: 0.5},
			want:  SearchOptions{TopK: 10, DistanceThreshold: 0.5},
		},
		{
			name:  "negative TopK gets default",
			input: SearchOptions{TopK: -1, DistanceThreshold: 0.5},
			want:  SearchOptions{TopK: DefaultTopK, DistanceThreshold: 0.5},
		},
		{
			name:  "zero DistanceThreshold means no filtering",
			input: SearchOptions{TopK: 3, DistanceThreshold: 0},
			want:  SearchOptions{TopK: 3, DistanceThreshold: 0},
		},
		{
			name:  "explicit threshold preserved",
			input: SearchOptions{TopK: 5, DistanceThreshold: 0.3},
			want:  SearchOptions{TopK: 5, DistanceThreshold: 0.3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.defaults()
			if got.TopK != tt.want.TopK {
				t.Errorf("TopK: got = %d, want = %d", got.TopK, tt.want.TopK)
			}
			if got.DistanceThreshold != tt.want.DistanceThreshold {
				t.Errorf("DistanceThreshold: got = %f, want = %f", got.DistanceThreshold, tt.want.DistanceThreshold)
			}
		})
	}
}

func TestSearchOptionsDefaultsDoesNotMutateOriginal(t *testing.T) {
	original := SearchOptions{}
	_ = original.defaults()

	if original.TopK != 0 {
		t.Errorf("original.TopK mutated: got = %d, want = 0", original.TopK)
	}
	if original.DistanceThreshold != 0 {
		t.Errorf("original.DistanceThreshold mutated: got = %f, want = 0", original.DistanceThreshold)
	}
}
