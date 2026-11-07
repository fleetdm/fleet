package fleet

import (
	"encoding/json"
	"time"
)

// AISessionAction is what an AI agent's tool call does, as fleetd classifies it. An event keeps the action as
// reported, so one a newer fleetd adds is stored too; the host_ai_sessions.actions SET lists only these.
type AISessionAction string

const (
	AISessionActionRead    AISessionAction = "read"
	AISessionActionWrite   AISessionAction = "write"
	AISessionActionExecute AISessionAction = "execute"
	AISessionActionSearch  AISessionAction = "search"
	AISessionActionNetwork AISessionAction = "network"
	AISessionActionMCP     AISessionAction = "mcp"
	AISessionActionOther   AISessionAction = "other"
)

// AISessionAgentCursor is the agent fleetd reports for Cursor sessions. Cursor records no tool call results, so its
// events never get a decision.
const AISessionAgentCursor = "cursor"

// HostAISessionsMaxAge is how far back sessions are collected: hosts report those that started this recently, and
// Fleet keeps a session for as long after it started, or after Fleet last saw it change when its start is unknown or
// in the future, including after its transcript is gone from the host.
const HostAISessionsMaxAge = 30 * 24 * time.Hour

// HostAISession is an AI coding agent session on a host, as collected from fleetd's ai_agent_sessions table.
type HostAISession struct {
	ID                uint   `db:"id"`
	HostID            uint   `db:"host_id"`
	Agent             string `db:"agent"`
	AgentSessionID    string `db:"agent_session_id"`
	AgentVersion      string `db:"agent_version"`
	Username          string `db:"username"`
	AccountIdentifier string `db:"account_identifier"`
	Model             string `db:"model"`
	ProjectPath       string `db:"project_path"`
	GitRepository     string `db:"git_repository"`
	GitBranch         string `db:"git_branch"`
	// StartedAt and EndedAt bound the session's activity on the agent's clock; nil when the agent doesn't record it
	// (Cursor has no start time).
	StartedAt *time.Time `db:"started_at"`
	EndedAt   *time.Time `db:"ended_at"`
	// UserMessageCount is the prompts the person typed: tool results, sub-agent instructions and text the agent
	// injects aren't counted. AssistantMessageCount is the model's responses, sub-agents' included. ToolCallCount is
	// the tools the agent invoked, sub-agents' included; a response can carry several or none. Fleet compares it with
	// the events it has stored to tell whether a session still needs collecting.
	UserMessageCount      uint `db:"user_message_count"`
	AssistantMessageCount uint `db:"assistant_message_count"`
	ToolCallCount         uint `db:"tool_call_count"`
	// Token counts are the provider's usage figures summed over the session, kept apart because each is billed at
	// its own rate. InputTokenCount is prompt tokens processed fresh, not counting cache reads. OutputTokenCount is
	// what the model generated. CacheReadTokenCount is prompt tokens served from the provider's prompt cache, usually
	// the largest on a long session since the whole conversation is re-sent on every turn. CacheWriteTokenCount is
	// prompt tokens written to the cache for later turns. All four are 0 for Cursor, which records no usage.
	InputTokenCount      uint64 `db:"input_token_count"`
	OutputTokenCount     uint64 `db:"output_token_count"`
	CacheReadTokenCount  uint64 `db:"cache_read_token_count"`
	CacheWriteTokenCount uint64 `db:"cache_write_token_count"`
	// Actions are the distinct actions of the session's stored events.
	Actions []AISessionAction `db:"-"`
	// TranscriptPath is the session's main transcript on the host, which the events query names to read its tool
	// calls. TranscriptSize is the size in bytes of all its transcript files, sub-agents' included.
	TranscriptPath string `db:"transcript_path"`
	TranscriptSize uint64 `db:"transcript_size"`
}

// HostAISessionEvent is a tool call of an AI agent session, as collected from fleetd's ai_agent_session_messages
// table. Agent and AgentSessionID identify its session on the host.
type HostAISessionEvent struct {
	Agent          string
	AgentSessionID string
	// SessionEndedAt is the session's ended_at that Fleet had stored when it asked for these events. Decisions recorded
	// after it are only looked for once the session's ended_at moves past it.
	SessionEndedAt *time.Time
	// ToolCallIndex is the call's position among the session's tool calls, from 0. fleetd numbers calls in the order
	// they were written, across the session's sub-agent transcripts, so a call keeps its index as the session goes on.
	ToolCallIndex uint
	Action        AISessionAction
	Target        string
	Decision      string
	ExecutedAt    *time.Time
	Payload       json.RawMessage
	// ContentLength is the length in bytes of the call's arguments as recorded on the host, and Truncated whether
	// fleetd cut them before they reached Payload.
	ContentLength uint
	Truncated     bool
}

// HostAISessionEventsCursor is a stored session that still needs events, and where to resume reading them.
type HostAISessionEventsCursor struct {
	Agent          string     `db:"agent"`
	AgentSessionID string     `db:"agent_session_id"`
	TranscriptPath string     `db:"transcript_path"`
	EndedAt        *time.Time `db:"ended_at"`
	// EventCount is how many of the session's calls are stored: always its first EventCount, so reading resumes there.
	EventCount uint `db:"event_count"`
	// MaxToolCalls, when set, is how many calls to read at most: the session is a backfill, read a few calls at a time.
	MaxToolCalls uint `db:"max_tool_calls"`
	// RereadFromToolCallIndex is set when the session has events without a decision and has been active since they
	// were read, so they're read again from the first of them.
	RereadFromToolCallIndex *uint `db:"reread_from_tool_call_index"`
}
