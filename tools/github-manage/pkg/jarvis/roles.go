package jarvis

import (
	"strconv"
	"strings"
	"text/template"
)

// Roles jarvis knows about. A user's primary role tailors the seed prompt sent to
// a freshly launched Claude session (and, for QA, which issues surface and what
// action they get). It's chosen during onboarding and stored in config.json.
const (
	RoleDeveloper = "developer"
	RoleManager   = "manager"
	RoleQA        = "qa"
	RoleDesign    = "design"
)

type roleMeta struct {
	Role  string
	Label string
	Desc  string
}

// RoleChoices are the roles offered at onboarding, in display order. Developer is
// first so it's the default landing selection.
var RoleChoices = []roleMeta{
	{RoleDeveloper, "Developer", "Branch off main and drive issues to a merged PR."},
	{RoleManager, "Manager", "Same per-issue flow as a developer, with your own prompt overrides."},
	{RoleQA, "QA", "Reproduce and verify fixes; see issues awaiting QA."},
	{RoleDesign, "Design", "Drive design work on assigned issues."},
}

// normalizeRole lowercases/trims a role string and maps it to a known role,
// falling back to RoleDeveloper for anything unrecognized (including "").
func normalizeRole(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case RoleManager:
		return RoleManager
	case RoleQA:
		return RoleQA
	case RoleDesign:
		return RoleDesign
	default:
		return RoleDeveloper
	}
}

// PromptData is the template context available to start-work prompts, both the
// built-in defaults and user overrides in config.json's start_prompts.
type PromptData struct {
	Issue  int
	Title  string
	URL    string // "" if unknown
	Branch string // the linked branch, if any (used by the QA prompt)
}

// defaultStartPrompts holds the built-in seed prompt for each role, as Go
// text/template strings rendered against PromptData. developer and manager share
// the plain "let's work on it" prompt; QA asks Claude to set up a repro/verify
// environment. A role missing here falls back to the developer prompt.
var defaultStartPrompts = map[string]string{
	RoleDeveloper: "Let's work on issue #{{.Issue}}: {{.Title}}{{if .URL}}\n{{.URL}}{{end}}",
	RoleManager:   "Let's work on issue #{{.Issue}}: {{.Title}}{{if .URL}}\n{{.URL}}{{end}}",
	RoleDesign:    "Let's work on the design for issue #{{.Issue}}: {{.Title}}{{if .URL}}\n{{.URL}}{{end}}",
	RoleQA: "Issue #{{.Issue}} is awaiting QA: {{.Title}}{{if .URL}}\n{{.URL}}{{end}}\n\n" +
		"Help me set up a local environment to reproduce the original problem and verify this fix. " +
		"If the fix has already merged to main, use the latest main{{if .Branch}}; otherwise check out the PR branch {{.Branch}}{{end}}. " +
		"Walk me through the steps to reproduce the issue and confirm the fix resolves it.",
}

// renderStartPrompt builds the seed prompt for a role. A per-role override in
// cfg.StartPrompts wins; otherwise the built-in default for the role is used, with
// a final fallback to the developer default. On a template parse/execute error it
// returns a plain, always-valid prompt so a bad override never blocks starting.
func renderStartPrompt(role string, cfg *Config, data PromptData) string {
	role = normalizeRole(role)

	tmpl := ""
	if cfg != nil {
		if s, ok := cfg.StartPrompts[role]; ok && strings.TrimSpace(s) != "" {
			tmpl = s
		}
	}
	if tmpl == "" {
		tmpl = defaultStartPrompts[role]
	}
	if tmpl == "" {
		tmpl = defaultStartPrompts[RoleDeveloper]
	}

	t, err := template.New("startPrompt").Parse(tmpl)
	if err != nil {
		return fallbackPrompt(data)
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return fallbackPrompt(data)
	}
	return b.String()
}

func fallbackPrompt(data PromptData) string {
	if data.Title == "" {
		return "Let's work on issue #" + strconv.Itoa(data.Issue) + "."
	}
	return "Let's work on issue #" + strconv.Itoa(data.Issue) + ": " + data.Title
}
