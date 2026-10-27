package ai_agent_sessions

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Codex CLI writes one rollout per thread to
// $CODEX_HOME/sessions/YYYY/MM/DD/rollout-<time>-<uuid>.jsonl (and moves old
// ones to archived_sessions/). Records are {timestamp, type, payload}; the
// oldest rollouts wrote items at the top level instead. Sub-agents get their
// own rollouts, merged here into their root session; a fork is a session of
// its own whose rollout starts with a copy of its parent's history.
var codexCLI = &agent{
	name:     agentCodexCLI,
	discover: discoverCodexCLI,
	parse:    parseCodexCLI,
	account:  codexAccount,
}

var codexUUID = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

type codexMeta struct {
	ID             string          `json:"id"`
	SessionID      string          `json:"session_id"`
	ParentThreadID string          `json:"parent_thread_id"`
	ForkedFromID   string          `json:"forked_from_id"`
	Source         json.RawMessage `json:"source"`
	Timestamp      string          `json:"timestamp"`
	CWD            string          `json:"cwd"`
	CLIVersion     string          `json:"cli_version"`
	ModelProvider  string          `json:"model_provider"`
	Git            struct {
		Branch        string `json:"branch"`
		RepositoryURL string `json:"repository_url"`
	} `json:"git"`
}

// codexSession groups a session's rollouts; main is false when only
// sub-agent rollouts were found.
type codexSession struct {
	main bool
}

func discoverCodexCLI(a *agent, sc *scope, cutoff time.Time) []*source {
	root := filepath.Join(sc.home.Dir, ".codex")
	type rollout struct {
		path string
		sub  bool
	}
	groups := map[string][]rollout{}
	names := map[string]set{} // key -> rollout file names seen
	var order []string
	files := 0
	// Rollouts older than the cutoff are only remembered by the thread id in
	// their name, in case a newer sub-agent rollout belongs to them.
	stale := map[string]string{}
	for _, dir := range []string{filepath.Join(root, "sessions"), filepath.Join(root, "archived_sessions")} {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return nil //nolint:nilerr // an unreadable subtree shouldn't stop the rest
			case files >= maxFilesPerWalk:
				return fs.SkipAll
			case d.IsDir():
				if walkDepth(dir, p) > maxWalkDepth {
					return fs.SkipDir
				}
				return nil
			case !d.Type().IsRegular() || !strings.HasSuffix(d.Name(), ".jsonl"):
				return nil
			}
			fi, ok := regularFile(p)
			if !ok {
				return nil
			}
			files++
			if !cutoff.IsZero() && fi.ModTime().Before(cutoff) {
				if id := codexUUID.FindString(d.Name()); id != "" {
					stale[id] = p
				}
				return nil
			}
			key, sub := codexSessionKey(sc, p)
			if _, ok := groups[key]; !ok {
				order = append(order, key)
				names[key] = set{}
			}
			// The same rollout can be in both sessions/ and archived_sessions/.
			if names[key].has(d.Name()) {
				return nil
			}
			names[key][d.Name()] = struct{}{}
			groups[key] = append(groups[key], rollout{p, sub})
			return nil
		})
	}

	var srcs []*source
	for _, key := range order {
		rs := groups[key]
		if len(rs) == 0 {
			continue
		}
		sort.SliceStable(rs, func(i, j int) bool { return !rs[i].sub && rs[j].sub })
		if rs[0].sub {
			if p, ok := stale[key]; ok {
				rs = append([]rollout{{p, false}}, rs...)
			}
		}
		files := make([]string, len(rs))
		for i, r := range rs {
			files[i] = r.path
		}
		if src, ok := newSource(a, sc, key, files, cutoff); ok {
			src.extra = codexSession{main: !rs[0].sub}
			srcs = append(srcs, src)
		}
	}
	return srcs
}

type codexKey struct {
	key string
	sub bool
}

var codexKeys = newDerivedCache[codexKey]()

// codexSessionKey reads a rollout's first record to find which session it
// belongs to, and whether it is a sub-agent's.
func codexSessionKey(sc *scope, path string) (key string, sub bool) {
	k := codexKeys.get(path, func() codexKey {
		discoveryReads.Add(1)
		var first struct {
			Type    string    `json:"type"`
			ID      string    `json:"id"`
			Payload codexMeta `json:"payload"`
		}
		name := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		fromName := firstNonEmpty(codexUUID.FindString(name), name)
		if !firstRecord(sc, path, &first) {
			return codexKey{fromName, false}
		}
		switch m := first.Payload; {
		case first.Type == "session_meta" && (m.ParentThreadID != "" || bytes.Contains(m.Source, []byte("subagent"))):
			return codexKey{firstNonEmpty(m.SessionID, m.ParentThreadID, m.ID, fromName), true}
		case first.Type == "session_meta":
			return codexKey{firstNonEmpty(m.ID, fromName), false}
		case first.Type == "" && first.ID != "":
			return codexKey{first.ID, false}
		}
		return codexKey{fromName, false}
	})
	return k.key, k.sub
}

type codexLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Payload   json.RawMessage `json:"payload"`
}

type codexItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role"`
	Name      string          `json:"name"`
	Namespace string          `json:"namespace"`
	CallID    string          `json:"call_id"`
	Content   json.RawMessage `json:"content"`
	Summary   json.RawMessage `json:"summary"`
	Arguments json.RawMessage `json:"arguments"`
	Input     json.RawMessage `json:"input"`
	Output    json.RawMessage `json:"output"`
	Action    json.RawMessage `json:"action"`
}

type codexUsage struct {
	Input      int64 `json:"input_tokens"`
	Cached     int64 `json:"cached_input_tokens"`
	CacheWrite int64 `json:"cache_write_input_tokens"`
	Output     int64 `json:"output_tokens"`
}

// minus subtracts o, never going below zero: a fork's totals and its seed
// may come from different kinds of record.
func (u codexUsage) minus(o codexUsage) codexUsage {
	return codexUsage{
		max(u.Input-o.Input, 0), max(u.Cached-o.Cached, 0),
		max(u.CacheWrite-o.CacheWrite, 0), max(u.Output-o.Output, 0),
	}
}

// codexTokens follows one rollout's running totals. token_usage_record is
// preferred; token_count totals are reset to zero when the context window
// overflows, so they may never go backwards.
type codexTokens struct {
	record, count *codexUsage
}

func (t *codexTokens) addCount(u codexUsage) {
	if t.count == nil || u.Input+u.Output >= t.count.Input+t.count.Output {
		t.count = &u
	}
}

func (t *codexTokens) total() codexUsage {
	switch {
	case t.record != nil:
		return *t.record
	case t.count != nil:
		return *t.count
	}
	return codexUsage{}
}

// codexParser is the state of reading one rollout.
type codexParser struct {
	r      *recorder
	sub    bool
	flat   bool
	ts     time.Time
	model  string
	cwd    string
	tokens codexTokens
}

func parseCodexCLI(src *source, r *recorder) {
	s := r.s
	ext, _ := src.extra.(codexSession)
	var total codexUsage
	for i, path := range src.files {
		p := &codexParser{r: r, sub: i > 0 || !ext.main}
		var seed codexTokens
		var forkStart time.Time
		first := true
		r.file(path)
		_ = scanJSONL(src.sc, path, func(line []byte) bool {
			if r.done() {
				return false
			}
			var rec codexLine
			if json.Unmarshal(line, &rec) != nil {
				return true
			}
			p.ts = parseTime(rec.Timestamp)
			if first {
				first = false
				switch rec.Type {
				case "session_meta":
					var m codexMeta
					_ = json.Unmarshal(rec.Payload, &m)
					if !p.sub {
						s.ProjectPath, s.AgentVersion = m.CWD, m.CLIVersion
						s.GitBranch, s.GitRepository = m.Git.Branch, sanitizeRepoURL(m.Git.RepositoryURL)
						if m.ModelProvider != "" && m.ModelProvider != "openai" {
							s.providerType = m.ModelProvider
						}
					}
					p.cwd = m.CWD
					if m.ForkedFromID != "" {
						forkStart = firstNonZero(parseTime(m.Timestamp), p.ts)
					}
					r.seen = true
					s.observe(p.ts)
					return true
				case "":
					p.flat = true
					r.seen = true
					s.observe(p.ts)
					return true
				}
			}
			// A fork's rollout opens with the parent's history, copied with
			// its original times; only what follows is the fork's own. The
			// parent's totals up to there are what the fork started from.
			if !forkStart.IsZero() && !p.ts.IsZero() && p.ts.Before(forkStart) {
				codexTokenRecord(&seed, rec)
				return true
			}
			r.seen = true
			s.observe(p.ts)
			if r.skipNext() {
				p.skip(rec, line)
				return true
			}
			switch rec.Type {
			case "turn_context":
				var tc struct {
					Model string `json:"model"`
					CWD   string `json:"cwd"`
				}
				_ = json.Unmarshal(rec.Payload, &tc)
				p.model, p.cwd = firstNonEmpty(tc.Model, p.model), firstNonEmpty(tc.CWD, p.cwd)
				if tc.Model != "" && !p.sub {
					s.Model = tc.Model
				}
			case "response_item":
				var it codexItem
				if json.Unmarshal(rec.Payload, &it) == nil {
					p.item(it)
				} else {
					p.skip(rec, line) // counted as the skip path would
				}
			case "event_msg":
				if !p.event(rec.Payload) {
					p.skip(rec, line)
				}
			case "token_usage_record", "session_meta", "compacted", "turn_aborted":
				codexTokenRecord(&p.tokens, rec)
			default:
				var it codexItem
				if !p.flat {
					break
				}
				if json.Unmarshal(line, &it) == nil {
					p.item(it)
				} else {
					p.skip(rec, line)
				}
			}
			return true
		})
		u := p.tokens.total()
		if seed.record != nil || seed.count != nil {
			u = u.minus(seed.total())
		}
		total.Input += u.Input
		total.Cached += u.Cached
		total.CacheWrite += u.CacheWrite
		total.Output += u.Output
	}
	// input_tokens includes the cached ones.
	s.InputTokens = total.Input - total.Cached
	s.CacheReadTokens = total.Cached
	s.CacheWriteTokens = total.CacheWrite
	s.OutputTokens = total.Output
}

