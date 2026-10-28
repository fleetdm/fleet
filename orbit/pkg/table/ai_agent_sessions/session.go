package ai_agent_sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Agent identifiers reported in the `agent` column.
const (
	agentClaudeCode = "claude_code"
	agentCopilotCLI = "copilot_cli"
	agentCursor     = "cursor"
	agentCodexCLI   = "codex_cli"
	agentGeminiCLI  = "gemini_cli"
)

// Session is one AI agent conversation, described only by metadata. Nothing
// derived from prompt or response text is ever stored here - see the README
// for the reasoning.
type Session struct {
	Agent             string
	SessionID         string
	Username          string
	ProjectPath       string
	GitBranch         string
	GitRepository     string
	AgentVersion      string
	Model             string
	StartedAt         time.Time
	EndedAt           time.Time
	UserMessages      int64
	AssistantMessages int64
	ToolCalls         int64
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	CacheWriteTokens  int64
	TranscriptPath    string
	TranscriptBytes   int64
}

// collector gathers the sessions one agent recorded under a single user's home
// directory. Collectors never return an error: an agent that is not installed,
// or whose files cannot be read, simply contributes no rows.
type collector func(u userHome, cutoff time.Time) []Session

// collectors is the full set of supported agents, in the order their rows are
// produced.
var collectors = []collector{
	collectClaudeCode,
	collectCopilotCLI,
	collectCursor,
	collectCodexCLI,
	collectGeminiCLI,
}

// row renders a session as an osquery result row.
func (s Session) row() map[string]string {
	return map[string]string{
		"agent":                 s.Agent,
		"session_id":            s.SessionID,
		"username":              s.Username,
		"project_path":          s.ProjectPath,
		"git_branch":            s.GitBranch,
		"git_repository":        s.GitRepository,
		"agent_version":         s.AgentVersion,
		"model":                 s.Model,
		"started_at":            unixString(s.StartedAt),
		"ended_at":              unixString(s.EndedAt),
		"duration_seconds":      strconv.FormatInt(s.durationSeconds(), 10),
		"user_messages":         strconv.FormatInt(s.UserMessages, 10),
		"assistant_messages":    strconv.FormatInt(s.AssistantMessages, 10),
		"tool_calls":            strconv.FormatInt(s.ToolCalls, 10),
		"input_tokens":          strconv.FormatInt(s.InputTokens, 10),
		"output_tokens":         strconv.FormatInt(s.OutputTokens, 10),
		"cache_read_tokens":     strconv.FormatInt(s.CacheReadTokens, 10),
		"cache_write_tokens":    strconv.FormatInt(s.CacheWriteTokens, 10),
		"transcript_path":       s.TranscriptPath,
		"transcript_size_bytes": strconv.FormatInt(s.TranscriptBytes, 10),
	}
}

// durationSeconds is the wall-clock span of the session, or 0 when either end
// of the range is unknown.
func (s Session) durationSeconds() int64 {
	if s.StartedAt.IsZero() || s.EndedAt.IsZero() || s.EndedAt.Before(s.StartedAt) {
		return 0
	}
	return int64(s.EndedAt.Sub(s.StartedAt).Seconds())
}

// observe widens the session's time range to include t.
func (s *Session) observe(t time.Time) {
	if t.IsZero() {
		return
	}
	if s.StartedAt.IsZero() || t.Before(s.StartedAt) {
		s.StartedAt = t
	}
	if s.EndedAt.IsZero() || t.After(s.EndedAt) {
		s.EndedAt = t
	}
}

func unixString(t time.Time) string {
	if t.IsZero() {
		return "0"
	}
	return strconv.FormatInt(t.Unix(), 10)
}

