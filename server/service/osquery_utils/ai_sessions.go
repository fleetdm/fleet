package osquery_utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// aiSessionsPerEventsQuery is how many sessions one events query reads. They share fleetd's ~1 MiB result budget, so a
// cut ends the query early and the next detail refresh resumes.
const aiSessionsPerEventsQuery = 5

var aiSessionsPlatforms = append([]string{"darwin", "windows"}, fleet.HostLinuxOSs...)

// aiSessionsDiscovery checks for a column the fleetd table only has once it reports what Fleet reads, so an older
// fleetd isn't asked for missing columns.
func aiSessionsDiscovery(table, column string) string {
	return fmt.Sprintf(`SELECT 1 WHERE EXISTS (%s) AND EXISTS (SELECT 1 FROM pragma_table_info('%s') WHERE name = '%s')`,
		discoveryTable(table), table, column)
}

// aiAgentSessions reads the sessions active in the last 7 days, by the host's own clock, however long ago they
// started: a session is reported for as long as it's in use, and one idle for a week can't change anymore. A host
// first reporting, as when the feature rolls out, sends only those. Sessions whose end is unknown (fleetd reports 0)
// are left out.
var aiAgentSessions = DetailQuery{
	Query: `
SELECT
  agent, session_id, agent_version, username, account_identifier, model, project_path, git_repository, git_branch,
  started_at, ended_at, user_messages, assistant_messages, tool_calls, input_tokens, output_tokens, cache_read_tokens,
  cache_write_tokens, transcript_path, transcript_size_bytes
FROM ai_agent_sessions
WHERE ended_at >= (SELECT unix_time FROM time) - 604800`,
	Platforms:        aiSessionsPlatforms,
	Discovery:        aiSessionsDiscovery("ai_agent_sessions", "account_identifier"),
	DirectIngestFunc: directIngestAIAgentSessions,
}

// aiAgentSessionEventsQuery reads the tool calls of sessions stored on earlier refreshes, a few sessions at a time,
// up to each session's first maxEvents and, when maxEventsPerRefresh is set, that many new ones per session.
func aiAgentSessionEventsQuery(maxEvents, maxEventsPerRefresh int) DetailQuery {
	if maxEvents <= 0 {
		maxEvents = fleet.DefaultHostAISessionMaxEvents
	}
	return DetailQuery{
		QueryFunc: func(ctx context.Context, logger *slog.Logger, host *fleet.Host, ds fleet.Datastore) (string, bool) {
			return buildAIAgentSessionEventsQuery(ctx, logger, host, ds, uint(maxEvents), uint(max(maxEventsPerRefresh, 0)))
		},
		Platforms: aiSessionsPlatforms,
		Discovery: aiSessionsDiscovery("ai_agent_session_messages", "tool_call_index"),
		DirectIngestFunc: func(ctx context.Context, logger *slog.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
			return directIngestAIAgentSessionEvents(ctx, logger, host, ds, rows, uint(maxEvents))
		},
	}
}

func sqliteQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// buildAIAgentSessionEventsQuery asks for the tool calls of the stored sessions that still need events, one UNION ALL
// arm per session, each preceded by a constant row naming the session as asked for, so the ingestion can tell a session
// that returned no tool calls from one it didn't ask about. Sessions are named by transcript_path rather than
// session_id: naming a session_id lifts fleetd's 8 KB cap on each content value, so a single large tool call could fill
// the result.
//
// A session is resumed after the calls Fleet has stored (or from its first undecided call, when rereading). fleetd
// applies the tool_call_index bound itself, so a call's first row is always one Fleet lacks: the arms share fleetd's
// ~1 MiB result budget, and once it is spent each arm returns only that row. Calls from maxEvents on aren't asked for,
// and a session catching up is asked for at most maxEventsPerRefresh calls when that is set. A reread isn't limited: its
// calls are stored already, and one cut before the last stored call is repeated at the next refresh.
func buildAIAgentSessionEventsQuery(ctx context.Context, logger *slog.Logger, host *fleet.Host, ds fleet.Datastore, maxEvents, maxEventsPerRefresh uint) (string, bool) {
	cursors, err := ds.ListHostAISessionsNeedingEvents(ctx, host.ID, aiSessionsPerEventsQuery, maxEvents)
	if err != nil {
		logger.ErrorContext(ctx, "QueryFunc - ai agent session events", "component", "service", "host_id", host.ID, "err", err)
		return "", false
	}

	var arms []string
	for _, c := range cursors {
		var endedAt int64
		if c.EndedAt != nil {
			endedAt = c.EndedAt.Unix()
		}
		from, to := c.EventCount, maxEvents
		if c.RereadFromToolCallIndex != nil {
			from = *c.RereadFromToolCallIndex
		} else if maxEventsPerRefresh > 0 {
			to = min(to, from+maxEventsPerRefresh)
		}
		arms = append(arms,
			fmt.Sprintf(`SELECT '%s' AS agent, '%s' AS session_id, 0 AS tool_call_index, '' AS tool_name, '' AS tool_use_id, '' AS action, '' AS target, '' AS decision, 0 AS timestamp, '' AS content, 0 AS content_length, 0 AS truncated, %d AS session_ended_at, 1 AS requested`,
				sqliteQuote(c.Agent), sqliteQuote(c.AgentSessionID), endedAt),
			fmt.Sprintf(`SELECT agent, session_id, tool_call_index, tool_name, tool_use_id, action, target, decision, timestamp, content, content_length, truncated, %d AS session_ended_at, 0 AS requested
FROM ai_agent_session_messages
WHERE transcript_path = '%s' AND block_type = 'tool_use' AND tool_call_index >= %d AND tool_call_index < %d`,
				endedAt, sqliteQuote(c.TranscriptPath), from, to))
	}
	if len(arms) == 0 {
		return "", false
	}
	return strings.Join(arms, "\nUNION ALL\n"), true
}

// Values come from files on the host, so those the columns can't hold are cut to fit rather than failing the host's
// ingestion, which would be retried with the same values at every refresh.
const (
	// maxAISessionJSONDepth is the deepest nesting MySQL accepts in a JSON column.
	maxAISessionJSONDepth = 100
	// maxAISessionUnix is the last second a MySQL DATETIME holds.
	maxAISessionUnix = 253402300799
)

func parseAISessionUint(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

// parseAISessionCount parses a count for an INT UNSIGNED column.
func parseAISessionCount(s string) uint {
	return uint(min(parseAISessionUint(s), math.MaxUint32))
}

// parseAISessionTime parses a fleetd Unix timestamp, where 0 means unknown.
func parseAISessionTime(s string) *time.Time {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v <= 0 || v > maxAISessionUnix {
		return nil
	}
	t := time.Unix(v, 0).UTC()
	return &t
}

// mysqlJSONInput returns content, a JSON object or array, re-encoded as MySQL accepts it, or false when it isn't valid
// JSON or MySQL would reject it: nested too deeply for the column counting the payload's own level, or holding a number
// past a double's range. Re-encoding replaces lone UTF-16 surrogates, which MySQL also rejects.
func mysqlJSONInput(content string) (json.RawMessage, bool) {
	if trimmed := strings.TrimSpace(content); !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return nil, false
	}
	dec := json.NewDecoder(strings.NewReader(content))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, false
	}
	if !mysqlJSONValue(v, 1) {
		return nil, false
	}
	raw, err := json.Marshal(v)
	return raw, err == nil
}

// mysqlJSONValue reports whether MySQL accepts v, found depth containers deep in the payload.
func mysqlJSONValue(v any, depth int) bool {
	switch v := v.(type) {
	case map[string]any:
		return depth < maxAISessionJSONDepth && mysqlJSONValues(maps.Values(v), depth+1)
	case []any:
		return depth < maxAISessionJSONDepth && mysqlJSONValues(slices.Values(v), depth+1)
	case json.Number:
		// Underflow parses to 0 without an error, as MySQL stores it.
		_, err := strconv.ParseFloat(string(v), 64)
		return err == nil
	}
	return true
}

func mysqlJSONValues(values iter.Seq[any], depth int) bool {
	for v := range values {
		if !mysqlJSONValue(v, depth) {
			return false
		}
	}
	return true
}

