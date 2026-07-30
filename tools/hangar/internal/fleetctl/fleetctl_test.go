package fleetctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveBinary(t *testing.T) {
	base := t.TempDir()

	// Settings path that exists wins.
	bin := filepath.Join(base, "fleetctl")
	os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755)
	got := ResolveBinary(filepath.Join(base, "repo"), bin)
	if got.Source != "settings" || !got.Exists || got.Path != bin {
		t.Errorf("settings resolve = %+v", got)
	}

	// No settings path → repo/build/fleetctl.
	repo := filepath.Join(base, "repo")
	os.MkdirAll(filepath.Join(repo, "build"), 0o755)
	os.WriteFile(filepath.Join(repo, "build", "fleetctl"), []byte("x"), 0o755)
	got = ResolveBinary(repo, "")
	if got.Source != "build" || !got.Exists {
		t.Errorf("build resolve = %+v", got)
	}

	// Neither → missing.
	got = ResolveBinary("", "")
	if got.Source != "missing" || got.Exists {
		t.Errorf("missing resolve = %+v", got)
	}
}

func TestParseContextsOrderAndToken(t *testing.T) {
	raw := []byte(`contexts:
  default:
    address: https://localhost:8080
    email: admin@example.com
    token: secrettoken
  staging:
    address: https://staging
    token: ""
  notoken:
    address: https://x
`)
	got, err := parseContexts(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d contexts, want 3", len(got))
	}
	// File order preserved.
	if got[0].Name != "default" || got[1].Name != "staging" || got[2].Name != "notoken" {
		t.Errorf("order not preserved: %v", []string{got[0].Name, got[1].Name, got[2].Name})
	}
	if !got[0].HasToken {
		t.Error("default should have a token")
	}
	if got[1].HasToken {
		t.Error("empty token should be has_token=false")
	}
	if got[2].HasToken {
		t.Error("absent token should be has_token=false")
	}
	if got[0].Address == nil || *got[0].Address != "https://localhost:8080" {
		t.Errorf("address = %v", got[0].Address)
	}
}

func TestParseContextsNoContexts(t *testing.T) {
	got, err := parseContexts([]byte("something_else: 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("want 0 contexts, got %d", len(got))
	}
}

func TestReadContext(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")

	// Missing file.
	info, err := ReadContext(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if info.Exists || info.Current != nil {
		t.Errorf("missing config: %+v", info)
	}

	// With a default context.
	os.WriteFile(cfg, []byte("contexts:\n  default:\n    address: https://x\n    token: t\n"), 0o644)
	info, err = ReadContext(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Exists || info.Current == nil || info.Current.Name != "default" {
		t.Errorf("expected current=default: %+v", info)
	}

	// Without a default context → current nil.
	os.WriteFile(cfg, []byte("contexts:\n  other:\n    address: https://y\n"), 0o644)
	info, _ = ReadContext(cfg)
	if info.Current != nil {
		t.Errorf("no default → current should be nil, got %+v", info.Current)
	}
}

func TestReadConfigRaw(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")

	rc, err := ReadConfigRaw(cfg)
	if err != nil || rc.Exists {
		t.Errorf("missing config: %+v err=%v", rc, err)
	}

	os.WriteFile(cfg, []byte("hello: world\n"), 0o644)
	rc, err = ReadConfigRaw(cfg)
	if err != nil || !rc.Exists || rc.Contents != "hello: world\n" {
		t.Errorf("present config: %+v err=%v", rc, err)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".fleet", "config") // parent doesn't exist yet

	if err := SaveConfig(cfg, "this: : invalid: yaml: ["); err == nil {
		t.Error("invalid yaml should be rejected")
	}

	if err := SaveConfig(cfg, "contexts:\n  default:\n    address: x\n"); err != nil {
		t.Fatalf("valid yaml should save: %v", err)
	}
	b, err := os.ReadFile(cfg)
	if err != nil || !strings.Contains(string(b), "default") {
		t.Errorf("config not written: %v", err)
	}
}

func TestRunCapture(t *testing.T) {
	// stdout + exit 0.
	run, err := RunCapture("sh", "", []string{"-c", "printf hello"}, nil, "", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if run.Stdout != "hello" || run.ExitCode == nil || *run.ExitCode != 0 {
		t.Errorf("echo run = %+v", run)
	}

	// Non-zero exit code captured.
	run, _ = RunCapture("sh", "", []string{"-c", "exit 3"}, nil, "", 5000)
	if run.ExitCode == nil || *run.ExitCode != 3 {
		t.Errorf("exit code = %v, want 3", run.ExitCode)
	}

	// stdin is piped through.
	run, _ = RunCapture("cat", "", nil, nil, "piped-in", 5000)
	if run.Stdout != "piped-in" {
		t.Errorf("stdin not piped: %q", run.Stdout)
	}

	// Caller env is applied.
	run, _ = RunCapture("sh", "", []string{"-c", "printf %s \"$HANGAR_TEST\""}, map[string]string{"HANGAR_TEST": "yes"}, "", 5000)
	if run.Stdout != "yes" {
		t.Errorf("env not applied: %q", run.Stdout)
	}

	// Timeout.
	if _, err := RunCapture("sleep", "", []string{"5"}, nil, "", 100); err == nil {
		t.Error("expected timeout error")
	}
}
