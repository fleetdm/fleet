package ai_agent_sessions

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// maxBlockBytes caps each content block by default.
	maxBlockBytes = 8 << 10
	// maxNamedBlockBytes caps a block, as JSON, when the query names its
	// session, so a single row still fits the result's size limit.
	maxNamedBlockBytes = maxResultBytes - 64<<10
	// maxFieldBytes caps every other text column. They come from files users
	// control, and one huge value must not crowd out every other row.
	maxFieldBytes = 4 << 10
	// decisionLookahead is how many messages are read past the end of a page
	// to find the results of tool calls the page returned.
	decisionLookahead = 64
)

// Roles reported in the `role` column.
const (
	roleUser      = "user"
	roleAssistant = "assistant"
	roleSystem    = "system"
)

// Block types reported in the `block_type` column.
const (
	blockText       = "text"
	blockThinking   = "thinking"
	blockToolUse    = "tool_use"
	blockToolResult = "tool_result"
)

// Values of the `decision` column.
const (
	decisionAllowed = "allowed"
	decisionDenied  = "denied"
	decisionError   = "error"
)

// Message is one content block of one turn of a session.
type Message struct {
	Agent          string
	SessionID      string
	Username       string
	MessageIndex   int64
	BlockIndex     int64
	Role           string
	BlockType      string
	ToolName       string
	ToolUseID      string
	Action         string
	Target         string
	Decision       string
	CWD            string
	Timestamp      time.Time
	Model          string
	Content        string
	ContentLength  int64
	Truncated      bool
	TranscriptPath string
}

func (m Message) row() map[string]string {
	truncated := "0"
	if m.Truncated {
		truncated = "1"
	}
	return map[string]string{
		"agent":           m.Agent,
		"session_id":      capField(m.SessionID),
		"username":        m.Username,
		"message_index":   strconv.FormatInt(m.MessageIndex, 10),
		"block_index":     strconv.FormatInt(m.BlockIndex, 10),
		"role":            m.Role,
		"block_type":      m.BlockType,
		"tool_name":       capField(m.ToolName),
		"tool_use_id":     capField(m.ToolUseID),
		"action":          m.Action,
		"target":          m.Target,
		"decision":        m.Decision,
		"cwd":             capField(m.CWD),
		"timestamp":       unixString(m.Timestamp),
		"model":           capField(m.Model),
		"content":         m.Content,
		"content_length":  strconv.FormatInt(m.ContentLength, 10),
		"truncated":       truncated,
		"transcript_path": capField(m.TranscriptPath),
	}
}

// wireSize is an upper bound on the row's size once JSON encoded, which is
// what the size limit counts.
func (m Message) wireSize() int {
	n := 2
	for k, v := range m.row() {
		n += len(k) + jsonLen(v) + 6
	}
	return n
}

// jsonLen is how many bytes s takes as a JSON string body, matching
// encoding/json's escaping.
func jsonLen(s string) int {
	n := 0
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			switch {
			case c == '"' || c == '\\' || c == '\n' || c == '\r' || c == '\t':
				n += 2
			case c < 0x20 || c == '<' || c == '>' || c == '&':
				n += 6
			default:
				n++
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if (r == utf8.RuneError && size == 1) || r == ' ' || r == ' ' {
			n += 6
		} else {
			n += size
		}
		i += size
	}
	return n
}

// recorder receives a session as an agent's parser reads it. It always fills
// in the Session; for the messages table it also builds rows, honouring the
// query's paging, block types and size limit.
//
// Parsers call next() at each message, at() to set the time, model and
// working directory of the blocks that follow, and then the block methods.
// Content is passed as a function so it is only decoded for rows actually
// returned.
type recorder struct {
	s    *Session
	seen bool

	q      *query // nil when only the Session is wanted
	path   string
	ts     time.Time
	model  string
	cwd    string
	source *source

	msgIndex   int64
	blockIndex int64
	out        []Message
	// pending maps a tool call id to its row, to record its outcome.
	pending   map[string]int
	undecided int
	// past is set once the page (or the size limit) is reached; reading then
	// only continues to settle the outcome of calls already returned.
	past   bool
	pastAt int64
	// cut is set when the size limit stopped the result.
	cut bool
}

func newMessageRecorder(src *source, q *query) *recorder {
	return &recorder{
		s:        &Session{},
		q:        q,
		source:   src,
		msgIndex: -1,
		pending:  map[string]int{},
	}
}

// file switches to the next transcript file of the session.
func (r *recorder) file(path string) { r.path = path }

// done reports whether the parser can stop reading.
func (r *recorder) done() bool {
	return r.past && (r.undecided == 0 || r.msgIndex-r.pastAt >= decisionLookahead)
}

func (r *recorder) stop() {
	if !r.past {
		r.past, r.pastAt = true, r.msgIndex
	}
}

// rows reports whether rows are being built at all.
func (r *recorder) rows() bool { return r.q != nil }

// skipNext reports whether the next message falls before the requested
// page, so the parser may skip decoding it beyond what next() needs.
func (r *recorder) skipNext() bool { return r.q != nil && r.msgIndex+1 < r.q.lo }

func (r *recorder) next() {
	r.seen = true
	r.msgIndex++
	r.blockIndex = 0
	if r.q != nil && r.q.hasHi && r.msgIndex > r.q.hi {
		r.stop()
	}
}

