package ai_agent_sessions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeFile creates path, including any missing parent directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func TestContentBlocks(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      contentBlocks
		wantTools int64
		wantOther bool
	}{
		{
			name:      "plain string body counts as text",
			input:     `"just some prompt text"`,
			want:      contentBlocks{"text"},
			wantOther: true,
		},
		{
			name:      "typed blocks",
			input:     `[{"type":"text","text":"hi"},{"type":"tool_use","name":"Read"},{"type":"tool_use","name":"Bash"}]`,
			want:      contentBlocks{"text", "tool_use", "tool_use"},
			wantTools: 2,
			wantOther: true,
		},
		{
			name:      "tool results only",
			input:     `[{"type":"tool_result","content":"output"}]`,
			want:      contentBlocks{"tool_result"},
			wantOther: false,
		},
		{
			name:  "unexpected shape yields no blocks",
			input: `{"type":"text"}`,
			want:  nil,
		},
		{
			name:  "null yields no blocks",
			input: `null`,
			want:  contentBlocks{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got contentBlocks
			if err := json.Unmarshal([]byte(tc.input), &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("blocks = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("blocks = %v, want %v", got, tc.want)
				}
			}
			if n := got.count("tool_use"); n != tc.wantTools {
				t.Errorf("count(tool_use) = %d, want %d", n, tc.wantTools)
			}
			if other := got.hasOtherThan("tool_result"); other != tc.wantOther {
				t.Errorf("hasOtherThan(tool_result) = %v, want %v", other, tc.wantOther)
			}
		})
	}
}

func TestScanJSONLSkipsMalformedAndOversizedRecords(t *testing.T) {
	type rec struct {
		N int `json:"n"`
	}

	// A record past the size cap must be dropped without derailing the ones
	// that follow it.
	huge := `{"n":99,"pad":"` + strings.Repeat("x", maxRecordBytes) + `"}`
	path := filepath.Join(t.TempDir(), "records.jsonl")
	writeFile(t, path, strings.Join([]string{
		`{"n":1}`,
		`not json at all`,
		huge,
		``,
		`{"n":2}`,
	}, "\n"))

	var got []int
	if err := scanJSONL(path, func(r *rec) { got = append(got, r.N) }); err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}

	want := []int{1, 2}
	if len(got) != len(want) {
		t.Fatalf("decoded %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("decoded %v, want %v", got, want)
		}
	}
}

func TestScanJSONLHandlesRecordLargerThanReadBuffer(t *testing.T) {
	type rec struct {
		N   int    `json:"n"`
		Pad string `json:"pad"`
	}

	// 256 KiB overflows the 64 KiB bufio reader, exercising the ErrBufferFull
	// path in readRecord.
	path := filepath.Join(t.TempDir(), "big.jsonl")
	writeFile(t, path, `{"n":7,"pad":"`+strings.Repeat("y", 256*1024)+`"}`+"\n")

	var got []int
	if err := scanJSONL(path, func(r *rec) { got = append(got, r.N) }); err != nil {
		t.Fatalf("scanJSONL: %v", err)
	}
	if len(got) != 1 || got[0] != 7 {
		t.Fatalf("decoded %v, want [7]", got)
	}
}

