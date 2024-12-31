package update

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/bitlocker"
	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	fleetscripts "github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type runCmdFunc func() error

type checkEnrollmentFunc func() (bool, string, error)

type checkAssignedEnrollmentProfileFunc func(url string) error

// renewEnrollmentProfileConfigReceiver is a kind of middleware that wraps an
// OrbitConfigFetcher and detects if the fleet server sent a notification to
// renew the enrollment profile. If so, it runs the command (as root) to
// bootstrap the renewal of the profile on the device (the user still needs to
// execute some manual steps to accept the new profile).
//
// It ensures only one renewal command is executed at any given time, and that
// it doesn't re-execute the command until a certain amount of time has passed.
type renewEnrollmentProfileConfigReceiver struct {
	// Frequency is the minimum amount of time that must pass between two
	// executions of the profile renewal command.
	Frequency time.Duration

	// for tests, to be able to mock command execution. If nil, will use
	// runRenewEnrollmentProfile.
	runCmdFn runCmdFunc

	// for tests, to be able to mock the function that checks for Fleet
	// enrollment
	checkEnrollmentFn checkEnrollmentFunc

	// for tests, to be able to mock the function that checks for the assigned enrollment profile
	checkAssignedEnrollmentProfileFn checkAssignedEnrollmentProfileFunc

	// ensures only one command runs at a time, protects access to lastRun
	cmdMu   sync.Mutex
	lastRun time.Time

	fleetURL string
}

func ApplyRenewEnrollmentProfileConfigFetcherMiddleware(fetcher OrbitConfigFetcher, frequency time.Duration, fleetURL string) fleet.OrbitConfigReceiver {
	return &renewEnrollmentProfileConfigReceiver{Frequency: frequency, fleetURL: fleetURL}
}

func (h *renewEnrollmentProfileConfigReceiver) Run(config *fleet.OrbitConfig) error {
	if config.Notifications.RenewEnrollmentProfile {
		if h.cmdMu.TryLock() {
			defer h.cmdMu.Unlock()

			// Note that the macOS notification popup will be shown periodically
			// until the Fleet server gets notified that the device is now properly
			// enrolled (after the user's manual steps, and osquery reporting the
			// updated mdm enrollment).
			// See https://github.com/fleetdm/fleet/pull/9409#discussion_r1084382455
			if time.Since(h.lastRun) >= h.Frequency {
				// we perform this check locally on the client too to avoid showing the
				// dialog if the client is enrolled to an MDM server.
				enrollFn := h.checkEnrollmentFn
				if enrollFn == nil {
					enrollFn = profiles.IsEnrolledInMDM
				}
				enrolled, mdmServerURL, err := enrollFn()
				if err != nil {
					log.Error().Err(err).Msg("fetching enrollment status")
					return nil
				}
				if enrolled {
					log.Info().Msgf("a request to renew the enrollment profile was processed but not executed because the host is enrolled into an MDM server with URL: %s", mdmServerURL)
					h.lastRun = time.Now().Add(-h.Frequency).Add(2 * time.Minute)
					return nil
				}

				// we perform this check locally on the client too to avoid showing the
				// dialog if the Fleet enrollment profile has not been assigned to the device in
				// Apple Business Manager.
				assignedFn := h.checkAssignedEnrollmentProfileFn
				if assignedFn == nil {
					assignedFn = profiles.CheckAssignedEnrollmentProfile
				}
				if err := assignedFn(h.fleetURL); err != nil {
					log.Error().Err(err).Msg("checking assigned enrollment profile")
					log.Info().Msg("a request to renew the enrollment profile was processed but not executed because there was an error checking the assigned enrollment profile.")
					// TODO: Design a better way to backoff `profiles show` so that the device doesn't get rate
					// limited by Apple. For now, wait at least 2 minutes before retrying.
					h.lastRun = time.Now().Add(-h.Frequency).Add(2 * time.Minute)
					return nil
				}

				fn := h.runCmdFn
				if fn == nil {
					fn = runRenewEnrollmentProfile
				}
				if err := fn(); err != nil {
					log.Info().Err(err).Msg("calling /usr/bin/profiles to renew enrollment profile failed")
					// TODO: Design a better way to backoff `profiles show` so that the device doesn't get rate
					// limited by Apple. For now, wait at least 2 minutes before retrying.
					h.lastRun = time.Now().Add(-h.Frequency).Add(2 * time.Minute)
					return nil
				}
				h.lastRun = time.Now()
				log.Info().Msg("successfully called /usr/bin/profiles to renew enrollment profile")

			} else {
				log.Debug().Msg("skipped calling /usr/bin/profiles to renew enrollment profile, last run was too recent")
			}
		}
	}
	return nil
}

