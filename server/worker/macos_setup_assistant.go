package worker

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Name of the macos setup assistant job as registered in the worker. Note that
// although it is a single job, it processes a number of different-but-related
// tasks, identified by the Task field in the job's payload.
const macosSetupAssistantJobName = "macos_setup_assistant" //nolint: gosec // somehow it detects this as credentials

type MacosSetupAssistantTask string

// List of supported tasks.
const (
	MacosSetupAssistantProfileChanged    MacosSetupAssistantTask = "profile_changed"
	MacosSetupAssistantProfileDeleted    MacosSetupAssistantTask = "profile_deleted"
	MacosSetupAssistantTeamDeleted       MacosSetupAssistantTask = "team_deleted"
	MacosSetupAssistantHostsTransferred  MacosSetupAssistantTask = "hosts_transferred"
	MacosSetupAssistantUpdateAllProfiles MacosSetupAssistantTask = "update_all_profiles"
	MacosSetupAssistantUpdateProfile     MacosSetupAssistantTask = "update_profile"
)

// MacosSetupAssistant is the job processor for the macos_setup_assistant job.
type MacosSetupAssistant struct {
	Datastore  fleet.Datastore
	Log        kitlog.Logger
	DEPService *apple_mdm.DEPService
	DEPClient  *godep.Client
}

// Name returns the name of the job.
func (m *MacosSetupAssistant) Name() string {
	return macosSetupAssistantJobName
}

// macosSetupAssistantArgs is the payload for the macos setup assistant job.
type macosSetupAssistantArgs struct {
	Task   MacosSetupAssistantTask `json:"task"`
	TeamID *uint                   `json:"team_id,omitempty"`
	// Note that only DEP-enrolled hosts in Fleet MDM should be provided.
	HostSerialNumbers []string `json:"host_serial_numbers,omitempty"`
}

// Run executes the macos_setup_assistant job.
func (m *MacosSetupAssistant) Run(ctx context.Context, argsJSON json.RawMessage) error {
	// if DEPService is nil, then mdm is not enabled, so just return without
	// error so we clean up any pending macos setup assistant jobs.
	if m.DEPService == nil {
		return nil
	}

	var args macosSetupAssistantArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case MacosSetupAssistantProfileChanged:
		return m.runProfileChanged(ctx, args)
	case MacosSetupAssistantProfileDeleted:
		return m.runProfileDeleted(ctx, args)
	case MacosSetupAssistantTeamDeleted:
		return m.runTeamDeleted(ctx, args)
	case MacosSetupAssistantHostsTransferred:
		return m.runHostsTransferred(ctx, args)
	case MacosSetupAssistantUpdateAllProfiles:
		return m.runUpdateAllProfiles(ctx, args)
	case MacosSetupAssistantUpdateProfile:
		return m.runUpdateProfile(ctx, args)
	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func (m *MacosSetupAssistant) runProfileChanged(ctx context.Context, args macosSetupAssistantArgs) error {
	team, err := m.getTeamNoTeam(ctx, args.TeamID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// team doesn't exist anymore, nothing to do (another job was enqueued to
			// take care of team deletion)
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get team")
	}

	// re-generate and register the profile with Apple. Since the profile has been
	// updated, then its profile UUID will have been cleared, so this single call
	// will do both tasks.
	profUUID, _, err := m.DEPService.EnsureCustomSetupAssistantIfExists(ctx, team)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "ensure custom setup assistant")
	}
	if profUUID == "" {
		// the custom setup assistant profile may have been deleted since the job
		// was enqueued, if so another job will take care of assigning the default
		// profile to the hosts, nothing to do.
		return nil
	}

	// get the team's mdm-enrolled hosts, assign the profile to all of that
	// team's hosts serials.
	serials, err := m.Datastore.ListMDMAppleDEPSerialsInTeam(ctx, args.TeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list mdm dep serials in team")
	}
	if len(serials) > 0 {
		if _, err := m.DEPClient.AssignProfile(ctx, apple_mdm.DEPName, profUUID, serials...); err != nil {
			return ctxerr.Wrap(ctx, err, "assign profile")
		}
	}
	return nil
}

func (m *MacosSetupAssistant) runProfileDeleted(ctx context.Context, args macosSetupAssistantArgs) error {
	team, err := m.getTeamNoTeam(ctx, args.TeamID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// team doesn't exist anymore, nothing to do (another job was enqueued to
			// take care of team deletion)
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get team")
	}

	// get the team's setup assistant, to make sure it is still absent. If it is
	// not, then it was re-created before this job ran, so nothing to do (another
	// job will take care of assigning the profile to the hosts).
	customProfUUID, _, err := m.DEPService.EnsureCustomSetupAssistantIfExists(ctx, team)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "ensure custom setup assistant")
	}
	if customProfUUID != "" {
		// a custom setup assistant was re-created, so nothing to do.
		return nil
	}

	// a custom setup assistant profile was deleted, so we get the profile uuid
	// of the default profile and assign it to all of the team's hosts. No need
	// to force a re-generate of the default profile, if it is already registered
	// with Apple this is fine and we use that profile uuid.
	profUUID, _, err := m.DEPService.EnsureDefaultSetupAssistant(ctx, team)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "ensure default setup assistant")
	}
	if profUUID == "" {
		// this should not happen, return an error
		return ctxerr.Errorf(ctx, "default setup assistant profile uuid is empty")
	}

	// get the team's mdm-enrolled hosts, assign the profile to all of that
	// team's hosts serials.
	serials, err := m.Datastore.ListMDMAppleDEPSerialsInTeam(ctx, args.TeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list mdm dep serials in team")
	}
	if len(serials) > 0 {
		if _, err := m.DEPClient.AssignProfile(ctx, apple_mdm.DEPName, profUUID, serials...); err != nil {
			return ctxerr.Wrap(ctx, err, "assign profile")
		}
	}
	return nil
}

