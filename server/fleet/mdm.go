package fleet

import "time"

type AppleMDM struct {
	CommonName   string    `json:"common_name"`
	SerialNumber string    `json:"serial_number"`
	Issuer       string    `json:"issuer"`
	RenewDate    time.Time `json:"renew_date"`
}

func (a AppleMDM) AuthzType() string {
	return "mdm_apple"
}

type AppleBM struct {
	AppleID      string    `json:"apple_id"`
	OrgName      string    `json:"org_name"`
	MDMServerURL string    `json:"mdm_server_url"`
	RenewDate    time.Time `json:"renew_date"`
	DefaultTeam  string    `json:"default_team"`
}

func (a AppleBM) AuthzType() string {
	return "mdm_apple"
}
