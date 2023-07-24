package fleet

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

const (
	MDMPlatformApple     = "apple"
	MDMPlatformMicrosoft = "microsoft"
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

// MDMIdPAccount contains account information of a third-party IdP that can be
// later used for MDM operations like creating local accounts.
type MDMIdPAccount struct {
	UUID     string
	Username string
	Fullname string
}

type MDMAppleBootstrapPackage struct {
	Name      string    `json:"name"`
	TeamID    uint      `json:"team_id" db:"team_id"`
	Bytes     []byte    `json:"bytes,omitempty" db:"bytes"`
	Sha256    []byte    `json:"sha256" db:"sha256"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

func (bp MDMAppleBootstrapPackage) AuthzType() string {
	return "mdm_apple_bootstrap_package"
}

func (bp *MDMAppleBootstrapPackage) URL(host string) (string, error) {
	pkgURL, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	pkgURL.Path = "/api/latest/fleet/mdm/apple/bootstrap"
	pkgURL.RawQuery = fmt.Sprintf("token=%s", bp.Token)
	return pkgURL.String(), nil
}

// MDMAppleEULA represents an EULA (End User License Agreement) file.
type MDMAppleEULA struct {
	Name      string    `json:"name"`
	Bytes     []byte    `json:"bytes"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (e MDMAppleEULA) AuthzType() string {
	return "mdm_apple"
}
