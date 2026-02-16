/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package toolcall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"chainguard.dev/driftlessaf/agents/agenttrace"
	"chainguard.dev/driftlessaf/agents/toolcall/callbacks"
	"chainguard.dev/driftlessaf/agents/toolcall/claudetool"
	"chainguard.dev/driftlessaf/agents/toolcall/googletool"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/chainguard-dev/clog"
	"google.golang.org/genai"
)

// WorktreeTools wraps a base tools type and adds worktree callbacks.
type WorktreeTools[T any] struct {
	base T
	callbacks.WorktreeCallbacks
}

// NewWorktreeTools creates a WorktreeTools wrapping the given base tools.
func NewWorktreeTools[T any](base T, cb callbacks.WorktreeCallbacks) WorktreeTools[T] {
	return WorktreeTools[T]{base: base, WorktreeCallbacks: cb}
}

// worktreeToolsProvider wraps a base ToolProvider and adds worktree tools.
type worktreeToolsProvider[Resp, T any] struct {
	baseProvider ToolProvider[Resp, T]
}

var _ ToolProvider[any, WorktreeTools[any]] = (*worktreeToolsProvider[any, any])(nil)

// NewWorktreeToolsProvider creates a provider that adds worktree tools
// on top of the base provider's tools.
func NewWorktreeToolsProvider[Resp, T any](base ToolProvider[Resp, T]) ToolProvider[Resp, WorktreeTools[T]] {
	return worktreeToolsProvider[Resp, T]{baseProvider: base}
}

