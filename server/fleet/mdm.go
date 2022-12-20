package fleet

import (
	"context"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
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

// AppConfigUpdated is the minimal interface required to get and update the
// AppConfig, as required to handle the DEP API errors to flag that Apple's
// terms have changed and must be accepted. The Fleet Datastore satisfies
// this interface.
type AppConfigUpdater interface {
	AppConfig(ctx context.Context) (*AppConfig, error)
	SaveAppConfig(ctx context.Context, info *AppConfig) error
}

type termsChangedDoer struct {
	doer    nanodep_client.Doer
	updater AppConfigUpdater
}

func (d termsChangedDoer) Do(req *http.Request) (*http.Response, error) {
	// make the actual DEP request
	res, err := d.doer.Do(req)

	// if the request failed due to terms not signed, or if it succeeded,
	// update the app config flag accordingly. If it failed for any other
	// reason, do not update the flag.
	termsExpired := err != nil && godep.IsTermsNotSigned(err)
	if err == nil || termsExpired {
		appCfg, err := d.updater.AppConfig(req.Context())
		if err != nil {
			// TODO: log at least
			return res, err
		}

		var mustSaveAppCfg bool
		if termsExpired && !appCfg.MDM.AppleBMTermsExpired {
			// flag the AppConfig that the terms have changed and must be accepted
			appCfg.MDM.AppleBMTermsExpired = true
			mustSaveAppCfg = true
		} else if appCfg.MDM.AppleBMTermsExpired {
			// flag the AppConfig that the terms have been accepted
			appCfg.MDM.AppleBMTermsExpired = false
			mustSaveAppCfg = true
		}

		if mustSaveAppCfg {
			if err := d.updater.SaveAppConfig(req.Context(), appCfg); err != nil {
				// TODO: log at least
			}
		}
	}

	return res, err
}

// NewDEPClient creates an Apple DEP API HTTP client based on the provided
// storage that will flag the AppConfig's AppleBMTermsExpired field whenever
// the status of the terms changes.
func NewDEPClient(storage godep.ClientStorage, appCfgUpdater AppConfigUpdater) *godep.Client {
	return godep.NewClient(storage, fleethttp.NewClient(), godep.WithMiddleware(func(d nanodep_client.Doer) nanodep_client.Doer {
		return termsChangedDoer{doer: d, updater: appCfgUpdater}
	}))
}
