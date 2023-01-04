package service

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/storage"
)

func (svc *Service) GetAppleBM(ctx context.Context) (*fleet.AppleBM, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleBM{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	// if there is no apple bm config, fail with a 404
	if !svc.config.MDM.IsAppleBMSet() {
		return nil, notFoundError{}
	}

	appCfg, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	tok, err := svc.config.MDM.AppleBM()
	if err != nil {
		return nil, err
	}

	appleBM, err := getAppleBMAccountDetail(ctx, svc.depStorage)
	if err != nil {
		return nil, err
	}

	// fill the rest of the AppleBM fields
	appleBM.RenewDate = tok.AccessTokenExpiry
	// TODO: default team will have to be set when https://github.com/fleetdm/fleet/issues/8733
	// is implemented.
	appleBM.DefaultTeam = appCfg.MDM.AppleBMDefaultTeam
	appleBM.MDMServerURL = appCfg.ServerSettings.ServerURL + apple_mdm.MDMPath

	return appleBM, nil
}

func getAppleBMAccountDetail(ctx context.Context, depStorage storage.AllStorage) (*fleet.AppleBM, error) {
	httpClient := fleethttp.NewClient()
	depTransport := client.NewTransport(httpClient.Transport, httpClient, depStorage, nil)
	depClient := client.NewClient(fleethttp.NewClient(), depTransport)

	req, err := client.NewRequestWithContext(ctx, apple_mdm.DEPName, depStorage, "GET", "/account", nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create apple GET /account request")
	}
	res, err := depClient.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "execute apple GET /account request")
	}
	defer res.Body.Close()

	// TODO: if it fails in a way that indicates the token is invalid/expired
	// (403 Forbidden), eventually we should surface that to the user.
	if res.StatusCode >= 400 {
		// read up to 512 bytes of the response body to get better error message if possible
		body, _ := ioutil.ReadAll(io.LimitReader(res.Body, 512))
		return nil, ctxerr.Wrapf(ctx, err, "apple GET /account request failed: status: %d; body: %s", res.StatusCode, string(body))
	}

	var account struct {
		AdminID       string `json:"admin_id"`
		FacilitatorID string `json:"facilitator_id"`
		OrgName       string `json:"org_name"`
	}
	if err := json.NewDecoder(res.Body).Decode(&account); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decode apple GET /account response")
	}

	if account.AdminID == "" {
		// fallback to facilitator ID, as this is the same information but for
		// older versions of the Apple API.
		// https://github.com/fleetdm/fleet/issues/7515#issuecomment-1346579398
		account.AdminID = account.FacilitatorID
	}
	return &fleet.AppleBM{
		AppleID: account.AdminID,
		OrgName: account.OrgName,
	}, nil
}
