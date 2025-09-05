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
	Name *string `json:"name" db:"name"`
	URL  *string `json:"url" db:"url"`

	// Digicert
	APIToken                      *string   `json:"api_token,omitempty" db:"-"`
	ProfileID                     *string   `json:"profile_id,omitempty" db:"profile_id"`
	CertificateCommonName         *string   `json:"certificate_common_name,omitempty" db:"certificate_common_name"`
	CertificateUserPrincipalNames *[]string `json:"certificate_user_principal_names,omitempty" db:"-"`
	CertificateSeatID             *string   `json:"certificate_seat_id,omitempty" db:"certificate_seat_id"`

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
	ID                            uint     `json:"-"`
	Name                          string   `json:"name"`
	URL                           string   `json:"url"`
	APIToken                      string   `json:"api_token"`
	ProfileID                     string   `json:"profile_id"`
	CertificateCommonName         string   `json:"certificate_common_name"`
	CertificateUserPrincipalNames []string `json:"certificate_user_principal_names"`
	CertificateSeatID             string   `json:"certificate_seat_id"`
}

// Equals checks if two DigiCertCA instances are equal, including sensitive fields (e.g., APIToken)
func (d *DigiCertCA) Equals(other *DigiCertCA) bool {
	return d.Name == other.Name &&
		d.URL == other.URL &&
		d.APIToken == other.APIToken &&
		d.ProfileID == other.ProfileID &&
		d.CertificateCommonName == other.CertificateCommonName &&
		slices.Equal(d.CertificateUserPrincipalNames, other.CertificateUserPrincipalNames) &&
		d.CertificateSeatID == other.CertificateSeatID
}

func (d *DigiCertCA) Preprocess() {
	d.Name = Preprocess(d.Name)
	d.URL = Preprocess(d.URL)
	d.ProfileID = Preprocess(d.ProfileID)
}