func (p worktreeToolsProvider[Resp, T]) ClaudeTools(cb WorktreeTools[T]) map[string]claudetool.Metadata[Resp] {
	tools := p.baseProvider.ClaudeTools(cb.base)

	tools["read_file"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "read_file",
			Description: anthropic.String("Read content from a file starting at a byte offset. Returns the content, next_offset to continue reading, and remaining bytes. Use list_directory to check file size before reading large files."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain why you are reading this file.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "The path to the file to read (relative to repository root)",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Byte offset to start reading from (default: 0)",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum bytes to read (default: 20000). Pass -1 to read the entire file, but avoid this if you don't know the file size as it may be very large.",
					},
				},
				Required: []string{"reasoning", "path"},
			},
		},
		Handler: claudeReadFileHandler[Resp](cb.ReadFile),
	}

	tools["edit_file"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "edit_file",
			Description: anthropic.String("Edit a file by replacing exact text. The old_string must appear exactly once in the file unless replace_all is true. Use this instead of write_file for modifying existing files to avoid sending the entire file through context."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain why you are making this edit.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "The path to the file to edit (relative to repository root)",
					},
					"old_string": map[string]any{
						"type":        "string",
						"description": "The exact text to find and replace. Maximum 32KB; use write_file for larger replacements.",
					},
					"new_string": map[string]any{
						"type":        "string",
						"description": "The replacement text. Pass an empty string to delete the matched text. Maximum 32KB; use write_file for larger replacements.",
					},
					"replace_all": map[string]any{
						"type":        "boolean",
						"description": "Replace all occurrences instead of requiring uniqueness (default: false)",
					},
				},
				Required: []string{"reasoning", "path", "old_string", "new_string"},
			},
		},
		Handler: claudeEditFileHandler[Resp](cb.EditFile),
	}

	tools["write_file"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "write_file",
			Description: anthropic.String("Create or overwrite a file. For targeted modifications to existing files, prefer edit_file to avoid sending the entire file through context."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain why you are writing this file.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "The path to the file to write (relative to repository root)",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "The complete content to write to the file",
					},
					"executable": map[string]any{
						"type":        "boolean",
						"description": "Whether the file should be executable (default: false)",
					},
				},
				Required: []string{"reasoning", "path", "content"},
			},
		},
		Handler: claudeWriteFileHandler[Resp](cb.WriteFile),
	}

	tools["delete_file"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "delete_file",
			Description: anthropic.String("Delete a file from the codebase."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain why you are deleting this file.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "The path to the file to delete (relative to repository root)",
					},
				},
				Required: []string{"reasoning", "path"},
			},
		},
		Handler: claudeDeleteFileHandler[Resp](cb.DeleteFile),
	}

	tools["move_file"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "move_file",
			Description: anthropic.String("Move or rename a file. No file content flows through context."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain why you are moving this file.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "The source path (relative to repository root)",
					},
					"destination": map[string]any{
						"type":        "string",
						"description": "The destination path (relative to repository root)",
					},
				},
				Required: []string{"reasoning", "path", "destination"},
			},
		},
		Handler: claudeMoveFileHandler[Resp](cb.MoveFile),
	}

	tools["copy_file"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "copy_file",
			Description: anthropic.String("Copy a file. No file content flows through context."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain why you are copying this file.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "The source path (relative to repository root)",
					},
					"destination": map[string]any{
						"type":        "string",
						"description": "The destination path (relative to repository root)",
					},
				},
				Required: []string{"reasoning", "path", "destination"},
			},
		},
		Handler: claudeCopyFileHandler[Resp](cb.CopyFile),
	}

	tools["chmod"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "chmod",
			Description: anthropic.String("Change file permissions. Mode is an octal string like \"0755\" or \"0644\"."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain why you are changing permissions.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "The path to the file (relative to repository root)",
					},
					"mode": map[string]any{
						"type":        "string",
						"description": "The file mode as an octal string (e.g., \"0755\", \"0644\")",
					},
				},
				Required: []string{"reasoning", "path", "mode"},
			},
		},
		Handler: claudeChmodHandler[Resp](cb.Chmod),
	}

	tools["symlink"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "symlink",
			Description: anthropic.String("Create a symbolic link. The target should be a relative path."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain why you are creating this symlink.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Where to create the symlink (relative to repository root)",
					},
					"target": map[string]any{
						"type":        "string",
						"description": "What the symlink points to (relative path)",
					},
				},
				Required: []string{"reasoning", "path", "target"},
			},
		},
		Handler: claudeSymlinkHandler[Resp](cb.CreateSymlink),
	}

	tools["list_directory"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "list_directory",
			Description: anthropic.String("List directory contents with ls -l style metadata (name, size, mode, type). Supports glob filtering with * wildcards or exact filename matching. Results are paginated."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain why you are listing this directory.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "The path to the directory to list (relative to repository root, use '.' for root)",
					},
					"filter": map[string]any{
						"type":        "string",
						"description": "Filter entries by glob pattern (e.g., \"*.go\") or exact filename (e.g., \"main.go\"). Only * wildcards are supported.",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Number of entries to skip (default: 0)",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum entries to return (default: 50)",
					},
				},
				Required: []string{"reasoning", "path"},
			},
		},
		Handler: claudeListDirectoryHandler[Resp](cb.ListDirectory),
	}

	tools["search_codebase"] = claudetool.Metadata[Resp]{
		Definition: anthropic.ToolParam{
			Name:        "search_codebase",
			Description: anthropic.String("Search for a regex pattern across files. Returns compact match pointers (path, byte offset, length) without content. Use read_file with the returned offset to view matches in context, padding the offset and limit as needed for surrounding context."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
				Properties: map[string]any{
					"reasoning": map[string]any{
						"type":        "string",
						"description": "Explain what you are searching for and why.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Directory to search within (relative to repository root, default: \".\")",
					},
					"pattern": map[string]any{
						"type":        "string",
						"description": "The regex pattern to search for",
					},
					"filter": map[string]any{
						"type":        "string",
						"description": "File filter — glob with * wildcards (e.g., \"*.go\") or exact filename (e.g., \"Makefile\")",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Number of matches to skip (default: 0)",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum matches to return (default: 50)",
					},
				},
				Required: []string{"reasoning", "pattern"},
			},
		},
		Handler: claudeSearchCodebaseHandler[Resp](cb.SearchCodebase),
	}

	return tools
}

