package mdmlifecycle

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// HostAction is a supported MDM lifecycle action that can be performed on a
// host.
type HostAction string

const (
	// HostActionTurnOn performs tasks right after a host turns on MDM.
	HostActionTurnOn HostAction = "turn-on"
	// HostActionTurnOn performs tasks right after a host turns off MDM.
	HostActionTurnOff HostAction = "turn-off"
	// HostActionTurnOn perform tasks to reset mdm-related information.
	HostActionReset HostAction = "reset"
	// HostActionDelete perform tasks to cleanup MDM information when a
	// host is deleted from fleet.
	HostActionDelete HostAction = "delete"
)

// HostOptions are the options that can be provided for an action.
//
// Not all options are required for all actions, each individual action should
// validate that it receives the required information.
type HostOptions struct {
	Action                  HostAction
	Platform                string
	UUID                    string
	HardwareSerial          string
	HardwareModel           string
	EnrollReference         string
	Host                    *fleet.Host
	HasSetupExperienceItems bool
}

// HostLifecycle manages MDM host lifecycle actions
type HostLifecycle struct {
	ds     fleet.Datastore
	logger kitlog.Logger
}

// New creates a new HostLifecycle struct
func New(ds fleet.Datastore, logger kitlog.Logger) *HostLifecycle {
	return &HostLifecycle{
		ds:     ds,
		logger: logger,
	}
}

// Do executes the provided HostAction based on the platform requested
func (t *HostLifecycle) Do(ctx context.Context, opts HostOptions) error {
	switch opts.Platform {
	case "darwin", "ios", "ipados":
		err := t.doDarwin(ctx, opts)
		return ctxerr.Wrapf(ctx, err, "running apple lifecycle action %s", opts.Action)
	case "windows":
		err := t.doWindows(ctx, opts)
		return ctxerr.Wrapf(ctx, err, "running windows lifecycle action %s", opts.Action)
	default:
		return ctxerr.Errorf(ctx, "unsupported platform %s", opts.Platform)
	}
}

func (t *HostLifecycle) doDarwin(ctx context.Context, opts HostOptions) error {
	switch opts.Action {
	case HostActionTurnOn:
		return t.turnOnDarwin(ctx, opts)

	case HostActionTurnOff:
		return t.doWithUUIDValidation(ctx, t.ds.MDMTurnOff, opts)

	case HostActionReset:
		return t.resetDarwin(ctx, opts)

	case HostActionDelete:
		return t.deleteDarwin(ctx, opts)

	default:
		return ctxerr.Errorf(ctx, "unknown action %s", opts.Action)

	}
}

func (t *HostLifecycle) doWindows(ctx context.Context, opts HostOptions) error {
	switch opts.Action {
	case HostActionReset, HostActionTurnOn:
		return t.doWithUUIDValidation(ctx, t.ds.MDMResetEnrollment, opts)

	case HostActionTurnOff:
		return t.doWithUUIDValidation(ctx, t.ds.MDMTurnOff, opts)

	case HostActionDelete:
		return nil

	default:
		return ctxerr.Errorf(ctx, "unknown action %s", opts.Action)
	}
}

type uuidFn func(ctx context.Context, uuid string) error

func (t *HostLifecycle) doWithUUIDValidation(ctx context.Context, action uuidFn, opts HostOptions) error {
	if opts.UUID == "" {
		return ctxerr.New(ctx, "UUID option is required for this action")
	}

	return action(ctx, opts.UUID)
}

func (t *HostLifecycle) resetDarwin(ctx context.Context, opts HostOptions) error {
	if opts.UUID == "" || opts.HardwareSerial == "" || opts.HardwareModel == "" {
		return ctxerr.New(ctx, "UUID, HardwareSerial and HardwareModel options are required for this action")
	}

	host := &fleet.Host{
		UUID:           opts.UUID,
		HardwareSerial: opts.HardwareSerial,
		HardwareModel:  opts.HardwareModel,
		Platform:       opts.Platform,
	}
	if err := t.ds.MDMAppleUpsertHost(ctx, host); err != nil {
		return ctxerr.Wrap(ctx, err, "upserting mdm host")
	}

	err := t.ds.MDMResetEnrollment(ctx, opts.UUID)
	return ctxerr.Wrap(ctx, err, "reset mdm enrollment")
}