type execWinAPIFunc func(WindowsMDMEnrollmentArgs) error

type windowsMDMEnrollmentConfigReceiver struct {
	// Frequency is the minimum amount of time that must pass between two
	// executions of the windows MDM enrollment attempt.
	Frequency time.Duration
	// HostUUID is the current host's UUID.
	HostUUID string

	// OrbitNodeKey is the current host's orbit node key.
	nodeKeyGetter OrbitNodeKeyGetter

	// for tests, to be able to mock API commands. If nil, will use
	// RunWindowsMDMEnrollment and RunWindowsMDMUnenrollment respectively.
	execEnrollFn   execWinAPIFunc
	execUnenrollFn execWinAPIFunc

	// ensures only one command runs at a time, protects access to lastXxxRun and
	// isWindowsServer.
	mu              sync.Mutex
	lastEnrollRun   time.Time
	lastUnenrollRun time.Time
	isWindowsServer bool
}

type OrbitNodeKeyGetter interface {
	GetNodeKey() (string, error)
}

func ApplyWindowsMDMEnrollmentFetcherMiddleware(
	frequency time.Duration,
	hostUUID string,
	nodeKeyGetter OrbitNodeKeyGetter,
) fleet.OrbitConfigReceiver {
	return &windowsMDMEnrollmentConfigReceiver{
		Frequency:     frequency,
		HostUUID:      hostUUID,
		nodeKeyGetter: nodeKeyGetter,
	}
}

var errIsWindowsServer = errors.New("device is a Windows Server")

// Run checks if the fleet server set the "needs windows {un}enrollment" flag
// to true, and executes the command to {un}enroll into Windows MDM (or not, if
// the device is a Windows Server). It also unenrolls the device if the flag
// "needs MDM migration" is set to true, so that the device can then be
// enrolled in Fleet MDM.
func (w *windowsMDMEnrollmentConfigReceiver) Run(cfg *fleet.OrbitConfig) error {
	switch {
	case cfg.Notifications.NeedsProgrammaticWindowsMDMEnrollment:
		w.attemptEnrollment(cfg.Notifications)
	case cfg.Notifications.NeedsProgrammaticWindowsMDMUnenrollment,
		cfg.Notifications.NeedsMDMMigration:
		label := "unenroll"
		if cfg.Notifications.NeedsMDMMigration {
			label = "migrate"
		}
		w.attemptUnenrollment(label)
	}
	return nil
}

func (w *windowsMDMEnrollmentConfigReceiver) attemptEnrollment(notifs fleet.OrbitConfigNotifications) {
	if notifs.WindowsMDMDiscoveryEndpoint == "" {
		log.Info().Err(errors.New("discovery endpoint is missing")).Msg("skipping enrollment, discovery endpoint is empty")
		return
	}

	if w.mu.TryLock() {
		defer w.mu.Unlock()

		// do not enroll Windows Servers, and do not attempt enrollment if the last
		// run is not at least Frequency ago.
		if w.isWindowsServer {
			log.Debug().Msg("skipped calling RegisterDeviceWithManagement to enroll Windows device, device is a server")
			return
		}
		if time.Since(w.lastEnrollRun) <= w.Frequency {
			log.Debug().Msg("skipped calling RegisterDeviceWithManagement to enroll Windows device, last run was too recent")
			return
		}

		nodeKey, err := w.nodeKeyGetter.GetNodeKey()
		if err != nil {
			log.Info().Err(err).Msg("failed to get orbit node key to enroll Windows device")
			return
		}

		fn := w.execEnrollFn
		if fn == nil {
			fn = RunWindowsMDMEnrollment
		}
		args := WindowsMDMEnrollmentArgs{
			DiscoveryURL: notifs.WindowsMDMDiscoveryEndpoint,
			HostUUID:     w.HostUUID,
			OrbitNodeKey: nodeKey,
		}
		if err := fn(args); err != nil {
			if errors.Is(err, errIsWindowsServer) {
				w.isWindowsServer = true
				log.Info().Msg("device is a Windows Server, skipping enrollment")
			} else {
				log.Info().Err(err).Msg("calling RegisterDeviceWithManagement to enroll Windows device failed")
			}
			return
		}

		w.lastEnrollRun = time.Now()
		log.Info().Msg("successfully called RegisterDeviceWithManagement to enroll Windows device")
	}
}

