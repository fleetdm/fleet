package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	mdmcrypto "github.com/fleetdm/fleet/v4/server/mdm/crypto"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
)

func (svc *Service) GetMDMAccountDrivenEnrollmentSSOURL(ctx context.Context, enrollmentToken string) (string, error) {
	// skipauth: The enroll profile endpoint is unauthenticated.
	svc.authz.SkipAuthorization(ctx)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err)
	}
	url := appConfig.MDMUrl() + "/mdm/apple/account_driven_enroll/sso"

	if enrollmentToken != "" {
		url = fmt.Sprintf("%s/%s", url, enrollmentToken)
	}

	return url, nil
}

func (svc *Service) GetMDMAppleAccountEnrollmentProfile(ctx context.Context, enrollRef string) (profile []byte, err error) {
	// skipauth: This enrollment endpoint is authenticated only by the enrollment reference.
	svc.authz.SkipAuthorization(ctx)

	enrollChallenge, err := svc.ds.ConsumeADUEEnrollmentChallenge(ctx, enrollRef)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "consuming account driven enrollment challenge")
	}
	if enrollChallenge == nil {
		return nil, &fleet.BadRequestError{Message: "account driven enrollment challenge not found"}
	}

	idpAccount, err := svc.ds.GetMDMIdPAccountByUUID(ctx, enrollChallenge.IdPAccountUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting MDM IdP account by UUID")
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	topic, err := apple_mdm.MDMPushCertTopic(ctx, svc.ds)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "extracting topic from APNs cert")
	}

	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetSCEPChallenge,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("loading SCEP challenge from the database: %w", err)
	}
	enrollURL := appConfig.MDMUrl()

	enrollmentProf, err := apple_mdm.GenerateAccountDrivenEnrollmentProfileMobileconfig(
		appConfig.OrgInfo.OrgName,
		enrollURL,
		string(assets[fleet.MDMAssetSCEPChallenge].Value),
		topic,
		idpAccount.Email,
		true, // fresh enrollment
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating enrollment profile")
	}

	signed, err := mdmcrypto.Sign(ctx, enrollmentProf, svc.ds)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "signing profile")
	}

	return signed, nil
}

func (svc *Service) ListAppleDDMAssets(ctx context.Context, teamID *uint) ([]*fleet.DDMAsset, error) {
	if err := svc.authz.Authorize(ctx, &fleet.DDMAssetAuthz{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	assets, err := svc.ds.ListAppleDDMAssets(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing Apple DDM assets")
	}

	return assets, nil
}

func (svc *Service) GetAppleDDMAsset(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
	if authzErr := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); authzErr != nil {
		return nil, authzErr
	}

	asset, err := svc.ds.GetAppleDDMAsset(ctx, assetUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting Apple DDM asset")
	}

	if authzErr := svc.authz.Authorize(ctx, &fleet.DDMAssetAuthz{TeamID: asset.TeamID}, fleet.ActionRead); authzErr != nil {
		// We return a not found error here to avoid leaking the existence of the asset to unauthorized users.
		return nil, common_mysql.NotFound("Asset").WithName(assetUUID)
	}

	return asset, nil
}

func (svc *Service) DownloadAppleDDMAsset(ctx context.Context, assetUUID string) (name string, data []byte, err error) {
	if authzErr := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); authzErr != nil {
		return "", nil, authzErr
	}

	asset, err := svc.ds.GetAppleDDMAssetForDownload(ctx, assetUUID)
	if err != nil {
		return "", nil, ctxerr.Wrap(ctx, err, "getting Apple DDM asset")
	}

	if authzErr := svc.authz.Authorize(ctx, &fleet.DDMAssetAuthz{TeamID: asset.TeamID}, fleet.ActionRead); authzErr != nil {
		// We return a not found error here to avoid leaking the existence of the asset to unauthorized users.
		return "", nil, common_mysql.NotFound("Asset").WithName(assetUUID)
	}

	return asset.Name + ".json", asset.Data, nil
}

func (svc *Service) CreateAppleDDMAsset(ctx context.Context, teamID *uint, name string, data []byte) (string, error) {
	if authzErr := svc.authz.Authorize(ctx, &fleet.DDMAssetAuthz{TeamID: teamID}, fleet.ActionWrite); authzErr != nil {
		return "", authzErr
	}

	identifier, err := svc.validateAppleDDMAsset(ctx, data)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "validating Apple DDM asset")
	}

	assetUUID, err := svc.ds.CreateAppleDDMAsset(ctx, name, identifier, data, teamID)
	if err != nil {
		if alreadyExistsErr, ok := err.(fleet.AlreadyExistsError); ok && alreadyExistsErr.IsExists() {
			switch {
			case strings.Contains(alreadyExistsErr.Error(), "asset_name"):
				return "", &fleet.ConflictError{Message: fmt.Sprintf("An asset with the name %q already exists for this team", name)}
			case strings.Contains(alreadyExistsErr.Error(), "asset_identifier"):
				return "", &fleet.ConflictError{Message: fmt.Sprintf("An asset with the identifier %q already exists for this team", identifier)}
			}
		}
		return "", ctxerr.Wrap(ctx, err, "creating Apple DDM asset")
	}

	return assetUUID, nil
}

func (svc *Service) validateAppleDDMAsset(ctx context.Context, data []byte) (identifier string, err error) {
	var rawAsset fleet.RawDDMAsset
	if err := json.Unmarshal(data, &rawAsset); err != nil {
		return "", ctxerr.Wrap(ctx, err, "unmarshaling asset data")
	}

	if rawAsset.Identifier == "" {
		return "", &fleet.BadRequestError{Message: "Asset must contain a non-empty identifier"}
	}

	if !strings.HasPrefix(rawAsset.Type, "com.apple.asset.") {
		return "", &fleet.BadRequestError{Message: "Asset type must be a valid Apple asset type beginning with 'com.apple.asset.'"}
	}

	// We disallow authentication, as we force MDM auth when serving the assets.
	if rawAsset.Payload.Authentication != nil {
		return "", &fleet.BadRequestError{Message: "Asset payload must not contain an authentication key"}
	}

	if rawAsset.Payload.Reference.DataURL == "" {
		return "", &fleet.BadRequestError{Message: "Asset payload must contain a non-empty reference data URL"}
	}

	if _, err := url.ParseRequestURI(rawAsset.Payload.Reference.DataURL); err != nil {
		return "", &fleet.BadRequestError{Message: fmt.Sprintf("Invalid payload data URL: %v", err)}
	}

	return rawAsset.Identifier, nil
}

func (svc *Service) DeleteAppleDDMAsset(ctx context.Context, assetUUID string) error {
	if authzErr := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); authzErr != nil {
		return authzErr
	}

	asset, err := svc.ds.GetAppleDDMAsset(ctx, assetUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting Apple DDM asset")
	}

	if authzErr := svc.authz.Authorize(ctx, &fleet.DDMAssetAuthz{TeamID: asset.TeamID}, fleet.ActionWrite); authzErr != nil {
		// We return a not found error here to avoid leaking the existence of the asset to unauthorized users.
		return common_mysql.NotFound("Asset").WithName(assetUUID)
	}

	if err := svc.ds.DeleteAppleDDMAsset(ctx, assetUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting Apple DDM asset")
	}

	return nil
}
