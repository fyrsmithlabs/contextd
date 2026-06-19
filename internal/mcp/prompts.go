package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts registers the static contextd MCP prompts. Each prompt
// returns an instruction template (no service calls) that directs the agent to
// invoke the appropriate contextd MCP tools. The prompts mirror the bundled
// slash commands under plugins/contextd/commands/.
func (s *Server) registerPrompts() {
	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "contextd_checkpoint",
		Title:       "Save a resumable context checkpoint",
		Description: "Save a resumable context checkpoint of this session via checkpoint_save.",
		Arguments: []*mcp.PromptArgument{
			{Name: "summary", Description: "Optional summary text to use for the checkpoint.", Required: false},
		},
	}, s.handleCheckpointPrompt)

	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "contextd_remember",
		Title:       "Record a learning into contextd memory",
		Description: "Record a durable learning from this session into the contextd ReasoningBank via memory_record.",
		Arguments: []*mcp.PromptArgument{
			{Name: "content", Description: "Optional explicit content to remember.", Required: false},
		},
	}, s.handleRememberPrompt)

	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "contextd_diagnose",
		Title:       "Diagnose an error and find a known fix",
		Description: "Diagnose an error via troubleshoot_diagnose and search for a known fix via remediation_search.",
		Arguments: []*mcp.PromptArgument{
			{Name: "error", Description: "The error message or description to diagnose.", Required: true},
		},
	}, s.handleDiagnosePrompt)

	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "contextd_resume",
		Title:       "Resume from a contextd checkpoint",
		Description: "List contextd checkpoints and resume from one via checkpoint_resume.",
		Arguments: []*mcp.PromptArgument{
			{Name: "checkpoint_id", Description: "Optional checkpoint id to resume directly.", Required: false},
		},
	}, s.handleResumePrompt)

	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "contextd_status",
		Title:       "Show contextd memories, checkpoints, and project context",
		Description: "Report the current contextd state for this session.",
	}, s.handleStatusPrompt)

	s.mcp.AddPrompt(&mcp.Prompt{
		Name:        "contextd_search",
		Title:       "Search contextd memories, remediations, and code",
		Description: "Search contextd memories, remediations, and code for a query.",
		Arguments: []*mcp.PromptArgument{
			{Name: "query", Description: "The search query.", Required: true},
		},
	}, s.handleSearchPrompt)
}

// promptMessage builds a single user-role prompt message wrapping text.
func promptMessage(text string) *mcp.PromptMessage {
	return &mcp.PromptMessage{
		Role:    "user",
		Content: &mcp.TextContent{Text: text},
	}
}

func promptArg(args map[string]string, key string) string {
	if args == nil {
		return ""
	}
	return args[key]
}

func (s *Server) handleCheckpointPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	summary := promptArg(req.Params.Arguments, "summary")

	text := "Save a resumable checkpoint of this session using the contextd `checkpoint_save` MCP tool.\n\n"
	if summary != "" {
		text += fmt.Sprintf("Use this summary text for the checkpoint: %q\n\n", summary)
	} else {
		text += "Build a resumable summary from the recent conversation covering:\n" +
			"- What was done: the concrete state reached so far.\n" +
			"- What's next: the immediate next step(s).\n" +
			"- Open questions / blockers: anything unresolved.\n\n"
	}
	text += "Then call `checkpoint_save` with that summary and report the returned checkpoint id with a one-line confirmation of what was saved."

	return &mcp.GetPromptResult{
		Description: "Save a resumable context checkpoint via checkpoint_save.",
		Messages:    []*mcp.PromptMessage{promptMessage(text)},
	}, nil
}

func (s *Server) handleRememberPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	content := promptArg(req.Params.Arguments, "content")

	text := "Record a durable learning into the contextd ReasoningBank using the `memory_record` MCP tool.\n\n"
	if content != "" {
		text += fmt.Sprintf("Record this content: %q\n\n", content)
	} else {
		text += "Distill the key insight from the recent conversation.\n\n"
	}
	text += "Capture the WHY, not just the what: the approach that worked, rejected alternatives, the deciding tradeoff, and any consequences or gotchas. " +
		"Call `memory_record` with the distilled content and confirm what was stored in one or two lines. " +
		"Never record secrets or credentials, and skip recording insights that are already obvious from the code or docs."

	return &mcp.GetPromptResult{
		Description: "Record a learning into contextd memory via memory_record.",
		Messages:    []*mcp.PromptMessage{promptMessage(text)},
	}, nil
}