func (p worktreeToolsProvider[Resp, T]) GoogleTools(cb WorktreeTools[T]) map[string]googletool.Metadata[Resp] {
	tools := p.baseProvider.GoogleTools(cb.base)

	tools["read_file"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "read_file",
			Description: "Read content from a file starting at a byte offset. Returns the content, next_offset to continue reading, and remaining bytes. Use list_directory to check file size before reading large files.",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning": {Type: "string", Description: "Explain why you are reading this file."},
					"path":      {Type: "string", Description: "The path to the file to read (relative to repository root)"},
					"offset":    {Type: "integer", Description: "Byte offset to start reading from (default: 0)"},
					"limit":     {Type: "integer", Description: "Maximum bytes to read (default: 20000). Pass -1 to read the entire file, but avoid this if you don't know the file size as it may be very large."},
				},
				Required: []string{"reasoning", "path"},
			},
		},
		Handler: googleReadFileHandler[Resp](cb.ReadFile),
	}

	tools["edit_file"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "edit_file",
			Description: "Edit a file by replacing exact text. The old_string must appear exactly once in the file unless replace_all is true. Use this instead of write_file for modifying existing files to avoid sending the entire file through context.",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning":   {Type: "string", Description: "Explain why you are making this edit."},
					"path":        {Type: "string", Description: "The path to the file to edit (relative to repository root)"},
					"old_string":  {Type: "string", Description: "The exact text to find and replace. Maximum 32KB; use write_file for larger replacements."},
					"new_string":  {Type: "string", Description: "The replacement text. Pass an empty string to delete the matched text. Maximum 32KB; use write_file for larger replacements."},
					"replace_all": {Type: "boolean", Description: "Replace all occurrences instead of requiring uniqueness (default: false)"},
				},
				Required: []string{"reasoning", "path", "old_string", "new_string"},
			},
		},
		Handler: googleEditFileHandler[Resp](cb.EditFile),
	}

	tools["write_file"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "write_file",
			Description: "Create or overwrite a file. For targeted modifications to existing files, prefer edit_file to avoid sending the entire file through context.",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning":  {Type: "string", Description: "Explain why you are writing this file."},
					"path":       {Type: "string", Description: "The path to the file to write (relative to repository root)"},
					"content":    {Type: "string", Description: "The complete content to write to the file"},
					"executable": {Type: "boolean", Description: "Whether the file should be executable (default: false)"},
				},
				Required: []string{"reasoning", "path", "content"},
			},
		},
		Handler: googleWriteFileHandler[Resp](cb.WriteFile),
	}

	tools["delete_file"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "delete_file",
			Description: "Delete a file from the codebase.",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning": {Type: "string", Description: "Explain why you are deleting this file."},
					"path":      {Type: "string", Description: "The path to the file to delete (relative to repository root)"},
				},
				Required: []string{"reasoning", "path"},
			},
		},
		Handler: googleDeleteFileHandler[Resp](cb.DeleteFile),
	}

	tools["move_file"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "move_file",
			Description: "Move or rename a file. No file content flows through context.",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning":   {Type: "string", Description: "Explain why you are moving this file."},
					"path":        {Type: "string", Description: "The source path (relative to repository root)"},
					"destination": {Type: "string", Description: "The destination path (relative to repository root)"},
				},
				Required: []string{"reasoning", "path", "destination"},
			},
		},
		Handler: googleMoveFileHandler[Resp](cb.MoveFile),
	}

	tools["copy_file"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "copy_file",
			Description: "Copy a file. No file content flows through context.",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning":   {Type: "string", Description: "Explain why you are copying this file."},
					"path":        {Type: "string", Description: "The source path (relative to repository root)"},
					"destination": {Type: "string", Description: "The destination path (relative to repository root)"},
				},
				Required: []string{"reasoning", "path", "destination"},
			},
		},
		Handler: googleCopyFileHandler[Resp](cb.CopyFile),
	}

	tools["chmod"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "chmod",
			Description: "Change file permissions. Mode is an octal string like \"0755\" or \"0644\".",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning": {Type: "string", Description: "Explain why you are changing permissions."},
					"path":      {Type: "string", Description: "The path to the file (relative to repository root)"},
					"mode":      {Type: "string", Description: "The file mode as an octal string (e.g., \"0755\", \"0644\")"},
				},
				Required: []string{"reasoning", "path", "mode"},
			},
		},
		Handler: googleChmodHandler[Resp](cb.Chmod),
	}

	tools["symlink"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "symlink",
			Description: "Create a symbolic link. The target should be a relative path.",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning": {Type: "string", Description: "Explain why you are creating this symlink."},
					"path":      {Type: "string", Description: "Where to create the symlink (relative to repository root)"},
					"target":    {Type: "string", Description: "What the symlink points to (relative path)"},
				},
				Required: []string{"reasoning", "path", "target"},
			},
		},
		Handler: googleSymlinkHandler[Resp](cb.CreateSymlink),
	}

	tools["list_directory"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "list_directory",
			Description: "List directory contents with ls -l style metadata (name, size, mode, type). Supports glob filtering with * wildcards or exact filename matching. Results are paginated.",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning": {Type: "string", Description: "Explain why you are listing this directory."},
					"path":      {Type: "string", Description: "The path to the directory to list (relative to repository root, use '.' for root)"},
					"filter":    {Type: "string", Description: "Filter entries by glob pattern (e.g., \"*.go\") or exact filename (e.g., \"main.go\"). Only * wildcards are supported."},
					"offset":    {Type: "integer", Description: "Number of entries to skip (default: 0)"},
					"limit":     {Type: "integer", Description: "Maximum entries to return (default: 50)"},
				},
				Required: []string{"reasoning", "path"},
			},
		},
		Handler: googleListDirectoryHandler[Resp](cb.ListDirectory),
	}

	tools["search_codebase"] = googletool.Metadata[Resp]{
		Definition: &genai.FunctionDeclaration{
			Name:        "search_codebase",
			Description: "Search for a regex pattern across files. Returns compact match pointers (path, byte offset, length) without content. Use read_file with the returned offset to view matches in context, padding the offset and limit as needed for surrounding context.",
			Parameters: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"reasoning": {Type: "string", Description: "Explain what you are searching for and why."},
					"path":      {Type: "string", Description: "Directory to search within (relative to repository root, default: \".\")"},
					"pattern":   {Type: "string", Description: "The regex pattern to search for"},
					"filter":    {Type: "string", Description: "File filter — glob with * wildcards (e.g., \"*.go\") or exact filename (e.g., \"Makefile\")"},
					"offset":    {Type: "integer", Description: "Number of matches to skip (default: 0)"},
					"limit":     {Type: "integer", Description: "Maximum matches to return (default: 50)"},
				},
				Required: []string{"reasoning", "pattern"},
			},
		},
		Handler: googleSearchCodebaseHandler[Resp](cb.SearchCodebase),
	}

	return tools
}

