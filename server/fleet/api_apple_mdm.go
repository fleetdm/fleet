package fleet

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
)

type GetMDMAppleCommandResultsRequest struct {
	CommandUUID string `query:"command_uuid,optional"`
}

type GetMDMAppleCommandResultsResponse struct {
	Results []*MDMCommandResult `json:"results,omitempty"`
	Err     error               `json:"error,omitempty"`
}

func (r GetMDMAppleCommandResultsResponse) Error() error { return r.Err }

type ListMDMAppleCommandsRequest struct {
	ListOptions ListOptions `url:"list_options"`
}

type ListMDMAppleCommandsResponse struct {
	Results []*MDMAppleCommand `json:"results"`
	Err     error              `json:"error,omitempty"`
}

func (r ListMDMAppleCommandsResponse) Error() error { return r.Err }

type NewMDMAppleConfigProfileRequest struct {
	TeamID  uint
	Profile *multipart.FileHeader
}

type NewMDMAppleConfigProfileResponse struct {
	ProfileID uint  `json:"profile_id"`
	Err       error `json:"error,omitempty"`
}

func (r NewMDMAppleConfigProfileResponse) Error() error { return r.Err }

type ListMDMAppleConfigProfilesRequest struct {
	TeamID uint `query:"team_id,optional" renameto:"fleet_id"`
}

type ListMDMAppleConfigProfilesResponse struct {
	ConfigProfiles []*MDMAppleConfigProfile `json:"profiles"`
	Err            error                    `json:"error,omitempty"`
}

func (r ListMDMAppleConfigProfilesResponse) Error() error { return r.Err }

type GetMDMAppleConfigProfileRequest struct {
	ProfileID uint `url:"profile_id"`
}

type GetMDMAppleConfigProfileResponse struct {
	Err error `json:"error,omitempty"`

	// file fields below are used in HijackRender for the response
	FileReader io.ReadCloser
	FileLength int64
	FileName   string
}

func (r GetMDMAppleConfigProfileResponse) Error() error { return r.Err }

func (r GetMDMAppleConfigProfileResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(r.FileLength, 10))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s.mobileconfig"`, r.FileName))

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	wl, err := io.Copy(w, r.FileReader)
	if err != nil {
		logging.WithExtras(ctx, "mobileconfig_copy_error", err, "bytes_copied", wl)
	}
	r.FileReader.Close()
}

type DeleteMDMAppleConfigProfileRequest struct {
	ProfileID uint `url:"profile_id"`
}

type DeleteMDMAppleConfigProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteMDMAppleConfigProfileResponse) Error() error { return r.Err }

type GetMDMAppleFileVaultSummaryRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetMDMAppleFileVaultSummaryResponse struct {
	*MDMAppleFileVaultSummary
	Err error `json:"error,omitempty"`
}

func (r GetMDMAppleFileVaultSummaryResponse) Error() error { return r.Err }

type GetMDMAppleProfilesSummaryRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetMDMAppleProfilesSummaryResponse struct {
	MDMProfilesSummary
	Err error `json:"error,omitempty"`
}

func (r GetMDMAppleProfilesSummaryResponse) Error() error { return r.Err }

type UploadAppleInstallerRequest struct {
	Installer *multipart.FileHeader
}

type UploadAppleInstallerResponse struct {
	ID  uint  `json:"installer_id"`
	Err error `json:"error,omitempty"`
}

func (r UploadAppleInstallerResponse) Error() error { return r.Err }

type GetAppleInstallerDetailsRequest struct {
	ID uint `url:"installer_id"`
}

type GetAppleInstallerDetailsResponse struct {
	Installer *MDMAppleInstaller
	Err       error `json:"error,omitempty"`
}

func (r GetAppleInstallerDetailsResponse) Error() error { return r.Err }

type DeleteAppleInstallerDetailsRequest struct {
	ID uint `url:"installer_id"`
}

type DeleteAppleInstallerDetailsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteAppleInstallerDetailsResponse) Error() error { return r.Err }

type ListMDMAppleDevicesRequest struct{}

