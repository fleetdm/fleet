package jarvis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Config is jarvis's user configuration. It's optional — every field has a
// sensible default — and lives at ~/.config/gm/jarvis/config.json.
type Config struct {
	// CloneBaseDirs are the directories jarvis scans (one level deep) for local
	// clones of the target repo, used by Start Work to offer a working copy.
	// Defaults to ["~/projects"]. Paths may use ~ for the home directory.
	CloneBaseDirs []string `json:"clone_base_dirs,omitempty"`

	// BranchScanGlob is the folder-name glob (relative to each CloneBaseDirs
	// entry) that the branch-cleanup view scans for local git repos. Defaults to
	// "fleet*" so it picks up ~/projects/fleet, fleet-plan, fleet-2, etc.
	BranchScanGlob string `json:"branch_scan_glob,omitempty"`

	// PrimaryProjects are the project boards whose assigned-to-you issues surface
	// in the top "YOUR PROJECTS" section. Each entry may be a project number, a
	// known gm alias, or a project name/title (e.g. "g-apple-at-work"). Managers
	// of multiple teams can list several.
	PrimaryProjects []string `json:"primary_projects,omitempty"`

	// Role is the user's primary role: "developer" (default), "manager", "qa", or
	// "design". It tailors the seed prompt sent to a launched Claude session and,
	// for "qa", surfaces issues awaiting QA with a test/verify action. Chosen at
	// onboarding. Empty means "never chosen" (treated as developer via EffectiveRole).
	Role string `json:"role,omitempty"`

	// StartPrompts optionally overrides the start-work prompt per role, keyed by
	// role name ("developer", "qa", …). Each value is a Go text/template rendered
	// against PromptData — available fields: {{.Issue}}, {{.Title}}, {{.URL}},
	// {{.Branch}}. Unset roles use the built-in default for that role.
	StartPrompts map[string]string `json:"start_prompts,omitempty"`
}

// EffectiveRole returns the user's role, defaulting to RoleDeveloper when unset or
// unrecognized. Use this everywhere behavior branches on role.
func (c *Config) EffectiveRole() string {
	if c == nil {
		return RoleDeveloper
	}
	return normalizeRole(c.Role)
}

// normalizeProjectName lowercases a project name and strips a leading '#' so
// "#g-apple-at-work" and "g-apple-at-work" compare equal.
func normalizeProjectName(s string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(s), "#"))
}

// projectHandle derives a clean, human-editable config entry from a project title.
// Titles carry emoji/`#` prefixes (e.g. "🍎 #g-apple-at-work"); we take the text
// after the first '#', else strip any leading non-letter/digit run (the emoji).
// The result (e.g. "g-apple-at-work") resolves back via resolveProject's name match.
func projectHandle(title string) string {
	if i := strings.IndexByte(title, '#'); i >= 0 {
		return strings.TrimSpace(title[i+1:])
	}
	t := strings.TrimSpace(title)
	for i, r := range t {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return strings.TrimSpace(t[i:])
		}
	}
	return t
}

// DefaultConfigPath returns ~/.config/gm/jarvis/config.json.
func DefaultConfigPath() string {
	return configPath("config.json")
}

// LoadConfig reads config from disk, filling defaults for any unset field.
func LoadConfig(path string) *Config {
	c := &Config{}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, c)
	}
	if len(c.CloneBaseDirs) == 0 {
		c.CloneBaseDirs = []string{"~/projects"}
	}
	if c.BranchScanGlob == "" {
		c.BranchScanGlob = "fleet*"
	}
	return c
}

// Save writes the config to disk as indented JSON, creating parent dirs as needed.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
		}
	}
	return p
}