func (r *recorder) at(ts time.Time, model, cwd string) {
	r.ts, r.model, r.cwd = ts, model, cwd
}

// wanted reports whether blocks of the current message become rows.
func (r *recorder) wanted() bool {
	return r.q != nil && !r.past && r.msgIndex >= r.q.lo
}

// Every block of a message takes a block index, whether or not it becomes a
// row, so the index doesn't depend on the query's filters.
func (r *recorder) text(role, blockType string, content func() string) {
	r.seen = true
	defer r.nextBlock()
	if !r.wanted() || !r.q.blockTypes.allows(blockType) {
		return
	}
	r.add(Message{Role: role, BlockType: blockType}, content())
}

func (r *recorder) nextBlock() { r.blockIndex++ }

// toolUse records a tool call. classify, which yields the action and target,
// only runs for rows that are returned.
func (r *recorder) toolUse(name, id string, input func() string, classify func() (action, target string)) {
	r.seen = true
	r.s.ToolCalls++
	defer r.nextBlock()
	if !r.wanted() || !r.q.blockTypes.allows(blockToolUse) {
		return
	}
	action, target := classify()
	if r.add(Message{Role: roleAssistant, BlockType: blockToolUse, ToolName: name, ToolUseID: id, Action: action, Target: target}, input()) && id != "" {
		r.pending[id] = len(r.out) - 1
		r.undecided++
	}
}

// toolResult records the result of call id, and decision as the call's
// outcome (decide is given the call's row).
func (r *recorder) toolResult(id string, decide func(call Message) string, content func() string) {
	r.seen = true
	defer r.nextBlock()
	if i, ok := r.pending[id]; ok && decide != nil {
		r.decide(id, decide(r.out[i]))
	}
	if !r.wanted() || !r.q.blockTypes.allows(blockToolResult) {
		return
	}
	r.add(Message{Role: roleSystem, BlockType: blockToolResult, ToolUseID: id}, content())
}

// decide records a call's outcome. A denial is never downgraded by a later
// failure report of the same call.
func (r *recorder) decide(id, decision string) {
	i, ok := r.pending[id]
	if !ok || decision == "" {
		return
	}
	switch prev := r.out[i].Decision; {
	case prev == decisionDenied, prev == decisionError && decision == decisionAllowed:
		return
	case prev == "":
		r.undecided--
	}
	r.out[i].Decision = decision
}

// add appends a row unless it is empty or the size limit is reached.
func (r *recorder) add(m Message, content string) bool {
	if (content == "" && m.ToolName == "") || !r.q.paths.allows(r.path) {
		return false
	}
	m.Agent = r.source.agent.name
	m.SessionID = r.source.id
	m.Username = r.source.sc.home.Username
	m.MessageIndex = r.msgIndex
	m.BlockIndex = r.blockIndex
	m.CWD = r.cwd
	m.Timestamp = r.ts
	m.TranscriptPath = r.path
	if m.Role == roleAssistant {
		m.Model = r.model
	}
	m.ContentLength = int64(len(content))
	switch {
	// Escaping can make content up to six times larger on the wire.
	case len(content) <= r.q.maxContent && jsonLen(content) <= maxNamedBlockBytes:
	case m.BlockType == blockToolUse:
		content = fitToolInput(content, r.q.maxContent)
		m.Truncated = true
	default:
		content = truncateUTF8(content, r.q.maxContent)
		if jsonLen(content) > maxNamedBlockBytes {
			content = truncateJSON(content, maxNamedBlockBytes)
		}
		m.Truncated = true
	}
	m.Content = content
	if !messageBudget.take(m.wireSize(), r.q.rows == 0) {
		r.stop()
		r.cut = true
		return false
	}
	r.q.rows++
	r.out = append(r.out, m)
	return true
}

// truncateUTF8 cuts s to at most limit bytes without splitting a rune.
func truncateUTF8(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// truncateJSON cuts s so its JSON-escaped form is at most limit bytes.
func truncateJSON(s string, limit int) string {
	n := 0
	for i, r := range s {
		w := jsonLen(string(r))
		if r == utf8.RuneError {
			w = 6
		}
		if n+w > limit {
			return s[:i]
		}
		n += w
	}
	return s
}

func capField(s string) string { return truncateUTF8(s, maxFieldBytes) }

// textOf flattens a message body: a string, a block with a "text" field, or
// an array of them, nested at most twice. Anything else (images, unknown
// shapes) yields nothing. The nesting limit keeps a hostile, deeply nested
// array from being re-scanned once per level.
func textOf(raw json.RawMessage) string { return textOfDepth(raw, 2) }

func textOfDepth(raw json.RawMessage, depth int) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}
	switch raw[0] {
	case '"':
		var s string
		_ = json.Unmarshal(raw, &s)
		return s
	case '{':
		var b struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(raw, &b)
		return b.Text
	case '[':
		if depth == 0 {
			return ""
		}
		var blocks []json.RawMessage
		if json.Unmarshal(raw, &blocks) != nil {
			return ""
		}
		var parts []string
		for _, b := range blocks {
			if t := textOfDepth(b, depth-1); t != "" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// rawString is raw as a string, or "" for an absent or null value.
func rawString(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	return string(raw)
}
