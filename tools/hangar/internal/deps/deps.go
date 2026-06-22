// Package deps runs the first-run dependency checks: detect required
// tooling, compare versions against what the Fleet repo declares, and return
// a structured report. Ported from src-tauri/src/deps.rs. macOS-first.
package deps

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

// DepCheck is one row of the dependency checklist.
type DepCheck struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Installed bool    `json:"installed"`
	Version   *string `json:"version"`
	Required  *string `json:"required"`
	// VersionOK is nil when there's no requirement to compare against.
	VersionOK *bool `json:"version_ok"`
	// RuntimeOK is the daemon/runtime state for tools that need more than a
	// binary on disk (Docker). nil when not applicable.
	RuntimeOK      *bool   `json:"runtime_ok"`
	InstallCommand string  `json:"install_command"`
	DocURL         *string `json:"doc_url"`
	Note           *string `json:"note"`
}

// DepReport is the full checklist.
type DepReport struct {
	Checks []DepCheck `json:"checks"`
}

func sp(s string) *string { return &s }

// ---- pure helpers (tested directly) ----

var versionRe = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?`)

// extractVersion pulls the first SemVer-looking token out of arbitrary CLI
// output ("git version 2.39.5", "go version go1.26.3 ...", "v24.10.0"),
// padding "x.y" to "x.y.0". Returns "" when nothing matches.
func extractVersion(s string) string {
	m := versionRe.FindString(s)
	if m == "" {
		return ""
	}
	if strings.Count(m, ".") == 1 {
		return m + ".0"
	}
	return m
}

// satisfies reports whether detected meets requirement, or nil if either
// fails to parse. NOTE: uses Masterminds/semver constraint syntax; for the
// values Fleet declares (e.g. "^24.10.0") it agrees with the Rust semver
// crate, but exotic ranges could differ in dialect.
func satisfies(detected, requirement string) *bool {
	c, err := semver.NewConstraint(requirement)
	if err != nil {
		return nil
	}
	v, err := semver.NewVersion(detected)
	if err != nil {
		return nil
	}
	ok := c.Check(v)
	return &ok
}

// readEngines reads engines.<key> from a package.json.
func readEngines(pkgPath, key string) (string, bool) {
	b, err := os.ReadFile(pkgPath)
	if err != nil {
		return "", false
	}
	var doc struct {
		Engines map[string]any `json:"engines"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return "", false
	}
	if s, ok := doc.Engines[key].(string); ok {
		return s, true
	}
	return "", false
}

// requiredNodeVersion prefers the repo's engines.node, falling back to a
// known-good value before a repo is picked.
func requiredNodeVersion(repo string) string {
	if repo != "" {
		if v, ok := readEngines(filepath.Join(repo, "package.json"), "node"); ok {
			return v
		}
	}
	return "^24.10.0"
}

// ---- command runners ----

// run executes cmd with PATH overridden to path. Returns (success, combined
// output, ran). ran is false only when the binary couldn't be spawned.
// Prefers stdout, falling back to stderr (some tools print --version there).
func run(path, cmd string, args ...string) (ok bool, out string, ran bool) {
	// Resolve against the probed PATH (not the app's bare process PATH that
	// exec.Command would otherwise search), then run with that PATH in env.
	prog := cmd
	if resolved, err := shellpath.LookPathIn(path, cmd); err == nil {
		prog = resolved
	}
	c := exec.Command(prog, args...)
	c.Env = shellpath.MergeEnv(os.Environ(), map[string]string{"PATH": path})
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return false, "", false // spawn failed (not found)
		}
	}
	combined := strings.TrimSpace(stdout.String())
	if combined == "" {
		combined = strings.TrimSpace(stderr.String())
	}
	return err == nil, combined, true
}

// runLoginShell runs a script through the login shell (for nvm, which is a
// sourced shell function, not a binary). Captures stdout only.
func runLoginShell(script string) (ok bool, out string, ran bool) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	output, err := exec.Command(shell, "-lc", script).Output()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return false, "", false
		}
	}
	return err == nil, strings.TrimSpace(string(output)), true
}

// versionFrom runs `cmd args...` and returns (installed, version*).
func versionFrom(path, cmd string, args ...string) (bool, *string) {
	if ok, out, ran := run(path, cmd, args...); ran && ok {
		if v := extractVersion(out); v != "" {
			return true, &v
		}
		return true, nil
	}
	return false, nil
}

// ---- individual checks ----

func checkXcode(path string) DepCheck {
	ok, _, ran := run(path, "xcode-select", "-p")
	return DepCheck{
		ID: "xcode-clt", Name: "Xcode Command Line Tools", Installed: ran && ok,
		InstallCommand: "xcode-select --install",
		DocURL:         sp("https://developer.apple.com/download/all/?q=command%20line%20tools"),
		Note:           sp("Provides git, make, and the compiler toolchain. Triggers a macOS install dialog — check behind other windows if you don't see it."),
	}
}

func checkBrew(path string) DepCheck {
	installed, version := versionFrom(path, "brew", "--version")
	return DepCheck{
		ID: "brew", Name: "Homebrew", Installed: installed, Version: version,
		InstallCommand: `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`,
		DocURL:         sp("https://brew.sh"),
		Note:           sp("Package manager. Installs go, node, yarn, and Docker Desktop."),
	}
}

func checkGit(path string) DepCheck {
	installed, version := versionFrom(path, "git", "--version")
	return DepCheck{
		ID: "git", Name: "git", Installed: installed, Version: version,
		InstallCommand: "brew install git",
		Note:           sp("Clones the Fleet repo and manages branches."),
	}
}

