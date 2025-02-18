package scripts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestRunner(t *testing.T) {
	cases := []struct {
		desc string

		// setup
		client  *mockClient
		execer  *mockExecCmd
		enabled bool
		execIDs []string

		// expected
		errContains string
		execCalls   int
	}{
		{
			desc:      "no exec ids",
			client:    &mockClient{},
			execer:    &mockExecCmd{},
			enabled:   true,
			execCalls: 0,
		},
		{
			desc:      "one exec id, success",
			client:    &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {}}},
			execer:    &mockExecCmd{},
			enabled:   true,
			execIDs:   []string{"a"},
			execCalls: 1,
		},
		{
			desc:      "one exec id disabled, success",
			client:    &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {}}},
			execer:    &mockExecCmd{},
			enabled:   false,
			execIDs:   []string{"a"},
			execCalls: 0,
		},
		{
			desc:        "one ok, one unknown",
			client:      &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {}}},
			execer:      &mockExecCmd{},
			enabled:     true,
			execIDs:     []string{"a", "b"},
			execCalls:   1,
			errContains: "no such script: b",
		},
		{
			desc:        "one ok, one unknown, disabled",
			client:      &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {}}},
			execer:      &mockExecCmd{},
			enabled:     false,
			execIDs:     []string{"a", "b"},
			execCalls:   0,
			errContains: "", // no error because when scripts are disabled, the script is not fetched (save will not update anything)
		},
		{
			desc:      "multiple, success",
			client:    &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {}, "b": {}, "c": {}}},
			execer:    &mockExecCmd{},
			enabled:   true,
			execIDs:   []string{"a", "b", "c"},
			execCalls: 3,
		},
		{
			desc:      "multiple, disabled, success",
			client:    &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {}, "b": {}, "c": {}}},
			execer:    &mockExecCmd{},
			enabled:   false,
			execIDs:   []string{"a", "b", "c"},
			execCalls: 0,
		},
		{
			desc:        "failed to get script",
			client:      &mockClient{getErr: io.ErrUnexpectedEOF, scripts: map[string]*fleet.HostScriptResult{"a": {}}},
			execer:      &mockExecCmd{},
			enabled:     true,
			execIDs:     []string{"a"},
			execCalls:   0,
			errContains: "get host script: unexpected EOF",
		},
		{
			desc:        "failed to save script",
			client:      &mockClient{saveErr: io.ErrUnexpectedEOF, scripts: map[string]*fleet.HostScriptResult{"a": {}}},
			execer:      &mockExecCmd{},
			enabled:     true,
			execIDs:     []string{"a"},
			execCalls:   1,
			errContains: "save script result: unexpected EOF",
		},
		{
			desc:        "run returns error",
			client:      &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {}}},
			execer:      &mockExecCmd{err: io.ErrUnexpectedEOF},
			enabled:     true,
			execIDs:     []string{"a"},
			execCalls:   1,
			errContains: "", // no error reported, the run error is included in the results
		},
		{
			desc:        "failed to save script, disabled",
			client:      &mockClient{saveErr: io.ErrUnexpectedEOF, scripts: map[string]*fleet.HostScriptResult{"a": {}}},
			execer:      &mockExecCmd{},
			enabled:     false,
			execIDs:     []string{"a"},
			execCalls:   0,
			errContains: "save script result: unexpected EOF",
		},
		{
			desc:        "script with existing results",
			client:      &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {ExitCode: ptr.Int64(0)}}},
			execer:      &mockExecCmd{},
			enabled:     true,
			execIDs:     []string{"a"},
			execCalls:   0,
			errContains: "", // no errors reported, script is just skipped
		},
		{
			desc:        "first script get error",
			client:      &mockClient{getErr: errFailOnce, scripts: map[string]*fleet.HostScriptResult{"a": {ExitCode: ptr.Int64(0)}}},
			execer:      &mockExecCmd{},
			enabled:     true,
			execIDs:     []string{"a", "b"},
			execCalls:   0,
			errContains: "get host script: fail once",
		},
		{
			desc:        "middle script get error",
			client:      &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {ExitCode: ptr.Int64(0)}, "b": {}, "c": {}}, erroredScripts: map[string]error{"b": errFailOnce}},
			execer:      &mockExecCmd{},
			enabled:     true,
			execIDs:     []string{"a", "b", "c"},
			execCalls:   0,
			errContains: "get host script: fail once",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			runner := &Runner{
				Client:                 c.client,
				ScriptExecutionEnabled: c.enabled,
				tempDirFn:              t.TempDir,
				execCmdFn:              c.execer.run,
			}
			err := runner.Run(c.execIDs)
			if c.errContains != "" {
				require.ErrorContains(t, err, c.errContains)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, c.execCalls, c.execer.count)
		})
	}
}

