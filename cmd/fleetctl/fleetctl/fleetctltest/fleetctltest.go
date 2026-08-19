// Package fleetctltest provides test helpers that drive fleetctl's CLI app
// in-process so tests can capture stdout/stderr and assert on its output.
//
// It imports the "testing" package and must therefore only ever be imported
// from test code; importing it from production code would pull "testing"
// into the resulting binary.
package fleetctltest

import (
	"bytes"
	"io"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// RunAppForTest runs the fleetctl CLI app with the given args and asserts it
// completes without error, returning the captured stdout.
func RunAppForTest(t *testing.T, args []string) string {
	w, err := RunAppNoChecks(args)
	require.NoError(t, err)
	return w.String()
}

// RunAppCheckErr runs the fleetctl CLI app with the given args and asserts
// it returns an error whose message contains errorMsg.
func RunAppCheckErr(t *testing.T, args []string, errorMsg string) string {
	w, err := RunAppNoChecks(args)
	require.Error(t, err)
	require.Contains(t, err.Error(), errorMsg)
	return w.String()
}

// RunWithErrWriter runs the fleetctl CLI app with the given args, sending
// stderr to errWriter and returning the captured stdout and any error.
func RunWithErrWriter(args []string, errWriter io.Writer) (*bytes.Buffer, error) {
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := fleetctl.CreateApp(nil, w, errWriter, noopExitErrHandler)
	fleetctl.StashRawArgs(app, args)
	err := app.Run(args)
	return w, err
}

func noopExitErrHandler(_ *cli.Context, _ error) {}

// RunAppNoChecks is an alias for fleetctl.RunApp; it captures stdout and
// returns the result for tests that want to inspect it without asserting
// pass/fail.
func RunAppNoChecks(args []string) (*bytes.Buffer, error) {
	return fleetctl.RunApp(args)
}
