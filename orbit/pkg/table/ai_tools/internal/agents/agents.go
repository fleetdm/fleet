// Package agents detects installed AI agent CLIs without ever executing the
// discovered binary (a security requirement: the extension runs as root and
// must not spawn untrusted code). Presence comes from file existence; version
// comes from adjacent manifests (npm package.json, pipx dist-info, Homebrew
// path), never from `--version`.
package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/evidence"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/paths"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

// Agent is a detected AI agent CLI.
type Agent struct {
	UID, Username string
	Name          string
	Binary        string
	Path          string
	BinaryPath    string // resolved path of the executable file (hashed)
	Version       string
	Runtime       string // node | bun | python | rust | go | native
	InstallMethod string // npm-global | pipx | homebrew | cargo | native | evidence
	Running       int
	PID           int

	// Classification / multi-signal detection.
	Category   string // agent-runtime | agent-harness | empty (legacy catalog)
	Confidence int    // 0–100; catalog = 100
	Evidence   string // CSV of signal tokens

	// Security posture (computed during Scan).
	SHA256         string // hash of the agent binary (diffable identity / threat-intel match)
	PermissionMode string // declared autonomy posture (bypassPermissions, acceptEdits, ...)
	RiskFlags      string // risk tokens, comma-separated (bypass_permissions, skip_permissions_runtime, ...)
}

type known struct {
	name     string
	binaries []string
	npmPkg   string
	pipxName string
	runtime  string
	// autoFlags are lowercased command-line substrings that indicate the agent
	// is running in an unattended auto-approve / sandbox-disabled mode — the
	// single highest-risk agentic posture on a host.
	autoFlags []string
}

func knownAgents() []known {
	return []known{
		{"claude-code", []string{"claude"}, "@anthropic-ai/claude-code", "", "node", []string{"--dangerously-skip-permissions", "skip-permissions"}},
		{"gemini-cli", []string{"gemini"}, "@google/gemini-cli", "", "node", []string{"--yolo", "--approval-mode yolo"}},
		{"codex", []string{"codex"}, "@openai/codex", "", "rust", []string{"--dangerously-bypass-approvals-and-sandbox", "--yolo", "--full-auto", "danger-full-access"}},
		{"aider", []string{"aider"}, "", "aider-chat", "python", []string{"--yes-always", "--yes"}},
		{"goose", []string{"goose"}, "", "", "rust", nil},
		{"opencode", []string{"opencode"}, "opencode-ai", "", "go", nil},
		{"cline", []string{"cline"}, "cline", "", "node", nil},
		{"continue-cli", []string{"cn"}, "@continuedev/cli", "", "node", nil},
		{"cursor-agent", []string{"cursor-agent"}, "", "", "native", nil},
		{"amazon-q", []string{"q", "kiro"}, "", "", "native", nil},
		// Catalog sugar for common CLIs; multi-signal evidence still covers unknowns.
		{"grok", []string{"grok"}, "", "", "native", nil},
	}
}

// Scan detects agent CLIs reachable from a home directory (and system dirs).
// When b is non-nil, Tier-B multi-signal candidates (tool homes, workspace
// shapes, frameworks) are merged with catalog hits.
func Scan(h homes.Home, snap *proc.Snapshot, b *evidence.Bundle) []Agent {
	r := paths.For(h.Dir)
	binDirs := agentBinDirs(h.Dir, r)
	// Include tool-private bin dirs (e.g. ~/.grok/bin) used by modern CLIs.
	binDirs = append(binDirs,
		filepath.Join(h.Dir, ".grok", "bin"),
		filepath.Join(h.Dir, ".hermes", "bin"),
		filepath.Join(h.Dir, ".openclaw", "bin"),
	)
	nmDirs := nodeModulesDirs(h.Dir, r)
	out := make([]Agent, 0, len(knownAgents()))
	seen := map[string]int{} // name -> index in out

	for _, k := range knownAgents() {
		a, ok := detect(k, h.Dir, binDirs, nmDirs)
		if !ok {
			continue
		}
		a.UID, a.Username = h.UID, h.Username
		a.Name = k.name
		if a.Runtime == "" {
			a.Runtime = k.runtime
		}
		a.Category = "agent-runtime"
		a.Confidence = 100
		a.Evidence = "catalog"
		cmdline := markRunning(&a, k, snap)
		if a.Running == 1 {
			a.Evidence = "catalog,running"
		}
		if a.BinaryPath != "" {
			a.Evidence += ",binary"
		}
		a.SHA256 = fsutil.SHA256(resolveSystemBinary(a.BinaryPath))
		enrichPosture(&a, k, h, cmdline)
		seen[a.Name] = len(out)
		out = append(out, a)
	}

	// Tier B: multi-signal candidates from the shared evidence bundle.
	if b != nil {
		for _, c := range evidence.AgentCandidates(h, snap, b) {
			if idx, ok := seen[c.Name]; ok && mergesInto(out[idx], c) {
				// Merge into catalog row: keep conf 100, union evidence.
				out[idx].Evidence = mergeEvidence(out[idx].Evidence, c.Signals.CSV())
				if c.Running == 1 && out[idx].Running == 0 {
					out[idx].Running, out[idx].PID = c.Running, c.PID
				}
				if out[idx].BinaryPath == "" && c.BinaryPath != "" {
					out[idx].BinaryPath = c.BinaryPath
					out[idx].SHA256 = fsutil.SHA256(resolveSystemBinary(c.BinaryPath))
				}
				if c.Category == "agent-harness" {
					out[idx].Category = "agent-harness"
				}
				continue
			}
			// Also skip if path already covered under another name.
			if pathSeen(out, c.Path) {
				continue
			}
			a := Agent{
				UID:           h.UID,
				Username:      h.Username,
				Name:          c.Name,
				Binary:        c.Binary,
				Path:          c.Path,
				BinaryPath:    c.BinaryPath,
				InstallMethod: "evidence",
				Runtime:       "native",
				Running:       c.Running,
				PID:           c.PID,
				Category:      c.Category,
				Confidence:    c.Confidence,
				Evidence:      c.Signals.CSV(),
			}
			if a.Category == "" {
				a.Category = "agent-runtime"
			}
			if a.BinaryPath != "" {
				a.SHA256 = fsutil.SHA256(resolveSystemBinary(a.BinaryPath))
			}
			seen[a.Name] = len(out)
			out = append(out, a)
		}
	}
	return out
}

