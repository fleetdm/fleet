package evidence

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

// Bundle is a per-query, read-only evidence set shared across collectors.
type Bundle struct {
	ToolHomes  []ToolHome
	Workspaces []Workspace
	Frameworks []Framework
}

// ToolHome is a detected agent/tool configuration home under a user directory.
type ToolHome struct {
	UID, Username string
	Name          string // e.g. grok, claude, openclaw
	Path          string
	BinaryPath    string // optional resolved binary
	BinaryName    string
	HasConfig     bool
}

// Workspace is a project/user root with agent-shaped markers.
type Workspace struct {
	UID, Username string
	Root          string
	Name          string // basename of root
	Markers       []string
	Strong        bool
	Category      string // agent-harness | agent-runtime
}

// Framework is a detected agent/harness library fingerprint.
type Framework struct {
	UID, Username string
	Name          string // crewai, hermes, openclaw, ...
	Path          string
	Source        string // package.json | pyproject | global-node | pipx
	Harness       bool
}

// Gather builds a Bundle for the given homes and optional process snapshot.
// types controls which gatherers run (keys match ai_tools type values).
// Failures degrade to empty slices — never panics.
func Gather(ctx context.Context, hs []homes.Home, snap *proc.Snapshot, types map[string]struct{}) *Bundle {
	b := &Bundle{}
	has := func(t string) bool { _, ok := types[t]; return ok }
	needAgents := types == nil || has("agents") || has("apps") || has("sockets") || has("mcp_server")
	if !needAgents {
		// Still useful for agent_instruction correlation when only instructions requested? skip.
		if types != nil && !has("agents") {
			return b
		}
	}
	for _, h := range hs {
		if ctx.Err() != nil {
			break
		}
		b.ToolHomes = append(b.ToolHomes, scanToolHomes(h)...)
		b.Workspaces = append(b.Workspaces, scanWorkspaces(h)...)
		b.Frameworks = append(b.Frameworks, scanFrameworks(h)...)
	}
	// Running state is applied later, where agents fuse these candidates
	// against the process snapshot; here the paths are only normalized.
	normalizeBinaryPaths(b)
	return b
}

// annotateRunning sets binary running hints by matching process names/paths.
func normalizeBinaryPaths(b *Bundle) {
	for i := range b.ToolHomes {
		if b.ToolHomes[i].BinaryPath != "" {
			b.ToolHomes[i].BinaryPath = filepath.Clean(b.ToolHomes[i].BinaryPath)
		}
	}
}

// AgentCandidate is a fused Tier-B agent suggestion for one home.
type AgentCandidate struct {
	UID, Username string
	Name          string
	Binary        string
	Path          string
	BinaryPath    string
	Category      string // agent-runtime | agent-harness
	Signals       Signals
	Confidence    int
	Running       int
	PID           int
}

