// Package classify is the AI/agent knowledge base. It maps known identifiers
// (extension ids, process command lines, listening ports, MCP capability
// markers) to an AI category. The KB is embedded as data (kb.json) so it can
// grow without code
// changes. This classification layer is what turns raw inventory into
// agentic-risk signal.
package classify

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

//go:embed kb.json
var kbBytes []byte

type knowledge struct {
	PluginIDs       map[string]string `json:"plugin_ids"`
	JetBrainsIDs    map[string]string `json:"jetbrains_ids"`
	NameRegex       []string          `json:"name_regex"`
	CmdlineMarkers  map[string]string `json:"cmdline_markers"`
	LocalPorts      map[string]string `json:"local_ports"`
	MCPCapabilities map[string]string `json:"mcp_capabilities"`
	BrowserExtIDs   map[string]string `json:"browser_extension_ids"`
}

var (
	data    knowledge
	nameRes []*regexp.Regexp
)

func init() {
	if err := json.Unmarshal(kbBytes, &data); err != nil {
		// Embedded KB is authored in-repo; a parse failure is a build-time bug.
		panic("classify: invalid kb.json: " + err.Error())
	}
	for _, p := range data.NameRegex {
		if re, err := regexp.Compile(p); err == nil {
			nameRes = append(nameRes, re)
		}
	}
}

// VSCodePlugin classifies a VS Code-family extension by id, falling back to a
// name heuristic. id should be "publisher.name".
func VSCodePlugin(id, displayName string) (bool, string) {
	if cat, ok := data.PluginIDs[strings.ToLower(id)]; ok {
		return true, cat
	}
	return matchName(id + " " + displayName)
}

// JetBrainsPlugin classifies a JetBrains plugin by its (case-sensitive) id.
func JetBrainsPlugin(id, name string) (bool, string) {
	if cat, ok := data.JetBrainsIDs[id]; ok {
		return true, cat
	}
	return matchName(id + " " + name)
}

// BrowserExtension classifies a browser extension by its (lowercased) store id,
// falling back to a name heuristic. Mirrors VSCodePlugin.
func BrowserExtension(id, displayName string) (bool, string) {
	if cat, ok := data.BrowserExtIDs[strings.ToLower(id)]; ok {
		return true, cat
	}
	return matchName(strings.ToLower(id) + " " + displayName)
}

// ByName classifies any free-form name/id via the regex heuristics.
func ByName(s string) (bool, string) { return matchName(s) }

func matchName(s string) (bool, string) {
	for _, re := range nameRes {
		if re.MatchString(s) {
			return true, "ai-tool"
		}
	}
	return false, ""
}

// Cmdline classifies a process command line. The returned category is one of
// mcp-server, agent-runtime, inference-api-local, or ai-tool. Returns
// (false, "") when the command line shows no AI/agent marker.
func Cmdline(cmdline string) (bool, string) {
	low := strings.ToLower(cmdline)
	// Deterministic order isn't required for correctness, but markers are
	// mutually distinctive enough that first-hit is fine.
	for marker, cat := range data.CmdlineMarkers {
		if strings.Contains(low, marker) {
			return true, cat
		}
	}
	return matchName(cmdline)
}

// LocalPortService maps a well-known local inference port to its service name.
func LocalPortService(port int) (string, bool) {
	svc, ok := data.LocalPorts[strconv.Itoa(port)]
	return svc, ok
}

// MCPCapabilities infers the capability tags of an MCP server from its launch
// hay (command + args + server name, lowercased). The extension never connects
// to the server to enumerate live tools — doing so would mean executing
// untrusted code — so capability is inferred statically from the known-server
// KB. Returns a sorted, de-duplicated tag list (e.g. ["fs-write","shell-exec"]).
func MCPCapabilities(hay string) []string {
	low := strings.ToLower(hay)
	set := map[string]struct{}{}
	for marker, tags := range data.MCPCapabilities {
		if strings.Contains(low, marker) {
			for t := range strings.SplitSeq(tags, ",") {
				if t = strings.TrimSpace(t); t != "" {
					set[t] = struct{}{}
				}
			}
		}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