func mergeEvidence(a, b string) string {
	s := evidence.Signals{}
	for _, csv := range []string{a, b} {
		for tok := range strings.SplitSeq(csv, ",") {
			s.Add(strings.TrimSpace(tok))
		}
	}
	return s.CSV()
}

// mergesInto reports whether an evidence candidate describes the same install
// as an existing row rather than merely sharing its name. Tool-home candidates
// are named after a known tool, so a name match really is the same tool.
// Workspace and framework candidates are named after a directory, so a project
// directory that happens to be called "grok" or "codex" must not fold into that
// tool's row, contribute unrelated evidence, and relabel its category.
func mergesInto(a Agent, c evidence.AgentCandidate) bool {
	if c.BinaryPath != "" && a.BinaryPath != "" &&
		filepath.Clean(c.BinaryPath) == filepath.Clean(a.BinaryPath) {
		return true
	}
	return c.Signals.Has("tool_home")
}

func pathSeen(out []Agent, path string) bool {
	if path == "" {
		return false
	}
	for _, a := range out {
		if a.Path == path || a.BinaryPath == path {
			return true
		}
	}
	return false
}

func detect(k known, home string, binDirs, nmDirs []string) (Agent, bool) {
	a := Agent{}

	// 1. npm global package (best version signal). Prefer installs under the
	// user's home over system-wide node_modules so multi-user hosts don't
	// attribute root's package version to every account.
	if k.npmPkg != "" {
		homeFound := false
		for _, nm := range nmDirs {
			pkgDir := filepath.Join(nm, filepath.FromSlash(k.npmPkg))
			ver, ok := npmVersion(filepath.Join(pkgDir, "package.json"))
			if !ok {
				continue
			}
			underHome := home != "" && strings.HasPrefix(pkgDir, home)
			if underHome {
				a.Path, a.Version, a.InstallMethod, a.Runtime = pkgDir, ver, "npm-global", "node"
				homeFound = true
			} else if !homeFound && a.Path == "" {
				a.Path, a.Version, a.InstallMethod, a.Runtime = pkgDir, ver, "npm-global", "node"
			}
		}
	}

	// 2. pipx venv.
	if a.Path == "" && k.pipxName != "" {
		venv := filepath.Join(home, ".local", "pipx", "venvs", k.pipxName)
		if isDir(venv) {
			a.Path, a.InstallMethod, a.Runtime = venv, "pipx", "python"
			a.Version = pipxVersion(venv, k.pipxName)
		}
	}

	// 3. binary on a known bin dir (presence; install method inferred from path).
	if bin, path, ok := findBinary(k.binaries, binDirs); ok {
		a.Binary = bin
		a.BinaryPath = path
		if a.Path == "" {
			a.Path = path
			a.InstallMethod = methodFromPath(path)
		}
	} else if a.Path == "" {
		return Agent{}, false
	}
	if a.Binary == "" && len(k.binaries) > 0 {
		a.Binary = k.binaries[0]
	}
	return a, true
}

// systemBinPrefixes are trusted system / package-manager directories where
// agent CLIs are commonly symlinked (Homebrew links /opt/homebrew/bin/<tool>
// into its Cellar, for example).
var systemBinPrefixes = []string{"/opt/homebrew/", "/usr/local/", "/usr/bin/", "/home/linuxbrew/"}

