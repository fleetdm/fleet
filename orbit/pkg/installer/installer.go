package installer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/osquery/osquery-go"
	osquery_gen "github.com/osquery/osquery-go/gen/osquery"
)

type QueryResponse = osquery_gen.ExtensionResponse

// Client defines the methods required for the API requests to the server. The
// fleet.OrbitClient type satisfies this interface.
type Client interface {
	GetHostScript(execID string) (*fleet.HostScriptResult, error)
	DownloadSoftwareInstaller(installerID uint, downloadDir string) (string, error)
	GetInstallerDetails(installId string) (*fleet.SoftwareInstallDetails, error)
	SaveInstallerResult(payload *fleet.HostSoftwareInstallResultPayload) error
}

type QueryClient interface {
	Query(context.Context, string) (*QueryResponse, error)
}

type Runner struct {
	OsqueryClient QueryClient
	OrbitClient   Client

	// tempDirFn is the function to call to get the temporary directory to use,
	// inside of which the script-specific subdirectories will be created. If nil,
	// the user's temp dir will be used (can be set to t.TempDir in tests).
	tempDirFn func() string

	// execCmdFn can be set for tests to mock actual execution of the script. If
	// nil, execCmd will be used, which has a different implementation on Windows
	// and non-Windows platforms.
	execCmdFn func(ctx context.Context, scriptPath string, env []string) ([]byte, int, error)

	// can be set for tests to replace os.RemoveAll, which is called to remove
	// the script's temporary directory after execution.
	removeAllFn func(string) error
}

func NewRunner(client Client, socketPath string, timeout time.Duration) (*Runner, error) {
	r := &Runner{
		OrbitClient: client,
	}

	osqueryClient, err := osquery.NewClient(socketPath, timeout)
	if err != nil {
		return nil, fmt.Errorf("creating new osquery client: %w", err)
	}

	r.OsqueryClient = osqueryClient.Client

	return r, nil
}

func (r *Runner) run(ctx context.Context, config *fleet.OrbitConfig) error {
	var errs []error
	for _, installerID := range config.Notifications.PendingSoftwareInstallerIDs {
		if ctx.Err() == nil {
			payload, err := r.installSoftware(ctx, installerID)
			if err != nil {
				errs = append(errs, err)
			}
			if err := r.OrbitClient.SaveInstallerResult(payload); err != nil {
				return ctxerr.Wrap(ctx, err, "saving software install results")
			}
		}
	}
	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (r *Runner) preConditionCheck(ctx context.Context, query string) (bool, string, error) {
	res, err := r.OsqueryClient.Query(ctx, query)
	if err != nil {
		return false, "", ctxerr.Wrap(ctx, err, "precondition check")
	}

	if res.Status == nil {
		return false, "", errors.New("no query status")
	}

	if res.Status.Code != 0 {
		return false, res.String(), ctxerr.Wrap(ctx, fmt.Errorf("non-zero query status: %d \"%s\"", res.Status.Code, res.Status.Message))
	}

	if len(res.Response) == 0 {
		return false, res.String(), nil
	}

	return true, res.String(), nil
}

func (r *Runner) installSoftware(ctx context.Context, installId string) (*fleet.HostSoftwareInstallResultPayload, error) {
	installer, err := r.OrbitClient.GetInstallerDetails(installId)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching software installer details")
	}

	payload := &fleet.HostSoftwareInstallResultPayload{}

	payload.InstallUUID = installId

	shouldInstall, output, err := r.preConditionCheck(ctx, installer.PreInstallCondition)
	payload.PreInstallConditionOutput = &output
	if err != nil {
		return payload, err
	}

	if !shouldInstall {
		return payload, nil
	}

	installScript, err := r.OrbitClient.GetHostScript(installer.InstallScript)
	if err != nil {
		return payload, err
	}

	postInstallScript, err := r.OrbitClient.GetHostScript(installer.PostInstallScript)
	if err != nil {
		return payload, err
	}

	if r.tempDirFn == nil {
		r.tempDirFn = os.TempDir
	}
	tmpDir := r.tempDirFn()

	installerPath, err := r.OrbitClient.DownloadSoftwareInstaller(installer.InstallerID, tmpDir)
	if err != nil {
		return payload, err
	}

	// remove tmp directory and installer
	defer func() {
		if r.removeAllFn == nil {
			r.removeAllFn = os.RemoveAll
		}
		r.removeAllFn(tmpDir)
	}()

	installOutput, installExitCode, err := r.runInstallerScript(ctx, installScript, installerPath)
	payload.InstallScriptOutput = &installOutput
	payload.InstallScriptExitCode = &installExitCode
	if err != nil {
		return payload, err
	}

	postOutput, postExitCode, err := r.runInstallerScript(ctx, postInstallScript, installerPath)
	payload.PostInstallScriptOutput = &postOutput
	payload.PostInstallScriptExitCode = &postExitCode
	if err != nil {
		return payload, err
	}

	return payload, nil
}

func (r *Runner) runInstallerScript(ctx context.Context, script *fleet.HostScriptResult, installerPath string) (string, int, error) {
	// run script in installer directory
	scriptPath := filepath.Join(installerPath, strconv.Itoa(int(script.ID)))
	if err := os.WriteFile(scriptPath, []byte(script.ScriptContents), 0700); err != nil {
		return "", -1, ctxerr.Wrap(ctx, err, "writing script")
	}

	if r.execCmdFn == nil {
		r.execCmdFn = scripts.ExecCmd
	}

	env := os.Environ()
	installerPathEnv := fmt.Sprintf("INSTALLER_PATH=%s", scriptPath)
	env = append(env, installerPathEnv)

	output, exitCode, err := r.execCmdFn(ctx, scriptPath, env)
	if err != nil {
		return string(output), exitCode, err
	}

	return string(output), exitCode, nil
}
