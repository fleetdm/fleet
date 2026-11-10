package ai_agent_sessions

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeClaudeToolUse(t *testing.T, home, sessionID, name, input string) {
	t.Helper()
	writeClaudeTranscript(t, home, sessionID,
		`{"type":"assistant","timestamp":"2026-08-01T10:00:00.000Z","message":{"id":"m1","content":[{"type":"tool_use","id":"a","name":"`+name+`","input":`+input+`}]}}`)
}

func TestLargeToolInputIsCut(t *testing.T) {
	t.Run("JSON stays valid", func(t *testing.T) {
		home := setup(t)
		input := `{"file_path": "/src/a.go", "content": "` + strings.Repeat("x", 20<<10) + `", "mode": "overwrite"}`
		writeClaudeToolUse(t, home, "s", "Write", input)

		rows := messageRows(t, eq("block_type", "tool_use"))
		require.Len(t, rows, 1)
		r := rows[0]
		require.Equal(t, "1", r["truncated"])
		require.Equal(t, strconv.Itoa(len(input)), r["content_length"])
		require.LessOrEqual(t, len(r["content"]), maxBlockBytes)
		require.True(t, json.Valid([]byte(r["content"])), r["content"][:64])
		// Key order and the short values are kept; only the long string is cut.
		require.True(t, strings.HasPrefix(r["content"], `{"file_path":"/src/a.go","content":"xxx`))
		require.True(t, strings.HasSuffix(r["content"], `","mode":"overwrite"}`))
	})
	t.Run("anything else is cut by bytes", func(t *testing.T) {
		home := setup(t)
		writeClaudeToolUse(t, home, "s", "Bash", `"`+strings.Repeat("y", 20<<10)+`"`)
		r := messageRows(t, eq("block_type", "tool_use"))[0]
		require.Equal(t, "1", r["truncated"])
		require.Len(t, r["content"], maxBlockBytes)
	})
}

func TestLongestStringsAreCutFirst(t *testing.T) {
	in := `{"a":"` + strings.Repeat("a", 6000) + `","b":"` + strings.Repeat("b", 5000) + `","c":"` + strings.Repeat("c", 100) + `","n":[1,2,3]}`
	out, ok := shortenJSON(in, 8<<10)
	require.True(t, ok)
	require.LessOrEqual(t, len(out), 8<<10)

	var v struct {
		A, B, C string
		N       []int
	}
	require.NoError(t, json.Unmarshal([]byte(out), &v))
	require.Equal(t, strings.Repeat("c", 100), v.C)
	require.Equal(t, []int{1, 2, 3}, v.N)
	// Both long strings end up the same length rather than one being dropped.
	require.Len(t, v.B, len(v.A))
	require.Greater(t, len(v.A), 3500)
}

func TestShortenJSONKeepsEscapesWhole(t *testing.T) {
	in := `["` + strings.Repeat(`é\"\\`, 3000) + `"]`
	for limit := 10; limit < 200; limit++ {
		out, ok := shortenJSON(in, limit)
		require.True(t, ok)
		require.LessOrEqual(t, len(out), limit)
		require.True(t, json.Valid([]byte(out)), out)
	}
}

func TestShortenJSONGivesUp(t *testing.T) {
	for name, in := range map[string]string{
		"not json":      "*** Begin Patch\n" + strings.Repeat("+x\n", 5000),
		"bare string":   `"` + strings.Repeat("x", 9000) + `"`,
		"invalid":       `{"a":"` + strings.Repeat("x", 9000),
		"no strings":    "[" + strings.Repeat("1,", 9000) + "1]",
		"too many keys": "{" + strings.Repeat(`"k":"",`, 2000) + `"k":""}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, ok := shortenJSON(in, 8<<10)
			require.False(t, ok)
		})
	}
}
