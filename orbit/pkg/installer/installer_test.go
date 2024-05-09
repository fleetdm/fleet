package installer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func (oc *TestOrbitClient) GetHostScript(scriptID string) (*fleet.HostScriptResult, error) {
	return oc.getHostScriptFn(scriptID)
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
	var hostScriptsRequested []string
	oc.getHostScriptFn = func(scriptID string) (*fleet.HostScriptResult, error) {
		getHostScriptFnCalled = true
		hostScriptsRequested = append(hostScriptsRequested, scriptID)
		id, err := strconv.Atoi(strings.ReplaceAll(scriptID, "script", ""))
		require.NoError(t, err, "test internal scriptID (string) to script id (int)")
		return &fleet.HostScriptResult{
			ID:             uint(id),
			ScriptContents: scriptID,
		}, nil
	}

	var getInstallerDetailsFnCalled bool
	var installIdRequested string
	installDetails := &fleet.SoftwareInstallDetails{
		ExecutionID:         "exec1",
		InstallerID:         1337,
		PreInstallCondition: "SELECT 1",
		InstallScript:       "script1",
		PostInstallScript:   "script2",
	}
	oc.getInstallerDetailsFn = func(installID string) (*fleet.SoftwareInstallDetails, error) {
		getInstallerDetailsFnCalled = true
		installIdRequested = installID
		return installDetails, nil
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

	q := &TestQueryClient{}

	var queryFnCalled bool
	var queryFnQuery string
	queryFnResMap := make(map[string]string, 0)
	queryFnResArr := []map[string]string{queryFnResMap}
	queryFnResStatus := &QueryResponseStatus{}
	queryFnResponse := &QueryResponse{
		Response: queryFnResArr,
		Status:   queryFnResStatus,
	}
	q.queryFn = func(ctx context.Context, query string) (*QueryResponse, error) {
		queryFnQuery = query
		queryFnCalled = true
		queryFnResMap["col"] = "true"
		return queryFnResponse, nil
	}

	r := &Runner{
		OrbitClient:   oc,
		OsqueryClient: q,
	}

	var execCalled bool
	var executedScripts []string
	var execEnv []string
	execOutput := []byte("execOutput")
	execExitCode := 0
	r.execCmdFn = func(ctx context.Context, scriptPath string, env []string) ([]byte, int, error) {
		execCalled = true
		execEnv = env
		executedScripts = append(executedScripts, scriptPath)
		return execOutput, execExitCode, nil
	}

	var tmpDirFnCalled bool
	var tmpDir string
	r.tempDirFn = func() string {
		tmpDirFnCalled = true
		tmpDir = os.TempDir()
		return tmpDir
	}

	var removeAllFnCalled bool
	var removedDir string
	r.removeAllFn = func(s string) error {
		removedDir = s
		removeAllFnCalled = true
		return nil
	}

	var config fleet.OrbitConfig
	config.Notifications.PendingSoftwareInstallerIDs = []string{"exec1"}

	err := r.run(context.Background(), &config)
	require.NoError(t, err)

	require.True(t, removeAllFnCalled)
	require.Equal(t, tmpDir, removedDir)

	require.True(t, tmpDirFnCalled)

	require.True(t, execCalled)
	require.Contains(t, executedScripts, filepath.Join(tmpDir, "1"))
	require.Contains(t, executedScripts, filepath.Join(tmpDir, "2"))
	require.Contains(t, execEnv, "INSTALLER_PATH="+filepath.Join(tmpDir, strconv.Itoa(int(installDetails.InstallerID))+".pkg"))

	require.True(t, queryFnCalled)
	require.Equal(t, installDetails.PreInstallCondition, queryFnQuery)

	require.NotNil(t, savedInstallerResult)
	require.Equal(t, execExitCode, *savedInstallerResult.InstallScriptExitCode)
	require.Equal(t, string(execOutput), *savedInstallerResult.InstallScriptOutput)
	require.Equal(t, execExitCode, *savedInstallerResult.PostInstallScriptExitCode)
	require.Equal(t, string(execOutput), *savedInstallerResult.PostInstallScriptOutput)
	require.Equal(t, installDetails.ExecutionID, savedInstallerResult.InstallUUID)

	require.True(t, downloadInstallerFnCalled)

	require.True(t, getInstallerDetailsFnCalled)
	require.Equal(t, installDetails.ExecutionID, installIdRequested)

	require.True(t, getHostScriptFnCalled)
	require.Contains(t, hostScriptsRequested, installDetails.InstallScript)
	require.Contains(t, hostScriptsRequested, installDetails.PostInstallScript)
}
