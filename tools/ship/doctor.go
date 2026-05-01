package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CheckStatus is the outcome of a single preflight check.
type CheckStatus int

const (
	CheckOK CheckStatus = iota
	CheckMissing
	CheckWarn // present but not quite right (e.g. wrong version)
)

// Check is one row in the doctor screen.
type Check struct {
	// Name is what gets shown in the left column ("Homebrew", "Go", etc.).
	Name string
	// Status is the outcome.
	Status CheckStatus
	// Detail is the right-column annotation: a version string when OK, or a
	// short reason when missing/warn ("not installed", "needs v24, you have v22").
	Detail string
}

// runChecks performs every preflight check and returns results in display order.
// Each check is short and runs sequentially; the whole sweep is well under a second.
func runChecks(ctx context.Context) []Check {
	checks := []Check{
		checkXcodeCLT(ctx),
		checkBrew(ctx),
		checkGo(ctx),
		checkNode(ctx),
		checkYarn(ctx),
		checkDocker(ctx),
	}
	if isAppleSilicon() {
		checks = append(checks, checkRosetta(ctx))
	}
	checks = append(checks, checkNgrok(ctx))
	return checks
}

// AllOK reports whether every check passed (no missing, no warns).
func AllOK(checks []Check) bool {
	for _, c := range checks {
		if c.Status != CheckOK {
			return false
		}
	}
	return true
}

// runCmd is a tiny wrapper that runs a command with a short timeout and
// returns trimmed combined output. Errors are returned alongside any output.
func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func isAppleSilicon() bool {
	return runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
}

// --- Individual checks --------------------------------------------------------

func checkXcodeCLT(ctx context.Context) Check {
	out, err := runCmd(ctx, "xcode-select", "-p")
	if err != nil || out == "" {
		return Check{Name: "Xcode CLT", Status: CheckMissing, Detail: "not installed"}
	}
	return Check{Name: "Xcode CLT", Status: CheckOK, Detail: "installed"}
}

func checkBrew(ctx context.Context) Check {
	out, err := runCmd(ctx, "brew", "--version")
	if err != nil {
		return Check{Name: "Homebrew", Status: CheckMissing, Detail: "not installed"}
	}
	// First line looks like "Homebrew 4.5.2"
	first := strings.SplitN(out, "\n", 2)[0]
	version := strings.TrimPrefix(first, "Homebrew ")
	return Check{Name: "Homebrew", Status: CheckOK, Detail: version}
}

func checkGo(ctx context.Context) Check {
	out, err := runCmd(ctx, "go", "version")
	if err != nil {
		return Check{Name: "Go", Status: CheckMissing, Detail: "not installed"}
	}
	// "go version go1.26.2 darwin/arm64"
	fields := strings.Fields(out)
	version := "unknown"
	if len(fields) >= 3 {
		version = strings.TrimPrefix(fields[2], "go")
	}
	return Check{Name: "Go", Status: CheckOK, Detail: version}
}

func checkNode(ctx context.Context) Check {
	out, err := runCmd(ctx, "node", "--version")
	if err != nil {
		return Check{Name: "Node", Status: CheckMissing, Detail: "not installed (needs v24)"}
	}
	// Output: "v24.10.0"
	version := strings.TrimPrefix(out, "v")
	major := strings.SplitN(version, ".", 2)[0]
	if major != "24" {
		return Check{Name: "Node", Status: CheckWarn, Detail: fmt.Sprintf("v%s — Fleet wants v24", version)}
	}
	return Check{Name: "Node", Status: CheckOK, Detail: "v" + version}
}

func checkYarn(ctx context.Context) Check {
	out, err := runCmd(ctx, "yarn", "--version")
	if err != nil {
		return Check{Name: "Yarn", Status: CheckMissing, Detail: "not installed"}
	}
	return Check{Name: "Yarn", Status: CheckOK, Detail: out}
}

func checkDocker(ctx context.Context) Check {
	if _, err := runCmd(ctx, "docker", "--version"); err != nil {
		return Check{Name: "Docker", Status: CheckMissing, Detail: "not installed"}
	}
	if _, err := runCmd(ctx, "docker", "info"); err != nil {
		return Check{Name: "Docker", Status: CheckWarn, Detail: "installed but not running — open Docker Desktop"}
	}
	return Check{Name: "Docker", Status: CheckOK, Detail: "running"}
}

func checkRosetta(ctx context.Context) Check {
	// `arch -x86_64 true` exits 0 only if Rosetta 2 is installed and ready.
	if _, err := runCmd(ctx, "arch", "-x86_64", "true"); err != nil {
		return Check{Name: "Rosetta 2", Status: CheckMissing, Detail: "not installed (Apple Silicon)"}
	}
	return Check{Name: "Rosetta 2", Status: CheckOK, Detail: "installed"}
}

func checkNgrok(ctx context.Context) Check {
	out, err := runCmd(ctx, "ngrok", "--version")
	if err != nil {
		return Check{Name: "ngrok", Status: CheckMissing, Detail: "not installed"}
	}
	// "ngrok version 3.5.0"
	version := "installed"
	if fields := strings.Fields(out); len(fields) >= 3 {
		version = fields[2]
	}
	return Check{Name: "ngrok", Status: CheckOK, Detail: version}
}

