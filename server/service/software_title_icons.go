package service

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/mux"
)

type getSoftwareTitleIconsRequest struct {
	TitleID uint  `url:"title_id"`
	TeamID  *uint `query:"team_id"`
}
type getSoftwareTitleIconsResponse struct {
	Err         error  `json:"error,omitempty"`
	ImageData   []byte `json:"-"`
	ContentType string `json:"-"`
	Filename    string `json:"-"`
	Size        int64  `json:"-"`
}

func (r getSoftwareTitleIconsResponse) Error() error { return r.Err }

type getSoftwareTitleIconsRedirectResponse struct {
	Err         error  `json:"error,omitempty"`
	RedirectURL string `json:"-"`
}

func (r getSoftwareTitleIconsRedirectResponse) Error() error { return r.Err }

func (r getSoftwareTitleIconsRedirectResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	if r.Err != nil {
		return
	}

	w.Header().Set("Location", r.RedirectURL)
	w.WriteHeader(http.StatusFound)
}

func (r getSoftwareTitleIconsResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	if r.Err != nil {
		return
	}

	w.Header().Set("Content-Type", r.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, r.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", r.Size))

	_, _ = w.Write(r.ImageData)
}

func getSoftwareTitleIconsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getSoftwareTitleIconsRequest)

	if req.TeamID == nil {
		return getSoftwareTitleIconsResponse{Err: &fleet.BadRequestError{Message: "team_id is required"}}, nil
	}
	if req.TitleID == 0 {
		return getSoftwareTitleIconsResponse{Err: &fleet.BadRequestError{Message: "invalid title_id"}}, nil
	}

	iconData, size, filename, err := svc.GetSoftwareTitleIcon(ctx, *req.TeamID, req.TitleID)
	if err != nil {
		var vppErr *fleet.VPPIconAvailable
		if errors.As(err, &vppErr) {
			// 302 redirect to vpp app IconURL
			return getSoftwareTitleIconsRedirectResponse{RedirectURL: vppErr.IconURL}, nil
		}
		return getSoftwareTitleIconsResponse{Err: err}, nil
	}

	return getSoftwareTitleIconsResponse{
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

type putSoftwareTitleIconRequest struct {
	TitleID    uint  `url:"title_id"`
	TeamID     *uint `query:"team_id"`
	File       *multipart.FileHeader
	HashSHA256 *string
	Filename   *string
}

type putSoftwareTitleIconResponse struct {
	Err     error  `json:"error,omitempty"`
	IconUrl string `json:"icon_url,omitempty"`
}

func (r putSoftwareTitleIconResponse) Error() error {
	return r.Err
}

func (putSoftwareTitleIconRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
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

	decoded := putSoftwareTitleIconRequest{
		TitleID: uint(titleIDUint64),
		TeamID:  &teamIDUint,
	}

	err = r.ParseMultipartForm(6 * units.MiB)
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

		if err := iconValidator(file); err != nil {
			return nil, err
		}
	}

	return &decoded, nil
}

func putSoftwareTitleIconEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*putSoftwareTitleIconRequest)

	payload := &fleet.UploadSoftwareTitleIconPayload{
		TitleID: req.TitleID,
		TeamID:  *req.TeamID,
	}

	if req.File != nil {
		file, err := req.File.Open()
		if err != nil {
			return putSoftwareTitleIconResponse{Err: err}, nil
		}
		defer file.Close()

		tfr, err := fleet.NewTempFileReader(file, nil)
		if err != nil {
			return putSoftwareTitleIconResponse{Err: err}, nil
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
		return putSoftwareTitleIconResponse{Err: err}, nil
	}

	return putSoftwareTitleIconResponse{
		IconUrl: softwareTitleIcon.IconUrl(),
	}, nil
}

func (svc *Service) UploadSoftwareTitleIcon(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (fleet.SoftwareTitleIcon, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.SoftwareTitleIcon{}, fleet.ErrMissingLicense
}

func iconValidator(file multipart.File) error {
	// Check file size first
	fileSize, err := file.Seek(0, 2) // Seek to end to get size
	if err != nil {
		return &fleet.BadRequestError{Message: "failed to read file size"}
	}
	if _, err := file.Seek(0, 0); err != nil { // Reset to beginning
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

type deleteSoftwareTitleIconRequest struct {
	TitleID uint  `url:"title_id"`
	TeamID  *uint `query:"team_id"`
}

type deleteSoftwareTitleIconResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSoftwareTitleIconResponse) Error() error {
	return r.Err
}

func deleteSoftwareTitleIconEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteSoftwareTitleIconRequest)

	if req.TeamID == nil {
		return getSoftwareTitleIconsResponse{Err: &fleet.BadRequestError{Message: "team_id is required"}}, nil
	}
	if req.TitleID == 0 {
		return getSoftwareTitleIconsResponse{Err: &fleet.BadRequestError{Message: "invalid title_id"}}, nil
	}

	err := svc.DeleteSoftwareTitleIcon(ctx, *req.TeamID, req.TitleID)
	if err != nil {
		return deleteSoftwareTitleIconResponse{Err: err}, nil
	}

	return deleteSoftwareTitleIconResponse{}, nil
}

func (svc *Service) DeleteSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
