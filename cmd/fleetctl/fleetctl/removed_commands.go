package fleetctl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/urfave/cli/v2"
)

// registerRemovedCommands adds a hidden command for each command in pkg/removednames that fails with the entry's message. The change
// that adds an entry removes the old command or alias, because urfave/cli fails every subcommand of a parent with a duplicated name.
func registerRemovedCommands(app *cli.App) {
	for _, entry := range removednames.Filter(removednames.KindCommand, 0) {
		path := strings.Fields(entry.Old)
		if len(path) < 2 || path[0] != app.Name {
			panic(fmt.Sprintf("removed command %q does not start with %q", entry.Old, app.Name))
		}
		siblings := &app.Commands
		for _, name := range path[1 : len(path)-1] {
			parent := findCommand(*siblings, name)
			if parent == nil {
				panic(fmt.Sprintf("removed command %q: no %q command to add it to", entry.Old, name))
			}
			siblings = &parent.Subcommands
		}
		name := path[len(path)-1]
		if findCommand(*siblings, name) != nil {
			panic(fmt.Sprintf("removed command %q is still served", entry.Old))
		}
		*siblings = append(*siblings, &cli.Command{
			Name:   name,
			Hidden: true,
			// Old invocations may pass flags that this command does not define, which would hide the message behind a flag error.
			SkipFlagParsing: true,
			Action: func(*cli.Context) error {
				return errors.New(entry.Message())
			},
		})
	}
}

func findCommand(cmds []*cli.Command, name string) *cli.Command {
	for _, cmd := range cmds {
		if cmd.HasName(name) {
			return cmd
		}
	}
	return nil
}
