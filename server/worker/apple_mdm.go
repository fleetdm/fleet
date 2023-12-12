package worker

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/appmanifest"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
)

// Name of the Apple MDM job as registered in the worker. Note that although it
// is a single job, it can process a number of different-but-related tasks,
// identified by the Task field in the job's payload.
const appleMDMJobName = "apple_mdm"

type AppleMDMTask string

// List of supported tasks.
const (
	AppleMDMPostDEPEnrollmentTask AppleMDMTask = "post_dep_enrollment"
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
	Task            AppleMDMTask `json:"task"`
	HostUUID        string       `json:"host_uuid"`
	TeamID          *uint        `json:"team_id,omitempty"`
	EnrollReference string       `json:"enroll_reference,omitempty"`
}

// Run executes the apple_mdm job.
func (a *AppleMDM) Run(ctx context.Context, argsJSON json.RawMessage) error {
	// if Commander is nil, then mdm is not enabled, so just return without
	// error so we clean up any pending jobs.
	if a.Commander == nil {
		return nil
	}

	var args appleMDMArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case AppleMDMPostDEPEnrollmentTask:
		return a.runPostDEPEnrollment(ctx, args)
	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func (a *AppleMDM) runPostDEPEnrollment(ctx context.Context, args appleMDMArgs) error {
	if err := a.installEnrollmentPackages(ctx, args.HostUUID, args.TeamID); err != nil {
		return ctxerr.Wrap(ctx, err, "installing post-enrollment packages")
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
			if err := a.Commander.AccountConfiguration(
				ctx,
				[]string{args.HostUUID},
				uuid.New().String(),
				acct.Fullname,
				acct.Username,
			); err != nil {
				return ctxerr.Wrap(ctx, err, "sending AccountConfiguration command")
			}
		}
	}
	return nil
}

func (a *AppleMDM) installEnrollmentPackages(ctx context.Context, hostUUID string, teamID *uint) error {
	cmdUUID := uuid.New().String()
	if err := a.Commander.InstallEnterpriseApplication(ctx, []string{hostUUID}, cmdUUID, apple_mdm.FleetdPublicManifestURL); err != nil {
		return err
	}
	a.Log.Log("info", "sent command to install fleetd", "host_uuid", hostUUID)

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
			return nil
		}

		return err
	}

	appCfg, err := a.Datastore.AppConfig(ctx)
	if err != nil {
		return err
	}

	url, err := meta.URL(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return err
	}

	manifest := appmanifest.NewFromSha(meta.Sha256, url)
	cmdUUID = uuid.New().String()
	err = a.Commander.InstallEnterpriseApplicationWithEmbeddedManifest(ctx, []string{hostUUID}, cmdUUID, manifest)
	if err != nil {
		return err
	}
	err = a.Datastore.RecordHostBootstrapPackage(ctx, cmdUUID, hostUUID)
	if err != nil {
		return err
	}
	a.Log.Log("info", "sent command to install bootstrap package", "host_uuid", hostUUID)
	return nil
}

// QueueAppleMDMJob queues a apple_mdm job for one of the supported tasks, to
// be processed asynchronously via the worker.
func QueueAppleMDMJob(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	task AppleMDMTask,
	hostUUID string,
	teamID *uint,
	enrollReference string,
) error {
	attrs := []interface{}{
		"enabled", "true",
		appleMDMJobName, task,
		"host_uuid", hostUUID,
		"with_enroll_reference", enrollReference != "",
	}
	if teamID != nil {
		attrs = append(attrs, "team_id", *teamID)
	}
	level.Info(logger).Log(attrs...)

	args := &appleMDMArgs{
		Task:            task,
		HostUUID:        hostUUID,
		TeamID:          teamID,
		EnrollReference: enrollReference,
	}
	job, err := QueueJob(ctx, ds, appleMDMJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
