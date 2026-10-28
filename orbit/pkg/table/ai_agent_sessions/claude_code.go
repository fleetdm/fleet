package ai_agent_sessions

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// Claude Code keeps one transcript per session in
// ~/.claude/projects/<project-slug>/<session-id>.jsonl, and each sub-agent it
// spawns in <project-slug>/<session-id>/subagents/[…/]agent-<id>.jsonl.
// Older versions wrote sub-agents flat, as <project-slug>/agent-<id>.jsonl,
// naming their session only inside; those are merged into it too.
var claudeCode = &agent{
	name:     agentClaudeCode,
	discover: discoverClaudeCode,
	parse:    parseClaudeCode,
	account:  claudeAccount,
}

func discoverClaudeCode(a *agent, sc *scope, cutoff time.Time) []*source {
	projects := filepath.Join(sc.home.Dir, ".claude", "projects")
	var srcs []*source
	for _, project := range subdirs(projects) {
		dir := filepath.Join(projects, project)
		var sessions []string
		flat := map[string][]string{}
		for _, path := range filesWithSuffix(dir, ".jsonl") {
			if strings.HasPrefix(filepath.Base(path), "agent-") {
				// Only a fresh file is worth opening to find its parent.
				if fi, ok := regularFile(path); ok && (cutoff.IsZero() || !fi.ModTime().Before(cutoff)) {
					if parent := claudeFlatAgentParent(sc, path); parent != "" {
						flat[parent] = append(flat[parent], path)
					}
				}
				continue
			}
			sessions = append(sessions, path)
		}
		for _, path := range sessions {
			id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
			files := append([]string{path}, claudeSubagentFiles(filepath.Join(dir, id, "subagents"))...)
			files = append(files, flat[id]...)
			if src, ok := newSource(a, sc, id, files, cutoff); ok {
				srcs = append(srcs, src)
			}
		}
	}
	return srcs
}

var claudeFlatParents = newDerivedCache[string]()

// claudeFlatAgentParent is the session a flat sub-agent transcript belongs
// to, from its first record.
func claudeFlatAgentParent(sc *scope, path string) string {
	return claudeFlatParents.get(path, func() string {
		discoveryReads.Add(1)
		var head struct {
			SessionID string `json:"sessionId"`
		}
		firstRecord(sc, path, &head)
		return head.SessionID
	})
}

// claudeSubagentFiles lists sub-agent transcripts, which may nest a few
// levels. WalkDir doesn't follow links.
func claudeSubagentFiles(dir string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil //nolint:nilerr // an unreadable subtree shouldn't stop the rest
		case len(out) >= maxFilesPerWalk:
			return fs.SkipAll
		case d.IsDir() && walkDepth(dir, p) > maxWalkDepth:
			return fs.SkipDir
		case d.Type().IsRegular() && strings.HasSuffix(d.Name(), ".jsonl"):
			out = append(out, p)
		}
		return nil
	})
	return out
}

// claudeHead is what every record says about itself.
type claudeHead struct {
	Type           string `json:"type"`
	Timestamp      string `json:"timestamp"`
	CWD            string `json:"cwd"`
	Version        string `json:"version"`
	GitBranch      string `json:"gitBranch"`
	ToolDenialKind string `json:"toolDenialKind"`
	IsSidechain    bool   `json:"isSidechain"`
	IsMeta         bool   `json:"isMeta"`
	IsCompact      bool   `json:"isCompactSummary"`
}

// claudeRecord is a record whose bodies decode as B: the raw JSON for the
// messages table, or claudeBody for the sessions table, which never needs
// the text and so doesn't copy it.
type claudeRecord[B any] struct {
	claudeHead
	Content B `json:"content"` // system records
	Message struct {
		ID      string       `json:"id"`
		Model   string       `json:"model"`
		Content B            `json:"content"`
		Usage   *claudeUsage `json:"usage"`
	} `json:"message"`
}

type claudeUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
}

