// Package ai_agent_sessions exposes two tables over the transcripts AI coding
// agents (Claude Code, Copilot CLI, Cursor CLI, Codex CLI, Gemini CLI) keep in
// every user's home directory:
//   - ai_agent_sessions: one row of metadata per session, cheap enough to
//     schedule (transcripts are parsed once and cached while unchanged);
//   - ai_agent_session_messages: the conversation itself, one row per content
//     block, for on-demand queries. Results are capped at about 1 MiB so they
//     stay within what Fleet's server ingests; page tool calls with
//     tool_call_index, and everything else with message_index.
//
// Ported from github.com/allenhouchins/fleet-extensions/ai_agent_sessions (MIT).
package ai_agent_sessions

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/internal/homes"
)

// Agent identifiers reported in the `agent` column.
const (
	agentClaudeCode = "claude_code"
	agentCopilotCLI = "copilot_cli"
	agentCursor     = "cursor"
	agentCodexCLI   = "codex_cli"
	agentGeminiCLI  = "gemini_cli"
)

// defaultMaxAge is how far back transcripts are read when the query doesn't
// bound ended_at/timestamp from below.
const defaultMaxAge = 30 * 24 * time.Hour

var (
	listHomes = homes.All
	now       = time.Now
)

// transcriptsParsed counts sessions read from disk (cache misses included,
// cache hits not), for the debug log.
var transcriptsParsed atomic.Int64

// discoveryReads counts files read during discovery to learn which session
// they belong to (cache misses only), for tests.
var discoveryReads atomic.Int64

// agent is one supported AI agent: where its sessions are, how to read one,
// and which account it is signed in with.
type agent struct {
	name     string
	discover func(a *agent, sc *scope, cutoff time.Time) []*source
	parse    func(src *source, r *recorder)
	account  func(sc *scope) account
}

// maxSourcesPerAgent bounds how many sessions of one agent in one home a
// call looks at; far above any real history, it keeps a planted tree from
// turning into unbounded work.
const maxSourcesPerAgent = 5000

var agents = []*agent{claudeCode, copilotCLI, cursorAgent, codexCLI, geminiCLI}

// SessionsColumns returns the ai_agent_sessions schema.
func SessionsColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("agent"),
		table.TextColumn("session_id"),
		table.TextColumn("username"),
		table.TextColumn("account_identifier"),
		table.TextColumn("project_path"),
		table.TextColumn("git_branch"),
		table.TextColumn("git_repository"),
		table.TextColumn("agent_version"),
		table.TextColumn("model"),
		table.BigIntColumn("started_at"),
		table.BigIntColumn("ended_at"),
		table.BigIntColumn("duration_seconds"),
		table.BigIntColumn("user_messages"),
		table.BigIntColumn("assistant_messages"),
		table.BigIntColumn("tool_calls"),
		table.BigIntColumn("input_tokens"),
		table.BigIntColumn("output_tokens"),
		table.BigIntColumn("cache_read_tokens"),
		table.BigIntColumn("cache_write_tokens"),
		table.TextColumn("transcript_path"),
		table.BigIntColumn("transcript_size_bytes"),
	}
}

// MessagesColumns returns the ai_agent_session_messages schema.
func MessagesColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("agent"),
		table.TextColumn("session_id"),
		table.TextColumn("username"),
		table.BigIntColumn("message_index"),
		table.BigIntColumn("block_index"),
		table.BigIntColumn("tool_call_index"),
		table.TextColumn("role"),
		table.TextColumn("block_type"),
		table.TextColumn("tool_name"),
		table.TextColumn("tool_use_id"),
		table.TextColumn("action"),
		table.TextColumn("target"),
		table.TextColumn("decision"),
		table.TextColumn("cwd"),
		table.BigIntColumn("timestamp"),
		table.TextColumn("model"),
		table.TextColumn("content"),
		table.BigIntColumn("content_length"),
		table.IntegerColumn("truncated"),
		table.TextColumn("transcript_path"),
	}
}

// GenerateSessions produces the ai_agent_sessions rows. Like every table in
// the root orbit daemon that parses files users control, a panic becomes a
// query error instead of taking the extension down.
func GenerateSessions(_ context.Context, qc table.QueryContext) (rows []map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			rows, err = nil, fmt.Errorf("ai_agent_sessions: recovered from panic: %v", r)
		}
	}()
	return generateSessions(qc), nil
}

// GenerateMessages produces the ai_agent_session_messages rows.
func GenerateMessages(_ context.Context, qc table.QueryContext) (rows []map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			rows, err = nil, fmt.Errorf("ai_agent_session_messages: recovered from panic: %v", r)
		}
	}()
	return generateMessages(qc), nil
}