func checkGo(path string) DepCheck {
	// No version_ok: Go's toolchain manages its own version from go.mod's
	// `go` directive, so flagging a "wrong" Go version would be noise.
	installed, version := versionFrom(path, "go", "version")
	return DepCheck{
		ID: "go", Name: "Go", Installed: installed, Version: version,
		InstallCommand: "brew install go",
		Note:           sp("Builds the Fleet server."),
	}
}

type nodeManager int

const (
	managerNone nodeManager = iota
	managerNvm
	managerN
)

type versionManagerCheck struct {
	dep      DepCheck
	detected nodeManager
}

func checkNodeVersionManager(path string) versionManagerCheck {
	nvmInstalled := false
	if home, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(filepath.Join(home, ".nvm/nvm.sh")); err == nil {
			nvmInstalled = true
		}
	}
	var nvmVersion *string
	if nvmInstalled {
		if ok, out, ran := runLoginShell("nvm --version"); ran && ok {
			if v := extractVersion(out); v != "" {
				nvmVersion = &v
			}
		}
	}
	nInstalled, nVersion := versionFrom(path, "n", "--version")

	var detected nodeManager
	var version *string
	switch {
	case nvmInstalled:
		detected = managerNvm
		if nvmVersion != nil {
			version = sp("nvm " + *nvmVersion)
		}
	case nInstalled:
		detected = managerN
		if nVersion != nil {
			version = sp("n " + *nVersion)
		}
	default:
		detected = managerNone
	}

	dep := DepCheck{
		ID: "node-version-manager", Name: "nvm or n", Installed: detected != managerNone, Version: version,
		InstallCommand: "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash",
		DocURL:         sp("https://github.com/nvm-sh/nvm#install--update-script"),
		Note:           sp("Lets you install/switch Node versions. Either nvm (default) or `n` (brew install n) works."),
	}
	return versionManagerCheck{dep: dep, detected: detected}
}

func checkNode(path, required string, manager nodeManager) DepCheck {
	installed, version := versionFrom(path, "node", "-v")
	var versionOK *bool
	if version != nil && required != "" {
		versionOK = satisfies(*version, required)
	}
	// Pin to the major Fleet requires; fall back to "24".
	major := "24"
	if v := extractVersion(required); v != "" {
		if before, _, found := strings.Cut(v, "."); found {
			major = before
		}
	}
	installCmd := "nvm install " + major + " && nvm use " + major
	note := "Fleet pins a specific Node major. Use nvm or `n` to install/switch."
	if manager == managerN {
		installCmd = "n " + major
		note = "Fleet pins a specific Node major. Use `n` to install/switch."
	}
	return DepCheck{
		ID: "node", Name: "Node.js", Installed: installed, Version: version,
		Required: sp(required), VersionOK: versionOK,
		InstallCommand: installCmd, Note: sp(note),
	}
}

func checkYarn(path string) DepCheck {
	installed, version := versionFrom(path, "yarn", "-v")
	return DepCheck{
		ID: "yarn", Name: "Yarn", Installed: installed, Version: version,
		InstallCommand: "brew install yarn",
		Note:           sp("Bundles Fleet's frontend."),
	}
}

func checkDocker(path string) DepCheck {
	cliOK, version := versionFrom(path, "docker", "--version")
	var runtimeOK *bool
	if cliOK {
		// `docker version --format` hits the daemon and fails fast when it
		// isn't running (avoids docker info's long stall on a dead socket).
		ok, _, _ := run(path, "docker", "version", "--format", "{{.Server.Version}}")
		runtimeOK = &ok
	}
	var note *string
	switch {
	case !cliOK:
		note = sp("Required for `docker compose up` (MySQL/Redis dev infra).")
	case runtimeOK != nil && !*runtimeOK:
		note = sp("Installed, but the daemon isn't running. Open Docker Desktop.")
	default:
		note = sp("Runs Fleet's MySQL/Redis dev infra.")
	}
	return DepCheck{
		ID: "docker", Name: "Docker", Installed: cliOK, Version: version,
		RuntimeOK:      runtimeOK,
		InstallCommand: "brew install --cask docker",
		DocURL:         sp("https://www.docker.com/products/docker-desktop/"),
		Note:           note,
	}
}

func checkRosetta() *DepCheck {
	if runtime.GOARCH != "arm64" {
		return nil
	}
	// oahd is the Rosetta daemon; pgrep -q exits 0 iff it's running.
	installed := exec.Command("/usr/bin/pgrep", "-q", "oahd").Run() == nil
	return &DepCheck{
		ID: "rosetta", Name: "Rosetta 2", Installed: installed,
		InstallCommand: "softwareupdate --install-rosetta --agree-to-license",
		Note:           sp("Apple Silicon only. Some `make generate` tools are x86_64."),
	}
}

// CheckDependencies runs the full checklist. refreshPath re-probes the
// login-shell PATH (the Recheck button) so newly-installed tools appear.
func CheckDependencies(repoPath string, refreshPath bool) DepReport {
	reqNode := requiredNodeVersion(repoPath)
	path := shellpath.ShellPath()
	if refreshPath {
		path = shellpath.Refresh()
	}

	vm := checkNodeVersionManager(path)
	checks := []DepCheck{
		checkXcode(path),
		checkBrew(path),
		checkGit(path),
		checkGo(path),
		vm.dep,
		checkNode(path, reqNode, vm.detected),
		checkYarn(path),
		checkDocker(path),
	}
	if r := checkRosetta(); r != nil {
		checks = append(checks, *r)
	}
	return DepReport{Checks: checks}
}
