package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
)

func (svc *Service) getVPPToken(ctx context.Context) (string, error) {
	configMap, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetVPPToken})
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "fetching vpp token")
	}

	var vppTokenData fleet.VPPTokenData
	if err := json.Unmarshal(configMap[fleet.MDMAssetVPPToken].Value, &vppTokenData); err != nil {
		return "", ctxerr.Wrap(ctx, err, "unmarshaling VPP token data")
	}

	return base64.StdEncoding.EncodeToString([]byte(vppTokenData.Token)), nil
}

func (svc *Service) BatchAssociateVPPApps(ctx context.Context, teamName string, payloads []fleet.VPPBatchPayload, dryRun bool) error {
	if teamName == "" {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("team_name", "must not be empty"))
	}

	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return err
	}

	team, err := svc.ds.TeamByName(ctx, teamName)
	if err != nil {
		// If this is a dry run, the team may not have been created yet
		if dryRun && fleet.IsNotFound(err) {
			return nil
		}
		return err
	}

	if team == nil {
		return ctxerr.Errorf(ctx, "team required for VPP app assocoation")
	}

	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: &team.ID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "validating authorization")
	}

	token, err := svc.getVPPToken(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving stored vpp token")
	}

	_, err = vpp.GetConfig(token)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating vpp token")
	}

	var adamIDs []string

	for _, payload := range payloads {
		adamIDs = append(adamIDs, payload.AppStoreID)
	}

	filter := &vpp.AssetFilter{
		AdamID: strings.Join(adamIDs, ","),
	}
	assets, err := vpp.GetAssets(token, filter)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unable to retrieve assets")
	}

	var missingAssets []string
	for _, adamID := range adamIDs {
		var assetAvailable bool
		for _, asset := range assets {
			if adamID == asset.AdamID {
				assetAvailable = true
				break
			}
		}
		if !assetAvailable {
			missingAssets = append(missingAssets, adamID)
		}
	}

	if len(missingAssets) != 0 {
		return ctxerr.Errorf(ctx, "requested app store not available on vpp account: %s", strings.Join(missingAssets, ","))
	}

	// we cheked if the team is null earlier
	serials, err := svc.ds.GetTeamAppleSerialNumbers(ctx, team.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cannot get team serials for association")
	}

	if !dryRun {
		if err := vpp.AssociateAssets(token, &vpp.AssociateAssetsRequest{
			Assets:        assets,
			SerialNumbers: serials,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "failed to associate vpp assets")
		}
	}

	return nil
}
