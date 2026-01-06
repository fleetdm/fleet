package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
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
	logger kitlog.Logger,
	verifyTimeout, verifyRequestDelay time.Duration,
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
			level.Warn(logger).Log("msg", "no apple MDM installs found for host", "host_uuid", installedAppResult.HostUUID(), "verification_command_uuid", installedAppResult.UUID())
			return nil
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
			activityFn func(ctx context.Context, results *mdm.CommandResults, fromSetupExp bool) (*fleet.User, fleet.ActivityDetails, error)
		}

		var poll, shouldRefetch bool
		setStatusForExpectedInstall := func(
			expectedInstall *fleet.HostVPPSoftwareInstall,
			setter installStatusSetter,
		) error {
			// If we don't find the app in the result, then we need to poll for it (within the timeout).
			appFromResult := installsByBundleID[expectedInstall.BundleIdentifier]

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
				return nil
			}

			// this might be a setup experience VPP install, so we'll try to update setup experience status
			var fromSetupExperience bool
			if updated, err := maybeUpdateSetupExperienceStatus(ctx, ds, fleet.SetupExperienceVPPInstallResult{
				HostUUID:      installedAppResult.HostUUID(),
				CommandUUID:   expectedInstall.InstallCommandUUID,
				CommandStatus: terminalStatus,
			}, true); err != nil {
				return ctxerr.Wrap(ctx, err, "updating setup experience status from VPP install result")
			} else if updated {
				fromSetupExperience = true
				level.Debug(logger).Log("msg", "setup experience VPP install result updated", "host_uuid", installedAppResult.HostUUID(), "execution_id", expectedInstall.InstallCommandUUID)
			}

			// create an activity for installing only if we're in a terminal state
			user, act, err := setter.activityFn(ctx, &mdm.CommandResults{CommandUUID: expectedInstall.InstallCommandUUID, Status: terminalStatus}, fromSetupExperience)
			if err != nil {
				if fleet.IsNotFound(err) {
					// Then this isn't an MDM-based install, so no activity generated
					return nil
				}

				return ctxerr.Wrap(ctx, err, "fetching data for installed app store app activity")
			}

			if err := newActivity(ctx, user, act, ds, logger); err != nil {
				return ctxerr.Wrap(ctx, err, "creating activity for installed app store app")
			}

			return nil
		}

		for _, expectedInstall := range expectedVPPInstalls {
			setter := installStatusSetter{
				ds.SetVPPInstallAsVerified,
				ds.SetVPPInstallAsFailed,
				func(ctx context.Context, results *mdm.CommandResults, fromSetupExp bool) (*fleet.User, fleet.ActivityDetails, error) {
					user, act, err := ds.GetPastActivityDataForVPPAppInstall(ctx, results)
					if err != nil {
						return nil, nil, err
					}

					act.FromSetupExperience = fromSetupExp

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
				func(ctx context.Context, results *mdm.CommandResults, _ bool) (*fleet.User, fleet.ActivityDetails, error) {
					return ds.GetPastActivityDataForInHouseAppInstall(ctx, results)
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
				worker.QueueVPPInstallVerificationJob(ctx, ds, logger, worker.VerifyVPPTask, verifyRequestDelay, installedAppResult.HostUUID(), installedAppResult.UUID()),
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
				err = commander.InstalledApplicationList(ctx, []string{installedAppResult.HostUUID()}, fleet.RefetchAppsCommandUUID(), false)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "refetch apps with MDM")
				}

				err = ds.AddHostMDMCommands(ctx, []fleet.HostMDMCommand{{HostID: hostID, CommandType: fleet.RefetchAppsCommandUUIDPrefix}})
				if err != nil {
					return ctxerr.Wrap(ctx, err, "add host mdm commands")
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
	logger kitlog.Logger,
) fleet.MDMCommandResultsHandler {
	return func(ctx context.Context, commandResults fleet.MDMCommandResults) error {
		deviceLocResult, ok := commandResults.(DeviceLocationResult)
		if !ok {
			return ctxerr.New(ctx, "unexpected results type")
		}

		fmt.Printf("deviceLocResult.Raw(): %v\n", deviceLocResult.Raw())

		err := ds.InsertHostLocationData(ctx, fleet.HostLocationData{
			HostID:    deviceLocResult.HostID(),
			Latitude:  deviceLocResult.Latitude(),
			Longitude: deviceLocResult.Longitude(),
		})
		return ctxerr.Wrap(ctx, err, "device location command result: insert host location data")
	}
}
