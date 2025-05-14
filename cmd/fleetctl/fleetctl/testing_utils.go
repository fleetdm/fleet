package fleetctl

import (
	"bytes"
	"io"
	"os"
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
	require.Equal(t, errorMsg, err.Error())
	return w.String()
}

func RunAppNoChecks(args []string) (*bytes.Buffer, error) {
	// first arg must be the binary name. Allow tests to omit it.
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := CreateApp(nil, w, os.Stderr, noopExitErrHandler)
	err := app.Run(args)
	return w, err
}

func RunWithErrWriter(args []string, errWriter io.Writer) (*bytes.Buffer, error) {
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := CreateApp(nil, w, errWriter, noopExitErrHandler)
	err := app.Run(args)
	return w, err
}

func noopExitErrHandler(c *cli.Context, err error) {}
