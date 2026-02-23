/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package worktreefs

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"chainguard.dev/driftlessaf/agents/toolcall/callbacks"
)

// epoch is the modification time reported for all files.
var epoch = time.Unix(0, 0)

// batchSize is the page size used when collecting all directory entries.
const batchSize = 500

// Compile-time interface assertions.
var (
	_ fs.FS         = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
)

// FS implements fs.FS backed by WorktreeCallbacks.
type FS struct {
	ctx context.Context
	cb  callbacks.WorktreeCallbacks
}

// New creates an FS backed by the given worktree callbacks. The context is
// captured and used for all subsequent callback invocations.
func New(ctx context.Context, cb callbacks.WorktreeCallbacks) *FS {
	return &FS{ctx: ctx, cb: cb}
}

// Open opens the named file or directory.
func (f *FS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	info, err := f.stat(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	if info.IsDir() {
		return &dirFile{info: info, path: name, cb: f.cb, ctx: f.ctx}, nil
	}
	result, err := f.cb.ReadFile(f.ctx, name, 0, -1)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return &regFile{
		info:   info,
		reader: bytes.NewReader([]byte(result.Content)),
	}, nil
}

// Stat returns file info for the named file or directory.
func (f *FS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrInvalid}
	}
	info, err := f.stat(name)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}
	return info, nil
}

// ReadFile reads the named file and returns its contents.
func (f *FS) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrInvalid}
	}
	result, err := f.cb.ReadFile(f.ctx, name, 0, -1)
	if err != nil {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: err}
	}
	return []byte(result.Content), nil
}

// ReadDir reads the named directory and returns its entries sorted by name.
func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	entries, err := f.listAll(name)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

// stat returns file info for the named path. For ".", it synthesizes root
// directory info. For all other paths, it lists the parent directory with
// a filter to find the specific entry.
func (f *FS) stat(name string) (*fileInfo, error) {
	if name == "." {
		return &fileInfo{name: ".", mode: fs.ModeDir | 0o755}, nil
	}
	result, err := f.cb.ListDirectory(f.ctx, path.Dir(name), path.Base(name), 0, 1)
	if err != nil {
		return nil, err
	}
	if len(result.Entries) == 0 {
		return nil, fs.ErrNotExist
	}
	return newFileInfo(result.Entries[0]), nil
}

// listAll collects all directory entries by paging through ListDirectory.
func (f *FS) listAll(name string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	offset := 0
	for {
		result, err := f.cb.ListDirectory(f.ctx, name, "", offset, batchSize)
		if err != nil {
			return nil, err
		}
		for _, e := range result.Entries {
			entries = append(entries, newDirEntry(e))
		}
		if result.NextOffset == nil {
			return entries, nil
		}
		offset = *result.NextOffset
	}
}

// newFileInfo converts a callbacks.DirEntry to a fileInfo with the correct
// mode type bits set.
func newFileInfo(e callbacks.DirEntry) *fileInfo {
	mode := e.Mode
	switch e.Type {
	case "directory":
		mode |= fs.ModeDir
	case "symlink":
		mode |= fs.ModeSymlink
	}
	return &fileInfo{name: e.Name, size: e.Size, mode: mode}
}

// newDirEntry converts a callbacks.DirEntry to an fs.DirEntry.
func newDirEntry(e callbacks.DirEntry) *dirEntry {
	return &dirEntry{info: newFileInfo(e)}
}

// regFile is an fs.File for regular files, backed by in-memory content.
type regFile struct {
	info   *fileInfo
	reader *bytes.Reader
}

func (f *regFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *regFile) Read(b []byte) (int, error) { return f.reader.Read(b) }
func (f *regFile) Close() error               { return nil }

// dirFile is an fs.ReadDirFile for directories, paging through ListDirectory.
type dirFile struct {
	info   *fileInfo
	path   string
	cb     callbacks.WorktreeCallbacks
	ctx    context.Context
	offset int
}

func (f *dirFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *dirFile) Close() error               { return nil }

func (f *dirFile) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: f.path, Err: fs.ErrInvalid}
}

func (f *dirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if n <= 0 {
		return f.readAll()
	}
	return f.readN(n)
}

func (f *dirFile) readAll() ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	for {
		result, err := f.cb.ListDirectory(f.ctx, f.path, "", f.offset, batchSize)
		if err != nil {
			return entries, err
		}
		for _, e := range result.Entries {
			entries = append(entries, newDirEntry(e))
		}
		if result.NextOffset != nil {
			f.offset = *result.NextOffset
		} else {
			f.offset += len(result.Entries)
			return entries, nil
		}
	}
}

func (f *dirFile) readN(n int) ([]fs.DirEntry, error) {
	result, err := f.cb.ListDirectory(f.ctx, f.path, "", f.offset, n)
	if err != nil {
		return nil, err
	}
	if len(result.Entries) == 0 {
		return nil, io.EOF
	}
	entries := make([]fs.DirEntry, len(result.Entries))
	for i, e := range result.Entries {
		entries[i] = newDirEntry(e)
	}
	if result.NextOffset != nil {
		f.offset = *result.NextOffset
	} else {
		f.offset += len(entries)
	}
	return entries, nil
}

// fileInfo implements fs.FileInfo with a fixed epoch ModTime.
type fileInfo struct {
	name string
	size int64
	mode fs.FileMode
}

func (fi *fileInfo) Name() string       { return fi.name }
func (fi *fileInfo) Size() int64        { return fi.size }
func (fi *fileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *fileInfo) ModTime() time.Time { return epoch }
func (fi *fileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi *fileInfo) Sys() any           { return nil }

// dirEntry implements fs.DirEntry backed by a fileInfo.
type dirEntry struct {
	info *fileInfo
}

func (de *dirEntry) Name() string               { return de.info.name }
func (de *dirEntry) IsDir() bool                { return de.info.IsDir() }
func (de *dirEntry) Type() fs.FileMode          { return de.info.mode.Type() }
func (de *dirEntry) Info() (fs.FileInfo, error) { return de.info, nil }