func (s *Server) handleDiagnosePrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	errMsg := promptArg(req.Params.Arguments, "error")

	text := "Diagnose an error using contextd.\n\n"
	if errMsg != "" {
		text += fmt.Sprintf("The error to diagnose is: %q\n\n", errMsg)
	} else {
		text += "Use the most recent error in the conversation.\n\n"
	}
	text += "Steps:\n" +
		"1. Call `troubleshoot_diagnose` with the error to get the likely cause (category + analysis).\n" +
		"2. Call `remediation_search` with the stable part of the error signature to find any fix that worked before.\n" +
		"3. Present the diagnosis, any matching known remediation clearly marked as a prior fix, and a recommended next step.\n" +
		"4. After the user applies a fix and confirms it works, offer to record it with `remediation_record` so the fix is reused next time."

	return &mcp.GetPromptResult{
		Description: "Diagnose an error via troubleshoot_diagnose and remediation_search.",
		Messages:    []*mcp.PromptMessage{promptMessage(text)},
	}, nil
}

func (s *Server) handleResumePrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	checkpointID := promptArg(req.Params.Arguments, "checkpoint_id")

	text := "Resume work from a previously saved contextd checkpoint.\n\n"
	if checkpointID != "" {
		text += fmt.Sprintf("Call `checkpoint_resume` with checkpoint id %q.\n\n", checkpointID)
	} else {
		text += "Call `checkpoint_list` and show the available checkpoints (id, summary, timestamp), then ask the user which one to resume before calling `checkpoint_resume` with the chosen id.\n\n"
	}
	text += "Default to the `context` resume level unless the user asks for `summary` (quick reorientation) or `full` (deep resumption after a long gap). " +
		"Summarize the restored state and state the immediate next step so work can continue."

	return &mcp.GetPromptResult{
		Description: "Resume from a contextd checkpoint via checkpoint_resume.",
		Messages:    []*mcp.PromptMessage{promptMessage(text)},
	}, nil
}

func (s *Server) handleStatusPrompt(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	text := "Report the current contextd state for this session.\n\n" +
		"Steps:\n" +
		"1. Call `checkpoint_list` to get available checkpoints (count + most recent).\n" +
		"2. Run a broad `memory_search` for the current project to gauge how many relevant memories exist.\n" +
		"3. Summarize in a compact status block:\n" +
		"   - The tenant / project context contextd is operating under (auto-derived from the repository).\n" +
		"   - The number of checkpoints and the most recent one.\n" +
		"   - Whether relevant memories exist for this project.\n" +
		"If the contextd MCP server is unavailable, say so and suggest checking that `contextd --mcp` is running."

	return &mcp.GetPromptResult{
		Description: "Show contextd memories, checkpoints, and project context.",
		Messages:    []*mcp.PromptMessage{promptMessage(text)},
	}, nil
}

func (s *Server) handleSearchPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	query := promptArg(req.Params.Arguments, "query")

	text := "Search contextd for anything relevant to the query.\n\n"
	if query != "" {
		text += fmt.Sprintf("The query is: %q\n\n", query)
	}
	text += "Run these searches for the query:\n" +
		"- `memory_search` — past strategies and decisions.\n" +
		"- `remediation_search` — known error fixes.\n" +
		"- `semantic_search` (with `project_path: \".\"`) — relevant code in this repository.\n\n" +
		"Merge and present the most relevant hits, grouped by source (Memories / Remediations / Code), each with a one-line relevance note. " +
		"If nothing relevant is found, say so plainly rather than padding with weak matches."

	return &mcp.GetPromptResult{
		Description: "Search contextd memories, remediations, and code.",
		Messages:    []*mcp.PromptMessage{promptMessage(text)},
	}, nil
}
