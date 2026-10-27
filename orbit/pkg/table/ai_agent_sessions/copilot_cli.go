package ai_agent_sessions

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

// GitHub Copilot CLI writes one event log per session to
// ~/.copilot/session-state/<session-id>/events.jsonl, in the shape of its
// published session-events schema. Sub-agent events are in the same file,
// tagged with agentId.
var copilotCLI = &agent{
	name:     agentCopilotCLI,
	discover: discoverCopilotCLI,
	parse:    parseCopilotCLI,
	account:  copilotAccount,
}

func discoverCopilotCLI(a *agent, sc *scope, cutoff time.Time) []*source {
	dir := filepath.Join(sc.home.Dir, ".copilot", "session-state")
	var srcs []*source
	for _, id := range subdirs(dir) {
		if src, ok := newSource(a, sc, id, []string{filepath.Join(dir, id, "events.jsonl")}, cutoff); ok {
			srcs = append(srcs, src)
		}
	}
	return srcs
}

type copilotTokens struct {
	TokenCount int64 `json:"tokenCount"`
}

type copilotEvent struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	AgentID   string      `json:"agentId"`
	Data      copilotData `json:"data"`
}

type copilotData struct {
	// session.start, session.context_changed
	CopilotVersion string          `json:"copilotVersion"`
	StartTime      string          `json:"startTime"`
	SelectedModel  string          `json:"selectedModel"`
	Context        *copilotContext `json:"context"`
	CWD            string          `json:"cwd"`
	Branch         string          `json:"branch"`
	Repository     string          `json:"repository"`

	// user.message, assistant.message
	Content                 json.RawMessage `json:"content"`
	Source                  string          `json:"source"`
	IsAutopilotContinuation bool            `json:"isAutopilotContinuation"`
	MessageID               string          `json:"messageId"`
	Model                   string          `json:"model"`
	NewModel                string          `json:"newModel"`

	// session.shutdown
	CurrentModel string                   `json:"currentModel"`
	TokenDetails map[string]copilotTokens `json:"tokenDetails"`
	ModelMetrics map[string]struct {
		Usage struct {
			InputTokens      int64 `json:"inputTokens"`
			OutputTokens     int64 `json:"outputTokens"`
			CacheReadTokens  int64 `json:"cacheReadTokens"`
			CacheWriteTokens int64 `json:"cacheWriteTokens"`
		} `json:"usage"`
	} `json:"modelMetrics"`

	// tool.execution_start, tool.execution_complete, permission.completed
	ToolCallID    string          `json:"toolCallId"`
	ToolName      string          `json:"toolName"`
	MCPServerName string          `json:"mcpServerName"`
	MCPToolName   string          `json:"mcpToolName"`
	Arguments     json.RawMessage `json:"arguments"`
	Success       *bool           `json:"success"`
	Result        json.RawMessage `json:"result"`
	Error         *struct {
		Message string `json:"message"`
	} `json:"error"`
	ShellExecution *copilotShell `json:"shellExecution"`
}

type copilotShell struct {
	ExitCode *int `json:"exitCode"`
}

type copilotContext struct {
	CWD        string `json:"cwd"`
	Branch     string `json:"branch"`
	Repository string `json:"repository"`
}

