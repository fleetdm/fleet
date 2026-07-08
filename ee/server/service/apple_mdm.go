package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	mdmcrypto "github.com/fleetdm/fleet/v4/server/mdm/crypto"
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
