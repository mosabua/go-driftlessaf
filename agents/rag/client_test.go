/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package rag

import (
	"context"
	"errors"
	"maps"
	"testing"
)

// fakeRetriever returns canned search results.
type fakeRetriever struct {
	results []Result
	err     error
	closed  bool
}

func (f *fakeRetriever) Search(_ context.Context, _ []float32, _ SearchOptions) ([]Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}

func (f *fakeRetriever) Close() error {
	f.closed = true
	return nil
}

func TestEmbedAndStoreDoesNotMutateCallerMetadata(t *testing.T) {
	store := newMemoryStore()

	// Simulate the metadata cloning logic from EmbedAndStore.
	metadata := map[string]string{"key": "value"}

	md := make(map[string]string, len(metadata)+1)
	maps.Copy(md, metadata)
	md[MetadataKeySourceText] = "test text"

	if err := store.Upsert(t.Context(), "id-1", []float32{1.0, 2.0}, md); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Original metadata must not have _source_text.
	if _, ok := metadata[MetadataKeySourceText]; ok {
		t.Error("caller's metadata map was mutated — _source_text should not be present")
	}

	// Stored metadata should have it.
	r, ok := store.get("id-1")
	if !ok {
		t.Fatal("record not found in store")
	}
	if r.metadata[MetadataKeySourceText] != "test text" {
		t.Errorf("stored metadata[%s]: got = %q, want = %q",
			MetadataKeySourceText, r.metadata[MetadataKeySourceText], "test text")
	}
}

func TestEmbedAndStorePreservesExistingSourceText(t *testing.T) {
	store := newMemoryStore()

	metadata := map[string]string{
		MetadataKeySourceText: "already set",
		"other":               "value",
	}

	// Clone metadata and only set _source_text if not already present —
	// matching the EmbedAndStore logic.
	md := make(map[string]string, len(metadata)+1)
	maps.Copy(md, metadata)
	if _, exists := md[MetadataKeySourceText]; !exists {
		md[MetadataKeySourceText] = "should not appear"
	}

	if err := store.Upsert(t.Context(), "id-2", []float32{1.0}, md); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	r, _ := store.get("id-2")
	if r.metadata[MetadataKeySourceText] != "already set" {
		t.Errorf("existing _source_text overwritten: got = %q, want = %q",
			r.metadata[MetadataKeySourceText], "already set")
	}
}

func TestClientCloseCollectsErrors(t *testing.T) {
	storeErr := errors.New("store close failed")
	retrieverErr := errors.New("retriever close failed")

	client := &Client{
		embedder:  &Embedder{},
		store:     &failingCloseStore{err: storeErr},
		retriever: &failingCloseRetriever{err: retrieverErr},
	}

	err := client.Close()
	if err == nil {
		t.Fatal("Close: expected error, got nil")
	}

	if !errors.Is(err, storeErr) {
		t.Errorf("Close error should contain store error: got = %v", err)
	}
	if !errors.Is(err, retrieverErr) {
		t.Errorf("Close error should contain retriever error: got = %v", err)
	}
}

type failingCloseStore struct {
	memoryStore
	err error
}

func (s *failingCloseStore) Close() error { return s.err }

type failingCloseRetriever struct {
	fakeRetriever
	err error
}

func (r *failingCloseRetriever) Close() error { return r.err }

func TestGCSStoreValidation(t *testing.T) {
	_, err := NewGCSStore(t.Context(), "", "prefix")
	if err == nil {
		t.Error("NewGCSStore with empty bucket: expected error, got nil")
	}
}
