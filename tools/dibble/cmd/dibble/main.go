// Command dibble is Fleet's one-stop test data seeder.
//
// Run `dibble` with no arguments for an interactive wizard. Pass subcommands
// like `dibble users` or `dibble all` to script it.
//
// See tools/dibble/README.md for the full design.
package main

import (
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/command"
)

func main() {
	if err := command.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "dibble:", err)
		os.Exit(1)
	}
}