func (w *windowsMDMEnrollmentConfigReceiver) attemptUnenrollment(actionLabel string) {
	if w.mu.TryLock() {
		defer w.mu.Unlock()

		// do not unenroll Windows Servers, and do not attempt unenrollment if the
		// last run is not at least Frequency ago.
		if w.isWindowsServer {
			log.Debug().Msgf("skipped calling UnregisterDeviceWithManagement to %s Windows device, device is a server", actionLabel)
			return
		}
		if time.Since(w.lastUnenrollRun) <= w.Frequency {
			log.Debug().Msgf("skipped calling UnregisterDeviceWithManagement to %s Windows device, last run was too recent", actionLabel)
			return
		}

		fn := w.execUnenrollFn
		if fn == nil {
			fn = RunWindowsMDMUnenrollment
		}
		// NOTE: args is actually unused by unenrollment, it is just for the
		// function signature consistency.
		args := WindowsMDMEnrollmentArgs{}
		if err := fn(args); err != nil {
			if errors.Is(err, errIsWindowsServer) {
				w.isWindowsServer = true
				log.Info().Msgf("device is a Windows Server, skipping %s", actionLabel)
			} else {
				log.Info().Err(err).Msgf("calling UnregisterDeviceWithManagement to %s Windows device failed", actionLabel)
			}
			return
		}

		w.lastUnenrollRun = time.Now()
		log.Info().Msgf("successfully called UnregisterDeviceWithManagement to %s Windows device", actionLabel)
	}
}

type runScriptsConfigReceiver struct {
	// ScriptsExecutionEnabled indicates if this agent allows scripts execution.
	// If it doesn't, scripts are not executed, but a response is returned to the
	// Fleet server so it knows the agent processed the request. Note that this
	// should be set to the value of the --scripts-enabled command-line flag. An
	// additional, dynamic check is done automatically by the
	// runScriptsConfigReceiver if this field is false to get the value from the
	// MDM configuration profile.
	ScriptsExecutionEnabled bool

	// ScriptsClient is the client to use to fetch the script to execute and save
	// back its results.
	ScriptsClient scripts.Client

	// the dynamic scripts enabled check is done to check via mdm configuration
	// profile if the host is allowed to run dynamic scripts. It is only done
	// on macos and only if ScriptsExecutionEnabled is false.
	dynamicScriptsEnabled              atomic.Bool
	dynamicScriptsEnabledCheckInterval time.Duration
	// for tests, if set will use this instead of profiles.GetFleetdConfig.
	testGetFleetdConfig func() (*fleet.MDMAppleFleetdConfig, error)

	// for tests, to be able to mock command execution. If nil, will use
	// (scripts.Runner{...}).Run. To help with testing, the function receives as
	// argument the scripts.Runner value that would've executed the call.
	runScriptsFn func(*scripts.Runner, []string) error

	// ensures only one script execution runs at a time
	mu sync.Mutex

	rootDirPath string
}

func ApplyRunScriptsConfigFetcherMiddleware(
	scriptsEnabled bool, scriptsClient scripts.Client, rootDirPath string,
) (fleet.OrbitConfigReceiver, func() bool) {
	scriptsFetcher := &runScriptsConfigReceiver{
		ScriptsExecutionEnabled:            scriptsEnabled,
		ScriptsClient:                      scriptsClient,
		dynamicScriptsEnabledCheckInterval: 5 * time.Minute,
		rootDirPath:                        rootDirPath,
	}
	// start the dynamic check for scripts enabled if required
	scriptsFetcher.runDynamicScriptsEnabledCheck()
	return scriptsFetcher, scriptsFetcher.scriptsEnabled
}

