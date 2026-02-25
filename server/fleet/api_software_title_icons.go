package fleet

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
)

type GetSoftwareTitleIconsRequest struct {
	TitleID uint  `url:"title_id"`
	TeamID  *uint `query:"team_id" renameto:"fleet_id"`
}

type GetSoftwareTitleIconsResponse struct {
	Err         error  `json:"error,omitempty"`
	ImageData   []byte `json:"-"`
	ContentType string `json:"-"`
	Filename    string `json:"-"`
	Size        int64  `json:"-"`
}

func (r GetSoftwareTitleIconsResponse) Error() error { return r.Err }

func (r GetSoftwareTitleIconsResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	if r.Err != nil {
		return
	}

	w.Header().Set("Content-Type", r.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, r.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", r.Size))

	_, _ = w.Write(r.ImageData)
}

type GetSoftwareTitleIconsRedirectResponse struct {
	Err         error  `json:"error,omitempty"`
	RedirectURL string `json:"-"`
}

func (r GetSoftwareTitleIconsRedirectResponse) Error() error { return r.Err }

func (r GetSoftwareTitleIconsRedirectResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	if r.Err != nil {
		return
	}

	w.Header().Set("Location", r.RedirectURL)
	w.WriteHeader(http.StatusFound)
}

type PutSoftwareTitleIconRequest struct {
	TitleID    uint  `url:"title_id"`
	TeamID     *uint `query:"team_id" renameto:"fleet_id"`
	File       *multipart.FileHeader
	HashSHA256 *string
	Filename   *string
}

type PutSoftwareTitleIconResponse struct {
	Err     error  `json:"error,omitempty"`
	IconUrl string `json:"icon_url,omitempty"`
}

func (r PutSoftwareTitleIconResponse) Error() error {
	return r.Err
}

type DeleteSoftwareTitleIconRequest struct {
	TitleID uint  `url:"title_id"`
	TeamID  *uint `query:"team_id" renameto:"fleet_id"`
}

type DeleteSoftwareTitleIconResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteSoftwareTitleIconResponse) Error() error {
	return r.Err
}
