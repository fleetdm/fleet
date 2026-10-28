package ai_agent_sessions

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/internal/homes"
)

// Gemini CLI records each session under ~/.gemini/tmp/<project>/chats/ as
// session-<time>-<id prefix>.json (one document, up to v0.38) or .jsonl
// (append-only, replayed with last-write-wins and $set/$rewindTo/$patch
// records). Sub-agents get chats/<parent session id>/<id>.jsonl. Older
// sessions may only be in the project's logs.json, which records prompts.
// Under the macOS sandbox the same layout lives in ~/.cache/.gemini.
var geminiCLI = &agent{
	name:     agentGeminiCLI,
	discover: discoverGeminiCLI,
	parse:    parseGeminiCLI,
	account:  geminiAccount,
}

func geminiRoots(h homes.Home) []string {
	return []string{filepath.Join(h.Dir, ".gemini"), filepath.Join(h.Dir, ".cache", ".gemini")}
}

type geminiSource struct {
	project string
	// logs is the logs.json a session only found there is read from.
	logs *geminiLogFile
}

// geminiLogFile reads one logs.json at most once per query.
type geminiLogFile struct {
	sc        *scope
	path      string
	once      sync.Once
	order     []string
	bySession map[string][]geminiLogEntry
}

func (f *geminiLogFile) load() map[string][]geminiLogEntry {
	f.once.Do(func() {
		discoveryReads.Add(1)
		var entries []geminiLogEntry
		readJSON(f.sc, f.path, &entries)
		f.bySession = map[string][]geminiLogEntry{}
		for _, e := range entries {
			if e.SessionID == "" {
				continue
			}
			if _, ok := f.bySession[e.SessionID]; !ok {
				f.order = append(f.order, e.SessionID)
			}
			f.bySession[e.SessionID] = append(f.bySession[e.SessionID], e)
		}
	})
	return f.bySession
}

// geminiLogSession is what discovery remembers of a session in logs.json: the
// file is never pruned, so this is how old sessions age out of the window.
type geminiLogSession struct {
	id   string
	last time.Time
}

var geminiLogIndexes = newDerivedCache[[]geminiLogSession]()

func (f *geminiLogFile) index() []geminiLogSession {
	return geminiLogIndexes.get(f.path, func() []geminiLogSession {
		bySession := f.load()
		idx := make([]geminiLogSession, 0, len(f.order))
		for _, id := range f.order {
			var last time.Time
			for _, e := range bySession[id] {
				if t := parseTime(e.Timestamp); t.After(last) {
					last = t
				}
			}
			idx = append(idx, geminiLogSession{id, last})
		}
		return idx
	})
}

// maxGeminiLogSessions bounds how many sessions one logs.json yields, and
// maxGeminiMessages how many messages a chat replay keeps.
const (
	maxGeminiLogSessions = 1000
	maxGeminiMessages    = 100_000
)

