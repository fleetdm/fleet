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

// TestRemovedFlags runs every command with each of its removed flags, in the forms an old invocation may use. A removed flag fails with
// the entry's message before the command does anything, so none of these runs needs a server.
func TestRemovedFlags(t *testing.T) {
	entries := removednames.Filter(removednames.KindFlag, 0)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		t.Run(entry.Context+" "+entry.Old, func(t *testing.T) {
			app := CreateApp(nil, io.Discard, io.Discard, func(*cli.Context, error) {})
			flag := "--" + entry.Old
			if len(entry.Old) == 1 {
				flag = "-" + entry.Old
			}
			path := strings.Fields(entry.Context)
			for _, args := range [][]string{
				{flag, "1"},
				{flag + "=1"},
				{"--debug", flag, "1"},
			} {
				full := append(slices.Clone(path), args...)
				StashRawArgs(app, full)
				require.EqualError(t, app.Run(full), entry.Message())
			}
		})
	}
}

// TestRemovedFlagValues covers the values that the error spec calls out: a removed flag with one of them gets the message of the value.
func TestRemovedFlagValues(t *testing.T) {
	entries := removednames.Filter(removednames.KindFlagValue, 0)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		t.Run(entry.Context+" "+entry.Old, func(t *testing.T) {
			app := CreateApp(nil, io.Discard, io.Discard, func(*cli.Context, error) {})
			full := append(strings.Fields(entry.Context), strings.Fields(entry.Old)...)
			StashRawArgs(app, full)
			require.EqualError(t, app.Run(full), entry.Message())
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
			app := CreateApp(nil, io.Discard, io.Discard, func(*cli.Context, error) {})
			// gitops is the only command with one, and it requires -f.
			require.EqualError(t, app.Run(append(strings.Fields(entry.Context), "-f", "unused.yml")), entry.Message())
		})
	}
}

// TestRemovedFlagsLeaveOtherUsageErrorsAlone checks that a command still reports a flag it never had the usual way.
func TestRemovedFlagsLeaveOtherUsageErrorsAlone(t *testing.T) {
	app := CreateApp(nil, io.Discard, io.Discard, func(*cli.Context, error) {})
	require.EqualError(t, app.Run([]string{"fleetctl", "get", "hosts", "--not-a-flag"}), "flag provided but not defined: -not-a-flag")
}