func parseCopilotCLI(src *source, r *recorder) {
	s := r.s
	var lastModel, shutdownModel, selectedModel, cwd string
	assistantIDs := map[string]struct{}{}

	r.file(src.files[0])
	_ = scanJSONL(src.sc, src.files[0], func(line []byte) bool {
		if r.done() {
			return false
		}
		var ev copilotEvent
		if json.Unmarshal(line, &ev) != nil {
			return true
		}
		r.seen = true
		d := &ev.Data
		ts := parseTime(ev.Timestamp)
		s.observe(ts)

		switch ev.Type {
		case "session.start":
			s.AgentVersion = d.CopilotVersion
			selectedModel = d.SelectedModel
			s.observe(parseTime(d.StartTime))
			if c := d.Context; c != nil {
				cwd, s.ProjectPath, s.GitBranch, s.GitRepository = c.CWD, c.CWD, c.Branch, sanitizeRepoURL(c.Repository)
			}
		case "session.context_changed":
			c := firstContext(d.Context, copilotContext{CWD: d.CWD, Branch: d.Branch, Repository: d.Repository})
			cwd = firstNonEmpty(c.CWD, cwd)
			s.GitBranch = firstNonEmpty(c.Branch, s.GitBranch)
			s.GitRepository = firstNonEmpty(sanitizeRepoURL(c.Repository), s.GitRepository)
		case "session.model_change":
			lastModel = firstNonEmpty(d.NewModel, lastModel)
		case "user.message":
			// Sub-agent instructions, skill text and autopilot nudges arrive
			// as user messages but nobody typed them.
			role := roleSystem
			if ev.AgentID == "" && !strings.HasPrefix(d.Source, "skill-") && !strings.HasPrefix(d.Source, "agent-") &&
				!d.IsAutopilotContinuation {
				s.UserMessages++
				role = roleUser
			}
			r.next()
			r.at(ts, "", cwd)
			r.text(role, blockText, func() string { return textOf(d.Content) })
		case "assistant.message":
			lastModel = firstNonEmpty(d.Model, lastModel)
			if rawString(d.Content) == "" || rawString(d.Content) == `""` {
				return true
			}
			// A response can be written in several chunks with the same id.
			if d.MessageID != "" {
				if _, dup := assistantIDs[d.MessageID]; dup {
					return true
				}
				assistantIDs[d.MessageID] = struct{}{}
			}
			s.AssistantMessages++
			r.next()
			r.at(ts, lastModel, cwd)
			r.text(roleAssistant, blockText, func() string { return textOf(d.Content) })
		case "tool.execution_start":
			r.next()
			r.at(ts, lastModel, cwd)
			r.toolUse(d.ToolName, d.ToolCallID, func() string { return rawString(d.Arguments) },
				func() (string, string) {
					a := parseArgs(d.Arguments)
					if d.MCPServerName != "" {
						return classifyMCP(d.MCPServerName, firstNonEmpty(d.MCPToolName, d.ToolName), a)
					}
					return classifyNative(agentCopilotCLI, d.ToolName, a)
				})
		case "permission.completed":
			var res struct {
				Kind string `json:"kind"`
			}
			_ = json.Unmarshal(d.Result, &res)
			switch {
			case strings.HasPrefix(res.Kind, "denied-"):
				r.decide(d.ToolCallID, decisionDenied)
			case res.Kind == "cancelled":
				r.decide(d.ToolCallID, decisionError)
			}
		case "tool.execution_complete":
			decision := decisionAllowed
			if (d.Success != nil && !*d.Success) || copilotExitCode(d.ShellExecution, d.Result) != 0 {
				decision = decisionError
			}
			r.next()
			r.at(ts, "", cwd)
			r.toolResult(d.ToolCallID, func(Message) string { return decision }, func() string {
				var res struct {
					Content json.RawMessage `json:"content"`
				}
				_ = json.Unmarshal(d.Result, &res)
				if t := textOf(res.Content); t != "" {
					return t
				}
				if d.Error != nil {
					return d.Error.Message
				}
				return ""
			})
		case "session.shutdown":
			// Each run of a resumed session writes its own; the last is the
			// session's total.
			shutdownModel = firstNonEmpty(d.CurrentModel, shutdownModel)
			copilotTokenTotals(s, d)
		}
		// system.message is the whole system prompt; it isn't conversation.
		return true
	})

	// "auto" is a routing preference, so prefer the model that answered.
	s.Model = firstNonEmpty(shutdownModel, lastModel, selectedModel)
}

func firstContext(c *copilotContext, fallback copilotContext) copilotContext {
	if c != nil {
		return *c
	}
	return fallback
}

// copilotExitCode is a shell call's exit code, which Copilot records on a
// "successful" completion.
func copilotExitCode(sh *copilotShell, result json.RawMessage) int {
	if sh != nil && sh.ExitCode != nil {
		return *sh.ExitCode
	}
	var res struct {
		Contents []struct {
			Type     string `json:"type"`
			ExitCode int    `json:"exitCode"`
		} `json:"contents"`
	}
	_ = json.Unmarshal(result, &res)
	for _, c := range res.Contents {
		if c.Type == "shell_exit" {
			return c.ExitCode
		}
	}
	return 0
}

func copilotTokenTotals(s *Session, d *copilotData) {
	if len(d.TokenDetails) > 0 {
		t := d.TokenDetails
		s.InputTokens = t["input"].TokenCount
		s.OutputTokens = t["output"].TokenCount
		s.CacheReadTokens = t["cache_read"].TokenCount
		s.CacheWriteTokens = t["cache_write"].TokenCount + t["cache_write_1h"].TokenCount
		return
	}
	if len(d.ModelMetrics) == 0 {
		return
	}
	s.InputTokens, s.OutputTokens, s.CacheReadTokens, s.CacheWriteTokens = 0, 0, 0, 0
	for _, m := range d.ModelMetrics {
		u := m.Usage
		// inputTokens includes the cached tokens here.
		s.InputTokens += u.InputTokens - u.CacheReadTokens
		s.OutputTokens += u.OutputTokens
		s.CacheReadTokens += u.CacheReadTokens
		s.CacheWriteTokens += u.CacheWriteTokens
	}
}
