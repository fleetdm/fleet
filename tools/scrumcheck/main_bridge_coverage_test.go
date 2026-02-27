package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestMainHelperProcess runs main() in a subprocess-controlled mode so tests
// can exercise os.Exit/log.Fatal paths safely.
func TestMainHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MAIN_HELPER") != "1" {
		t.Skip("helper process only")
	}

	switch os.Getenv("MAIN_HELPER_MODE") {
	case "no-project":
		os.Args = []string{"qacheck"}
		main()
	case "bad-stale-days":
		os.Args = []string{"qacheck", "-p", "71", "-stale-days", "0"}
		main()
	case "missing-token":
		os.Args = []string{
			"qacheck",
			"-p", "71",
			"-stale-days=1",
			"-bridge-idle-minutes=1",
		}
		_ = os.Unsetenv("GITHUB_TOKEN")
		main()
	case "happy-open-fails":
		os.Args = []string{
			"qacheck",
			"-p", "71",
			"-l", "#g-orchestration",
			"-bridge-idle-minutes=1",
			"-stale-days=1",
			"-limit=1",
		}
		main()
	default:
		t.Fatalf("unknown helper mode: %q", os.Getenv("MAIN_HELPER_MODE"))
	}
}

func runMainHelper(t *testing.T, mode string, extraEnv ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestMainHelperProcess")
	cmd.Env = append(os.Environ(),
		"GO_WANT_MAIN_HELPER=1",
		"MAIN_HELPER_MODE="+mode,
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return 0, string(out), ""
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), string(out), ee.Error()
	}
	t.Fatalf("unexpected helper execution error: %v", err)
	return -1, "", ""
}

func TestMainExitWhenProjectMissing(t *testing.T) {
	code, out, _ := runMainHelper(t, "no-project")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d\noutput:\n%s", code, out)
	}
	if !strings.Contains(out, "at least one project is required") {
		t.Fatalf("expected missing project message, got:\n%s", out)
	}
}

func TestMainExitWhenStaleDaysInvalid(t *testing.T) {
	code, out, _ := runMainHelper(t, "bad-stale-days")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d\noutput:\n%s", code, out)
	}
	if !strings.Contains(out, "-stale-days must be >= 1") {
		t.Fatalf("expected stale-days validation message, got:\n%s", out)
	}
}

func TestMainExitWhenTokenMissing(t *testing.T) {
	code, out, errText := runMainHelper(
		t,
		"missing-token",
	)
	if code != 1 {
		t.Fatalf("expected missing token exit code, got %d (%s)\noutput:\n%s", code, errText, out)
	}
	if !strings.Contains(out, "GITHUB_TOKEN env var is required") {
		t.Fatalf("expected missing token message, got:\n%s", out)
	}
}

func TestMainHappyPathWhenBrowserOpenFails(t *testing.T) {
	code, out, errText := runMainHelper(
		t,
		"happy-open-fails",
		"GITHUB_TOKEN=test-token",
		"PATH=/path/that/does/not/exist",
	)
	if code != 1 {
		t.Fatalf("expected bridge-required exit code 1, got %d (%s)\noutput:\n%s", code, errText, out)
	}
	if !strings.Contains(out, "bridge unavailable") {
		t.Fatalf("expected bridge-unavailable message in output, got:\n%s", out)
	}
}

