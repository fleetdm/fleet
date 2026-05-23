// Command dibble is Fleet's one-stop test data seeder.
//
// Run `dibble` with no arguments for an interactive wizard. Pass subcommands
// like `dibble users` or `dibble all` to script it.
//
// See tools/dibble/README.md and tools/dibble/dibble-plan.md for the full design.
package main

import (
	"fmt"
	"os"
)

func main() {
	rootCmd := newRootCmd()

	// No subcommand and not a help/completion request → launch the wizard.
	if shouldRunWizard(os.Args[1:]) {
		if err := runWizard(rootCmd); err != nil {
			fmt.Fprintln(os.Stderr, "dibble:", err)
			os.Exit(1)
		}
		return
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// shouldRunWizard returns true when the user invoked `dibble` with no
// subcommand. Flags-only invocations (e.g. `dibble --fleet-url X`) still
// trigger the wizard so it can fill in any missing config.
func shouldRunWizard(args []string) bool {
	for _, a := range args {
		switch a {
		case "help", "--help", "-h", "completion", "--version":
			return false
		}
		// First non-flag positional → a subcommand was given.
		if len(a) > 0 && a[0] != '-' {
			return false
		}
		// --no-wizard explicitly opts out.
		if a == "--no-wizard" {
			return false
		}
	}
	return true
}
