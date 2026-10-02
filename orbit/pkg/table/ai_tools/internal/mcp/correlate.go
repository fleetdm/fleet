package mcp

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

// mcpProcessMarkers identify a process as an MCP server regardless of any
// config entry. Kept narrow (MCP-specific) to avoid mislabeling generic
// runtimes — broader AI classification lives in the classify package.
var mcpProcessMarkers = []string{
	"modelcontextprotocol",
	"@modelcontextprotocol/",
	"mcp-server",
	"mcp_server",
}

// Correlate reconciles declared servers against a process snapshot: it fills
// Running/PID/ListeningPort on stdio servers it can match to a live process,
// and appends heuristic rows (source="process") for running MCP servers that
// no config declared.
func Correlate(declared []Server, snap *proc.Snapshot) []Server {
	if snap == nil {
		return declared
	}
	matched := map[int]struct{}{}

	for i := range declared {
		s := &declared[i]
		if s.Command == "" { // remote servers have no local process to match
			continue
		}
		base := baseCmd(s.Command)
		if base == "" {
			continue
		}
		var args []string
		if s.Args != "" {
			_ = json.Unmarshal([]byte(s.Args), &args)
		}
		for pid, p := range snap.Procs {
			if processMatches(p.Cmdline, base, args) {
				s.Running, s.PID = 1, pid
				if s.Source == "config" {
					s.Source = "both"
				}
				if port := snap.ListenPort(pid); port != 0 {
					s.ListeningPort = port
				}
				matched[pid] = struct{}{}
				break
			}
		}
	}

	for pid, p := range snap.Procs {
		if _, ok := matched[pid]; ok || !isMCPProcess(p.Cmdline) {
			continue
		}
		argv := processArgv(p)
		s := Server{
			ServerName: deriveName(argv),
			Client:     "process",
			Scope:      "global",
			Transport:  "stdio",
			Location:   "local",
			Command:    argv0(argv),
			Source:     "process",
			Running:    1,
			PID:        pid,
			Username:   p.Username,
			Enabled:    -1,
		}
		if port := snap.ListenPort(pid); port != 0 {
			s.ListeningPort = port
		}
		s.Args = argsJSON(argv)
		s.enrichRisk()
		declared = append(declared, s)
	}
	return declared
}

// processArgv returns the process's true argv boundaries. gopsutil's Cmdline
// is those same elements joined with a plain space, so a single quoted
// argument that itself contains spaces (e.g. the inline script body of
// `node -e "<script>"`) becomes indistinguishable from several separate
// arguments once joined. Falling back to strings.Fields(p.Cmdline) is only
// safe when the platform couldn't supply CmdlineSlice at all.
func processArgv(p proc.Process) []string {
	if len(p.CmdlineSlice) > 0 {
		return p.CmdlineSlice
	}
	return strings.Fields(p.Cmdline)
}

// argsJSON encodes the launch arguments (everything after the executable) as
// the JSON array the risk logic and table row expect.
func argsJSON(argv []string) string {
	if len(argv) <= 1 {
		return ""
	}
	b, err := json.Marshal(argv[1:])
	if err != nil {
		return ""
	}
	return string(b)
}

func processMatches(cmdline, base string, args []string) bool {
	low := strings.ToLower(cmdline)
	if !strings.Contains(low, strings.ToLower(base)) {
		return false
	}
	if len(args) == 0 {
		return true
	}
	// Require a distinctive arg token to also appear, so a bare "node" command
	// doesn't match every Node process.
	for _, a := range args {
		tok := strings.ToLower(lastSegment(a))
		if tok != "" && strings.Contains(low, tok) {
			return true
		}
	}
	return false
}

func isMCPProcess(cmdline string) bool {
	low := strings.ToLower(cmdline)
	for _, m := range mcpProcessMarkers {
		if strings.Contains(low, m) {
			return true
		}
	}
	return false
}

func baseCmd(cmd string) string {
	cmd = strings.Trim(cmd, `"'`)
	b := filepath.Base(cmd)
	return strings.TrimSuffix(strings.TrimSuffix(b, ".exe"), ".cmd")
}

func argv0(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	return argv[0]
}

func lastSegment(s string) string {
	s = strings.TrimRight(s, "/\\")
	if i := strings.LastIndexAny(s, "/\\"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// scriptPathRe pulls a path-like, extension-terminated token (e.g.
// "mcp-server.cjs" or "/home/x/.claude-mem/mcp-server.cjs") out of a larger
// blob of text, so a launcher name can still be recovered when the matching
// argv element is an inline script body rather than a clean file path.
var scriptPathRe = regexp.MustCompile(`[^\s'"` + "`" + `()]+\.(?:c?js|mjs|c?ts)\b`)

// looksLikeCode reports whether argv element f is itself inline source (e.g.
// the body of `node -e "<script>"`) rather than a plain path or flag. Real
// argv elements never contain these characters; only pasted-in code does.
func looksLikeCode(f string) bool {
	return strings.ContainsAny(f, "(){};=`\n")
}

// deriveName picks the most server-identifying token from an MCP process's
// argv (e.g. the package after @modelcontextprotocol/, or the launcher
// filename). argv must be the process's true argument boundaries — not a
// command line re-split on whitespace, which shreds any single argument
// that itself contains spaces (inline eval scripts, quoted paths) into
// arbitrary fragments.
func deriveName(argv []string) string {
	for _, f := range argv {
		l := strings.ToLower(f)
		if !strings.Contains(l, "modelcontextprotocol") && !strings.Contains(l, "mcp-server") && !strings.Contains(l, "mcp_server") {
			continue
		}
		if !looksLikeCode(f) {
			return lastSegment(f)
		}
		if m := scriptPathRe.FindString(f); m != "" {
			return lastSegment(m)
		}
		// Marker found only inside inline code with no extractable file
		// path — fall through rather than surface the raw code fragment.
	}
	if f := argv0(argv); f != "" {
		return baseCmd(f)
	}
	return "unknown"
}
