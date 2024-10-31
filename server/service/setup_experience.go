package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

type putSetupExperienceSoftwareRequest struct {
	TeamID   uint   `json:"team_id"`
	TitleIDs []uint `json:"software_title_ids"`
}

type putSetupExperienceSoftwareResponse struct {
	Err error `json:"error,omitempty"`
}

func (r putSetupExperienceSoftwareResponse) error() error { return r.Err }

func putSetupExperienceSoftware(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*putSetupExperienceSoftwareRequest)

	err := svc.SetSetupExperienceSoftware(ctx, req.TeamID, req.TitleIDs)
	if err != nil {
		return &putSetupExperienceSoftwareResponse{Err: err}, nil
	}

	return &putSetupExperienceSoftwareResponse{}, nil
}

func (svc *Service) SetSetupExperienceSoftware(ctx context.Context, teamID uint, titleIDs []uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

type getSetupExperienceSoftwareRequest struct {
	fleet.ListOptions
	TeamID uint `query:"team_id"`
}

type getSetupExperienceSoftwareResponse struct {
	SoftwareTitles []fleet.SoftwareTitleListResult `json:"software_titles"`
	Count          int                             `json:"count"`
	Meta           *fleet.PaginationMetadata       `json:"meta"`
	Err            error                           `json:"error,omitempty"`
}

func (r getSetupExperienceSoftwareResponse) error() error { return r.Err }

func getSetupExperienceSoftware(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSetupExperienceSoftwareRequest)

	titles, count, meta, err := svc.ListSetupExperienceSoftware(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return &getSetupExperienceSoftwareResponse{Err: err}, nil
	}

	return &getSetupExperienceSoftwareResponse{SoftwareTitles: titles, Count: count, Meta: meta}, nil
}

func (svc *Service) ListSetupExperienceSoftware(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, 0, nil, fleet.ErrMissingLicense
}

type getSetupExperienceScriptRequest struct {
	TeamID *uint  `query:"team_id,optional"`
	Alt    string `query:"alt,optional"`
}

type getSetupExperienceScriptResponse struct {
	*fleet.Script
	Err error `json:"error,omitempty"`
}

func (r getSetupExperienceScriptResponse) error() error { return r.Err }

func getSetupExperienceScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSetupExperienceScriptRequest)
	downloadRequested := req.Alt == "media"
	// // TODO: do we want to allow end users to specify team_id=0? if so, we'll need convert it to nil here so that we can
	// // use it in the auth layer where team_id=0 is not allowed?
	script, content, err := svc.GetSetupExperienceScript(ctx, req.TeamID, downloadRequested)
	if err != nil {
		return getSetupExperienceScriptResponse{Err: err}, nil
	}

	if downloadRequested {
		return downloadFileResponse{
			content:  content,
			filename: fmt.Sprintf("%s %s", time.Now().Format(time.DateOnly), script.Name),
		}, nil
	}

	return getSetupExperienceScriptResponse{Script: script}, nil
}

func (svc *Service) GetSetupExperienceScript(ctx context.Context, teamID *uint, withContent bool) (*fleet.Script, []byte, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, fleet.ErrMissingLicense
}

type setSetupExperienceScriptRequest struct {
	TeamID *uint
	Script *multipart.FileHeader
}

func (setSetupExperienceScriptRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var decoded setSetupExperienceScriptRequest

	err := r.ParseMultipartForm(512 * units.MiB) // same in-memory size as for other multipart requests we have
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	val := r.MultipartForm.Value["team_id"]
	if len(val) > 0 {
		teamID, err := strconv.ParseUint(val[0], 10, 64)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode team_id in multipart form: %s", err.Error())}
		}
		// // TODO: do we want to allow end users to specify team_id=0? if so, we'll need to convert it to nil here so that we can
		// // use it in the auth layer where team_id=0 is not allowed?
		decoded.TeamID = ptr.Uint(uint(teamID))
	}

	fhs, ok := r.MultipartForm.File["script"]
	if !ok || len(fhs) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for script"}
	}
	decoded.Script = fhs[0]

	return &decoded, nil
}

