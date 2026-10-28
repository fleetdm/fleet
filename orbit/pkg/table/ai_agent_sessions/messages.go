package ai_agent_sessions

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
)

// Message is one content block from one turn of a session. Unlike the Session
// table, this one deliberately carries the text.
type Message struct {
	Agent          string
	SessionID      string
	Username       string
	MessageIndex   int64
	BlockIndex     int64
	Role           string
	BlockType      string
	ToolName       string
	Timestamp      time.Time
	Model          string
	Content        string
	ContentLength  int64
	Truncated      bool
	TranscriptPath string
}

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

// transcriptRef locates one session's transcript before it is parsed, so the
// driver can filter and order files without reading them.
type transcriptRef struct {
	Agent     string
	SessionID string
	Username  string
	Path      string
	ModTime   time.Time
	// Parse reads at most `remaining` messages out of the transcript,
	// truncating each block's content to maxContentBytes (0 for no limit).
	Parse func(ref transcriptRef, remaining, maxContentBytes int) []Message
}

// transcriptDiscoverers lists the transcripts each agent has on disk.
var transcriptDiscoverers = []func(u userHome, cutoff time.Time) []transcriptRef{
	discoverClaudeTranscripts,
	discoverCopilotTranscripts,
	discoverCursorTranscripts,
	discoverCodexTranscripts,
	discoverGeminiTranscripts,
}

func aiAgentSessionMessagesColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("agent"),
		table.TextColumn("session_id"),
		table.TextColumn("username"),
		table.BigIntColumn("message_index"),
		table.BigIntColumn("block_index"),
		table.TextColumn("role"),
		table.TextColumn("block_type"),
		table.TextColumn("tool_name"),
		table.BigIntColumn("timestamp"),
		table.TextColumn("model"),
		table.TextColumn("content"),
		table.BigIntColumn("content_length"),
		table.IntegerColumn("truncated"),
		table.TextColumn("transcript_path"),
	}
}

func (m Message) row() map[string]string {
	truncated := "0"
	if m.Truncated {
		truncated = "1"
	}
	return map[string]string{
		"agent":           m.Agent,
		"session_id":      m.SessionID,
		"username":        m.Username,
		"message_index":   strconv.FormatInt(m.MessageIndex, 10),
		"block_index":     strconv.FormatInt(m.BlockIndex, 10),
		"role":            m.Role,
		"block_type":      m.BlockType,
		"tool_name":       m.ToolName,
		"timestamp":       unixString(m.Timestamp),
		"model":           m.Model,
		"content":         m.Content,
		"content_length":  strconv.FormatInt(m.ContentLength, 10),
		"truncated":       truncated,
		"transcript_path": m.TranscriptPath,
	}
}

// messageFilter narrows which transcripts are read, from the query's WHERE
// clause. An empty set means the column is unconstrained.
type messageFilter struct {
	agents     map[string]bool
	usernames  map[string]bool
	sessionIDs map[string]bool
	paths      map[string]bool
}

// newMessageFilter reads the equality constraints osquery pushed down.
func newMessageFilter(qc table.QueryContext) messageFilter {
	return messageFilter{
		agents:     equalityConstraints(qc, "agent"),
		usernames:  equalityConstraints(qc, "username"),
		sessionIDs: equalityConstraints(qc, "session_id"),
		paths:      equalityConstraints(qc, "transcript_path"),
	}
}