type ListMDMAppleDevicesResponse struct {
	Devices []MDMAppleDevice `json:"devices"`
	Err     error            `json:"error,omitempty"`
}

func (r ListMDMAppleDevicesResponse) Error() error { return r.Err }

type NewMDMAppleDEPKeyPairResponse struct {
	PublicKey  []byte `json:"public_key,omitempty"`
	PrivateKey []byte `json:"private_key,omitempty"`
	Err        error  `json:"error,omitempty"`
}

func (r NewMDMAppleDEPKeyPairResponse) Error() error { return r.Err }

type EnqueueMDMAppleCommandRequest struct {
	Command   string   `json:"command"`
	DeviceIDs []string `json:"device_ids"`
}

type EnqueueMDMAppleCommandResponse struct {
	*CommandEnqueueResult
	Err error `json:"error,omitempty"`
}

func (r EnqueueMDMAppleCommandResponse) Error() error { return r.Err }

type MdmAppleEnrollRequest struct {
	// Token is expected to be a UUID string that identifies a template MDM Apple enrollment profile.
	Token string `query:"token"`
	// EnrollmentReference is expected to be a UUID string that identifies the MDM IdP account used
	// to authenticate the end user as part of the MDM IdP flow.
	EnrollmentReference string `query:"enrollment_reference,optional"`
	// DeviceInfo is expected to be a base64 encoded string extracted during MDM IdP enrollment from the
	// x-apple-aspen-deviceinfo header of the original configuration web view request and
	// persisted by the client in local storage for inclusion in a subsequent enrollment request as
	// part of the MDM IdP flow.
	// See https://developer.apple.com/documentation/devicemanagement/device_assignment/authenticating_through_web_views
	DeviceInfo string `query:"deviceinfo,optional"`
	// MachineInfo is the decoded deviceinfo URL query param for MDM IdP enrollments or the decoded
	// x-apple-aspen-deviceinfo header for non-IdP enrollments.
	MachineInfo *MDMAppleMachineInfo
}

type MdmAppleEnrollResponse struct {
	Err error `json:"error,omitempty"`

	// Profile field is used in HijackRender for the response.
	Profile []byte

	SoftwareUpdateRequired *MDMAppleSoftwareUpdateRequired
}

func (r MdmAppleEnrollResponse) Error() error { return r.Err }

func (r MdmAppleEnrollResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	if r.SoftwareUpdateRequired != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(r.SoftwareUpdateRequired); err != nil {
			logging.WithErr(ctx, err)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = fmt.Fprint(w, `{"error":"failed to encode software update required"}`)
		}
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.Profile)), 10))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "attachment;fleet-enrollment-profile.mobileconfig")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided.
	if n, err := w.Write(r.Profile); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

type MdmAppleAccountEnrollRequest struct {
	EnrollReference *string
	DeviceInfo      MDMAppleAccountDrivenUserEnrollDeviceInfo
}

type MdmAppleAccountEnrollAuthenticateResponse struct {
	Err       error `json:"error,omitempty"`
	MdmSSOUrl string
}

func (r MdmAppleAccountEnrollAuthenticateResponse) Error() error { return r.Err }

func (r MdmAppleAccountEnrollAuthenticateResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate",
		`Bearer method="apple-as-web" `+
			`url="`+r.MdmSSOUrl+`"`,
	)
	w.WriteHeader(http.StatusUnauthorized)
}

type MdmAppleGetInstallerRequest struct {
	Token string `query:"token"`
}

type MdmAppleGetInstallerResponse struct {
	Err error `json:"error,omitempty"`

	// Head is used by HijackRender for the response.
	Head bool
	// Name field is used in HijackRender for the response.
	Name string
	// Size field is used in HijackRender for the response.
	Size int64
	// Installer field is used in HijackRender for the response.
	Installer []byte
}

func (r MdmAppleGetInstallerResponse) Error() error { return r.Err }

func (r MdmAppleGetInstallerResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(r.Size, 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.Name))

	if r.Head {
		w.WriteHeader(http.StatusOK)
		return
	}

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.Installer); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

