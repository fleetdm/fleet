package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) SetSetupExperienceSoftware(ctx context.Context, teamID uint, titleIDs []uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.SetSetupExperienceSoftwareTitles(ctx, teamID, titleIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "setting setup experience titles")
	}

	return nil
}

func (svc *Service) ListSetupExperienceSoftware(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: &teamID,
	}, fleet.ActionRead); err != nil {
		return nil, 0, nil, err
	}

	titles, count, meta, err := svc.ds.ListSetupExperienceSoftwareTitles(ctx, teamID, opts)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "retrieving list of software setup experience titles")
	}

	return titles, count, meta, nil
}

func (svc *Service) GetSetupExperienceScript(ctx context.Context, teamID *uint, withContent bool) (*fleet.Script, []byte, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	script, err := svc.ds.GetSetupExperienceScript(ctx, teamID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get setup experience script")
	}

	var content []byte
	if withContent {
		content, err = svc.ds.GetAnyScriptContents(ctx, script.ScriptContentID)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "get setup experience script contents")
		}
	}

	return script, content, nil
}

func (svc *Service) SetSetupExperienceScript(ctx context.Context, teamID *uint, name string, r io.Reader) error {
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read setup experience script contents")
	}

	script := &fleet.Script{
		TeamID:         teamID,
		Name:           name,
		ScriptContents: string(b),
	}

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, []string{script.ScriptContents}); err != nil {
		return fleet.NewInvalidArgumentError("script", err.Error())
	}

	// setup experience is only supported for macOS currently so we need to override the file
	// extension check in the general script validation
	if filepath.Ext(script.Name) != ".sh" {
		return fleet.NewInvalidArgumentError("script", "File type not supported. Only .sh file type is allowed.")
	}
	// now we can do our normal script validation
	if err := script.ValidateNewScript(); err != nil {
		return fleet.NewInvalidArgumentError("script", err.Error())
	}

	if err := svc.ds.SetSetupExperienceScript(ctx, script); err != nil {
		var (
			existsErr fleet.AlreadyExistsError
			fkErr     fleet.ForeignKeyError
		)
		if errors.As(err, &existsErr) {
			err = fleet.NewInvalidArgumentError("script", err.Error()).WithStatus(http.StatusConflict) // TODO: confirm error message with product/frontend
		} else if errors.As(err, &fkErr) {
			err = fleet.NewInvalidArgumentError("team_id", "The team does not exist.").WithStatus(http.StatusNotFound)
		}
		return ctxerr.Wrap(ctx, err, "create setup experience script")
	}

	// NOTE: there is no activity specified for set setup experience script

	return nil
}

func (svc *Service) DeleteSetupExperienceScript(ctx context.Context, teamID *uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteSetupExperienceScript(ctx, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete setup experience script")
	}

	// NOTE: there is no activity specified for delete setup experience script

	return nil
}

func (svc *Service) SetupExperienceNextStep(ctx context.Context, hostUUID string) (bool, error) {
	statuses, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "retrieving setup experience status results for next step")
	}

	var installersPending, appsPending, scriptsPending []*fleet.SetupExperienceStatusResult
	var installersRunning, appsRunning, scriptsRunning int

	for _, status := range statuses {
		if err := status.IsValid(); err != nil {
			return false, ctxerr.Wrap(ctx, err, "invalid row")
		}

		switch {
		case status.SoftwareInstallerID != nil:
			switch status.Status {
			case fleet.SetupExperienceStatusPending:
				installersPending = append(installersPending, status)
			case fleet.SetupExperienceStatusRunning:
				installersRunning++
			}
		case status.VPPAppTeamID != nil:
			switch status.Status {
			case fleet.SetupExperienceStatusPending:
				appsPending = append(appsPending, status)
			case fleet.SetupExperienceStatusRunning:
				appsRunning++
			}
		case status.SetupExperienceScriptID != nil:
			switch status.Status {
			case fleet.SetupExperienceStatusPending:
				scriptsPending = append(scriptsPending, status)
			case fleet.SetupExperienceStatusRunning:
				scriptsRunning++
			}
		}
	}

	// This step is called internally, not by a user
	filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
	hosts, err := svc.ds.ListHostsLiteByUUIDs(ctx, filter, []string{hostUUID})
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "fetching host details using UUID")
	}

	if len(hosts) == 0 {
		return false, ctxerr.Errorf(ctx, "could not find host id for host UUID %q", hostUUID)
	}

	host := hosts[0]

	switch {
	case len(installersPending) > 0:
		// enqueue installers
		for _, installer := range installersPending {
			installUUID, err := svc.ds.InsertSoftwareInstallRequest(ctx, host.ID, *installer.SoftwareInstallerID, false, nil)
			if err != nil {
				return false, ctxerr.Wrap(ctx, err, "queueing setup experience install request")
			}
			installer.HostSoftwareInstallsExecutionID = &installUUID
			installer.Status = fleet.SetupExperienceStatusRunning
			if err := svc.ds.UpdateSetupExperienceStatusResult(ctx, installer); err != nil {
				return false, ctxerr.Wrap(ctx, err, "updating setup experience result with install uuid")
			}
		}
	case installersRunning == 0 && len(appsPending) > 0:
		// enqueue vpp apps
		for _, app := range appsPending {
			vppAppID, err := app.VPPAppID()
			if err != nil {
				return false, ctxerr.Wrap(ctx, err, "constructing vpp app details for installation")
			}

			if app.SoftwareTitleID == nil {
				return false, ctxerr.Errorf(ctx, "setup experience software title id missing from vpp app install request: %d", app.ID)
			}

			vppApp := &fleet.VPPApp{
				TitleID: *app.SoftwareTitleID,
				VPPAppTeam: fleet.VPPAppTeam{
					VPPAppID: *vppAppID,
				},
			}

			cmdUUID, err := svc.installSoftwareFromVPP(ctx, host, vppApp, true, false)
			if err != nil {
				return false, ctxerr.Wrap(ctx, err, "queueing vpp app installation")
			}
			app.NanoCommandUUID = &cmdUUID
			app.Status = fleet.SetupExperienceStatusRunning
			if err := svc.ds.UpdateSetupExperienceStatusResult(ctx, app); err != nil {
				return false, ctxerr.Wrap(ctx, err, "updating setup experience with vpp install command uuid")
			}
		}
	case installersRunning == 0 && appsRunning == 0 && len(scriptsPending) > 0:
		// enqueue scripts
		for _, script := range scriptsPending {
			if script.ScriptContentID == nil {
				return false, ctxerr.Errorf(ctx, "setup experience script missing content id: %d", *script.SetupExperienceScriptID)
			}
			req := &fleet.HostScriptRequestPayload{
				HostID:                  host.ID,
				ScriptName:              script.Name,
				ScriptContentID:         *script.ScriptContentID,
				SetupExperienceScriptID: script.SetupExperienceScriptID,
			}
			// TODO(mna): setup experience scripts go to the unified queue, but must be higher priority.
			res, err := svc.ds.NewHostScriptExecutionRequest(ctx, req)
			if err != nil {
				return false, ctxerr.Wrap(ctx, err, "queueing setup experience script execution request")
			}
			script.ScriptExecutionID = &res.ExecutionID
			script.Status = fleet.SetupExperienceStatusRunning
			if err := svc.ds.UpdateSetupExperienceStatusResult(ctx, script); err != nil {
				return false, ctxerr.Wrap(ctx, err, "updating setup experience script execution id")
			}
		}
	case installersRunning == 0 && appsRunning == 0 && scriptsRunning == 0:
		// finished
		return true, nil
	}

	return false, nil
}
