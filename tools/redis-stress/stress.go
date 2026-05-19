// redis-stress: cluster-aware Redis stress tool with multiple modes.
//
// See ./README.md for full docs.
package main

import (
	"fmt"
	"os"
)

const usage = `redis-stress: cluster-aware Redis stress tool with multiple modes.

USAGE:
  redis-stress <command> [flags]

COMMANDS:
  write   Steady SET-only load (fill the cluster at a configurable rate).
  race    Tight SET-then-GET race detection (chase SET-visibility issues
          like Redis cluster replica-read lag, MOVED redirect blips, etc).

For per-command help:
  redis-stress <command> -h
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	// Backward-compat with the original (subcommand-less) tool: if the first arg
	// starts with `-`, treat it as flags for write mode. Lets old invocations
	// like `redis-stress -addr=X -wait=10m` keep working.
	if len(os.Args[1]) > 0 && os.Args[1][0] == '-' {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			fmt.Fprint(os.Stdout, usage)
			return
		}
		runWrite(os.Args[1:])
		return
	}
	switch os.Args[1] {
	case "write":
		runWrite(os.Args[2:])
	case "race":
		runRace(os.Args[2:])
	case "help":
		fmt.Fprint(os.Stdout, usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}