// Claude handlers

func claudeReadFileHandler[Resp any](readFile func(context.Context, string, int64, int) (callbacks.ReadResult, error)) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := claudetool.Param[string](params, "path")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing path parameter"))
			return errResp
		}

		offset, _ := claudetool.OptionalParam[int64](params, "offset", 0)
		limit, _ := claudetool.OptionalParam[int](params, "limit", 20000)

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"path": path, "offset": offset, "limit": limit})

		result, err := readFile(ctx, path, offset, limit)
		if err != nil {
			log.With("path", path).With("error", err).Error("Failed to read file")
			resp := claudetool.ErrorWithContext(err, map[string]any{"path": path})
			tc.Complete(resp, err)
			return resp
		}

		resp := map[string]any{
			"path":    path,
			"content": result.Content,
		}
		if result.NextOffset != nil {
			resp["next_offset"] = *result.NextOffset
		}
		resp["remaining"] = result.Remaining
		tc.Complete(resp, nil)
		return resp
	}
}

func claudeEditFileHandler[Resp any](
	editFile func(context.Context, string, string, string, bool) (callbacks.EditResult, error),
) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := claudetool.Param[string](params, "path")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing path parameter"))
			return errResp
		}

		oldString, errResp := claudetool.Param[string](params, "old_string")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing old_string parameter"))
			return errResp
		}

		newString, errResp := claudetool.Param[string](params, "new_string")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing new_string parameter"))
			return errResp
		}

		replaceAll, _ := claudetool.OptionalParam[bool](params, "replace_all", false)

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"path": path, "replace_all": replaceAll})

		result, err := editFile(ctx, path, oldString, newString, replaceAll)
		if err != nil {
			log.With("path", path).With("error", err).Error("Failed to edit file")
			errResp := claudetool.ErrorWithContext(err, map[string]any{"path": path})
			tc.Complete(errResp, err)
			return errResp
		}

		resp := map[string]any{"path": path, "replacements": result.Replacements}
		tc.Complete(resp, nil)
		return resp
	}
}

func claudeWriteFileHandler[Resp any](writeFile func(context.Context, string, string, os.FileMode) error) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := claudetool.Param[string](params, "path")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing path parameter"))
			return errResp
		}

		content, errResp := claudetool.Param[string](params, "content")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing content parameter"))
			return errResp
		}

		executable, _ := claudetool.OptionalParam[bool](params, "executable", false)

		mode := os.FileMode(0644)
		if executable {
			mode = 0755
		}

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"path": path, "size": len(content), "executable": executable})

		if err := writeFile(ctx, path, content, mode); err != nil {
			log.With("path", path).With("error", err).Error("Failed to write file")
			result := claudetool.ErrorWithContext(err, map[string]any{"path": path})
			tc.Complete(result, err)
			return result
		}

		result := map[string]any{"path": path, "written": len(content), "mode": fmt.Sprintf("%04o", mode)}
		tc.Complete(result, nil)
		return result
	}
}