// parseTime reads the ISO-8601 timestamps every agent writes. An unparseable
// or absent value yields the zero time, which callers treat as "unknown".
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// parseUnixMillis converts a millisecond epoch, as written by some agents, to
// a time.
func parseUnixMillis(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// contentBlocks captures only the block *types* of a message body. The body
// may be a plain string or an array of typed blocks; either way the prompt and
// response text is skipped rather than read into memory.
type contentBlocks []string

func (c *contentBlocks) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil
	}
	if b[0] == '"' {
		// A bare string body is a plain text message.
		*c = contentBlocks{"text"}
		return nil
	}
	var blocks []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(b, &blocks); err != nil {
		// An unrecognized shape contributes no blocks rather than failing the
		// whole record.
		return nil
	}
	out := make(contentBlocks, 0, len(blocks))
	for _, blk := range blocks {
		out = append(out, blk.Type)
	}
	*c = out
	return nil
}

// count returns how many blocks have the given type.
func (c contentBlocks) count(blockType string) int64 {
	var n int64
	for _, t := range c {
		if t == blockType {
			n++
		}
	}
	return n
}

// hasOtherThan reports whether any block has a type other than the one given.
// It distinguishes a real user prompt from a turn that only carries tool
// results back to the model.
func (c contentBlocks) hasOtherThan(blockType string) bool {
	for _, t := range c {
		if t != blockType {
			return true
		}
	}
	return false
}

// maxRecordBytes bounds how much of a single JSONL record is buffered. Records
// larger than this are skipped; a transcript line that big is a pasted file or
// a large tool result, and it carries no metadata we need.
const maxRecordBytes = 8 << 20

// scanJSONL decodes each line of a JSON Lines file into T and hands it to fn.
// Malformed lines are skipped so one bad record does not discard a session.
func scanJSONL[T any](path string, fn func(*T)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 64*1024)
	for {
		line, readErr := readRecord(r)
		if len(line) > 0 {
			var rec T
			if json.Unmarshal(line, &rec) == nil {
				fn(&rec)
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			return readErr
		}
	}
}

// readRecord reads one newline-terminated record. Records over maxRecordBytes
// are drained and reported as empty rather than buffered.
func readRecord(r *bufio.Reader) ([]byte, error) {
	var (
		buf      []byte
		oversize bool
	)
	for {
		chunk, err := r.ReadSlice('\n')
		if !oversize {
			if len(buf)+len(chunk) > maxRecordBytes {
				oversize = true
				buf = nil
			} else {
				buf = append(buf, chunk...)
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return bytes.TrimSpace(buf), err
	}
}

// readJSONFile decodes a whole JSON document into v.
func readJSONFile(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// fileInfo returns a file's size and modification time, or zero values when it
// cannot be stat'ed.
func fileInfo(path string) (int64, time.Time) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}
	}
	return info.Size(), info.ModTime().UTC()
}

// isStale reports whether a file was last written before the cutoff. A zero
// cutoff disables the check.
func isStale(path string, cutoff time.Time) bool {
	if cutoff.IsZero() {
		return false
	}
	_, modTime := fileInfo(path)
	if modTime.IsZero() {
		return false
	}
	return modTime.Before(cutoff)
}

// subdirs lists the directory names directly under dir, or nothing when dir is
// absent or unreadable.
func subdirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// resolveSlugPath turns a dash-encoded project slug back into a filesystem
// path. Encoding a path as "Users-alice-GitHub-fleet-extensions" is lossy,
// because directory names may themselves contain dashes, so this walks the
// filesystem and prefers the longest segment run that names a real directory.
// When the path no longer exists, the naive expansion is returned so the
// column still identifies the project.
func resolveSlugPath(root, slug string) string {
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ""
	}
	parts := strings.Split(slug, "-")
	naive := filepath.Join(root, filepath.Join(parts...))

	current := root
	for i := 0; i < len(parts); {
		matched := false
		// Longest first: prefer "fleet-extensions" over "fleet" + "extensions".
		for j := len(parts); j > i; j-- {
			candidate := filepath.Join(current, strings.Join(parts[i:j], "-"))
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				current, i, matched = candidate, j, true
				break
			}
		}
		if !matched {
			return naive
		}
	}
	return current
}
