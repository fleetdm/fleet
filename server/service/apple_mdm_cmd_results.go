package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/google/uuid"
	"github.com/micromdm/plist"
)

type InstalledApplicationListResult interface {
	fleet.MDMCommandResults
	AvailableApps() []fleet.Software
	HostPlatform() string
}

type installedApplicationListResult struct {
	raw           []byte
	availableApps []fleet.Software
	uuid          string
	hostUUID      string
	hostPlatform  string
}

func (i *installedApplicationListResult) Raw() []byte                     { return i.raw }
func (i *installedApplicationListResult) UUID() string                    { return i.uuid }
func (i *installedApplicationListResult) HostUUID() string                { return i.hostUUID }
func (i *installedApplicationListResult) AvailableApps() []fleet.Software { return i.availableApps }
func (i *installedApplicationListResult) HostPlatform() string            { return i.hostPlatform }

func NewInstalledApplicationListResult(ctx context.Context, rawResult []byte, uuid, hostUUID, hostPlatform string) (InstalledApplicationListResult, error) {
	var source string
	switch hostPlatform {
	case "ios":
		source = "ios_apps"
	case "ipados":
		source = "ipados_apps"
	default:
		source = "apps"
	}
	list, err := unmarshalAppList(ctx, rawResult, source)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshal app list for new installed application list result")
	}

	return &installedApplicationListResult{
		raw:           rawResult,
		uuid:          uuid,
		availableApps: list,
		hostUUID:      hostUUID,
		hostPlatform:  hostPlatform,
	}, nil
}