func claudeDeleteFileHandler[Resp any](deleteFile func(context.Context, string) error) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := claudetool.Param[string](params, "path")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing path parameter"))
			return errResp
		}

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"path": path})

		if err := deleteFile(ctx, path); err != nil {
			log.With("path", path).With("error", err).Error("Failed to delete file")
			result := claudetool.ErrorWithContext(err, map[string]any{"path": path})
			tc.Complete(result, err)
			return result
		}

		result := map[string]any{"path": path, "deleted": true}
		tc.Complete(result, nil)
		return result
	}
}

func claudeMoveFileHandler[Resp any](moveFile func(context.Context, string, string) error) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := claudetool.Param[string](params, "path")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing path parameter"))
			return errResp
		}

		destination, errResp := claudetool.Param[string](params, "destination")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing destination parameter"))
			return errResp
		}

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"source": path, "destination": destination})

		if err := moveFile(ctx, path, destination); err != nil {
			log.With("source", path).With("destination", destination).With("error", err).Error("Failed to move file")
			result := claudetool.ErrorWithContext(err, map[string]any{"source": path, "destination": destination})
			tc.Complete(result, err)
			return result
		}

		result := map[string]any{"source": path, "destination": destination}
		tc.Complete(result, nil)
		return result
	}
}

func claudeCopyFileHandler[Resp any](copyFile func(context.Context, string, string) error) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := claudetool.Param[string](params, "path")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing path parameter"))
			return errResp
		}

		destination, errResp := claudetool.Param[string](params, "destination")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing destination parameter"))
			return errResp
		}

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"source": path, "destination": destination})

		if err := copyFile(ctx, path, destination); err != nil {
			log.With("source", path).With("destination", destination).With("error", err).Error("Failed to copy file")
			result := claudetool.ErrorWithContext(err, map[string]any{"source": path, "destination": destination})
			tc.Complete(result, err)
			return result
		}

		result := map[string]any{"source": path, "destination": destination}
		tc.Complete(result, nil)
		return result
	}
}

func claudeChmodHandler[Resp any](chmod func(context.Context, string, os.FileMode) error) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := claudetool.Param[string](params, "path")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing path parameter"))
			return errResp
		}

		modeStr, errResp := claudetool.Param[string](params, "mode")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing mode parameter"))
			return errResp
		}

		mode, err := parseOctalMode(modeStr)
		if err != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), err)
			return claudetool.Error("invalid mode %q: %v", modeStr, err)
		}

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"path": path, "mode": modeStr})

		if err := chmod(ctx, path, mode); err != nil {
			log.With("path", path).With("error", err).Error("Failed to chmod")
			result := claudetool.ErrorWithContext(err, map[string]any{"path": path})
			tc.Complete(result, err)
			return result
		}

		result := map[string]any{"path": path, "mode": fmt.Sprintf("%04o", mode)}
		tc.Complete(result, nil)
		return result
	}
}

func claudeSymlinkHandler[Resp any](createSymlink func(context.Context, string, string) error) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := claudetool.Param[string](params, "path")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing path parameter"))
			return errResp
		}

		target, errResp := claudetool.Param[string](params, "target")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing target parameter"))
			return errResp
		}

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"path": path, "target": target})

		if err := createSymlink(ctx, path, target); err != nil {
			log.With("path", path).With("target", target).With("error", err).Error("Failed to create symlink")
			result := claudetool.ErrorWithContext(err, map[string]any{"path": path, "target": target})
			tc.Complete(result, err)
			return result
		}

		result := map[string]any{"path": path, "target": target}
		tc.Complete(result, nil)
		return result
	}
}

func claudeListDirectoryHandler[Resp any](listDirectory func(context.Context, string, string, int, int) (callbacks.ListResult, error)) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := claudetool.Param[string](params, "path")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing path parameter"))
			return errResp
		}

		filter, _ := claudetool.OptionalParam[string](params, "filter", "")
		offset, _ := claudetool.OptionalParam[int](params, "offset", 0)
		limit, _ := claudetool.OptionalParam[int](params, "limit", 50)

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"path": path, "filter": filter, "offset": offset, "limit": limit})

		result, err := listDirectory(ctx, path, filter, offset, limit)
		if err != nil {
			log.With("path", path).With("error", err).Error("Failed to list directory")
			resp := claudetool.ErrorWithContext(err, map[string]any{"path": path})
			tc.Complete(resp, err)
			return resp
		}

		resp := formatListResult(path, result)
		tc.Complete(resp, nil)
		return resp
	}
}

