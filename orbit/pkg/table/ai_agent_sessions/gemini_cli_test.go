package ai_agent_sessions

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/stretchr/testify/require"
)

const geminiID = "aaaaaaaa-1111-2222-3333-444444444444"

var geminiChat = lines(
	`{"sessionId":"`+geminiID+`","projectHash":"h","startTime":"2026-08-01T10:00:00.000Z","lastUpdated":"2026-08-01T10:00:00.000Z","kind":"main"}`,
	`{"id":"m1","timestamp":"2026-08-01T10:00:01.000Z","type":"user","content":[{"text":"list files"}]}`,
	`{"id":"m2","timestamp":"2026-08-01T10:00:05.000Z","type":"gemini","content":"Sure.","model":"gemini-3-pro","thoughts":[{"subject":"Plan","description":"look around","timestamp":"2026-08-01T10:00:04.000Z"}],"tokens":{"input":100,"output":10,"cached":40,"total":110},"toolCalls":[{"id":"tc1","name":"run_shell_command","args":{"command":"ls -la"},"status":"scheduled","timestamp":"2026-08-01T10:00:05.000Z"}]}`,
	// The same message is re-appended as tokens and tool status update.
	`{"id":"m2","timestamp":"2026-08-01T10:00:05.000Z","type":"gemini","content":"Sure.","model":"gemini-3-pro","thoughts":[{"subject":"Plan","description":"look around","timestamp":"2026-08-01T10:00:04.000Z"}],"tokens":{"input":120,"output":12,"cached":40,"total":132},"toolCalls":[{"id":"tc1","name":"run_shell_command","args":{"command":"ls -la"},"status":"success","result":[{"functionResponse":{"id":"tc1","name":"run_shell_command","response":{"output":"Command: ls -la\nExit Code: 2"}}}],"timestamp":"2026-08-01T10:00:05.000Z"}]}`,
	`{"id":"m3","timestamp":"2026-08-01T10:00:10.000Z","type":"user","content":[{"functionResponse":{"id":"tc1","name":"run_shell_command","response":{"output":"Exit Code: 2"}}}]}`,
	`{"id":"m4","timestamp":"2026-08-01T10:00:20.000Z","type":"gemini","content":"","model":"gemini-3-pro","tokens":{"input":50,"output":5,"cached":0,"total":55},"toolCalls":[{"id":"tc2","name":"read_file","args":{"file_path":"a.go"},"status":"cancelled","resultDisplay":"User denied execution.","timestamp":"2026-08-01T10:00:20.000Z"},{"id":"tc3","name":"create_issue","displayName":"create_issue (github MCP Server)","args":{"title":"x"},"status":"success","result":[{"functionResponse":{"id":"tc3","name":"create_issue","response":{"output":"ok"}}}],"timestamp":"2026-08-01T10:00:21.000Z"}]}`,
	`{"id":"m5","timestamp":"2026-08-01T10:00:30.000Z","type":"user","content":"oops"}`,
	`{"$rewindTo":"m5"}`,
	`{"$set":{"lastUpdated":"2026-08-01T10:05:00.000Z"}}`,
	`{"id":"m6","timestamp":"2026-08-01T10:01:00.000Z","type":"info","content":"Switched model"}`,
)

func writeGeminiFixture(t *testing.T, home string) string {
	t.Helper()
	proj := filepath.Join(home, ".gemini", "tmp", "fleet")
	writeFile(t, filepath.Join(proj, ".project_root"), "/Users/alice/GitHub/fleet\n")
	chat := filepath.Join(proj, "chats", "session-2026-08-01T10-00-aaaaaaaa.jsonl")
	writeFile(t, chat, geminiChat)

	// A copy left behind in the older format is the same session.
	old := filepath.Join(proj, "chats", "session-2026-08-01T10-00-aaaaaaaa.json")
	writeFile(t, old, `{"sessionId":"`+geminiID+`","projectHash":"h","startTime":"2026-08-01T10:00:00.000Z","lastUpdated":"2026-08-01T10:00:01.000Z","messages":[{"id":"m1","timestamp":"2026-08-01T10:00:01.000Z","type":"user","content":"list files"}]}`)
	touch(t, old, time.Now().Add(-time.Hour))

	writeFile(t, filepath.Join(proj, "chats", geminiID, "sub1.jsonl"), lines(
		`{"sessionId":"sub-1","projectHash":"h","startTime":"2026-08-01T10:00:06.000Z","kind":"subagent"}`,
		`{"id":"s1","timestamp":"2026-08-01T10:00:07.000Z","type":"gemini","content":"","toolCalls":[{"id":"st1","name":"glob","args":{"pattern":"*.go"},"status":"success"}]}`,
	))
	// logs.json is never pruned, so a session only found there is in the
	// window by its own timestamps.
	recent := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	writeFile(t, filepath.Join(proj, "logs.json"), `[
	  {"sessionId":"`+geminiID+`","messageId":0,"type":"user","message":"list files","timestamp":"2026-08-01T10:00:01.000Z"},
	  {"sessionId":"bbbb","messageId":0,"type":"user","message":"only in logs","timestamp":"`+recent.Format(time.RFC3339)+`"},
	  {"sessionId":"bbbb","messageId":1,"type":"user","message":"/help","timestamp":"`+recent.Add(30*time.Second).Format(time.RFC3339)+`"}
	]`)
	return chat
}

