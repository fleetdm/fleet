package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleetdbase"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/appmanifest"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
)

// Name of the Apple MDM job as registered in the worker. Note that although it
// is a single job, it can process a number of different-but-related tasks,
// identified by the Task field in the job's payload.
const appleMDMJobName = "apple_mdm"

type AppleMDMTask string

// List of supported tasks.
const (
	AppleMDMPostDEPEnrollmentTask    AppleMDMTask = "post_dep_enrollment"
	AppleMDMPostManualEnrollmentTask AppleMDMTask = "post_manual_enrollment"
	// PostDEPReleaseDevice is not enqueued anymore for macOS but remains for
	// backward compatibility (processing existing jobs after a fleet upgrade)
	// and for ios/ipados. Macs are now released via the swift dialog UI of the
	// setup experience flow.
	AppleMDMPostDEPReleaseDeviceTask AppleMDMTask = "post_dep_release_device"
)

// AppleMDM is the job processor for the apple_mdm job.
type AppleMDM struct {
	Datastore fleet.Datastore
	Log       kitlog.Logger
	Commander *apple_mdm.MDMAppleCommander
}

// Name returns the name of the job.
func (a *AppleMDM) Name() string {
	return appleMDMJobName
}

// appleMDMArgs is the payload for the Apple MDM job.
type appleMDMArgs struct {
	Task                   AppleMDMTask `json:"task"`
	HostUUID               string       `json:"host_uuid"`
	TeamID                 *uint        `json:"team_id,omitempty"`
	EnrollReference        string       `json:"enroll_reference,omitempty"`
	EnrollmentCommands     []string     `json:"enrollment_commands,omitempty"`
	Platform               string       `json:"platform,omitempty"`
	UseWorkerDeviceRelease bool         `json:"use_worker_device_release,omitempty"`
}

// Run executes the apple_mdm job.
func (a *AppleMDM) Run(ctx context.Context, argsJSON json.RawMessage) error {
	appCfg, err := a.Datastore.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving app config")
	}
	if !appCfg.MDM.EnabledAndConfigured || a.Commander == nil {
		return nil
	}

	var args appleMDMArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case AppleMDMPostDEPEnrollmentTask:
		err := a.runPostDEPEnrollment(ctx, args)
		return ctxerr.Wrap(ctx, err, "running post Apple DEP enrollment task")

	case AppleMDMPostManualEnrollmentTask:
		err := a.runPostManualEnrollment(ctx, args)
		return ctxerr.Wrap(ctx, err, "running post Apple manual enrollment task")

	case AppleMDMPostDEPReleaseDeviceTask:
		err := a.runPostDEPReleaseDevice(ctx, args)
		return ctxerr.Wrap(ctx, err, "running post Apple DEP release device task")

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func isMacOS(platform string) bool {
	// For backwards compatibility, we assume empty platform in job arguments is macOS.
	return platform == "" ||
		platform == "darwin"
}

func (a *AppleMDM) runPostManualEnrollment(ctx context.Context, args appleMDMArgs) error {
	if isMacOS(args.Platform) {
		if _, err := a.installFleetd(ctx, args.HostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "installing post-enrollment packages")
		}
	}

	return nil
}