func claudeSearchCodebaseHandler[Resp any](searchCodebase func(context.Context, string, string, string, int, int) (callbacks.SearchResult, error)) func(context.Context, anthropic.ToolUseBlock, *agenttrace.Trace[Resp], *Resp) map[string]any {
	return func(ctx context.Context, toolUse anthropic.ToolUseBlock, trace *agenttrace.Trace[Resp], _ *Resp) map[string]any {
		log := clog.FromContext(ctx)

		params, errResp := claudetool.NewParams(toolUse)
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, map[string]any{"input": toolUse.Input}, errors.New("failed to parse params"))
			return errResp
		}

		reasoning, errResp := claudetool.Param[string](params, "reasoning")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		pattern, errResp := claudetool.Param[string](params, "pattern")
		if errResp != nil {
			trace.BadToolCall(toolUse.ID, toolUse.Name, params.RawInputs(), errors.New("missing pattern parameter"))
			return errResp
		}

		searchPath, _ := claudetool.OptionalParam[string](params, "path", ".")
		filter, _ := claudetool.OptionalParam[string](params, "filter", "")
		offset, _ := claudetool.OptionalParam[int](params, "offset", 0)
		limit, _ := claudetool.OptionalParam[int](params, "limit", 50)

		tc := trace.StartToolCall(toolUse.ID, toolUse.Name, map[string]any{"path": searchPath, "pattern": pattern, "filter": filter, "offset": offset, "limit": limit})

		result, err := searchCodebase(ctx, searchPath, pattern, filter, offset, limit)
		if err != nil {
			log.With("pattern", pattern).With("error", err).Error("Failed to search codebase")
			resp := claudetool.ErrorWithContext(err, map[string]any{"pattern": pattern})
			tc.Complete(resp, err)
			return resp
		}

		resp := formatSearchResult(searchPath, pattern, filter, result)
		tc.Complete(resp, nil)
		return resp
	}
}

// Google handlers

func googleReadFileHandler[Resp any](readFile func(context.Context, string, int64, int) (callbacks.ReadResult, error)) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := googletool.Param[string](call, "path")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing path parameter"))
			return errResp
		}

		offset, _ := googletool.OptionalParam[int64](call, "offset", 0)
		limit, _ := googletool.OptionalParam[int](call, "limit", 20000)

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"path": path, "offset": offset, "limit": limit})

		result, err := readFile(ctx, path, offset, limit)
		if err != nil {
			log.With("path", path).With("error", err).Error("Failed to read file")
			resp := googletool.ErrorWithContext(call, err, map[string]any{"path": path})
			tc.Complete(resp.Response, err)
			return resp
		}

		resp := map[string]any{"path": path, "content": result.Content}
		if result.NextOffset != nil {
			resp["next_offset"] = *result.NextOffset
		}
		resp["remaining"] = result.Remaining
		tc.Complete(resp, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: resp}
	}
}

func googleEditFileHandler[Resp any](
	editFile func(context.Context, string, string, string, bool) (callbacks.EditResult, error),
) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := googletool.Param[string](call, "path")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing path parameter"))
			return errResp
		}

		oldString, errResp := googletool.Param[string](call, "old_string")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing old_string parameter"))
			return errResp
		}

		newString, errResp := googletool.Param[string](call, "new_string")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing new_string parameter"))
			return errResp
		}

		replaceAll, _ := googletool.OptionalParam(call, "replace_all", false)

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"path": path, "replace_all": replaceAll})

		result, err := editFile(ctx, path, oldString, newString, replaceAll)
		if err != nil {
			log.With("path", path).With("error", err).Error("Failed to edit file")
			errResult := googletool.ErrorWithContext(call, err, map[string]any{"path": path})
			tc.Complete(errResult.Response, err)
			return errResult
		}

		resp := map[string]any{"path": path, "replacements": result.Replacements}
		tc.Complete(resp, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: resp}
	}
}

