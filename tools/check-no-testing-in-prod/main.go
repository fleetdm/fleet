// Command check-no-testing-in-prod fails the build if any Fleet-owned (i.e.
// github.com/fleetdm/fleet/v4/...) package reachable from cmd/fleet,
// cmd/fleetctl, or orbit/cmd/orbit directly imports Go's "testing"
// package.
//
// Background: see https://github.com/fleetdm/fleet/issues/45220. Test
// helpers used to be sprinkled across production packages as
// testing_utils.go files that imported "testing"; that pulled the test
// scaffolding (and its -test.* flags, etc.) into the shipping binaries.
// The cleanup moved each such helper into a sibling *test subpackage or
// _test.go file. This guard makes sure nothing slips back.
//
// Usage:
//
//	go run ./tools/check-no-testing-in-prod
//
// Wired into make lint-go (see Makefile).
//
// Note: third-party dependencies (currently a few pkcs7 forks and
// apache/thrift) are intentionally NOT checked here -- they're out of our
// control without an upstream swap. The intent of this check is to make
// sure Fleet code never reintroduces the issue.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
)

// Production binaries to audit. Any first-party package reachable from
// these must not import "testing" directly.
//
// Important: these are bare binary paths, NOT "./cmd/fleet/..." — the
// "..." form expands to include _test.go-only sibling packages that aren't
// actually linked into the binary.
var roots = []string{
	"./cmd/fleet",
	"./cmd/fleetctl",
	"./orbit/cmd/orbit",
}

const fleetModulePrefix = "github.com/fleetdm/fleet/v4/"

type pkgInfo struct {
	ImportPath string   `json:"ImportPath"`
	Imports    []string `json:"Imports"`
	Standard   bool     `json:"Standard"`
}

func main() {
	// Use the same build tags as the production build (see Makefile).
	args := []string{
		"list",
		"-tags", "full,fts5,netgo",
		"-deps",
		"-json",
	}
	args = append(args, roots...)

	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "go list failed: %v\n", err)
		os.Exit(2)
	}

	dec := json.NewDecoder(strings.NewReader(string(out)))
	var offenders []string
	seen := make(map[string]struct{})
	for {
		var p pkgInfo
		if err := dec.Decode(&p); err != nil {
			break
		}
		if p.Standard {
			continue
		}
		if !strings.HasPrefix(p.ImportPath, fleetModulePrefix) {
			continue
		}
		if slices.Contains(p.Imports, "testing") {
			if _, ok := seen[p.ImportPath]; !ok {
				offenders = append(offenders, p.ImportPath)
				seen[p.ImportPath] = struct{}{}
			}
		}
	}

	if len(offenders) == 0 {
		return
	}

	sort.Strings(offenders)
	fmt.Fprintln(os.Stderr, "check-no-testing-in-prod: the following Fleet packages are reachable from")
	fmt.Fprintln(os.Stderr, "the production binaries (cmd/fleet, cmd/fleetctl, orbit/cmd/orbit) AND import")
	fmt.Fprintln(os.Stderr, "the \"testing\" package directly. Move the test-only code into a sibling")
	fmt.Fprintln(os.Stderr, "*test subpackage or a _test.go file. See https://github.com/fleetdm/fleet/issues/45220.")
	fmt.Fprintln(os.Stderr)
	for _, o := range offenders {
		fmt.Fprintf(os.Stderr, "  - %s\n", o)
	}
	os.Exit(1)
}