func TestRunnerTempDir(t *testing.T) {
	t.Run("deletes temp dir", func(t *testing.T) {
		tempDir := t.TempDir()

		client := &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {ScriptContents: "echo 'Hi'", ExecutionID: "a"}}}
		execer := &mockExecCmd{output: []byte("output"), exitCode: 0, err: nil}
		runner := &Runner{
			Client:                 client,
			ScriptExecutionEnabled: true,
			tempDirFn:              func() string { return tempDir },
			execCmdFn:              execer.run,
		}

		err := runner.Run([]string{"a"})
		require.NoError(t, err)
		require.Equal(t, 1, execer.count)
		require.Equal(t, "output", client.results["a"].Output)

		// ensure the temp directory was removed after execution
		entries, err := os.ReadDir(tempDir)
		require.NoError(t, err)
		require.Empty(t, entries)
	})

	t.Run("remove fails, returns original error", func(t *testing.T) {
		tempDir := t.TempDir()

		// client will fail saving the results, this is the error that should be
		// returned (i.e. the remove dir error should not override it).
		client := &mockClient{saveErr: io.ErrUnexpectedEOF, scripts: map[string]*fleet.HostScriptResult{"a": {ScriptContents: "echo 'Hi'"}}}
		execer := &mockExecCmd{output: []byte("output"), exitCode: 0, err: nil}

		runner := &Runner{
			Client:                 client,
			ScriptExecutionEnabled: true,
			tempDirFn:              func() string { return tempDir },
			execCmdFn:              execer.run,
			removeAllFn:            func(s string) error { return errors.New("remove failed") },
		}

		err := runner.Run([]string{"a"})
		require.ErrorContains(t, err, "save script result: unexpected EOF")
		require.Equal(t, 1, execer.count)
	})

	t.Run("remove fails, returns this error if the rest succeeded", func(t *testing.T) {
		tempDir := t.TempDir()

		client := &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {ScriptContents: "echo 'Hi'"}}}
		execer := &mockExecCmd{output: []byte("output"), exitCode: 0, err: nil}

		runner := &Runner{
			Client:                 client,
			ScriptExecutionEnabled: true,
			tempDirFn:              func() string { return tempDir },
			execCmdFn:              execer.run,
			removeAllFn:            func(s string) error { return errors.New("remove failed") },
		}

		err := runner.Run([]string{"a"})
		require.ErrorContains(t, err, "remove temp dir: remove failed")
		require.Equal(t, 1, execer.count)
	})

	t.Run("keeps temp dir", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("FLEET_PREVENT_SCRIPT_TEMPDIR_DELETION", "1")

		client := &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {ScriptContents: "echo 'Hi'", ExecutionID: "a"}}}
		execer := &mockExecCmd{output: []byte("output"), exitCode: 0, err: nil}
		runner := &Runner{
			Client:                 client,
			ScriptExecutionEnabled: true,
			tempDirFn:              func() string { return tempDir },
			execCmdFn:              execer.run,
		}

		err := runner.Run([]string{"a"})
		require.NoError(t, err)
		require.Equal(t, 1, execer.count)
		require.Equal(t, "output", client.results["a"].Output)

		entries, err := os.ReadDir(tempDir)
		require.NoError(t, err)
		require.Len(t, entries, 1)

		// the entry is the script's execution directory
		require.True(t, entries[0].IsDir())
		require.Contains(t, entries[0].Name(), "fleet-a-")

		runDir := filepath.Join(tempDir, entries[0].Name())
		runEntries, err := os.ReadDir(runDir)
		require.NoError(t, err)
		require.Len(t, runEntries, 1) // run directory contains the script

		b, err := os.ReadFile(filepath.Join(runDir, runEntries[0].Name()))
		require.NoError(t, err)
		require.Equal(t, "echo 'Hi'", string(b))
	})
}

