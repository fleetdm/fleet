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

// AISessionCategory is how the API groups tool calls, mapped from their action when read.
type AISessionCategory string

const (
	AISessionCategoryCommand    AISessionCategory = "command"
	AISessionCategoryFileAccess AISessionCategory = "file_access"
)

// Category is the action's API category, or "" for the actions that have none (network, mcp, other, and any a newer
// fleetd adds).
func (a AISessionAction) Category() AISessionCategory {
	switch a {
	case AISessionActionExecute:
		return AISessionCategoryCommand
	case AISessionActionRead, AISessionActionWrite, AISessionActionSearch:
		return AISessionCategoryFileAccess
	default:
		return ""
	}
}

// AISessionAgentCursor is the agent fleetd reports for Cursor sessions. Cursor records no tool call results, so its
// events never get a decision.
const AISessionAgentCursor = "cursor"

// HostAISessionsMaxAge is how far back sessions are collected: hosts report those that started this recently, and
// Fleet keeps a session for as long after it started, or after Fleet last saw it change when its start is unknown or
// in the future, including after its transcript is gone from the host.
const HostAISessionsMaxAge = 30 * 24 * time.Hour

// AISessionAgents are the agents fleetd reports sessions for.
var AISessionAgents = []string{"claude_code", "copilot_cli", AISessionAgentCursor, "codex_cli", "gemini_cli"}

// HostAISessionAuthz is the authorization object for reading a host's AI sessions, which can include secrets in
// commands and their arguments.
type HostAISessionAuthz struct {
	TeamID *uint `json:"team_id"` // nolint:apiparamcheck // used for rego policy, and we only support team_id there.
}

// AuthzType implements authz.AuthzTyper.
func (HostAISessionAuthz) AuthzType() string {
	return "host_ai_session"
}

// HostAISessionListOptions are the options to list a host's AI sessions.
type HostAISessionListOptions struct {
	ListOptions
	// Agent, when set, filters to the sessions of that agent.
	Agent string
	// Category, when set, filters to the sessions with a tool call in that category.
	Category AISessionCategory
}

// HostAISessionEventListOptions are the options to list an AI session's events.
type HostAISessionEventListOptions struct {
	ListOptions
	// Category, when set, filters to the events in that category.
	Category AISessionCategory
}

// HostAISession is an AI coding agent session on a host, as collected from fleetd's ai_agent_sessions table.
type HostAISession struct {
	ID                uint   `json:"id" db:"id"`
	HostID            uint   `json:"host_id" db:"host_id"`
	Agent             string `json:"agent" db:"agent"`
	AgentVersion      string `json:"agent_version" db:"agent_version"`
	AgentSessionID    string `json:"agent_session_id" db:"agent_session_id"`
	Username          string `json:"username" db:"username"`
	AccountIdentifier string `json:"account_identifier" db:"account_identifier"`
	Model             string `json:"model" db:"model"`
	ProjectPath       string `json:"project_path" db:"project_path"`
	GitRepository     string `json:"git_repository" db:"git_repository"`
	GitBranch         string `json:"git_branch" db:"git_branch"`
	// StartedAt and EndedAt bound the session's activity on the agent's clock; nil when the agent doesn't record it
	// (Cursor has no start time).
	StartedAt *time.Time `json:"started_at" db:"started_at"`
	EndedAt   *time.Time `json:"ended_at" db:"ended_at"`
	// UserMessageCount is the prompts the person typed: tool results, sub-agent instructions and text the agent
	// injects aren't counted. AssistantMessageCount is the model's responses, sub-agents' included. ToolCallCount is
	// the tools the agent invoked, sub-agents' included; a response can carry several or none. Fleet compares it with
	// the events it has stored to tell whether a session still needs collecting.
	UserMessageCount      uint `json:"user_message_count" db:"user_message_count"`
	AssistantMessageCount uint `json:"assistant_message_count" db:"assistant_message_count"`
	ToolCallCount         uint `json:"tool_call_count" db:"tool_call_count"`
	// Actions are the distinct actions of the session's stored events.
	Actions    []AISessionAction   `json:"-" db:"-"`
	Categories []AISessionCategory `json:"categories" db:"-"`
	// Backfill is set while Fleet is still collecting the calls the session made before Fleet first stored it, so
	// fewer events than ToolCallCount may be listed.
	Backfill bool `json:"backfill" db:"backfill"`
	// Token counts are the provider's usage figures summed over the session, kept apart because each is billed at
	// its own rate. InputTokenCount is prompt tokens processed fresh, not counting cache reads. OutputTokenCount is
	// what the model generated. CacheReadTokenCount is prompt tokens served from the provider's prompt cache, usually
	// the largest on a long session since the whole conversation is re-sent on every turn. CacheWriteTokenCount is
	// prompt tokens written to the cache for later turns. All four are 0 for Cursor, which records no usage.
	InputTokenCount      uint64 `json:"input_token_count" db:"input_token_count"`
	OutputTokenCount     uint64 `json:"output_token_count" db:"output_token_count"`
	CacheReadTokenCount  uint64 `json:"cache_read_token_count" db:"cache_read_token_count"`
	CacheWriteTokenCount uint64 `json:"cache_write_token_count" db:"cache_write_token_count"`
	// TranscriptPath is the session's main transcript on the host, which the events query names to read its tool
	// calls. TranscriptSize is the size in bytes of all its transcript files, sub-agents' included.
	TranscriptPath string `json:"transcript_path" db:"transcript_path"`
	TranscriptSize uint64 `json:"transcript_size" db:"transcript_size"`
}

// HostAISessionEvent is a tool call of an AI agent session, as collected from fleetd's ai_agent_session_messages
// table. Agent and AgentSessionID identify its session on the host.
type HostAISessionEvent struct {
	ID             uint   `json:"id" db:"id"`
	Agent          string `json:"-" db:"-"`
	AgentSessionID string `json:"-" db:"-"`
	// SessionEndedAt is the session's ended_at that Fleet had stored when it asked for these events. Decisions recorded
	// after it are only looked for once the session's ended_at moves past it.
	SessionEndedAt *time.Time `json:"-" db:"-"`
	// ToolCallIndex is the call's position among the session's tool calls, from 0. fleetd numbers calls in the order
	// they were written, across the session's sub-agent transcripts, so a call keeps its index as the session goes on.
	ToolCallIndex uint                `json:"-" db:"tool_call_index"`
	Action        AISessionAction     `json:"-" db:"action"`
	Categories    []AISessionCategory `json:"categories" db:"-"`
	Target        string              `json:"target" db:"target"`
	Decision      string              `json:"decision" db:"decision"`
	ExecutedAt    *time.Time          `json:"executed_at" db:"executed_at"`
	Payload       json.RawMessage     `json:"payload" db:"payload"`
	// ContentLength is the length in bytes of the call's arguments as recorded on the host, and Truncated whether
	// fleetd cut them before they reached Payload.
	ContentLength uint `json:"-" db:"content_length"`
	Truncated     bool `json:"truncated" db:"truncated"`
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
