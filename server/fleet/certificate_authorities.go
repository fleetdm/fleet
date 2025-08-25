package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
)

// TODO HCA these types can/should be removed once appconfig CA support is removed
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

type CertificateAuthoritySummary struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

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

type CertificateAuthorityPayload struct {
	DigiCert        *DigiCertCA        `json:"digicert,omitempty"`
	NDESSCEPProxy   *NDESSCEPProxyCA   `json:"ndes_scep_proxy,omitempty"`
	CustomSCEPProxy *CustomSCEPProxyCA `json:"custom_scep_proxy,omitempty"`
	Hydrant         *HydrantCA         `json:"hydrant,omitempty"`
}

// If you update this struct, make sure to adjust the Equals and NeedToVerify methods below
type DigiCertCA struct {
	Name                          string   `json:"name"`
	URL                           string   `json:"url"`
	APIToken                      string   `json:"api_token"`
	ProfileID                     string   `json:"profile_id"`
	CertificateCommonName         string   `json:"certificate_common_name"`
	CertificateUserPrincipalNames []string `json:"certificate_user_principal_names"`
	CertificateSeatID             string   `json:"certificate_seat_id"`
}

func (d *DigiCertCA) Equals(other *DigiCertCA) bool {
	return d.Name == other.Name &&
		d.URL == other.URL &&
		(d.APIToken == "" || d.APIToken == MaskedPassword || d.APIToken == other.APIToken) &&
		d.ProfileID == other.ProfileID &&
		d.CertificateCommonName == other.CertificateCommonName &&
		slices.Equal(d.CertificateUserPrincipalNames, other.CertificateUserPrincipalNames) &&
		d.CertificateSeatID == other.CertificateSeatID
}

func (d *DigiCertCA) StrictEquals(other *DigiCertCA) bool {
	return d.Name == other.Name &&
		d.URL == other.URL &&
		d.APIToken == other.APIToken &&
		d.ProfileID == other.ProfileID &&
		d.CertificateCommonName == other.CertificateCommonName &&
		slices.Equal(d.CertificateUserPrincipalNames, other.CertificateUserPrincipalNames) &&
		d.CertificateSeatID == other.CertificateSeatID
}

func (d *DigiCertCA) NeedToVerify(other *DigiCertCA) bool {
	return d.Name != other.Name ||
		d.URL != other.URL ||
		!(d.APIToken == "" || d.APIToken == MaskedPassword || d.APIToken == other.APIToken) ||
		d.ProfileID != other.ProfileID
}

type HydrantCA struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	ClientID     string `json:"client_id,omitempty" db:"client_id"`
	ClientSecret string `json:"client_secret,omitempty" db:"-"`
}

func (h *HydrantCA) Equals(other *HydrantCA) bool {
	return h.Name == other.Name &&
		h.URL == other.URL &&
		h.ClientID == other.ClientID &&
		(h.ClientSecret == "" || h.ClientSecret == MaskedPassword || h.ClientSecret == other.ClientSecret)
}

func (h *HydrantCA) NeedToVerify(other *HydrantCA) bool {
	return h.Name != other.Name ||
		h.URL != other.URL ||
		h.ClientID != other.ClientID ||
		!(h.ClientSecret == "" || h.ClientSecret == MaskedPassword || h.ClientSecret == other.ClientSecret)
}

// NDESSCEPProxyCA configures SCEP proxy for NDES SCEP server. Premium feature.
type NDESSCEPProxyCA struct {
	URL      string `json:"url"`
	AdminURL string `json:"admin_url"`
	Username string `json:"username"`
	Password string `json:"password"` // not stored here -- encrypted in DB
}

type SCEPConfigService interface {
	ValidateNDESSCEPAdminURL(ctx context.Context, proxy NDESSCEPProxyCA) error
	GetNDESSCEPChallenge(ctx context.Context, proxy NDESSCEPProxyCA) (string, error)
	ValidateSCEPURL(ctx context.Context, url string) error
}

type CustomSCEPProxyCA struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Challenge string `json:"challenge"`
}

func (s *CustomSCEPProxyCA) Equals(other *CustomSCEPProxyCA) bool {
	return s.Name == other.Name &&
		s.URL == other.URL &&
		(s.Challenge == "" || s.Challenge == MaskedPassword || s.Challenge == other.Challenge)
}

func (c *CertificateAuthority) AuthzType() string {
	return "certificate_authority"
}

type GroupedCertificateAuthorities struct {
	Hydrant         []HydrantCA
	DigiCert        []DigiCertCA
	NDESSCEP        *NDESSCEPProxyCA
	CustomScepProxy []CustomSCEPProxyCA
}

