package fleet

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
)

type OrbitGetConfigRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

func (r *OrbitGetConfigRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *OrbitGetConfigRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

func (r *OrbitGetConfigRequest) DecodeBody(_ context.Context, reader io.Reader, _ url.Values, _ []*x509.Certificate) error {
	if err := json.NewDecoder(reader).Decode(r); err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return &BadRequestError{
				Message:     "request body read timeout",
				InternalErr: err,
			}
		}
		return err
	}
	return nil
}

type OrbitGetConfigResponse struct {
	OrbitConfig
	Err error `json:"error,omitempty"`
}

func (r OrbitGetConfigResponse) Error() error { return r.Err }

type OrbitPingRequest struct{}

type OrbitPingResponse struct{}

func (r OrbitPingResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	writeCapabilitiesHeader(w, GetServerOrbitCapabilities())
}

func (r OrbitPingResponse) Error() error { return nil }

type SetOrUpdateDeviceTokenRequest struct {
	OrbitNodeKey    string `json:"orbit_node_key"`
	DeviceAuthToken string `json:"device_auth_token"`
}

func (r *SetOrUpdateDeviceTokenRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *SetOrUpdateDeviceTokenRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type SetOrUpdateDeviceTokenResponse struct {
	Err error `json:"error,omitempty"`
}

func (r SetOrUpdateDeviceTokenResponse) Error() error { return r.Err }

type OrbitGetScriptRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	ExecutionID  string `json:"execution_id"`
}

func (r *OrbitGetScriptRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *OrbitGetScriptRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type OrbitGetScriptResponse struct {
	Err error `json:"error,omitempty"`
	*HostScriptResult
}

func (r OrbitGetScriptResponse) Error() error { return r.Err }

type OrbitPostScriptResultRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	*HostScriptResultPayload
}

func (r *OrbitPostScriptResultRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *OrbitPostScriptResultRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type OrbitPostScriptResultResponse struct {
	Err error `json:"error,omitempty"`
}

func (r OrbitPostScriptResultResponse) Error() error { return r.Err }

type OrbitPutDeviceMappingRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	Email        string `json:"email"`
}

func (r *OrbitPutDeviceMappingRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *OrbitPutDeviceMappingRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type OrbitPutDeviceMappingResponse struct {
	Err error `json:"error,omitempty"`
}

func (r OrbitPutDeviceMappingResponse) Error() error { return r.Err }

type OrbitPostDiskEncryptionKeyRequest struct {
	OrbitNodeKey  string `json:"orbit_node_key"`
	EncryptionKey []byte `json:"encryption_key"`
	ClientError   string `json:"client_error"`
}

func (r *OrbitPostDiskEncryptionKeyRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *OrbitPostDiskEncryptionKeyRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type OrbitPostDiskEncryptionKeyResponse struct {
	Err error `json:"error,omitempty"`
}

func (r OrbitPostDiskEncryptionKeyResponse) Error() error { return r.Err }

func (r OrbitPostDiskEncryptionKeyResponse) Status() int { return http.StatusNoContent }

type OrbitPostLUKSRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	Passphrase   string `json:"passphrase"`
	Salt         string `json:"salt"`
	KeySlot      *uint  `json:"key_slot"`
	ClientError  string `json:"client_error"`
}

func (r *OrbitPostLUKSRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *OrbitPostLUKSRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type OrbitPostLUKSResponse struct {
	Err error `json:"error,omitempty"`
}

func (r OrbitPostLUKSResponse) Error() error { return r.Err }

func (r OrbitPostLUKSResponse) Status() int { return http.StatusNoContent }

type OrbitGetSoftwareInstallRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	OrbotNodeKey string `json:"orbot_node_key"` // legacy typo -- keep for backwards compatibility with orbit <= 1.38.0
	InstallUUID  string `json:"install_uuid"`
}

func (r *OrbitGetSoftwareInstallRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
	r.OrbotNodeKey = nodeKey // legacy typo -- keep for backwards compatability with fleet server < 4.63.0
}

func (r *OrbitGetSoftwareInstallRequest) OrbitHostNodeKey() string {
	if r.OrbitNodeKey != "" {
		return r.OrbitNodeKey
	}
	return r.OrbotNodeKey
}

type OrbitGetSoftwareInstallResponse struct {
	Err error `json:"error,omitempty"`
	*SoftwareInstallDetails
}

func (r OrbitGetSoftwareInstallResponse) Error() error { return r.Err }

type OrbitDownloadSoftwareInstallerRequest struct {
	Alt          string `query:"alt"`
	OrbitNodeKey string `json:"orbit_node_key"`
	InstallerID  uint   `json:"installer_id"`
}

func (r *OrbitDownloadSoftwareInstallerRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *OrbitDownloadSoftwareInstallerRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type OrbitPostSoftwareInstallResultRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	*HostSoftwareInstallResultPayload
}

func (r *OrbitPostSoftwareInstallResultRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *OrbitPostSoftwareInstallResultRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type OrbitPostSoftwareInstallResultResponse struct {
	Err error `json:"error,omitempty"`
}

func (r OrbitPostSoftwareInstallResultResponse) Error() error { return r.Err }

func (r OrbitPostSoftwareInstallResultResponse) Status() int { return http.StatusNoContent }

type GetOrbitSetupExperienceStatusRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	ForceRelease bool   `json:"force_release"`
	// Whether to re-enqueue canceled setup experience steps after a previous
	// software install failure on MacOS.
	ResetFailedSetupSteps bool `json:"reset_failed_setup_steps"`
}

func (r *GetOrbitSetupExperienceStatusRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *GetOrbitSetupExperienceStatusRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type GetOrbitSetupExperienceStatusResponse struct {
	Results *SetupExperienceStatusPayload `json:"setup_experience_results,omitempty"`
	Err     error                         `json:"error,omitempty"`
}

func (r GetOrbitSetupExperienceStatusResponse) Error() error { return r.Err }

type OrbitSetupExperienceInitRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

func (r *OrbitSetupExperienceInitRequest) SetOrbitNodeKey(nodeKey string) {
	r.OrbitNodeKey = nodeKey
}

func (r *OrbitSetupExperienceInitRequest) OrbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type OrbitSetupExperienceInitResponse struct {
	Result SetupExperienceInitResult `json:"result"`
	Err    error                     `json:"error,omitempty"`
}

func (r OrbitSetupExperienceInitResponse) Error() error {
	return r.Err
}
