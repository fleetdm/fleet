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
	// Annotate tool homes / workspaces with running processes when snap present.
	if snap != nil {
		annotateRunning(b, snap)
	}
	return b
}

// annotateRunning sets binary running hints by matching process names/paths.
func annotateRunning(b *Bundle, snap *proc.Snapshot) {
	// Running is applied when agents fuse candidates; here we only ensure
	// binary paths from tool homes are absolute and cleaned.
	for i := range b.ToolHomes {
		if b.ToolHomes[i].BinaryPath != "" {
			b.ToolHomes[i].BinaryPath = filepath.Clean(b.ToolHomes[i].BinaryPath)
		}
	}
	_ = snap
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
		if h.Dir != "" && !strings.HasPrefix(th.Path, h.Dir+string(filepath.Separator)) && th.Path != h.Dir {
			// Allow prefix match with separator to avoid /Users/bob matching /Users/bobby.
			if !strings.HasPrefix(th.Path, h.Dir) {
				continue
			}
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
		if h.Dir != "" && !strings.HasPrefix(ws.Root, h.Dir) {
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
		if h.Dir != "" && fw.Path != "" && !strings.HasPrefix(fw.Path, h.Dir) {
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

	// Running processes: match binary name / exe only (not workspace path
	// substrings — those false-positive on builds, IDEs, and osquery itself).
	if snap != nil {
		for pid, p := range snap.Procs {
			lowName := strings.ToLower(p.Name)
			lowCmd := strings.ToLower(p.Cmdline)
			lowExe := strings.ToLower(p.Exe)
			for _, a := range byKey {
				bin := strings.ToLower(a.c.Binary)
				if bin == "" {
					bin = strings.ToLower(a.c.Name)
				}
				// Skip path-like or dotdir "names" (e.g. ".claude", "agentic-detector").
				if bin == "" || strings.Contains(bin, "/") || strings.HasPrefix(bin, ".") {
					continue
				}
				if lowName == bin || lowName == bin+".exe" ||
					strings.HasSuffix(lowExe, "/"+bin) ||
					strings.Contains(lowCmd, "/"+bin+" ") ||
					strings.HasSuffix(lowCmd, "/"+bin) {
					// Prefer binary path alignment when we have one.
					if a.c.BinaryPath != "" {
						if !strings.Contains(lowExe, strings.ToLower(filepath.Base(a.c.BinaryPath))) &&
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
