package scripts

import (
	"context"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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

type mockExecCmd struct {
	output   []byte
	exitCode int
	err      error
	count    int
}

func (m *mockExecCmd) run(ctx context.Context, scriptPath string) ([]byte, int, error) {
	m.count++
	return m.output, m.exitCode, m.err
}

type mockClient struct {
	scripts map[string]*fleet.HostScriptResult
	getErr  error
	saveErr error
}

func (m *mockClient) GetHostScript(execID string) (*fleet.HostScriptResult, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}

	script := m.scripts[execID]
	if script == nil {
		return nil, fmt.Errorf("no such script: %s", execID)
	}
	return script, nil
}

func (m *mockClient) SaveHostScriptResult(result *fleet.HostScriptResultPayload) error {
	return m.saveErr
}