func firstNonZero(ts ...time.Time) time.Time {
	for _, t := range ts {
		if !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

// codexTokenRecord takes the totals from a token_usage_record or token_count.
func codexTokenRecord(t *codexTokens, rec codexLine) {
	switch rec.Type {
	case "token_usage_record":
		var p struct {
			Thread *codexUsage `json:"thread_token_usage"`
		}
		if json.Unmarshal(rec.Payload, &p) == nil && p.Thread != nil {
			t.record = p.Thread
		}
	case "event_msg":
		var p struct {
			Type string `json:"type"`
			Info *struct {
				Total codexUsage `json:"total_token_usage"`
			} `json:"info"`
		}
		if json.Unmarshal(rec.Payload, &p) == nil && p.Type == "token_count" && p.Info != nil {
			t.addCount(p.Info.Total)
		}
	}
}

// skip counts a record before the requested page, decoding only what says
// whether it is a message, and keeps the model and directory current.
func (p *codexParser) skip(rec codexLine, line []byte) {
	var h struct {
		Type  string `json:"type"`
		Role  string `json:"role"`
		Model string `json:"model"`
		CWD   string `json:"cwd"`
		Item  *struct {
			Type string `json:"type"`
		} `json:"item"`
	}
	switch rec.Type {
	case "turn_context":
		if json.Unmarshal(rec.Payload, &h) == nil {
			p.model, p.cwd = firstNonEmpty(h.Model, p.model), firstNonEmpty(h.CWD, p.cwd)
		}
	case "response_item":
		if json.Unmarshal(rec.Payload, &h) == nil && codexItemIsMessage(h.Type, h.Role, p.flat) {
			p.r.next()
		}
	case "event_msg":
		if json.Unmarshal(rec.Payload, &h) == nil && codexEventIsMessage(h.Type, h.Item != nil && h.Item.Type == "UserMessage") {
			p.r.next()
		}
	default:
		if p.flat && json.Unmarshal(line, &h) == nil && codexItemIsMessage(h.Type, h.Role, p.flat) {
			p.r.next()
		}
	}
}

// codexItemIsMessage says whether a response item is a message of the
// conversation; item and skip must agree on it.
func codexItemIsMessage(typ, role string, flat bool) bool {
	switch typ {
	case "message":
		return role == "assistant" || (role == "user" && flat)
	case "reasoning", "function_call", "custom_tool_call", "local_shell_call", "web_search_call",
		"function_call_output", "custom_tool_call_output":
		return true
	}
	return false
}

func codexEventIsMessage(typ string, userItem bool) bool {
	return typ == "user_message" || (typ == "item_completed" && userItem)
}

// event handles an event_msg, reporting false when it couldn't be decoded.
func (p *codexParser) event(raw json.RawMessage) bool {
	var e struct {
		Type    string          `json:"type"`
		Message json.RawMessage `json:"message"`
		Item    *struct {
			Type    string          `json:"type"`
			Content json.RawMessage `json:"content"`
		} `json:"item"`
		Info *struct {
			Total codexUsage `json:"total_token_usage"`
		} `json:"info"`
	}
	if json.Unmarshal(raw, &e) != nil {
		return false
	}
	userItem := e.Item != nil && e.Item.Type == "UserMessage"
	switch {
	case !codexEventIsMessage(e.Type, userItem):
	case e.Type == "user_message": // legacy history mode
		p.prompt(func() string { return textOf(e.Message) })
	case userItem: // paginated history mode
		p.prompt(func() string { return textOf(e.Item.Content) })
	}
	if e.Type == "token_count" {
		if e.Info != nil {
			p.tokens.addCount(e.Info.Total)
		}
	}
	// agent_message duplicates the assistant's response_item.
	return true
}

// prompt records what the user typed. Codex also writes user-role
// response_items, but most of those are context it injects itself.
func (p *codexParser) prompt(text func() string) {
	role := roleSystem
	if !p.sub {
		p.r.s.UserMessages++
		role = roleUser
	}
	p.r.next()
	p.r.at(p.ts, "", p.cwd)
	p.r.text(role, blockText, text)
}

func (p *codexParser) item(it codexItem) {
	r := p.r
	if !codexItemIsMessage(it.Type, it.Role, p.flat) {
		// tool_search_call looks up tool schemas; like Cursor's
		// GetDynamicTools it isn't an event.
		return
	}
	switch it.Type {
	case "message":
		if it.Role == "assistant" {
			r.s.AssistantMessages++
			r.next()
			r.at(p.ts, p.model, p.cwd)
			r.text(roleAssistant, blockText, func() string { return textOf(it.Content) })
		} else {
			p.prompt(func() string { return textOf(it.Content) })
		}
	case "reasoning":
		r.next()
		r.at(p.ts, p.model, p.cwd)
		r.text(roleAssistant, blockThinking, func() string { return textOf(it.Summary) })
	case "function_call", "custom_tool_call":
		input := it.Arguments
		if it.Type == "custom_tool_call" {
			input = it.Input
		}
		// Arguments are a JSON document encoded as a string.
		text := firstNonEmpty(unquote(input), rawString(input))
		server, isMCP := strings.CutPrefix(it.Namespace, "mcp__")
		r.next()
		r.at(p.ts, p.model, p.cwd)
		r.toolUse(it.Name, it.CallID, func() string { return text }, func() (string, string) {
			if isMCP {
				return classifyMCP(strings.TrimSuffix(server, "__"), it.Name, parseArgs([]byte(text)))
			}
			return classify(agentCodexCLI, it.Name, []byte(text))
		})
	case "local_shell_call", "web_search_call":
		name := strings.TrimSuffix(it.Type, "_call")
		r.next()
		r.at(p.ts, p.model, p.cwd)
		r.toolUse(name, it.CallID, func() string { return rawString(it.Action) },
			func() (string, string) { return classifyNative(agentCodexCLI, name, parseArgs(it.Action)) })
	case "function_call_output", "custom_tool_call_output":
		r.next()
		r.at(p.ts, "", p.cwd)
		r.toolResult(it.CallID, func(call Message) string { return codexDecision(it.Output, call.Action == actionMCP) },
			func() string { return codexOutputText(it.Output) })
	}
}

// codexOutputText flattens a tool output: a string, content items, or an
// object carrying them.
func codexOutputText(raw json.RawMessage) string {
	if t := textOf(raw); t != "" {
		return t
	}
	var o struct {
		Content json.RawMessage `json:"content"`
		Output  json.RawMessage `json:"output"`
	}
	if json.Unmarshal(raw, &o) == nil {
		if t := firstNonEmpty(textOf(o.Content), textOf(o.Output)); t != "" {
			return t
		}
	}
	return rawString(raw)
}

var codexExitCode = regexp.MustCompile(`(?m)^(?:Process exited with code|Exit code:) (-?\d+)`)

// codexDecision maps a tool output to the call's outcome, from the fixed
// messages Codex writes for refusals and failures. An MCP server's output is
// its own text, which could be worded like a refusal, so only its structured
// error flags count.
func codexDecision(raw json.RawMessage, mcp bool) string {
	t := strings.TrimSpace(codexOutputText(raw))
	if mcp {
		return codexStructuredDecision(raw, t)
	}
	switch {
	case t == "exec command rejected by user", t == "patch rejected by user", t == "user rejected MCP tool call",
		strings.HasPrefix(t, "rejected by configuration"),
		strings.HasPrefix(t, "automatic approval review denied the action"):
		return decisionDenied
	case strings.HasPrefix(t, "aborted by user"):
		return decisionError
	}
	if m := codexExitCode.FindStringSubmatch(t); m != nil {
		if m[1] != "0" {
			return decisionError
		}
		return decisionAllowed
	}
	return codexStructuredDecision(raw, t)
}

// codexStructuredDecision reads the flags a tool output may carry: the
// oldest rollouts wrote {"output", "metadata": {"exit_code"}}, MCP results
// carry isError, and structured outputs a success flag.
func codexStructuredDecision(raw json.RawMessage, t string) string {
	for _, doc := range []string{t, rawString(raw)} {
		var o struct {
			Metadata *struct {
				ExitCode int `json:"exit_code"`
			} `json:"metadata"`
			IsError bool  `json:"isError"`
			Success *bool `json:"success"`
		}
		if json.Unmarshal([]byte(doc), &o) != nil {
			continue
		}
		if (o.Metadata != nil && o.Metadata.ExitCode != 0) || o.IsError || (o.Success != nil && !*o.Success) {
			return decisionError
		}
	}
	return decisionAllowed
}