func NewInstalledApplicationListResultsHandler(
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger *slog.Logger,
	verifyTimeout, verifyRequestDelay time.Duration,
	newActivityFn fleet.NewActivityFunc,
) fleet.MDMCommandResultsHandler {
	return func(ctx context.Context, commandResults fleet.MDMCommandResults) error {
		installedAppResult, ok := commandResults.(InstalledApplicationListResult)
		if !ok {
			return ctxerr.New(ctx, "unexpected results type")
		}

		// Then it's not a command sent by Fleet, so skip it
		if !strings.HasPrefix(installedAppResult.UUID(), fleet.VerifySoftwareInstallVPPPrefix) {
			return nil
		}

		installedApps := installedAppResult.AvailableApps()

		expectedVPPInstalls, err := ds.GetUnverifiedVPPInstallsForHost(ctx, installedAppResult.HostUUID())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "InstalledApplicationList handler: getting install record")
		}

		expectedInHouseInstalls, err := ds.GetUnverifiedInHouseAppInstallsForHost(ctx, installedAppResult.HostUUID())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "InstalledApplicationList handler: get unverified in house installs")
		}

		if len(expectedVPPInstalls) == 0 && len(expectedInHouseInstalls) == 0 {
			logger.WarnContext(ctx, "no apple MDM installs found for host", "host_uuid", installedAppResult.HostUUID(), "verification_command_uuid", installedAppResult.UUID())
			// Nothing is left to verify, so release the verify command the same way the
			// terminal path below does. Holding it would suppress the next install's
			// acknowledgement on this host until the daily cleanup removes it.
			return ctxerr.Wrap(
				ctx,
				ds.RemoveHostMDMCommandByHostUUID(ctx, installedAppResult.HostUUID(), fleet.VerifySoftwareInstallVPPPrefix),
				"InstalledApplicationList handler: removing host mdm command with no installs to verify",
			)
		}

		installsByBundleID := map[string]fleet.Software{}
		for _, install := range installedApps {
			installsByBundleID[install.BundleIdentifier] = install
		}

		// We've handled the "no installs found" case above,
		// and installs are scoped to a single host via the host UUID, so this is OK.
		var hostID uint
		switch {
		case len(expectedInHouseInstalls) > 0:
			hostID = expectedInHouseInstalls[0].HostID
		case len(expectedVPPInstalls) > 0:
			hostID = expectedVPPInstalls[0].HostID
		}

		type installStatusSetter struct {
			// Used to mark the install as verified
			verifyFn func(ctx context.Context, hostID uint, installUUID string, verificationUUID string) error
			// Used to mark the install as failed
			failFn func(ctx context.Context, hostID uint, installUUID string, verificationUUID string) error
			// Used to get the activity data for an install
			activityFn func(ctx context.Context, results *mdm.CommandResults, fromSetupExp bool, fromAutoUpdate bool) (*fleet.User, fleet.ActivityDetails, error)
		}

		// The requireXcodeSpecialCase is used to identify if we need to poll the list of apps
		// with managedonly=false to verify the Xcode VPP app, which only reports during Installing=true
		// as managed-only, and then disappears from the list once installed.
		// See https://github.com/fleetdm/fleet/issues/37290#issuecomment-3774473552
		const xcodeBundleID = "com.apple.dt.Xcode"
		var poll, shouldRefetch, requireXcodeSpecialCase bool
		setStatusForExpectedInstall := func(
			expectedInstall *fleet.HostVPPSoftwareInstall,
			setter installStatusSetter,
		) error {
			fromAutoUpdate, err := ds.IsAutoUpdateVPPInstall(ctx, expectedInstall.InstallCommandUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "checking if vpp install is from auto update")
			}
			// If we don't find the app in the result, then we need to poll for it (within the timeout).
			appFromResult, appWasReported := installsByBundleID[expectedInstall.BundleIdentifier]

			var terminalStatus string
			switch {
			case appFromResult.Installed:
				if err := setter.verifyFn(ctx, expectedInstall.HostID, expectedInstall.InstallCommandUUID, installedAppResult.UUID()); err != nil {
					return ctxerr.Wrap(ctx, err, "InstalledApplicationList handler: set vpp install verified")
				}

				terminalStatus = fleet.MDMAppleStatusAcknowledged
				shouldRefetch = true
			case expectedInstall.InstallCommandAckAt != nil && time.Since(*expectedInstall.InstallCommandAckAt) > verifyTimeout:
				if err := setter.failFn(ctx, expectedInstall.HostID, expectedInstall.InstallCommandUUID, installedAppResult.UUID()); err != nil {
					return ctxerr.Wrap(ctx, err, "InstalledApplicationList handler: set vpp install failed")
				}

				terminalStatus = fleet.MDMAppleStatusError
			}

			if terminalStatus == "" {
				poll = true
				// use the Xcode special-case (managedonly=false) only if it wasn't reported
				// in the current result (if it was reported and gets here, it means is still "Installing"),
				// so we will list the full apps for verification only after it finished "installing", until
				// it gets verified or times out doing so (and possibly once _before_ it starts installing).
				// This minimizes the number of times we request the (~100KB large) payload of all apps.
				requireXcodeSpecialCase = expectedInstall.BundleIdentifier == xcodeBundleID &&
					installedAppResult.HostPlatform() == "darwin" && !appWasReported
				return nil
			}

			// this might be a setup experience VPP install, so we'll try to update setup experience status
			var fromSetupExperience bool
			if updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, fleet.SetupExperienceVPPInstallResult{
				HostUUID:      installedAppResult.HostUUID(),
				CommandUUID:   expectedInstall.InstallCommandUUID,
				CommandStatus: terminalStatus,
			}, newActivityFn); err != nil {
				return ctxerr.Wrap(ctx, err, "updating setup experience status from VPP install result")
			} else if updated {
				fromSetupExperience = true
				logger.DebugContext(ctx, "setup experience VPP install result updated", "host_uuid", installedAppResult.HostUUID(), "execution_id", expectedInstall.InstallCommandUUID)
			}

			// create an activity for installing only if we're in a terminal state
			user, act, err := setter.activityFn(ctx, &mdm.CommandResults{CommandUUID: expectedInstall.InstallCommandUUID, Status: terminalStatus}, fromSetupExperience, fromAutoUpdate)
			if err != nil {
				if fleet.IsNotFound(err) {
					// Then this isn't an MDM-based install, so no activity generated
					return nil
				}

				return ctxerr.Wrap(ctx, err, "fetching data for installed app store app activity")
			}

			if err := newActivityFn(ctx, user, act); err != nil {
				return ctxerr.Wrap(ctx, err, "creating activity for installed app store app")
			}

			return nil
		}

		for _, expectedInstall := range expectedVPPInstalls {
			setter := installStatusSetter{
				ds.SetVPPInstallAsVerified,
				ds.SetVPPInstallAsFailed,
				func(ctx context.Context, results *mdm.CommandResults, fromSetupExp bool, fromAutoUpdate bool) (*fleet.User, fleet.ActivityDetails, error) {
					user, act, err := ds.GetPastActivityDataForVPPAppInstall(ctx, results)
					if err != nil {
						return nil, nil, err
					}

					act.FromSetupExperience = fromSetupExp
					act.FromAutoUpdate = fromAutoUpdate

					return user, act, nil
				},
			}

			if err := setStatusForExpectedInstall(expectedInstall, setter); err != nil {
				return ctxerr.Wrap(ctx, err, "setting status for vpp installs")
			}
		}

		for _, expectedInstall := range expectedInHouseInstalls {
			setter := installStatusSetter{
				ds.SetInHouseAppInstallAsVerified,
				ds.SetInHouseAppInstallAsFailed,
				// fromAutoUpdate is ignored: in-house apps have no auto-update flow
				// and ActivityTypeInstalledSoftware has no field for it.
				func(ctx context.Context, results *mdm.CommandResults, fromSetupExp bool, _ bool) (*fleet.User, fleet.ActivityDetails, error) {
					user, act, err := ds.GetPastActivityDataForInHouseAppInstall(ctx, results)
					if err != nil {
						return nil, nil, err
					}
					act.FromSetupExperience = fromSetupExp
					return user, act, nil
				},
			}
			if err := setStatusForExpectedInstall(expectedInstall, setter); err != nil {
				return ctxerr.Wrap(ctx, err, "setting status for in-house app installs")
			}
		}

		if poll {
			// Queue a job to verify the VPP install.
			return ctxerr.Wrap(
				ctx,
				worker.QueueVPPInstallVerificationJob(ctx, ds, logger, verifyRequestDelay,
					installedAppResult.HostUUID(), installedAppResult.UUID(), requireXcodeSpecialCase),
				"InstalledApplicationList handler: queueing vpp install verification job",
			)
		}

		if shouldRefetch {
			switch installedAppResult.HostPlatform() {
			case "darwin":
				// Request host refetch to get the most up to date software data ASAP.
				if err := ds.UpdateHostRefetchRequested(ctx, hostID, true); err != nil {
					return ctxerr.Wrap(ctx, err, "request refetch for host after vpp install verification")
				}
			default:
				// Track before enqueueing so a fast device ack can't race the
				// insert and leave an orphaned row; on enqueue failure nothing
				// was queued and the row is removed again, but it stays when
				// only the APNs notification failed since the command is
				// durably queued.
				hostCmd := fleet.HostMDMCommand{HostID: hostID, CommandType: fleet.RefetchAppsCommandUUIDPrefix}
				err = ds.AddHostMDMCommands(ctx, []fleet.HostMDMCommand{hostCmd})
				if err != nil {
					return ctxerr.Wrap(ctx, err, "add host mdm commands")
				}

				err = commander.InstalledApplicationList(ctx, []string{installedAppResult.HostUUID()}, fleet.RefetchAppsCommandUUID(), false)
				if err != nil {
					if _, isNotifErr := errors.AsType[*apple_mdm.NotificationFailedError](err); !isNotifErr {
						if rmErr := ds.RemoveHostMDMCommand(ctx, hostCmd); rmErr != nil {
							logger.ErrorContext(ctx, "untrack refetch apps command after enqueue failure",
								"err", rmErr, "host_id", hostID)
						}
					}
					return ctxerr.Wrap(ctx, err, "refetch apps with MDM")
				}
			}
		}

		// If we get here, we're in a terminal state, so we can remove the verify command.
		return ctxerr.Wrap(
			ctx,
			ds.RemoveHostMDMCommand(ctx, fleet.HostMDMCommand{CommandType: fleet.VerifySoftwareInstallVPPPrefix, HostID: hostID}),
			"InstalledApplicationList handler: removing host mdm command",
		)
	}
}

