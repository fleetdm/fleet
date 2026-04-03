package fleetctl

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func RunAppForTest(t *testing.T, args []string) string {
	w, err := RunAppNoChecks(args)
	require.NoError(t, err)
	return w.String()
}

func RunAppCheckErr(t *testing.T, args []string, errorMsg string) string {
	w, err := RunAppNoChecks(args)
	require.Error(t, err)
	require.Contains(t, err.Error(), errorMsg)
	return w.String()
}

func RunWithErrWriter(args []string, errWriter io.Writer) (*bytes.Buffer, error) {
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := CreateApp(nil, w, errWriter, noopExitErrHandler)
	StashRawArgs(app, args)
	err := app.Run(args)
	return w, err
}

func noopExitErrHandler(c *cli.Context, err error) {}

// Alias for RunApp; added rather than changing all existing calls to `RunApp`,
// to avoid confusion and in case the behavior of `RunApp` needs to diverge in the future.
func RunAppNoChecks(args []string) (*bytes.Buffer, error) {
	return RunApp(args)
}
