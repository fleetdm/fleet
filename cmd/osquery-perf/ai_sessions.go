package main

import (
	"errors"
	"fmt"
	"log"
	// osquery-perf shares one global math/rand RNG seeded from the --seed flag so load-test runs are reproducible
	"math/rand" //nolint:depguard
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// fleetd's limits on ai_agent_session_messages, which decide how many refreshes a session's tool calls take.
const (
	// aiSessionMaxContent is the cut fleetd applies to each content value when sessions are named by transcript_path.
	aiSessionMaxContent = 8 << 10
	// aiSessionMaxResultBytes is fleetd's budget for one query's result, shared by all its table calls.
	aiSessionMaxResultBytes = 1 << 20
	// aiSessionMaxPerAgent is how many sessions per agent fleetd reads; it skips the rest.
	aiSessionMaxPerAgent = 5000
	// aiSessionWindow is how far back fleetd reports sessions by default.
	aiSessionWindow = 30 * 24 * time.Hour
)

// aiSessionsConfig sets how many AI agent sessions the simulated fleetd hosts report, and how many of them can't be
// collected in full.
type aiSessionsConfig struct {
	// prob is the share of fleetd hosts that report sessions.
	prob float64
	// initialCount is how many sessions, from the last 30 days, a host has when it starts.
	initialCount int
	// perDay is how many new sessions a host gets each day.
	perDay float64
	// longProb is the share of sessions with thousands of tool calls, which take several refreshes to collect.
	longProb float64
	// deletedProb is the share of sessions whose transcript is deleted from the host after Fleet collected the session.
	deletedProb float64
	// uncountedProb is the share of sessions whose tool call count includes calls fleetd doesn't return.
	uncountedProb float64
}

func (c aiSessionsConfig) validate() error {
	for _, p := range []float64{c.prob, c.longProb, c.deletedProb, c.uncountedProb} {
		if p < 0 || p > 1 {
			return errors.New("ai_sessions probabilities must be between 0 and 1")
		}
	}
	if c.initialCount < 0 || c.perDay < 0 {
		return errors.New("ai_sessions_initial_count and ai_sessions_per_day cannot be negative")
	}
	if float64(c.initialCount)+c.perDay*aiSessionWindow.Hours()/24 > aiSessionMaxPerAgent {
		return fmt.Errorf("ai_sessions_initial_count plus 30 days of ai_sessions_per_day must stay under fleetd's %d sessions per agent", aiSessionMaxPerAgent)
	}
	return nil
}

// aiSessionMix returns a well-mixed value for (seed, i), so a session's details are derived rather than stored.
func aiSessionMix(seed, i uint64) uint64 {
	z := seed + 0x9e3779b97f4a7c15*(i+1)
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// Streams of aiSessionMix values, so the derived details don't correlate.
const (
	aiMixGroupSize = iota + 1
	aiMixKind
	aiMixSize
	aiMixDecision
	aiMixID
	aiMixDetail
)

// mix returns the session's value for item i (a tool call, a message group, ...) of a stream.
func (s *aiSession) mix(stream uint64, i int) uint64 {
	return aiSessionMix(s.seed, uint64(i)<<8|stream) //nolint:gosec // i is a non-negative index
}

// aiSessionRepos are the projects sessions run in. One has a quote, which the events query must escape.
var aiSessionRepos = []string{"api", "web", "infra", "docs", "o'brien-tools"}

// aiSession is a simulated session. Its tool calls are written over time, one every gap from started, so a session
// still running grows and its ended_at moves at each refresh.
type aiSession struct {
	seed    uint64
	agent   int // index into fleet.AISessionAgents
	repo    int // index into aiSessionRepos
	started time.Time
	gap     time.Duration
	target  int // tool calls once the session is over
	long    bool
	// uncounted is how many tool calls fleetd counts but doesn't return.
	uncounted int
	// deleteAfter is how many times the session is reported before its transcript is deleted, 0 for never.
	deleteAfter int
	reported    int

	home, sep string
}

func (s *aiSession) agentName() string { return fleet.AISessionAgents[s.agent] }

func (s *aiSession) cursor() bool { return s.agentName() == fleet.AISessionAgentCursor }

func (s *aiSession) id() string {
	a, b := s.mix(aiMixID, 0), s.mix(aiMixID, 1)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", a>>32, a>>16&0xffff, a&0xffff, b>>48, b&0xffffffffffff)
}

func (s *aiSession) projectPath() string {
	return s.home + "src" + s.sep + aiSessionRepos[s.repo]
}

func (s *aiSession) transcriptPath() string {
	p := func(parts ...string) string { return s.home + strings.Join(parts, s.sep) }
	switch s.agentName() {
	case "claude_code":
		return p(".claude", "projects", strings.NewReplacer("/", "-", `\`, "-", ":", "-").Replace(s.projectPath()), s.id()+".jsonl")
	case "copilot_cli":
		return p(".copilot", "session-state", s.id()+".jsonl")
	case "cursor":
		return p(".cursor", "chats", fmt.Sprintf("%016x", s.mix(aiMixDetail, 0)), s.id(), "store.db")
	case "codex_cli":
		return p(".codex", "sessions", s.started.Format("2006"+s.sep+"01"+s.sep+"02"), "rollout-"+s.started.Format("2006-01-02T15-04-05")+"-"+s.id()+".jsonl")
	default:
		return p(".gemini", "tmp", fmt.Sprintf("%016x", s.mix(aiMixDetail, 0)), "chats", "session-"+s.id()+".json")
	}
}

// callsAt is how many tool calls the session has written by now.
func (s *aiSession) callsAt(now time.Time) int {
	if now.Before(s.started) {
		return 0
	}
	return min(s.target, int(now.Sub(s.started)/s.gap))
}

func (s *aiSession) active(now time.Time) bool { return s.callsAt(now) < s.target }

// endedAt is the time of the session's last write.
func (s *aiSession) endedAt(now time.Time) time.Time {
	return s.started.Add(time.Duration(s.callsAt(now)) * s.gap)
}

// forEachCall calls fn with the position of each of the first n tool calls, until it returns false. An assistant
// message holds one to three calls after its text block, and is followed by a message with their results.
func (s *aiSession) forEachCall(n int, fn func(i int, msg, block uint) bool) {
	for i, g := 0, uint(0); i < n; g++ {
		size := uint(1 + s.mix(aiMixGroupSize, int(g))%3) //nolint:gosec // g counts the groups of at most n calls
		for b := uint(1); b <= size && i < n; b++ {
			if !fn(i, 1+2*g, b) {
				return
			}
			i++
		}
	}
}

// aiSessionFiller is code-like text, without characters JSON escapes, so any prefix of it is a valid string body.
var aiSessionFiller = strings.Repeat("func handle(ctx context.Context, req *Request) (*Response, error) { return svc.Do(ctx, req) } ", 100)

type aiToolCall struct {
	name, action, target, content string
}

func (s *aiSession) toolCall(i int) aiToolCall {
	k := s.mix(aiMixKind, i)
	rawFile := s.projectPath() + s.sep + fmt.Sprintf("pkg%d%sfile%d.go", k%20, s.sep, k%97)
	file := strings.ReplaceAll(rawFile, `\`, `\\`) // as a JSON string body
	size := int(200 + s.mix(aiMixSize, i)%3000)
	if s.long {
		size = int(1000 + s.mix(aiMixSize, i)%7000)
	}
	filler := func(n int) string { return aiSessionFiller[:min(n, len(aiSessionFiller))] }

	switch k % 20 {
	case 0, 1, 2, 3, 4, 5, 6:
		cmd := []string{"go test ./server/...", "make lint-go-incremental", "git status", "yarn test", "ls -la"}[k%5]
		return aiToolCall{"Bash", "execute", cmd, fmt.Sprintf(`{"command":%q,"description":"Run %s"}`, cmd, cmd)}
	case 7, 8, 9, 10, 11:
		return aiToolCall{"Read", "read", rawFile, fmt.Sprintf(`{"file_path":"%s"}`, file)}
	case 12, 13, 14:
		half := filler(size / 2)
		return aiToolCall{"Edit", "write", rawFile, fmt.Sprintf(`{"file_path":"%s","old_string":"%s","new_string":"%s"}`, file, half, half)}
	case 15:
		return aiToolCall{"Write", "write", rawFile, fmt.Sprintf(`{"file_path":"%s","content":"%s"}`, file, filler(size))}
	case 16, 17:
		return aiToolCall{"Grep", "search", "func handle", `{"pattern":"func handle","path":"."}`}
	case 18:
		return aiToolCall{"WebFetch", "network", "https://fleetdm.com/docs", `{"url":"https://fleetdm.com/docs","prompt":"Summarize"}`}
	default:
		return aiToolCall{"mcp__github__create_issue", "mcp", "github: create_issue", `{"owner":"fleetdm","repo":"fleet","title":"Load test"}`}
	}
}

// decision is the outcome of call i: none yet for the calls of the latest message of a session still running, and
// never for Cursor, which records none.
func (s *aiSession) decision(i int, msg, lastMsg uint, active bool) string {
	if s.cursor() || (active && msg == lastMsg) {
		return ""
	}
	switch d := s.mix(aiMixDecision, i) % 100; {
	case d < 90:
		return "allowed"
	case d < 94:
		return "denied"
	default:
		return "error"
	}
}

func (s *aiSession) timestamp(i int) int64 {
	ts := s.started.Add(time.Duration(i+1) * s.gap).Unix()
	if s.cursor() {
		ts -= ts % 60
	}
	return ts
}

var (
	aiSessionVersions = map[string]string{
		"claude_code": "2.0.15", "copilot_cli": "0.0.339", "cursor": "1.7.46", "codex_cli": "0.46.0", "gemini_cli": "0.9.0",
	}
	aiSessionModels = map[string]string{
		"claude_code": "claude-opus-5-5", "copilot_cli": "claude-sonnet-5-5", "cursor": "auto", "codex_cli": "gpt-5-codex",
		"gemini_cli": "gemini-2.5-pro",
	}
)

func (s *aiSession) accountIdentifier() string {
	switch s.agentName() {
	case "claude_code":
		if s.mix(aiMixDetail, 1)%10 == 0 {
			return "API KEY"
		}
		return "fleet@example.com"
	case "codex_cli", "cursor":
		return "fleet@example.com"
	default:
		return ""
	}
}

func (s *aiSession) row(now time.Time) map[string]string {
	calls := s.callsAt(now)
	var contentBytes int
	if !s.long { // a long session's size is estimated rather than generated at every refresh
		s.forEachCall(calls, func(i int, _, _ uint) bool {
			contentBytes += len(s.toolCall(i).content)
			return true
		})
	} else {
		contentBytes = calls * 4500
	}
	groups := 0
	s.forEachCall(calls, func(_ int, msg, _ uint) bool {
		groups = int(msg/2) + 1
		return true
	})
	branch := "main"
	if b := s.mix(aiMixDetail, 2) % 4; b > 0 {
		branch = fmt.Sprintf("feature-%d", b)
	}
	u := strconv.Itoa
	return map[string]string{
		"agent":                 s.agentName(),
		"session_id":            s.id(),
		"agent_version":         aiSessionVersions[s.agentName()],
		"username":              "fleet",
		"account_identifier":    s.accountIdentifier(),
		"model":                 aiSessionModels[s.agentName()],
		"project_path":          s.projectPath(),
		"git_repository":        "github.com/example/" + aiSessionRepos[s.repo],
		"git_branch":            branch,
		"started_at":            strconv.FormatInt(s.started.Unix(), 10),
		"ended_at":              strconv.FormatInt(s.endedAt(now).Unix(), 10),
		"user_messages":         u(groups/4 + 1),
		"assistant_messages":    u(groups),
		"tool_calls":            u(calls + s.uncounted),
		"input_tokens":          u(calls * 3000),
		"output_tokens":         u(calls * 400),
		"cache_read_tokens":     u(calls * 40000),
		"cache_write_tokens":    u(calls * 2000),
		"transcript_path":       s.transcriptPath(),
		"transcript_size_bytes": u(contentBytes * 3),
	}
}

// aiSessionsSim is a host's AI agent sessions, answering the ai_agent_sessions and ai_agent_session_events detail
// queries as fleetd would.
type aiSessionsSim struct {
	mu       sync.Mutex
	cfg      aiSessionsConfig
	rng      *rand.Rand
	home     string
	sep      string
	sessions []*aiSession
	// last and pending track arrivals of new sessions between refreshes.
	last    time.Time
	pending float64
	// lastEventsBytes is the size fleetd counted for the last events result.
	lastEventsBytes int
}

func newAISessionsSim(cfg aiSessionsConfig, agentOS string, seed int64, now time.Time) *aiSessionsSim {
	s := &aiSessionsSim{cfg: cfg, rng: rand.New(rand.NewSource(seed)), last: now, sep: "/"} //nolint:gosec // ignore weak randomizer
	switch agentOS {
	case "windows":
		s.home, s.sep = `C:\Users\fleet\`, `\`
	case "macos":
		s.home = "/Users/fleet/"
	default:
		s.home = "/home/fleet/"
	}
	// Spread over a bit less than fleetd's window, so a starting host doesn't lose sessions to it right away.
	for range cfg.initialCount {
		s.newSession(now.Add(-time.Duration(s.rng.Int63n(int64(28 * 24 * time.Hour)))))
	}
	return s
}

// aiSessionAgentWeights is how often each agent of fleet.AISessionAgents starts a session.
var aiSessionAgentWeights = []int{50, 5, 20, 20, 5}

func (s *aiSessionsSim) newSession(started time.Time) *aiSession {
	agent, w := 0, s.rng.Intn(100)
	for ; w >= aiSessionAgentWeights[agent]; agent++ {
		w -= aiSessionAgentWeights[agent]
	}
	long := s.rng.Float64() < s.cfg.longProb
	gap := time.Duration(5+s.rng.Intn(55)) * time.Second
	target := 1 + min(int(s.rng.ExpFloat64()*40), 500)
	if long {
		gap = time.Duration(3+s.rng.Intn(17)) * time.Second
		target = 1000 + s.rng.Intn(3000)
	}
	sess := s.addSession(agent, started, gap, target, long)
	sess.repo = s.rng.Intn(len(aiSessionRepos))
	if s.rng.Float64() < s.cfg.uncountedProb {
		sess.uncounted = 1 + s.rng.Intn(3)
	}
	if s.rng.Float64() < s.cfg.deletedProb {
		sess.deleteAfter = 1 + s.rng.Intn(3)
	}
	return sess
}

func (s *aiSessionsSim) addSession(agent int, started time.Time, gap time.Duration, target int, long bool) *aiSession {
	sess := &aiSession{
		seed: s.rng.Uint64(), agent: agent, started: started, gap: gap, target: target, long: long,
		home: s.home, sep: s.sep,
	}
	s.sessions = append(s.sessions, sess)
	return sess
}

func (s *aiSessionsSim) byID(id string) *aiSession {
	for _, sess := range s.sessions {
		if sess.id() == id {
			return sess
		}
	}
	return nil
}

// sessionsRows answers the ai_agent_sessions detail query: the sessions written in the last 30 days. A session whose
// transcript is to be deleted is gone once it was reported deleteAfter times.
func (s *aiSessionsSim) sessionsRows(now time.Time) []map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if now.After(s.last) {
		s.pending += s.cfg.perDay * now.Sub(s.last).Hours() / 24
		for ; s.pending >= 1; s.pending-- {
			s.newSession(s.last.Add(time.Duration(s.rng.Int63n(int64(now.Sub(s.last)) + 1))))
		}
		s.last = now
	}

	rows := make([]map[string]string, 0, len(s.sessions))
	kept := s.sessions[:0]
	for _, sess := range s.sessions {
		if sess.endedAt(now).Before(now.Add(-aiSessionWindow)) || (sess.deleteAfter > 0 && sess.reported >= sess.deleteAfter) {
			continue
		}
		kept = append(kept, sess)
		if sess.started.After(now) {
			continue
		}
		rows = append(rows, sess.row(now))
		sess.reported++
	}
	clear(s.sessions[len(kept):])
	s.sessions = kept
	return rows
}

// The parts of Fleet's events query (see buildAIAgentSessionEventsQuery): per session, a constant row naming it as
// requested, then its tool calls from a resume point.
var (
	aiRequestedArmRe = regexp.MustCompile(`(?s)^SELECT '((?:[^']|'')*)' AS agent, '((?:[^']|'')*)' AS session_id, .*?(-?\d+) AS session_ended_at, 1 AS requested$`)
	aiEventsArmRe    = regexp.MustCompile(`(?s)^SELECT agent, .*?(-?\d+) AS session_ended_at, 0 AS requested\nFROM ai_agent_session_messages\nWHERE transcript_path = '((?:[^']|'')*)' AND block_type = 'tool_use'(.*)$`)
	aiResumeRe       = regexp.MustCompile(`^ AND tool_call_index >= (\d+)$`)
)

func unquoteSQLite(s string) string { return strings.ReplaceAll(s, "''", "'") }

// aiSessionBudget is fleetd's shared result budget: a table call's rows stop once it's spent, but each call still
// returns its first row until twice the budget is used.
type aiSessionBudget struct{ used int }

func (b *aiSessionBudget) take(n int, first bool) bool {
	if b.used+n > aiSessionMaxResultBytes && (!first || b.used >= 2*aiSessionMaxResultBytes) {
		b.used = max(b.used, aiSessionMaxResultBytes)
		return false
	}
	b.used += n
	return true
}

// aiSessionWireSize is the size fleetd counts for a row: every column of its messages table, JSON encoded.
func aiSessionWireSize(cols ...string) int {
	n := 2
	for _, c := range cols {
		n += len(c) + strings.Count(c, `"`) + strings.Count(c, `\`) + 6
	}
	// The column names.
	return n + 170
}

// eventsRows answers the ai_agent_session_events detail query. Each UNION ALL arm is one table call, and all calls share
// fleetd's result budget. fleetd applies the tool_call_index bound itself, so calls before it cost nothing.
func (s *aiSessionsSim) eventsRows(query string, now time.Time) []map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	byPath := make(map[string]*aiSession, len(s.sessions))
	for _, sess := range s.sessions {
		byPath[sess.transcriptPath()] = sess
	}

	var (
		rows   []map[string]string
		budget aiSessionBudget
	)
	for arm := range strings.SplitSeq(query, "\nUNION ALL\n") {
		if m := aiRequestedArmRe.FindStringSubmatch(arm); m != nil {
			rows = append(rows, map[string]string{
				"agent": unquoteSQLite(m[1]), "session_id": unquoteSQLite(m[2]), "tool_call_index": "0",
				"tool_name": "", "tool_use_id": "", "action": "", "target": "", "decision": "", "timestamp": "0",
				"content": "", "session_ended_at": m[3], "requested": "1",
			})
			continue
		}
		m := aiEventsArmRe.FindStringSubmatch(arm)
		if m == nil {
			log.Printf("ai_agent_session_events: unrecognized query part: %q", arm)
			continue
		}
		sess := byPath[unquoteSQLite(m[2])]
		if sess == nil || sess.started.After(now) {
			continue
		}

		var from int
		if resume := m[3]; resume != "" {
			r := aiResumeRe.FindStringSubmatch(resume)
			if r == nil {
				log.Printf("ai_agent_session_events: unrecognized resume point: %q", resume)
				continue
			}
			from, _ = strconv.Atoi(r[1])
		}

		n := sess.callsAt(now)
		active := sess.active(now)
		var lastMsg uint
		sess.forEachCall(n, func(_ int, msg, _ uint) bool { lastMsg = msg; return true })
		toolUseID := func(i int) string {
			if sess.cursor() {
				return ""
			}
			return fmt.Sprintf("toolu_%016x", sess.mix(aiMixID, i+2))
		}
		{
			first := true
			sess.forEachCall(n, func(i int, msg, block uint) bool {
				if i < from {
					return true
				}
				call := sess.toolCall(i)
				row := map[string]string{
					"agent": sess.agentName(), "session_id": sess.id(),
					"tool_call_index": strconv.Itoa(i),
					"tool_name":       call.name, "tool_use_id": toolUseID(i), "action": call.action, "target": call.target,
					"decision": sess.decision(i, msg, lastMsg, active), "timestamp": strconv.FormatInt(sess.timestamp(i), 10),
					"content": call.content, "session_ended_at": m[1], "requested": "0",
				}
				size := aiSessionWireSize(row["agent"], row["session_id"], "fleet", strconv.FormatUint(uint64(msg), 10),
					strconv.FormatUint(uint64(block), 10), row["tool_call_index"],
					"assistant", "tool_use", row["tool_name"], row["tool_use_id"], row["action"], row["target"], row["decision"],
					sess.projectPath(), row["timestamp"], aiSessionModels[sess.agentName()], row["content"],
					strconv.Itoa(len(row["content"])), "0", sess.transcriptPath())
				if !budget.take(size, first) {
					return false
				}
				first = false
				rows = append(rows, row)
				return true
			})
		}
	}
	s.lastEventsBytes = budget.used
	return rows
}