type deviceLocationResult struct {
	raw       []byte
	uuid      string
	hostID    uint
	latitude  float64 `plist:"Latitude"`
	longitude float64 `plist:"Longitude"`
	hostUUID  string
}

func (i *deviceLocationResult) Raw() []byte        { return i.raw }
func (i *deviceLocationResult) UUID() string       { return i.uuid }
func (i *deviceLocationResult) HostUUID() string   { return i.hostUUID }
func (i *deviceLocationResult) HostID() uint       { return i.hostID }
func (i *deviceLocationResult) Latitude() float64  { return i.latitude }
func (i *deviceLocationResult) Longitude() float64 { return i.longitude }

type DeviceLocationResult interface {
	fleet.MDMCommandResults
	HostID() uint
	Latitude() float64
	Longitude() float64
}

func NewDeviceLocationResult(result *mdm.CommandResults, hostID uint) (DeviceLocationResult, error) {
	ret := &deviceLocationResult{
		hostID: hostID,
	}

	// parse results
	var deviceLocResult struct {
		Latitude  float64 `plist:"Latitude"`
		Longitude float64 `plist:"Longitude"`
	}

	if err := plist.Unmarshal(result.Raw, &deviceLocResult); err != nil {
		return nil, fmt.Errorf("device location command result: xml unmarshal: %w", err)
	}

	ret.latitude = deviceLocResult.Latitude
	ret.longitude = deviceLocResult.Longitude

	return ret, nil
}

