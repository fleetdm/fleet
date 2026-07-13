// Package instructions discovers agent instruction files — the natural-language
// directives coding agents auto-load and obey (CLAUDE.md, AGENTS.md, GEMINI.md,
// .cursorrules, .github/copilot-instructions.md, Cursor .mdc rules, ...).
//
// These files are an under-monitored attack surface: a malicious instruction
// file committed to a repo is prompt injection / agent-hijack with no code
// execution required. This collector inventories them, hashes each for change
// detection, and flags content that carries injection markers or hidden Unicode
// (zero-width / tag characters used to smuggle instructions past human review).
//
// Files are read but never interpreted or executed, preserving the extension's
// no-exec posture.
package instructions

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

// maxReadBytes bounds how much of a file we scan for injection markers. A
// legitimate instruction file is small; a multi-MB "instruction file" is itself
// suspicious, so scanning only the head is sufficient and bounds cost.
const maxReadBytes = 256 << 10 // 256 KiB

// Instruction is one discovered agent instruction file.
type Instruction struct {
	UID, Username string
	Path          string
	Name          string // base file name
	Tool          string // claude | codex | gemini | cursor | copilot | cline | windsurf | generic
	Scope         string // user | project
	SHA256        string
	Size          int64
	RiskFlags     string // injection_markers, hidden_unicode, world_readable
	Markers       string // matched injection-marker keywords, comma-separated
}

// userProbe is a fixed instruction-file location relative to a home directory.
type userProbe struct {
	rel  string
	tool string
}

func userProbes() []userProbe {
	return []userProbe{
		{filepath.Join(".claude", "CLAUDE.md"), "claude"},
		{filepath.Join(".codex", "AGENTS.md"), "codex"},
		{filepath.Join(".codex", "instructions.md"), "codex"},
		{filepath.Join(".gemini", "GEMINI.md"), "gemini"},
		{filepath.Join(".config", "opencode", "AGENTS.md"), "opencode"},
		{".claude.md", "claude"},
	}
}

// projectFiles are the per-directory instruction files probed during the
// bounded walk of dev-project roots.
var projectFiles = []struct {
	rel  string
	tool string
}{
	{"CLAUDE.md", "claude"},
	{"CLAUDE.local.md", "claude"},
	{"AGENTS.md", "codex"},
	{"GEMINI.md", "gemini"},
	{".cursorrules", "cursor"},
	{".windsurfrules", "windsurf"},
	{".clinerules", "cline"},
	{".roorules", "roo"},
	{filepath.Join(".github", "copilot-instructions.md"), "copilot"},
}

// Scan returns every agent instruction file discoverable under a home dir:
// fixed user-scope locations plus a bounded walk of common dev-project roots.
func Scan(h homes.Home) []Instruction {
	seen := map[string]struct{}{}
	var out []Instruction

	emit := func(path, tool, scope string) {
		if _, ok := seen[path]; path == "" || ok || !fsutil.Exists(path) {
			return
		}
		seen[path] = struct{}{}
		out = append(out, build(h, path, tool, scope))
	}

	for _, p := range userProbes() {
		emit(filepath.Join(h.Dir, p.rel), p.tool, "user")
	}

	for _, root := range projectRoots(h.Dir) {
		fsutil.WalkBounded(root, 3, func(dir string) {
			for _, pf := range projectFiles {
				emit(filepath.Join(dir, pf.rel), pf.tool, "project")
			}
			// Cursor's newer rule format: .cursor/rules/*.mdc
			if matches, err := filepath.Glob(filepath.Join(dir, ".cursor", "rules", "*.mdc")); err == nil {
				for _, m := range matches {
					emit(m, "cursor", "project")
				}
			}
		})
	}
	return out
}

func projectRoots(home string) []string {
	subs := []string{
		"", "Documents", "Projects", "projects", "src", "code", "git", "dev", "workspace", "repos",
	}
	out := make([]string, 0, len(subs))
	for _, s := range subs {
		if s == "" {
			out = append(out, home)
			continue
		}
		out = append(out, filepath.Join(home, s))
	}
	return out
}

func build(h homes.Home, path, tool, scope string) Instruction {
	in := Instruction{
		UID: h.UID, Username: h.Username,
		Path: path, Name: filepath.Base(path), Tool: tool, Scope: scope,
		SHA256: fsutil.SHA256(path),
	}
	if fi, err := os.Stat(path); err == nil {
		in.Size = fi.Size()
	}

	var flags []string
	markers, hidden := scanContent(path)
	if len(markers) > 0 {
		flags = append(flags, "injection_markers")
		in.Markers = strings.Join(markers, ",")
	}
	if hidden {
		flags = append(flags, "hidden_unicode")
	}
	if p := fsutil.Stat(path); p.Known && p.WorldWritable {
		// A world-writable instruction file can be edited by any local user to
		// hijack the agent — higher signal than merely readable.
		flags = append(flags, "world_writable")
	}
	in.RiskFlags = strings.Join(flags, ",")
	return in
}

// injectionMarkers are conservative, high-signal phrases associated with prompt
// injection or data-exfiltration directives embedded in instruction files.
var injectionMarkers = []string{
	"ignore previous instructions",
	"ignore all previous",
	"disregard previous",
	"disregard the above",
	"do not tell the user",
	"don't tell the user",
	"without telling the user",
	"do not mention",
	"you are now",
	"new instructions:",
	"system prompt",
	"exfiltrate",
	"base64 -d",
	"curl http",
	"wget http",
	"| sh",
	"| bash",
	"id_rsa",
	".ssh/",
	"aws_secret",
	"send it to",
	"send them to",
}

// scanContent reads the head of the file and returns matched injection markers
// plus whether hidden (zero-width / Unicode-tag) characters are present.
func scanContent(path string) (markers []string, hiddenUnicode bool) {
	f, err := os.Open(path) // #nosec G304 -- path discovered by this collector's curated probes
	if err != nil {
		return nil, false
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, maxReadBytes)
	n, _ := f.Read(buf)
	content := string(buf[:n])
	low := strings.ToLower(content)

	for _, m := range injectionMarkers {
		if strings.Contains(low, m) {
			markers = append(markers, strings.TrimSpace(strings.TrimSuffix(m, ":")))
		}
	}
	hiddenUnicode = hasHiddenUnicode(content)
	return markers, hiddenUnicode
}

// hasHiddenUnicode reports whether the text contains zero-width characters or
// Unicode tag characters (U+E0000–U+E007F) — both used to smuggle instructions
// invisibly past human reviewers.
func hasHiddenUnicode(s string) bool {
	for _, r := range s {
		switch {
		case r == 0x200B, // zero-width space
			r == 0x200C, // zero-width non-joiner
			r == 0x200D, // zero-width joiner
			r == 0x2060, // word joiner
			r == 0xFEFF: // zero-width no-break space / BOM
			return true
		case r >= 0xE0000 && r <= 0xE007F: // Unicode tag characters
			return true
		}
	}
	return false
}