func googleWriteFileHandler[Resp any](writeFile func(context.Context, string, string, os.FileMode) error) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := googletool.Param[string](call, "path")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing path parameter"))
			return errResp
		}

		content, errResp := googletool.Param[string](call, "content")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing content parameter"))
			return errResp
		}

		executable, _ := googletool.OptionalParam(call, "executable", false)

		mode := os.FileMode(0644)
		if executable {
			mode = 0755
		}

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"path": path, "size": len(content), "executable": executable})

		if err := writeFile(ctx, path, content, mode); err != nil {
			log.With("path", path).With("error", err).Error("Failed to write file")
			result := googletool.ErrorWithContext(call, err, map[string]any{"path": path})
			tc.Complete(result.Response, err)
			return result
		}

		result := map[string]any{"path": path, "written": len(content), "mode": fmt.Sprintf("%04o", mode)}
		tc.Complete(result, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: result}
	}
}

func googleDeleteFileHandler[Resp any](deleteFile func(context.Context, string) error) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := googletool.Param[string](call, "path")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing path parameter"))
			return errResp
		}

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"path": path})

		if err := deleteFile(ctx, path); err != nil {
			log.With("path", path).With("error", err).Error("Failed to delete file")
			result := googletool.ErrorWithContext(call, err, map[string]any{"path": path})
			tc.Complete(result.Response, err)
			return result
		}

		result := map[string]any{"path": path, "deleted": true}
		tc.Complete(result, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: result}
	}
}

func googleMoveFileHandler[Resp any](moveFile func(context.Context, string, string) error) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := googletool.Param[string](call, "path")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing path parameter"))
			return errResp
		}

		destination, errResp := googletool.Param[string](call, "destination")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing destination parameter"))
			return errResp
		}

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"source": path, "destination": destination})

		if err := moveFile(ctx, path, destination); err != nil {
			log.With("source", path).With("destination", destination).With("error", err).Error("Failed to move file")
			result := googletool.ErrorWithContext(call, err, map[string]any{"source": path, "destination": destination})
			tc.Complete(result.Response, err)
			return result
		}

		result := map[string]any{"source": path, "destination": destination}
		tc.Complete(result, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: result}
	}
}

func googleCopyFileHandler[Resp any](copyFile func(context.Context, string, string) error) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := googletool.Param[string](call, "path")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing path parameter"))
			return errResp
		}

		destination, errResp := googletool.Param[string](call, "destination")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing destination parameter"))
			return errResp
		}

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"source": path, "destination": destination})

		if err := copyFile(ctx, path, destination); err != nil {
			log.With("source", path).With("destination", destination).With("error", err).Error("Failed to copy file")
			result := googletool.ErrorWithContext(call, err, map[string]any{"source": path, "destination": destination})
			tc.Complete(result.Response, err)
			return result
		}

		result := map[string]any{"source": path, "destination": destination}
		tc.Complete(result, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: result}
	}
}

func googleChmodHandler[Resp any](chmod func(context.Context, string, os.FileMode) error) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := googletool.Param[string](call, "path")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing path parameter"))
			return errResp
		}

		modeStr, errResp := googletool.Param[string](call, "mode")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing mode parameter"))
			return errResp
		}

		mode, err := parseOctalMode(modeStr)
		if err != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, err)
			return googletool.Error(call, "invalid mode %q: %v", modeStr, err)
		}

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"path": path, "mode": modeStr})

		if err := chmod(ctx, path, mode); err != nil {
			log.With("path", path).With("error", err).Error("Failed to chmod")
			result := googletool.ErrorWithContext(call, err, map[string]any{"path": path})
			tc.Complete(result.Response, err)
			return result
		}

		result := map[string]any{"path": path, "mode": fmt.Sprintf("%04o", mode)}
		tc.Complete(result, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: result}
	}
}

func googleSymlinkHandler[Resp any](createSymlink func(context.Context, string, string) error) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := googletool.Param[string](call, "path")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing path parameter"))
			return errResp
		}

		target, errResp := googletool.Param[string](call, "target")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing target parameter"))
			return errResp
		}

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"path": path, "target": target})

		if err := createSymlink(ctx, path, target); err != nil {
			log.With("path", path).With("target", target).With("error", err).Error("Failed to create symlink")
			result := googletool.ErrorWithContext(call, err, map[string]any{"path": path, "target": target})
			tc.Complete(result.Response, err)
			return result
		}

		result := map[string]any{"path": path, "target": target}
		tc.Complete(result, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: result}
	}
}