func NewDeviceLocationResultsHandler(
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger *slog.Logger,
) fleet.MDMCommandResultsHandler {
	return func(ctx context.Context, commandResults fleet.MDMCommandResults) error {
		deviceLocResult, ok := commandResults.(DeviceLocationResult)
		if !ok {
			return ctxerr.New(ctx, "unexpected results type")
		}

		err := ds.InsertHostLocationData(ctx, fleet.HostLocationData{
			HostID:    deviceLocResult.HostID(),
			Latitude:  deviceLocResult.Latitude(),
			Longitude: deviceLocResult.Longitude(),
		})
		return ctxerr.Wrap(ctx, err, "device location command result: insert host location data")
	}
}

///////////////////////////////////////////////////////////////////////////////
// Apple MDM Recovery Lock Password

// recoveryLockResult wraps mdm.CommandResults to implement fleet.MDMCommandResults
type recoveryLockResult struct {
	cmdResult *mdm.CommandResults
}

func (r *recoveryLockResult) Raw() []byte                  { return r.cmdResult.Raw }
func (r *recoveryLockResult) UUID() string                 { return r.cmdResult.CommandUUID }
func (r *recoveryLockResult) HostUUID() string             { return r.cmdResult.UDID } // SetRecoveryLock is device-only, UDID is always present
func (r *recoveryLockResult) CmdStatus() string            { return r.cmdResult.Status }
func (r *recoveryLockResult) ErrorChain() []mdm.ErrorChain { return r.cmdResult.ErrorChain }

// passwordVerified reports the device's verdict on a VerifyRecoveryLock. The device
// acknowledges the command whether or not the password matched, and carries the verdict in
// the payload instead:
//
//	<key>PasswordVerified</key>
//	<false/>
//
// Apple requires the key in the response, so an absent one is not read as a pass: claiming
// a lock is verified when the device never said so is the failure mode worth avoiding.
func (r *recoveryLockResult) passwordVerified() (bool, error) {
	var resp struct {
		PasswordVerified bool `plist:"PasswordVerified"`
	}
	if err := plist.Unmarshal(r.cmdResult.Raw, &resp); err != nil {
		return false, fmt.Errorf("verify recovery lock command result: xml unmarshal: %w", err)
	}
	return resp.PasswordVerified, nil
}

// NewRecoveryLockResult wraps an mdm.CommandResults to implement fleet.MDMCommandResults
func NewRecoveryLockResult(cmdResult *mdm.CommandResults) fleet.MDMCommandResults {
	return &recoveryLockResult{cmdResult: cmdResult}
}

const maxRecoveryLockRetries = 2 // 3 attempts total (initial + 2 retries)

