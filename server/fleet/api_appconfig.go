package fleet

import (
	"encoding/json"

	"github.com/fleetdm/fleet/v4/pkg/rawjson"
	"github.com/fleetdm/fleet/v4/server/version"
)

// AppConfigResponseFields are grouped separately to aid with JSON unmarshaling
type AppConfigResponseFields struct {
	UpdateInterval  *UpdateIntervalConfig  `json:"update_interval"`
	Vulnerabilities *VulnerabilitiesConfig `json:"vulnerabilities"`

	// License is loaded from the service
	License *LicenseInfo `json:"license,omitempty"`
	// Logging is loaded on the fly rather than from the database.
	Logging *Logging `json:"logging,omitempty"`
	// Email is returned when the email backend is something other than SMTP, for example SES
	Email *EmailConfig `json:"email,omitempty"`
	// SandboxEnabled is true if fleet serve was ran with server.sandbox_enabled=true
	SandboxEnabled bool         `json:"sandbox_enabled,omitempty"`
	Err            error        `json:"error,omitempty"`
	Partnerships   *Partnerships `json:"partnerships,omitempty"`
}

type AppConfigResponse struct {
	AppConfig
	AppConfigResponseFields
}

func (r *AppConfigResponse) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &r.AppConfig); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &r.AppConfigResponseFields); err != nil {
		return err
	}
	return nil
}

func (r AppConfigResponse) MarshalJSON() ([]byte, error) {
	// Marshal only the response fields
	responseData, err := json.Marshal(r.AppConfigResponseFields)
	if err != nil {
		return nil, err
	}

	// Marshal the base AppConfig
	appConfigData, err := json.Marshal(r.AppConfig)
	if err != nil {
		return nil, err
	}

	// we need to marshal and combine both groups separately because
	// AppConfig has a custom marshaler.
	return rawjson.CombineRoots(responseData, appConfigData)
}

func (r AppConfigResponse) Error() error { return r.Err }

type ModifyAppConfigRequest struct {
	Force     bool `json:"-" query:"force,optional"`     // if true, bypass strict incoming json validation
	DryRun    bool `json:"-" query:"dry_run,optional"`   // if true, apply validation but do not save changes
	Overwrite bool `json:"-" query:"overwrite,optional"` // if true, overwrite any existing settings with the incoming ones
	json.RawMessage
}

type ApplyEnrollSecretSpecRequest struct {
	Spec   *EnrollSecretSpec `json:"spec"`
	DryRun bool              `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
}

type ApplyEnrollSecretSpecResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ApplyEnrollSecretSpecResponse) Error() error { return r.Err }

type GetEnrollSecretSpecResponse struct {
	Spec *EnrollSecretSpec `json:"spec"`
	Err  error             `json:"error,omitempty"`
}

func (r GetEnrollSecretSpecResponse) Error() error { return r.Err }

type VersionResponse struct {
	*version.Info
	Err error `json:"error,omitempty"`
}

func (r VersionResponse) Error() error { return r.Err }

type GetCertificateResponse struct {
	CertificateChain []byte `json:"certificate_chain"`
	Err              error  `json:"error,omitempty"`
}

func (r GetCertificateResponse) Error() error { return r.Err }
