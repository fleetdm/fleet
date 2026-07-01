// Command dibble is Fleet's one-stop test data seeder.
//
// Run `dibble` with no arguments for an interactive wizard. Pass subcommands
// like `dibble users` or `dibble all` to script it.
//
// See tools/dibble/README.md for the full design.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/command"
)

func main() {
	if err := command.Execute(os.Args[1:]); err != nil {
		// reportErrors already wrote each seeder error to stderr; only
		// surface errors from other paths (config, network, etc.) here.
		if !errors.Is(err, command.ErrSeederFailed) {
			fmt.Fprintln(os.Stderr, "dibble:", err)
		}
		os.Exit(1)
	}
}
