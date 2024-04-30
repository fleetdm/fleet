package installer

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

type TestOrbitClient struct {
	getHostScriptFn      func(string) (*fleet.HostScriptResult, error)
	getInstallerFn       func(string, string) (string, error)
	saveHostScriptResult func(*fleet.HostScriptResultPayload) error
}

func (oc *TestOrbitClient) GetHostScript(execID string) (*fleet.HostScriptResult, error) {
	return oc.getHostScriptFn(execID)
}

func (oc *TestOrbitClient) GetInstaller(installerID, downloadDir string) (string, error) {
	return oc.getInstallerFn(installerID, downloadDir)
}

func (oc *TestOrbitClient) SaveHostScriptResult(result *fleet.HostScriptResultPayload) error {
	return oc.saveHostScriptResult(result)
}

type TestQueryClient struct {
	queryFn func(context.Context, string) (*QueryResponse, error)
}

func (qc *TestQueryClient) Query(ctx context.Context, query string) (*QueryResponse, error) {
	return qc.queryFn(ctx, query)
}

func TestRunInstallScript(t *testing.T) {
	oc := &TestOrbitClient{}
	r := Runner{OrbitClient: oc}

	var savedHostScriptResult *fleet.HostScriptResultPayload
	oc.saveHostScriptResult = func(hsrp *fleet.HostScriptResultPayload) error {
		savedHostScriptResult = hsrp
		return nil
	}

	var executedScriptPath string
	var executed bool
	execCmd := func(ctx context.Context, spath string) ([]byte, int, error) {
		executed = true
		executedScriptPath = spath
		return []byte("bye"), 2, nil
	}
	r.execCmdFn = execCmd

	installerDir := t.TempDir()

	r.runInstallerScript(context.Background(), &fleet.HostScriptResult{
		ID:             12,
		ExecutionID:    "55",
		ScriptContents: "hello",
	}, installerDir)

	require.Equal(t, executedScriptPath, filepath.Join(installerDir, "12"))
	require.Contains(t, executedScriptPath, installerDir)
	require.True(t, executed)

	require.Equal(t, "55", savedHostScriptResult.ExecutionID)
	require.Equal(t, "bye", savedHostScriptResult.Output)
	require.Equal(t, 2, savedHostScriptResult.ExitCode)
}
