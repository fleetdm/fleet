package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) SetSetupExperienceSoftware(ctx context.Context, platform string, teamID uint, titleIDs []uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	macosHasManualAgentInstall := false
	var teamName string
	if teamID == 0 {
		teamName = ""
		ac, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting app config")
		}
		macosHasManualAgentInstall = ac.MDM.MacOSSetup.ManualAgentInstall.Value
	} else {
		team, err := svc.ds.TeamLite(ctx, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "load team")
		}
		teamName = team.Name
		macosHasManualAgentInstall = team.Config.MDM.MacOSSetup.ManualAgentInstall.Value
	}

	if macosHasManualAgentInstall && fleet.IsMacOSPlatform(platform) && len(titleIDs) != 0 {
		return fleet.NewUserMessageError(errors.New("Couldn’t add setup experience software. To add software, first disable macos_manual_agent_install."), http.StatusUnprocessableEntity)
	}

	if err := svc.ds.SetSetupExperienceSoftwareTitles(ctx, platform, teamID, titleIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "setting setup experience titles")
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityEditedSetupExperienceSoftware{
			Platform: platform,
			TeamID:   teamID,
			TeamName: teamName,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for set setup experience software")
	}

	return nil
}

func (svc *Service) ListSetupExperienceSoftware(ctx context.Context, platform string, teamID uint, opts fleet.ListOptions) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: &teamID,
	}, fleet.ActionRead); err != nil {
		return nil, 0, nil, err
	}

	titles, count, meta, err := svc.ds.ListSetupExperienceSoftwareTitles(ctx, platform, teamID, opts)
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

	if teamID == nil {
		ac, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting app config")
		}
		if ac.MDM.MacOSSetup.ManualAgentInstall.Value {
			return fleet.NewUserMessageError(errors.New("Couldn’t add setup experience script. To add script, first disable macos_manual_agent_install."), http.StatusUnprocessableEntity)
		}
	} else {
		team, err := svc.ds.TeamLite(ctx, *teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "load team")
		}
		if team.Config.MDM.MacOSSetup.ManualAgentInstall.Value {
			return fleet.NewUserMessageError(errors.New("Couldn’t add setup experience script. To add script, first disable macos_manual_agent_install."), http.StatusUnprocessableEntity)
		}
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
			err = fleet.NewInvalidArgumentError("team_id/fleet_id", "The fleet does not exist.").WithStatus(http.StatusNotFound)
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

