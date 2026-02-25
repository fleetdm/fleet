package fleet

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
)

type GetAppStoreAppsRequest struct {
	TeamID uint `query:"team_id" renameto:"fleet_id"`
}

type GetAppStoreAppsResponse struct {
	AppStoreApps []*VPPApp `json:"app_store_apps"`
	Err          error     `json:"error,omitempty"`
}

func (r GetAppStoreAppsResponse) Error() error { return r.Err }

type AddAppStoreAppRequest struct {
	TeamID           *uint                     `json:"team_id" renameto:"fleet_id"`
	AppStoreID       string                    `json:"app_store_id"`
	Platform         InstallableDevicePlatform `json:"platform"`
	SelfService      bool                      `json:"self_service"`
	AutomaticInstall bool                      `json:"automatic_install"`
	LabelsIncludeAny []string                  `json:"labels_include_any"`
	LabelsExcludeAny []string                  `json:"labels_exclude_any"`
	Categories       []string                  `json:"categories"`
	Configuration    json.RawMessage           `json:"configuration,omitempty"`
}

type AddAppStoreAppResponse struct {
	TitleID uint  `json:"software_title_id,omitempty"`
	Err     error `json:"error,omitempty"`
}

func (r AddAppStoreAppResponse) Error() error { return r.Err }

type UpdateAppStoreAppRequest struct {
	TitleID           uint            `url:"title_id"`
	TeamID            *uint           `json:"team_id" renameto:"fleet_id"`
	SelfService       *bool           `json:"self_service"`
	LabelsIncludeAny  []string        `json:"labels_include_any"`
	LabelsExcludeAny  []string        `json:"labels_exclude_any"`
	Categories        []string        `json:"categories"`
	Configuration     json.RawMessage `json:"configuration,omitempty"`
	DisplayName       *string         `json:"display_name"`
	AutoUpdateEnabled *bool           `json:"auto_update_enabled,omitempty"`
	// AutoUpdateStartTime is the beginning of the maintenance window for the software title.
	// This is only applicable when viewing a title in the context of a team.
	AutoUpdateStartTime *string `json:"auto_update_window_start,omitempty"`
	// AutoUpdateStartTime is the end of the maintenance window for the software title.
	// If the end time is less than the start time, the window wraps to the next day.
	// This is only applicable when viewing a title in the context of a team.
	AutoUpdateEndTime *string `json:"auto_update_window_end,omitempty"`
}

type UpdateAppStoreAppResponse struct {
	AppStoreApp *VPPAppStoreApp `json:"app_store_app,omitempty"`
	Err         error           `json:"error,omitempty"`
}

func (r UpdateAppStoreAppResponse) Error() error { return r.Err }

type UploadVPPTokenRequest struct {
	File *multipart.FileHeader
}

type UploadVPPTokenResponse struct {
	Err   error       `json:"error,omitempty"`
	Token *VPPTokenDB `json:"token,omitempty"`
}

func (r UploadVPPTokenResponse) Status() int { return http.StatusAccepted }

func (r UploadVPPTokenResponse) Error() error {
	return r.Err
}

type PatchVPPTokenRenewRequest struct {
	ID   uint `url:"id"`
	File *multipart.FileHeader
}

type PatchVPPTokenRenewResponse struct {
	Err   error       `json:"error,omitempty"`
	Token *VPPTokenDB `json:"token,omitempty"`
}

func (r PatchVPPTokenRenewResponse) Status() int { return http.StatusAccepted }

func (r PatchVPPTokenRenewResponse) Error() error {
	return r.Err
}

type PatchVPPTokensTeamsRequest struct {
	ID      uint   `url:"id"`
	TeamIDs []uint `json:"teams" renameto:"fleets"`
}

type PatchVPPTokensTeamsResponse struct {
	Token *VPPTokenDB `json:"token,omitempty"`
	Err   error       `json:"error,omitempty"`
}

func (r PatchVPPTokensTeamsResponse) Error() error { return r.Err }

type GetVPPTokensRequest struct{}

type GetVPPTokensResponse struct {
	Tokens []*VPPTokenDB `json:"vpp_tokens"`
	Err    error         `json:"error,omitempty"`
}

func (r GetVPPTokensResponse) Error() error { return r.Err }

type DeleteVPPTokenRequest struct {
	ID uint `url:"id"`
}

type DeleteVPPTokenResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteVPPTokenResponse) Error() error { return r.Err }

func (r DeleteVPPTokenResponse) Status() int { return http.StatusNoContent }