func TestGeminiCLISessions(t *testing.T) {
	home := setup(t)
	chat := writeGeminiFixture(t, home)

	rows := sessionRows(t, eq("agent", agentGeminiCLI))
	require.Len(t, rows, 2)
	byID := rowsByID(rows)
	requireRow(t, map[string]string{
		"project_path":  "/Users/alice/GitHub/fleet",
		"model":         "gemini-3-pro",
		"started_at":    "1785578400",
		"ended_at":      "1785578700",
		"user_messages": "1",
		// Including the sub-agent's.
		"assistant_messages": "3",
		"tool_calls":         "4",
		"input_tokens":       "130",
		"output_tokens":      "17",
		"cache_read_tokens":  "40",
		"cache_write_tokens": "0",
		"transcript_path":    chat,
	}, byID[geminiID])

	logOnly := byID["bbbb"]
	require.Equal(t, "2", logOnly["user_messages"])
	require.Equal(t, "/Users/alice/GitHub/fleet", logOnly["project_path"])
	require.Equal(t, "30", logOnly["duration_seconds"])
}

func TestGeminiCLIMessages(t *testing.T) {
	home := setup(t)
	writeGeminiFixture(t, home)

	rows := messageRows(t, eq("session_id", geminiID))
	require.Equal(t, []string{
		"user/text list files",
		"assistant/thinking Plan: look around",
		"assistant/text Sure.",
		`assistant/tool_use {"command":"ls -la"}`,
		"system/tool_result Command: ls -la\nExit Code: 2",
		// The sub-agent's call, placed by its timestamp.
		`assistant/tool_use {"pattern":"*.go"}`,
		`assistant/tool_use {"file_path":"a.go"}`,
		"system/tool_result User denied execution.",
		`assistant/tool_use {"title":"x"}`,
		"system/tool_result ok",
	}, render(rows, false, "content"))
	require.Equal(t, "gemini-3-pro", rows[2]["model"])
	require.Equal(t, "tc1", rows[3]["tool_use_id"])

	require.Equal(t, []string{
		"run_shell_command | execute | ls -la | error",
		"glob | search | *.go | allowed",
		"read_file | read | a.go | denied",
		"create_issue | mcp | github: create_issue | allowed",
	}, summary(toolRows(t, geminiID)))

	logRows := messageRows(t, eq("session_id", "bbbb"))
	require.Len(t, logRows, 2)
	require.Equal(t, "only in logs", logRows[0]["content"])
}

func TestGeminiCLILegacyHashDirAndSandboxRoot(t *testing.T) {
	home := setup(t)
	writeFile(t, filepath.Join(home, ".gemini", "projects.json"), `{"projects":{"/Users/alice/old":"old"}}`)
	hash := "cee9819bdc76d42895d8b90c2b727802ef7bd78abc6fc34db063641eec42a070"
	writeFile(t, filepath.Join(home, ".cache", ".gemini", "tmp", hash, "chats", "session-2026-08-03T10-00-cccccccc.json"),
		`{"sessionId":"cccccccc","projectHash":"`+hash+`","startTime":"2026-08-03T10:00:00.000Z","lastUpdated":"2026-08-03T10:01:00.000Z","messages":[`+
			`{"id":"x1","timestamp":"2026-08-03T10:00:00.000Z","type":"user","content":"legacy json"},`+
			`{"id":"x2","timestamp":"2026-08-03T10:00:30.000Z","type":"gemini","content":"ok","model":"gemini-2.5-pro","tokens":{"input":10,"output":1,"cached":0,"total":11}}]}`)
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "bin", "chats", "session-x.json"), `{"sessionId":"bin"}`)

	s := onlySession(t, agentGeminiCLI)
	require.Equal(t, "cccccccc", s["session_id"])
	require.Equal(t, "/Users/alice/old", s["project_path"])
	require.Equal(t, "60", s["duration_seconds"])
	require.Equal(t, "1", s["assistant_messages"])
}

func TestGeminiCLILogOnlySessionsAgeOut(t *testing.T) {
	home := setup(t)
	// Gemini prunes chats/ after 30 days but never logs.json, so an old
	// session's prompts stay in a file that is rewritten with every prompt.
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "p", "logs.json"), `[
	  {"sessionId":"ancient","messageId":0,"type":"user","message":"a year ago","timestamp":"2025-10-01T10:00:00.000Z"},
	  {"sessionId":"recent","messageId":0,"type":"user","message":"today","timestamp":"`+time.Now().UTC().Format(time.RFC3339)+`"}
	]`)
	rows := sessionRows(t, eq("agent", agentGeminiCLI))
	require.Len(t, rows, 1)
	require.Equal(t, "recent", rows[0]["session_id"])
	require.Len(t, sessionRows(t, eq("agent", agentGeminiCLI), cons{"ended_at", table.OperatorLessThan, "9999999999"}), 2)
	require.Len(t, messageRows(t, eq("session_id", "ancient")), 1)
}

