/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"

	"chainguard.dev/driftlessaf/agents/metaagent"
	"chainguard.dev/driftlessaf/agents/promptbuilder"
	"chainguard.dev/driftlessaf/agents/toolcall"
	"chainguard.dev/driftlessaf/agents/toolcall/callbacks"
)

// Request contains the modernize diagnostics to be applied by the agent.
type Request struct {
	// Findings lists the diagnostics or CI findings that need to be addressed.
	// On a fresh run these come from the Go modernize analyzer.
	// On iteration they come from CI check failures.
	Findings []callbacks.Finding `json:"findings" xml:"findings"`
}

// Bind implements promptbuilder.Bindable for Request.
func (r *Request) Bind(prompt *promptbuilder.Prompt) (*promptbuilder.Prompt, error) {
	return prompt.BindXML("request", r)
}

// Result contains the outcome of the modernization process.
type Result struct {
	// Summary describes what was modernized
	Summary string `json:"summary"`

	// CommitMessage is the message for the git commit
	CommitMessage string `json:"commit_message"`
}

// GetCommitMessage implements metapathreconciler.Result.
func (r *Result) GetCommitMessage() string {
	return r.CommitMessage
}

// systemInstructions contains the shared system instructions for the agent.
var systemInstructions = promptbuilder.MustNewPrompt(`ROLE: Go modernizer

TASK: You are an AI agent that applies Go modernize fixes to a codebase. Given a set of diagnostics from the Go modernize analysis suite, you read the relevant files, understand the context, and apply the suggested fixes.

MODES:
- FRESH: If the findings come from the modernize analyzer, apply each fix methodically.
- ITERATION: If the findings come from CI failures, the codebase already contains a previous attempt. Fix the issues while preserving the working parts. Do NOT start over from scratch.

CORE PRINCIPLES:
1. UNDERSTAND FIRST: Read the file around each diagnostic to understand the context before making changes
2. MINIMAL CHANGES: Only modify what is necessary to apply the modernize fix - don't refactor or "improve" unrelated code
3. MATCH STYLE: Follow existing code patterns, naming conventions, and formatting
4. COMPLETE: Address every diagnostic in the findings list
5. SAFE: Ensure changes preserve the existing behavior

COMMON MODERNIZE FIXES:
- Replace for-append loops with slices.Collect or slices.AppendSeq
- Replace sort.Slice with slices.SortFunc
- Replace manual min/max with built-in min/max
- Replace if/else chains with switch statements
- Replace loop-based string building with strings.Join
- Replace manual contains checks with slices.Contains

WORKFLOW (FRESH - analyzer diagnostics):
1. Get details for each finding to understand what needs to change
2. Read the affected files to understand the surrounding code
3. Apply the modernize fixes one at a time
4. Ensure imports are updated (add new imports, remove unused ones)

WORKFLOW (ITERATION - CI failures):
1. Get details for each finding to understand what went wrong
2. Read the files that were modified in the previous attempt
3. Make targeted fixes to address the specific failures
4. Verify changes don't introduce new issues

COMMIT MESSAGE FORMAT:
- Title: "fix(modernize): apply modernize fixes to <package>"
- Body: List the specific fixes applied

OUTPUT FORMAT:
When you have completed your changes, you MUST call the submit_result tool with your final result.
DO NOT return JSON as text output - use the submit_result tool instead.

DO NOT:
- Make changes unrelated to the diagnostics
- Add unnecessary comments or documentation
- Refactor or "improve" existing code beyond the modernize fixes
- Change code formatting/style unless required by the fix`)

// userPrompt is the prompt template for the modernizer.
var userPrompt = promptbuilder.MustNewPrompt(`{{request}}

Use the available tools to read and modify the codebase.

Start by getting details for each finding to understand what needs to change. Then read the relevant files and apply the fixes.

IMPORTANT: When you have finished applying all fixes, you MUST call submit_result with your summary and commit message. Do NOT output JSON as text.`)

// newAgent creates a new modernizer agent.
// The model parameter determines which provider implementation is used:
//   - Models starting with "gemini-" use Google's Generative AI SDK
//   - Models starting with "claude-" use Anthropic's SDK via Vertex AI
func newAgent[CB any](ctx context.Context, projectID, region, model string, tools toolcall.ToolProvider[*Result, CB]) (metaagent.Agent[*Request, *Result, CB], error) {
	return metaagent.New[*Request](ctx, projectID, region, model, metaagent.Config[*Result, CB]{
		SystemInstructions: systemInstructions,
		UserPrompt:         userPrompt,
		Tools:              tools,
	})
}
