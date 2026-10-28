package ai_agent_sessions

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Cursor's CLI writes transcripts to
// ~/.cursor/projects/<slug>/agent-transcripts/<id>/<id>.jsonl (sub-agents in
// <id>/subagents/, older ones flat as <id>.jsonl). They hold no timestamps,
// model, tokens or tool results; a sidecar
// ~/.cursor/chats/<md5(cwd)>/<id>/meta.json has the times and directory.
// The chat's store.db would hold more, but its protobuf schema is Cursor's
// own and unpublished, so it is not read.
var cursorAgent = &agent{
	name:     agentCursor,
	discover: discoverCursor,
	parse:    parseCursor,
	account:  cursorAccount,
}

type cursorSource struct {
	slug string
	meta string
}

func discoverCursor(a *agent, sc *scope, cutoff time.Time) []*source {
	root := filepath.Join(sc.home.Dir, ".cursor")
	var metas map[string]string
	var srcs []*source
	for _, project := range subdirs(filepath.Join(root, "projects")) {
		dir := filepath.Join(root, "projects", project, "agent-transcripts")
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			var id string
			var files []string
			switch {
			case e.IsDir():
				id = e.Name()
				files = append([]string{filepath.Join(dir, id, id+".jsonl")},
					filesWithSuffix(filepath.Join(dir, id, "subagents"), ".jsonl")...)
			case e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".jsonl"):
				id = strings.TrimSuffix(e.Name(), ".jsonl")
				files = []string{filepath.Join(dir, e.Name())}
			default:
				continue
			}
			src, ok := newSource(a, sc, id, files, cutoff)
			if !ok {
				continue
			}
			if metas == nil {
				metas = cursorMetas(filepath.Join(root, "chats"))
			}
			ext := cursorSource{slug: project, meta: metas[id]}
			if ext.meta != "" {
				src.deps = []string{ext.meta}
			}
			src.extra = ext
			srcs = append(srcs, src)
		}
	}
	return srcs
}

// cursorMetas maps chat ids to their meta.json. Transcript directories use
// the chat id with "%" replaced by "_".
func cursorMetas(chats string) map[string]string {
	m := map[string]string{}
	for _, hash := range subdirs(chats) {
		for _, chat := range subdirs(filepath.Join(chats, hash)) {
			p := filepath.Join(chats, hash, chat, "meta.json")
			if _, ok := regularFile(p); ok {
				m[strings.ReplaceAll(chat, "%", "_")] = p
			}
		}
	}
	return m
}

type cursorBlock struct {
	Type    string          `json:"type"`
	Text    json.RawMessage `json:"text"`
	Name    string          `json:"name"`
	Input   json.RawMessage `json:"input"`
	Content json.RawMessage `json:"content"`
}

// cursorDynamicCall is the input of CallDynamicTool, through which Cursor
// reaches its own extra tools (namespace "cursor") and MCP servers (the
// server's name).
type cursorDynamicCall struct {
	Namespace string          `json:"namespace"`
	ToolName  string          `json:"toolName"`
	Arguments json.RawMessage `json:"arguments"`
}

const (
	cursorDynamicCallTool   = "CallDynamicTool"
	cursorDynamicLookupTool = "GetDynamicTools"
)