// equalityConstraints collects the `=` constraints on a column. osquery
// expands IN lists into several equality constraints, so this covers both.
func equalityConstraints(qc table.QueryContext, column string) map[string]bool {
	list, ok := qc.Constraints[column]
	if !ok {
		return nil
	}
	values := map[string]bool{}
	for _, c := range list.Constraints {
		if c.Operator == table.OperatorEquals {
			values[c.Expression] = true
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

// matchesTranscript reports whether a transcript could contain wanted rows. A
// transcript whose session id is not known until it is parsed (Gemini packs
// many sessions into one log) passes here and is filtered per message instead.
func (f messageFilter) matchesTranscript(ref transcriptRef) bool {
	if !allows(f.agents, ref.Agent) || !allows(f.usernames, ref.Username) || !allows(f.paths, ref.Path) {
		return false
	}
	return ref.SessionID == "" || allows(f.sessionIDs, ref.SessionID)
}

func (f messageFilter) matchesMessage(m Message) bool {
	return allows(f.sessionIDs, m.SessionID)
}

func allows(set map[string]bool, value string) bool {
	return len(set) == 0 || set[value]
}

// collectMessages reads session content, newest transcript first, stopping
// once maxRows messages have been gathered. A non-positive maxRows or
// maxContentBytes disables that limit.
func collectMessages(users []userHome, cutoff time.Time, f messageFilter, maxRows, maxContentBytes int) []Message {
	var refs []transcriptRef
	for _, u := range users {
		for _, discover := range transcriptDiscoverers {
			for _, ref := range discover(u, cutoff) {
				if f.matchesTranscript(ref) {
					refs = append(refs, ref)
				}
			}
		}
	}

	// Newest first, so a truncated result is the most recent activity rather
	// than an arbitrary slice of it.
	sort.SliceStable(refs, func(i, j int) bool {
		if !refs[i].ModTime.Equal(refs[j].ModTime) {
			return refs[i].ModTime.After(refs[j].ModTime)
		}
		return refs[i].Path < refs[j].Path
	})

	messages := make([]Message, 0, 128)
	for _, ref := range refs {
		remaining := 0 // unlimited
		if maxRows > 0 {
			remaining = maxRows - len(messages)
			if remaining <= 0 {
				break
			}
		}
		for _, m := range ref.Parse(ref, remaining, maxContentBytes) {
			if f.matchesMessage(m) {
				messages = append(messages, m)
			}
		}
	}

	if maxRows > 0 && len(messages) > maxRows {
		messages = messages[:maxRows]
	}
	return messages
}

// blockBuilder accumulates the rows for one transcript while enforcing the row
// budget and content limit.
type blockBuilder struct {
	ref             transcriptRef
	remaining       int
	unlimited       bool
	maxContentBytes int
	messages        []Message
	messageIndex    int64
	blockIndex      int64
}

func newBlockBuilder(ref transcriptRef, remaining, maxContentBytes int) *blockBuilder {
	return &blockBuilder{
		ref:             ref,
		remaining:       remaining,
		unlimited:       remaining <= 0,
		maxContentBytes: maxContentBytes,
		messageIndex:    -1,
	}
}

// full reports whether the row budget is exhausted.
func (b *blockBuilder) full() bool {
	return !b.unlimited && b.remaining <= 0
}

// nextMessage starts a new turn, resetting the per-message block counter.
func (b *blockBuilder) nextMessage() {
	b.messageIndex++
	b.blockIndex = 0
}

// add appends one content block. Empty blocks that carry no text and no tool
// name are dropped rather than producing blank rows.
func (b *blockBuilder) add(role, blockType, toolName, content string, ts time.Time, model string) {
	if b.full() {
		return
	}
	if content == "" && toolName == "" {
		return
	}

	full := int64(len(content))
	truncated := false
	if b.maxContentBytes > 0 && len(content) > b.maxContentBytes {
		content = truncateUTF8(content, b.maxContentBytes)
		truncated = true
	}

	b.messages = append(b.messages, Message{
		Agent:          b.ref.Agent,
		SessionID:      b.ref.SessionID,
		Username:       b.ref.Username,
		MessageIndex:   b.messageIndex,
		BlockIndex:     b.blockIndex,
		Role:           role,
		BlockType:      blockType,
		ToolName:       toolName,
		Timestamp:      ts,
		Model:          model,
		Content:        content,
		ContentLength:  full,
		Truncated:      truncated,
		TranscriptPath: b.ref.Path,
	})
	b.blockIndex++
	if !b.unlimited {
		b.remaining--
	}
}

// truncateUTF8 cuts a string to at most limit bytes without splitting a rune,
// so the column never carries a mangled character.
func truncateUTF8(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := limit
	for cut > 0 && !utf8Boundary(s[cut]) {
		cut--
	}
	return s[:cut]
}

// utf8Boundary reports whether b can start a UTF-8 sequence, which is true for
// anything that is not a continuation byte.
func utf8Boundary(b byte) bool {
	return b&0xC0 != 0x80
}

// richContent is a message body that may be a plain string or an array of
// typed blocks. Unlike contentBlocks, which is used by the metadata table and
// deliberately discards text, this keeps it.
type richContent struct {
	Text   string
	Blocks []contentBlockDetail
}

// contentBlockDetail is one typed block of a message body.
type contentBlockDetail struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Thinking string          `json:"thinking"`
	Name     string          `json:"name"`
	Input    json.RawMessage `json:"input"`
	Content  flexText        `json:"content"`
}

func (r *richContent) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return nil
		}
		r.Text = s
		return nil
	}
	// An unrecognized shape yields no blocks rather than failing the record.
	_ = json.Unmarshal(b, &r.Blocks)
	return nil
}

// isPlainText reports whether the body was a bare string.
func (r richContent) isPlainText() bool {
	return len(r.Blocks) == 0 && r.Text != ""
}

// flexText is a field that may hold a string, or an array of text blocks, and
// is flattened to a single string either way.
type flexText string

func (f *flexText) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return nil
		}
		*f = flexText(s)
		return nil
	}
	var blocks []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(b, &blocks); err != nil {
		return nil
	}
	var parts []string
	for _, blk := range blocks {
		if blk.Text != "" {
			parts = append(parts, blk.Text)
		}
	}
	*f = flexText(strings.Join(parts, "\n"))
	return nil
}

// addBlocks emits a row per block of a typed message body, mapping each block
// to its role and type.
func (b *blockBuilder) addBlocks(role string, content richContent, ts time.Time, model string) {
	if content.isPlainText() {
		b.add(role, blockText, "", content.Text, ts, model)
		return
	}
	for _, blk := range content.Blocks {
		switch blk.Type {
		case "text", "output_text", "input_text":
			b.add(role, blockText, "", blk.Text, ts, model)
		case "thinking", "reasoning":
			b.add(role, blockThinking, "", firstNonEmpty(blk.Thinking, blk.Text), ts, model)
		case "tool_use", "tool_call":
			// The tool's arguments are the interesting part; they are kept as
			// the JSON the agent actually sent.
			b.add(role, blockToolUse, blk.Name, string(blk.Input), ts, model)
		case "tool_result", "tool_use_result":
			b.add(role, blockToolResult, blk.Name, string(blk.Content), ts, model)
		}
	}
}

// contentRole is the role a message body should be attributed to. A user turn
// that only carries tool results back to the model is machine output, not
// something the person typed.
func contentRole(recordRole string, content richContent) string {
	if recordRole != roleUser || content.isPlainText() {
		return recordRole
	}
	for _, blk := range content.Blocks {
		if blk.Type != "tool_result" && blk.Type != "tool_use_result" {
			return recordRole
		}
	}
	if len(content.Blocks) == 0 {
		return recordRole
	}
	return roleSystem
}