type claudeBlock struct {
	Type      string          `json:"type"`
	Text      json.RawMessage `json:"text"`
	Thinking  json.RawMessage `json:"thinking"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	IsError   bool            `json:"is_error"`
	Content   json.RawMessage `json:"content"`
}

// claudeBody is a decoded message body: a plain string, or typed blocks.
type claudeBody struct {
	// text is the string body, kept only by claudeFullBody.
	text     json.RawMessage
	isText   bool
	injected bool
	blocks   []claudeBlock
}

// UnmarshalJSON decodes only the block fields the sessions table needs, and
// whether a string body is one of the CLI's own records; no text is kept.
func (b *claudeBody) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] == '"' {
		b.isText, b.injected = true, claudeInjectedText(raw)
		return nil
	}
	var heads []struct {
		Type      string `json:"type"`
		ID        string `json:"id"`
		Name      string `json:"name"`
		ToolUseID string `json:"tool_use_id"`
		IsError   bool   `json:"is_error"`
	}
	_ = json.Unmarshal(raw, &heads)
	b.blocks = make([]claudeBlock, len(heads))
	for i, h := range heads {
		b.blocks[i] = claudeBlock{Type: h.Type, ID: h.ID, Name: h.Name, ToolUseID: h.ToolUseID, IsError: h.IsError}
	}
	return nil
}

// claudeFullBody decodes a body with its text, for the messages table.
func claudeFullBody(raw json.RawMessage) claudeBody {
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] == '"' {
		return claudeBody{text: raw, isText: true, injected: claudeInjectedText(raw)}
	}
	var b claudeBody
	_ = json.Unmarshal(raw, &b.blocks)
	return b
}

// claudeInjectedPrefixes open user records Claude Code writes itself: slash
// commands and their output, and `!` shell commands. They look like prompts
// but the model never saw them as such.
var claudeInjectedPrefixes = []string{
	`"<command-message>`, `"<command-name>`, `"<local-command-stdout>`, `"<local-command-caveat>`,
	`"<bash-input>`, `"<bash-stdout>`, `"<bash-stderr>`,
}

func claudeInjectedText(raw []byte) bool {
	for _, p := range claudeInjectedPrefixes {
		if bytes.HasPrefix(raw, []byte(p)) {
			return true
		}
	}
	return false
}

func parseClaudeCode(src *source, r *recorder) {
	s := r.s
	// An API response is written as one record per content block, each
	// repeating the response's usage, so usage is counted once per id.
	usage := map[string]claudeUsage{}
	for i, path := range src.files {
		r.file(path)
		_ = scanJSONL(src.sc, path, func(line []byte) bool {
			if r.done() {
				return false
			}
			var head struct {
				Type string `json:"type"`
			}
			if r.skipNext() {
				if json.Unmarshal(line, &head) == nil && isClaudeMessage(head.Type) {
					r.next()
				}
				return true
			}
			var (
				rec     claudeRecord[json.RawMessage]
				body    claudeBody
				sysBody claudeBody
				ok      bool
			)
			if r.rows() {
				if ok = json.Unmarshal(line, &rec) == nil; ok {
					body, sysBody = claudeFullBody(rec.Message.Content), claudeFullBody(rec.Content)
				}
			} else {
				var slim claudeRecord[claudeBody]
				if ok = json.Unmarshal(line, &slim) == nil; ok {
					rec.claudeHead, rec.Message.ID, rec.Message.Model, rec.Message.Usage =
						slim.claudeHead, slim.Message.ID, slim.Message.Model, slim.Message.Usage
					body = slim.Message.Content
				}
			}
			if !ok {
				// Count it as the skip path above would, so a message's index
				// doesn't depend on the page asked for.
				if json.Unmarshal(line, &head) == nil && isClaudeMessage(head.Type) {
					r.next()
				}
				return true
			}
			r.seen = true
			// Sub-agent turns: their own files, or older versions' records in
			// the session's file.
			sub := i > 0 || rec.IsSidechain
			ts := parseTime(rec.Timestamp)
			s.observe(ts)
			if !sub {
				// The first directory is where the session started; later ones
				// may be wherever the agent cd'd into.
				if s.ProjectPath == "" {
					s.ProjectPath = rec.CWD
				}
				if rec.GitBranch != "" {
					s.GitBranch = rec.GitBranch
				}
				if rec.Version != "" {
					s.AgentVersion = rec.Version
				}
			}

			switch rec.Type {
			case "user":
				r.message(ts, "", rec.CWD)
				claudeUser(r, rec.claudeHead, body, sub)
			case "assistant":
				model := rec.Message.Model
				if strings.HasPrefix(model, "<") { // "<synthetic>" isn't a model
					model = ""
				}
				if model != "" && !sub {
					s.Model = model
				}
				if id := rec.Message.ID; id == "" {
					s.AssistantMessages++
					if u := rec.Message.Usage; u != nil {
						addClaudeUsage(s, *u)
					}
				} else {
					u, dup := usage[id]
					if !dup {
						s.AssistantMessages++
					}
					if rec.Message.Usage != nil {
						u = *rec.Message.Usage
					}
					usage[id] = u
				}
				r.message(ts, model, rec.CWD)
				claudeAssistant(r, body)
			case "system":
				r.message(ts, "", rec.CWD)
				r.text(roleSystem, blockText, func() string { return textOf(sysBody.text) })
			}
			// Other record types (attachments, queued prompts, titles) aren't
			// part of the conversation.
			return true
		})
	}
	for _, u := range usage {
		addClaudeUsage(s, u)
	}
}