type setSetupExperienceScriptResponse struct {
	Err error `json:"error,omitempty"`
}

func (r setSetupExperienceScriptResponse) error() error { return r.Err }

func setSetupExperienceScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*setSetupExperienceScriptRequest)

	scriptFile, err := req.Script.Open()
	if err != nil {
		return setSetupExperienceScriptResponse{Err: err}, nil
	}
	defer scriptFile.Close()

	if err := svc.SetSetupExperienceScript(ctx, req.TeamID, filepath.Base(req.Script.Filename), scriptFile); err != nil {
		return setSetupExperienceScriptResponse{Err: err}, nil
	}

	return setSetupExperienceScriptResponse{}, nil
}

func (svc *Service) SetSetupExperienceScript(ctx context.Context, teamID *uint, name string, r io.Reader) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

type deleteSetupExperienceScriptRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type deleteSetupExperienceScriptResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSetupExperienceScriptResponse) error() error { return r.Err }

// func (r deleteSetupExperienceScriptResponse) Status() int  { return http.StatusNoContent }

func deleteSetupExperienceScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteSetupExperienceScriptRequest)
	// // TODO: do we want to allow end users to specify team_id=0? if so, we'll need convert it to nil here so that we can
	// // use it in the auth layer where team_id=0 is not allowed?
	if err := svc.DeleteSetupExperienceScript(ctx, req.TeamID); err != nil {
		return deleteSetupExperienceScriptResponse{Err: err}, nil
	}

	return deleteSetupExperienceScriptResponse{}, nil
}

func (svc *Service) DeleteSetupExperienceScript(ctx context.Context, teamID *uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

func (svc *Service) SetupExperienceNextStep(ctx context.Context, hostUUID string) (bool, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return false, fleet.ErrMissingLicense
}

// maybeUpdateSetupExperienceStatus attempts to update the status of a setup experience result in
// the database. If the given result is of a supported type (namely SetupExperienceScriptResult,
// SetupExperienceSoftwareInstallResult, and SetupExperienceVPPInstallResult), it returns a boolean
// indicating whether the datastore was updated and an error if one occurred. If the result is not of a
// supported type, it returns false and an error indicated that the type is not supported.
// If the skipPending parameter is true, the datastore will only be updated if the given result
// status is not pending.
func maybeUpdateSetupExperienceStatus(ctx context.Context, ds fleet.Datastore, result interface{}, requireTerminalStatus bool) (bool, error) {
	switch v := result.(type) {
	case fleet.SetupExperienceScriptResult:
		status := v.SetupExperienceStatus()
		if !status.IsValid() {
			return false, fmt.Errorf("invalid status: %s", status)
		} else if requireTerminalStatus && !status.IsTerminalStatus() {
			return false, nil
		}
		return ds.MaybeUpdateSetupExperienceScriptStatus(ctx, v.HostUUID, v.ExecutionID, status)

	case fleet.SetupExperienceSoftwareInstallResult:
		status := v.SetupExperienceStatus()
		fmt.Println(status)
		if !status.IsValid() {
			return false, fmt.Errorf("invalid status: %s", status)
		} else if requireTerminalStatus && !status.IsTerminalStatus() {
			return false, nil
		}
		return ds.MaybeUpdateSetupExperienceSoftwareInstallStatus(ctx, v.HostUUID, v.ExecutionID, status)

	case fleet.SetupExperienceVPPInstallResult:
		// NOTE: this case is also implemented in the CommandAndReportResults method of
		// MDMAppleCheckinAndCommandService
		status := v.SetupExperienceStatus()
		if !status.IsValid() {
			return false, fmt.Errorf("invalid status: %s", status)
		} else if requireTerminalStatus && !status.IsTerminalStatus() {
			return false, nil
		}
		return ds.MaybeUpdateSetupExperienceVPPStatus(ctx, v.HostUUID, v.CommandUUID, status)

	default:
		return false, fmt.Errorf("unsupported result type: %T", result)
	}
}