func generateSessions(qc table.QueryContext) []map[string]string {
	started := now()
	q := parseQuery(qc, "ended_at", started)
	parsedBefore := transcriptsParsed.Load()
	gen := derivedGen.Load()

	// Only a scan that saw every session can tell which cache entries are gone.
	complete := len(q.agents) == 0 && len(q.usernames) == 0 && len(q.sessionIDs) == 0 && q.defaultWindow
	seen := map[string]struct{}{}

	reads := newReadBudget()
	var sessions []Session
	homes := eachAgent(&q, reads, func(sc *scope, a *agent, srcs []*source) {
		var acct *account
		for _, src := range srcs {
			if src.id != "" && !q.sessionIDs.allows(src.id) {
				continue
			}
			s, ok := cache.session(src, seen)
			if !ok || !q.sessionIDs.allows(s.SessionID) {
				continue
			}
			if acct == nil {
				acct = new(a.account(sc))
			}
			s.account = *acct
			sessions = append(sessions, s)
		}
	})
	// A scan the budget cut hasn't seen everything, and must not evict.
	complete = complete && !reads.exhausted
	if complete {
		cache.evict(seen)
		sweepDerived(gen)
	}

	slices.SortStableFunc(sessions, func(a, b Session) int {
		return cmp.Or(b.EndedAt.Compare(a.EndedAt), cmp.Compare(a.SessionID, b.SessionID))
	})
	rows := make([]map[string]string, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, s.row(s.account))
	}

	if reads.exhausted {
		log.Warn().Int64("max_bytes", maxReadBytesPerCall).Msg("ai_agent_sessions: read budget exhausted; the rest is read next time")
	}
	log.Debug().Int("homes", homes).Int("sessions", len(rows)).
		Int64("parsed", transcriptsParsed.Load()-parsedBefore).Dur("took", now().Sub(started)).
		Msg("ai_agent_sessions: scan done")
	return rows
}

func generateMessages(qc table.QueryContext) []map[string]string {
	started := now()
	q := parseQuery(qc, "timestamp", started)
	// Naming a session asks for that conversation, however old.
	if len(q.sessionIDs) > 0 {
		q.cutoff = time.Time{}
		q.maxContent = maxNamedBlockBytes
	}

	messageBudget.begin(started)
	defer func() { messageBudget.end(now()) }()

	reads := newReadBudget()
	var srcs []*source
	homes := eachAgent(&q, reads, func(_ *scope, _ *agent, found []*source) {
		for _, src := range found {
			// A transcript_path names a session by its main transcript and
			// selects all of it; every row reports that path (see add).
			if q.sessionIDs.allows(src.id) && q.paths.allows(src.files[0]) {
				srcs = append(srcs, src)
			}
		}
	})

	// Newest first, so a result cut at the size limit is the latest activity.
	slices.SortStableFunc(srcs, func(a, b *source) int {
		return cmp.Or(b.modTime.Compare(a.modTime), cmp.Compare(a.files[0], b.files[0]))
	})

	rows := []map[string]string{}
	for _, src := range srcs {
		if reads.exhausted {
			break
		}
		r := newMessageRecorder(src, &q)
		transcriptsParsed.Add(1)
		if !parse(src, r) {
			continue
		}
		for _, m := range r.out {
			rows = append(rows, m.row())
		}
		if r.cut {
			ev := log.Warn().Int("max_bytes", maxResultBytes).Str("session_id", logField(src.id)).Int64("next_tool_call_index", r.nextCall)
			if n := len(r.out); n > 0 {
				ev = ev.Int64("message_index", r.out[n-1].MessageIndex)
			}
			ev.Msg("ai_agent_session_messages: result cut at the size limit; to read on, query the session with block_type = 'tool_use' AND tool_call_index >= next_tool_call_index, or with message_index >= the last row's and drop the rows already read")
			break
		}
	}

	if reads.exhausted {
		log.Warn().Int64("max_bytes", maxReadBytesPerCall).Msg("ai_agent_session_messages: read budget exhausted; result is partial")
	}
	log.Debug().Int("homes", homes).Int("transcripts", len(srcs)).Int("rows", len(rows)).
		Dur("took", now().Sub(started)).Msg("ai_agent_session_messages: scan done")
	return rows
}

// eachAgent discovers the sessions of every agent in every home the query
// allows, calling fn with each agent's (bounded) sources, and returns how
// many homes it saw.
func eachAgent(q *query, reads *readBudget, fn func(sc *scope, a *agent, srcs []*source)) int {
	hs := listHomes()
	for _, h := range hs {
		if !q.usernames.allows(h.Username) {
			continue
		}
		sc := &scope{home: h, reads: reads}
		for _, a := range agents {
			if !q.agents.allows(a.name) {
				continue
			}
			srcs := a.discover(a, sc, q.cutoff)
			if len(srcs) > maxSourcesPerAgent {
				log.Warn().Str("agent", a.name).Str("username", h.Username).Int("sessions", len(srcs)).
					Int("max", maxSourcesPerAgent).Msg("ai_agent_sessions: too many sessions; some are skipped")
				srcs = srcs[:maxSourcesPerAgent]
			}
			fn(sc, a, srcs)
		}
	}
	return len(hs)
}

