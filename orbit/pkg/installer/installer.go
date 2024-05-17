package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
	QueryContext(context.Context, string) (*QueryResponse, error)
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

	osqueryConnectionMutex sync.Mutex
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
	connectOsqueryFn := r.connectOsquery
	if connectOsqueryFn == nil {
		connectOsqueryFn = connectOsquery
	}

	if err := connectOsqueryFn(r); err != nil {
		return fmt.Errorf("software installer runner connecting to osquery: %w", err)
	}
	return r.run(context.Background(), config)
}

func connectOsquery(r *Runner) error {
	r.osqueryConnectionMutex.Lock()
	defer r.osqueryConnectionMutex.Unlock()

	if r.OsqueryClient == nil {
		osqueryClient, err := osquery.NewClient(r.osquerySocketPath, 10*time.Second)
		if err != nil {
			log.Err(err).Msg("establishing osquery connection for software install runner")
			return err
		}

		r.OsqueryClient = osqueryClient
	}

	return nil
}

func (r *Runner) run(ctx context.Context, config *fleet.OrbitConfig) error {
	log.Debug().Msg("starting software installers run")
	var errs []error
	for _, installerID := range config.Notifications.PendingSoftwareInstallerIDs {
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}
		payload, err := r.installSoftware(ctx, installerID)
		if err != nil {
			errs = append(errs, err)
			if payload == nil {
				continue
			}
		}
		if err := r.OrbitClient.SaveInstallerResult(payload); err != nil {
			errs = append(errs, fmt.Errorf("saving software install results: %w", err))
		}
	}
	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (r *Runner) preConditionCheck(ctx context.Context, query string) (bool, string, error) {
	res, err := r.OsqueryClient.QueryContext(ctx, query)
	if err != nil {
		return false, "", fmt.Errorf("precondition check: %w", err)
	}

	if res.Status == nil {
		return false, "", errors.New("no query status")
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

	response, err := json.Marshal(res.Response)
	if err != nil {
		return false, "", fmt.Errorf("marshalling query response: %w", err)
	}

	return true, string(response), nil
}

func (r *Runner) installSoftware(ctx context.Context, installID string) (*fleet.HostSoftwareInstallResultPayload, error) {
	log.Debug().Msgf("about to install software with installer id: %s", installID)
	installer, err := r.OrbitClient.GetInstallerDetails(installID)
	if err != nil {
		return nil, fmt.Errorf("fetching software installer details: %w", err)
	}

	payload := &fleet.HostSoftwareInstallResultPayload{}
	payload.InstallUUID = installID

	if installer.PreInstallCondition != "" {
		log.Debug().Msgf("pre-condition is not empty, about to run the query")
		shouldInstall, output, err := r.preConditionCheck(ctx, installer.PreInstallCondition)
		payload.PreInstallConditionOutput = &output
		if err != nil {
			return payload, err
		}

		if !shouldInstall {
			log.Debug().Msgf("pre-condition didn't pass, stopping installation")
			return payload, nil
		}
	}

	if !r.scriptsEnabled() {
		// fleetctl knows that -2 means script was disabled on host
		log.Debug().Msgf("scripts are disabled for this host, stopping installation")
		payload.InstallScriptExitCode = ptr.Int(-2)
		payload.InstallScriptOutput = ptr.String("Scripts are disabled")
		return payload, nil
	}

	tmpDirFn := r.tempDirFn
	if tmpDirFn == nil {
		tmpDirFn = os.MkdirTemp
	}
	tmpDir, err := tmpDirFn("", "")
	if err != nil {
		return payload, fmt.Errorf("creating temporary directory: %w", err)
	}

	log.Debug().Msgf("about to download software installer")
	installerPath, err := r.OrbitClient.DownloadSoftwareInstaller(installer.InstallerID, tmpDir)
	if err != nil {
		return payload, err
	}

	// remove tmp directory and installer
	defer func() {
		removeAllFn := r.removeAllFn
		if removeAllFn == nil {
			removeAllFn = os.RemoveAll
		}
		err := removeAllFn(tmpDir)
		if err != nil {
			log.Err(err)
		}
	}()

	scriptExtension := ".sh"
	if runtime.GOOS == "windows" {
		scriptExtension = ".ps1"
	}
	log.Debug().Msgf("about to run install script")
	installOutput, installExitCode, err := r.runInstallerScript(ctx, installer.InstallScript, installerPath, "install-script"+scriptExtension)
	payload.InstallScriptOutput = &installOutput
	payload.InstallScriptExitCode = &installExitCode
	if err != nil {
		return payload, err
	}

	if installer.PostInstallScript != "" {
		log.Debug().Msgf("about to run post-install script")
		postOutput, postExitCode, postErr := r.runInstallerScript(ctx, installer.PostInstallScript, installerPath, "post-install-script"+scriptExtension)
		payload.PostInstallScriptOutput = &postOutput
		payload.PostInstallScriptExitCode = &postExitCode

		if postErr != nil || postExitCode != 0 {
			log.Info().Msgf("installation of %s failed, attempting rollback. Exit code: %d, error: %s", installerPath, postExitCode, postErr)
			ext := filepath.Ext(installerPath)
			ext = strings.TrimPrefix(ext, ".")
			uninstallScript := file.GetRemoveScript(ext)
			uninstallOutput, uninstallExitCode, uninstallErr := r.runInstallerScript(ctx, uninstallScript, installerPath, "rollback-script")
			log.Info().Msgf(
				"rollback staus: exit code: %d, error: %s, output: %s",
				uninstallExitCode, uninstallErr, uninstallOutput,
			)

			return payload, uninstallErr
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

	execFn := r.execCmdFn
	if execFn == nil {
		execFn = scripts.ExecCmd
	}

	env := os.Environ()
	installerPathEnv := fmt.Sprintf("INSTALLER_PATH=%s", installerPath)
	env = append(env, installerPathEnv)

	output, exitCode, err := execFn(ctx, scriptPath, env)
	if err != nil {
		return string(output), exitCode, err
	}

	return string(output), exitCode, nil
}
