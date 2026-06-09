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

	// Re-check any policy-gated software item that is running but awaiting its policy result (it has a policy but no install
	// enqueued yet). This must run on every poll because, unlike an installing item, no install-completion callback will
	// re-trigger this flow for it. Such an item counts toward softwareRunning, so it also correctly blocks the next item from
	// starting until it resolves (skip or install).
	for _, status := range statuses {
		if status.IsForSoftware() && status.Status == fleet.SetupExperienceStatusRunning &&
			status.PolicyID != nil && status.HostSoftwareInstallsExecutionID == nil {
			if err := svc.advancePolicyGatedSetupExperienceItem(ctx, host, status); err != nil {
				return false, err
			}
			// One software item is in flight at a time; defer further progress to the next poll or install callback.
			return false, nil
		}
	}

	switch {
	case len(softwarePending) > 0 && softwareRunning == 0:
		// Enqueue only the first pending software item (installer or VPP app).
		// On the next call, this item will be in "running" state and the next
		// pending item will be picked up. This ensures software is installed
		// one at a time in the alphabetical display-name order determined at
		// enqueue time (rows are ordered by sesr.id).
		//
		// Un-gated items are started before policy-gated ones: a gated item may sit in "running" while it waits for the host's
		// labels and the gating policy result, and we don't want that wait to delay the un-gated installs (which need neither).
		// Alphabetical order is preserved within each group (softwarePending is already ordered by sesr.id).
		sw := softwarePending[0]
		for _, pending := range softwarePending {
			if pending.PolicyID == nil {
				sw = pending
				break
			}
		}

		switch {
		case sw.SoftwareInstallerID != nil:
			if sw.PolicyID != nil {
				// Policy-gated (Windows/Linux): run the associated policy as a gate. Flip to running (awaiting policy) and act on
				// a fresh result if one is already available; otherwise wait for the next poll. The install, when needed, is
				// performed through the normal setup-experience path below so it inherits the retry count and the
				// RequireAllSoftwareWindows handling.
				if err := svc.advancePolicyGatedSetupExperienceItem(ctx, host, sw); err != nil {
					return false, err
				}
				return false, nil
			}
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
		// finished: if any item was policy-gated, reset the host policy clock now (the host has left setup experience) so its full
		// policy set re-runs promptly instead of waiting up to a full policy update interval.
		if err := svc.resetPolicyClockAfterGatedSetup(ctx, host, statuses); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// advancePolicyGatedSetupExperienceItem drives a policy-gated Windows/Linux setup-experience software item. The associated policy
// is used purely as a gate: when it passes (app present and up-to-date) the install is skipped (terminal success); when it fails
// the item is installed through the normal setup-experience path so it inherits the setup-experience retry count and
// RequireAllSoftwareWindows handling. The item is held running until a fresh (this-enrollment) result arrives. If the gating
// policy's scope excludes the host (it will never run), the item falls back to installing rather than waiting forever.
//
// Order matters: scope is resolved before any result is trusted. A gating policy's scope (platform + include/exclude labels) is
// only knowable once the host has reported its labels for this enrollment, and a result reported before scope is known cannot be
// trusted (e.g. an exclude-label policy can run on the first read, before its exclusion is computed). So we (1) wait for labels,
// (2) check deliverability and install if the policy is out of scope, and only then (3) act on the policy result.
func (svc *Service) advancePolicyGatedSetupExperienceItem(ctx context.Context, host *fleet.Host, sw *fleet.SetupExperienceStatusResult) error {
	// Mark running (in-memory); the single persist below holds the softwareRunning==0 guard so no other item starts while we wait.
	alreadyRunning := sw.Status == fleet.SetupExperienceStatusRunning
	sw.Status = fleet.SetupExperienceStatusRunning

	// keepWaiting holds the item in "running" awaiting label/policy results. It persists the pending->running transition only once;
	// on later polls nothing has changed, so it skips the redundant UPDATE to avoid write amplification while waiting.
	keepWaiting := func() error {
		if alreadyRunning {
			return nil
		}
		return svc.ds.UpdateSetupExperienceStatusResult(ctx, sw)
	}

	// (1) The gate is only meaningful once the host has reported its labels for this enrollment: the policy's scope (platform plus
	// include/exclude labels) decides whether it applies, and a freshly enrolled host hasn't computed dynamic label membership yet.
	// Until then neither the scope check nor a result can be trusted -- an include-label policy would look out of scope, and an
	// exclude-label policy can report a result on the first read before its exclusion is known. Every host receives the platform=''
	// built-in dynamic labels (e.g. "All Linux"), so label_updated_at is guaranteed to advance on the first post-enroll checkin;
	// until then, keep waiting.
	if host.LabelUpdatedAt.Before(host.LastEnrolledAt) {
		return keepWaiting()
	}

	// (2) Labels are computed, so scope is now reliable. If the policy's scope excludes the host (platform, not in an include label,
	// or in an exclude label), it does not gate this host -> install via the normal setup-experience path. This must run before
	// consuming any result, so a result reported by a now-out-of-scope policy can't wrongly skip the install.
	deliverable, err := svc.ds.PolicyQueriesForHostFiltered(ctx, host, []uint{*sw.PolicyID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "check gating policy deliverability")
	}
	if len(deliverable) == 0 {
		svc.logger.InfoContext(ctx, "setup experience gating policy not applicable to host; installing item",
			"host_id", host.ID, "policy_id", *sw.PolicyID, "software_installer_id", *sw.SoftwareInstallerID)
		return svc.enqueueSetupExperienceSoftwareInstall(ctx, host, sw)
	}

	// (3) In scope. Act on a fresh (this-enrollment) result if one is available; otherwise keep waiting for it.
	passes, err := svc.ds.GetSetupExperiencePolicyResult(ctx, host.ID, *sw.PolicyID, host.LastEnrolledAt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get setup experience policy result")
	}
	switch {
	case passes == nil:
		return keepWaiting()
	case *passes:
		// App present and up-to-date: skip the install (terminal success). The host policy clock is reset once, when setup
		// experience finishes (see the "finished" branch of SetupExperienceNextStep), not here.
		sw.Status = fleet.SetupExperienceStatusSuccess
		svc.logger.DebugContext(ctx, "setup experience gating policy passed; skipping install",
			"host_id", host.ID, "policy_id", *sw.PolicyID, "software_installer_id", *sw.SoftwareInstallerID)
		return svc.ds.UpdateSetupExperienceStatusResult(ctx, sw)
	default:
		// Policy failed (app missing or outdated): install via the normal setup-experience path.
		return svc.enqueueSetupExperienceSoftwareInstall(ctx, host, sw)
	}
}

// resetPolicyClockAfterGatedSetup resets the host's "last reported policies" clock when setup experience finishes, if any of its
// items was policy-gated (Windows/Linux). During setup only the gated policy subset is distributed/reported, which advances the
// host-wide policy_updated_at and would otherwise delay the host's remaining policies by up to a full policy update interval once
// setup ends. This must run at completion, not per gated result: resetting mid-setup makes policies due again, so the host can
// re-report the gated subset and re-stamp the clock before setup ends. By the time setup is finished the host has left setup
// experience, so its next checkin carries the full policy set and the reset is not undone.
func (svc *Service) resetPolicyClockAfterGatedSetup(ctx context.Context, host *fleet.Host, statuses []*fleet.SetupExperienceStatusResult) error {
	gated := false
	for _, s := range statuses {
		if s.PolicyID != nil {
			gated = true
			break
		}
	}
	if !gated {
		return nil
	}
	if err := svc.ds.ClearHostPolicyUpdatedAt(ctx, host.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "reset host policy clock after setup experience")
	}
	return nil
}

// enqueueSetupExperienceSoftwareInstall enqueues a software installer item the same way an un-gated setup-experience item is
// enqueued (ForSetupExperience priority, the item owns its host_software_installs execution), and marks the item running.
func (svc *Service) enqueueSetupExperienceSoftwareInstall(ctx context.Context, host *fleet.Host, sw *fleet.SetupExperienceStatusResult) error {
	installUUID, err := svc.ds.InsertSoftwareInstallRequest(ctx, host.ID, *sw.SoftwareInstallerID, fleet.HostSoftwareInstallOptions{
		SelfService:        false,
		ForSetupExperience: true,
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing setup experience install request")
	}
	sw.HostSoftwareInstallsExecutionID = &installUUID
	sw.Status = fleet.SetupExperienceStatusRunning
	if err := svc.ds.UpdateSetupExperienceStatusResult(ctx, sw); err != nil {
		return ctxerr.Wrap(ctx, err, "updating setup experience result with install uuid")
	}
	return nil
}