// resolveSystemBinary resolves a symlink to its target only when the link lives
// under a trusted system bin directory, so package-manager-symlinked agent
// binaries are still hashed by the (symlink-refusing) fsutil.SHA256. Paths under
// user homes are returned unchanged and never symlink-resolved, so a low-priv
// user cannot use a home-dir symlink to steer the root scanner at a file it
// should not read.
func resolveSystemBinary(p string) string {
	if p == "" {
		return p
	}
	for _, prefix := range systemBinPrefixes {
		if strings.HasPrefix(p, prefix) {
			if resolved, err := filepath.EvalSymlinks(p); err == nil {
				return resolved
			}
			break
		}
	}
	return p
}

func agentBinDirs(home string, _ paths.Roots) []string {
	dirs := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "bin"),
		filepath.Join(home, ".bun", "bin"),
		filepath.Join(home, ".cargo", "bin"),
		filepath.Join(home, ".deno", "bin"),
		filepath.Join(home, ".opencode", "bin"),
		filepath.Join(home, ".npm-global", "bin"),
		filepath.Join(home, "go", "bin"),
	}
	if runtime.GOOS == "windows" {
		dirs = append(dirs, filepath.Join(home, "AppData", "Roaming", "npm"))
	} else {
		dirs = append(dirs, "/usr/local/bin", "/opt/homebrew/bin", "/usr/bin")
	}
	return dirs
}

func nodeModulesDirs(home string, _ paths.Roots) []string {
	dirs := []string{
		filepath.Join(home, ".npm-global", "lib", "node_modules"),
		filepath.Join(home, ".bun", "install", "global", "node_modules"),
	}
	if runtime.GOOS == "windows" {
		dirs = append(dirs, filepath.Join(home, "AppData", "Roaming", "npm", "node_modules"))
	} else {
		dirs = append(dirs, "/usr/local/lib/node_modules", "/opt/homebrew/lib/node_modules")
	}
	// nvm-managed node versions
	if matches, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "lib", "node_modules")); matches != nil {
		dirs = append(dirs, matches...)
	}
	return dirs
}

func findBinary(names, dirs []string) (string, string, bool) {
	exts := []string{""}
	if runtime.GOOS == "windows" {
		exts = []string{".exe", ".cmd", ".bat", ""}
	}
	for _, name := range names {
		for _, dir := range dirs {
			for _, ext := range exts {
				p := filepath.Join(dir, name+ext)
				if fi, err := os.Lstat(p); err == nil && !fi.IsDir() {
					return name, p, true
				}
			}
		}
	}
	return "", "", false
}

func npmVersion(packageJSON string) (string, bool) {
	b, err := fsutil.ReadFileBounded(packageJSON)
	if err != nil {
		return "", false
	}
	var m struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return "", false
	}
	return m.Version, true
}

func pipxVersion(venv, dist string) string {
	matches, _ := filepath.Glob(filepath.Join(venv, "lib", "python*", "site-packages", dist+"*.dist-info", "METADATA"))
	for _, mp := range matches {
		b, err := fsutil.ReadFileBounded(mp)
		if err != nil {
			continue
		}
		for line := range strings.SplitSeq(string(b), "\n") {
			if v, ok := strings.CutPrefix(line, "Version:"); ok {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func methodFromPath(path string) string {
	low := strings.ToLower(path)
	switch {
	case strings.Contains(low, "node_modules"):
		return "npm-global"
	case strings.Contains(low, "pipx"):
		return "pipx"
	case strings.Contains(low, "homebrew") || strings.Contains(low, "/cellar/"):
		return "homebrew"
	case strings.Contains(low, ".cargo"):
		return "cargo"
	default:
		return "native"
	}
}

// markRunning sets Running/PID when a process matches the agent and returns the
// matched process command line (or "") so the caller can inspect runtime flags.
// Matching is exact name / exe basename / path-token only — never process-name
// suffix (catalog binary "q" must not match processes like "icq").
func markRunning(a *Agent, k known, snap *proc.Snapshot) string {
	if snap == nil {
		return ""
	}
	for pid, p := range snap.Procs {
		for _, bin := range k.binaries {
			if procMatchesBin(p.Name, p.Exe, p.Cmdline, bin) {
				a.Running, a.PID = 1, pid
				return p.Cmdline
			}
		}
		if k.npmPkg != "" {
			pkg := strings.ToLower(k.npmPkg)
			cmd := strings.ToLower(p.Cmdline)
			// Package path token (scoped npm pkgs), not bare substring of short names.
			if strings.Contains(cmd, "/"+pkg) || strings.Contains(cmd, "\\"+pkg) ||
				strings.Contains(cmd, pkg+"/") || strings.Contains(cmd, pkg+"@") {
				a.Running, a.PID = 1, pid
				return p.Cmdline
			}
		}
	}
	return ""
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
	for _, sep := range []string{"/", "\\"} {
		tok := sep + bin
		if strings.HasSuffix(cmd, tok) || strings.Contains(cmd, tok+" ") ||
			strings.HasSuffix(cmd, tok+".exe") || strings.Contains(cmd, tok+".exe ") {
			return true
		}
	}
	return false
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	return fi.IsDir()
}