func TestStartUIBridgeBasicLifecycle(t *testing.T) {
	if _, err := startUIBridge("", time.Minute, nil, bridgePolicy{}); err == nil {
		t.Fatal("expected error for missing token")
	}

	bridge, err := startUIBridge("token", 10*time.Second, nil, bridgePolicy{})
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("sandbox blocks local listen: %v", err)
		}
		t.Fatalf("startUIBridge failed: %v", err)
	}
	t.Cleanup(func() { _ = bridge.stop("test cleanup") })

	if bridge.idleTimeout != 15*time.Minute {
		t.Fatalf("expected minimum idle timeout to be applied, got %s", bridge.idleTimeout)
	}

	resp, err := http.Get(bridge.baseURL + "/healthz")
	if err != nil {
		t.Fatalf("healthz request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if err := bridge.stop("test shutdown"); err != nil {
		t.Fatalf("bridge stop failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	reason := bridge.waitUntilDone(ctx)
	if reason == "" {
		t.Fatal("expected non-empty bridge stop reason")
	}
}

func TestMainInProcessCoveragePath(t *testing.T) {
	oldArgs := os.Args
	oldToken := os.Getenv("GITHUB_TOKEN")
	oldFlagSet := flag.CommandLine
	oldStartBridge := startUIBridgeFn
	oldOpenBrowser := openInBrowserFn
	t.Cleanup(func() {
		os.Args = oldArgs
		if oldToken == "" {
			_ = os.Unsetenv("GITHUB_TOKEN")
		} else {
			_ = os.Setenv("GITHUB_TOKEN", oldToken)
		}
		flag.CommandLine = oldFlagSet
		startUIBridgeFn = oldStartBridge
		openInBrowserFn = oldOpenBrowser
	})

	// Use a fresh FlagSet so main can register its flags in-process during tests.
	flag.CommandLine = flag.NewFlagSet("qacheck-test", flag.ContinueOnError)
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	startUIBridgeFn = func(_ string, _ time.Duration, _ func(string), _ bridgePolicy) (*uiBridge, error) {
		done := make(chan struct{})
		close(done)
		return &uiBridge{
			baseURL: "http://127.0.0.1:1",
			session: "sess",
			done:    done,
			reason:  "test done",
		}, nil
	}
	openInBrowserFn = func(_ string) error { return nil }
	os.Args = []string{
		"qacheck",
		"-p", "71",
		"-l", "#g-orchestration",
		"-bridge-idle-minutes=1",
		"-stale-days=1",
		"-limit=1",
	}

	if code := run(); code != 0 {
		t.Fatalf("expected success run code, got %d", code)
	}
}

func TestMainInProcessOpenDisabledPath(t *testing.T) {
	oldArgs := os.Args
	oldToken := os.Getenv("GITHUB_TOKEN")
	oldFlagSet := flag.CommandLine
	oldStartBridge := startUIBridgeFn
	oldOpenBrowser := openInBrowserFn
	t.Cleanup(func() {
		os.Args = oldArgs
		if oldToken == "" {
			_ = os.Unsetenv("GITHUB_TOKEN")
		} else {
			_ = os.Setenv("GITHUB_TOKEN", oldToken)
		}
		flag.CommandLine = oldFlagSet
		startUIBridgeFn = oldStartBridge
		openInBrowserFn = oldOpenBrowser
	})

	flag.CommandLine = flag.NewFlagSet("qacheck-test", flag.ContinueOnError)
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	startUIBridgeFn = func(_ string, _ time.Duration, _ func(string), _ bridgePolicy) (*uiBridge, error) {
		done := make(chan struct{})
		close(done)
		return &uiBridge{
			baseURL: "http://127.0.0.1:1",
			session: "sess",
			done:    done,
			reason:  "test done",
		}, nil
	}
	openInBrowserFn = func(_ string) error { return nil }
	os.Args = []string{
		"qacheck",
		"-p", "71",
		"-open-report=false",
		"-bridge-idle-minutes=1",
		"-stale-days=1",
		"-limit=1",
	}

	if code := run(); code != 0 {
		t.Fatalf("expected success run code, got %d", code)
	}
}

func TestRunValidationNoProject(t *testing.T) {
	oldArgs := os.Args
	oldFlagSet := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagSet
	})
	flag.CommandLine = flag.NewFlagSet("qacheck-test", flag.ContinueOnError)
	os.Args = []string{"qacheck"}
	if code := run(); code != 2 {
		t.Fatalf("expected code 2 for missing project, got %d", code)
	}
}

func TestRunValidationBadStaleDays(t *testing.T) {
	oldArgs := os.Args
	oldFlagSet := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagSet
	})
	flag.CommandLine = flag.NewFlagSet("qacheck-test", flag.ContinueOnError)
	os.Args = []string{"qacheck", "-p", "71", "-stale-days", "0"}
	if code := run(); code != 2 {
		t.Fatalf("expected code 2 for invalid stale-days, got %d", code)
	}
}

func TestRunValidationTokenMissing(t *testing.T) {
	oldArgs := os.Args
	oldToken := os.Getenv("GITHUB_TOKEN")
	oldFlagSet := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		if oldToken == "" {
			_ = os.Unsetenv("GITHUB_TOKEN")
		} else {
			_ = os.Setenv("GITHUB_TOKEN", oldToken)
		}
		flag.CommandLine = oldFlagSet
	})
	flag.CommandLine = flag.NewFlagSet("qacheck-test", flag.ContinueOnError)
	_ = os.Unsetenv("GITHUB_TOKEN")
	os.Args = []string{"qacheck", "-p", "71"}
	if code := run(); code != 1 {
		t.Fatalf("expected code 1 when token missing, got %d", code)
	}
}