func TestGeminiCLIDecisionCannotBeSpoofed(t *testing.T) {
	out := func(s string) json.RawMessage {
		b, err := json.Marshal([]map[string]any{{"functionResponse": map[string]any{"response": map[string]any{"output": s}}}})
		require.NoError(t, err)
		return b
	}
	shell := "run_shell_command"
	require.Equal(t, decisionError, geminiDecision(geminiToolCall{Name: shell, Status: "success", Result: out("Exit Code: 0\nlater\nExit Code: 1")}))
	require.Equal(t, decisionAllowed, geminiDecision(geminiToolCall{Name: shell, Status: "success", Result: out("printed Exit Code: 3 inline\nExit Code: 0")}))
	require.Equal(t, decisionAllowed, geminiDecision(geminiToolCall{Name: "read_file", Status: "success", Result: out("a script that prints\nExit Code: 1")}),
		"only the shell tool reports an exit code")
	require.Equal(t, decisionError, geminiDecision(geminiToolCall{Status: "cancelled", Result: out("file says User denied execution.")}))
}

func TestGeminiCLILogSessionsCapKeepsNewest(t *testing.T) {
	home := setup(t)
	// Oldest first, as logs.json is appended; one more than the cap, all
	// within the default window.
	var entries []string
	for i := maxGeminiLogSessions; i >= 0; i-- {
		ts := time.Now().UTC().Add(-time.Duration(i) * time.Minute).Format(time.RFC3339)
		entries = append(entries, `{"sessionId":"s`+strconv.Itoa(i)+`","messageId":0,"type":"user","message":"m","timestamp":"`+ts+`"}`)
	}
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "p", "logs.json"), "["+strings.Join(entries, ",")+"]")

	byID := rowsByID(sessionRows(t, eq("agent", agentGeminiCLI)))
	require.Len(t, byID, maxGeminiLogSessions)
	require.Contains(t, byID, "s0", "the newest session is kept")
	require.NotContains(t, byID, "s"+strconv.Itoa(maxGeminiLogSessions), "the oldest is dropped")
}

func TestGeminiCLIProjectRootIsBounded(t *testing.T) {
	home := setup(t)
	proj := filepath.Join(home, ".gemini", "tmp", "p")
	writeFile(t, filepath.Join(proj, ".project_root"), "/"+strings.Repeat("d", 1<<20))
	writeFile(t, filepath.Join(proj, "chats", "session-2026-08-01T10-00-aaaaaaaa.jsonl"), lines(
		`{"sessionId":"aaaa","startTime":"2026-08-01T10:00:00.000Z","kind":"main"}`,
		`{"id":"m1","timestamp":"2026-08-01T10:00:01.000Z","type":"user","content":"hi"}`,
	))
	// A directory with no session never has its .project_root read.
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "empty", ".project_root"), "/nothing")
	setReadBudget(t, 64<<10)
	s := onlySession(t, agentGeminiCLI)
	require.Len(t, s["project_path"], maxFieldBytes)
}

func TestGeminiCLIReplayIsLinear(t *testing.T) {
	home := setup(t)
	var sb strings.Builder
	sb.WriteString(`{"sessionId":"aaaa","startTime":"2026-08-01T10:00:00.000Z","kind":"main"}` + "\n")
	for i := range 5000 {
		sb.WriteString(`{"id":"m` + strings.Repeat("0", 5-len(strconv.Itoa(i))) + strconv.Itoa(i) + `","timestamp":"2026-08-01T10:00:01.000Z","type":"user","content":"p"}` + "\n")
	}
	for range 50000 {
		sb.WriteString(`{"$patch":{"removeIds":["zzz"],"orderIds":["m00001","m00000"]}}` + "\n")
	}
	sb.WriteString(`{"$patch":{"removeIds":["m00002"]}}` + "\n")
	sb.WriteString(`{"$rewindTo":"m04000"}` + "\n")
	writeFile(t, filepath.Join(home, ".gemini", "tmp", "p", "chats", "session-2026-08-01T10-00-aaaaaaaa.jsonl"), sb.String())

	start := time.Now()
	s := onlySession(t, agentGeminiCLI)
	require.Less(t, time.Since(start), 3*time.Second)
	require.Equal(t, "3999", s["user_messages"])
	rows := messageRows(t, eq("session_id", "aaaa"), cons{"message_index", table.OperatorLessThan, "2"})
	require.Equal(t, []string{"user/text p", "user/text p"}, render(rows, false, "content"))
}