func (h *runScriptsConfigReceiver) runDynamicScriptsEnabledCheck() {
	getFleetdConfig := h.testGetFleetdConfig
	if getFleetdConfig == nil {
		getFleetdConfig = profiles.GetFleetdConfig
	}

	// only run on macos and only if scripts are disabled by default for the
	// agent (but always run if a test get fleetd config function is set).
	if (runtime.GOOS == "darwin" && !h.ScriptsExecutionEnabled) || (h.testGetFleetdConfig != nil) {
		go func() {
			runCheck := func() {
				cfg, err := getFleetdConfig()
				if err != nil {
					if err != profiles.ErrNotImplemented {
						// note that an unenrolled host will not return an error, it will
						// return the zero-value struct, so this logging should not be too
						// noisy unless something goes wrong.
						log.Info().Err(err).Msg("get fleetd configuration failed")
					}
					return
				}
				h.dynamicScriptsEnabled.Store(cfg.EnableScripts)
			}

			// check immediately at startup, before checking at the interval
			runCheck()

			// check every minute
			for range time.Tick(h.dynamicScriptsEnabledCheckInterval) {
				runCheck()
			}
		}()
	}
}

// GetConfig calls the wrapped Fetcher's GetConfig method, and if the fleet
// server sent a list of scripts to execute, starts a goroutine to execute
// them.
func (h *runScriptsConfigReceiver) Run(cfg *fleet.OrbitConfig) error {
	timeout := fleetscripts.MaxHostExecutionTime
	if cfg.ScriptExeTimeout > 0 {
		timeout = time.Duration(cfg.ScriptExeTimeout) * time.Second
	}

	if runtime.GOOS == "darwin" {
		if cfg.Notifications.RunSetupExperience && !CanRun(h.rootDirPath, "swiftDialog", SwiftDialogMacOSTarget) {
			log.Debug().Msg("exiting scripts config runner early during setup experience: swiftDialog is not installed")
			return nil
		}
	}

	if len(cfg.Notifications.PendingScriptExecutionIDs) > 0 {
		if h.mu.TryLock() {
			log.Debug().Msgf("received request to run scripts %v", cfg.Notifications.PendingScriptExecutionIDs)

			runner := &scripts.Runner{
				ScriptExecutionEnabled: h.scriptsEnabled(),
				Client:                 h.ScriptsClient,
				ScriptExecutionTimeout: timeout,
			}
			fn := runner.Run
			if h.runScriptsFn != nil {
				fn = func(execIDs []string) error {
					return h.runScriptsFn(runner, execIDs)
				}
			}

			go func() {
				defer h.mu.Unlock()

				if err := fn(cfg.Notifications.PendingScriptExecutionIDs); err != nil {
					log.Info().Err(err).Msg("running scripts failed")
					return
				}
				log.Debug().Msgf("running scripts %v succeeded", cfg.Notifications.PendingScriptExecutionIDs)
			}()
		}
	}
	return nil
}

func (h *runScriptsConfigReceiver) scriptsEnabled() bool {
	// scripts are always enabled if the agent is started with the
	// --scripts-enabled flag. If it is not started with this flag, then
	// scripts are enabled only if the mdm profile says so.
	return h.ScriptsExecutionEnabled || h.dynamicScriptsEnabled.Load()
}

type DiskEncryptionKeySetter interface {
	SetOrUpdateDiskEncryptionKey(diskEncryptionStatus fleet.OrbitHostDiskEncryptionKeyPayload) error
}

// execEncryptVolumeFunc handles the encryption of a volume identified by its
// string identifier (e.g., "C:").
//
// It returns a string representing the recovery key and an error if any occurs during the process.
type execEncryptVolumeFunc func(volumeID string) (recoveryKey string, err error)

// execGetEncryptionStatusFunc retrieves the encryption status of all volumes
// managed by Bitlocker.
//
// It returns a slice of bitlocker.VolumeStatus, each representing the
// encryption status of a volume, and an error if the operation fails.
type execGetEncryptionStatusFunc func() (status []bitlocker.VolumeStatus, err error)