func (a *AppleMDM) runPostDEPEnrollment(ctx context.Context, args appleMDMArgs) error {
	var awaitCmdUUIDs []string

	if isMacOS(args.Platform) {
		fleetdCmdUUID, err := a.installFleetd(ctx, args.HostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "installing post-enrollment packages")
		}
		awaitCmdUUIDs = append(awaitCmdUUIDs, fleetdCmdUUID)

		bootstrapCmdUUID, err := a.installBootstrapPackage(ctx, args.HostUUID, args.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "installing post-enrollment packages")
		}
		if bootstrapCmdUUID != "" {
			awaitCmdUUIDs = append(awaitCmdUUIDs, bootstrapCmdUUID)
		}
	}

	if ref := args.EnrollReference; ref != "" {
		a.Log.Log("info", "got an enroll_reference", "host_uuid", args.HostUUID, "ref", ref)
		appCfg, err := a.Datastore.AppConfig(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting app config")
		}

		acct, err := a.Datastore.GetMDMIdPAccountByUUID(ctx, ref)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "getting idp account details for enroll reference %s", ref)
		}

		ssoEnabled := appCfg.MDM.MacOSSetup.EnableEndUserAuthentication
		if args.TeamID != nil {
			team, err := a.Datastore.Team(ctx, *args.TeamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "fetch team to send AccountConfiguration")
			}
			ssoEnabled = team.Config.MDM.MacOSSetup.EnableEndUserAuthentication
		}

		if ssoEnabled {
			a.Log.Log("info", "setting username and fullname", "host_uuid", args.HostUUID)
			cmdUUID := uuid.New().String()
			if err := a.Commander.AccountConfiguration(
				ctx,
				[]string{args.HostUUID},
				cmdUUID,
				acct.Fullname,
				acct.Username,
			); err != nil {
				return ctxerr.Wrap(ctx, err, "sending AccountConfiguration command")
			}
			awaitCmdUUIDs = append(awaitCmdUUIDs, cmdUUID)
		}
	}

	// proceed to release the device if it is not a macos, as those are released
	// via the setup experience flow, or if we were told to use the worker based
	// release.
	if !isMacOS(args.Platform) || args.UseWorkerDeviceRelease {
		var manualRelease bool
		if args.TeamID == nil {
			ac, err := a.Datastore.AppConfig(ctx)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get AppConfig to read enable_release_device_manually")
			}
			manualRelease = ac.MDM.MacOSSetup.EnableReleaseDeviceManually.Value
		} else {
			tm, err := a.Datastore.Team(ctx, *args.TeamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get Team to read enable_release_device_manually")
			}
			manualRelease = tm.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value
		}

		if !manualRelease {
			// send all command uuids for the commands sent here during post-DEP
			// enrollment and enqueue a job to look for the status of those commands to
			// be final and same for MDM profiles of that host; it means the DEP
			// enrollment process is done and the device can be released.
			if err := QueueAppleMDMJob(ctx, a.Datastore, a.Log, AppleMDMPostDEPReleaseDeviceTask,
				args.HostUUID, args.Platform, args.TeamID, args.EnrollReference, false, awaitCmdUUIDs...); err != nil {
				return ctxerr.Wrap(ctx, err, "queue Apple Post-DEP release device job")
			}
		}
	}

	return nil
}

// This job is deprecated for macos because releasing devices is now done via
// the orbit endpoint /setup_experience/status that is polled by a swift dialog
// UI window during the setup process (unless there are no setup experience
// items, in which case this worker job is used), and automatically releases
// the device once all pending setup tasks are done. However, it must remain
// implemented for iOS and iPadOS and in case there are such jobs to process
// after a Fleet migration to a new version.
func (a *AppleMDM) runPostDEPReleaseDevice(ctx context.Context, args appleMDMArgs) error {
	// Edge cases:
	//   - if the device goes offline for a long time, should we go ahead and
	//   release after a while?
	//   - if some commands/profiles failed (a final state), should we go ahead
	//   and release?
	//   - if the device keeps moving team, or profiles keep being added/removed
	//   from its team, it's possible that its profiles will never settle and
	//   always have pending statuses. Same as going offline, should we release
	//   after a while?
	//
	// We opted "yes" to all those, and we want to release after a few minutes,
	// not hours, so we'll allow only a couple retries.

	level.Debug(a.Log).Log(
		"task", "runPostDEPReleaseDevice",
		"msg", fmt.Sprintf("awaiting commands %v and profiles to settle for host %s", args.EnrollmentCommands, args.HostUUID),
	)

	if retryNum, _ := ctx.Value(retryNumberCtxKey).(int); retryNum > 2 {
		// give up and release the device
		a.Log.Log("info", "releasing device after too many attempts", "host_uuid", args.HostUUID, "retries", retryNum)
		if err := a.Commander.DeviceConfigured(ctx, args.HostUUID, uuid.NewString()); err != nil {
			return ctxerr.Wrapf(ctx, err, "failed to enqueue DeviceConfigured command after %d retries", retryNum)
		}
		return nil
	}

	for _, cmdUUID := range args.EnrollmentCommands {
		if cmdUUID == "" {
			continue
		}

		res, err := a.Datastore.GetMDMAppleCommandResults(ctx, cmdUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "failed to get MDM command results")
		}

		var completed bool
		for _, r := range res {
			// succeeded or failed, it is done (final state)
			if r.Status == fleet.MDMAppleStatusAcknowledged || r.Status == fleet.MDMAppleStatusError ||
				r.Status == fleet.MDMAppleStatusCommandFormatError {
				completed = true
				break
			}
		}

		if !completed {
			// DEP enrollment commands are not done being delivered to that device,
			// cannot release it now.
			return fmt.Errorf("device not ready for release, still awaiting result for command %s, will retry", cmdUUID)
		}
		level.Debug(a.Log).Log(
			"task", "runPostDEPReleaseDevice",
			"msg", fmt.Sprintf("command %s has completed", cmdUUID),
		)
	}

	// all DEP-enrollment commands are done, check the host's profiles
	profs, err := a.Datastore.GetHostMDMAppleProfiles(ctx, args.HostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get host MDM profiles")
	}
	for _, prof := range profs {
		// NOTE: DDM profiles (declarations) are ignored because while a device is
		// awaiting to be released, it cannot process a DDM session (at least
		// that's what we noticed during testing).
		if strings.HasPrefix(prof.ProfileUUID, fleet.MDMAppleDeclarationUUIDPrefix) {
			continue
		}

		// if it has any pending profiles, then its profiles are not done being
		// delivered (installed or removed).
		if prof.Status == nil || *prof.Status == fleet.MDMDeliveryPending {
			return fmt.Errorf("device not ready for release, profile %s is still pending, will retry", prof.Identifier)
		}
		level.Debug(a.Log).Log(
			"task", "runPostDEPReleaseDevice",
			"msg", fmt.Sprintf("profile %s has been deployed", prof.Identifier),
		)
	}

	// release the device
	a.Log.Log("info", "releasing device, all DEP enrollment commands and profiles have completed", "host_uuid", args.HostUUID)
	if err := a.Commander.DeviceConfigured(ctx, args.HostUUID, uuid.NewString()); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to enqueue DeviceConfigured command")
	}
	return nil
}

