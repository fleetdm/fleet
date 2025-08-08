package fleet

import "slices"

type CertificateAuthoritySummary struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type CertificateAuthority struct {
	ID                           uint     `json:"id"`
	Name                         string   `json:"name"`
	Type                         string   `json:"type"`
	URL                          string   `json:"url"`
	APIToken                     string   `json:"api_token"`
	ProfileID                    uint     `json:"profile_id"`
	CertificateCommonName        string   `json:"certificate_common_name"`
	CertificateUserPrincipalName []string `json:"certificate_user_principal_name"`
	CertificateSeatIdentID       string   `json:"certificate_seat_id"`
}

type DigiCertCertAuthority struct {
	Name                          string   `json:"name"`
	URL                           string   `json:"url"`
	APIToken                      string   `json:"api_token"`
	ProfileID                     string   `json:"profile_id"`
	CertificateCommonName         string   `json:"certificate_common_name"`
	CertificateUserPrincipalNames []string `json:"certificate_user_principal_names"`
	CertificateSeatID             string   `json:"certificate_seat_id"`
}

func (d *DigiCertCertAuthority) Equals(other *DigiCertCertAuthority) bool {
	return d.Name == other.Name &&
		d.URL == other.URL &&
		(d.APIToken == "" || d.APIToken == MaskedPassword || d.APIToken == other.APIToken) &&
		d.ProfileID == other.ProfileID &&
		d.CertificateCommonName == other.CertificateCommonName &&
		slices.Equal(d.CertificateUserPrincipalNames, other.CertificateUserPrincipalNames) &&
		d.CertificateSeatID == other.CertificateSeatID
}

func (d *DigiCertCertAuthority) NeedToVerify(other *DigiCertCertAuthority) bool {
	return d.Name != other.Name ||
		d.URL != other.URL ||
		!(d.APIToken == "" || d.APIToken == MaskedPassword || d.APIToken == other.APIToken) ||
		d.ProfileID != other.ProfileID
}

// NDESSCEPProxyCertAuthority configures SCEP proxy for NDES SCEP server. Premium feature.
type NDESSCEPProxyCertAuthority struct {
	URL      string `json:"url"`
	AdminURL string `json:"admin_url"`
	Username string `json:"username"`
	Password string `json:"password"` // not stored here -- encrypted in DB
}

type CustomSCEPProxyCertAuthority struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Challenge string `json:"challenge"`
}

func (s *CustomSCEPProxyCertAuthority) Equals(other *CustomSCEPProxyCertAuthority) bool {
	return s.Name == other.Name &&
		s.URL == other.URL &&
		(s.Challenge == "" || s.Challenge == MaskedPassword || s.Challenge == other.Challenge)
}
