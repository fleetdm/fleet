package ai_agent_sessions

import (
	"encoding/json"
	"regexp"
	"strings"
)

// maxTargetBytes caps the `target` column.
const maxTargetBytes = 256

// Values of the `action` column: what kind of thing a tool call does.
const (
	actionRead    = "read"
	actionWrite   = "write"
	actionExecute = "execute"
	actionSearch  = "search"
	actionNetwork = "network"
	actionMCP     = "mcp"
	actionOther   = "other"
)

// args is a tool call's input: its JSON object when it is one, and the raw
// text either way (some tools, like Codex's apply_patch, take plain text).
type args struct {
	m   map[string]any
	raw string
}

func parseArgs(input []byte) args {
	a := args{raw: string(input)}
	_ = json.Unmarshal(input, &a.m)
	return a
}

// str returns the first of keys holding a string, or an array of strings
// (an argv), joined with spaces.
func (a args) str(keys ...string) string {
	for _, k := range keys {
		switch v := a.m[k].(type) {
		case string:
			if v != "" {
				return v
			}
		case []any:
			if s := joinStrings(v, " "); s != "" {
				return s
			}
		}
	}
	return ""
}

func joinStrings(vs []any, sep string) string {
	var parts []string
	for _, v := range vs {
		if s, ok := v.(string); ok && s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, sep)
}

// rule says what one native tool does and how to describe its target.
type rule struct {
	action string
	// actionFor, when set, decides the action from the input instead.
	actionFor func(a args) string
	target    func(a args) string
}

func field(keys ...string) func(args) string {
	return func(a args) string { return a.str(keys...) }
}

// grep renders a content search as `"<pattern>" in <path>`.
func grep(patternKey string, pathKeys ...string) func(args) string {
	return func(a args) string {
		p := a.str(patternKey)
		if p == "" {
			return ""
		}
		t := `"` + p + `"`
		if dir := a.str(pathKeys...); dir != "" {
			t += " in " + dir
		}
		return t
	}
}

// glob renders a file-name search as `<pattern> in <dir>`.
func glob(patternKey, dirKey string) func(args) string {
	return func(a args) string {
		p := a.str(patternKey)
		if dir := a.str(dirKey); p != "" && dir != "" {
			return p + " in " + dir
		}
		return p
	}
}

var patchFileLine = regexp.MustCompile(`(?m)^\*\*\* (?:Add|Update|Delete) File: (.+?)\s*$`)

// patchFiles lists the files an apply_patch input touches.
func patchFiles(a args) string {
	text := firstNonEmpty(a.str("input", "patch"), a.raw)
	var files []string
	for _, m := range patchFileLine.FindAllStringSubmatch(text, -1) {
		files = append(files, m[1])
	}
	return strings.Join(files, ", ")
}

var urlInText = regexp.MustCompile(`https?://\S+`)

