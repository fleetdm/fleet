package mcp

import (
	"encoding/json"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/classify"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
)

// fetchRunners launch code that is downloaded at invocation time rather than
// installed and pinned. Any MCP server launched through one of these executes
// remote code every time the client starts it — the core MCP supply-chain risk.
var fetchRunners = map[string]struct{}{
	"npx": {}, "npx.cmd": {},
	"bunx": {}, "bunx.cmd": {},
	"pnpx": {}, "pnpx.cmd": {},
	"uvx": {}, "uvx.cmd": {},
	"dlx": {},
}

// secretKeyMarkers identify an env-var NAME that conventionally holds a secret.
// MCP configs store env values inline, so a secret-shaped key means a plaintext
// credential sits in the config file on disk.
var secretKeyMarkers = []string{
	"token", "secret", "password", "passwd", "passphrase",
	"apikey", "api_key", "access_key", "private_key", "credential",
	"client_secret", "bearer", "session", "auth_key",
}

// enrichRisk computes the static security posture of a server in place:
// capability tags, a launch-spec hash (rug-pull diffing), a config-file hash,
// and the risk_flags token set. Safe to call on both config- and
// process-sourced servers (fields that don't apply are simply skipped).
func (s *Server) enrichRisk() {
	hay := strings.ToLower(s.ServerName + " " + s.Command + " " + s.Args + " " + s.URL)
	caps := classify.MCPCapabilities(hay)
	s.Capabilities = strings.Join(caps, ",")
	s.LaunchHash = launchHash(s.Command, s.Args, s.URL)
	if s.ConfigPath != "" {
		s.SHA256 = fsutil.SHA256(s.ConfigPath)
	}

	var flags []string
	add := func(f string) { flags = append(flags, f) }

	// Remote fetch-and-run supply-chain surface.
	if _, pkg, ok := fetchSpec(s.Command, s.Args); ok {
		add("remote_fetch_exec")
		if !isPinned(pkg) {
			add("unpinned_dependency")
		}
	}

	// Inferred high-risk capabilities.
	for _, c := range caps {
		switch c {
		case "shell-exec":
			add("mcp_shell_exec")
		case "fs-write":
			add("mcp_fs_write")
		}
	}

	// Plaintext secret in config (env value stored inline).
	if hasSecretEnv(s.EnvKeys) {
		add("plaintext_secret")
	}

	// Config file readable beyond its owner.
	if s.ConfigPath != "" {
		if p := fsutil.Stat(s.ConfigPath); p.Known && p.WorldReadable {
			add("world_readable_config")
		}
	}

	// Remote MCP reached over cleartext HTTP.
	if s.Location == "remote" && strings.HasPrefix(strings.ToLower(s.URL), "http://") {
		add("cleartext_endpoint")
	}

	s.RiskFlags = strings.Join(flags, ",")
}

// fetchSpec reports whether the command is a fetch-runner and, if so, the first
// non-flag argument (the package spec being fetched).
func fetchSpec(command, argsJSON string) (runner, pkg string, ok bool) {
	base := baseCmd(command)
	if _, ok := fetchRunners[strings.ToLower(base)]; base == "" || !ok {
		return "", "", false
	}
	var args []string
	if argsJSON != "" {
		_ = json.Unmarshal([]byte(argsJSON), &args)
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue // skip flags like -y / --yes
		}
		return base, a, true
	}
	return base, "", true // runner with no package arg is still a fetch surface
}

// isPinned reports whether an npm/pip-style package spec carries an exact
// version. Scoped names (@scope/name) carry a leading '@' that is not a version
// separator, so the version '@' must appear after the first character.
func isPinned(pkg string) bool {
	if pkg == "" {
		return false
	}
	at := strings.LastIndex(pkg, "@")
	if at <= 0 { // no '@', or only the leading scope '@'
		return false
	}
	ver := strings.ToLower(pkg[at+1:])
	if ver == "" || ver == "latest" || ver == "next" || ver == "*" {
		return false
	}
	return true
}

// hasSecretEnv reports whether any env-var NAME in the JSON array looks like a
// secret holder.
func hasSecretEnv(envKeysJSON string) bool {
	if envKeysJSON == "" {
		return false
	}
	var keys []string
	if err := json.Unmarshal([]byte(envKeysJSON), &keys); err != nil {
		return false
	}
	for _, k := range keys {
		low := strings.ToLower(k)
		for _, m := range secretKeyMarkers {
			if strings.Contains(low, m) {
				return true
			}
		}
	}
	return false
}

// launchHash is a stable fingerprint of how a server is launched. A change
// between scans flags a silently-mutated launch vector (rug-pull). Uses the
// content hasher over a normalized string so it shares the SHA-256 format with
// the file hashes.
func launchHash(command, argsJSON, url string) string {
	spec := strings.TrimSpace(command + "\x00" + argsJSON + "\x00" + url)
	if spec == "\x00\x00" || spec == "" {
		return ""
	}
	return fsutil.SHA256Bytes([]byte(spec))
}
