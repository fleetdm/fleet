package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"gopkg.in/yaml.v3"
)

// NgrokTunnel is one tunnel definition from ngrok.yml.
type NgrokTunnel struct {
	Name  string `json:"name"`
	Proto string `json:"proto"`
	Addr  string `json:"addr"`
}

// NgrokYamlInfo summarizes an ngrok.yml for the Settings UI badge.
type NgrokYamlInfo struct {
	Valid        bool          `json:"valid"`
	Error        *string       `json:"error"`
	ResolvedPath string        `json:"resolved_path"`
	HasAuthtoken bool          `json:"has_authtoken"`
	Tunnels      []NgrokTunnel `json:"tunnels"`
}

// DefaultNgrokYmlPath is ngrok's discovery default on macOS.
func DefaultNgrokYmlPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/Library/Application Support/ngrok/ngrok.yml"
	}
	return filepath.Join(home, "Library/Application Support/ngrok/ngrok.yml")
}

type ngrokRaw struct {
	// ngrok v2 put authtoken at the top level; v3 moved it under agent:.
	// Accept either so the badge doesn't lie about a missing token.
	Authtoken *string `yaml:"authtoken"`
	Agent     *struct {
		Authtoken *string `yaml:"authtoken"`
	} `yaml:"agent"`
	Tunnels map[string]struct {
		Proto string `yaml:"proto"`
		// addr may be a bare port number or a host:port string. A yaml.Node
		// keeps the scalar's string form (.Value) for either.
		Addr yaml.Node `yaml:"addr"`
	} `yaml:"tunnels"`
}

func nonEmptyTrim(p *string) bool {
	return p != nil && strings.TrimSpace(*p) != ""
}

// ParseNgrokYml reads and summarizes an ngrok.yml. An empty path resolves
// to ngrok's macOS default. Parse/IO problems are reported via the returned
// struct (Valid=false, Error set), not as a Go error — matching the Rust
// command which always returned Ok.
func ParseNgrokYml(path string) NgrokYamlInfo {
	resolved := DefaultNgrokYmlPath()
	if path != "" {
		resolved = paths.Expand(path)
	}

	fail := func(msg string) NgrokYamlInfo {
		return NgrokYamlInfo{Valid: false, Error: &msg, ResolvedPath: resolved}
	}

	raw, err := os.ReadFile(resolved)
	if os.IsNotExist(err) {
		return fail("file not found")
	}
	if err != nil {
		return fail(fmt.Sprintf("read error: %v", err))
	}
	var parsed ngrokRaw
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		return fail(fmt.Sprintf("parse error: %v", err))
	}

	hasToken := nonEmptyTrim(parsed.Authtoken)
	if !hasToken && parsed.Agent != nil {
		hasToken = nonEmptyTrim(parsed.Agent.Authtoken)
	}

	tunnels := make([]NgrokTunnel, 0, len(parsed.Tunnels))
	for name, tn := range parsed.Tunnels {
		tunnels = append(tunnels, NgrokTunnel{
			Name:  name,
			Proto: tn.Proto,
			Addr:  tn.Addr.Value,
		})
	}
	sort.Slice(tunnels, func(i, j int) bool { return tunnels[i].Name < tunnels[j].Name })

	return NgrokYamlInfo{
		Valid:        true,
		ResolvedPath: resolved,
		HasAuthtoken: hasToken,
		Tunnels:      tunnels,
	}
}