type MdmAppleHeadInstallerRequest struct {
	Token string `query:"token"`
}

type ListMDMAppleInstallersRequest struct{}

type ListMDMAppleInstallersResponse struct {
	Installers []MDMAppleInstaller `json:"installers"`
	Err        error               `json:"error,omitempty"`
}

func (r ListMDMAppleInstallersResponse) Error() error { return r.Err }

type DeviceLockRequest struct {
	HostID uint `url:"id"`
}

type DeviceLockResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeviceLockResponse) Error() error { return r.Err }

func (r DeviceLockResponse) Status() int { return http.StatusNoContent }

type DeviceWipeRequest struct {
	HostID uint `url:"id"`
}

type DeviceWipeResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeviceWipeResponse) Error() error { return r.Err }

func (r DeviceWipeResponse) Status() int { return http.StatusNoContent }

type GetHostProfilesRequest struct {
	ID uint `url:"id"`
}

type GetHostProfilesResponse struct {
	HostID   uint                     `json:"host_id"`
	Profiles []*MDMAppleConfigProfile `json:"profiles"`
	Err      error                    `json:"error,omitempty"`
}

func (r GetHostProfilesResponse) Error() error { return r.Err }

type BatchSetMDMAppleProfilesRequest struct {
	TeamID   *uint    `json:"-" query:"team_id,optional" renameto:"fleet_id"`
	TeamName *string  `json:"-" query:"team_name,optional" renameto:"fleet_name"`
	DryRun   bool     `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	Profiles [][]byte `json:"profiles"`
}

type BatchSetMDMAppleProfilesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r BatchSetMDMAppleProfilesResponse) Error() error { return r.Err }

func (r BatchSetMDMAppleProfilesResponse) Status() int { return http.StatusNoContent }

type PreassignMDMAppleProfileRequest struct {
	MDMApplePreassignProfilePayload
}

type PreassignMDMAppleProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r PreassignMDMAppleProfileResponse) Error() error { return r.Err }

func (r PreassignMDMAppleProfileResponse) Status() int { return http.StatusNoContent }

type MatchMDMApplePreassignmentRequest struct {
	ExternalHostIdentifier string `json:"external_host_identifier"`
}

type MatchMDMApplePreassignmentResponse struct {
	Err error `json:"error,omitempty"`
}

func (r MatchMDMApplePreassignmentResponse) Error() error { return r.Err }

func (r MatchMDMApplePreassignmentResponse) Status() int { return http.StatusNoContent }

type UpdateMDMAppleSettingsRequest struct {
	MDMAppleSettingsPayload
}

type UpdateMDMAppleSettingsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r UpdateMDMAppleSettingsResponse) Error() error { return r.Err }

func (r UpdateMDMAppleSettingsResponse) Status() int { return http.StatusNoContent }

type UploadBootstrapPackageRequest struct {
	Package *multipart.FileHeader
	DryRun  bool `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	TeamID  uint
}

type UploadBootstrapPackageResponse struct {
	Err error `json:"error,omitempty"`
}

func (r UploadBootstrapPackageResponse) Error() error { return r.Err }

type DownloadBootstrapPackageRequest struct {
	Token string `query:"token"`
}

type DownloadBootstrapPackageResponse struct {
	Err error `json:"error,omitempty"`

	// Pkg is used by HijackRender for the response.
	Pkg *MDMAppleBootstrapPackage
}

func (r DownloadBootstrapPackageResponse) Error() error { return r.Err }

func (r DownloadBootstrapPackageResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(len(r.Pkg.Bytes)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.Pkg.Name))

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.Pkg.Bytes); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

type BootstrapPackageMetadataRequest struct {
	TeamID uint `url:"fleet_id"`

	// ForUpdate is used to indicate that the authorization should be for a
	// "write" instead of a "read", this is needed specifically for the gitops
	// user which is a write-only user, but needs to call this endpoint to check
	// if it needs to upload the bootstrap package (if the hashes are different).
	//
	// NOTE: this parameter is going to be removed in a future version.
	// Prefer other ways to allow gitops read access.
	// For context, see: https://github.com/fleetdm/fleet/issues/15337#issuecomment-1932878997
	ForUpdate bool `query:"for_update,optional"`
}

