package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	pkgscripts "github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/osquery/osquery-go"
	osquery_gen "github.com/osquery/osquery-go/gen/osquery"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type (
	QueryResponse       = osquery_gen.ExtensionResponse
	QueryResponseStatus = osquery_gen.ExtensionStatus
)

// Client defines the methods required for the API requests to the server. The
// fleet.OrbitClient type satisfies this interface.
type Client interface {
	GetInstallerDetails(installID string) (*fleet.SoftwareInstallDetails, error)
	DownloadSoftwareInstaller(installerID uint, downloadDir string, progressFunc func(int)) (string, error)
	DownloadSoftwareInstallerFromURL(url string, filename string, downloadDir string, progressFunc func(int)) (string, error)
	SaveInstallerResult(payload *fleet.HostSoftwareInstallResultPayload) error
}

type QueryClient interface {
	QueryContext(context.Context, string) (*QueryResponse, error)
}

type Runner struct {
	OsqueryClient QueryClient
	OrbitClient   Client

	// limit execution time of the various scripts run during software installation
	installerExecutionTimeout time.Duration

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

	// extractTarGzFn is the function to call to extract a tarball. If nil, the
	// implementation in the updates package, wrapped with an OS file open, will be used.
	extractTarGzFn func(path string, destDir string) error

	// can be set for tests to replace os.RemoveAll, which is called to remove
	// the script's temporary directory after execution.
	removeAllFn func(string) error

	connectOsquery func(*Runner) error

	scriptsEnabled func() bool

	osqueryConnectionMutex sync.Mutex

	rootDirPath string

	retryOpts []retry.Option

	logger zerolog.Logger
}

const extractionDirectoryName = "extracted"

func NewRunner(client Client, socketPath string, scriptsEnabled func() bool, rootDirPath string) *Runner {
	r := &Runner{
		OrbitClient:               client,
		osquerySocketPath:         socketPath,
		scriptsEnabled:            scriptsEnabled,
		installerExecutionTimeout: pkgscripts.MaxHostSoftwareInstallExecutionTime,
		rootDirPath:               rootDirPath,
		retryOpts:                 []retry.Option{retry.WithMaxAttempts(5)},
		logger:                    log.With().Str("runner", "installer").Logger(),
	}

	return r
}

func (r *Runner) Run(config *fleet.OrbitConfig) error {
	if runtime.GOOS == "darwin" {
		if config.Notifications.RunSetupExperience && !update.CanRun(r.rootDirPath, "swiftDialog", update.SwiftDialogMacOSTarget) {
			log.Info().Msg("exiting software installer config runner early during setup experience: swiftDialog is not installed")
			return nil
		}
	}

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
			log.Error().Err(err).Msg("establishing osquery connection for software install runner")
			return err
		}

		r.OsqueryClient = osqueryClient
	}

	return nil
}