// toolRules are the native tools whose meaning each agent's own source,
// package or a capture confirmed. Anything else gets the generic treatment.
var toolRules = map[string]map[string]rule{
	agentClaudeCode: {
		"Read":         {action: actionRead, target: field("file_path")},
		"NotebookRead": {action: actionRead, target: field("file_path", "notebook_path")},
		"LS":           {action: actionRead, target: field("path")},
		"Write":        {action: actionWrite, target: field("file_path")},
		"Edit":         {action: actionWrite, target: field("file_path")},
		"MultiEdit":    {action: actionWrite, target: field("file_path")},
		"NotebookEdit": {action: actionWrite, target: field("notebook_path")},
		"Glob":         {action: actionSearch, target: field("pattern")},
		"Grep":         {action: actionSearch, target: grep("pattern", "path")},
		"Bash":         {action: actionExecute, target: field("command")},
		"PowerShell":   {action: actionExecute, target: field("command")},
		"Monitor":      {action: actionExecute, target: field("command")},
		"WebFetch":     {action: actionNetwork, target: field("url")},
		"WebSearch":    {action: actionNetwork, target: field("query")},
		"Agent":        {action: actionOther, target: field("description")},
		"Task":         {action: actionOther, target: field("description")},
	},
	agentCursor: {
		"Read":     {action: actionRead, target: field("path")},
		"Write":    {action: actionWrite, target: field("path")},
		"Glob":     {action: actionSearch, target: glob("glob_pattern", "target_directory")},
		"Grep":     {action: actionSearch, target: grep("pattern", "path")},
		"Shell":    {action: actionExecute, target: field("command")},
		"WebFetch": {action: actionNetwork, target: field("url")},
	},
	agentCodexCLI: {
		"exec_command":  {action: actionExecute, target: field("cmd")},
		"shell":         {action: actionExecute, target: field("command")},
		"shell_command": {action: actionExecute, target: field("command")},
		"local_shell":   {action: actionExecute, target: field("command")},
		"apply_patch":   {action: actionWrite, target: patchFiles},
		"web_search":    {action: actionNetwork, target: field("query", "url")},
		"view_image":    {action: actionRead, target: field("path")},
		"update_plan":   {action: actionOther, target: field("explanation")},
	},
	agentCopilotCLI: {
		"bash":        {action: actionExecute, target: field("command")},
		"powershell":  {action: actionExecute, target: field("command")},
		"local_shell": {action: actionExecute, target: field("command")},
		"view":        {action: actionRead, target: field("path")},
		"create":      {action: actionWrite, target: field("path")},
		"edit":        {action: actionWrite, target: field("path")},
		"str_replace": {action: actionWrite, target: field("path")},
		"apply_patch": {action: actionWrite, target: func(a args) string { return firstNonEmpty(a.str("path"), patchFiles(a)) }},
		"str_replace_editor": {
			actionFor: func(a args) string {
				if a.str("command") == "view" {
					return actionRead
				}
				return actionWrite
			},
			target: field("path"),
		},
		"grep":       {action: actionSearch, target: field("pattern")},
		"rg":         {action: actionSearch, target: field("pattern")},
		"glob":       {action: actionSearch, target: field("pattern")},
		"web_fetch":  {action: actionNetwork, target: field("url")},
		"fetch":      {action: actionNetwork, target: field("url")},
		"web_search": {action: actionNetwork, target: field("query")},
		"task":       {action: actionOther, target: field("description")},
	},
	agentGeminiCLI: {
		"read_file":           {action: actionRead, target: field("file_path", "absolute_path")},
		"read_many_files":     {action: actionRead, target: func(a args) string { return manyFiles(a, "include", "paths") }},
		"list_directory":      {action: actionRead, target: field("dir_path", "path")},
		"write_file":          {action: actionWrite, target: field("file_path")},
		"replace":             {action: actionWrite, target: field("file_path")},
		"glob":                {action: actionSearch, target: field("pattern")},
		"grep_search":         {action: actionSearch, target: grep("pattern", "dir_path", "path")},
		"search_file_content": {action: actionSearch, target: grep("pattern", "dir_path", "path")},
		"run_shell_command":   {action: actionExecute, target: field("command")},
		"web_fetch": {action: actionNetwork, target: func(a args) string {
			p := a.str("prompt")
			return firstNonEmpty(urlInText.FindString(p), p)
		}},
		"google_web_search": {action: actionNetwork, target: field("query")},
	},
}

func manyFiles(a args, keys ...string) string {
	for _, k := range keys {
		if vs, ok := a.m[k].([]any); ok {
			if s := joinStrings(vs, ", "); s != "" {
				return s
			}
		}
	}
	return ""
}

// genericKeys are tried in order for tools without a rule.
var genericKeys = []string{"command", "file_path", "path", "pattern", "glob_pattern", "url", "query"}

// classify returns the action and target of a call to tool by agent. Names
// of the form mcp__<server>__<tool> (Claude Code, older Codex) are MCP calls.
func classify(agentName, tool string, input []byte) (action, target string) {
	a := parseArgs(input)
	if server, name, ok := splitMCPName(tool); ok {
		return classifyMCP(server, name, a)
	}
	return classifyNative(agentName, tool, a)
}

func classifyNative(agentName, tool string, a args) (string, string) {
	r, ok := toolRules[agentName][tool]
	if !ok {
		return actionOther, finishTarget(actionOther, firstNonEmpty(a.str(genericKeys...), tool))
	}
	action := r.action
	if r.actionFor != nil {
		action = r.actionFor(a)
	}
	return action, finishTarget(action, r.target(a))
}

// classifyMCP describes a call to an MCP server's tool as
// `<server>: <tool> <main argument>`.
func classifyMCP(server, tool string, a args) (string, string) {
	t := server + ": " + tool
	if g := a.str(genericKeys...); g != "" {
		t += " " + g
	}
	return actionMCP, finishTarget(actionMCP, t)
}

// splitMCPName splits mcp__<server>__<tool>. Servers can't contain "__"
// (other characters are already replaced by "_"), but tools can.
func splitMCPName(tool string) (server, name string, ok bool) {
	rest, found := strings.CutPrefix(tool, "mcp__")
	if !found {
		return "", "", false
	}
	server, name, found = strings.Cut(rest, "__")
	if !found || server == "" || name == "" {
		return "", "", false
	}
	return server, name, true
}

var oneLine = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", " ")

// finishTarget keeps a command's first line, flattens anything else to one
// line, and caps the length.
func finishTarget(action, t string) string {
	if action == actionExecute {
		t, _, _ = strings.Cut(t, "\n")
	} else {
		t = oneLine.Replace(t)
	}
	return truncateUTF8(strings.TrimSpace(t), maxTargetBytes)
}
