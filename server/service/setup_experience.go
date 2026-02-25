package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func putSetupExperienceSoftware(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.PutSetupExperienceSoftwareRequest)
	platform := transformPlatformForSetupExperience(req.Platform)
	err := svc.SetSetupExperienceSoftware(ctx, platform, req.TeamID, req.TitleIDs)
	if err != nil {
		return &fleet.PutSetupExperienceSoftwareResponse{Err: err}, nil
	}
	return &fleet.PutSetupExperienceSoftwareResponse{}, nil
}

func (svc *Service) SetSetupExperienceSoftware(ctx context.Context, platform string, teamID uint, titleIDs []uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

func getSetupExperienceSoftware(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetSetupExperienceSoftwareRequest)
	platform := transformPlatformListForSetupExperience(req.Platforms)
	titles, count, meta, err := svc.ListSetupExperienceSoftware(ctx, platform, req.TeamID, req.ListOptions)
	if err != nil {
		return &fleet.GetSetupExperienceSoftwareResponse{Err: err}, nil
	}
	return &fleet.GetSetupExperienceSoftwareResponse{SoftwareTitles: titles, Count: count, Meta: meta}, nil
}

func (svc *Service) ListSetupExperienceSoftware(ctx context.Context, platform string, teamID uint, opts fleet.ListOptions) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, 0, nil, fleet.ErrMissingLicense
}

func getSetupExperienceScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetSetupExperienceScriptRequest)
	downloadRequested := req.Alt == "media"
	// // TODO: do we want to allow end users to specify team_id=0? if so, we'll need convert it to nil here so that we can
	// // use it in the auth layer where team_id=0 is not allowed?
	script, content, err := svc.GetSetupExperienceScript(ctx, req.TeamID, downloadRequested)
	if err != nil {
		return fleet.GetSetupExperienceScriptResponse{Err: err}, nil
	}

	if downloadRequested {
		return fleet.DownloadFileResponse{
			Content:  content,
			Filename: fmt.Sprintf("%s %s", time.Now().Format(time.DateOnly), script.Name),
		}, nil
	}

	return fleet.GetSetupExperienceScriptResponse{Script: script}, nil
}

func (svc *Service) GetSetupExperienceScript(ctx context.Context, teamID *uint, withContent bool) (*fleet.Script, []byte, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, fleet.ErrMissingLicense
}

type decodeSetSetupExperienceScriptRequest struct{}

func (decodeSetSetupExperienceScriptRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	var decoded fleet.SetSetupExperienceScriptRequest

	err := parseMultipartForm(ctx, r, platform_http.MaxMultipartFormSize)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	val := r.MultipartForm.Value["fleet_id"]
	if len(val) > 0 {
		fleetID, err := strconv.ParseUint(val[0], 10, 64)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode fleet_id in multipart form: %s", err.Error())}
		}
		// // TODO: do we want to allow end users to specify team_id=0? if so, we'll need to convert it to nil here so that we can
		// // use it in the auth layer where team_id=0 is not allowed?
		decoded.TeamID = ptr.Uint(uint(fleetID)) // nolint:gosec // ignore G115
	}

	fhs, ok := r.MultipartForm.File["script"]
	if !ok || len(fhs) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for script"}
	}
	decoded.Script = fhs[0]

	return &decoded, nil
}

func setSetupExperienceScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.SetSetupExperienceScriptRequest)

	scriptFile, err := req.Script.Open()
	if err != nil {
		return fleet.SetSetupExperienceScriptResponse{Err: err}, nil
	}
	defer scriptFile.Close()

	if err := svc.SetSetupExperienceScript(ctx, req.TeamID, filepath.Base(req.Script.Filename), scriptFile); err != nil {
		return fleet.SetSetupExperienceScriptResponse{Err: err}, nil
	}

	return fleet.SetSetupExperienceScriptResponse{}, nil
}

func (svc *Service) SetSetupExperienceScript(ctx context.Context, teamID *uint, name string, r io.Reader) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

func deleteSetupExperienceScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteSetupExperienceScriptRequest)
	// // TODO: do we want to allow end users to specify team_id=0? if so, we'll need convert it to nil here so that we can
	// // use it in the auth layer where team_id=0 is not allowed?
	if err := svc.DeleteSetupExperienceScript(ctx, req.TeamID); err != nil {
		return fleet.DeleteSetupExperienceScriptResponse{Err: err}, nil
	}

	return fleet.DeleteSetupExperienceScriptResponse{}, nil
}

func (svc *Service) DeleteSetupExperienceScript(ctx context.Context, teamID *uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

func (svc *Service) SetupExperienceNextStep(ctx context.Context, host *fleet.Host) (bool, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return false, fleet.ErrMissingLicense
}

func (svc *Service) IsAllSetupExperienceSoftwareRequired(ctx context.Context, host *fleet.Host) (bool, error) {
	return isAllSetupExperienceSoftwareRequired(ctx, svc.ds, host)
}

func isAllSetupExperienceSoftwareRequired(ctx context.Context, ds fleet.Datastore, host *fleet.Host) (bool, error) {
	teamID := host.TeamID
	requireAllSoftware := false
	if teamID == nil || *teamID == 0 {
		ac, err := ds.AppConfig(ctx)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "getting app config")
		}
		requireAllSoftware = ac.MDM.MacOSSetup.RequireAllSoftware
	} else {
		team, err := ds.TeamLite(ctx, *teamID)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "load team")
		}
		requireAllSoftware = team.Config.MDM.MacOSSetup.RequireAllSoftware
	}
	return requireAllSoftware, nil
}