func TestRunBridgeStartFailurePath(t *testing.T) {
	oldArgs := os.Args
	oldToken := os.Getenv("GITHUB_TOKEN")
	oldFlagSet := flag.CommandLine
	oldStartBridge := startUIBridgeFn
	oldOpenBrowser := openInBrowserFn
	t.Cleanup(func() {
		os.Args = oldArgs
		if oldToken == "" {
			_ = os.Unsetenv("GITHUB_TOKEN")
		} else {
			_ = os.Setenv("GITHUB_TOKEN", oldToken)
		}
		flag.CommandLine = oldFlagSet
		startUIBridgeFn = oldStartBridge
		openInBrowserFn = oldOpenBrowser
	})

	flag.CommandLine = flag.NewFlagSet("qacheck-test", flag.ContinueOnError)
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	startUIBridgeFn = func(_ string, _ time.Duration, _ func(string), _ bridgePolicy) (*uiBridge, error) {
		return nil, context.DeadlineExceeded
	}
	openInBrowserFn = func(_ string) error { return nil }
	os.Args = []string{
		"qacheck",
		"-p", "71",
		"-open-report=false",
		"-bridge-idle-minutes=1",
		"-stale-days=1",
		"-limit=1",
	}

	if code := run(); code != 1 {
		t.Fatalf("expected run failure code when bridge cannot start, got %d", code)
	}
}

func TestRunWriteReportFailurePath(t *testing.T) {
	oldArgs := os.Args
	oldToken := os.Getenv("GITHUB_TOKEN")
	oldFlagSet := flag.CommandLine
	oldStartBridge := startUIBridgeFn
	oldOpenBrowser := openInBrowserFn
	t.Cleanup(func() {
		os.Args = oldArgs
		if oldToken == "" {
			_ = os.Unsetenv("GITHUB_TOKEN")
		} else {
			_ = os.Setenv("GITHUB_TOKEN", oldToken)
		}
		flag.CommandLine = oldFlagSet
		startUIBridgeFn = oldStartBridge
		openInBrowserFn = oldOpenBrowser
	})

	flag.CommandLine = flag.NewFlagSet("qacheck-test", flag.ContinueOnError)
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	startUIBridgeFn = func(_ string, _ time.Duration, _ func(string), _ bridgePolicy) (*uiBridge, error) {
		done := make(chan struct{})
		close(done)
		return &uiBridge{baseURL: "http://127.0.0.1:1", session: "sess", done: done, reason: "done"}, nil
	}
	openInBrowserFn = func(_ string) error { return nil }
	os.Args = []string{
		"qacheck",
		"-p", "71",
		"-bridge-idle-minutes=1",
		"-stale-days=1",
		"-limit=1",
	}

	if code := run(); code != 0 {
		t.Fatalf("expected run success code, got %d", code)
	}
}

func TestRunOpenBrowserFailurePath(t *testing.T) {
	oldArgs := os.Args
	oldToken := os.Getenv("GITHUB_TOKEN")
	oldFlagSet := flag.CommandLine
	oldStartBridge := startUIBridgeFn
	oldOpenBrowser := openInBrowserFn
	t.Cleanup(func() {
		os.Args = oldArgs
		if oldToken == "" {
			_ = os.Unsetenv("GITHUB_TOKEN")
		} else {
			_ = os.Setenv("GITHUB_TOKEN", oldToken)
		}
		flag.CommandLine = oldFlagSet
		startUIBridgeFn = oldStartBridge
		openInBrowserFn = oldOpenBrowser
	})

	flag.CommandLine = flag.NewFlagSet("qacheck-test", flag.ContinueOnError)
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	startUIBridgeFn = func(_ string, _ time.Duration, _ func(string), _ bridgePolicy) (*uiBridge, error) {
		done := make(chan struct{})
		close(done)
		return &uiBridge{baseURL: "http://127.0.0.1:1", session: "sess", done: done, reason: "done"}, nil
	}
	openInBrowserFn = func(_ string) error { return context.DeadlineExceeded }
	os.Args = []string{
		"qacheck",
		"-p", "71",
		"-bridge-idle-minutes=1",
		"-stale-days=1",
		"-limit=1",
	}

	if code := run(); code != 0 {
		t.Fatalf("expected run success code, got %d", code)
	}
}