func isClaudeMessage(t string) bool { return t == "user" || t == "assistant" || t == "system" }

func addClaudeUsage(s *Session, u claudeUsage) {
	s.InputTokens += u.InputTokens
	s.OutputTokens += u.OutputTokens
	s.CacheReadTokens += u.CacheReadInputTokens
	s.CacheWriteTokens += u.CacheCreationInputTokens
}

// claudeUser handles a user record. A sub-agent's "user" is the parent agent,
// a turn of only tool results is the agent loop, and some records are the
// CLI's own bookkeeping; none of those is a prompt.
func claudeUser(r *recorder, head claudeHead, body claudeBody, sub bool) {
	if body.isText {
		role := roleUser
		if sub || head.IsMeta || head.IsCompact || body.injected {
			role = roleSystem
		} else {
			r.s.UserMessages++
		}
		r.text(role, blockText, func() string { return textOf(body.text) })
		return
	}

	prompt := false
	for _, b := range body.blocks {
		if b.Type != "tool_result" {
			prompt = true
		}
	}
	role := roleSystem
	if prompt && !sub {
		r.s.UserMessages++
		role = roleUser
	}
	for _, b := range body.blocks {
		switch b.Type {
		case "text":
			r.text(role, blockText, func() string { return textOf(b.Text) })
		case "tool_result":
			r.toolResult(b.ToolUseID, func(call Message) string {
				return claudeDecision(head.ToolDenialKind, b, call.ToolName)
			}, func() string { return textOf(b.Content) })
		default: // images and other blocks have no row, but keep their index
			r.nextBlock()
		}
	}
}

func claudeAssistant(r *recorder, body claudeBody) {
	if body.isText {
		r.text(roleAssistant, blockText, func() string { return textOf(body.text) })
		return
	}
	for _, b := range body.blocks {
		switch b.Type {
		case "text":
			r.text(roleAssistant, blockText, func() string { return textOf(b.Text) })
		case "thinking":
			r.text(roleAssistant, blockThinking, func() string { return textOf(b.Thinking) })
		case "tool_use":
			r.toolUse(b.Name, b.ID, func() string { return rawString(b.Input) },
				func() (string, string) { return classify(agentClaudeCode, b.Name, b.Input) })
		default:
			r.nextBlock()
		}
	}
}

// claudeDecision maps a tool result to the call's outcome. Every refusal and
// failure is marked is_error; newer transcripts say which kind of refusal in
// toolDenialKind, older ones only in the result text.
func claudeDecision(denialKind string, b claudeBlock, tool string) string {
	switch {
	case denialKind == "user-rejected" || denialKind == "permission-rule" || strings.HasPrefix(denialKind, "automode-"):
		return decisionDenied
	case denialKind == "interrupted" || denialKind == "cancelled":
		return decisionError
	case !b.IsError:
		return decisionAllowed
	case denialKind != "" || strings.HasPrefix(tool, "mcp__"):
		// An MCP server chooses its own error text, and could word it like a
		// refusal.
		return decisionError
	}
	first, _, _ := strings.Cut(textOf(b.Content), "\n")
	switch {
	case strings.HasPrefix(first, "The user doesn't want to proceed"),
		strings.HasPrefix(first, "The user doesn't want to take this action"),
		strings.HasPrefix(first, "Permission to use"),
		strings.HasPrefix(first, "Permission for this action"),
		strings.HasPrefix(first, "Hook PreToolUse:") && strings.Contains(first, "denied this tool"),
		strings.HasPrefix(first, "Blocked by hook"),
		strings.HasPrefix(first, "Permission denied by hook"):
		return decisionDenied
	}
	return decisionError
}
