package fleetctl

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// TestRemovedCommands runs every removed command, with and without the flags and arguments an old invocation may still pass.
func TestRemovedCommands(t *testing.T) {
	entries := removednames.Filter(removednames.KindCommand, 0)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		t.Run(entry.Old, func(t *testing.T) {
			app := CreateApp(nil, io.Discard, io.Discard, func(*cli.Context, error) {})
			path := strings.Fields(entry.Old)
			for _, args := range [][]string{path, append(slices.Clone(path), "--json", "extra-arg")} {
				require.EqualError(t, app.Run(args), entry.Message())
			}
		})
	}
}

// runRemoved runs fleetctl with the command path of a pkg/removednames context followed by args.
func runRemoved(context string, args ...string) error {
	app := CreateApp(nil, io.Discard, io.Discard, func(*cli.Context, error) {})
	full := append(strings.Fields(context), args...)
	StashRawArgs(app, full)
	return app.Run(full)
}

// TestRemovedFlags runs every command with each removed flag, and with each removed flag value that the error spec calls out, in the
// forms an old invocation may use. These fail with the entry's message before the command does anything, so none needs a server.
func TestRemovedFlags(t *testing.T) {
	type invocation struct {
		entry       removednames.Entry
		name, value string
	}
	var cases []invocation
	for _, entry := range removednames.Filter(removednames.KindFlag, 0) {
		cases = append(cases, invocation{entry, entry.Old, "1"})
	}
	for _, entry := range removednames.Filter(removednames.KindFlagValue, 0) {
		name, value, _ := strings.Cut(strings.TrimPrefix(entry.Old, "--"), " ")
		cases = append(cases, invocation{entry, name, value})
	}
	require.NotEmpty(t, cases)

	for _, c := range cases {
		t.Run(c.entry.Context+" "+c.entry.Old, func(t *testing.T) {
			dash := "--"
			if len(c.name) == 1 { // for one-letter flags
				dash = "-"
			}
			for _, args := range [][]string{{dash + c.name, c.value}, {dash + c.name + "=" + c.value}, {"-" + c.name + "=" + c.value}} {
				require.EqualError(t, runRemoved(c.entry.Context, args...), c.entry.Message(), strings.Join(args, " "))
			}
		})
	}
}

// TestRemovedEnvVars sets each removed environment variable, which fails the command before it does anything.
func TestRemovedEnvVars(t *testing.T) {
	entries := removednames.Filter(removednames.KindEnvVar, 0)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		t.Run(entry.Context+" "+entry.Old, func(t *testing.T) {
			t.Setenv(entry.Old, "true")
			// gitops is the only command with one, and it requires -f.
			require.EqualError(t, runRemoved(entry.Context, "-f", "unused.yml"), entry.Message())
		})
	}
}

// TestRemovedFlagsLeaveOtherUsageErrorsAlone checks that a command still reports a flag it never had the usual way.
func TestRemovedFlagsLeaveOtherUsageErrorsAlone(t *testing.T) {
	app := CreateApp(nil, io.Discard, io.Discard, func(*cli.Context, error) {})
	require.EqualError(t, app.Run([]string{"fleetctl", "get", "hosts", "--not-a-flag"}), "flag provided but not defined: -not-a-flag")
}
