package fleetctl

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/urfave/cli/v2"
)

// undefinedFlagError is how urfave/cli (through the flag package) reports a flag that the command does not define.
var undefinedFlagError = regexp.MustCompile(`^flag provided but not defined: -{1,2}(.+)$`)

// registerRemovedFlags makes each removed flag and environment variable in pkg/removednames fail with the entry's message, before the
// command does anything. The change that adds an entry removes the flag, its alias, or its environment variable from the command, so
// that a command still defining one panics here.
//
// A removed flag is not defined: urfave/cli rejects it while parsing, before any command code runs, so the command's OnUsageError
// turns that error into the message. An environment variable is checked in the command's Before.
func registerRemovedFlags(app *cli.App) {
	byContext := make(map[string][]removednames.Entry)
	for _, kind := range []removednames.Kind{removednames.KindFlag, removednames.KindEnvVar} {
		for _, entry := range removednames.Filter(kind, 0) {
			byContext[entry.Context] = append(byContext[entry.Context], entry)
		}
	}

	for context, entries := range byContext {
		path := strings.Fields(context)
		cmd := findCommand(*commandsAtPath(app, path[:len(path)-1], context), path[len(path)-1])
		if cmd == nil {
			panic(fmt.Sprintf("removed flag context %q: no %q command", context, path[len(path)-1]))
		}
		if cmd.OnUsageError != nil {
			panic(fmt.Sprintf("%q defines OnUsageError, which removed flags would replace", context))
		}
		var removedFlags, removedEnvVars []removednames.Entry
		for _, entry := range entries {
			switch entry.Kind {
			case removednames.KindFlag:
				if flagDefined(cmd, entry.Old) {
					panic(fmt.Sprintf("removed flag %q is still defined by %q", entry.Old, context))
				}
				removedFlags = append(removedFlags, entry)
			case removednames.KindEnvVar:
				if slices.ContainsFunc(cmd.Flags, func(f cli.Flag) bool { return slices.Contains(envVarsOf(f), entry.Old) }) {
					panic(fmt.Sprintf("removed environment variable %q is still read by %q", entry.Old, context))
				}
				removedEnvVars = append(removedEnvVars, entry)
			}
		}
		previousBefore := cmd.Before
		cmd.Before = func(c *cli.Context) error {
			for _, entry := range removedEnvVars {
				if _, ok := os.LookupEnv(entry.Old); ok {
					return errors.New(entry.Message())
				}
			}
			if previousBefore != nil {
				return previousBefore(c)
			}
			return nil
		}
		cmd.OnUsageError = func(c *cli.Context, err error, _ bool) error {
			if match := undefinedFlagError.FindStringSubmatch(err.Error()); match != nil {
				if i := slices.IndexFunc(removedFlags, func(entry removednames.Entry) bool { return entry.Old == match[1] }); i >= 0 {
					// A value that the error spec calls out for this flag ("--team no-team") gets its own message.
					if value := rawFlagValue(c, match[1]); value != "" {
						if entry, ok := removednames.Find(removednames.KindFlagValue, "--"+match[1]+" "+value, context); ok {
							return errors.New(entry.Message())
						}
					}
					return errors.New(removedFlags[i].Message())
				}
			}
			// What urfave/cli does when a command has no OnUsageError.
			_, _ = fmt.Fprintf(c.App.Writer, "Incorrect Usage: %s\n\n", err.Error())
			if lineage := c.Lineage(); len(lineage) > 1 {
				_ = cli.ShowCommandHelp(lineage[1], c.Command.Name)
			}
			return err
		}
	}
}

func flagDefined(cmd *cli.Command, name string) bool {
	return slices.ContainsFunc(cmd.Flags, func(f cli.Flag) bool { return slices.Contains(f.Names(), name) })
}

// envVarsOf returns the environment variables a flag reads, for the flag types fleetctl uses.
func envVarsOf(f cli.Flag) []string {
	if withEnv, ok := f.(interface{ GetEnvVars() []string }); ok {
		return withEnv.GetEnvVars()
	}
	return nil
}

// rawFlagValue returns the value of "--name", "-name", "--name=value" or "-name=value" in the invocation, or "" when there is none.
// urfave/cli drops the parsed arguments when parsing fails, which is when this is needed.
func rawFlagValue(c *cli.Context, name string) string {
	args := rawArgs(c)
	for i, arg := range args {
		for _, flag := range []string{"--" + name, "-" + name} {
			if arg == flag && i+1 < len(args) {
				return args[i+1]
			}
			if value, ok := strings.CutPrefix(arg, flag+"="); ok {
				return value
			}
		}
	}
	return ""
}