func TestScanJSONLMissingFile(t *testing.T) {
	err := scanJSONL(filepath.Join(t.TempDir(), "absent.jsonl"), func(*struct{}) {})
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestResolveSlugPath(t *testing.T) {
	root := t.TempDir()
	// A directory whose own name contains a dash is the case a naive split
	// gets wrong.
	if err := os.MkdirAll(filepath.Join(root, "Users", "alice", "GitHub", "fleet-extensions"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		slug string
		want string
	}{
		{
			name: "rejoins dashed directory names",
			slug: "Users-alice-GitHub-fleet-extensions",
			want: filepath.Join(root, "Users", "alice", "GitHub", "fleet-extensions"),
		},
		{
			name: "plain path",
			slug: "Users-alice-GitHub",
			want: filepath.Join(root, "Users", "alice", "GitHub"),
		},
		{
			name: "falls back to naive expansion when the path is gone",
			slug: "Users-bob-Projects-thing",
			want: filepath.Join(root, "Users", "bob", "Projects", "thing"),
		},
		{
			name: "empty slug",
			slug: "-",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveSlugPath(root, tc.slug); got != tc.want {
				t.Errorf("resolveSlugPath(%q) = %q, want %q", tc.slug, got, tc.want)
			}
		})
	}
}

func TestSessionDurationSeconds(t *testing.T) {
	start := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		s    Session
		want int64
	}{
		{
			name: "normal range",
			s:    Session{StartedAt: start, EndedAt: start.Add(90 * time.Second)},
			want: 90,
		},
		{"missing start", Session{EndedAt: start}, 0},
		{"missing end", Session{StartedAt: start}, 0},
		{"end before start", Session{StartedAt: start, EndedAt: start.Add(-time.Hour)}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.durationSeconds(); got != tc.want {
				t.Errorf("durationSeconds() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestSessionObserveWidensRange(t *testing.T) {
	base := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

	var s Session
	s.observe(time.Time{}) // zero times are ignored
	s.observe(base)
	s.observe(base.Add(-time.Hour))
	s.observe(base.Add(time.Hour))

	if !s.StartedAt.Equal(base.Add(-time.Hour)) {
		t.Errorf("StartedAt = %v, want %v", s.StartedAt, base.Add(-time.Hour))
	}
	if !s.EndedAt.Equal(base.Add(time.Hour)) {
		t.Errorf("EndedAt = %v, want %v", s.EndedAt, base.Add(time.Hour))
	}
}

func TestCutoffTime(t *testing.T) {
	now := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

	if got := cutoffTime(now, 0); !got.IsZero() {
		t.Errorf("cutoffTime(now, 0) = %v, want zero", got)
	}
	if got := cutoffTime(now, -5); !got.IsZero() {
		t.Errorf("cutoffTime(now, -5) = %v, want zero", got)
	}
	want := now.AddDate(0, 0, -30)
	if got := cutoffTime(now, 30); !got.Equal(want) {
		t.Errorf("cutoffTime(now, 30) = %v, want %v", got, want)
	}
}

func TestIsStale(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.jsonl")
	writeFile(t, path, "{}\n")

	if isStale(path, time.Time{}) {
		t.Error("a zero cutoff must disable the age check")
	}
	if isStale(path, time.Now().Add(-time.Hour)) {
		t.Error("a file written just now is not stale")
	}
	if !isStale(path, time.Now().Add(time.Hour)) {
		t.Error("a file written before the cutoff is stale")
	}
}

func TestSessionRowRendersEveryColumn(t *testing.T) {
	start := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	s := Session{
		Agent:     agentClaudeCode,
		SessionID: "abc",
		StartedAt: start,
		EndedAt:   start.Add(time.Minute),
	}

	row := s.row()
	for _, col := range aiAgentSessionsColumns() {
		if _, ok := row[col.Name]; !ok {
			t.Errorf("row is missing column %q", col.Name)
		}
	}
	if len(row) != len(aiAgentSessionsColumns()) {
		t.Errorf("row has %d keys, want %d", len(row), len(aiAgentSessionsColumns()))
	}
	if row["duration_seconds"] != "60" {
		t.Errorf("duration_seconds = %q, want %q", row["duration_seconds"], "60")
	}
	// An unknown timestamp is reported as 0 rather than a 1970 date.
	if got := (Session{}).row()["started_at"]; got != "0" {
		t.Errorf("started_at for an unknown time = %q, want %q", got, "0")
	}
}

func TestCollectSessionsSortsNewestFirst(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".claude", "projects", "-tmp-proj")

	writeFile(t, filepath.Join(dir, "older.jsonl"),
		`{"type":"user","timestamp":"2026-08-01T10:00:00.000Z","cwd":"/tmp/proj","message":{"content":"hi"}}`+"\n")
	writeFile(t, filepath.Join(dir, "newer.jsonl"),
		`{"type":"user","timestamp":"2026-08-10T10:00:00.000Z","cwd":"/tmp/proj","message":{"content":"hi"}}`+"\n")

	sessions := collectSessions([]userHome{{Username: "alice", HomeDir: home}}, time.Time{})
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if sessions[0].SessionID != "newer" || sessions[1].SessionID != "older" {
		t.Errorf("session order = [%s %s], want [newer older]", sessions[0].SessionID, sessions[1].SessionID)
	}
}

func TestCollectSessionsIgnoresEmptyHome(t *testing.T) {
	sessions := collectSessions([]userHome{{Username: "alice", HomeDir: t.TempDir()}}, time.Time{})
	if len(sessions) != 0 {
		t.Errorf("got %d sessions from an empty home directory, want 0", len(sessions))
	}
}

// touch sets a file's modification time so ordering tests are deterministic.
func touch(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("Chtimes(%s): %v", path, err)
	}
}
