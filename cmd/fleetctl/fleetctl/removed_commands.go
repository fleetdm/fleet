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
		siblings := commandsAtPath(app, path[:len(path)-1], entry.Old)
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

// commandsAtPath returns the subcommands of the command at path (the app's own commands for a path of just the app name), and panics when
// the path does not lead to one. A path is a pkg/removednames context or command, such as "fleetctl get hosts". The panics are for
// developers: they depend only on the static table and the command tree, so any test that builds the app catches them before release.
func commandsAtPath(app *cli.App, path []string, context string) *[]*cli.Command {
	if len(path) == 0 || path[0] != app.Name {
		panic(fmt.Sprintf("removed name %q does not start with %q", context, app.Name))
	}
	commands := &app.Commands
	for _, name := range path[1:] {
		cmd := findCommand(*commands, name)
		if cmd == nil {
			panic(fmt.Sprintf("removed name %q: no %q command", context, name))
		}
		commands = &cmd.Subcommands
	}
	return commands
}