// NewSetRecoveryLockResultsHandler processes SetRecoveryLock command results.
// It mainly enqueues the verify step, or attempt retry if non terminal failure.
func NewSetRecoveryLockResultsHandler(
	ds fleet.Datastore,
	logger *slog.Logger,
	commander *apple_mdm.MDMAppleCommander,
) fleet.MDMCommandResultsHandler {
	return func(ctx context.Context, results fleet.MDMCommandResults) error {
		// Get the underlying result to access status and error chain
		rlResult, ok := results.(*recoveryLockResult)
		if !ok {
			return ctxerr.New(ctx, "SetRecoveryLock handler: unexpected results type")
		}

		hostUUID := results.HostUUID()

		pendingRecoveryLock, err := ds.GetPendingRecoveryLock(ctx, hostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: get pending recovery lock")
		}
		if pendingRecoveryLock == nil {
			// no-op the result if there is no pending recovery lock
			return nil
		}
		if pendingRecoveryLock.PendingSetCommandUUID == nil || *pendingRecoveryLock.PendingSetCommandUUID != results.UUID() {
			// no-op the result if the pending set command UUID doesn't match the current result
			return nil
		}

		logger.DebugContext(ctx, "SetRecoveryLock command result received",
			"host_uuid", hostUUID,
			"command_uuid", results.UUID(),
			"status", rlResult.CmdStatus(),
			"operation_type", pendingRecoveryLock.OperationType,
		)

		switch rlResult.CmdStatus() {
		case fleet.MDMAppleStatusAcknowledged:
			// PROMOTE TO VERIFYING and issue VerifyRecoveryLock with empty password
			pendingVerifyCmdUUID := uuid.NewString()
			logger.InfoContext(ctx, "acknowledged recovery lock, promoting to verifying",
				"host_uuid", hostUUID,
				"command_uuid", results.UUID(),
				"verify_command_uuid", pendingVerifyCmdUUID,
			)
			if err := ds.SetRecoveryLockVerifying(ctx, hostUUID, results.UUID(), pendingVerifyCmdUUID); err != nil {
				return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: set recovery lock verifying")
			}

			if pendingRecoveryLock.OperationType == fleet.MDMOperationTypeRemove {
				if err := commander.VerifyClearRecoveryLock(ctx, hostUUID, pendingVerifyCmdUUID); err != nil {
					if apnsErr, ok := errors.AsType[*apple_mdm.APNSDeliveryError](err); ok {
						// Do not fail on APNS push failures.
						logger.WarnContext(ctx, "VerifyClearRecoveryLock command enqueued but APNs push failed",
							"host_uuid", hostUUID,
							"command_uuid", pendingVerifyCmdUUID,
							"error", apnsErr,
						)
						return nil
					}

					// Reset to retry on actual cmd enqueue failures
					if err := ds.RetryRecoveryLock(ctx, hostUUID, pendingVerifyCmdUUID); err != nil {
						return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: reset recovery lock for retry")
					}
					return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: verify clear recovery lock")
				}
			} else {
				if err := commander.VerifyRecoveryLock(ctx, hostUUID, pendingVerifyCmdUUID); err != nil {
					if apnsErr, ok := errors.AsType[*apple_mdm.APNSDeliveryError](err); ok {
						// Do not fail on APNS push failures.
						logger.WarnContext(ctx, "VerifyRecoveryLock command enqueued but APNs push failed",
							"host_uuid", hostUUID,
							"command_uuid", pendingVerifyCmdUUID,
							"error", apnsErr,
						)
						return nil
					}
					// Reset to retry on actual cmd enqueue failures
					if err := ds.RetryRecoveryLock(ctx, hostUUID, pendingVerifyCmdUUID); err != nil {
						return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: reset recovery lock for retry")
					}
					return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: verify recovery lock")
				}
			}

		case fleet.MDMAppleStatusError, fleet.MDMAppleStatusCommandFormatError:
			errorMsg := apple_mdm.FmtErrorChain(rlResult.cmdResult.ErrorChain)
			if errorMsg == "" {
				// If no specific error message is available, provide a generic one based on the operation type
				if pendingRecoveryLock.OperationType == fleet.MDMOperationTypeRemove {
					errorMsg = "ClearRecoveryLock command failed"
				} else {
					errorMsg = "SetRecoveryLock command failed"
				}
			}

			if apple_mdm.IsRecoveryLockPasswordMismatchError(rlResult.cmdResult.ErrorChain) && pendingRecoveryLock.HasCurrentPassword {
				// The device kept a lock that the new password can't replace, but Fleet has a
				// password on file — verify that one rather than failing outright. Common after a
				// re-enrollment: the row is soft-deleted, the cron re-SETs as if the host were
				// fresh, but the device never actually dropped the lock.
				pendingVerifyCmdUUID := uuid.NewString()
				if err := ds.SetRecoveryLockVerifyingLastKnownPassword(ctx, hostUUID, results.UUID(), pendingVerifyCmdUUID); err != nil {
					return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: set recovery lock verifying with last known password")
				}
				if err := commander.VerifyRecoveryLockLastKnownPassword(ctx, hostUUID, pendingVerifyCmdUUID); err != nil {
					if apnsErr, ok := errors.AsType[*apple_mdm.APNSDeliveryError](err); ok {
						// Do not fail on APNS push failures.
						logger.WarnContext(ctx, "VerifyRecoveryLock with last known password command enqueued but APNs push failed",
							"host_uuid", hostUUID,
							"command_uuid", pendingVerifyCmdUUID,
							"error", apnsErr,
						)
						return nil
					}
					// Reset to retry on actual cmd enqueue failures
					if err := ds.RetryRecoveryLock(ctx, hostUUID, pendingVerifyCmdUUID); err != nil {
						return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock with last known password handler: reset recovery lock for retry")
					}
					return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock with last known password handler: verify recovery lock")
				}

				return nil
			}

			// Failed Clear operations aren't retried
			// Command format errors are terminal - command is malformed and won't succeed on retry.
			// Password mismatch errors are also terminal - requires admin intervention.
			// Retries exhausted - treat as terminal error.
			if pendingRecoveryLock.OperationType == fleet.MDMOperationTypeRemove ||
				rlResult.CmdStatus() == fleet.MDMAppleStatusCommandFormatError ||
				apple_mdm.IsRecoveryLockPasswordMismatchError(rlResult.cmdResult.ErrorChain) ||
				pendingRecoveryLock.Retries >= maxRecoveryLockRetries {
				if err := ds.SetRecoveryLockFailed(ctx, hostUUID, results.UUID(), errorMsg); err != nil {
					return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: set recovery lock failed")
				}
				logger.WarnContext(ctx, "RecoveryLock failed with terminal error",
					"host_uuid", hostUUID,
					"error", errorMsg,
				)
				return nil
			}
			// Transient error - reset to install/verified for retry on next cron cycle
			if err := ds.RetryRecoveryLock(ctx, hostUUID, results.UUID()); err != nil {
				return ctxerr.Wrap(ctx, err, "SetRecoveryLock handler: reset recovery lock for retry")
			}
			logger.InfoContext(ctx, "RecoveryLock failed with transient error, will retry",
				"host_uuid", hostUUID,
				"error", errorMsg,
			)
		}

		return nil
	}
}