type BootstrapPackageMetadataResponse struct {
	Err                       error `json:"error,omitempty"`
	*MDMAppleBootstrapPackage `json:",omitempty"`
}

func (r BootstrapPackageMetadataResponse) Error() error { return r.Err }

type DeleteBootstrapPackageRequest struct {
	TeamID uint `url:"fleet_id"`
	DryRun bool `query:"dry_run,optional"` // if true, apply validation but do not delete
}

type DeleteBootstrapPackageResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteBootstrapPackageResponse) Error() error { return r.Err }

type GetMDMAppleBootstrapPackageSummaryRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetMDMAppleBootstrapPackageSummaryResponse struct {
	MDMAppleBootstrapPackageSummary
	Err error `json:"error,omitempty"`
}

func (r GetMDMAppleBootstrapPackageSummaryResponse) Error() error { return r.Err }

type CreateMDMAppleSetupAssistantRequest struct {
	TeamID            *uint           `json:"team_id" renameto:"fleet_id"`
	Name              string          `json:"name"`
	EnrollmentProfile json.RawMessage `json:"enrollment_profile"`
}

type CreateMDMAppleSetupAssistantResponse struct {
	MDMAppleSetupAssistant
	Err error `json:"error,omitempty"`
}

func (r CreateMDMAppleSetupAssistantResponse) Error() error { return r.Err }

type GetMDMAppleSetupAssistantRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetMDMAppleSetupAssistantResponse struct {
	MDMAppleSetupAssistant
	Err error `json:"error,omitempty"`
}

func (r GetMDMAppleSetupAssistantResponse) Error() error { return r.Err }

type DeleteMDMAppleSetupAssistantRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type DeleteMDMAppleSetupAssistantResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteMDMAppleSetupAssistantResponse) Error() error { return r.Err }

func (r DeleteMDMAppleSetupAssistantResponse) Status() int { return http.StatusNoContent }

type UpdateMDMAppleSetupRequest struct {
	MDMAppleSetupPayload
}

type UpdateMDMAppleSetupResponse struct {
	Err error `json:"error,omitempty"`
}

func (r UpdateMDMAppleSetupResponse) Error() error { return r.Err }

func (r UpdateMDMAppleSetupResponse) Status() int { return http.StatusNoContent }

type InitiateMDMSSORequest struct {
	Initiator      string `json:"initiator,omitempty"`       // optional, passed by the UI during account-driven enrollment, or by Orbit for non-Apple IdP auth.
	UserIdentifier string `json:"user_identifier,omitempty"` // optional, passed by Apple for account-driven enrollment
	HostUUID       string `json:"host_uuid,omitempty"`       // optional, passed by Orbit for non-Apple IdP auth
}

type InitiateMDMSSOResponse struct {
	URL          string `json:"url,omitempty"`
	Err          error  `json:"error,omitempty"`
	SetCookiesFn func(context.Context, http.ResponseWriter) `json:"-"`
}

func (r InitiateMDMSSOResponse) Error() error { return r.Err }

func (r InitiateMDMSSOResponse) SetCookies(ctx context.Context, w http.ResponseWriter) {
	if r.SetCookiesFn != nil {
		r.SetCookiesFn(ctx, w)
	}
}

type CallbackMDMSSORequest struct {
	SessionID    string
	SAMLResponse []byte
}

type CallbackMDMSSOResponse struct {
	RedirectURL           string
	ByodEnrollCookieValue string
	SetCookiesFn          func(context.Context, http.ResponseWriter) `json:"-"`
}

func (r CallbackMDMSSOResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Location", r.RedirectURL)
	w.WriteHeader(http.StatusSeeOther)
}

func (r CallbackMDMSSOResponse) SetCookies(ctx context.Context, w http.ResponseWriter) {
	if r.SetCookiesFn != nil {
		r.SetCookiesFn(ctx, w)
	}
}