func (m *MacosSetupAssistant) runTeamDeleted(ctx context.Context, args macosSetupAssistantArgs) error {
	// team deletion is semantically equivalent to moving hosts to "no team"
	args.TeamID = nil // should already be this way, but just to make sure
	return m.runHostsTransferred(ctx, args)
}

func (m *MacosSetupAssistant) runHostsTransferred(ctx context.Context, args macosSetupAssistantArgs) error {
	team, err := m.getTeamNoTeam(ctx, args.TeamID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// team doesn't exist anymore, nothing to do (another job was enqueued to
			// take care of team deletion)
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get team")
	}

	// get the new team's setup assistant if it exists.
	profUUID, _, err := m.DEPService.EnsureCustomSetupAssistantIfExists(ctx, team)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "ensure custom setup assistant")
	}
	if profUUID == "" {
		// get the default setup assistant.
		defProfUUID, _, err := m.DEPService.EnsureDefaultSetupAssistant(ctx, team)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "ensure default setup assistant")
		}
		profUUID = defProfUUID
		if profUUID == "" {
			// this should not happen, return an error
			return ctxerr.Errorf(ctx, "default setup assistant profile uuid is empty")
		}
	}

	_, err = m.DEPClient.AssignProfile(ctx, apple_mdm.DEPName, profUUID, args.HostSerialNumbers...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "assign profile")
	}
	return nil
}

func (m *MacosSetupAssistant) runUpdateAllProfiles(ctx context.Context, args macosSetupAssistantArgs) error {
	// for all teams and no-team, run the UpdateProfile task
	teams, err := m.Datastore.TeamsSummary(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get all teams")
	}

	processTeam := func(team *fleet.TeamSummary) error {
		var teamID *uint
		if team != nil {
			teamID = &team.ID
		}

		if err := QueueMacosSetupAssistantJob(ctx, m.Datastore, m.Log, MacosSetupAssistantUpdateProfile, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "queue macos setup assistant update profile job")
		}
		return nil
	}

	for _, tm := range teams {
		if err := processTeam(tm); err != nil {
			return err
		}
	}
	// and finally for no-team
	if err := processTeam(nil); err != nil {
		return err
	}
	return nil
}

func (m *MacosSetupAssistant) runUpdateProfile(ctx context.Context, args macosSetupAssistantArgs) error {
	// clear the profile uuid for the default setup assistant for that team/no-team
	if err := m.Datastore.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, args.TeamID, ""); err != nil {
		return ctxerr.Wrap(ctx, err, "clear default setup assistant profile uuid")
	}

	// clear the profile uuid for the custom setup assistant
	if err := m.Datastore.SetMDMAppleSetupAssistantProfileUUID(ctx, args.TeamID, ""); err != nil {
		if fleet.IsNotFound(err) {
			// no setup assistant for that team, enqueue a profile deleted task so
			// the default profile is assigned to the hosts.
			if err := QueueMacosSetupAssistantJob(ctx, m.Datastore, m.Log, MacosSetupAssistantProfileDeleted, args.TeamID); err != nil {
				return ctxerr.Wrap(ctx, err, "queue macos setup assistant profile deleted job")
			}
			return nil
		}
		return ctxerr.Wrap(ctx, err, "clear custom setup assistant profile uuid")
	}

	// no error means that the setup assistant existed for that team, enqueue a profile
	// changed task so the custom profile is assigned to the hosts.
	if err := QueueMacosSetupAssistantJob(ctx, m.Datastore, m.Log, MacosSetupAssistantProfileChanged, args.TeamID); err != nil {
		return ctxerr.Wrap(ctx, err, "queue macos setup assistant profile changed job")
	}
	return nil
}

func (m *MacosSetupAssistant) getTeamNoTeam(ctx context.Context, tmID *uint) (*fleet.Team, error) {
	var team *fleet.Team
	if tmID != nil {
		tm, err := m.Datastore.Team(ctx, *tmID)
		if err != nil {
			return nil, err
		}
		team = tm
	}
	return team, nil
}

// QueueMacosSetupAssistantJob queues a macos_setup_assistant job for one of
// the supported tasks, to be processed asynchronously via the worker.
func QueueMacosSetupAssistantJob(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	task MacosSetupAssistantTask,
	teamID *uint,
	serialNumbers ...string,
) error {
	attrs := []interface{}{
		"enabled", "true",
		macosSetupAssistantJobName, task,
		"hosts_count", len(serialNumbers),
	}
	if teamID != nil {
		attrs = append(attrs, "team_id", *teamID)
	}
	level.Info(logger).Log(attrs...)

	args := &macosSetupAssistantArgs{
		Task:              task,
		TeamID:            teamID,
		HostSerialNumbers: serialNumbers,
	}
	job, err := QueueJob(ctx, ds, macosSetupAssistantJobName, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