func NewVerifyRecoveryLockResultsHandler(
	ds fleet.Datastore,
	logger *slog.Logger,
	commander *apple_mdm.MDMAppleCommander,
	newActivityFn fleet.NewActivityFunc,
) fleet.MDMCommandResultsHandler {
	return func(ctx context.Context, results fleet.MDMCommandResults) error {
		logger.DebugContext(ctx, "VerifyRecoveryLock results received",
			"host_uuid", results.HostUUID(),
			"command_uuid", results.UUID(),
		)

		rlResult, ok := results.(*recoveryLockResult)
		if !ok {
			return ctxerr.New(ctx, "VerifyRecoveryLock handler: unexpected results type")
		}

		pending, err := ds.GetPendingRecoveryLock(ctx, results.HostUUID())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: get pending recovery lock failed")
		}

		if pending == nil || pending.PendingVerifyCommandUUID == nil || *pending.PendingVerifyCommandUUID != results.UUID() {
			logger.DebugContext(ctx, "VerifyRecoveryLock handler: no pending recovery lock or command UUID mismatch",
				"host_uuid", results.HostUUID(),
				"command_uuid", results.UUID(),
			)
			return nil
		}

		switch rlResult.CmdStatus() {
		case fleet.MDMAppleStatusAcknowledged:
			verified, err := rlResult.passwordVerified()
			if err != nil {
				return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: read password verified")
			}
			if !verified {
				// The device answered, and the answer is no: it holds a lock that isn't the
				// password we sent. Nothing to retry, the next attempt gets the same answer.
				errorMsg := "Device reported that the recovery lock password does not match"
				if pending.OperationType == fleet.MDMOperationTypeRemove {
					errorMsg = "Device reported that a recovery lock password is still set"
				}
				if err := ds.SetRecoveryLockFailed(ctx, results.HostUUID(), results.UUID(), errorMsg); err != nil {
					return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: set recovery lock failed")
				}
				logger.WarnContext(ctx, "VerifyRecoveryLock acknowledged but the device did not verify the password",
					"host_uuid", results.HostUUID(),
					"command_uuid", results.UUID(),
					"operation_type", pending.OperationType,
				)
				return nil
			}

			if pending.OperationType == fleet.MDMOperationTypeInstall {
				// verified the last set password, mark as verified and promote
				if err := ds.SetRecoveryLockVerified(ctx, results.HostUUID(), results.UUID()); err != nil {
					return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: set recovery lock verified failed")
				}
			} else {
				if err := ds.DeleteHostRecoveryLockPassword(ctx, results.HostUUID(), results.UUID()); err != nil {
					return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: delete host recovery lock password failed")
				}
			}
		case fleet.MDMAppleStatusCommandFormatError, fleet.MDMAppleStatusError:
			shouldRetry := pending.Retries < maxRecoveryLockRetries
			appleErr := apple_mdm.FmtErrorChain(rlResult.ErrorChain())

			// Do not retry on:
			// - Clear operation
			// - Command format error
			// - Recovery lock password not set
			// - Retries exhausted
			if pending.OperationType == fleet.MDMOperationTypeRemove ||
				rlResult.CmdStatus() == fleet.MDMAppleStatusCommandFormatError ||
				apple_mdm.IsRecoveryLockPasswordNotSetError(rlResult.ErrorChain()) ||
				!shouldRetry {

				// Terminal errors or retries exhausted - no point in retrying
				if err := ds.SetRecoveryLockFailed(ctx, results.HostUUID(), results.UUID(), appleErr); err != nil {
					return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: set recovery lock failed")
				}

				logger.WarnContext(ctx, "VerifyRecoveryLock command failed",
					"host_uuid", results.HostUUID(),
					"error", appleErr,
					"retries", pending.Retries,
				)
				return nil
			}

			// Re-issue the verify, not the set — see RetryRecoveryLockVerify.
			newVerifyCmdUUID := uuid.NewString()
			logger.InfoContext(ctx, "VerifyRecoveryLock command retryable error, re-issuing verify",
				"host_uuid", results.HostUUID(),
				"error", appleErr,
				"verify_command_uuid", newVerifyCmdUUID,
			)
			if err := ds.RetryRecoveryLockVerify(ctx, results.HostUUID(), results.UUID(), newVerifyCmdUUID); err != nil {
				return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: retry recovery lock verify failed")
			}
			if err := commander.VerifyRecoveryLock(ctx, results.HostUUID(), newVerifyCmdUUID); err != nil {
				if apnsErr, ok := errors.AsType[*apple_mdm.APNSDeliveryError](err); ok {
					// Command is persisted; the device picks it up on its next checkin.
					logger.WarnContext(ctx, "VerifyRecoveryLock retry enqueued but APNs push failed",
						"host_uuid", results.HostUUID(),
						"command_uuid", newVerifyCmdUUID,
						"error", apnsErr,
					)
					return nil
				}
				return ctxerr.Wrap(ctx, err, "VerifyRecoveryLock handler: re-enqueue verify recovery lock")
			}
			return nil
		}

		// Log a set activity if this was an install operation and the host did not have a current password therefore it wasn't rotated
		// rotation activity logs happen at rotation enqueuement
		if !pending.HasCurrentPassword && pending.OperationType == fleet.MDMOperationTypeInstall {

			host, err := ds.HostLiteByIdentifier(ctx, results.HostUUID())
			if err != nil || host == nil {
				logger.WarnContext(ctx, "VerifyRecoveryLock handler: failed to get host for activity logging",
					"host_uuid", results.HostUUID(),
					"err", err,
				)
			} else {
				// Log the activity only if we could identify the host (fleet-initiated via WasFromAutomation)
				if err := newActivityFn(ctx, nil, fleet.ActivityTypeSetHostRecoveryLockPassword{
					HostID:          host.ID,
					HostDisplayName: host.DisplayName(),
				}); err != nil {
					logger.WarnContext(ctx, "VerifyRecoveryLock handler: failed to create activity",
						"host_uuid", results.HostUUID(),
						"err", err,
					)
				}
			}
		}

		return nil
	}
}
