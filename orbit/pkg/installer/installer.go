package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/osquery/osquery-go"
	osquery_gen "github.com/osquery/osquery-go/gen/osquery"
	"github.com/rs/zerolog/log"
)

type QueryResponse = osquery_gen.ExtensionResponse
type QueryResponseStatus = osquery_gen.ExtensionStatus

// Client defines the methods required for the API requests to the server. The
// fleet.OrbitClient type satisfies this interface.
type Client interface {
	GetHostScript(execID string) (*fleet.HostScriptResult, error)
	GetInstallerDetails(installId string) (*fleet.SoftwareInstallDetails, error)
	DownloadSoftwareInstaller(installerID uint, downloadDir string) (string, error)
	SaveInstallerResult(payload *fleet.HostSoftwareInstallResultPayload) error
}

type QueryClient interface {
	Query(context.Context, string) (*QueryResponse, error)
}

type Runner struct {
	OsqueryClient QueryClient
	OrbitClient   Client

	// osquerySocketPaht is used to establish the osquery connection
	// if it's ever lost or disconnected
	osquerySocketPaht string

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

	connectOsquery func(*Runner) error
}

func NewRunner(client Client, socketPath string, queryTimeout time.Duration) (*Runner, error) {
	r := &Runner{
		OrbitClient:       client,
		osquerySocketPaht: socketPath,
	}

	return r, nil
}

func (r *Runner) Run(config *fleet.OrbitConfig) error {
	if r.connectOsquery == nil {
		r.connectOsquery = connectOsquery
	}

	if err := r.connectOsquery(r); err != nil {
		return fmt.Errorf("software installer runner connecting to osquery: %w", err)
	}
	return r.run(context.Background(), config)
}

func connectOsquery(r *Runner) error {
	if r.OsqueryClient == nil {
		osqueryClient, err := osquery.NewClient(r.osquerySocketPaht, 2*time.Second)
		if err != nil {
			log.Err(err).Msg("establishing osquery connection for software install runner")
			return err
		}

		r.OsqueryClient = osqueryClient.Client
	}

	return nil
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
				errs = append(errs, fmt.Errorf("saving software install results: %w", err))
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
		return false, "", fmt.Errorf("precondition check: %w", err)
	}

	if res.Status == nil {
		return false, "", errors.New("no query status")
	}

	response, err := json.Marshal(res.Response)
	if err != nil {
		return false, "", fmt.Errorf("marshalling query response: %w", err)
	}

	if res.Status.Code != 0 {
		return false, string(response), fmt.Errorf("non-zero query status: %d \"%s\"", res.Status.Code, res.Status.Message)
	}

	if len(res.Response) == 0 {
		return false, string(response), nil
	}

	return true, string(response), nil
}

func (r *Runner) installSoftware(ctx context.Context, installId string) (*fleet.HostSoftwareInstallResultPayload, error) {
	installer, err := r.OrbitClient.GetInstallerDetails(installId)
	if err != nil {
		return nil, fmt.Errorf("fetching software installer details: %w", err)
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
		err := r.removeAllFn(tmpDir)
		if err != nil {
			log.Err(err)
		}
	}()

	installOutput, installExitCode, err := r.runInstallerScript(ctx, installer.InstallScript, installerPath)
	payload.InstallScriptOutput = &installOutput
	payload.InstallScriptExitCode = &installExitCode
	if err != nil {
		return payload, err
	}

	postOutput, postExitCode, err := r.runInstallerScript(ctx, installer.PostInstallScript, installerPath)
	payload.PostInstallScriptOutput = &postOutput
	payload.PostInstallScriptExitCode = &postExitCode
	if err != nil {
		return payload, err
	}

	return payload, nil
}

func (r *Runner) runInstallerScript(ctx context.Context, scriptContents string, installerPath string) (string, int, error) {
	// run script in installer directory
	installerDir := filepath.Dir(installerPath)
	scriptPath := filepath.Join(installerDir, uuid.NewString())
	if err := os.WriteFile(scriptPath, []byte(scriptContents), constant.DefaultFileMode); err != nil {
		return "", -1, fmt.Errorf("writing script: %w", err)
	}

	if r.execCmdFn == nil {
		r.execCmdFn = scripts.ExecCmd
	}

	env := os.Environ()
	installerPathEnv := fmt.Sprintf("INSTALLER_PATH=%s", installerPath)
	env = append(env, installerPathEnv)

	output, exitCode, err := r.execCmdFn(ctx, scriptPath, env)
	if err != nil {
		return string(output), exitCode, err
	}

	return string(output), exitCode, nil
}
