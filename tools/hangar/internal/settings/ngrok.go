package settings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

// NgrokRunningTunnel is one live tunnel from ngrok's local API, including the
// public URL it's forwarding from.
type NgrokRunningTunnel struct {
	Name      string `json:"name"`
	PublicURL string `json:"public_url"`
	Proto     string `json:"proto"`
	Addr      string `json:"addr"`
}

// ngrokAPIURL is ngrok's local inspection API. Default is 127.0.0.1:4040;
// overridable for a non-default web_addr via NGROK_API_ADDR (host:port).
func ngrokAPIURL() string {
	if v := strings.TrimSpace(os.Getenv("NGROK_API_ADDR")); v != "" {
		return "http://" + v + "/api/tunnels"
	}
	return "http://127.0.0.1:4040/api/tunnels"
}

// FetchNgrokTunnels queries ngrok's local API for the currently-running tunnels
// and their public URLs. When ngrok isn't running (or its API is disabled) the
// request fails; we return an empty slice with no error so the UI just shows
// nothing rather than surfacing a scary error.
func FetchNgrokTunnels() ([]NgrokRunningTunnel, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(ngrokAPIURL())
	if err != nil {
		return []NgrokRunningTunnel{}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []NgrokRunningTunnel{}, nil
	}
	var parsed struct {
		Tunnels []struct {
			Name      string `json:"name"`
			PublicURL string `json:"public_url"`
			Proto     string `json:"proto"`
			Config    struct {
				Addr string `json:"addr"`
			} `json:"config"`
		} `json:"tunnels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return []NgrokRunningTunnel{}, nil
	}
	out := make([]NgrokRunningTunnel, 0, len(parsed.Tunnels))
	for _, t := range parsed.Tunnels {
		out = append(out, NgrokRunningTunnel{
			Name:      t.Name,
			PublicURL: t.PublicURL,
			Proto:     t.Proto,
			Addr:      t.Config.Addr,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