func (a *AppleMDM) installFleetd(ctx context.Context, hostUUID string) (string, error) {
	manifestURL := fleetdbase.GetPKGManifestURL()
	cmdUUID := uuid.New().String()
	if err := a.Commander.InstallEnterpriseApplication(ctx, []string{hostUUID}, cmdUUID, manifestURL); err != nil {
		return "", err
	}
	a.Log.Log("info", "sent command to install fleetd", "host_uuid", hostUUID)
	return cmdUUID, nil
}

func (a *AppleMDM) installBootstrapPackage(ctx context.Context, hostUUID string, teamID *uint) (string, error) {
	// GetMDMAppleBootstrapPackageMeta expects team id 0 for no team
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}
	meta, err := a.Datastore.GetMDMAppleBootstrapPackageMeta(ctx, tmID)
	if err != nil {
		var nfe fleet.NotFoundError
		if errors.As(err, &nfe) {
			a.Log.Log("info", "unable to find a bootstrap package for DEP enrolled device, skipping installation", "host_uuid", hostUUID)
			return "", nil
		}

		return "", err
	}

	appCfg, err := a.Datastore.AppConfig(ctx)
	if err != nil {
		return "", err
	}

	url, err := meta.URL(appCfg.MDMUrl())
	if err != nil {
		return "", err
	}

	manifest := appmanifest.NewFromSha(meta.Sha256, url)
	cmdUUID := uuid.New().String()
	err = a.Commander.InstallEnterpriseApplicationWithEmbeddedManifest(ctx, []string{hostUUID}, cmdUUID, manifest)
	if err != nil {
		return "", err
	}
	err = a.Datastore.RecordHostBootstrapPackage(ctx, cmdUUID, hostUUID)
	if err != nil {
		return "", err
	}
	a.Log.Log("info", "sent command to install bootstrap package", "host_uuid", hostUUID)
	return cmdUUID, nil
}

// QueueAppleMDMJob queues a apple_mdm job for one of the supported tasks, to
// be processed asynchronously via the worker.
func QueueAppleMDMJob(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	task AppleMDMTask,
	hostUUID string,
	platform string,
	teamID *uint,
	enrollReference string,
	useWorkerDeviceRelease bool,
	enrollmentCommandUUIDs ...string,
) error {
	attrs := []interface{}{
		"enabled", "true",
		appleMDMJobName, task,
		"host_uuid", hostUUID,
		"platform", platform,
		"with_enroll_reference", enrollReference != "",
	}
	if teamID != nil {
		attrs = append(attrs, "team_id", *teamID)
	}
	if len(enrollmentCommandUUIDs) > 0 {
		attrs = append(attrs, "enrollment_commands", fmt.Sprintf("%v", enrollmentCommandUUIDs))
	}
	level.Info(logger).Log(attrs...)

	args := &appleMDMArgs{
		Task:                   task,
		HostUUID:               hostUUID,
		TeamID:                 teamID,
		EnrollReference:        enrollReference,
		EnrollmentCommands:     enrollmentCommandUUIDs,
		Platform:               platform,
		UseWorkerDeviceRelease: useWorkerDeviceRelease,
	}

	// the release device task is always added with a delay
	var delay time.Duration
	if task == AppleMDMPostDEPReleaseDeviceTask {
		delay = 30 * time.Second
	}
	job, err := QueueJobWithDelay(ctx, ds, appleMDMJobName, args, delay)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