func (svc *Service) SetupExperienceNextStep(ctx context.Context, host *fleet.Host) (bool, error) {
	// NOTE: currently, the Android platform does not go through the step-by-step setup experience flow as it
	// doesn't support any on-device UI (such as the screen showing setup progress) nor any
	// ordering of installs - all software to install is provided as part of the Android policy
	// when the host enrolls in Fleet.
	// See https://github.com/fleetdm/fleet/issues/33761#issuecomment-3548996114

	hostUUID, err := fleet.HostUUIDForSetupExperience(host)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "failed to get host's UUID for the setup experience")
	}
	statuses, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID, ptr.ValOrZero(host.TeamID))
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "retrieving setup experience status results for next step")
	}

	// Software (installers and VPP apps) are treated as a single group,
	// ordered alphabetically by display name (falling back to name). This
	// ordering is determined at enqueue time by enqueueSetupExperienceItems,
	// which inserts them with auto-incremented IDs in the correct order.
	// ListSetupExperienceResultsByHostUUID returns rows ordered by sesr.id.
	// Scripts always run after all software is done.
	var softwarePending, scriptsPending []*fleet.SetupExperienceStatusResult
	var softwareRunning, scriptsRunning int

	for _, status := range statuses {
		if err := status.IsValid(); err != nil {
			return false, ctxerr.Wrap(ctx, err, "invalid row")
		}

		switch {
		case status.IsForSoftware():
			switch status.Status {
			case fleet.SetupExperienceStatusPending:
				softwarePending = append(softwarePending, status)
			case fleet.SetupExperienceStatusRunning:
				softwareRunning++
			}
		case status.IsForScript():
			switch status.Status {
			case fleet.SetupExperienceStatusPending:
				scriptsPending = append(scriptsPending, status)
			case fleet.SetupExperienceStatusRunning:
				scriptsRunning++
			}
		}
	}

	switch {
	case len(softwarePending) > 0 && softwareRunning == 0:
		// Enqueue only the first pending software item (installer or VPP app).
		// On the next call, this item will be in "running" state and the next
		// pending item will be picked up. This ensures software is installed
		// one at a time in the alphabetical display-name order determined at
		// enqueue time (rows are ordered by sesr.id).
		sw := softwarePending[0]

		switch {
		case sw.SoftwareInstallerID != nil:
			installUUID, err := svc.ds.InsertSoftwareInstallRequest(ctx, host.ID, *sw.SoftwareInstallerID, fleet.HostSoftwareInstallOptions{
				SelfService:        false,
				ForSetupExperience: true,
			})
			if err != nil {
				return false, ctxerr.Wrap(ctx, err, "queueing setup experience install request")
			}
			sw.HostSoftwareInstallsExecutionID = &installUUID
			sw.Status = fleet.SetupExperienceStatusRunning
			if err := svc.ds.UpdateSetupExperienceStatusResult(ctx, sw); err != nil {
				return false, ctxerr.Wrap(ctx, err, "updating setup experience result with install uuid")
			}

		case sw.VPPAppTeamID != nil:
			vppAppID, err := sw.VPPAppID()
			if err != nil {
				return false, ctxerr.Wrap(ctx, err, "constructing vpp app details for installation")
			}

			if sw.SoftwareTitleID == nil {
				return false, ctxerr.Errorf(ctx, "setup experience software title id missing from vpp app install request: %d", sw.ID)
			}

			vppApp := &fleet.VPPApp{
				TitleID: *sw.SoftwareTitleID,
				VPPAppTeam: fleet.VPPAppTeam{
					VPPAppID: *vppAppID,
				},
			}

			cmdUUID, err := svc.installSoftwareFromVPP(ctx, host, vppApp, true, fleet.HostSoftwareInstallOptions{
				SelfService:        false,
				ForSetupExperience: true,
			})

			if err != nil {
				// if we get an error (e.g. no available licenses) while attempting to enqueue the
				// install, then we should immediately go to an error state so setup experience
				// isn't blocked.
				svc.logger.WarnContext(ctx, "got an error when attempting to enqueue VPP app install", "err", err, "adam_id", sw.VPPAppAdamID)
				sw.Status = fleet.SetupExperienceStatusFailure
				sw.Error = ptr.String(err.Error())
				// Persist the failure before cancelling other steps, so that
				// maybeCancelPendingSetupExperienceSteps can find the failed
				// item from its loaded statuses.
				if err := svc.ds.UpdateSetupExperienceStatusResult(ctx, sw); err != nil {
					return false, ctxerr.Wrap(ctx, err, "updating setup experience with vpp install failure")
				}
				failActivity := fleet.ActivityInstalledAppStoreApp{
					HostID:              host.ID,
					HostDisplayName:     host.DisplayName(),
					SoftwareTitle:       sw.Name,
					AppStoreID:          ptr.ValOrZero(sw.VPPAppAdamID),
					Status:              string(fleet.SoftwareInstallFailed),
					HostPlatform:        host.Platform,
					FromSetupExperience: true,
				}
				if actErr := svc.NewActivity(ctx, nil, failActivity); actErr != nil {
					svc.logger.WarnContext(ctx, "failed to create activity for VPP app install failure during setup experience", "err", actErr)
				}
				// At this point we need to check whether the "cancel if software install fails" setting is active,
				// in which case we'll cancel the remaining pending items.
				requireAllSoftware, err := svc.IsAllSetupExperienceSoftwareRequired(ctx, host)
				if err != nil {
					return false, ctxerr.Wrap(ctx, err, "checking if all software is required after vpp app install failure")
				}
				if requireAllSoftware {
					err := svc.MaybeCancelPendingSetupExperienceSteps(ctx, host)
					if err != nil {
						return false, ctxerr.Wrap(ctx, err, "cancelling remaining setup experience steps after vpp app install failure")
					}
				}
			} else {
				sw.NanoCommandUUID = &cmdUUID
				sw.Status = fleet.SetupExperienceStatusRunning
				if err := svc.ds.UpdateSetupExperienceStatusResult(ctx, sw); err != nil {
					return false, ctxerr.Wrap(ctx, err, "updating setup experience with vpp install command uuid")
				}
			}
		}
	case softwareRunning == 0 && len(scriptsPending) > 0:
		// enqueue scripts
		for _, script := range scriptsPending {
			if script.ScriptContentID == nil {
				return false, ctxerr.Errorf(ctx, "setup experience script missing content id: %d", *script.SetupExperienceScriptID)
			}
			req := &fleet.HostScriptRequestPayload{
				HostID:          host.ID,
				ScriptName:      script.Name,
				ScriptContentID: *script.ScriptContentID,
				// because the script execution request is associated with setup experience,
				// it will be enqueued with a higher priority and will run before other
				// items in the queue.
				SetupExperienceScriptID: script.SetupExperienceScriptID,
			}
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
	case softwareRunning == 0 && scriptsRunning == 0:
		// finished
		return true, nil
	}

	return false, nil
}
