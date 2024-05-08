package installer

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	osquery_gen "github.com/osquery/osquery-go/gen/osquery"
	"github.com/stretchr/testify/require"
)

type TestOrbitClient struct {
	getHostScriptFn       func(string) (*fleet.HostScriptResult, error)
	downloadInstallerFn   func(uint, string) (string, error)
	getInstallerDetailsFn func(string) (*fleet.SoftwareInstallDetails, error)
	saveInstallerResultFn func(*fleet.HostSoftwareInstallResultPayload) error
}

func (oc *TestOrbitClient) GetHostScript(execID string) (*fleet.HostScriptResult, error) {
	return oc.getHostScriptFn(execID)
}

func (oc *TestOrbitClient) DownloadSoftwareInstaller(installerID uint, downloadDir string) (string, error) {
	return oc.downloadInstallerFn(installerID, downloadDir)
}

func (oc *TestOrbitClient) GetInstallerDetails(installId string) (*fleet.SoftwareInstallDetails, error) {
	return oc.getInstallerDetailsFn(installId)
}

func (oc *TestOrbitClient) SaveInstallerResult(payload *fleet.HostSoftwareInstallResultPayload) error {
	return oc.saveInstallerResultFn(payload)
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

	var executedScriptPath string
	var executed bool
	var executedEnv []string
	execCmd := func(ctx context.Context, spath string, env []string) ([]byte, int, error) {
		executed = true
		executedScriptPath = spath
		executedEnv = env
		return []byte("bye"), 2, nil
	}
	r.execCmdFn = execCmd

	installerDir := t.TempDir()
	installerPath := filepath.Join(installerDir, "installer.pkg")

	output, exitCode, err := r.runInstallerScript(context.Background(), &fleet.HostScriptResult{
		ID:             12,
		ExecutionID:    "55",
		ScriptContents: "hello",
	}, installerPath)

	require.Equal(t, executedScriptPath, filepath.Join(installerDir, "12"))
	require.Contains(t, executedScriptPath, installerDir)
	require.True(t, executed)

	require.Nil(t, err)
	require.Equal(t, "bye", output)
	require.Equal(t, 2, exitCode)
	require.Contains(t, executedEnv, "INSTALLER_PATH="+installerPath)
}

func TestPreconditionCheck(t *testing.T) {
	qc := &TestQueryClient{}
	r := &Runner{OsqueryClient: qc}

	qc.queryFn = func(ctx context.Context, s string) (*QueryResponse, error) {
		qr := &QueryResponse{
			Status: &osquery_gen.ExtensionStatus{},
		}

		switch s {
		case "empty":
		case "error":
			return nil, errors.New("something bad")
		case "badstatus":
			qr.Status.Code = 1
			qr.Status.Message = "something bad"
		case "nostatus":
			qr.Status = nil
		case "response":
			row := make(map[string]string)
			row["key"] = "value"
			qr.Response = append(qr.Response, row)
		default:
			t.Error("invalid query test case")
		}

		return qr, nil
	}

	ctx := context.Background()

	// empty query response
	success, output, err := r.preConditionCheck(ctx, "empty")
	require.False(t, success)
	require.Nil(t, err)
	require.Equal(t, "null", output)

	success, output, err = r.preConditionCheck(ctx, "response")
	require.True(t, success)
	require.Nil(t, err)
	require.Equal(t, "[{\"key\":\"value\"}]", output)

	success, output, err = r.preConditionCheck(ctx, "error")
	require.False(t, success)
	require.Error(t, err)
	require.Equal(t, "", output)

	success, output, err = r.preConditionCheck(ctx, "badstatus")
	require.False(t, success)
	require.Error(t, err)
	require.Equal(t, "null", output)

	success, output, err = r.preConditionCheck(ctx, "nostatus")
	require.False(t, success)
	require.Error(t, err)
	require.Equal(t, "", output)
}

func TestInstallerRun(t *testing.T) {
	oc := &TestOrbitClient{}

	var getHostScriptFnCalled bool
	oc.getHostScriptFn = func(execID string) (*fleet.HostScriptResult, error) {
		getHostScriptFnCalled = true
		return &fleet.HostScriptResult{
			ScriptContents: execID,
		}, nil
	}

	var getInstallerDetailsFnCalled bool
	installDetails := &fleet.SoftwareInstallDetails{
		ExecutionID:         "exec1",
		InstallerID:         1337,
		PreInstallCondition: "SELECT 1",
		InstallScript:       "script1",
		PostInstallScript:   "script2",
	}
	oc.getInstallerDetailsFn = func(installID string) (*fleet.SoftwareInstallDetails, error) {
		getInstallerDetailsFnCalled = true
		return installDetails
	}

	var downloadInstallerFnCalled bool
	oc.downloadInstallerFn = func(installerID uint, downloadDir string) (string, error) {
		downloadInstallerFnCalled = true
		return filepath.Join(downloadDir, strconv.Itoa(int(installerID))+".pkg"), nil
	}

	var savedInstallerResult *fleet.HostSoftwareInstallResultPayload
	oc.saveInstallerResultFn = func(hsirp *fleet.HostSoftwareInstallResultPayload) error {
		savedInstallerResult = hsirp
		return nil
	}

	r := &Runner{OrbitClient: oc}

	var execCalled bool
	var executedScripts []string
	r.execCmdFn = func(ctx context.Context, scriptPath string, env []string) ([]byte, int, error) {
		execCalled = true
		executedScripts = append(executedScripts, scriptPath)
		return []byte("execOutput"), 0, nil
	}

	var tmpDirFnCalled bool
	r.tempDirFn = func() string {
		tmpDirFnCalled = true
		return "/tmp/installer"
	}

	var removeAllFnCalled bool
	var removedDir string
	r.removeAllFn = func(s string) error {
		removedDir = s
		removeAllFnCalled = true
		return nil
	}
}