// execDecryptVolumeFunc handles the decryption of a volume identified by its
// string identifier (e.g., "C:")
//
// It returns an error if the process fails.
type execDecryptVolumeFunc func(volumeID string) error

type windowsMDMBitlockerConfigReceiver struct {
	// Frequency is the minimum amount of time that must pass between two
	// executions of the windows MDM enrollment attempt.
	Frequency time.Duration

	// Bitlocker Operation Results
	EncryptionResult DiskEncryptionKeySetter

	// tracks last time a disk encryption has successfully run
	lastRun time.Time

	// ensures only one script execution runs at a time
	mu sync.Mutex

	// for tests, to be able to mock API commands. If nil, will use
	// bitlocker.EncryptVolume
	execEncryptVolumeFn execEncryptVolumeFunc

	// for tests, to be able to mock API commands. If nil, will use
	// bitlocker.GetEncryptionStatus
	execGetEncryptionStatusFn execGetEncryptionStatusFunc

	// for tests, to be able to mock the decryption process. If nil, will use
	// bitlocker.DecryptVolume
	execDecryptVolumeFn execDecryptVolumeFunc
}

func ApplyWindowsMDMBitlockerFetcherMiddleware(
	frequency time.Duration,
	encryptionResult DiskEncryptionKeySetter,
) fleet.OrbitConfigReceiver {
	return &windowsMDMBitlockerConfigReceiver{
		Frequency:        frequency,
		EncryptionResult: encryptionResult,
	}
}

// GetConfig calls the wrapped Fetcher's GetConfig method, and if the fleet
// server set the "EnforceBitLockerEncryption" flag to true, executes the command
// to attempt BitlockerEncryption (or not, if the device is a Windows Server).
func (w *windowsMDMBitlockerConfigReceiver) Run(cfg *fleet.OrbitConfig) error {
	if cfg.Notifications.EnforceBitLockerEncryption {
		if w.mu.TryLock() {
			defer w.mu.Unlock()

			w.attemptBitlockerEncryption(cfg.Notifications)
		}
	}

	return nil
}

func (w *windowsMDMBitlockerConfigReceiver) attemptBitlockerEncryption(notifs fleet.OrbitConfigNotifications) {
	if time.Since(w.lastRun) <= w.Frequency {
		log.Debug().Msg("skipped encryption process, last run was too recent")
		return
	}

	// Windows servers are not supported. Check and skip if that's the case.
	if isServer, err := IsRunningOnWindowsServer(); isServer || err != nil {
		if err != nil {
			log.Error().Err(err).Msg("checking if the host is a Windows server")
		} else {
			log.Debug().Msg("device is a Windows Server, encryption is not going to be performed")
		}
		return
	}

	const targetVolume = "C:"
	encryptionStatus, err := w.getEncryptionStatusForVolume(targetVolume)
	if err != nil {
		log.Debug().Err(err).Msgf("unable to get encryption status for target volume %s, continuing anyway", targetVolume)
	}

	// don't do anything if the disk is being encrypted/decrypted
	if w.bitLockerActionInProgress(encryptionStatus) {
		log.Debug().Msgf("skipping encryption as the disk is not available. Disk conversion status: %d", encryptionStatus.ConversionStatus)
		return
	}

	// if the disk is encrypted, try to decrypt it first.
	if encryptionStatus != nil &&
		encryptionStatus.ConversionStatus == bitlocker.ConversionStatusFullyEncrypted {
		log.Debug().Msg("disk was previously encrypted. Attempting to decrypt it")

		if err := w.decryptVolume(targetVolume); err != nil {
			log.Error().Err(err).Msg("decryption failed")

			if serverErr := w.updateFleetServer("", err); serverErr != nil {
				log.Error().Err(serverErr).Msg("failed to send decryption failure to Fleet Server")
				return
			}
		}

		// return regardless of the operation output.
		//
		// the decryption process takes an unknown amount of time (depending on
		// factors outside of our control) and the next tick will be a noop if the
		// disk is not ready to be encrypted yet (due to the
		// w.bitLockerActionInProgress check above)
		return
	}

	recoveryKey, encryptionErr := w.performEncryption(targetVolume)
	// before reporting the error to the server, check if the error we've got is valid.
	// see the description of w.isMisreportedDecryptionError and issue #15916.
	var pErr *bitlocker.EncryptionError
	if errors.As(encryptionErr, &pErr) && w.isMisreportedDecryptionError(pErr, encryptionStatus) {
		log.Error().Msg("disk encryption failed due to previous unsuccessful attempt, user action required")
		return
	}

	if serverErr := w.updateFleetServer(recoveryKey, encryptionErr); serverErr != nil {
		log.Error().Err(serverErr).Msg("failed to send encryption result to Fleet Server")
		return
	}

	if encryptionErr != nil {
		log.Error().Err(err).Msg("failed to encrypt the volume")
		return
	}

	w.lastRun = time.Now()
}

