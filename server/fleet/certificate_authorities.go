package fleet

import (
	"time"
)

type CAConfigAssetType string

const (
	CAConfigNDES            CAConfigAssetType = "ndes"
	CAConfigDigiCert        CAConfigAssetType = "digicert"
	CAConfigCustomSCEPProxy CAConfigAssetType = "custom_scep_proxy"
)

type CAConfigAsset struct {
	Name  string            `db:"name"`
	Value []byte            `db:"value"`
	Type  CAConfigAssetType `db:"type"`
}

type CAType string

const (
	CATypeNDESSCEPProxy   CAType = "ndes_scep_proxy"
	CATypeDigiCert        CAType = "digicert"
	CATypeCustomSCEPProxy CAType = "custom_scep_proxy"
	CATypeHydrant         CAType = "hydrant"
)

type CertificateAuthority struct {
	ID   uint   `json:"id" db:"id"`
	Type string `json:"type" db:"type"`

	// common
	Name string `json:"name" db:"name"`
	URL  string `json:"url" db:"url"`

	// Digicert
	APIToken                      *string  `json:"api_token,omitempty" db:"-"`
	ProfileID                     *string  `json:"profile_id,omitempty" db:"profile_id"`
	CertificateCommonName         *string  `json:"certificate_common_name,omitempty" db:"certificate_common_name"`
	CertificateUserPrincipalNames []string `json:"certificate_user_principal_names,omitempty" db:"-"`
	CertificateSeatID             *string  `json:"certificate_seat_id,omitempty" db:"certificate_seat_id"`

	// NDES SCEP Proxy
	AdminURL *string `json:"admin_url,omitempty" db:"admin_url"`
	Username *string `json:"username,omitempty" db:"username"`
	Password *string `json:"password,omitempty" db:"-"`

	// Custom SCEP Proxy
	Challenge *string `json:"challenge,omitempty" db:"-"`

	// Hydrant
	ClientID     *string `json:"client_id,omitempty" db:"client_id"`
	ClientSecret *string `json:"client_secret,omitempty" db:"-"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