func directIngestAIAgentSessions(ctx context.Context, logger *slog.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
	if len(rows) == 0 {
		return nil
	}
	sessions := make([]*fleet.HostAISession, 0, len(rows))
	for _, row := range rows {
		// The events query names a session by these, and its SQL can't hold a NUL.
		if row["agent"] == "" || row["session_id"] == "" ||
			strings.ContainsRune(row["agent"]+row["session_id"]+row["transcript_path"], 0) {
			logger.DebugContext(ctx, "ai agent session without a usable agent, session id or transcript path",
				"component", "service", "host_id", host.ID)
			continue
		}
		sessions = append(sessions, &fleet.HostAISession{
			Agent:                 row["agent"],
			AgentSessionID:        row["session_id"],
			AgentVersion:          row["agent_version"],
			Username:              row["username"],
			AccountIdentifier:     row["account_identifier"],
			Model:                 row["model"],
			ProjectPath:           row["project_path"],
			GitRepository:         row["git_repository"],
			GitBranch:             row["git_branch"],
			StartedAt:             parseAISessionTime(row["started_at"]),
			EndedAt:               parseAISessionTime(row["ended_at"]),
			UserMessageCount:      parseAISessionCount(row["user_messages"]),
			AssistantMessageCount: parseAISessionCount(row["assistant_messages"]),
			ToolCallCount:         parseAISessionCount(row["tool_calls"]),
			InputTokenCount:       parseAISessionUint(row["input_tokens"]),
			OutputTokenCount:      parseAISessionUint(row["output_tokens"]),
			CacheReadTokenCount:   parseAISessionUint(row["cache_read_tokens"]),
			CacheWriteTokenCount:  parseAISessionUint(row["cache_write_tokens"]),
			TranscriptPath:        row["transcript_path"],
			TranscriptSize:        parseAISessionUint(row["transcript_size_bytes"]),
		})
	}
	if err := ds.UpsertHostAISessions(ctx, host.ID, sessions); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert host ai sessions")
	}
	return nil
}

func directIngestAIAgentSessionEvents(ctx context.Context, logger *slog.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string, maxEvents uint) error {
	if len(rows) == 0 {
		return nil
	}
	// The sessions the events were asked for: those with none in the result count an empty read.
	var requested []*fleet.HostAISessionEventsCursor
	events := make([]*fleet.HostAISessionEvent, 0, len(rows))
	for _, row := range rows {
		if row["requested"] == "1" {
			if row["agent"] != "" && row["session_id"] != "" {
				requested = append(requested, &fleet.HostAISessionEventsCursor{Agent: row["agent"], AgentSessionID: row["session_id"]})
			}
			continue
		}
		toolCallIndex, err := strconv.ParseUint(row["tool_call_index"], 10, 32)
		if row["agent"] == "" || row["session_id"] == "" || err != nil {
			logger.DebugContext(ctx, "ai agent session event without session or position", "component", "service", "host_id", host.ID)
			continue
		}
		// The query doesn't ask for these, so they only come from a host that ignored its bound.
		if toolCallIndex >= uint64(maxEvents) {
			continue
		}
		events = append(events, &fleet.HostAISessionEvent{
			Agent:          row["agent"],
			AgentSessionID: row["session_id"],
			SessionEndedAt: parseAISessionTime(row["session_ended_at"]),
			ToolCallIndex:  uint(toolCallIndex),
			Action:         fleet.AISessionAction(row["action"]),
			Target:         row["target"],
			Decision:       row["decision"],
			ExecutedAt:     parseAISessionTime(row["timestamp"]),
			Payload:        aiSessionEventPayload(row["tool_name"], row["tool_use_id"], row["content"]),
			ContentLength:  parseAISessionCount(row["content_length"]),
			Truncated:      row["truncated"] == "1",
		})
	}
	if len(requested) == 0 && len(events) == 0 {
		return nil
	}
	if err := ds.UpsertHostAISessionEvents(ctx, host.ID, requested, events); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert host ai session events")
	}
	return nil
}

// aiSessionEventPayload builds the tool call as stored: its name, id, and arguments. fleetd keeps JSON object and array
// arguments valid even when it shortens them, so those are kept as JSON when MySQL accepts them; anything else is kept
// as a string.
func aiSessionEventPayload(toolName, toolUseID, content string) json.RawMessage {
	var input any = content
	if raw, ok := mysqlJSONInput(content); ok {
		input = raw
	}
	// Neither a string nor JSON just encoded can fail to marshal.
	payload, _ := json.Marshal(struct {
		Name  string `json:"name"`
		ID    string `json:"id"`
		Input any    `json:"input"`
	}{toolName, toolUseID, input})
	return payload
}
