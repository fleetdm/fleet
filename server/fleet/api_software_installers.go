package fleet

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
)

type UploadSoftwareInstallerRequest struct {
	File              *multipart.FileHeader
	TeamID            *uint
	InstallScript     string
	PreInstallQuery   string
	PostInstallScript string
	SelfService       bool
	UninstallScript   string
	LabelsIncludeAny  []string
	LabelsExcludeAny  []string
	AutomaticInstall  bool
}

type UpdateSoftwareInstallerRequest struct {
	TitleID           uint `url:"id"`
	File              *multipart.FileHeader
	TeamID            *uint
	InstallScript     *string
	PreInstallQuery   *string
	PostInstallScript *string
	UninstallScript   *string
	SelfService       *bool
	LabelsIncludeAny  []string
	LabelsExcludeAny  []string
	Categories        []string
	DisplayName       *string
}

type UploadSoftwareInstallerResponse struct {
	SoftwarePackage *SoftwareInstaller `json:"software_package,omitempty"`
	Err             error              `json:"error,omitempty"`
}

func (r UploadSoftwareInstallerResponse) Error() error { return r.Err }

type DeleteSoftwareInstallerRequest struct {
	TeamID  *uint `query:"team_id" renameto:"fleet_id"`
	TitleID uint  `url:"title_id"`
}

type DeleteSoftwareInstallerResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteSoftwareInstallerResponse) Error() error { return r.Err }

func (r DeleteSoftwareInstallerResponse) Status() int { return http.StatusNoContent }

type GetSoftwareInstallerRequest struct {
	Alt     string `query:"alt,optional"`
	TeamID  *uint  `query:"team_id" renameto:"fleet_id"`
	TitleID uint   `url:"title_id"`
}

type DownloadSoftwareInstallerRequest struct {
	TitleID uint   `url:"title_id"`
	Token   string `url:"token"`
}

type GetSoftwareInstallerResponse struct {
	SoftwareInstaller *SoftwareInstaller `json:"software_installer,omitempty"`
	Err               error              `json:"error,omitempty"`
}

func (r GetSoftwareInstallerResponse) Error() error { return r.Err }

type GetSoftwareInstallerTokenResponse struct {
	Err   error  `json:"error,omitempty"`
	Token string `json:"token"`
}

func (r GetSoftwareInstallerTokenResponse) Error() error { return r.Err }

type OrbitDownloadSoftwareInstallerResponse struct {
	Err error `json:"error,omitempty"`
	// Payload is used by HijackRender for the response.
	Payload *DownloadSoftwareInstallerPayload
}

func (r OrbitDownloadSoftwareInstallerResponse) Error() error { return r.Err }

func (r OrbitDownloadSoftwareInstallerResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(int(r.Payload.Size)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.Payload.Filename))

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := io.Copy(w, r.Payload.Installer); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
	r.Payload.Installer.Close()
}

type InstallSoftwareRequest struct {
	HostID          uint `url:"host_id"`
	SoftwareTitleID uint `url:"software_title_id"`
}

type InstallSoftwareResponse struct {
	Err error `json:"error,omitempty"`
}

func (r InstallSoftwareResponse) Error() error { return r.Err }

func (r InstallSoftwareResponse) Status() int { return http.StatusAccepted }

type UninstallSoftwareRequest struct {
	HostID          uint `url:"host_id"`
	SoftwareTitleID uint `url:"software_title_id"`
}

type GetSoftwareInstallResultsRequest struct {
	InstallUUID string `url:"install_uuid"`
}

type GetDeviceSoftwareInstallResultsRequest struct {
	Token       string `url:"token"`
	InstallUUID string `url:"install_uuid"`
}

func (r *GetDeviceSoftwareInstallResultsRequest) DeviceAuthToken() string {
	return r.Token
}

type GetSoftwareInstallResultsResponse struct {
	Err     error                        `json:"error,omitempty"`
	Results *HostSoftwareInstallerResult `json:"results,omitempty"`
}

func (r GetSoftwareInstallResultsResponse) Error() error { return r.Err }

type GetDeviceSoftwareUninstallResultsRequest struct {
	Token       string `url:"token"`
	ExecutionID string `url:"execution_id"`
}

func (r *GetDeviceSoftwareUninstallResultsRequest) DeviceAuthToken() string {
	return r.Token
}