func (r *Runner) run(ctx context.Context, config *fleet.OrbitConfig) error {
	if len(config.Notifications.PendingSoftwareInstallerIDs) > 0 {
		r.logger.Info().Msgf("received notification for software installers: %v", config.Notifications.PendingSoftwareInstallerIDs)
	} else {
		r.logger.Debug().Msg("starting software installers run")
	}

	var errs []error
	for _, installerID := range config.Notifications.PendingSoftwareInstallerIDs {
		logger := r.logger.With().Str("installerID", installerID).Logger()

		logger.Info().Msg("processing")
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}
		payload, err := r.installSoftware(ctx, installerID, logger)
		if err != nil {
			errs = append(errs, err)
			if payload == nil {
				continue
			}
		}
		attemptNum := 1
		err = retry.Do(func() error {
			if err := r.OrbitClient.SaveInstallerResult(payload); err != nil {
				logger.Info().Err(err).Msgf("failed to save installer result, attempt #%d", attemptNum)
				attemptNum++
				return err
			}
			return nil
		}, r.retryOpts...)
		if err != nil {
			errs = append(errs, fmt.Errorf("saving software install results: %w", err))
		}

	}
	if len(errs) != 0 {
		r.logger.Error().Errs("errs", errs).Msg("failures found when processing installers")
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

func (r *Runner) installSoftware(ctx context.Context, installID string, logger zerolog.Logger) (*fleet.HostSoftwareInstallResultPayload, error) {
	logger.Info().Msg("fetching installer details")
	installer, err := r.OrbitClient.GetInstallerDetails(installID)
	if err != nil {
		logger.Err(err).Msg("fetch installer details")
		return nil, fmt.Errorf("fetching software installer details: %w", err)
	}

	payload := &fleet.HostSoftwareInstallResultPayload{}
	payload.InstallUUID = installID

	if installer.PreInstallCondition != "" {
		logger.Info().Msg("pre-condition is not empty, about to run the query")
		shouldInstall, output, err := r.preConditionCheck(ctx, installer.PreInstallCondition)
		payload.PreInstallConditionOutput = &output
		if err != nil {
			logger.Err(err).Msg("pre-condition check failed")
			return payload, err
		}

		if !shouldInstall {
			logger.Info().Msg("pre-condition didn't pass, stopping installation")
			return payload, nil
		}
	}

	if !r.scriptsEnabled() {
		// Fleet knows that -2 means script was disabled on host
		logger.Info().Msg("scripts are disabled for this host, stopping installation")
		payload.InstallScriptExitCode = ptr.Int(fleet.ExitCodeScriptsDisabled)
		payload.InstallScriptOutput = ptr.String("Scripts are disabled")
		return payload, nil
	}

	// Perform the installation with retry logic if MaxRetries > 0
	if installer.MaxRetries > 0 {
		logger.Info().Msgf("Installation configured with %d retries", installer.MaxRetries)
		return r.installWithRetry(ctx, installer, payload, logger)
	}

	// No retries configured, perform single installation attempt
	return r.attemptInstall(ctx, installer, payload, logger)
}

// installWithRetry attempts installation with retry logic for setup experience
func (r *Runner) installWithRetry(ctx context.Context, installer *fleet.SoftwareInstallDetails, payload *fleet.HostSoftwareInstallResultPayload, logger zerolog.Logger) (*fleet.HostSoftwareInstallResultPayload, error) {
	// MaxRetries is the number of retry attempts (0 means no retries, just one attempt)
	// maxAttempts is the total number of attempts (initial + retries)
	maxAttempts := installer.MaxRetries + 1

	var lastErr error

	for attempt := uint(1); attempt <= maxAttempts; attempt++ {
		logger.Debug().Msgf("Installation attempt %d of %d", attempt, maxAttempts)

		// Attempt installation
		resultPayload, err := r.attemptInstall(ctx, installer, payload, logger)

		if err == nil && resultPayload.Status() == fleet.SoftwareInstalled {
			// Success
			logger.Debug().Msgf("Installation succeeded on attempt %d", attempt)
			return resultPayload, nil
		}

		lastErr = err

		if attempt < maxAttempts {
			// Report intermediate failure to server with retries remaining
			resultPayload.RetriesRemaining = maxAttempts - attempt
			if saveErr := r.OrbitClient.SaveInstallerResult(resultPayload); saveErr != nil {
				// Log error but continue with retries
				logger.Err(saveErr).Msg("Failed to report intermediate installation failure to server, continuing with retry")
			} else {
				logger.Debug().Msgf("Reported intermediate failure to server with %d retries remaining", resultPayload.RetriesRemaining)
			}

			// Calculate delay: 10s * 2^(attempt-1) for network/transient errors, immediate retry otherwise
			var delay time.Duration
			if isNetworkOrTransientError(err) {
				// Exponential backoff: 10s, 20s, etc.
				delay = time.Duration(10*(1<<(attempt-1))) * time.Second
				logger.Info().Msgf(
					"Network or transient error detected, waiting %v before retry (attempt %d failed)",
					delay,
					attempt,
				)
			} else {
				// Non-transient error, retry immediately
				logger.Info().Msgf(
					"Retrying software install immediately (attempt %d failed)",
					attempt,
				)
			}

			if delay > 0 && attempt < maxAttempts {
				// Wait before retry
				select {
				case <-ctx.Done():
					return resultPayload, ctx.Err()
				case <-time.After(delay):
				}
			}
		}
	}

	// All retries exhausted
	payload.RetriesRemaining = 0
	logger.Err(lastErr).Msgf("Installation failed after %d attempts", maxAttempts)
	return payload, lastErr
}

// attemptInstall performs a single installation attempt (download + install)
func (r *Runner) attemptInstall(ctx context.Context, installer *fleet.SoftwareInstallDetails, payload *fleet.HostSoftwareInstallResultPayload, logger zerolog.Logger) (*fleet.HostSoftwareInstallResultPayload, error) {
	tmpDirFn := r.tempDirFn
	if tmpDirFn == nil {
		tmpDirFn = os.MkdirTemp
	}
	tmpDir, err := tmpDirFn("", "")
	if err != nil {
		logger.Err(err).Msg("creating temporary directory")
		return payload, fmt.Errorf("creating temporary directory: %w", err)
	}

	// remove tmp directory and installer
	defer func() {
		removeAllFn := r.removeAllFn
		if removeAllFn == nil {
			removeAllFn = os.RemoveAll
		}
		err := removeAllFn(tmpDir)
		if err != nil {
			log.Error().Err(err).Msg("failed to remove tmp dir")
		}
	}()

	progressFn := func() func(n int) {
		chunk := 0
		return func(n int) {
			if n == 0 {
				logger.Info().Msg("done downloading")
				return
			}
			chunk += n
			if chunk >= 10*units.MB {
				logger.Debug().Msgf("downloaded %d bytes", chunk)
				chunk = 0
			}
		}
	}

	var installerPath string
	if installer.SoftwareInstallerURL != nil && installer.SoftwareInstallerURL.URL != "" {
		logger.Info().Msg("about to download software installer from URL")
		installerPath, err = r.OrbitClient.DownloadSoftwareInstallerFromURL(
			installer.SoftwareInstallerURL.URL,
			installer.SoftwareInstallerURL.Filename,
			tmpDir,
			progressFn(),
		)
		if err != nil {
			logger.Err(err).Msg("downloading software installer from URL")
			// If download fails, we will fall back to downloading the installer directly from Fleet server
			installerPath = ""
		}
	}

	if installerPath == "" {
		logger.Info().Msg("about to download software installer from Fleet")
		installerPath, err = r.OrbitClient.DownloadSoftwareInstaller(
			installer.InstallerID,
			tmpDir,
			progressFn(),
		)
		if err != nil {
			logger.Err(err).Msg("failed to download software installer")
			// Set a special exit code to indicate that the installer download failed, so that Fleet
			// will mark this installation as failed.
			payload.InstallScriptExitCode = ptr.Int(fleet.ExitCodeInstallerDownloadFailed)
			payload.InstallScriptOutput = ptr.String("Installer download failed")
			return payload, err
		}
		logger.Info().Str("installerPath", installerPath).Msg("software installer downloaded")
	}

	if strings.HasSuffix(installerPath, ".tgz") || strings.HasSuffix(installerPath, ".tar.gz") {
		logger.Info().Msg("detected tar.gz archive, extracting to subdirectory")
		extractDestination := filepath.Join(tmpDir, extractionDirectoryName)
		err := os.Mkdir(extractDestination, 0o700)
		if err != nil {
			logger.Err(err).Msg("failed to create directory for .tar.gz extraction")
			// Using download failed exit code here to indicate that installer extraction failed
			payload.InstallScriptExitCode = ptr.Int(fleet.ExitCodeInstallerDownloadFailed)
			payload.InstallScriptOutput = ptr.String("Installer extraction failed")
			return payload, err
		}

		extractFn := r.extractTarGzFn
		if extractFn == nil {
			extractFn = func(path string, destDir string) error {
				return file.ExtractTarGz(path, destDir, 2*1024*1024*1024*1024, logger) // 2 TiB limit per extracted file
			}
		}

		if err = extractFn(installerPath, extractDestination); err != nil {
			logger.Err(err).Msg("failed extract .tar.gz archive")
			payload.InstallScriptExitCode = ptr.Int(fleet.ExitCodeInstallerDownloadFailed)
			payload.InstallScriptOutput = ptr.String("Installer extraction failed")
			return payload, err
		}

		// install script will be run inside extracted dir rather than in parent; both dirs
		// will be cleaned up when we're done
		installerPath = extractDestination
	}

	scriptExtension := ".sh"
	if runtime.GOOS == "windows" {
		scriptExtension = ".ps1"
	}

	logger.Info().Msg("about to run install script")
	installOutput, installExitCode, err := r.runInstallerScript(ctx, installer.InstallScript, installerPath, "install-script"+scriptExtension)
	payload.InstallScriptOutput = &installOutput
	payload.InstallScriptExitCode = &installExitCode
	if err != nil {
		logger.Err(err).Msg("install script")
		return payload, err
	}
	logger.Info().Int("exitCode", installExitCode).Msgf("install script")

	if installer.PostInstallScript != "" {
		logger.Info().Str("installerPath", installerPath).Msg("about to run post-install script")
		postOutput, postExitCode, postErr := r.runInstallerScript(ctx, installer.PostInstallScript, installerPath, "post-install-script"+scriptExtension)
		payload.PostInstallScriptOutput = &postOutput
		payload.PostInstallScriptExitCode = &postExitCode

		if postErr != nil || postExitCode != 0 {
			logger.Info().Str(
				"installerPath", installerPath,
			).Int(
				"exitCode", postExitCode,
			).Err(postErr).Msg("installation failed, attempting rollback")
			ext := filepath.Ext(installerPath)
			ext = strings.TrimPrefix(ext, ".")
			uninstallScript := installer.UninstallScript
			var builder strings.Builder
			builder.WriteString(*payload.PostInstallScriptOutput)
			builder.WriteString("\nAttempting rollback by running uninstall script...\n")
			if uninstallScript == "" {
				// The Fleet server is < v4.57.0, so we need to use the old method.
				// If all customers have updated to v4.57.0 or later, we can remove this method.
				uninstallScript = file.GetRemoveScript(ext)
			}
			uninstallOutput, uninstallExitCode, uninstallErr := r.runInstallerScript(ctx, uninstallScript, installerPath,
				"rollback-script"+scriptExtension)
			logger.Info().Msgf(
				"rollback status: exit code: %d, error: %s, output: %s",
				uninstallExitCode, uninstallErr, uninstallOutput,
			)
			builder.WriteString(fmt.Sprintf("Uninstall script exit code: %d\n", uninstallExitCode))
			builder.WriteString(uninstallOutput)
			payload.PostInstallScriptOutput = ptr.String(builder.String())
			return payload, uninstallErr
		}
	}

	return payload, nil
}

// isNetworkOrTransientError determines if an error is network-related or otherwise transient
// and would benefit from waiting before retry
func isNetworkOrTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Check standard library errors using errors.Is
	// These are errors that the client creates directly
	if errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	// Check if it's a net.Error with Timeout() method
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	// Check for DNS errors (these implement net.Error but we can also check the type)
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true // DNS errors are transient
	}

	type statusCodeError interface {
		StatusCode() int
	}
	var scErr statusCodeError
	if errors.As(err, &scErr) {
		code := scErr.StatusCode()
		switch code {
		case 429, // Too Many Requests
			500, // Internal Server Error
			502, // Bad Gateway
			503, // Service Unavailable
			504: // Gateway Timeout
			return true
		}
	}

	// Fall back to string matching only for error messages that come from the server
	// or from libraries that don't expose typed errors
	errStr := err.Error()

	// TLS handshake errors (crypto/tls doesn't expose typed errors for all cases)
	if strings.Contains(errStr, "TLS") && strings.Contains(errStr, "handshake") ||
		// Additional timeout patterns not caught by net.Error.Timeout()
		strings.Contains(errStr, "timeout") ||
		// Network errors that might be string-wrapped or come from server messages
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "unexpected EOF") ||
		strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "context canceled") ||
		// Fall back to string matching for HTTP status errors that might come from
		// other sources (e.g., proxy servers) that don't use our statusCodeErr type
		strings.Contains(errStr, "status 429") || strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "status 503") || strings.Contains(errStr, "service unavailable") ||
		strings.Contains(errStr, "status 502") || strings.Contains(errStr, "bad gateway") ||
		strings.Contains(errStr, "status 504") || strings.Contains(errStr, "gateway timeout") ||
		strings.Contains(errStr, "status 500") || strings.Contains(errStr, "internal server error") ||
		// "resource busy" - file locks, device busy errors
		strings.Contains(errStr, "resource busy") || strings.Contains(errStr, "device or resource busy") ||
		// "file is locked" - file system lock contention
		strings.Contains(errStr, "file is locked") || strings.Contains(errStr, "locked by another process") ||
		// "no space left" - disk full, might resolve if something else cleans up
		strings.Contains(errStr, "no space left") ||
		// "input/output error" - transient I/O errors
		strings.Contains(errStr, "input/output error") {
		return true
	}

	return false
}

func (r *Runner) runInstallerScript(ctx context.Context, scriptContents string, installerPath string, fileName string) (string, int, error) {
	// run script in installer directory
	installerDir := filepath.Dir(installerPath)
	scriptPath := filepath.Join(installerDir, fileName)
	if err := os.WriteFile(scriptPath, []byte(scriptContents), constant.DefaultFileMode); err != nil {
		return "", -1, fmt.Errorf("writing script: %w", err)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, r.installerExecutionTimeout)
	defer cancel()

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