func googleListDirectoryHandler[Resp any](listDirectory func(context.Context, string, string, int, int) (callbacks.ListResult, error)) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		path, errResp := googletool.Param[string](call, "path")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing path parameter"))
			return errResp
		}

		filter, _ := googletool.OptionalParam[string](call, "filter", "")
		offset, _ := googletool.OptionalParam[int](call, "offset", 0)
		limit, _ := googletool.OptionalParam[int](call, "limit", 50)

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"path": path, "filter": filter, "offset": offset, "limit": limit})

		result, err := listDirectory(ctx, path, filter, offset, limit)
		if err != nil {
			log.With("path", path).With("error", err).Error("Failed to list directory")
			resp := googletool.ErrorWithContext(call, err, map[string]any{"path": path})
			tc.Complete(resp.Response, err)
			return resp
		}

		resp := formatListResult(path, result)
		tc.Complete(resp, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: resp}
	}
}

func googleSearchCodebaseHandler[Resp any](searchCodebase func(context.Context, string, string, string, int, int) (callbacks.SearchResult, error)) func(context.Context, *genai.FunctionCall, *agenttrace.Trace[Resp], *Resp) *genai.FunctionResponse {
	return func(ctx context.Context, call *genai.FunctionCall, trace *agenttrace.Trace[Resp], _ *Resp) *genai.FunctionResponse {
		log := clog.FromContext(ctx)

		reasoning, errResp := googletool.Param[string](call, "reasoning")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing reasoning parameter"))
			return errResp
		}
		log.With("reasoning", reasoning).Info("Tool call reasoning")

		pattern, errResp := googletool.Param[string](call, "pattern")
		if errResp != nil {
			trace.BadToolCall(call.ID, call.Name, call.Args, errors.New("missing pattern parameter"))
			return errResp
		}

		searchPath, _ := googletool.OptionalParam[string](call, "path", ".")
		filter, _ := googletool.OptionalParam[string](call, "filter", "")
		offset, _ := googletool.OptionalParam[int](call, "offset", 0)
		limit, _ := googletool.OptionalParam[int](call, "limit", 50)

		tc := trace.StartToolCall(call.ID, call.Name, map[string]any{"path": searchPath, "pattern": pattern, "filter": filter, "offset": offset, "limit": limit})

		result, err := searchCodebase(ctx, searchPath, pattern, filter, offset, limit)
		if err != nil {
			log.With("pattern", pattern).With("error", err).Error("Failed to search codebase")
			resp := googletool.ErrorWithContext(call, err, map[string]any{"pattern": pattern})
			tc.Complete(resp.Response, err)
			return resp
		}

		resp := formatSearchResult(searchPath, pattern, filter, result)
		tc.Complete(resp, nil)
		return &genai.FunctionResponse{ID: call.ID, Name: call.Name, Response: resp}
	}
}

// Shared helpers

// formatListResult converts a ListResult into the JSON response map.
func formatListResult(path string, result callbacks.ListResult) map[string]any {
	entries := make([]map[string]any, 0, len(result.Entries))
	for _, e := range result.Entries {
		entry := map[string]any{
			"name": e.Name,
			"size": e.Size,
			"mode": fmt.Sprintf("%04o", e.Mode.Perm()),
			"type": e.Type,
		}
		if e.Target != "" {
			entry["target"] = e.Target
		}
		entries = append(entries, entry)
	}

	resp := map[string]any{
		"path":    path,
		"entries": entries,
		"count":   len(entries),
	}
	if result.NextOffset != nil {
		resp["next_offset"] = *result.NextOffset
	}
	resp["remaining"] = result.Remaining
	return resp
}

// formatSearchResult converts a SearchResult into the JSON response map.
func formatSearchResult(searchPath, pattern, filter string, result callbacks.SearchResult) map[string]any {
	matches := make([]map[string]any, 0, len(result.Matches))
	for _, m := range result.Matches {
		matches = append(matches, map[string]any{
			"path":   m.Path,
			"offset": m.Offset,
			"length": m.Length,
		})
	}

	resp := map[string]any{
		"path":    searchPath,
		"pattern": pattern,
		"matches": matches,
		"count":   len(matches),
	}
	if filter != "" {
		resp["filter"] = filter
	}
	if result.NextOffset != nil {
		resp["next_offset"] = *result.NextOffset
	}
	resp["has_more"] = result.HasMore
	return resp
}

// parseOctalMode parses an octal mode string like "0755" into os.FileMode.
func parseOctalMode(s string) (os.FileMode, error) {
	parsed, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("parse octal mode: %w", err)
	}
	return os.FileMode(parsed), nil
}
