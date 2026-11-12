package ai_agent_sessions

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Session is one AI agent conversation, described by metadata only.
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

	// provider replaces the account's sign-in when the session ran against a
	// third-party model provider (Codex records it per session).
	provider string
	account  account
}

func (s Session) row(a account) map[string]string {
	if s.provider != "" {
		a.Identifier = s.provider
	}
	return map[string]string{
		"agent":                 s.Agent,
		"session_id":            capField(s.SessionID),
		"username":              s.Username,
		"account_identifier":    capField(a.Identifier),
		"project_path":          capField(s.ProjectPath),
		"git_branch":            capField(s.GitBranch),
		"git_repository":        capField(s.GitRepository),
		"agent_version":         capField(s.AgentVersion),
		"model":                 capField(s.Model),
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
		"transcript_path":       capField(s.TranscriptPath),
		"transcript_size_bytes": strconv.FormatInt(s.TranscriptBytes, 10),
	}
}

// durationSeconds is the session's wall-clock span, or 0 when either end is
// unknown.
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

// source is one session's files, found without reading them.
type source struct {
	agent *agent
	sc    *scope
	// id is the session id, when the layout gives it before parsing.
	id string
	// files are parsed in order: the session's transcript, then any
	// sub-agent transcripts merged into it.
	files []string
	// deps are other files the parse reads, which the cache must also watch.
	deps    []string
	modTime time.Time
	size    int64
	// extra is agent-specific discovery data.
	extra any
}

// newSource stats files, which must all be regular files, and reports false
// when the first isn't or every file predates cutoff.
func newSource(a *agent, sc *scope, id string, files []string, cutoff time.Time) (*source, bool) {
	src := &source{agent: a, sc: sc, id: id}
	for i, p := range files {
		fi, ok := regularFile(p)
		if !ok {
			if i == 0 {
				return nil, false
			}
			continue
		}
		src.files = append(src.files, p)
		src.size += fi.Size()
		if mt := fi.ModTime(); mt.After(src.modTime) {
			src.modTime = mt
		}
	}
	if !cutoff.IsZero() && src.modTime.Before(cutoff) {
		return nil, false
	}
	return src, true
}

// parseSession reads a source into a Session. It reports false when the
// transcript held nothing usable, or couldn't be read whole.
func parseSession(src *source) (Session, bool) {
	s := Session{
		Agent:           src.agent.name,
		SessionID:       src.id,
		Username:        src.sc.home.Username,
		TranscriptPath:  src.files[0],
		TranscriptBytes: src.size,
	}
	r := &recorder{s: &s, msgIndex: -1}
	transcriptsParsed.Add(1)
	if !parse(src, r) || !r.seen || src.sc.reads.exhausted {
		return Session{}, false
	}
	// Transcripts without timestamps still have a last write.
	if s.EndedAt.IsZero() {
		s.EndedAt = src.modTime.UTC()
	}
	s.capFields()
	return s, true
}

// capFields bounds the strings a parser took from a file, since the Session
// is kept in the cache.
func (s *Session) capFields() {
	for _, f := range []*string{&s.SessionID, &s.ProjectPath, &s.GitBranch, &s.GitRepository, &s.AgentVersion, &s.Model, &s.TranscriptPath, &s.provider} {
		*f = capField(*f)
	}
}

// sanitizeRepoURL drops credentials from a git remote URL: tokens embedded as
// https://user:token@host/… are common.
func sanitizeRepoURL(s string) string {
	if u, err := url.Parse(s); err == nil && u.Scheme != "" && u.User != nil {
		u.User = nil
		return u.String()
	}
	// scp-style user:secret@host:path
	if before, after, found := strings.Cut(s, "@"); found && before != "" && !strings.Contains(before, "/") {
		if user, _, found := strings.Cut(before, ":"); found {
			return user + "@" + after
		}
	}
	return s
}

func unixString(t time.Time) string {
	if t.IsZero() {
		return "0"
	}
	return strconv.FormatInt(t.Unix(), 10)
}

// parseTime reads the RFC 3339 timestamps agents write; anything else is the
// zero time, which means unknown.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func parseUnixMillis(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// regularFile stats path without following a link and reports whether it is
// a regular file.
func regularFile(path string) (os.FileInfo, bool) {
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return nil, false
	}
	return fi, true
}

// Bounds on directory walks, which users control the shape of.
const (
	maxWalkDepth    = 6
	maxFilesPerWalk = maxSourcesPerAgent
)

// walkDepth is how many levels p is below root.
func walkDepth(root, p string) int {
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

// subdirs lists the real directories (not links to them) directly under dir.
func subdirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// filesWithSuffix lists the regular files directly under dir ending in one
// of suffixes.
func filesWithSuffix(dir string, suffixes ...string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		for _, suf := range suffixes {
			if strings.HasSuffix(e.Name(), suf) {
				out = append(out, filepath.Join(dir, e.Name()))
				break
			}
		}
	}
	return out
}

// maxSlugSegments bounds the directory probes resolveSlugPath makes, which
// grow with the square of the segment count.
const maxSlugSegments = 48

// resolveSlugPath turns a dash-encoded project slug back into a path. The
// encoding is lossy (directory names can contain dashes), so it prefers the
// longest run of segments that names a real directory, and falls back to the
// naive expansion when the path no longer exists.
func resolveSlugPath(root, slug string) string {
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ""
	}
	parts := strings.Split(slug, "-")
	if len(parts) > maxSlugSegments {
		return filepath.Join(root, filepath.Join(parts...))
	}
	naive := filepath.Join(root, filepath.Join(parts...))

	current := root
	for i := 0; i < len(parts); {
		matched := false
		for j := len(parts); j > i; j-- {
			candidate := filepath.Join(current, strings.Join(parts[i:j], "-"))
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
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