func (t *HostLifecycle) turnOnDarwin(ctx context.Context, opts HostOptions) error {
	if opts.UUID == "" {
		return ctxerr.New(ctx, "UUID option is required for this action")
	}

	nanoEnroll, err := t.ds.GetNanoMDMEnrollment(ctx, opts.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving nano enrollment info")
	}

	if nanoEnroll == nil ||
		!nanoEnroll.Enabled ||
		nanoEnroll.Type != "Device" ||
		nanoEnroll.TokenUpdateTally != 1 {
		return nil
	}

	info, err := t.ds.GetHostMDMCheckinInfo(ctx, opts.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting checkin info")
	}

	var tmID *uint
	if info.TeamID != 0 {
		tmID = &info.TeamID
	}

	// TODO: improve this to not enqueue the job if a host that is
	// assigned in ABM is manually enrolling for some reason.
	if info.DEPAssignedToFleet || info.InstalledFromDEP {
		t.logger.Log("info", "queueing post-enroll task for newly enrolled DEP device", "host_uuid", opts.UUID)
		err := worker.QueueAppleMDMJob(
			ctx,
			t.ds,
			t.logger,
			worker.AppleMDMPostDEPEnrollmentTask,
			opts.UUID,
			opts.Platform,
			tmID,
			opts.EnrollReference,
			!opts.HasSetupExperienceItems,
		)
		return ctxerr.Wrap(ctx, err, "queue DEP post-enroll task")
	}

	// manual MDM enrollments
	if !info.InstalledFromDEP {
		if err := worker.QueueAppleMDMJob(
			ctx,
			t.ds,
			t.logger,
			worker.AppleMDMPostManualEnrollmentTask,
			opts.UUID,
			opts.Platform,
			tmID,
			opts.EnrollReference,
			false,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "queue manual post-enroll task")
		}
	}

	return nil
}

func (t *HostLifecycle) deleteDarwin(ctx context.Context, opts HostOptions) error {
	if opts.Host == nil {
		return ctxerr.New(ctx, "a non-nil Host option is required to perform this action")
	}

	// NOTE: deletion of mdm-related tables is handled by the ds.DeleteHost method.

	// Try to immediately restore a host if it's assigned to us in ABM
	if !license.IsPremium(ctx) {
		// only premium tier supports DEP so nothing more to do
		return nil
	}

	ac, err := t.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	} else if !ac.MDM.AppleBMEnabledAndConfigured {
		// if ABM is not enabled and configured, nothing more to do
		return nil
	}

	dep, err := t.ds.GetHostDEPAssignment(ctx, opts.Host.ID)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "get host dep assignment")
	}

	if dep != nil && dep.DeletedAt == nil {
		return t.restorePendingDEPHost(ctx, opts.Host, dep.ABMTokenID)
	}

	// no DEP assignment was found or the DEP assignment was deleted in ABM
	// so nothing more to do
	return nil
}

func (t *HostLifecycle) restorePendingDEPHost(ctx context.Context, host *fleet.Host, abmTokenID *uint) error {
	if abmTokenID == nil {
		return ctxerr.New(ctx, "cannot restore pending dep host without valid ABM token id")
	}

	tmID, err := t.getDefaultTeamForABMToken(ctx, host, *abmTokenID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "restore pending dep host")
	}
	host.TeamID = tmID

	if err := t.ds.RestoreMDMApplePendingDEPHost(ctx, host); err != nil {
		return ctxerr.Wrap(ctx, err, "restore pending dep host")
	}

	if _, err := worker.QueueMacosSetupAssistantJob(ctx, t.ds, t.logger,
		worker.MacosSetupAssistantHostsTransferred, tmID, host.HardwareSerial); err != nil {
		return ctxerr.Wrap(ctx, err, "queue macos setup assistant update profile job")
	}

	return nil
}

func (t *HostLifecycle) getDefaultTeamForABMToken(ctx context.Context, host *fleet.Host, abmTokenID uint) (*uint, error) {
	var abmDefaultTeamID *uint
	tok, err := t.ds.GetABMTokenByID(ctx, abmTokenID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting ABM token by id")
	}

	switch host.FleetPlatform() {
	case "darwin":
		abmDefaultTeamID = tok.MacOSDefaultTeamID
	case "ios":
		abmDefaultTeamID = tok.IOSDefaultTeamID
	case "ipados":
		abmDefaultTeamID = tok.IPadOSDefaultTeamID
	default:
		return nil, ctxerr.NewWithData(ctx, "attempting to get default ABM team for host with invalid platform", map[string]any{"host_platform": host.FleetPlatform(), "host_id": host.ID})
	}

	if abmDefaultTeamID == nil {
		// The default team is "No team", so we can return nil
		return nil, nil
	}

	exists, err := t.ds.TeamExists(ctx, *abmDefaultTeamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get default team for mdm devices")
	}

	if !exists {
		level.Debug(t.logger).Log(
			"msg",
			"unable to find default team assigned to abm token, mdm devices won't be assigned to a team",
			"team_id",
			abmDefaultTeamID,
		)
		return nil, nil
	}

	return abmDefaultTeamID, nil
}
