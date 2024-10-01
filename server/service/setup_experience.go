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
)

type getSetupExperienceScriptRequest struct {
	TeamID uint   `query:"team_id"`
	Alt    string `query:"alt,optional"`
}

type getSetupExperienceScriptResponse struct {
	*fleet.Script
	Err error `json:"error,omitempty"`
}

func (r getSetupExperienceScriptResponse) error() error { return r.Err }

func getSetupExperienceScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(getSetupExperienceScriptRequest)
	downloadRequested := req.Alt == "media"
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

func (svc Service) GetSetupExperienceScript(ctx context.Context, teamID uint, withContent bool) (*fleet.Script, []byte, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, fleet.ErrMissingLicense
}

type setSetupExperienceScriptRequest struct {
	TeamID uint
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
		decoded.TeamID = uint(teamID)
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
	req := request.(setSetupExperienceScriptRequest)

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

func (svc Service) SetSetupExperienceScript(ctx context.Context, teamID uint, name string, r io.Reader) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

type deleteSetupExperienceScriptRequest struct {
	TeamID uint `query:"team_id"`
}

type deleteSetupExperienceScriptResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSetupExperienceScriptResponse) error() error { return r.Err }

// func (r deleteSetupExperienceScriptResponse) Status() int  { return http.StatusNoContent }

func deleteSetupExperienceScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(deleteSetupExperienceScriptRequest)
	if err := svc.DeleteSetupExperienceScript(ctx, req.TeamID); err != nil {
		return deleteSetupExperienceScriptResponse{Err: err}, nil
	}

	return deleteSetupExperienceScriptResponse{}, nil
}

func (svc Service) DeleteSetupExperienceScript(ctx context.Context, teamID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