func GroupCertificateAuthoritiesByType(cas []*CertificateAuthority) (*GroupedCertificateAuthorities, error) {
	grouped := &GroupedCertificateAuthorities{
		DigiCert:        []DigiCertCA{},
		Hydrant:         []HydrantCA{},
		CustomScepProxy: []CustomSCEPProxyCA{},
		NDESSCEP:        nil,
	}

	for _, ca := range cas {
		switch ca.Type {
		case string(CATypeDigiCert):
			grouped.DigiCert = append(grouped.DigiCert, DigiCertCA{
				Name:                          ca.Name,
				CertificateCommonName:         *ca.CertificateCommonName,
				CertificateSeatID:             *ca.CertificateSeatID,
				CertificateUserPrincipalNames: ca.CertificateUserPrincipalNames,
				APIToken:                      *ca.APIToken,
				URL:                           ca.URL,
				ProfileID:                     *ca.ProfileID,
			})
		case string(CATypeNDESSCEPProxy):
			if grouped.NDESSCEP != nil {
				return nil, errors.New("multiple NDESSCEP proxy CAs found when grouping")
			}

			grouped.NDESSCEP = &NDESSCEPProxyCA{
				URL:      ca.URL,
				AdminURL: *ca.AdminURL,
				Username: *ca.Username,
				Password: *ca.Password,
			}

		case string(CATypeHydrant):
			grouped.Hydrant = append(grouped.Hydrant, HydrantCA{
				Name:         ca.Name,
				URL:          ca.URL,
				ClientID:     *ca.ClientID,
				ClientSecret: *ca.ClientSecret,
			})
		case string(CATypeCustomSCEPProxy):
			grouped.CustomScepProxy = append(grouped.CustomScepProxy, CustomSCEPProxyCA{
				Name:      ca.Name,
				URL:       ca.URL,
				Challenge: *ca.Challenge,
			})
		}
	}

	return grouped, nil
}

// TODO: confirm this type works; may need to use pointers; confirm expected "patch" behaviors in
// contxt of gitops
type CertificateAuthoritiesSpec struct {
	DigiCert optjson.Slice[DigiCertCA] `json:"digicert"`
	// NDESSCEPProxy settings. In JSON, not specifying this field means keep current setting, null means clear settings.
	NDESSCEPProxy   optjson.Any[NDESSCEPProxyCA]     `json:"ndes_scep_proxy"`
	CustomSCEPProxy optjson.Slice[CustomSCEPProxyCA] `json:"custom_scep_proxy"`
	Hydrant         optjson.Slice[HydrantCA]         `json:"hydrant"`
}

// TODO: handle optjson appropriately
func ValidateCertificateAuthoritiesSpec(incoming interface{}) (*CertificateAuthoritiesSpec, error) {
	var spec interface{}
	spec, ok := incoming.(map[string]interface{})
	if !ok || incoming == nil {
		spec = map[string]interface{}{}
		fmt.Println("org_settings.certificate_authorities config is not a map, using empty map")
		// group.AppConfig.(map[string]interface{})["integrations"] = integrations
	} else {
		spec = incoming.(map[string]interface{})
	}
	spec, ok = spec.(map[string]interface{})
	if !ok {
		return nil, errors.New("foobar: org_settings.certificate_authorities config is not a map")
	}

	if ndesSCEPProxy, ok := spec.(map[string]interface{})["ndes_scep_proxy"]; !ok || ndesSCEPProxy == nil {
		// Per backend patterns.md, best practice is to clear a JSON config field with `null`
		spec.(map[string]interface{})["ndes_scep_proxy"] = nil
	} else {
		if _, ok = ndesSCEPProxy.(map[string]interface{}); !ok {
			return nil, errors.New("org_settings.certificate_authorities.ndes_scep_proxy config is not a map")
		}
	}
	if digicertIntegration, ok := spec.(map[string]interface{})["digicert"]; !ok || digicertIntegration == nil {
		spec.(map[string]interface{})["digicert"] = nil
	} else {
		// We unmarshal DigiCert integration into its dedicated type for additional validation.
		digicertJSON, err := json.Marshal(spec.(map[string]interface{})["digicert"])
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.digicert cannot be marshalled into JSON: %w", err)
		}
		var digicertData optjson.Slice[DigiCertCA]
		err = json.Unmarshal(digicertJSON, &digicertData)
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.digicert cannot be parsed: %w", err)
		}
		spec.(map[string]interface{})["digicert"] = digicertData
	}
	if customSCEPIntegration, ok := spec.(map[string]interface{})["custom_scep_proxy"]; !ok || customSCEPIntegration == nil {
		spec.(map[string]interface{})["custom_scep_proxy"] = nil
	} else {
		// We unmarshal Custom SCEP integration into its dedicated type for additional validation
		custonSCEPJSON, err := json.Marshal(spec.(map[string]interface{})["custom_scep_proxy"])
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.custom_scep_proxy cannot be marshalled into JSON: %w", err)
		}
		var customSCEPData optjson.Slice[CustomSCEPProxyCA]
		err = json.Unmarshal(custonSCEPJSON, &customSCEPData)
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.custom_scep_proxy cannot be parsed: %w", err)
		}
		spec.(map[string]interface{})["custom_scep_proxy"] = customSCEPData
	}
	return nil, nil
}