type HydrantCA struct {
	ID           uint   `json:"-"`
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
	ID       uint   `json:"-"`
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
	ID        uint   `json:"-"`
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

type CertificateAuthorityUpdatePayload struct {
	*DigiCertCAUpdatePayload        `json:"digicert,omitempty"`
	*NDESSCEPProxyCAUpdatePayload   `json:"ndes_scep_proxy,omitempty"`
	*CustomSCEPProxyCAUpdatePayload `json:"custom_scep_proxy,omitempty"`
	*HydrantCAUpdatePayload         `json:"hydrant,omitempty"`
}

// ValidatePayload checks that only one CA type is specified in the update payload and that the private key is provided.
func (p *CertificateAuthorityUpdatePayload) ValidatePayload(privateKey string, errPrefix string) error {
	caInPayload := 0
	if p.DigiCertCAUpdatePayload != nil {
		caInPayload++
	}
	if p.HydrantCAUpdatePayload != nil {
		caInPayload++
	}
	if p.NDESSCEPProxyCAUpdatePayload != nil {
		caInPayload++
	}
	if p.CustomSCEPProxyCAUpdatePayload != nil {
		caInPayload++
	}
	if caInPayload == 0 {
		return &BadRequestError{Message: fmt.Sprintf("%sA certificate authority must be specified", errPrefix)}
	}
	if caInPayload > 1 {
		return &BadRequestError{Message: fmt.Sprintf("%sOnly one certificate authority can be edited at a time", errPrefix)}
	}

	if len(privateKey) == 0 {
		return &BadRequestError{Message: fmt.Sprintf("%sPrivate key must be configured. Learn more: https://fleetdm.com/learn-more-about/fleet-server-private-key", errPrefix)}
	}
	return nil
}

type DigiCertCAUpdatePayload struct {
	Name                          *string   `json:"name"`
	URL                           *string   `json:"url"`
	APIToken                      *string   `json:"api_token"`
	ProfileID                     *string   `json:"profile_id"`
	CertificateCommonName         *string   `json:"certificate_common_name"`
	CertificateUserPrincipalNames *[]string `json:"certificate_user_principal_names"`
	CertificateSeatID             *string   `json:"certificate_seat_id"`
}

// IsEmpty checks if the struct only has all empty values
func (dcp DigiCertCAUpdatePayload) IsEmpty() bool {
	return dcp.Name == nil && dcp.URL == nil && dcp.APIToken == nil &&
		dcp.ProfileID == nil && dcp.CertificateCommonName == nil &&
		dcp.CertificateUserPrincipalNames == nil && dcp.CertificateSeatID == nil
}

// ValidateRelatedFields verifies that fields that are related to each other are set correctly.
// For example if the URL is provided then the API token must also be provided.
func (dcp *DigiCertCAUpdatePayload) ValidateRelatedFields(errPrefix string, certName string) error {
	// TODO: add cert name
	if (dcp.URL != nil || dcp.ProfileID != nil) && dcp.APIToken == nil {
		return &BadRequestError{Message: fmt.Sprintf(`%s"api_token" must be set when modifying "url", or "profile_id" of an existing certificate authority: %s`, errPrefix, certName)}
	}
	return nil
}

func (dcp *DigiCertCAUpdatePayload) Preprocess() {
	if dcp.Name != nil {
		*dcp.Name = Preprocess(*dcp.Name)
	}
	if dcp.URL != nil {
		*dcp.URL = Preprocess(*dcp.URL)
	}
	if dcp.ProfileID != nil {
		*dcp.ProfileID = Preprocess(*dcp.ProfileID)
	}
}

type NDESSCEPProxyCAUpdatePayload struct {
	URL      *string `json:"url"`
	AdminURL *string `json:"admin_url"`
	Username *string `json:"username"`
	Password *string `json:"password"`
}

// IsEmpty checks if the struct only has all empty values
func (ndesp *NDESSCEPProxyCAUpdatePayload) IsEmpty() bool {
	return ndesp.URL == nil && ndesp.AdminURL == nil && ndesp.Username == nil && ndesp.Password == nil
}

// ValidateRelatedFields verifies that fields that are related to each other are set correctly.
// For example if the Admin URL is provided then the Password must also be provided.
func (ndesp *NDESSCEPProxyCAUpdatePayload) ValidateRelatedFields(errPrefix string, certName string) error {
	// TODO: add cert name
	if (ndesp.URL != nil || ndesp.AdminURL != nil || ndesp.Username != nil) && ndesp.Password == nil {
		return &BadRequestError{Message: fmt.Sprintf(`%s"password" must be set when modifying an existing certificate authority: %s`, errPrefix, certName)}
	}
	return nil
}

func (ndesp *NDESSCEPProxyCAUpdatePayload) Preprocess() {
	if ndesp.URL != nil {
		*ndesp.URL = Preprocess(*ndesp.URL)
	}
	if ndesp.AdminURL != nil {
		*ndesp.AdminURL = Preprocess(*ndesp.AdminURL)
	}
	if ndesp.Username != nil {
		*ndesp.Username = Preprocess(*ndesp.Username)
	}
}

type CustomSCEPProxyCAUpdatePayload struct {
	Name      *string `json:"name"`
	URL       *string `json:"url"`
	Challenge *string `json:"challenge"`
}

// IsEmpty checks if the struct only has all empty values
func (cscepp CustomSCEPProxyCAUpdatePayload) IsEmpty() bool {
	return cscepp.Name == nil && cscepp.URL == nil && cscepp.Challenge == nil
}

// ValidateRelatedFields verifies that fields that are related to each other are set correctly.
// For example if the Name is provided then the Challenge must also be provided.
func (cscepp *CustomSCEPProxyCAUpdatePayload) ValidateRelatedFields(errPrefix string, certName string) error {
	// TODO: add cert name
	if cscepp.URL != nil && cscepp.Challenge == nil {
		return &BadRequestError{Message: fmt.Sprintf(`%s"challenge" must be set when modifying "url" of an existing certificate authority: %s`, errPrefix, certName)}
	}
	return nil
}

func (cscepp *CustomSCEPProxyCAUpdatePayload) Preprocess() {
	if cscepp.Name != nil {
		*cscepp.Name = Preprocess(*cscepp.Name)
	}
	if cscepp.URL != nil {
		*cscepp.URL = Preprocess(*cscepp.URL)
	}
}

type HydrantCAUpdatePayload struct {
	Name         *string `json:"name"`
	URL          *string `json:"url"`
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
}

// IsEmpty checks if the struct only has all empty values
func (hp HydrantCAUpdatePayload) IsEmpty() bool {
	return hp.Name == nil && hp.URL == nil && hp.ClientID == nil && hp.ClientSecret == nil
}

// ValidateRelatedFields verifies that fields that are related to each other are set correctly.
// For example if the Name is provided then the Client ID and Client Secret must also be provided
func (hp *HydrantCAUpdatePayload) ValidateRelatedFields(errPrefix string, certName string) error {
	if hp.URL != nil && (hp.ClientID == nil || hp.ClientSecret == nil) {
		return &BadRequestError{Message: fmt.Sprintf(`%s"client_secret" must be set when modifying "url" of an existing certificate authority: %s.`, errPrefix, certName)}
	}
	return nil
}

func (hp *HydrantCAUpdatePayload) Preprocess() {
	if hp.Name != nil {
		*hp.Name = Preprocess(*hp.Name)
	}
	if hp.URL != nil {
		*hp.URL = Preprocess(*hp.URL)
	}
}

type RequestCertificatePayload struct {
	ID          uint    `url:"id"`             // ID Of the CA the cert is to be requested from.
	CSR         string  `json:"csr"`           // PEM-encoded CSR
	IDPOauthURL *string `json:"idp_oauth_url"` // OAuth introspection URL for validating IDP Authentication
	IDPToken    *string `json:"idp_token"`     // Token for IDP Authentication
	IDPClientID *string `json:"idp_client_id"` // Client ID for IDP Authentication
}

func (c *RequestCertificatePayload) AuthzType() string {
	return "certificate_request"
}

type GroupedCertificateAuthorities struct {
	Hydrant         []HydrantCA         `json:"hydrant"`
	DigiCert        []DigiCertCA        `json:"digicert"`
	NDESSCEP        *NDESSCEPProxyCA    `json:"ndes_scep_proxy"`
	CustomScepProxy []CustomSCEPProxyCA `json:"custom_scep_proxy"`
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
				ID:                            ca.ID,
				Name:                          *ca.Name,
				CertificateCommonName:         *ca.CertificateCommonName,
				CertificateSeatID:             *ca.CertificateSeatID,
				CertificateUserPrincipalNames: *ca.CertificateUserPrincipalNames,
				APIToken:                      *ca.APIToken,
				URL:                           *ca.URL,
				ProfileID:                     *ca.ProfileID,
			})
		case string(CATypeNDESSCEPProxy):
			if grouped.NDESSCEP != nil {
				return nil, errors.New("multiple NDESSCEP proxy CAs found when grouping")
			}

			grouped.NDESSCEP = &NDESSCEPProxyCA{
				ID:       ca.ID,
				URL:      *ca.URL,
				AdminURL: *ca.AdminURL,
				Username: *ca.Username,
				Password: *ca.Password,
			}

		case string(CATypeHydrant):
			grouped.Hydrant = append(grouped.Hydrant, HydrantCA{
				ID:           ca.ID,
				Name:         *ca.Name,
				URL:          *ca.URL,
				ClientID:     *ca.ClientID,
				ClientSecret: *ca.ClientSecret,
			})
		case string(CATypeCustomSCEPProxy):
			grouped.CustomScepProxy = append(grouped.CustomScepProxy, CustomSCEPProxyCA{
				ID:        ca.ID,
				Name:      *ca.Name,
				URL:       *ca.URL,
				Challenge: *ca.Challenge,
			})
		}
	}

	return grouped, nil
}

