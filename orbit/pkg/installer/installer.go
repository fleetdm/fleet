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
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/osquery/osquery-go"
	osquery_gen "github.com/osquery/osquery-go/gen/osquery"
	"github.com/rs/zerolog/log"
)

type QueryResponse = osquery_gen.ExtensionResponse
type QueryResponseStatus = osquery_gen.ExtensionStatus

// Client defines the methods required for the API requests to the server. The
// fleet.OrbitClient type satisfies this interface.
type Client interface {
	GetInstallerDetails(installID string) (*fleet.SoftwareInstallDetails, error)
	DownloadSoftwareInstaller(installerID uint, downloadDir string) (string, error)
	SaveInstallerResult(payload *fleet.HostSoftwareInstallResultPayload) error
}

type QueryClient interface {
	Query(context.Context, string) (*QueryResponse, error)
}

type Runner struct {
	OsqueryClient QueryClient
	OrbitClient   Client

	// osquerySocketPath is used to establish the osquery connection
	// if it's ever lost or disconnected
	osquerySocketPath string

	// tempDirFn is the function to call to get the temporary directory to use,
	// inside of which the script-specific subdirectories will be created. If nil,
	// the user's temp dir will be used (can be set to t.TempDir in tests).
	tempDirFn func(dir, pattern string) (string, error)

	// execCmdFn can be set for tests to mock actual execution of the script. If
	// nil, execCmd will be used, which has a different implementation on Windows
	// and non-Windows platforms.
	execCmdFn func(ctx context.Context, scriptPath string, env []string) ([]byte, int, error)

	// can be set for tests to replace os.RemoveAll, which is called to remove
	// the script's temporary directory after execution.
	removeAllFn func(string) error

	connectOsquery func(*Runner) error

	scriptsEnabled func() bool
}

func NewRunner(client Client, socketPath string, scriptsEnabled func() bool) *Runner {
	r := &Runner{
		OrbitClient:       client,
		osquerySocketPath: socketPath,
		scriptsEnabled:    scriptsEnabled,
	}

	return r
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
		osqueryClient, err := osquery.NewClient(r.osquerySocketPath, 10*time.Second)
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
		// TODO(roberto): we can't return the error as the
		// result because the back-end considers any non-empty
		// string as a success.
		// return false, fmt.Sprintf("osqueryd returned error (%d): %s", res.Status.Code, res.Status.Message), fmt.Errorf("non-zero query status: %d \"%s\"", res.Status.Code, res.Status.Message)
		return false, "", fmt.Errorf("non-zero query status: %d \"%s\"", res.Status.Code, res.Status.Message)
	}

	if len(res.Response) == 0 {
		return false, "", nil
	}

	return true, string(response), nil
}

func (r *Runner) installSoftware(ctx context.Context, installID string) (*fleet.HostSoftwareInstallResultPayload, error) {
	installer, err := r.OrbitClient.GetInstallerDetails(installID)
	if err != nil {
		return nil, fmt.Errorf("fetching software installer details: %w", err)
	}

	payload := &fleet.HostSoftwareInstallResultPayload{}
	payload.InstallUUID = installID

	if installer.PreInstallCondition != "" {
		shouldInstall, output, err := r.preConditionCheck(ctx, installer.PreInstallCondition)
		payload.PreInstallConditionOutput = &output
		if err != nil {
			return payload, err
		}

		if !shouldInstall {
			return payload, nil
		}
	}

	if !r.scriptsEnabled() {
		// fleetctl knows that -2 means script was disabled on host
		payload.InstallScriptExitCode = ptr.Int(-2)
		payload.InstallScriptOutput = ptr.String("Scripts are disabled")
		return payload, nil
	}

	if r.tempDirFn == nil {
		r.tempDirFn = os.MkdirTemp
	}
	tmpDir, err := r.tempDirFn("", "")
	if err != nil {
		return payload, fmt.Errorf("creating temporary directory: %w", err)
	}

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

	installOutput, installExitCode, err := r.runInstallerScript(ctx, installer.InstallScript, installerPath, "install-script")
	payload.InstallScriptOutput = &installOutput
	payload.InstallScriptExitCode = &installExitCode
	if err != nil {
		return payload, err
	}

	if installer.PostInstallScript != "" {
		postOutput, postExitCode, postErr := r.runInstallerScript(ctx, installer.PostInstallScript, installerPath, "post-install-script")
		payload.PostInstallScriptOutput = &postOutput
		payload.PostInstallScriptExitCode = &postExitCode

		if postErr != nil || postExitCode != 0 {
			log.Info().Msgf("installation of %s failed, attempting rollback. Exit code: %d, error: %s", installerPath, postExitCode, postErr)
			uninstallScript := file.GetRemoveScript(installerPath)
			uninstallOutput, uninstallExitCode, uninstallErr := r.runInstallerScript(ctx, uninstallScript, installerPath, "rollback-script")
			log.Info().Msgf(
				"rollback staus: exit code: %d, error: %s, output: %s",
				uninstallExitCode, uninstallErr, uninstallOutput,
			)

			return payload, postErr
		}
	}

	return payload, nil
}

func (r *Runner) runInstallerScript(ctx context.Context, scriptContents string, installerPath string, fileName string) (string, int, error) {
	// run script in installer directory
	installerDir := filepath.Dir(installerPath)
	scriptPath := filepath.Join(installerDir, fileName)
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
