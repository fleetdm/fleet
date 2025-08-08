package fleet

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
