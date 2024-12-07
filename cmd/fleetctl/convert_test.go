package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestConvertFileOutput(t *testing.T) {
	// setup the cli and the convert command
	app := cli.NewApp()
	app.Commands = []*cli.Command{convertCommand()}
	app.Reader = os.Stdin
	app.Writer = os.Stdout
	app.Setup()

	// read the expected output file
	expected, err := os.ReadFile(filepath.Join("testdata", "convert_output.yml"))
	require.NoError(t, err)

	// setup a file for the convert command to write to
	file, err := os.CreateTemp(t.TempDir(), "convert_output.yml")
	require.NoError(t, err)
	defer file.Close()

	// get the program name
	args := os.Args[0:1]
	args = append(args, []string{"convert", "-f", filepath.Join("testdata", "convert_input.conf"), "-o", file.Name()}...)
	err = app.Run(args)
	require.NoError(t, err)

	// convert command ran and wrote converted file to output destination
	got, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	require.YAMLEq(t, string(expected), string(got))
}

func TestConvertFileStdout(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	// setup the cli and the convert command
	app := cli.NewApp()
	app.Commands = []*cli.Command{convertCommand()}
	app.Reader = os.Stdin
	app.Writer = os.Stdout
	app.Setup()

	// read the expected output file
	expected, err := os.ReadFile(filepath.Join("testdata", "convert_output.yml"))
	require.NoError(t, err)

	// get the program name
	args := os.Args[0:1]
	args = append(args, []string{"convert", "-f", filepath.Join("testdata", "convert_input.conf")}...)
	err = app.Run(args)
	require.NoError(t, err)

	os.Stdout = oldStdout
	w.Close()
	out, _ := io.ReadAll(r)
	require.YAMLEq(t, string(expected), string(out))
}
