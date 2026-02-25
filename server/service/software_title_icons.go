package service

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"

	"github.com/gorilla/mux"
)

func getSoftwareTitleIconsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetSoftwareTitleIconsRequest)

	if req.TeamID == nil {
		return fleet.GetSoftwareTitleIconsResponse{Err: &fleet.BadRequestError{Message: "team_id is required"}}, nil
	}
	if req.TitleID == 0 {
		return fleet.GetSoftwareTitleIconsResponse{Err: &fleet.BadRequestError{Message: "invalid title_id"}}, nil
	}

	iconData, size, filename, err := svc.GetSoftwareTitleIcon(ctx, *req.TeamID, req.TitleID)
	if err != nil {
		var vppErr *fleet.VPPIconAvailable
		if errors.As(err, &vppErr) {
			// 302 redirect to vpp app IconURL
			return fleet.GetSoftwareTitleIconsRedirectResponse{RedirectURL: vppErr.IconURL}, nil
		}
		return fleet.GetSoftwareTitleIconsResponse{Err: err}, nil
	}

	return fleet.GetSoftwareTitleIconsResponse{
		ImageData:   iconData,
		ContentType: "image/png", // only type of icon we currently allow
		Filename:    filename,
		Size:        size,
	}, nil
}

func (svc *Service) GetSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) ([]byte, int64, string, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, 0, "", fleet.ErrMissingLicense
}

type decodePutSoftwareTitleIconRequest struct{}

func (decodePutSoftwareTitleIconRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	urlVars := mux.Vars(r)
	titleID, ok := urlVars["title_id"]
	if !ok {
		return nil, &fleet.BadRequestError{Message: "title_id is required"}
	}
	titleIDUint64, err := strconv.ParseUint(titleID, 10, 64)
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "invalid title_id"}
	}
	if titleIDUint64 > math.MaxUint {
		return nil, &fleet.BadRequestError{Message: "title_id value too large"}
	}
	teamID := r.URL.Query().Get("team_id")
	if teamID == "" {
		return nil, &fleet.BadRequestError{Message: "team_id is required"}
	}
	teamIDUint64, err := strconv.ParseUint(teamID, 10, 64)
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "invalid team_id"}
	}
	if teamIDUint64 > math.MaxUint {
		return nil, &fleet.BadRequestError{Message: "team_id value too large"}
	}
	teamIDUint := uint(teamIDUint64)

	decoded := fleet.PutSoftwareTitleIconRequest{
		TitleID: uint(titleIDUint64),
		TeamID:  &teamIDUint,
	}

	err = r.ParseMultipartForm(platform_http.MaxMultipartFormSize)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	if values := r.MultipartForm.Value["hash_sha256"]; len(values) > 0 {
		decoded.HashSHA256 = &values[0]
	}
	if values := r.MultipartForm.Value["filename"]; len(values) > 0 {
		decoded.Filename = &values[0]
	}
	if len(r.MultipartForm.File["icon"]) > 0 {
		decoded.File = r.MultipartForm.File["icon"][0]
	}

	if decoded.File == nil && (decoded.HashSHA256 == nil || decoded.Filename == nil) {
		return nil, &fleet.BadRequestError{
			Message: "either icon multipart field or hashSHA256 and filename are required",
		}
	}
	if decoded.File != nil && (decoded.HashSHA256 != nil || decoded.Filename != nil) {
		return nil, &fleet.BadRequestError{
			Message: "cannot specify both icon file and hashSHA256/filename",
		}
	}

	// Validate the file if one is provided
	if decoded.File != nil {
		file, err := decoded.File.Open()
		if err != nil {
			return nil, &fleet.BadRequestError{Message: "failed to open file"}
		}
		defer file.Close()

		if err := ValidateIcon(file); err != nil {
			return nil, err
		}
	}

	return &decoded, nil
}

func putSoftwareTitleIconEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.PutSoftwareTitleIconRequest)

	payload := &fleet.UploadSoftwareTitleIconPayload{
		TitleID: req.TitleID,
		TeamID:  *req.TeamID,
	}

	if req.File != nil {
		file, err := req.File.Open()
		if err != nil {
			return fleet.PutSoftwareTitleIconResponse{Err: err}, nil
		}
		defer file.Close()

		tfr, err := fleet.NewTempFileReader(file, nil)
		if err != nil {
			return fleet.PutSoftwareTitleIconResponse{Err: err}, nil
		}
		defer tfr.Close()
		payload.IconFile = tfr
		payload.Filename = req.File.Filename
	}

	if req.HashSHA256 != nil && req.Filename != nil {
		payload.StorageID = *req.HashSHA256
		payload.Filename = *req.Filename
	}

	softwareTitleIcon, err := svc.UploadSoftwareTitleIcon(ctx, payload)
	if err != nil {
		return fleet.PutSoftwareTitleIconResponse{Err: err}, nil
	}

	return fleet.PutSoftwareTitleIconResponse{
		IconUrl: softwareTitleIcon.IconUrl(),
	}, nil
}

func (svc *Service) UploadSoftwareTitleIcon(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (fleet.SoftwareTitleIcon, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.SoftwareTitleIcon{}, fleet.ErrMissingLicense
}

func ValidateIcon(file io.ReadSeeker) error {
	// Check file size first
	fileSize, err := file.Seek(0, io.SeekEnd) // Seek to end to get size
	if err != nil {
		return &fleet.BadRequestError{Message: "failed to read file size"}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil { // Reset to beginning
		return &fleet.BadRequestError{Message: "failed to rewind file"}
	}

	maxSize := int64(100 * 1024) // 100KB
	if fileSize > maxSize {
		return &fleet.BadRequestError{Message: "icon must be less than 100KB"}
	}

	config, format, err := image.DecodeConfig(file)
	if err != nil || format != "png" {
		return &fleet.BadRequestError{Message: "icon must be a PNG image"}
	}

	maxWidth, maxHeight := 1024, 1024
	minWidth, minHeight := 120, 120

	if config.Width > maxWidth || config.Height > maxHeight {
		return &fleet.BadRequestError{Message: fmt.Sprintf("icon must be no larger than %dx%d pixels", maxWidth, maxHeight)}
	}
	if config.Width < minWidth || config.Height < minHeight {
		return &fleet.BadRequestError{Message: fmt.Sprintf("icon must be at least %dx%d pixels", minWidth, minHeight)}
	}
	if config.Width != config.Height {
		return &fleet.BadRequestError{Message: fmt.Sprintf("icon must be a square image (detected %dx%d pixels)", config.Width, config.Height)}
	}

	if _, err := file.Seek(0, 0); err != nil {
		return &fleet.BadRequestError{Message: "failed to rewind file"}
	}

	return nil
}

func deleteSoftwareTitleIconEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteSoftwareTitleIconRequest)

	if req.TeamID == nil {
		return fleet.GetSoftwareTitleIconsResponse{Err: &fleet.BadRequestError{Message: "team_id is required"}}, nil
	}
	if req.TitleID == 0 {
		return fleet.GetSoftwareTitleIconsResponse{Err: &fleet.BadRequestError{Message: "invalid title_id"}}, nil
	}

	err := svc.DeleteSoftwareTitleIcon(ctx, *req.TeamID, req.TitleID)
	if err != nil {
		return fleet.DeleteSoftwareTitleIconResponse{Err: err}, nil
	}

	return fleet.DeleteSoftwareTitleIconResponse{}, nil
}

func (svc *Service) DeleteSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
