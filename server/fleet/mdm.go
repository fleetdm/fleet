package fleet

import (
	"context"
	"time"
)

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

type AppleCSR struct {
	// NOTE: []byte automatically JSON-encodes as a base64-encoded string
	APNsKey  []byte `json:"apns_key"`
	SCEPCert []byte `json:"scep_cert"`
	SCEPKey  []byte `json:"scep_key"`
}

func (a AppleCSR) AuthzType() string {
	return "mdm_apple"
}

// AppConfigUpdated is the minimal interface required to get and update the
// AppConfig, as required to handle the DEP API errors to flag that Apple's
// terms have changed and must be accepted. The Fleet Datastore satisfies
// this interface.
type AppConfigUpdater interface {
	AppConfig(ctx context.Context) (*AppConfig, error)
	SaveAppConfig(ctx context.Context, info *AppConfig) error
}