type geminiLogEntry struct {
	SessionID string `json:"sessionId"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func discoverGeminiCLI(a *agent, sc *scope, cutoff time.Time) []*source {
	h := sc.home
	type chat struct {
		path    string
		project func() string
		deps    []string
		mod     time.Time
	}
	fresh := func(p string) (time.Time, bool) {
		fi, ok := regularFile(p)
		if !ok || (!cutoff.IsZero() && fi.ModTime().Before(cutoff)) {
			return time.Time{}, false
		}
		return fi.ModTime(), true
	}

	projects := geminiProjects(sc)
	var projectFiles []string
	for _, root := range geminiRoots(h) {
		projectFiles = append(projectFiles, filepath.Join(root, "projects.json"))
	}
	mains := map[string]chat{}
	subs := map[string][]string{}
	type logFile struct {
		path    string
		project func() string
		deps    []string
	}
	var logs []logFile
	for _, root := range geminiRoots(h) {
		tmp := filepath.Join(root, "tmp")
		for _, name := range subdirs(tmp) {
			if name == "bin" {
				continue
			}
			dir := filepath.Join(tmp, name)
			// Resolved once, and only for a directory that yields a session.
			project := sync.OnceValue(func() string { return geminiProjectPath(sc, dir, name, projects) })
			deps := append([]string{filepath.Join(dir, ".project_root")}, projectFiles...)
			chats := filepath.Join(dir, "chats")
			for _, p := range filesWithSuffix(chats, ".json", ".jsonl") {
				if !strings.HasPrefix(filepath.Base(p), "session-") {
					continue
				}
				mod, ok := fresh(p)
				if !ok {
					continue
				}
				id := geminiChatID(sc, p)
				// Upgrades and resumes copy a session rather than move it;
				// the newest copy is the most complete.
				if c, dup := mains[id]; id == "" || (dup && !mod.After(c.mod)) {
					continue
				}
				mains[id] = chat{p, project, deps, mod}
			}
			for _, parent := range subdirs(chats) {
				for _, p := range filesWithSuffix(filepath.Join(chats, parent), ".json", ".jsonl") {
					if _, ok := fresh(p); ok {
						subs[parent] = append(subs[parent], p)
					}
				}
			}
			if p := filepath.Join(dir, "logs.json"); func() bool { _, ok := fresh(p); return ok }() {
				logs = append(logs, logFile{p, project, deps})
			}
		}
	}

	var srcs []*source
	for _, id := range slices.Sorted(maps.Keys(mains)) {
		c := mains[id]
		if src, ok := newSource(a, sc, id, append([]string{c.path}, subs[id]...), cutoff); ok {
			src.extra = geminiSource{project: c.project()}
			src.deps = c.deps
			srcs = append(srcs, src)
		}
	}
	for _, l := range logs {
		file := &geminiLogFile{sc: sc, path: l.path}
		n := 0
		for _, ls := range file.index() {
			if _, inChats := mains[ls.id]; inChats || (!cutoff.IsZero() && !ls.last.IsZero() && ls.last.Before(cutoff)) {
				continue
			}
			if n == maxGeminiLogSessions {
				break
			}
			if src, ok := newSource(a, sc, ls.id, []string{l.path}, time.Time{}); ok {
				src.extra = geminiSource{project: l.project(), logs: file}
				src.deps = l.deps
				srcs = append(srcs, src)
				n++
			}
		}
	}
	return srcs
}

var geminiChatIDs = newDerivedCache[string]()

// geminiChatID reads a chat file's session id. A legacy .json chat has to be
// parsed whole for it, so the id is kept while the file is unchanged.
func geminiChatID(sc *scope, path string) string {
	return geminiChatIDs.get(path, func() string {
		discoveryReads.Add(1)
		var meta struct {
			SessionID string `json:"sessionId"`
		}
		if strings.HasSuffix(path, ".jsonl") {
			firstRecord(sc, path, &meta)
		} else {
			readJSON(sc, path, &meta)
		}
		return meta.SessionID
	})
}

// geminiProjectIndex maps project slugs to paths, from projects.json.
type geminiProjectIndex struct {
	bySlug map[string]string
	paths  []string
}

func geminiProjects(sc *scope) geminiProjectIndex {
	idx := geminiProjectIndex{bySlug: map[string]string{}}
	for _, root := range geminiRoots(sc.home) {
		var pj struct {
			Projects map[string]string `json:"projects"`
		}
		readJSON(sc, filepath.Join(root, "projects.json"), &pj)
		for path, slug := range pj.Projects {
			idx.bySlug[slug] = path
			idx.paths = append(idx.paths, path)
		}
	}
	return idx
}

// geminiProjectPath recovers the project a tmp/<name> directory belongs to:
// from its .project_root file, from projects.json, or, for directories named
// by the hash of the path (before v0.29), by hashing the known paths.
func geminiProjectPath(sc *scope, dir, name string, idx geminiProjectIndex) string {
	if p := readText(sc, filepath.Join(dir, ".project_root"), maxFieldBytes); p != "" {
		return p
	}
	if p, ok := idx.bySlug[name]; ok {
		return p
	}
	for _, p := range idx.paths {
		sum := sha256.Sum256([]byte(p))
		if hex.EncodeToString(sum[:]) == name {
			return p
		}
	}
	return ""
}

type geminiToolCall struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	DisplayName   string          `json:"displayName"`
	Args          json.RawMessage `json:"args"`
	Status        string          `json:"status"`
	Result        json.RawMessage `json:"result"`
	ResultDisplay json.RawMessage `json:"resultDisplay"`
}

type geminiMessage struct {
	dead      bool            // removed by a patch; dropped when the replay ends
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Model     string          `json:"model"`
	Content   json.RawMessage `json:"content"`
	Thoughts  []struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
	} `json:"thoughts"`
	Tokens *struct {
		Input  int64 `json:"input"`
		Output int64 `json:"output"`
		Cached int64 `json:"cached"`
	} `json:"tokens"`
	ToolCalls []geminiToolCall `json:"toolCalls"`
}

type geminiConversation struct {
	SessionID   string          `json:"sessionId"`
	StartTime   string          `json:"startTime"`
	LastUpdated string          `json:"lastUpdated"`
	Kind        string          `json:"kind"`
	Messages    []geminiMessage `json:"messages"`

	ids map[string]int // message id -> index, built for replays
	// order is the last ordering a patch asked for, applied once at the end.
	order []string
}

// mergeMeta copies the metadata set in o.
func (c *geminiConversation) mergeMeta(o geminiConversation) {
	c.SessionID = cmp.Or(o.SessionID, c.SessionID)
	c.StartTime = cmp.Or(o.StartTime, c.StartTime)
	c.LastUpdated = cmp.Or(o.LastUpdated, c.LastUpdated)
	c.Kind = cmp.Or(o.Kind, c.Kind)
}

func (c *geminiConversation) index(id string) int {
	if c.ids == nil {
		c.reindex()
	}
	if i, ok := c.ids[id]; ok {
		return i
	}
	return -1
}

func (c *geminiConversation) reindex() {
	c.ids = make(map[string]int, len(c.Messages))
	for i, m := range c.Messages {
		c.ids[m.ID] = i
	}
}

// loadGeminiConversation reads a chat file, replaying a .jsonl one. Every
// replay record costs time proportional to its own size, never to the
// conversation's, so a file of patches can't pin the daemon.
func loadGeminiConversation(sc *scope, path string) (geminiConversation, bool) {
	var c geminiConversation
	if !strings.HasSuffix(path, ".jsonl") {
		return c, readJSON(sc, path, &c)
	}
	ok := false
	_ = scanJSONL(sc, path, func(line []byte) bool {
		var probe struct {
			ID        *string         `json:"id"`
			Type      string          `json:"type"`
			SessionID string          `json:"sessionId"`
			RewindTo  *string         `json:"$rewindTo"`
			Set       json.RawMessage `json:"$set"`
			Patch     json.RawMessage `json:"$patch"`
		}
		if json.Unmarshal(line, &probe) != nil {
			return true
		}
		ok = true
		switch {
		case probe.RewindTo != nil:
			// Drop that message and everything after it.
			if i := c.index(*probe.RewindTo); i >= 0 {
				for _, m := range c.Messages[i:] {
					delete(c.ids, m.ID)
				}
				c.Messages = c.Messages[:i]
			}
		case probe.Set != nil:
			var set struct {
				geminiConversation
				Messages *[]geminiMessage `json:"messages"`
			}
			if json.Unmarshal(probe.Set, &set) == nil {
				c.mergeMeta(set.geminiConversation)
				if set.Messages != nil {
					c.Messages = *set.Messages
					c.reindex()
				}
			}
		case probe.Patch != nil:
			c.patch(probe.Patch)
		case probe.ID != nil && *probe.ID != "" && probe.Type != "":
			var m geminiMessage
			if json.Unmarshal(line, &m) == nil {
				// The same message is re-appended as it updates.
				if i := c.index(m.ID); i >= 0 {
					c.Messages[i] = m
				} else if len(c.Messages) < maxGeminiMessages {
					c.Messages = append(c.Messages, m)
					c.ids[m.ID] = len(c.Messages) - 1
				}
			}
		case probe.SessionID != "":
			var meta geminiConversation
			if json.Unmarshal(line, &meta) == nil {
				c.mergeMeta(meta)
			}
		}
		return true
	})
	c.finish()
	return c, ok
}

// finish applies what the replay deferred: removals and the final order.
func (c *geminiConversation) finish() {
	kept := c.Messages[:0]
	for _, m := range c.Messages {
		if !m.dead {
			kept = append(kept, m)
		}
	}
	c.Messages = kept
	if len(c.order) > 0 {
		rank := make(map[string]int, len(c.order))
		for i, id := range c.order {
			rank[id] = i
		}
		slices.SortStableFunc(c.Messages, func(a, b geminiMessage) int {
			ra, aok := rank[a.ID]
			rb, bok := rank[b.ID]
			switch {
			case aok && !bok:
				return -1
			case bok && !aok:
				return 1
			}
			return cmp.Compare(ra, rb)
		})
	}
	c.ids = nil
}

type geminiMessagePatch struct {
	ID        string          `json:"id"`
	Content   json.RawMessage `json:"content"`
	ToolCalls []struct {
		ID     string          `json:"id"`
		Result json.RawMessage `json:"result"`
	} `json:"toolCalls"`
}

func (c *geminiConversation) patch(raw json.RawMessage) {
	var p struct {
		geminiMessagePatch
		Updates   []geminiMessagePatch `json:"updates"`
		RemoveIDs []string             `json:"removeIds"`
		OrderIDs  []string             `json:"orderIds"`
	}
	if json.Unmarshal(raw, &p) != nil {
		return
	}
	for _, u := range append([]geminiMessagePatch{p.geminiMessagePatch}, p.Updates...) {
		i := c.index(u.ID)
		if u.ID == "" || i < 0 {
			continue
		}
		if u.Content != nil {
			c.Messages[i].Content = u.Content
		}
		for _, tp := range u.ToolCalls {
			for j := range c.Messages[i].ToolCalls {
				if c.Messages[i].ToolCalls[j].ID == tp.ID {
					c.Messages[i].ToolCalls[j].Result = tp.Result
				}
			}
		}
	}
	for _, id := range p.RemoveIDs {
		if i := c.index(id); i >= 0 {
			c.Messages[i].dead = true
			delete(c.ids, id)
		}
	}
	if len(p.OrderIDs) > 0 {
		c.order = p.OrderIDs
	}
}

func parseGeminiCLI(src *source, r *recorder) {
	s := r.s
	ext, _ := src.extra.(geminiSource)
	s.ProjectPath = ext.project
	if ext.logs != nil {
		parseGeminiLogs(src, ext.logs.load()[src.id], r)
		return
	}
	for i, path := range src.files {
		sub := i > 0
		c, ok := loadGeminiConversation(src.sc, path)
		if !ok {
			continue
		}
		r.file(path)
		r.seen = true
		if !sub {
			s.observe(parseTime(c.StartTime))
			s.observe(parseTime(c.LastUpdated))
		}
		for _, m := range c.Messages {
			if r.done() {
				break
			}
			geminiMessageRows(r, m, sub)
		}
	}
}

func geminiMessageRows(r *recorder, m geminiMessage, sub bool) {
	s := r.s
	ts := parseTime(m.Timestamp)
	s.observe(ts)
	switch m.Type {
	case "user":
		// Since v0.44 tool results are also recorded as user messages; they
		// are already shown with their calls.
		if geminiHasFunctionResponse(m.Content) {
			return
		}
		role := roleSystem
		if !sub {
			s.UserMessages++
			role = roleUser
		}
		r.message(ts, "", "")
		r.text(role, blockText, func() string { return textOf(m.Content) })
	case "gemini":
		s.AssistantMessages++
		if m.Model != "" && !sub {
			s.Model = m.Model
		}
		// cached is part of input.
		if t := m.Tokens; t != nil {
			s.InputTokens += t.Input - t.Cached
			s.CacheReadTokens += t.Cached
			s.OutputTokens += t.Output
		}
		r.message(ts, m.Model, "")
		for _, th := range m.Thoughts {
			r.text(roleAssistant, blockThinking, func() string {
				if th.Subject == "" {
					return th.Description
				}
				return th.Subject + ": " + th.Description
			})
		}
		r.text(roleAssistant, blockText, func() string { return textOf(m.Content) })
		for _, tc := range m.ToolCalls {
			r.toolUse(tc.Name, tc.ID, func() string { return rawString(tc.Args) }, func() (string, string) {
				if server, tool, ok := geminiMCP(tc.DisplayName); ok {
					return classifyMCP(server, tool, parseArgs(tc.Args))
				}
				return classifyNative(agentGeminiCLI, tc.Name, parseArgs(tc.Args))
			})
			switch tc.Status {
			case "success", "error", "cancelled":
				r.toolResult(tc.ID, func(Message) string { return geminiDecision(tc) }, func() string { return geminiResult(tc) })
			}
		}
	}
	// info, error and warning messages are the CLI's own notices.
}

func geminiHasFunctionResponse(content json.RawMessage) bool {
	var parts []struct {
		FunctionResponse json.RawMessage `json:"functionResponse"`
	}
	if json.Unmarshal(content, &parts) != nil {
		return false
	}
	for _, p := range parts {
		if p.FunctionResponse != nil {
			return true
		}
	}
	return false
}

var geminiMCPDisplay = regexp.MustCompile(`^(.+) \((.+) MCP Server\)$`)

// geminiMCP reads an MCP call's server and tool from its display name, the
// only place every version records them unambiguously.
func geminiMCP(displayName string) (server, tool string, ok bool) {
	m := geminiMCPDisplay.FindStringSubmatch(displayName)
	if m == nil {
		return "", "", false
	}
	return m[2], m[1], true
}

// geminiResult is a tool call's output: the function responses sent back
// to the model, else what the CLI displayed.
func geminiResult(tc geminiToolCall) string {
	var parts []struct {
		Text             string `json:"text"`
		FunctionResponse *struct {
			Response map[string]any `json:"response"`
		} `json:"functionResponse"`
	}
	var out []string
	if json.Unmarshal(tc.Result, &parts) == nil {
		for _, p := range parts {
			if p.Text != "" {
				out = append(out, p.Text)
			}
			if p.FunctionResponse == nil {
				continue
			}
			for _, k := range []string{"output", "error"} {
				switch v := p.FunctionResponse.Response[k].(type) {
				case nil:
				case string:
					out = append(out, v)
				default:
					if b, err := json.Marshal(v); err == nil {
						out = append(out, string(b))
					}
				}
			}
		}
	}
	if len(out) == 0 {
		return cmp.Or(unquote(tc.Result), unquote(tc.ResultDisplay))
	}
	return strings.Join(out, "\n")
}

var geminiExitCode = regexp.MustCompile(`(?m)^Exit Code: (-?\d+)`)

// geminiDecision maps a tool call's final status to its outcome. Failed
// shell commands are recorded as successful; only the output says.
func geminiDecision(tc geminiToolCall) string {
	output, display := geminiResult(tc), unquote(tc.ResultDisplay)
	switch tc.Status {
	case "cancelled":
		for _, t := range []string{output, display} {
			if strings.HasPrefix(strings.TrimSpace(t), "User denied execution.") {
				return decisionDenied
			}
		}
		return decisionError
	case "error":
		return decisionError
	}
	// The shell tool ends its output with the exit code; a command can print
	// the same words earlier, so the last one counts.
	if ms := geminiExitCode.FindAllStringSubmatch(output, -1); len(ms) > 0 && ms[len(ms)-1][1] != "0" {
		return decisionError
	}
	return decisionAllowed
}

func parseGeminiLogs(src *source, entries []geminiLogEntry, r *recorder) {
	r.file(src.files[0])
	for _, e := range entries {
		if e.Type != "user" || r.done() {
			continue
		}
		ts := parseTime(e.Timestamp)
		r.s.observe(ts)
		r.s.UserMessages++
		r.message(ts, "", "")
		r.text(roleUser, blockText, func() string { return e.Message })
	}
}
