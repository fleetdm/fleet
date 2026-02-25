package fleet

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// writeCapabilitiesHeader writes the capabilities header to the response writer.
func writeCapabilitiesHeader(w http.ResponseWriter, capabilities CapabilityMap) {
	if len(capabilities) == 0 {
		return
	}
	w.Header().Set(CapabilitiesHeader, capabilities.String())
}

type DevicePingRequest struct{}

type DeviceAuthPingRequest struct {
	Token string `url:"token"`
}

func (r *DeviceAuthPingRequest) deviceAuthToken() string {
	return r.Token
}

type DevicePingResponse struct{}

func (r DevicePingResponse) Error() error { return nil }

func (r DevicePingResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	writeCapabilitiesHeader(w, GetServerDeviceCapabilities())
}

type FleetDesktopResponse struct {
	Err error `json:"error,omitempty"`
	DesktopSummary
}

func (r FleetDesktopResponse) Error() error { return r.Err }

type GetFleetDesktopRequest struct {
	Token string `url:"token"`
}

func (r *GetFleetDesktopRequest) deviceAuthToken() string {
	return r.Token
}

type GetDeviceHostRequest struct {
	Token           string `url:"token"`
	ExcludeSoftware bool   `query:"exclude_software,optional"`
}

func (r *GetDeviceHostRequest) deviceAuthToken() string {
	return r.Token
}

type GetDeviceHostResponse struct {
	Host                      *HostDetailResponse `json:"host"`
	SelfService               bool                `json:"self_service"`
	OrgLogoURL                string              `json:"org_logo_url"`
	OrgLogoURLLightBackground string              `json:"org_logo_url_light_background"`
	OrgContactURL             string              `json:"org_contact_url"`
	Err                       error               `json:"error,omitempty"`
	License                   LicenseInfo         `json:"license"`
	GlobalConfig              DeviceGlobalConfig  `json:"global_config"`
}

func (r GetDeviceHostResponse) Error() error { return r.Err }

type RefetchDeviceHostRequest struct {
	Token string `url:"token"`
}

func (r *RefetchDeviceHostRequest) deviceAuthToken() string {
	return r.Token
}

type ListDeviceHostDeviceMappingRequest struct {
	Token string `url:"token"`
}

func (r *ListDeviceHostDeviceMappingRequest) deviceAuthToken() string {
	return r.Token
}

type GetDeviceMacadminsDataRequest struct {
	Token string `url:"token"`
}

func (r *GetDeviceMacadminsDataRequest) deviceAuthToken() string {
	return r.Token
}

type ListDevicePoliciesRequest struct {
	Token string `url:"token"`
}

func (r *ListDevicePoliciesRequest) deviceAuthToken() string {
	return r.Token
}

type ListDevicePoliciesResponse struct {
	Err      error         `json:"error,omitempty"`
	Policies []*HostPolicy `json:"policies"`
}

func (r ListDevicePoliciesResponse) Error() error { return r.Err }

type BypassConditionalAccessRequest struct {
	Token string `url:"token"`
}

func (r *BypassConditionalAccessRequest) deviceAuthToken() string {
	return r.Token
}

type BypassConditionalAccessResponse struct {
	Err error `json:"error,omitempty"`
}

func (r BypassConditionalAccessResponse) Error() error { return r.Err }

type ResendDeviceConfigurationProfileRequest struct {
	Token       string `url:"token"`
	ProfileUUID string `url:"profile_uuid"`
}

func (r *ResendDeviceConfigurationProfileRequest) deviceAuthToken() string {
	return r.Token
}

type ResendDeviceConfigurationProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ResendDeviceConfigurationProfileResponse) Error() error { return r.Err }

func (r ResendDeviceConfigurationProfileResponse) Status() int { return http.StatusAccepted }

type GetDeviceMDMCommandResultsRequest struct {
	Token       string `url:"token"`
	CommandUUID string `url:"command_uuid"`
}

func (r *GetDeviceMDMCommandResultsRequest) deviceAuthToken() string {
	return r.Token
}

type TransparencyURLRequest struct {
	Token string `url:"token"`
}

func (r *TransparencyURLRequest) deviceAuthToken() string {
	return r.Token
}

type TransparencyURLResponse struct {
	RedirectURL string `json:"-"` // used to control the redirect, see HijackRender method
	Err         error  `json:"error,omitempty"`
}

func (r TransparencyURLResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Location", r.RedirectURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (r TransparencyURLResponse) Error() error { return r.Err }

type GetDeviceSoftwareIconRequest struct {
	Token           string `url:"token"`
	SoftwareTitleID uint   `url:"software_title_id"`
}

func (r *GetDeviceSoftwareIconRequest) deviceAuthToken() string {
	return r.Token
}

type GetDeviceSoftwareIconResponse struct {
	Err         error  `json:"error,omitempty"`
	ImageData   []byte `json:"-"`
	ContentType string `json:"-"`
	Filename    string `json:"-"`
	Size        int64  `json:"-"`
}

func (r GetDeviceSoftwareIconResponse) Error() error { return r.Err }

func (r GetDeviceSoftwareIconResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", r.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, r.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", r.Size))

	_, _ = w.Write(r.ImageData)
}