// TODO: Do we need optjson here? We aren't supporting patch behavior for batch updates
type CertificateAuthoritiesSpec struct {
	DigiCert        optjson.Slice[DigiCertCA]        `json:"digicert"`
	NDESSCEPProxy   optjson.Any[NDESSCEPProxyCA]     `json:"ndes_scep_proxy"`
	CustomSCEPProxy optjson.Slice[CustomSCEPProxyCA] `json:"custom_scep_proxy"`
	Hydrant         optjson.Slice[HydrantCA]         `json:"hydrant"`
}

func ValidateCertificateAuthoritiesSpec(incoming interface{}) (*GroupedCertificateAuthorities, error) {
	var groupedCAs GroupedCertificateAuthorities

	// TODO(hca): Is all of this really necessary? It was pulled over from the old implementation,
	// but maybe it is overly complex for the current usage where we aren't supporting patch behavior.
	var spec interface{}
	if incoming == nil {
		spec = map[string]interface{}{}
	} else {
		spec = incoming
	}
	spec, ok := spec.(map[string]interface{})
	if !ok {
		return nil, errors.New("org_settings.certificate_authorities config is not a map")
	}

	if ndesSCEPProxy, ok := spec.(map[string]interface{})["ndes_scep_proxy"]; !ok || ndesSCEPProxy == nil {
		groupedCAs.NDESSCEP = nil
	} else {
		if _, ok = ndesSCEPProxy.(map[string]interface{}); !ok {
			return nil, errors.New("org_settings.certificate_authorities.ndes_scep_proxy config is not a map")
		}
		// We unmarshal NDES SCEP Proxy integration into its dedicated type for additional
		// validation.
		var ndesSCEPProxyData NDESSCEPProxyCA
		ndesSCEPProxyJSON, err := json.Marshal(ndesSCEPProxy)
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.ndes_scep_proxy cannot be marshalled into JSON: %w", err)
		}
		err = json.Unmarshal(ndesSCEPProxyJSON, &ndesSCEPProxyData)
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.ndes_scep_proxy cannot be parsed: %w", err)
		}
		groupedCAs.NDESSCEP = &ndesSCEPProxyData
	}

	if digicertIntegration, ok := spec.(map[string]interface{})["digicert"]; !ok || digicertIntegration == nil {
		groupedCAs.DigiCert = []DigiCertCA{}
	} else {
		// We unmarshal DigiCert integration into its dedicated type for additional validation.
		digicertJSON, err := json.Marshal(spec.(map[string]interface{})["digicert"])
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.digicert cannot be marshalled into JSON: %w", err)
		}
		var digicertData []DigiCertCA
		err = json.Unmarshal(digicertJSON, &digicertData)
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.digicert cannot be parsed: %w", err)
		}
		groupedCAs.DigiCert = digicertData
	}

	if customSCEPIntegration, ok := spec.(map[string]interface{})["custom_scep_proxy"]; !ok || customSCEPIntegration == nil {
		groupedCAs.CustomScepProxy = []CustomSCEPProxyCA{}
	} else {
		// We unmarshal Custom SCEP integration into its dedicated type for additional validation
		customSCEPJSON, err := json.Marshal(spec.(map[string]interface{})["custom_scep_proxy"])
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.custom_scep_proxy cannot be marshalled into JSON: %w", err)
		}
		var customSCEPData []CustomSCEPProxyCA
		err = json.Unmarshal(customSCEPJSON, &customSCEPData)
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.custom_scep_proxy cannot be parsed: %w", err)
		}
		groupedCAs.CustomScepProxy = customSCEPData
	}

	if hydrantCA, ok := spec.(map[string]interface{})["hydrant"]; !ok || hydrantCA == nil {
		groupedCAs.Hydrant = []HydrantCA{}
	} else {
		// We unmarshal Hydrant CA integration into its dedicated type for additional validation
		hydrantJSON, err := json.Marshal(spec.(map[string]interface{})["hydrant"])
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.hydrant cannot be marshalled into JSON: %w", err)
		}
		var hydrantData []HydrantCA
		err = json.Unmarshal(hydrantJSON, &hydrantData)
		if err != nil {
			return nil, fmt.Errorf("org_settings.certificate_authorities.hydrant cannot be parsed: %w", err)
		}
		groupedCAs.Hydrant = hydrantData
	}

	return &groupedCAs, nil
}

// CertificateAuthoritiesBatchOperations groups the operations for batch processing of certificate authorities.
type CertificateAuthoritiesBatchOperations struct {
	Delete []*CertificateAuthority
	Add    []*CertificateAuthority
	Update []*CertificateAuthority
}