// AgentCandidates fuses bundle facts into emit-worthy agent candidates for home h.
func AgentCandidates(h homes.Home, snap *proc.Snapshot, b *Bundle) []AgentCandidate {
	if b == nil {
		return nil
	}
	// Index by canonical path or name to merge signals.
	type acc struct {
		c AgentCandidate
	}
	byKey := map[string]*acc{}

	ensure := func(key, name, path string) *acc {
		if a, ok := byKey[key]; ok {
			return a
		}
		a := &acc{c: AgentCandidate{
			UID:      h.UID,
			Username: h.Username,
			Name:     name,
			Path:     path,
			Category: "agent-runtime",
			Signals:  Signals{},
		}}
		byKey[key] = a
		return a
	}

	for _, th := range b.ToolHomes {
		if !underHome(th.Path, h.Dir) {
			continue
		}
		key := "home:" + th.Path
		a := ensure(key, th.Name, th.Path)
		a.c.Signals.Add("tool_home")
		if th.HasConfig {
			a.c.Signals.Add("tool_home")
		}
		if th.BinaryPath != "" {
			a.c.Signals.Add("binary")
			a.c.BinaryPath = th.BinaryPath
			a.c.Binary = th.BinaryName
			if a.c.Binary == "" {
				a.c.Binary = th.Name
			}
		}
		if a.c.Name == "" {
			a.c.Name = th.Name
		}
	}

	for _, ws := range b.Workspaces {
		if !underHome(ws.Root, h.Dir) {
			continue
		}
		key := "ws:" + ws.Root
		a := ensure(key, ws.Name, ws.Root)
		if ws.Strong {
			a.c.Signals.Add("workspace_shape")
		} else {
			a.c.Signals.Add("workspace_shape_weak")
		}
		for _, m := range ws.Markers {
			switch m {
			case "mcp_config":
				a.c.Signals.Add("mcp_config")
			case "instructions":
				a.c.Signals.Add("instructions")
			case "skills", "loop_config", "state":
				// contribute to shape only
			}
		}
		if ws.Category == "agent-harness" {
			a.c.Category = "agent-harness"
		}
	}

	for _, fw := range b.Frameworks {
		if fw.Path != "" && !underHome(fw.Path, h.Dir) {
			continue
		}
		// Attach framework to nearest workspace root or use package path.
		key := "fw:" + fw.Path
		name := fw.Name
		if dir := filepath.Dir(fw.Path); dir != "" {
			// Prefer project root two levels up from package.json
			if strings.HasSuffix(fw.Path, "package.json") || strings.HasSuffix(fw.Path, "pyproject.toml") {
				key = "ws:" + filepath.Dir(fw.Path)
				name = filepath.Base(filepath.Dir(fw.Path))
			}
		}
		a := ensure(key, name, filepath.Dir(fw.Path))
		a.c.Signals.Add("framework:" + fw.Name)
		if fw.Harness {
			a.c.Category = "agent-harness"
		}
	}

	// Running processes: exact name / exe base / path-token only (never
	// suffix-match process names — "q" must not match "icq").
	if snap != nil {
		for pid, p := range snap.Procs {
			for _, a := range byKey {
				bin := strings.ToLower(a.c.Binary)
				if bin == "" {
					bin = strings.ToLower(a.c.Name)
				}
				// Skip path-like or dotdir "names" (e.g. ".claude", "agentic-detector").
				if bin == "" || strings.Contains(bin, "/") || strings.Contains(bin, "\\") || strings.HasPrefix(bin, ".") {
					continue
				}
				if !procMatchesBin(p.Name, p.Exe, p.Cmdline, bin) {
					continue
				}
				// Prefer binary path alignment when we have one.
				if a.c.BinaryPath != "" {
					baseWant := strings.ToLower(filepath.Base(a.c.BinaryPath))
					lowName := strings.ToLower(p.Name)
					lowExeBase := strings.ToLower(filepath.Base(p.Exe))
					if lowExeBase != baseWant && lowExeBase != baseWant+".exe" &&
						lowName != bin && lowName != bin+".exe" {
						continue
					}
				}
				a.c.Signals.Add("running")
				a.c.Running = 1
				a.c.PID = pid
			}
		}
	}

	var out []AgentCandidate
	for _, a := range byKey {
		// Drop pure weak noise: only weak shape or only instructions already handled by ShouldEmit.
		if !ShouldEmit(a.c.Signals) {
			continue
		}
		a.c.Confidence = Score(a.c.Signals)
		if a.c.Binary == "" {
			a.c.Binary = a.c.Name
		}
		out = append(out, a.c)
	}
	return out
}

// underHome reports whether path is home itself or a descendant of home.
// Bare strings.HasPrefix is wrong: /Users/bob is a prefix of /Users/bobby.
func underHome(path, home string) bool {
	if home == "" {
		return true
	}
	if path == "" {
		return false
	}
	// Compare resolved paths when both resolve. A symlink such as
	// $HOME/.claude -> /etc satisfies a purely lexical prefix check while its
	// target sits outside the home, and the callers reach this guard through
	// os.Stat, which follows symlinks. Paths that cannot be resolved (a broken
	// link, or one that no longer exists) fall back to the lexical comparison;
	// nothing readable lies behind them either way.
	if rp, err := filepath.EvalSymlinks(path); err == nil {
		if rh, err := filepath.EvalSymlinks(home); err == nil {
			path, home = rp, rh
		}
	}
	path = filepath.Clean(path)
	home = filepath.Clean(home)
	if path == home {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(path, home+sep)
}

// procMatchesBin reports whether a live process is the given binary.
// Matches exact process name, exe basename, or a path-token in the command
// line — never name suffix (short bins like "q" would false-positive).
func procMatchesBin(name, exe, cmdline, bin string) bool {
	bin = strings.ToLower(bin)
	if bin == "" {
		return false
	}
	name = strings.ToLower(name)
	if name == bin || name == bin+".exe" {
		return true
	}
	base := strings.ToLower(filepath.Base(exe))
	if base == bin || base == bin+".exe" {
		return true
	}
	cmd := strings.ToLower(cmdline)
	// Path token: .../bin or .../bin <args> (also Windows backslash).
	for _, sep := range []string{"/", "\\"} {
		tok := sep + bin
		if strings.HasSuffix(cmd, tok) || strings.Contains(cmd, tok+" ") ||
			strings.HasSuffix(cmd, tok+".exe") || strings.Contains(cmd, tok+".exe ") {
			return true
		}
	}
	return false
}