func (r CallbackMDMSSOResponse) Error() error { return nil }

type GetManualEnrollmentProfileRequest struct{}

type GetManualEnrollmentProfileResponse struct {
	// Profile field is used in HijackRender for the response.
	Profile []byte

	Err error `json:"error,omitempty"`
}

func (r GetManualEnrollmentProfileResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	// make the browser download the content to a file
	w.Header().Add("Content-Disposition", `attachment; filename="fleet-mdm-enrollment-profile.mobileconfig"`)
	// explicitly set the content length before the write, so the caller can
	// detect short writes (if it fails to send the full content properly)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.Profile)), 10))
	// this content type will make macos open the profile with the proper application
	w.Header().Set("Content-Type", "application/x-apple-aspen-config; charset=utf-8")
	// prevent detection of content, obey the provided content-type
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if n, err := w.Write(r.Profile); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func (r GetManualEnrollmentProfileResponse) Error() error { return r.Err }

type GenerateABMKeyPairResponse struct {
	PublicKey []byte `json:"public_key,omitempty"`
	Err       error  `json:"error,omitempty"`
}

func (r GenerateABMKeyPairResponse) Error() error { return r.Err }

type UploadABMTokenRequest struct {
	Token *multipart.FileHeader
}

type UploadABMTokenResponse struct {
	Token *ABMToken `json:"abm_token,omitempty"`
	Err   error     `json:"error,omitempty"`
}

func (r UploadABMTokenResponse) Error() error { return r.Err }

type DeleteABMTokenRequest struct {
	TokenID uint `url:"id"`
}

type DeleteABMTokenResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteABMTokenResponse) Error() error { return r.Err }

func (r DeleteABMTokenResponse) Status() int { return http.StatusNoContent }

type ListABMTokensResponse struct {
	Err    error       `json:"error,omitempty"`
	Tokens []*ABMToken `json:"abm_tokens"`
}

func (r ListABMTokensResponse) Error() error { return r.Err }

type CountABMTokensResponse struct {
	Err   error `json:"error,omitempty"`
	Count int   `json:"count"`
}

func (r CountABMTokensResponse) Error() error { return r.Err }

type UpdateABMTokenTeamsRequest struct {
	TokenID      uint  `url:"id"`
	MacOSTeamID  *uint `json:"macos_team_id" renameto:"macos_fleet_id"`
	IOSTeamID    *uint `json:"ios_team_id" renameto:"ios_fleet_id"`
	IPadOSTeamID *uint `json:"ipados_team_id" renameto:"ipados_fleet_id"`
}

type UpdateABMTokenTeamsResponse struct {
	ABMToken *ABMToken `json:"abm_token,omitempty"`
	Err      error     `json:"error,omitempty"`
}

func (r UpdateABMTokenTeamsResponse) Error() error { return r.Err }

type RenewABMTokenRequest struct {
	TokenID uint `url:"id"`
	Token   *multipart.FileHeader
}

type RenewABMTokenResponse struct {
	ABMToken *ABMToken `json:"abm_token,omitempty"`
	Err      error     `json:"error,omitempty"`
}

func (r RenewABMTokenResponse) Error() error { return r.Err }

type GetOTAProfileRequest struct {
	EnrollSecret string `query:"enroll_secret"`
	IdpUUID      string // The UUID of the mdm_idp_account that was used if any, can be empty, will be taken from cookies
}

type MdmAppleOTARequest struct {
	EnrollSecret string `query:"enroll_secret"`
	IdpUUID      string `query:"idp_uuid"`
	Certificates []*x509.Certificate
	RootSigner   *x509.Certificate
	DeviceInfo   MDMAppleMachineInfo
}

type MdmAppleOTAResponse struct {
	Err error `json:"error,omitempty"`
	XML []byte
}

func (r MdmAppleOTAResponse) Error() error { return r.Err }

func (r MdmAppleOTAResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(r.XML)))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := w.Write(r.XML); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