func (svc *Service) MaybeCancelPendingSetupExperienceSteps(ctx context.Context, host *fleet.Host) error {
	return maybeCancelPendingSetupExperienceSteps(ctx, svc.ds, host)
}

func maybeCancelPendingSetupExperienceSteps(ctx context.Context, ds fleet.Datastore, host *fleet.Host) error {
	// If the host is not MacOS, we do nothing.
	if host.Platform != "darwin" {
		return nil
	}

	requireAllSoftware, err := isAllSetupExperienceSoftwareRequired(ctx, ds, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if all software is required")
	}
	if !requireAllSoftware {
		return nil
	}
	hostUUID, err := fleet.HostUUIDForSetupExperience(host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get host's UUID for the setup experience")
	}
	statuses, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving setup experience status results for next step")
	}

	for _, status := range statuses {
		if err := status.IsValid(); err != nil {
			return ctxerr.Wrap(ctx, err, "invalid row")
		}
		if status.Status != fleet.SetupExperienceStatusPending && status.Status != fleet.SetupExperienceStatusRunning {
			continue
		}
		// Cancel any upcoming software installs, vpp installs or script runs.
		var executionID string
		switch {
		case status.HostSoftwareInstallsExecutionID != nil:
			executionID = *status.HostSoftwareInstallsExecutionID
		case status.NanoCommandUUID != nil:
			executionID = *status.NanoCommandUUID
		case status.ScriptExecutionID != nil:
			executionID = *status.ScriptExecutionID
		default:
			continue
		}
		if _, err := ds.CancelHostUpcomingActivity(ctx, host.ID, executionID); err != nil {
			return ctxerr.Wrap(ctx, err, "cancelling upcoming setup experience activity")
		}
	}
	// Cancel any pending setup experience steps for the host in the database.
	if err := ds.CancelPendingSetupExperienceSteps(ctx, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "cancelling pending setup experience steps")
	}
	return nil
}

// maybeUpdateSetupExperienceStatus attempts to update the status of a setup experience result in
// the database. If the given result is of a supported type (namely SetupExperienceScriptResult,
// SetupExperienceSoftwareInstallResult, and SetupExperienceVPPInstallResult), it returns a boolean
// indicating whether the datastore was updated and an error if one occurred. If the result is not of a
// supported type, it returns false and an error indicated that the type is not supported.
// If the skipPending parameter is true, the datastore will only be updated if the given result
// status is not pending.
func maybeUpdateSetupExperienceStatus(ctx context.Context, ds fleet.Datastore, result interface{}, requireTerminalStatus bool) (bool, error) {
	var updated bool
	var err error
	var status fleet.SetupExperienceStatusResultStatus
	var hostUUID string
	switch v := result.(type) {
	case fleet.SetupExperienceScriptResult:
		status = v.SetupExperienceStatus()
		if !status.IsValid() {
			return false, fmt.Errorf("invalid status: %s", status)
		} else if requireTerminalStatus && !status.IsTerminalStatus() {
			return false, nil
		}
		return ds.MaybeUpdateSetupExperienceScriptStatus(ctx, v.HostUUID, v.ExecutionID, status)

	case fleet.SetupExperienceSoftwareInstallResult:
		status = v.SetupExperienceStatus()
		hostUUID = v.HostUUID
		if !status.IsValid() {
			return false, fmt.Errorf("invalid status: %s", status)
		} else if requireTerminalStatus && !status.IsTerminalStatus() {
			return false, nil
		}
		updated, err = ds.MaybeUpdateSetupExperienceSoftwareInstallStatus(ctx, v.HostUUID, v.ExecutionID, status)

	case fleet.SetupExperienceVPPInstallResult:
		// NOTE: this case is also implemented in the CommandAndReportResults method of
		// MDMAppleCheckinAndCommandService
		status = v.SetupExperienceStatus()
		hostUUID = v.HostUUID
		if !status.IsValid() {
			return false, fmt.Errorf("invalid status: %s", status)
		} else if requireTerminalStatus && !status.IsTerminalStatus() {
			return false, nil
		}
		updated, err = ds.MaybeUpdateSetupExperienceVPPStatus(ctx, v.HostUUID, v.CommandUUID, status)

	default:
		return false, fmt.Errorf("unsupported result type: %T", result)
	}

	// For software / vpp installs, if we updated the status to failure and the host is macOS,
	// we may need to cancel the rest of the setup experience.
	if updated && err == nil && status == fleet.SetupExperienceStatusFailure {
		// Look up the host by UUID to get its platform and team.
		host, getHostUUIDErr := ds.HostByIdentifier(ctx, hostUUID)
		if getHostUUIDErr != nil {
			return updated, fmt.Errorf("getting host by UUID: %w", getHostUUIDErr)
		}
		cancelErr := maybeCancelPendingSetupExperienceSteps(ctx, ds, host)
		if cancelErr != nil {
			return updated, fmt.Errorf("cancel setup experience after macos software install failure: %w", cancelErr)
		}
	}
	return updated, err
}

func transformPlatformForSetupExperience(platform string) string {
	if platform == "" || platform == "macos" {
		return "darwin"
	}
	return platform
}

func transformPlatformListForSetupExperience(platforms string) string {
	if platforms == "" {
		return "darwin"
	}
	return strings.ReplaceAll(platforms, "macos", "darwin")
}
