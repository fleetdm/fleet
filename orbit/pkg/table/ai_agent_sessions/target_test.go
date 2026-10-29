package ai_agent_sessions

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenericTarget(t *testing.T) {
	for _, tc := range []struct {
		tool, input, action, target string
	}{
		{"SomeNewTool", `{"query":"q","path":"p"}`, "other", "p"},
		{"SomeNewTool", `{"other":1}`, "other", "SomeNewTool"},
		{"mcp__claude_ai_Gmail__search__threads", `{"query":"from:bob"}`, "mcp", "claude_ai_Gmail: search__threads from:bob"},
		{"Bash", `{"command":"echo ` + strings.Repeat("é", 200) + `"}`, "execute", "echo " + strings.Repeat("é", 125)},
	} {
		action, target := classify(agentClaudeCode, tc.tool, []byte(tc.input))
		require.Equal(t, tc.action, action, tc.tool)
		require.Equal(t, tc.target, target, tc.tool)
		require.LessOrEqual(t, len(target), maxTargetBytes)
	}
}