func parseCursor(src *source, r *recorder) {
	s := r.s
	ext, _ := src.extra.(cursorSource)
	var meta struct {
		CreatedAtMs int64  `json:"createdAtMs"`
		UpdatedAtMs int64  `json:"updatedAtMs"`
		CWD         string `json:"cwd"`
	}
	if ext.meta != "" {
		readJSON(src.sc, ext.meta, &meta)
	}
	s.StartedAt, s.EndedAt = parseUnixMillis(meta.CreatedAtMs), parseUnixMillis(meta.UpdatedAtMs)
	if s.StartedAt.IsZero() {
		if fi, ok := regularFile(src.files[0]); ok {
			s.StartedAt = birthTime(src.files[0], fi)
		}
	}
	cwd := cmp.Or(meta.CWD, resolveSlugPath(slugRoot(src.sc.home.Dir), ext.slug))
	s.ProjectPath = cwd

	var ts time.Time
	for i, path := range src.files {
		sub := i > 0
		r.file(path)
		_ = scanJSONL(src.sc, path, func(line []byte) bool {
			if r.done() {
				return false
			}
			var rec struct {
				Role    string `json:"role"`
				Message struct {
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(line, &rec) != nil || rec.Role == "" {
				return true // turn_ended and other bookkeeping
			}
			var blocks []cursorBlock
			if body := claudeFullBody(rec.Message.Content); body.isText {
				blocks = []cursorBlock{{Type: "text", Text: body.text}}
			} else {
				_ = json.Unmarshal(rec.Message.Content, &blocks)
			}
			switch rec.Role {
			case "user":
				cursorUser(r, blocks, sub, &ts, cwd)
			case "assistant":
				cursorAssistant(r, blocks, ts, cwd)
			}
			return true
		})
	}
}

func cursorUser(r *recorder, blocks []cursorBlock, sub bool, ts *time.Time, cwd string) {
	prompt := false
	for _, b := range blocks {
		if b.Type != "tool_result" {
			prompt = true
		}
	}
	role := roleSystem
	if prompt && !sub {
		r.s.UserMessages++
		role = roleUser
	}
	texts := make([]string, len(blocks))
	if r.rows() {
		// The prompt's own timestamp dates the whole turn, so it is read even
		// for a prompt before the requested page.
		for i, b := range blocks {
			if b.Type == "text" {
				var t time.Time
				texts[i], t = cursorPrompt(textOf(b.Text))
				if !t.IsZero() || i == 0 {
					*ts = t
				}
			}
		}
	}
	r.message(*ts, "", cwd)
	for i, b := range blocks {
		switch b.Type {
		case "text":
			r.text(role, blockText, func() string { return texts[i] })
		case "tool_result":
			r.toolResult("", nil, func() string { return textOf(b.Content) })
		default:
			r.nextBlock()
		}
	}
}

func cursorAssistant(r *recorder, blocks []cursorBlock, ts time.Time, cwd string) {
	// A record holding only schema lookups isn't a message.
	lookupsOnly := len(blocks) > 0
	for _, b := range blocks {
		if b.Type != "tool_use" || b.Name != cursorDynamicLookupTool {
			lookupsOnly = false
		}
	}
	if lookupsOnly {
		return
	}
	r.s.AssistantMessages++
	r.message(ts, "", cwd)
	for _, b := range blocks {
		switch b.Type {
		case "text":
			r.text(roleAssistant, blockText, func() string { return textOf(b.Text) })
		case "tool_use":
			switch b.Name {
			case cursorDynamicLookupTool:
				// Fetches a tool's schema before calling it; not an event.
			case cursorDynamicCallTool:
				var call cursorDynamicCall
				_ = json.Unmarshal(b.Input, &call)
				r.toolUse(cmp.Or(call.ToolName, b.Name), "", func() string { return rawString(b.Input) },
					func() (string, string) {
						a := parseArgs(call.Arguments)
						if call.Namespace == "cursor" {
							return classifyNative(agentCursor, call.ToolName, a)
						}
						return classifyMCP(call.Namespace, call.ToolName, a)
					})
			default:
				r.toolUse(b.Name, "", func() string { return rawString(b.Input) },
					func() (string, string) { return classify(agentCursor, b.Name, b.Input) })
			}
		default:
			r.nextBlock()
		}
	}
}

var (
	cursorTimestamp = regexp.MustCompile(`(?s)^\s*<timestamp>(.*?)</timestamp>\s*`)
	// The prompt may itself mention the tags, so it ends at the last close.
	cursorQuery = regexp.MustCompile(`(?s)<user_query>\s*(.*)</user_query>`)
)

// cursorPrompt strips the wrappers Cursor puts around a prompt and returns
// the time in its <timestamp> line, if that parses.
func cursorPrompt(text string) (string, time.Time) {
	var ts time.Time
	if m := cursorTimestamp.FindStringSubmatchIndex(text); m != nil {
		ts = parseCursorTime(text[m[2]:m[3]])
		text = text[m[1]:]
	}
	if m := cursorQuery.FindStringSubmatch(text); m != nil {
		text = m[1]
	}
	return strings.TrimSpace(text), ts
}

// parseCursorTime reads e.g. "Tuesday, Oct 27, 2026, 3:49 PM (UTC-4)".
func parseCursorTime(s string) time.Time {
	body, zone, ok := strings.Cut(strings.TrimSpace(s), " (UTC")
	if !ok {
		return time.Time{}
	}
	zone = strings.TrimSuffix(zone, ")")
	offset := 0
	if zone != "" {
		sign := 1
		switch zone[0] {
		case '-':
			sign = -1
		case '+':
		default:
			return time.Time{}
		}
		hh, mm, _ := strings.Cut(zone[1:], ":")
		h, err := strconv.Atoi(hh)
		if err != nil {
			return time.Time{}
		}
		m := 0
		if mm != "" {
			if m, err = strconv.Atoi(mm); err != nil {
				return time.Time{}
			}
		}
		if h < 0 || h > 23 || m < 0 || m > 59 {
			return time.Time{}
		}
		offset = sign * (h*3600 + m*60)
	}
	t, err := time.ParseInLocation("Monday, Jan 2, 2006, 3:04 PM", body, time.FixedZone("", offset))
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// slugRoot is what a Cursor project slug is relative to: the root of the
// home directory's volume.
func slugRoot(homeDir string) string {
	if vol := filepath.VolumeName(homeDir); vol != "" {
		return vol + string(filepath.Separator)
	}
	return string(filepath.Separator)
}
