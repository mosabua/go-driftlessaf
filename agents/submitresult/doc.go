/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package submitresult provides tool definitions for AI agents to submit their
// final results.
//
// It exposes ClaudeTool and GoogleTool constructors that build executor tool
// metadata for the submit_result tool, which agents call to return a structured
// response at the end of a conversation.
package submitresult
