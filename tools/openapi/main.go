// Command openapi generates Fleet's pilot OpenAPI 3.1 spec from the canonical
// REST API Markdown, and verifies it against a live server.
// See DESIGN.md and https://github.com/fleetdm/fleet/issues/45279.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/tools/openapi/parser"
	"github.com/fleetdm/fleet/tools/openapi/spec"
)

func main() {
	args := os.Args[1:]
	cmd := "generate"
	if len(args) > 0 && !isFlag(args[0]) {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "generate":
		os.Exit(runGenerate(args))
	case "verify":
		os.Exit(runVerify(args))
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (want generate or verify)\n", cmd)
		os.Exit(1)
	}
}

func isFlag(s string) bool { return len(s) > 0 && s[0] == '-' }

func runGenerate(args []string) int {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	markdown := fs.String("markdown", "../../docs/REST API/rest-api.md", "path to rest-api.md")
	out := fs.String("out", "openapi.yml", "output path")
	allowPath := fs.String("allowlist", "allowlist.yml", "allowlist path")
	fs.Parse(args)

	md, err := os.ReadFile(*markdown)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	allow, err := spec.LoadAllowlist(*allowPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	res := parser.Parse(string(md))
	total := len(res.Endpoints) + len(res.Skipped)
	fmt.Fprintf(os.Stderr, "coverage: parsed %d/%d endpoint sections\n", len(res.Endpoints), total)
	for _, s := range res.Skipped {
		fmt.Fprintf(os.Stderr, "  skipped %q: %s\n", s.Heading, s.Reason)
	}

	doc, err := spec.Build(res, allow)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	rendered, err := doc.Render()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if err := spec.Validate(rendered); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}

	if err := os.WriteFile(*out, rendered, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "wrote %d endpoint(s) to %s\n", len(allow.Endpoints), *out)
	return 0
}
