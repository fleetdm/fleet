package ai_agent_sessions

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var copilotEvents = lines(
	`{"type":"session.start","data":{"sessionId":"24b9ca9f","copilotVersion":"1.0.104","startTime":"2026-07-06T02:36:32.043Z","selectedModel":"auto","context":{"cwd":"/Users/alice/GitHub/fleet","branch":"dogfood","repository":"https://x-access-token:ghp_FAKE@github.com/fleetdm/fleet.git"}},"id":"e1","timestamp":"2026-07-06T02:36:32.058Z"}`,
	`{"type":"system.message","data":{"role":"system","content":"You are the GitHub Copilot CLI"},"timestamp":"2026-07-06T02:36:32.100Z"}`,
	`{"type":"user.message","data":{"content":"summarize this diff"},"timestamp":"2026-07-06T02:36:32.200Z"}`,
	`{"type":"assistant.message","data":{"messageId":"a1","model":"claude-haiku-4.5","content":"Looking."},"timestamp":"2026-07-06T02:36:33.000Z"}`,
	`{"type":"tool.execution_start","data":{"toolCallId":"c1","toolName":"bash","arguments":{"command":"git diff\nmore"}},"timestamp":"2026-07-06T02:36:33.100Z"}`,
	`{"type":"tool.execution_complete","data":{"toolCallId":"c1","success":true,"result":{"content":"fatal: bad"},"shellExecution":{"exitCode":128}},"timestamp":"2026-07-06T02:36:33.200Z"}`,
	`{"type":"tool.execution_start","data":{"toolCallId":"c2","toolName":"view","arguments":{"path":"README.md"}},"timestamp":"2026-07-06T02:36:33.300Z"}`,
	`{"type":"permission.completed","data":{"toolCallId":"c2","result":{"kind":"denied-interactively-by-user"}},"timestamp":"2026-07-06T02:36:33.400Z"}`,
	`{"type":"tool.execution_complete","data":{"toolCallId":"c2","success":false,"error":{"message":"denied"}},"timestamp":"2026-07-06T02:36:33.500Z"}`,
	`{"type":"tool.execution_start","data":{"toolCallId":"c3","toolName":"github-create_issue","mcpServerName":"github","mcpToolName":"create_issue","arguments":{"title":"x"}},"timestamp":"2026-07-06T02:36:34.000Z"}`,
	`{"type":"tool.execution_complete","data":{"toolCallId":"c3","success":true,"result":{"content":"created"}},"timestamp":"2026-07-06T02:36:34.100Z"}`,
	`{"type":"user.message","agentId":"sub1","data":{"content":"sub-agent instructions"},"timestamp":"2026-07-06T02:36:34.200Z"}`,
	`{"type":"user.message","data":{"content":"skill text","source":"skill-pdf"},"timestamp":"2026-07-06T02:36:34.300Z"}`,
	`{"type":"assistant.message","data":{"messageId":"a1","model":"claude-haiku-4.5","content":"Looking."},"timestamp":"2026-07-06T02:36:35.000Z"}`,
	`{"type":"assistant.message","data":{"messageId":"a2","model":"claude-haiku-4.5","content":"Here is the summary."},"timestamp":"2026-07-06T02:36:36.000Z"}`,
	`{"type":"assistant.message","data":{"messageId":"a3","model":"claude-haiku-4.5","content":""},"timestamp":"2026-07-06T02:36:36.500Z"}`,
	`{"type":"session.shutdown","data":{"currentModel":"claude-haiku-4.5","tokenDetails":{"input":{"tokenCount":9},"cache_read":{"tokenCount":0},"cache_write":{"tokenCount":21105},"cache_write_1h":{"tokenCount":5},"output":{"tokenCount":355}}},"timestamp":"2026-07-06T02:36:37.407Z"}`,
)

func copilotPath(home, id string) string {
	return filepath.Join(home, ".copilot", "session-state", id, "events.jsonl")
}

func TestCopilotCLISession(t *testing.T) {
	home := setup(t)
	path := copilotPath(home, "24b9ca9f")
	writeFile(t, path, copilotEvents)

	s := onlySession(t, agentCopilotCLI)
	requireRow(t, map[string]string{
		"session_id":         "24b9ca9f",
		"project_path":       "/Users/alice/GitHub/fleet",
		"git_branch":         "dogfood",
		"git_repository":     "https://github.com/fleetdm/fleet.git",
		"agent_version":      "1.0.104",
		"model":              "claude-haiku-4.5",
		"user_messages":      "1",
		"assistant_messages": "2",
		"tool_calls":         "3",
		"input_tokens":       "9",
		"output_tokens":      "355",
		"cache_read_tokens":  "0",
		"cache_write_tokens": "21110",
		"duration_seconds":   "5",
		"transcript_path":    path,
	}, s)
	requireAbsent(t, []map[string]string{s}, "ghp_FAKE")
}

func TestCopilotCLIWithoutShutdown(t *testing.T) {
	home := setup(t)
	ls := strings.Split(strings.TrimSpace(copilotEvents), "\n")
	metrics := `{"type":"session.shutdown","data":{"modelMetrics":{"m":{"usage":{"inputTokens":100,"cacheReadTokens":40,"cacheWriteTokens":7,"outputTokens":20}}}},"timestamp":"2026-07-06T02:36:38.000Z"}`
	writeFile(t, copilotPath(home, "a"), lines(ls[:4]...))
	writeFile(t, copilotPath(home, "b"), lines(append(ls[:4:4], metrics)...))

	for _, r := range sessionRows(t, eq("agent", agentCopilotCLI)) {
		require.Equal(t, "claude-haiku-4.5", r["model"], "auto is a preference, not a model")
		switch r["transcript_path"] {
		case copilotPath(home, "a"):
			require.Equal(t, "0", r["input_tokens"])
		default:
			require.Equal(t, []string{"60", "20", "40", "7"},
				[]string{r["input_tokens"], r["output_tokens"], r["cache_read_tokens"], r["cache_write_tokens"]})
		}
	}
}

func TestCopilotCLIMessages(t *testing.T) {
	home := setup(t)
	writeFile(t, copilotPath(home, "24b9ca9f"), copilotEvents)

	rows := messageRows(t, eq("session_id", "24b9ca9f"))
	require.Equal(t, []string{
		"user/text summarize this diff",
		"assistant/text Looking.",
		`assistant/tool_use {"command":"git diff\nmore"}`,
		"system/tool_result fatal: bad",
		`assistant/tool_use {"path":"README.md"}`,
		"system/tool_result denied",
		`assistant/tool_use {"title":"x"}`,
		"system/tool_result created",
		"system/text sub-agent instructions",
		"system/text skill text",
		"assistant/text Here is the summary.",
	}, render(rows, false, "content"))
	require.Equal(t, "c1", rows[2]["tool_use_id"])
	require.Equal(t, "/Users/alice/GitHub/fleet", rows[2]["cwd"])

	require.Equal(t, []string{
		"bash | execute | git diff | error",
		"view | read | README.md | denied",
		"github-create_issue | mcp | github: create_issue | allowed",
	}, summary(toolRows(t, "24b9ca9f")))
}