type GetDeviceSoftwareIconRedirectResponse struct {
	Err         error  `json:"error,omitempty"`
	RedirectURL string `json:"-"`
}

func (r GetDeviceSoftwareIconRedirectResponse) Error() error { return r.Err }

func (r GetDeviceSoftwareIconRedirectResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	if r.Err != nil {
		return
	}

	w.Header().Set("Location", r.RedirectURL)
	w.WriteHeader(http.StatusFound)
}

type FleetdErrorRequest struct {
	Token string `url:"token"`
	FleetdError
}

func (f *FleetdErrorRequest) deviceAuthToken() string {
	return f.Token
}

func (f *FleetdErrorRequest) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	decoder := json.NewDecoder(r)

	for {
		if err := decoder.Decode(&f.FleetdError); err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			return &BadRequestError{Message: "payload exceeds maximum accepted size"}
		} else if err != nil {
			return &BadRequestError{Message: "invalid payload"}
		}
	}

	return nil
}

type FleetdErrorResponse struct{}

func (r FleetdErrorResponse) Error() error { return nil }

type GetDeviceMDMManualEnrollProfileRequest struct {
	Token string `url:"token"`
}

func (r *GetDeviceMDMManualEnrollProfileRequest) deviceAuthToken() string {
	return r.Token
}

type GetDeviceMDMManualEnrollProfileResponse struct {
	// EnrollURL field is used in HijackRender for the response.
	EnrollURL string `json:"enroll_url,omitempty"`

	Err error `json:"error,omitempty"`
}

func (r GetDeviceMDMManualEnrollProfileResponse) Error() error { return r.Err }

type DeviceMigrateMDMRequest struct {
	Token string `url:"token"`
}

func (r *DeviceMigrateMDMRequest) deviceAuthToken() string {
	return r.Token
}

type DeviceMigrateMDMResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeviceMigrateMDMResponse) Error() error { return r.Err }

func (r DeviceMigrateMDMResponse) Status() int { return http.StatusNoContent }

type TriggerLinuxDiskEncryptionEscrowRequest struct {
	Token string `url:"token"`
}

func (r *TriggerLinuxDiskEncryptionEscrowRequest) deviceAuthToken() string {
	return r.Token
}

type TriggerLinuxDiskEncryptionEscrowResponse struct {
	Err error `json:"error,omitempty"`
}

func (r TriggerLinuxDiskEncryptionEscrowResponse) Error() error { return r.Err }

func (r TriggerLinuxDiskEncryptionEscrowResponse) Status() int { return http.StatusNoContent }

type GetDeviceSoftwareRequest struct {
	Token string `url:"token"`
	HostSoftwareTitleListOptions
}

func (r *GetDeviceSoftwareRequest) deviceAuthToken() string {
	return r.Token
}

type GetDeviceSoftwareResponse struct {
	Software []*HostSoftwareWithInstaller `json:"software"`
	Count    int                          `json:"count"`
	Meta     *PaginationMetadata          `json:"meta,omitempty"`
	Err      error                        `json:"error,omitempty"`
}

func (r GetDeviceSoftwareResponse) Error() error { return r.Err }

type ListDeviceCertificatesRequest struct {
	Token string `url:"token"`
	ListOptions
}

func (r *ListDeviceCertificatesRequest) ValidateRequest() error {
	if r.ListOptions.OrderKey != "" && !listHostCertificatesSortCols[r.ListOptions.OrderKey] {
		return &BadRequestError{Message: "invalid order key"}
	}
	return nil
}

func (r *ListDeviceCertificatesRequest) deviceAuthToken() string {
	return r.Token
}

type ListDeviceCertificatesResponse struct {
	Certificates []*HostCertificatePayload `json:"certificates"`
	Meta         *PaginationMetadata       `json:"meta,omitempty"`
	Count        uint                      `json:"count"`
	Err          error                     `json:"error,omitempty"`
}

func (r ListDeviceCertificatesResponse) Error() error { return r.Err }

type GetDeviceSetupExperienceStatusRequest struct {
	Token string `url:"token"`
}

func (r *GetDeviceSetupExperienceStatusRequest) deviceAuthToken() string {
	return r.Token
}

type GetDeviceSetupExperienceStatusResponse struct {
	Results *DeviceSetupExperienceStatusPayload `json:"setup_experience_results,omitempty"`
	Err     error                               `json:"error,omitempty"`
}

func (r GetDeviceSetupExperienceStatusResponse) Error() error { return r.Err }