// parse runs an agent's parser over a source. A transcript that trips a bug
// in its parser costs that session, not the table.
func parse(src *source, r *recorder) (ok bool) {
	defer func() {
		if p := recover(); p != nil {
			log.Warn().Str("agent", src.agent.name).Str("path", src.files[0]).Interface("panic", p).
				Msg("ai_agent_sessions: parser failed; session skipped")
			ok = false
		}
	}()
	src.agent.parse(src, r)
	return true
}

// logField caps a value from a file before it goes in a log line.
func logField(s string) string { return truncateUTF8(s, 128) }

// set is the values of a column's equality constraints. Empty means the
// column is unconstrained.
type set map[string]struct{}

func (s set) has(v string) bool {
	_, ok := s[v]
	return ok
}

func (s set) allows(v string) bool {
	if len(s) == 0 {
		return true
	}
	_, ok := s[v]
	return ok
}

// query is what a QueryContext asks of the tables.
//
// LIMIT and OFFSET are deliberately ignored: osquery passes them down even
// when it still has to sort or filter the rows itself, so honouring them here
// would return the wrong rows.
type query struct {
	agents, usernames, sessionIDs, paths, blockTypes set

	// cutoff is the oldest transcript modification time worth reading;
	// defaultWindow says it is the default rather than the query's.
	cutoff        time.Time
	defaultWindow bool
	// messages bounds message_index; calls bounds tool_call_index, and when
	// set at all, only tool_use rows can match.
	messages, calls bounds
	// blockLo bounds block_index on message messages.lo, to resume inside a
	// message the size limit cut.
	blockLo int64
	// maxContent caps each content block, in bytes.
	maxContent int
	// rows counts the rows this call has returned so far.
	rows int
}

// bounds is the range the constraints on an index column allow.
type bounds struct {
	lo, hi int64
	hasHi  bool
	// set reports whether the column was constrained at all.
	set bool
}

func (b *bounds) atLeast(v int64) { b.lo = max(b.lo, v) }

func (b *bounds) atMost(v int64) {
	if !b.hasHi || v < b.hi {
		b.hi, b.hasHi = v, true
	}
}

func (b bounds) allows(v int64) bool { return v >= b.lo && (!b.hasHi || v <= b.hi) }

// parseBounds reads the single-column comparisons on column.
func parseBounds(qc table.QueryContext, column string) bounds {
	var b bounds
	for _, c := range qc.Constraints[column].Constraints {
		v, err := strconv.ParseInt(c.Expression, 10, 64)
		if err != nil {
			continue
		}
		b.set = true
		switch c.Operator {
		case table.OperatorEquals:
			b.atLeast(v)
			b.atMost(v)
		case table.OperatorGreaterThan:
			b.atLeast(v + 1)
		case table.OperatorGreaterThanOrEquals:
			b.atLeast(v)
		case table.OperatorLessThan:
			b.atMost(v - 1)
		case table.OperatorLessThanOrEquals:
			b.atMost(v)
		}
	}
	return b
}

func parseQuery(qc table.QueryContext, timeColumn string, at time.Time) query {
	q := query{
		agents:        equalities(qc, "agent"),
		usernames:     equalities(qc, "username"),
		sessionIDs:    equalities(qc, "session_id"),
		paths:         equalities(qc, "transcript_path"),
		blockTypes:    equalities(qc, "block_type"),
		cutoff:        at.Add(-defaultMaxAge),
		defaultWindow: true,
		maxContent:    maxBlockBytes,
	}

	// A lower bound on the time column replaces the default age; an upper
	// bound alone asks for history, so it lifts it.
	var lower *int64
	var upper bool
	for _, c := range qc.Constraints[timeColumn].Constraints {
		v, err := strconv.ParseInt(c.Expression, 10, 64)
		if err != nil {
			continue
		}
		switch c.Operator {
		case table.OperatorEquals, table.OperatorGreaterThan, table.OperatorGreaterThanOrEquals:
			if lower == nil || v > *lower {
				lower = &v
			}
		case table.OperatorLessThan, table.OperatorLessThanOrEquals:
			upper = true
		}
	}
	switch {
	case lower != nil && *lower > 0:
		q.cutoff, q.defaultWindow = time.Unix(*lower, 0), false
	case lower != nil || upper:
		q.cutoff, q.defaultWindow = time.Time{}, false
	}

	q.messages = parseBounds(qc, "message_index")
	q.calls = parseBounds(qc, "tool_call_index")
	for _, c := range qc.Constraints["block_index"].Constraints {
		v, err := strconv.ParseInt(c.Expression, 10, 64)
		if err != nil {
			continue
		}
		switch c.Operator {
		case table.OperatorEquals, table.OperatorGreaterThanOrEquals:
			q.blockLo = max(q.blockLo, v)
		case table.OperatorGreaterThan:
			q.blockLo = max(q.blockLo, v+1)
		}
	}
	return q
}

// equalities collects a column's `=` constraints; osquery turns an IN list
// into one call per value, each with a single equality.
func equalities(qc table.QueryContext, column string) set {
	var s set
	for _, c := range qc.Constraints[column].Constraints {
		if c.Operator != table.OperatorEquals {
			continue
		}
		if s == nil {
			s = set{}
		}
		s[c.Expression] = struct{}{}
	}
	return s
}
