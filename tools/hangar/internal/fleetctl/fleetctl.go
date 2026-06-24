// Package fleetctl resolves the fleetctl binary, reads/writes the fleetctl
// config (~/.fleet/config), and runs one-shot fleetctl invocations with
// captured output. Ported from src-tauri/src/fleetctl.rs.
package fleetctl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
	"gopkg.in/yaml.v3"
)

// ResolvedBinary reports where fleetctl was found and how.
type ResolvedBinary struct {
	Path   string `json:"path"`
	Source string `json:"source"` // "settings" | "build" | "missing"
	Exists bool   `json:"exists"`
}

// ResolveBinary prefers an explicit settings path (the user may be testing a
// release binary), then <repo>/build/fleetctl, then reports missing.
func ResolveBinary(repo, settingsPath string) ResolvedBinary {
	if settingsPath != "" {
		expanded := paths.Expand(settingsPath)
		return ResolvedBinary{Path: expanded, Source: "settings", Exists: fileExists(expanded)}
	}
	if repo != "" {
		candidate := filepath.Join(repo, "build", "fleetctl")
		return ResolvedBinary{Path: candidate, Source: "build", Exists: fileExists(candidate)}
	}
	return ResolvedBinary{Source: "missing"}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ContextSummary is one fleetctl context (sensitive token reduced to a bool).
type ContextSummary struct {
	Name     string  `json:"name"`
	Address  *string `json:"address"`
	Email    *string `json:"email"`
	HasToken bool    `json:"has_token"`
}

// ContextInfo describes the parsed fleetctl config.
type ContextInfo struct {
	ConfigPath string           `json:"config_path"`
	Exists     bool             `json:"exists"`
	Current    *ContextSummary  `json:"current"`
	Contexts   []ContextSummary `json:"contexts"`
}

// RawConfig is the raw fleetctl config file contents.
type RawConfig struct {
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
	Contents string `json:"contents"`
}

// DefaultConfigPath is ~/.fleet/config — what fleetctl reads with no --config.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.fleet/config"
	}
	return filepath.Join(home, ".fleet", "config")
}

type rawContext struct {
	Address *string `yaml:"address"`
	Email   *string `yaml:"email"`
	Token   *string `yaml:"token"`
}

// parseContexts walks the contexts mapping in file order (using a yaml.Node
// so we don't lose ordering the way a Go map would), skipping non-string
// keys and undecodable values rather than failing the whole read.
func parseContexts(raw []byte) ([]ContextSummary, error) {
	var doc struct {
		Contexts yaml.Node `yaml:"contexts"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := []ContextSummary{}
	c := doc.Contexts
	if c.Kind != yaml.MappingNode {
		return out, nil
	}
	for i := 0; i+1 < len(c.Content); i += 2 {
		keyNode, valNode := c.Content[i], c.Content[i+1]
		if keyNode.Kind != yaml.ScalarNode {
			continue
		}
		var rc rawContext
		_ = valNode.Decode(&rc) // tolerate malformed entries (unwrap_or_default)
		out = append(out, ContextSummary{
			Name:     keyNode.Value,
			Address:  rc.Address,
			Email:    rc.Email,
			HasToken: rc.Token != nil && *rc.Token != "",
		})
	}
	return out, nil
}

// ReadConfigRaw returns the raw fleetctl config contents (empty if absent).
func ReadConfigRaw(configPath string) (RawConfig, error) {
	b, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return RawConfig{Path: configPath, Exists: false}, nil
	}
	if err != nil {
		return RawConfig{}, err
	}
	return RawConfig{Path: configPath, Exists: true, Contents: string(b)}, nil
}

// SaveConfig validates that yaml parses, then writes it to configPath
// (creating the parent dir).
func SaveConfig(configPath, yamlText string) error {
	var probe any
	if err := yaml.Unmarshal([]byte(yamlText), &probe); err != nil {
		return fmt.Errorf("YAML parse error: %w", err)
	}
	if parent := filepath.Dir(configPath); parent != "" {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", parent, err)
		}
	}
	return os.WriteFile(configPath, []byte(yamlText), 0o644)
}

// ReadContext parses the fleetctl config at configPath. The current context
// is "default" (Hangar always launches fleetctl with no overrides).
func ReadContext(configPath string) (ContextInfo, error) {
	b, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return ContextInfo{ConfigPath: configPath, Exists: false, Contexts: []ContextSummary{}}, nil
	}
	if err != nil {
		return ContextInfo{}, err
	}
	contexts, err := parseContexts(b)
	if err != nil {
		return ContextInfo{}, fmt.Errorf("parse %s: %w", configPath, err)
	}
	var current *ContextSummary
	for i := range contexts {
		if contexts[i].Name == "default" {
			c := contexts[i]
			current = &c
			break
		}
	}
	return ContextInfo{ConfigPath: configPath, Exists: true, Current: current, Contexts: contexts}, nil
}

// CapturedRun is the result of a one-shot command.
type CapturedRun struct {
	ExitCode *int   `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// RunCapture runs program synchronously with the login-shell PATH (plus any
// caller env), optional cwd and stdin, and a timeout (default 60s). Use for
// short, finite commands; long/streamy commands go through the process
// engine so output lands in the Logs tab.
func RunCapture(program, cwd string, args []string, env map[string]string, stdinData string, timeoutMS uint64) (CapturedRun, error) {
	if timeoutMS == 0 {
		timeoutMS = 60_000
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	cmd := shellpath.CommandContext(ctx, program, args...)
	cmd.Env = shellpath.MergeEnv(cmd.Env, env)
	if cwd != "" {
		cmd.Dir = cwd
	}
	if stdinData != "" {
		cmd.Stdin = strings.NewReader(stdinData)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return CapturedRun{}, fmt.Errorf("timed out after %dms", timeoutMS)
	}

	run := CapturedRun{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		zero := 0
		run.ExitCode = &zero
		return run, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		// Exited normally → real code; killed by signal → leave nil (matches
		// Rust's ExitStatus::code() returning None on signal termination).
		if ee.ProcessState.Exited() {
			code := ee.ExitCode()
			run.ExitCode = &code
		}
		return run, nil
	}
	return CapturedRun{}, fmt.Errorf("spawn %s: %w", program, err)
}