func TestRunnerResults(t *testing.T) {
	output40K := strings.Repeat("a", 4000) +
		strings.Repeat("b", 4000) +
		strings.Repeat("c", 4000) +
		strings.Repeat("d", 4000) +
		strings.Repeat("e", 4000) +
		strings.Repeat("f", 4000) +
		strings.Repeat("g", 4000) +
		strings.Repeat("h", 4000) +
		strings.Repeat("i", 4000) +
		strings.Repeat("j", 4000)

	output44K := output40K + strings.Repeat("k", 4000)

	errSuffix := "\nscript execution error: " + io.ErrUnexpectedEOF.Error()

	cases := []struct {
		desc       string
		output     string
		exitCode   int
		runErr     error
		wantOutput string
	}{
		{
			desc:       "exactly the limit",
			output:     output40K,
			exitCode:   1,
			runErr:     nil,
			wantOutput: output40K,
		},
		{
			desc:       "too many bytes",
			output:     output44K,
			exitCode:   1,
			runErr:     nil,
			wantOutput: output44K[strings.Index(output44K, "b"):], //nolint:gocritic // ignore offBy1 since this is a test
		},
		{
			desc:       "empty with error",
			output:     "",
			exitCode:   -1,
			runErr:     io.ErrUnexpectedEOF,
			wantOutput: errSuffix,
		},
		{
			desc:       "limit with error",
			output:     output40K,
			exitCode:   -1,
			runErr:     io.ErrUnexpectedEOF,
			wantOutput: output40K[len(errSuffix):] + errSuffix,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			client := &mockClient{scripts: map[string]*fleet.HostScriptResult{"a": {ScriptContents: "echo 'Hi'", ExecutionID: "a"}}}
			execer := &mockExecCmd{output: []byte(c.output), exitCode: c.exitCode, err: c.runErr}
			runner := &Runner{
				Client:                 client,
				ScriptExecutionEnabled: true,
				tempDirFn:              t.TempDir,
				execCmdFn:              execer.run,
			}
			err := runner.Run([]string{"a"})
			require.NoError(t, err)
			require.Equal(t, 1, execer.count)
			require.Equal(t, c.wantOutput, client.results["a"].Output)
			require.Equal(t, c.exitCode, client.results["a"].ExitCode)
		})
	}
}

type mockExecCmd struct {
	output   []byte
	exitCode int
	err      error
	count    int
	execFn   func() ([]byte, int, error)
}

func (m *mockExecCmd) run(ctx context.Context, scriptPath string, env []string) ([]byte, int, error) {
	m.count++
	if m.execFn != nil {
		return m.execFn()
	}
	return m.output, m.exitCode, m.err
}

var errFailOnce = errors.New("fail once")

type mockClient struct {
	scripts        map[string]*fleet.HostScriptResult
	results        map[string]*fleet.HostScriptResultPayload
	getErr         error
	saveErr        error
	erroredScripts map[string]error
}

func (m *mockClient) GetHostScript(execID string) (*fleet.HostScriptResult, error) {
	if m.getErr != nil {
		err := m.getErr
		if err == errFailOnce {
			m.getErr = nil
		}
		return nil, err
	}

	if m.erroredScripts == nil {
		m.erroredScripts = make(map[string]error)
	}

	script := m.scripts[execID]
	if script == nil {
		return nil, fmt.Errorf("no such script: %s", execID)
	}

	if err, ok := m.erroredScripts[execID]; ok {
		return nil, err
	}

	return script, nil
}

func (m *mockClient) SaveHostScriptResult(result *fleet.HostScriptResultPayload) error {
	if m.results == nil {
		m.results = make(map[string]*fleet.HostScriptResultPayload)
	}
	m.results[result.ExecutionID] = result

	err := m.saveErr
	if err == errFailOnce {
		m.saveErr = nil
	}
	return err
}