type BatchSetSoftwareInstallersRequest struct {
	TeamName string                      `json:"-" query:"team_name,optional" renameto:"fleet_name"`
	DryRun   bool                        `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	Software []*SoftwareInstallerPayload `json:"software"`
}

type BatchSetSoftwareInstallersResponse struct {
	RequestUUID string `json:"request_uuid"`
	Err         error  `json:"error,omitempty"`
}

func (r BatchSetSoftwareInstallersResponse) Error() error { return r.Err }

func (r BatchSetSoftwareInstallersResponse) Status() int { return http.StatusAccepted }

type BatchSetSoftwareInstallersResultRequest struct {
	RequestUUID string `url:"request_uuid"`
	TeamName    string `query:"team_name,optional" renameto:"fleet_name"`
	DryRun      bool   `query:"dry_run,optional"` // if true, apply validation but do not save changes
}

type BatchSetSoftwareInstallersResultResponse struct {
	Status   string                    `json:"status"`
	Message  string                    `json:"message"`
	Packages []SoftwarePackageResponse `json:"packages"`

	Err error `json:"error,omitempty"`
}

func (r BatchSetSoftwareInstallersResultResponse) Error() error { return r.Err }

type FleetSelfServiceSoftwareInstallRequest struct {
	Token           string `url:"token"`
	SoftwareTitleID uint   `url:"software_title_id"`
}

func (r *FleetSelfServiceSoftwareInstallRequest) DeviceAuthToken() string {
	return r.Token
}

type SubmitSelfServiceSoftwareInstallResponse struct {
	Err error `json:"error,omitempty"`
}

func (r SubmitSelfServiceSoftwareInstallResponse) Error() error { return r.Err }

func (r SubmitSelfServiceSoftwareInstallResponse) Status() int { return http.StatusAccepted }

type FleetDeviceSoftwareUninstallRequest struct {
	Token           string `url:"token"`
	SoftwareTitleID uint   `url:"software_title_id"`
}

func (r *FleetDeviceSoftwareUninstallRequest) DeviceAuthToken() string {
	return r.Token
}

type SubmitDeviceSoftwareUninstallResponse struct {
	Err error `json:"error,omitempty"`
}

func (r SubmitDeviceSoftwareUninstallResponse) Error() error { return r.Err }

func (r SubmitDeviceSoftwareUninstallResponse) Status() int { return http.StatusAccepted }

type BatchAssociateAppStoreAppsRequest struct {
	TeamName string            `json:"-" query:"team_name,optional" renameto:"fleet_name"`
	DryRun   bool              `json:"-" query:"dry_run,optional"`
	Apps     []VPPBatchPayload `json:"app_store_apps"`
}

func (b *BatchAssociateAppStoreAppsRequest) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	if err := json.NewDecoder(r).Decode(b); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return ctxerr.Wrap(ctx, NewUserMessageError(fmt.Errorf("Couldn't edit software. %q must be a %s, found %s", typeErr.Field, typeErr.Type.String(), typeErr.Value), http.StatusBadRequest))
		}
	}

	return nil
}

type BatchAssociateAppStoreAppsResponse struct {
	Apps []VPPAppResponse `json:"app_store_apps"`
	Err  error            `json:"error,omitempty"`
}

func (r BatchAssociateAppStoreAppsResponse) Error() error { return r.Err }

type GetInHouseAppManifestRequest struct {
	TitleID uint  `url:"title_id"`
	TeamID  *uint `query:"team_id" renameto:"fleet_id"`
}

type GetInHouseAppManifestResponse struct {
	// Manifest field is used in HijackRender for the response.
	Manifest []byte

	Err error `json:"error,omitempty"`
}

func (r GetInHouseAppManifestResponse) Error() error { return r.Err }

func (r GetInHouseAppManifestResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	// make the browser download the content to a file
	w.Header().Add("Content-Disposition", `attachment; filename="in-house-app-manifest.plist"`)
	// explicitly set the content length before the write, so the caller can
	// detect short writes (if it fails to send the full content properly)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.Manifest)), 10))
	// this content type will make macos open the profile with the proper application
	w.Header().Set("Content-Type", "application/x-apple-aspen-config; charset=utf-8")
	// prevent detection of content, obey the provided content-type
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if n, err := w.Write(r.Manifest); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

type GetInHouseAppPackageRequest struct {
	TitleID uint  `url:"title_id"`
	TeamID  *uint `query:"team_id" renameto:"fleet_id"`
}

type GetInHouseAppPackageResponse struct {
	Payload *DownloadSoftwareInstallerPayload

	Err error `json:"error,omitempty"`
}

func (r GetInHouseAppPackageResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(int(r.Payload.Size)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.Payload.Filename))

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := io.Copy(w, r.Payload.Installer); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
	r.Payload.Installer.Close()
}

func (r GetInHouseAppPackageResponse) Error() error { return r.Err }