// getEncryptionStatusForVolume retrieves the encryption status for a specific volume.
func (w *windowsMDMBitlockerConfigReceiver) getEncryptionStatusForVolume(volume string) (*bitlocker.EncryptionStatus, error) {
	fn := w.execGetEncryptionStatusFn
	if fn == nil {
		fn = bitlocker.GetEncryptionStatus
	}
	status, err := fn()
	if err != nil {
		return nil, err
	}

	for _, s := range status {
		if s.DriveVolume == volume {
			return s.Status, nil
		}
	}

	return nil, nil
}

// bitLockerActionInProgress determines an encryption/decription action is in
// progress based on the reported status.
func (w *windowsMDMBitlockerConfigReceiver) bitLockerActionInProgress(status *bitlocker.EncryptionStatus) bool {
	if status == nil {
		return false
	}

	// Check if the status matches any of the specified conditions
	return status.ConversionStatus == bitlocker.ConversionStatusDecryptionInProgress ||
		status.ConversionStatus == bitlocker.ConversionStatusDecryptionPaused ||
		status.ConversionStatus == bitlocker.ConversionStatusEncryptionInProgress ||
		status.ConversionStatus == bitlocker.ConversionStatusEncryptionPaused
}

// performEncryption executes the encryption process.
func (w *windowsMDMBitlockerConfigReceiver) performEncryption(volume string) (string, error) {
	fn := w.execEncryptVolumeFn
	if fn == nil {
		fn = bitlocker.EncryptVolume
	}

	recoveryKey, err := fn(volume)
	if err != nil {
		return "", err
	}

	return recoveryKey, nil
}

func (w *windowsMDMBitlockerConfigReceiver) decryptVolume(targetVolume string) error {
	fn := w.execDecryptVolumeFn
	if fn == nil {
		fn = bitlocker.DecryptVolume
	}

	return fn(targetVolume)
}

// isMisreportedDecryptionError checks whether the given error is a potentially
// misreported decryption error.
//
// It addresses cases where a previous encryption attempt failed due to other
// errors but subsequent attempts to encrypt the disk could erroneously return
// a bitlocker.FVE_E_NOT_DECRYPTED error.
//
// This function checks if the disk is actually fully decrypted
// (status.ConversionStatus == bitlocker.CONVERSION_STATUS_FULLY_DECRYPTED) and
// whether the reported error is bitlocker.FVE_E_NOT_DECRYPTED. If these
// conditions are met, the error is not accurately reflecting the disk's actual
// encryption state.
//
// For more context, see issue #15916
func (w *windowsMDMBitlockerConfigReceiver) isMisreportedDecryptionError(err *bitlocker.EncryptionError, status *bitlocker.EncryptionStatus) bool {
	return err.Code() == bitlocker.ErrorCodeNotDecrypted &&
		status != nil &&
		status.ConversionStatus == bitlocker.ConversionStatusFullyDecrypted
}

func (w *windowsMDMBitlockerConfigReceiver) updateFleetServer(key string, err error) error {
	// Getting Bitlocker encryption operation error message if any
	// This is going to be sent to Fleet Server
	bitlockerError := ""
	if err != nil {
		bitlockerError = err.Error()
	}

	// Update Fleet Server with encryption result
	payload := fleet.OrbitHostDiskEncryptionKeyPayload{
		EncryptionKey: []byte(key),
		ClientError:   bitlockerError,
	}

	return w.EncryptionResult.SetOrUpdateDiskEncryptionKey(payload)
}
