package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
)

// Deprecated: use the new GetABMTokens API endpoint.
func (svc *Service) GetAppleBM(ctx context.Context) (*fleet.AppleBM, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleBM{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	appCfg, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return nil, err
	}
	mdmServerURL, err := apple_mdm.ResolveAppleMDMURL(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return nil, err
	}

	tokens, err := svc.ds.ListABMTokens(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "GetAppleBM: listing ABM tokens")
	}

	if len(tokens) == 0 {
		return nil, notFoundError{}
	}

	if len(tokens) > 1 {
		return nil, errors.New("This API endpoint has been deprecated. Please use the new GET /abm_tokens API endpoint documented here: https://fleetdm.com/learn-more-about/apple-business-manager-tokens-api")
	}

	abmToken := tokens[0]

	var appleBM fleet.AppleBM
	// transfer the abmToken metadata info to the appleBM struct
	appleBM.AppleID = abmToken.AppleID
	appleBM.OrgName = abmToken.OrganizationName
	appleBM.RenewDate = abmToken.RenewAt
	appleBM.DefaultTeam = appCfg.MDM.DeprecatedAppleBMDefaultTeam
	appleBM.MDMServerURL = mdmServerURL

	return &appleBM, nil
}

func (svc *Service) MDMAppleDeviceLock(ctx context.Context, hostID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "host lite")
	}

	// TODO: define and use right permissions according to the spec.
	if err := svc.authz.Authorize(ctx, host, fleet.ActionWrite); err != nil {
		return err
	}

	_, err = svc.mdmAppleCommander.DeviceLock(ctx, host, uuid.New().String())
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) MDMAppleEraseDevice(ctx context.Context, hostID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "host lite")
	}

	// TODO: define and use right permissions according to the spec.
	if err := svc.authz.Authorize(ctx, host, fleet.ActionWrite); err != nil {
		return err
	}

	err = svc.mdmAppleCommander.EraseDevice(ctx, host, uuid.New().String())
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) MDMListHostConfigurationProfiles(ctx context.Context, hostID uint) ([]*fleet.MDMAppleConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionSelectiveList); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "find host to list profiles")
	}

	var tmID uint
	if host.TeamID != nil {
		tmID = *host.TeamID
	}

	// NOTE: the service method also does all the right authorization checks
	sums, err := svc.ListMDMAppleConfigProfiles(ctx, tmID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list config profiles")
	}

	return sums, nil
}

func (svc *Service) MDMAppleEnableFileVaultAndEscrow(ctx context.Context, teamID *uint) error {
	cert, err := assets.X509Cert(ctx, svc.ds, fleet.MDMAssetCACert)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving CA cert")
	}

	var contents bytes.Buffer
	params := fileVaultProfileOptions{
		PayloadIdentifier:    mobileconfig.FleetFileVaultPayloadIdentifier,
		PayloadName:          mdm.FleetFileVaultProfileName,
		Base64DerCertificate: base64.StdEncoding.EncodeToString(cert.Raw),
	}
	if err := fileVaultProfileTemplate.Execute(&contents, params); err != nil {
		return ctxerr.Wrap(ctx, err, "enabling FileVault")
	}

	cp, err := fleet.NewMDMAppleConfigProfile(contents.Bytes(), teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enabling FileVault")
	}

	_, err = svc.ds.NewMDMAppleConfigProfile(ctx, *cp)
	return ctxerr.Wrap(ctx, err, "enabling FileVault")
}

func (svc *Service) MDMAppleDisableFileVaultAndEscrow(ctx context.Context, teamID *uint) error {
	err := svc.ds.DeleteMDMAppleConfigProfileByTeamAndIdentifier(ctx, teamID, mobileconfig.FleetFileVaultPayloadIdentifier)
	return ctxerr.Wrap(ctx, err, "disabling FileVault")
}

func (svc *Service) UpdateMDMAppleSetup(ctx context.Context, payload fleet.MDMAppleSetupPayload) error {
	if err := svc.authz.Authorize(ctx, payload, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.validateMDMAppleSetupPayload(ctx, payload); err != nil {
		return err
	}

	if payload.TeamID != nil && *payload.TeamID != 0 {
		tm, err := svc.teamByIDOrName(ctx, payload.TeamID, nil)
		if err != nil {
			return err
		}
		return svc.updateTeamMDMAppleSetup(ctx, tm, payload)
	}
	return svc.updateAppConfigMDMAppleSetup(ctx, payload)
}

func (svc *Service) updateAppConfigMDMAppleSetup(ctx context.Context, payload fleet.MDMAppleSetupPayload) error {
	// appconfig is only used internally, it's fine to read it unobfuscated
	// (svc.AppConfigObfuscated must not be used because the write-only users
	// such as gitops will fail to access it).
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}

	var didUpdate, didUpdateMacOSEndUserAuth bool
	if payload.EnableEndUserAuthentication != nil {
		if ac.MDM.MacOSSetup.EnableEndUserAuthentication != *payload.EnableEndUserAuthentication {
			ac.MDM.MacOSSetup.EnableEndUserAuthentication = *payload.EnableEndUserAuthentication
			didUpdate = true
			didUpdateMacOSEndUserAuth = true
		}
	}

	if payload.EnableReleaseDeviceManually != nil {
		if ac.MDM.MacOSSetup.EnableReleaseDeviceManually.Value != *payload.EnableReleaseDeviceManually {
			ac.MDM.MacOSSetup.EnableReleaseDeviceManually = optjson.SetBool(*payload.EnableReleaseDeviceManually)
			didUpdate = true
		}
	}

	if didUpdate {
		if err := svc.ds.SaveAppConfig(ctx, ac); err != nil {
			return err
		}
		if didUpdateMacOSEndUserAuth {
			if err := svc.updateMacOSSetupEnableEndUserAuth(ctx, ac.MDM.MacOSSetup.EnableEndUserAuthentication, nil, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (svc *Service) updateMacOSSetupEnableEndUserAuth(ctx context.Context, enable bool, teamID *uint, teamName *string) error {
	if _, err := worker.QueueMacosSetupAssistantJob(ctx, svc.ds, svc.logger, worker.MacosSetupAssistantUpdateProfile, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "queue macos setup assistant update profile job")
	}

	var act fleet.ActivityDetails
	if enable {
		act = fleet.ActivityTypeEnabledMacosSetupEndUserAuth{TeamID: teamID, TeamName: teamName}
	} else {
		act = fleet.ActivityTypeDisabledMacosSetupEndUserAuth{TeamID: teamID, TeamName: teamName}
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for macos enable end user auth change")
	}
	return nil
}

func (svc *Service) validateMDMAppleSetupPayload(ctx context.Context, payload fleet.MDMAppleSetupPayload) error {
	// appconfig is only used internally, it's fine to read it unobfuscated
	// (svc.AppConfigObfuscated must not be used because the write-only users
	// such as gitops will fail to access it).
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.MDM.EnabledAndConfigured {
		return &fleet.MDMNotConfiguredError{}
	}

	if payload.EnableEndUserAuthentication != nil && *payload.EnableEndUserAuthentication {
		if ac.MDM.EndUserAuthentication.IsEmpty() {
			// TODO: update this error message to include steps to resolve the issue once docs for IdP
			// config are available
			return fleet.NewInvalidArgumentError("enable_end_user_authentication",
				`Couldn't enable macos_setup.enable_end_user_authentication because no IdP is configured for MDM features.`)
		}

		hasCustomConfigurationWebURL, err := svc.HasCustomSetupAssistantConfigurationWebURL(ctx, payload.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking setup assistant configuration web url")
		}

		if hasCustomConfigurationWebURL {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("macos_setup.enable_end_user_authentication", fleet.EndUserAuthDEPWebURLConfiguredErrMsg))
		}
	}

	return nil
}

func (svc *Service) MDMAppleUploadBootstrapPackage(ctx context.Context, name string, pkg io.Reader, teamID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleBootstrapPackage{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	var ptrTeamName *string
	var ptrTeamId *uint
	if teamID >= 1 {
		tm, err := svc.teamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get team name for upload bootstrap package activity details")
		}
		ptrTeamName = &tm.Name
		ptrTeamId = &teamID
	}

	hashBuf := bytes.NewBuffer(nil)
	if err := file.CheckPKGSignature(io.TeeReader(pkg, hashBuf)); err != nil {
		msg := "invalid package"
		if errors.Is(err, file.ErrInvalidType) || errors.Is(err, file.ErrNotSigned) {
			msg = err.Error()
		}

		return &fleet.BadRequestError{
			Message:     msg,
			InternalErr: err,
		}
	}

	pkgBuf := bytes.NewBuffer(nil)
	hash := sha256.New()
	if _, err := io.Copy(hash, io.TeeReader(hashBuf, pkgBuf)); err != nil {
		return err
	}

	bp := &fleet.MDMAppleBootstrapPackage{
		TeamID: teamID,
		Name:   name,
		Token:  uuid.New().String(),
		Sha256: hash.Sum(nil),
		Bytes:  pkgBuf.Bytes(),
	}
	if err := svc.ds.InsertMDMAppleBootstrapPackage(ctx, bp, svc.bootstrapPackageStore); err != nil {
		return err
	}

	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx),
		fleet.ActivityTypeAddedBootstrapPackage{BootstrapPackageName: name, TeamID: ptrTeamId, TeamName: ptrTeamName},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for upload bootstrap package")
	}

	return nil
}

func (svc *Service) GetMDMAppleBootstrapPackageBytes(ctx context.Context, token string) (*fleet.MDMAppleBootstrapPackage, error) {
	// skipauth: bootstrap packages are gated by token
	svc.authz.SkipAuthorization(ctx)

	pkg, err := svc.ds.GetMDMAppleBootstrapPackageBytes(ctx, token, svc.bootstrapPackageStore)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return pkg, nil
}

func (svc *Service) GetMDMAppleBootstrapPackageMetadata(ctx context.Context, teamID uint, forUpdate bool) (*fleet.MDMAppleBootstrapPackage, error) {
	act := fleet.ActionRead
	if forUpdate {
		act = fleet.ActionWrite
	}
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleBootstrapPackage{TeamID: teamID}, act); err != nil {
		return nil, err
	}

	meta, err := svc.ds.GetMDMAppleBootstrapPackageMeta(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching bootstrap package metadata")
	}

	return meta, nil
}

func (svc *Service) DeleteMDMAppleBootstrapPackage(ctx context.Context, teamID *uint) error {
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleBootstrapPackage{TeamID: tmID}, fleet.ActionWrite); err != nil {
		return err
	}

	var ptrTeamID *uint
	var ptrTeamName *string
	if tmID >= 1 {
		tm, err := svc.teamByIDOrName(ctx, &tmID, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get team name for delete bootstrap package activity details")
		}
		ptrTeamID = &tm.ID
		ptrTeamName = &tm.Name
	}

	meta, err := svc.ds.GetMDMAppleBootstrapPackageMeta(ctx, tmID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching bootstrap package metadata")
	}

	if err := svc.ds.DeleteMDMAppleBootstrapPackage(ctx, tmID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting bootstrap package")
	}

	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedBootstrapPackage{BootstrapPackageName: meta.Name, TeamID: ptrTeamID, TeamName: ptrTeamName},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for delete bootstrap package")
	}

	return nil
}

func (svc *Service) GetMDMAppleBootstrapPackageSummary(ctx context.Context, teamID *uint) (*fleet.MDMAppleBootstrapPackageSummary, error) {
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleBootstrapPackage{TeamID: tmID}, fleet.ActionRead); err != nil {
		return &fleet.MDMAppleBootstrapPackageSummary{}, err
	}

	if teamID != nil {
		_, err := svc.ds.Team(ctx, tmID)
		if err != nil {
			return &fleet.MDMAppleBootstrapPackageSummary{}, err
		}
	}

	summary, err := svc.ds.GetMDMAppleBootstrapPackageSummary(ctx, tmID)
	if err != nil {
		return &fleet.MDMAppleBootstrapPackageSummary{}, ctxerr.Wrap(ctx, err, "getting bootstrap package summary")
	}

	return summary, nil
}

func (svc *Service) MDMCreateEULA(ctx context.Context, name string, f io.ReadSeeker) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMEULA{}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := file.CheckPDF(f); err != nil {
		if errors.Is(err, file.ErrInvalidType) {
			return &fleet.BadRequestError{
				Message:     err.Error(),
				InternalErr: err,
			}
		}

		return ctxerr.Wrap(ctx, err, "checking pdf")
	}

	// ensure we read the file from the start
	_, err := f.Seek(0, io.SeekStart)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "seeking start of PDF file")
	}

	bytes, err := io.ReadAll(f)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading EULA bytes")
	}

	eula := &fleet.MDMEULA{
		Name:  name,
		Token: uuid.New().String(),
		Bytes: bytes,
	}

	if err := svc.ds.MDMInsertEULA(ctx, eula); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting EULA")
	}

	return nil
}

func (svc *Service) MDMGetEULABytes(ctx context.Context, token string) (*fleet.MDMEULA, error) {
	// skipauth: this resource is authorized using the token provided in the
	// request.
	svc.authz.SkipAuthorization(ctx)

	return svc.ds.MDMGetEULABytes(ctx, token)
}

func (svc *Service) MDMDeleteEULA(ctx context.Context, token string) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMEULA{}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.MDMDeleteEULA(ctx, token); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting EULA")
	}

	return nil
}

func (svc *Service) MDMGetEULAMetadata(ctx context.Context) (*fleet.MDMEULA, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMEULA{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	eula, err := svc.ds.MDMGetEULAMetadata(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting EULA metadata")
	}

	return eula, nil
}

func (svc *Service) SetOrUpdateMDMAppleSetupAssistant(ctx context.Context, asst *fleet.MDMAppleSetupAssistant) (*fleet.MDMAppleSetupAssistant, error) {
	if err := svc.authz.Authorize(ctx, asst, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// In order to validate if a configuration_web_url can be set for this setup
	// assistant configuration, we need to know if end user authentication is
	// enabled (either globally or for a specific team, if provided)
	var endUserAuthEnabled bool
	var teamName *string
	var tm *fleet.Team
	if asst.TeamID != nil {
		var err error
		tm, err = svc.ds.Team(ctx, *asst.TeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get team")
		}
		teamName = &tm.Name
		endUserAuthEnabled = tm.Config.MDM.MacOSSetup.EnableEndUserAuthentication
	} else {
		appCfg, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting app config")
		}
		endUserAuthEnabled = appCfg.MDM.MacOSSetup.EnableEndUserAuthentication
	}

	var m map[string]any
	if err := json.Unmarshal(asst.Profile, &m); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "json unmarshal setup assistant profile")
	}
	if _, ok := m["configuration_web_url"]; ok && endUserAuthEnabled {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("profile", `Couldn't edit macos_setup_assistant. First, disable end user authentication before adding an automatic enrollment (DEP) profile with a configuration_web_url.`))
	}

	if _, ok := m["url"]; ok {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("profile", `Couldn't edit macos_setup_assistant. The automatic enrollment profile can't include url.`))
	}
	if _, ok := m["await_device_configured"]; ok {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("profile", `Couldn't edit macos_setup_assistant. The profile can't include "await_device_configured" option.`))
	}

	// Validate the profile with Apple's API. Don't save the profile if it isn't valid.
	err := svc.depService.ValidateSetupAssistant(ctx, tm, asst, "")
	if err != nil {
		return nil, fleet.NewInvalidArgumentError("profile", err.Error())
	}

	// must read the existing setup assistant first to detect if it did change
	// (so that the changed activity is not created if the same assistant was
	// uploaded).
	prevAsst, err := svc.ds.GetMDMAppleSetupAssistant(ctx, asst.TeamID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "get previous setup assistant")
	}
	newAsst, err := svc.ds.SetOrUpdateMDMAppleSetupAssistant(ctx, asst)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "set or update setup assistant")
	}

	// if the name is the same and the content did not change, uploaded at will stay the same
	if prevAsst == nil || newAsst.Name != prevAsst.Name || newAsst.UploadedAt.After(prevAsst.UploadedAt) {
		if _, err := worker.QueueMacosSetupAssistantJob(
			ctx,
			svc.ds,
			svc.logger,
			worker.MacosSetupAssistantProfileChanged,
			newAsst.TeamID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "enqueue macos setup assistant profile changed job")
		}

		if err := svc.NewActivity(
			ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeChangedMacosSetupAssistant{
				TeamID:   newAsst.TeamID,
				TeamName: teamName,
				Name:     newAsst.Name,
			}); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for changed macos setup assistant")
		}
	}
	return newAsst, nil
}

func (svc *Service) GetMDMAppleSetupAssistant(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleSetupAssistant{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}
	return svc.ds.GetMDMAppleSetupAssistant(ctx, teamID)
}

func (svc *Service) DeleteMDMAppleSetupAssistant(ctx context.Context, teamID *uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleSetupAssistant{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	// must read the existing setup assistant first to detect if it did delete
	// and to get the name of the deleted assistant.
	prevAsst, err := svc.ds.GetMDMAppleSetupAssistant(ctx, teamID)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "get previous setup assistant")
	}

	if err := svc.ds.DeleteMDMAppleSetupAssistant(ctx, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete setup assistant")
	}

	if prevAsst != nil {
		if _, err := worker.QueueMacosSetupAssistantJob(
			ctx,
			svc.ds,
			svc.logger,
			worker.MacosSetupAssistantProfileDeleted,
			teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "enqueue macos setup assistant profile deleted job")
		}

		var teamName *string
		if teamID != nil {
			tm, err := svc.ds.Team(ctx, *teamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get team")
			}
			teamName = &tm.Name
		}
		if err := svc.NewActivity(
			ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeDeletedMacosSetupAssistant{
				TeamID:   teamID,
				TeamName: teamName,
				Name:     prevAsst.Name,
			}); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for deleted macos setup assistant")
		}
	}

	return nil
}

func (svc *Service) InitiateMDMAppleSSO(ctx context.Context) (string, error) {
	// skipauth: User context does not yet exist. Unauthenticated users may
	// initiate SSO.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithNoUser(ctx), level.Info)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting app config")
	}

	settings := appConfig.MDM.EndUserAuthentication.SSOProviderSettings

	// For now, until we get to #10999, we assume that SSO is disabled if
	// no settings are provided.
	if settings.IsEmpty() {
		err := &fleet.BadRequestError{Message: "organization not configured to use sso"}
		return "", ctxerr.Wrap(ctx, err, "initiate mdm sso")
	}

	metadata, err := sso.GetMetadata(&settings)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "InitiateSSO getting metadata")
	}

	authSettings := sso.Settings{
		Metadata:                    metadata,
		AssertionConsumerServiceURL: appConfig.MDMUrl() + svc.config.Server.URLPrefix + "/api/v1/fleet/mdm/sso/callback",
		SessionStore:                svc.ssoSessionStore,
		OriginalURL:                 "/api/v1/fleet/mdm/sso/callback",
	}

	idpURL, err := sso.CreateAuthorizationRequest(&authSettings, settings.EntityID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "InitiateSSO creating authorization")
	}

	return idpURL, nil
}

func (svc *Service) InitiateMDMAppleSSOCallback(ctx context.Context, auth fleet.Auth) string {
	// skipauth: User context does not yet exist. Unauthenticated users may
	// hit the SSO callback.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithNoUser(ctx), level.Info)

	profileToken, enrollmentRef, eulaToken, err := svc.mdmSSOHandleCallbackAuth(ctx, auth)
	if err != nil {
		logging.WithErr(ctx, err)
		return apple_mdm.FleetUISSOCallbackPath + "?error=true"
	}

	q := url.Values{
		"profile_token":        {profileToken},
		"enrollment_reference": {enrollmentRef},
	}

	if eulaToken != "" {
		q.Add("eula_token", eulaToken)
	}

	return fmt.Sprintf("%s?%s", apple_mdm.FleetUISSOCallbackPath, q.Encode())
}

func (svc *Service) mdmSSOHandleCallbackAuth(ctx context.Context, auth fleet.Auth) (string, string, string, error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", "", "", ctxerr.Wrap(ctx, err, "get config for sso")
	}

	_, metadata, err := svc.ssoSessionStore.Fullfill(auth.RequestID())
	if err != nil {
		return "", "", "", ctxerr.Wrap(ctx, err, "validate request in session")
	}

	settings := appConfig.MDM.EndUserAuthentication.SSOProviderSettings
	// For now, until we get to #10999, we assume that SSO is disabled if
	// no settings are provided.
	if settings.IsEmpty() {
		err := &fleet.BadRequestError{Message: "organization not configured to use sso"}
		return "", "", "", ctxerr.Wrap(ctx, err, "get config for mdm sso callback")
	}

	err = sso.ValidateAudiences(
		*metadata,
		auth,
		settings.EntityID,
		appConfig.MDMUrl(),
		appConfig.MDMUrl()+svc.config.Server.URLPrefix+"/api/v1/fleet/mdm/sso/callback",
	)
	if err != nil {
		return "", "", "", ctxerr.Wrap(ctx, err, "validating sso response")
	}

	// Store information for automatic account population/creation
	//
	// For now, we just grab whatever comes before the `@` in UserID, which
	// must be an email.
	//
	// For more details, check https://github.com/fleetdm/fleet/issues/10744#issuecomment-1540605146
	username, _, found := strings.Cut(auth.UserID(), "@")
	if !found {
		svc.logger.Log("mdm-sso-callback", "IdP UserID doesn't look like an email, using raw value")
		username = auth.UserID()
	}

	err = svc.ds.InsertMDMIdPAccount(ctx, &fleet.MDMIdPAccount{
		Username: username,
		Fullname: auth.UserDisplayName(),
		Email:    auth.UserID(),
	})
	if err != nil {
		return "", "", "", ctxerr.Wrap(ctx, err, "saving account data from IdP")
	}

	idpAcc, err := svc.ds.GetMDMIdPAccountByEmail(
		// use the primary db as the account might have been just
		// inserted
		ctxdb.RequirePrimary(ctx, true),
		auth.UserID(),
	)
	if err != nil {
		return "", "", "", ctxerr.Wrap(ctx, err, "retrieving new account data from IdP")
	}

	eula, err := svc.ds.MDMGetEULAMetadata(ctx)
	if err != nil && !fleet.IsNotFound(err) {
		return "", "", "", ctxerr.Wrap(ctx, err, "getting EULA metadata")
	}

	var eulaToken string
	if eula != nil {
		eulaToken = eula.Token
	}

	// get the automatic profile to access the authentication token.
	depProf, err := svc.getAutomaticEnrollmentProfile(ctx)
	if err != nil {
		return "", "", "", ctxerr.Wrap(ctx, err, "listing profiles")
	}

	if depProf == nil {
		return "", "", "", ctxerr.Wrap(ctx, err, "missing profile")
	}

	// using the idp token as a reference just because that's the
	// only thing we're referencing later on during enrollment.
	return depProf.Token, idpAcc.UUID, eulaToken, nil
}

func (svc *Service) mdmAppleSyncDEPProfiles(ctx context.Context) error {
	if _, err := worker.QueueMacosSetupAssistantJob(ctx, svc.ds, svc.logger, worker.MacosSetupAssistantUpdateAllProfiles, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "queue macos setup assistant update all profiles job")
	}
	return nil
}

// returns the default automatic enrollment profile, or nil (without error) if none exists.
func (svc *Service) getAutomaticEnrollmentProfile(ctx context.Context) (*fleet.MDMAppleEnrollmentProfile, error) {
	prof, err := svc.ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "get automatic profile")
	}
	return prof, nil
}

func (svc *Service) MDMApplePreassignProfile(ctx context.Context, payload fleet.MDMApplePreassignProfilePayload) error {
	// for the preassign and match features, we don't know yet what team(s) will
	// be affected, so we authorize only users with write-access to the no-team
	// config profiles and with team-write access.
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	if err := svc.profileMatcher.PreassignProfile(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "preassign profile")
	}
	return nil
}

func (svc *Service) MDMAppleMatchPreassignment(ctx context.Context, externalHostIdentifier string) error {
	// for the preassign and match features, we don't know yet what team(s) will
	// be affected, so we authorize only users with write-access to the no-team
	// config profiles and with team-write access.
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	profs, err := svc.profileMatcher.RetrieveProfiles(ctx, externalHostIdentifier)
	if err != nil {
		return err
	}
	if len(profs.Profiles) == 0 || profs.HostUUID == "" {
		return nil // nothing to do
	}

	// force-use the primary DB instance for all the following reads/writes
	// (we may create a team and need to read it back, but also to get latest
	// team matching the profiles)
	ctx = ctxdb.RequirePrimary(ctx, true)

	// load the host and ensure it is enrolled in Fleet MDM
	host, err := svc.ds.HostByIdentifier(ctx, profs.HostUUID)
	if err != nil {
		return err // will return a not found error if host does not exist
	}

	connected, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if host is connected to Fleet")
	}
	if !connected {
		err = errors.New("host is not enrolled in Fleet MDM")
		return ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     err.Error(),
			InternalErr: err,
		})
	}

	// Collect the profiles' groups in case we need to create a new team,
	// and the list of raw profiles bytes.
	groups, rawProfiles := make([]string, 0, len(profs.Profiles)),
		make([][]byte, 0, len(profs.Profiles))
	for _, prof := range profs.Profiles {
		if prof.Group != "" {
			groups = append(groups, prof.Group)
		}

		if !prof.Exclude {
			rawProfiles = append(rawProfiles, prof.Profile)
		}
	}

	team, err := svc.getOrCreatePreassignTeam(ctx, groups)
	if err != nil {
		return err
	}

	// create profiles for that team via the service call, so that uniqueness
	// of profile identifier/name is verified, activity created, etc.
	if err := svc.BatchSetMDMAppleProfiles(ctx, &team.ID, nil, rawProfiles, false, true); err != nil {
		return err
	}

	// assign host to that team via the service call, which will trigger
	// deployment of the profiles.
	if err := svc.AddHostsToTeam(ctx, &team.ID, []uint{host.ID}, true); err != nil {
		return err
	}

	return nil
}

func (svc *Service) getOrCreatePreassignTeam(ctx context.Context, groups []string) (*fleet.Team, error) {
	teamName := teamNameFromPreassignGroups(groups)
	team, err := svc.ds.TeamByName(ctx, teamName)
	if err != nil {
		if !fleet.IsNotFound(err) {
			return nil, err
		}

		// Create a new team for this set of groups. Creating via the service
		// call so that it properly assigns the agent options and creates audit
		// activities, etc.
		payload := fleet.TeamPayload{Name: &teamName}
		team, err = svc.NewTeam(ctx, payload)
		if err != nil {
			return nil, err
		}

		// Get default bootstrap package and end user auth settings for no team.
		ac, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return nil, err
		}

		spec := &fleet.TeamSpec{
			Name: teamName,
			MDM: fleet.TeamSpecMDM{
				EnableDiskEncryption: optjson.SetBool(true),
				MacOSSetup: fleet.MacOSSetup{
					MacOSSetupAssistant: ac.MDM.MacOSSetup.MacOSSetupAssistant,
					// NOTE: BootstrapPackage gets set by
					// CopyDefaultMDMAppleBootstrapPackage below
					// BootstrapPackage:            ac.MDM.MacOSSetup.BootstrapPackage,
					EnableEndUserAuthentication: ac.MDM.MacOSSetup.EnableEndUserAuthentication,
					EnableReleaseDeviceManually: ac.MDM.MacOSSetup.EnableReleaseDeviceManually,
				},
			},
		}
		if _, err := svc.ApplyTeamSpecs(ctx, []*fleet.TeamSpec{spec}, fleet.ApplyTeamSpecOptions{}); err != nil {
			return nil, err
		}
		if err := svc.ds.CopyDefaultMDMAppleBootstrapPackage(ctx, ac, team.ID); err != nil {
			return nil, err
		}

		// get the global setup assistant contents (this is different
		// from MDM.MacOSSetup.MacOSSetupAssistant we set above, the
		// prior is the path to the file, this is the actual file
		// contents.
		asst, err := svc.ds.GetMDMAppleSetupAssistant(ctx, nil)
		if err != nil {
			// if "no team" doesn't have custom setup assistant
			// settings configured, this team won't have either.
			if fleet.IsNotFound(err) {
				return team, nil
			}
			return nil, ctxerr.Wrap(ctx, err, "get global setup assistant")

		}
		_, err = svc.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
			TeamID:  &team.ID,
			Name:    asst.Name,
			Profile: asst.Profile,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "set setup assistant for new team")
		}
	}
	return team, nil
}

// teamNameFromPreassignGroups returns the team name to use for a new team
// created to match the set of profiles preassigned to a host. The team name is
// derived from the "group" field provided with each request to pre-assign a
// profile to a host (in fleet.MDMApplePreassignProfilePayload). That field is
// optional, and empty groups are ignored.
func teamNameFromPreassignGroups(groups []string) string {
	const defaultName = "default"

	dedupeGroups := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if group != "" {
			dedupeGroups[group] = struct{}{}
		}
	}
	groups = groups[:0]
	for group := range dedupeGroups {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	if len(groups) == 0 {
		groups = []string{defaultName}
	}

	return strings.Join(groups, " - ")
}

func (svc *Service) GetMDMDiskEncryptionSummary(ctx context.Context, teamID *uint) (*fleet.MDMDiskEncryptionSummary, error) {
	if err := svc.authz.Authorize(ctx, fleet.MDMConfigProfileAuthz{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	var macOS fleet.MDMAppleFileVaultSummary
	if m, err := svc.ds.GetMDMAppleFileVaultSummary(ctx, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting filevault summary")
	} else if m != nil {
		macOS = *m
	}

	var windows fleet.MDMWindowsBitLockerSummary
	if w, err := svc.ds.GetMDMWindowsBitLockerSummary(ctx, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting bitlocker summary")
	} else if w != nil {
		windows = *w
	}

	linux, err := svc.ds.GetLinuxDiskEncryptionSummary(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting linux disk encryption summary")
	}

	return &fleet.MDMDiskEncryptionSummary{
		Verified: fleet.MDMPlatformsCounts{
			MacOS:   macOS.Verified,
			Windows: windows.Verified,
			Linux:   linux.Verified,
		},
		Verifying: fleet.MDMPlatformsCounts{
			MacOS:   macOS.Verifying,
			Windows: windows.Verifying,
		},
		ActionRequired: fleet.MDMPlatformsCounts{
			MacOS:   macOS.ActionRequired,
			Windows: windows.ActionRequired,
			Linux:   linux.ActionRequired,
		},
		Enforcing: fleet.MDMPlatformsCounts{
			MacOS:   macOS.Enforcing,
			Windows: windows.Enforcing,
		},
		Failed: fleet.MDMPlatformsCounts{
			MacOS:   macOS.Failed,
			Windows: windows.Failed,
			Linux:   linux.Failed,
		},
		RemovingEnforcement: fleet.MDMPlatformsCounts{
			MacOS:   macOS.RemovingEnforcement,
			Windows: windows.RemovingEnforcement,
		},
	}, nil
}

func (svc *Service) mdmAppleEditedAppleOSUpdates(ctx context.Context, teamID *uint, appleDevice fleet.AppleDevice, updates fleet.AppleOSUpdateSettings) error {
	const (
		softwareUpdateType        = `com.apple.configuration.softwareupdate.enforcement.specific`
		softwareUpdateIdentSuffix = `-software-update-94f4bbdf-f439-4fb1-8d27-ae1bb793e105`
	)
	var (
		softwareUpdateIdentifier string
		osUpdatesProfileName     string
		labelName                string
	)
	switch appleDevice {
	case fleet.MacOS:
		softwareUpdateIdentifier = "macos" + softwareUpdateIdentSuffix
		osUpdatesProfileName = mdm.FleetMacOSUpdatesProfileName
		labelName = fleet.BuiltinLabelMacOS14Plus // OS update DDMs are supported on macOS 14+ devices.
	case fleet.IOS:
		softwareUpdateIdentifier = "ios" + softwareUpdateIdentSuffix
		osUpdatesProfileName = mdm.FleetIOSUpdatesProfileName
		labelName = fleet.BuiltinLabelIOS
	case fleet.IPadOS:
		softwareUpdateIdentifier = "ipados" + softwareUpdateIdentSuffix
		osUpdatesProfileName = mdm.FleetIPadOSUpdatesProfileName
		labelName = fleet.BuiltinLabelIPadOS
	default:
		panic(fmt.Sprintf("invalid AppleDevice: %d", appleDevice))
	}

	if updates.MinimumVersion.Value == "" {
		// OS updates disabled, remove the profile
		if err := svc.ds.DeleteMDMAppleDeclarationByName(ctx, teamID, osUpdatesProfileName); err != nil {
			return err
		}
		var globalOrTeamID uint
		if teamID != nil {
			globalOrTeamID = *teamID
		}
		// This only sets profiles that haven't been queued by the cron to 'pending' (both removes and installs, which includes
		// the OS updates we just deleted). It doesn't have a functional difference because if you don't call this function
		// the cron will catch up, but it's important for the UX to mark them as pending immediately so it's reflected in the UI.
		if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{globalOrTeamID}, nil, nil); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
		}
		return nil
	}

	// OS updates enabled, create or update the profile with the current settings.

	rawDecl := []byte(fmt.Sprintf(`{
	"Identifier": %q,
	"Type": %q,
	"Payload": {
		"TargetOSVersion": %q,
		"TargetLocalDateTime": "%sT12:00:00"
	}
}`, softwareUpdateIdentifier, softwareUpdateType, updates.MinimumVersion.Value, updates.Deadline.Value))

	d := fleet.NewMDMAppleDeclaration(rawDecl, teamID, osUpdatesProfileName, softwareUpdateType, softwareUpdateIdentifier)

	// Associate the profile with the built-in label to ensure that the profile is applied to the targeted devices.
	lblIDs, err := svc.ds.LabelIDsByName(ctx, []string{labelName})
	if err != nil {
		return err
	}
	d.LabelsIncludeAll = []fleet.ConfigurationProfileLabel{
		{LabelName: labelName, LabelID: lblIDs[labelName]},
	}

	decl, err := svc.ds.SetOrUpdateMDMAppleDeclaration(ctx, d)
	if err != nil {
		return err
	}

	if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{decl.DeclarationUUID}, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending host declarations")
	}
	return nil
}

func (svc *Service) mdmWindowsEnableOSUpdates(ctx context.Context, teamID *uint, updates fleet.WindowsUpdates) error {
	var contents bytes.Buffer
	params := windowsOSUpdatesProfileOptions{
		Deadline:    updates.DeadlineDays.Value,
		GracePeriod: updates.GracePeriodDays.Value,
	}
	if err := windowsOSUpdatesProfileTemplate.Execute(&contents, params); err != nil {
		return ctxerr.Wrap(ctx, err, "enabling Windows OS updates")
	}

	err := svc.ds.SetOrUpdateMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		TeamID: teamID,
		Name:   mdm.FleetWindowsOSUpdatesProfileName,
		SyncML: contents.Bytes(),
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create Windows OS updates profile")
	}

	return nil
}

func (svc *Service) mdmWindowsDisableOSUpdates(ctx context.Context, teamID *uint) error {
	err := svc.ds.DeleteMDMWindowsConfigProfileByTeamAndName(ctx, teamID, mdm.FleetWindowsOSUpdatesProfileName)
	return ctxerr.Wrap(ctx, err, "delete Windows OS updates profile")
}

func (svc *Service) GetMDMManualEnrollmentProfile(ctx context.Context) ([]byte, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleManualEnrollmentProfile{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	topic, err := assets.APNSTopic(ctx, svc.ds)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "extracting topic from APNs cert")
	}

	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetSCEPChallenge,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("loading SCEP challenge from the database: %w", err)
	}

	mobileConfig, err := apple_mdm.GenerateEnrollmentProfileMobileconfig(
		appConfig.OrgInfo.OrgName,
		appConfig.MDMUrl(),
		string(assets[fleet.MDMAssetSCEPChallenge].Value),
		topic,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// NOTE: the profile returned by this endpoint is intentionally not
	// signed so it can be modified and signed by the IT admin with a
	// custom certificate.
	//
	// Per @marko-lisica, we can add a parameter like `signed=true` if the
	// need arises.
	return mobileConfig, nil
}

func (svc *Service) UploadABMToken(ctx context.Context, token io.Reader) (*fleet.ABMToken, error) {
	encryptedToken, decryptedToken, err := svc.decryptUploadedABMToken(ctx, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypting uploaded ABM token")
	}

	if err := svc.authz.Authorize(ctx, &fleet.AppleBM{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	tok := &fleet.ABMToken{
		EncryptedToken: encryptedToken,
	}

	if err := apple_mdm.SetDecryptedABMTokenMetadata(ctx, tok, decryptedToken, svc.depStorage, svc.ds, svc.logger, false); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "setting ABM token metadata")
	}

	tok, err = svc.ds.InsertABMToken(ctx, tok)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "save ABM token")
	}

	// flip the app config flag
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "retrieving app config")
	}

	appCfg.MDM.AppleBMEnabledAndConfigured = true

	if err := svc.ds.SaveAppConfig(ctx, appCfg); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving app config after ABM enablement")
	}

	return tok, nil
}

func (svc *Service) DeleteABMToken(ctx context.Context, tokenID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppleBM{}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteABMToken(ctx, tokenID); err != nil {
		return ctxerr.Wrap(ctx, err, "removing ABM token")
	}

	count, err := svc.ds.GetABMTokenCount(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting ABM token count")
	}

	if count == 0 {
		// flip the app config flag
		appCfg, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving app config")
		}

		appCfg.MDM.AppleBMEnabledAndConfigured = false
		return svc.ds.SaveAppConfig(ctx, appCfg)
	}

	return nil
}

func (svc *Service) ListABMTokens(ctx context.Context) ([]*fleet.ABMToken, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleBM{}, fleet.ActionList); err != nil {
		return nil, err
	}

	tokens, err := svc.ds.ListABMTokens(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list ABM tokens")
	}

	return tokens, nil
}

func (svc *Service) CountABMTokens(ctx context.Context) (uint32, error) {
	// Authorizing using the more general AppConfig object because:
	// - this service method returns a count, which is not sensitive information
	// - gitops role, which needs this info, is not authorized for AppleBM access (as of 2024/12/02)
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return 0, err
	}

	tokens, err := svc.ds.CountABMTokens(ctx)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count ABM tokens")
	}

	return tokens, nil
}

func (svc *Service) UpdateABMTokenTeams(ctx context.Context, tokenID uint, macOSTeamID, iOSTeamID, iPadOSTeamID *uint) (*fleet.ABMToken, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleBM{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	token, err := svc.ds.GetABMTokenByID(ctx, tokenID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get ABM token to update teams")
	}

	// validate the team IDs

	token.MacOSTeam = fleet.ABMTokenTeam{Name: fleet.TeamNameNoTeam}
	token.MacOSDefaultTeamID = nil
	token.IOSTeam = fleet.ABMTokenTeam{Name: fleet.TeamNameNoTeam}
	token.IOSDefaultTeamID = nil
	token.IPadOSTeam = fleet.ABMTokenTeam{Name: fleet.TeamNameNoTeam}
	token.IPadOSDefaultTeamID = nil

	if macOSTeamID != nil && *macOSTeamID != 0 {
		macOSTeam, err := svc.ds.Team(ctx, *macOSTeamID)
		if err != nil {
			return nil, &fleet.BadRequestError{
				Message:     fmt.Sprintf("team with ID %d not found", *macOSTeamID),
				InternalErr: ctxerr.Wrap(ctx, err, "checking existence of macOS team"),
			}
		}

		token.MacOSTeam.Name = macOSTeam.Name
		token.MacOSTeam.ID = *macOSTeamID
		token.MacOSDefaultTeamID = macOSTeamID
	}

	if iOSTeamID != nil && *iOSTeamID != 0 {
		iOSTeam, err := svc.ds.Team(ctx, *iOSTeamID)
		if err != nil {
			return nil, &fleet.BadRequestError{
				Message:     fmt.Sprintf("team with ID %d not found", *iOSTeamID),
				InternalErr: ctxerr.Wrap(ctx, err, "checking existence of iOS team"),
			}
		}
		token.IOSTeam.Name = iOSTeam.Name
		token.IOSTeam.ID = *iOSTeamID
		token.IOSDefaultTeamID = iOSTeamID
	}

	if iPadOSTeamID != nil && *iPadOSTeamID != 0 {
		iPadOSTeam, err := svc.ds.Team(ctx, *iPadOSTeamID)
		if err != nil {
			return nil, &fleet.BadRequestError{
				Message:     fmt.Sprintf("team with ID %d not found", *iPadOSTeamID),
				InternalErr: ctxerr.Wrap(ctx, err, "checking existence of iPadOS team"),
			}
		}
		token.IPadOSTeam.Name = iPadOSTeam.Name
		token.IPadOSTeam.ID = *iPadOSTeamID
		token.IPadOSDefaultTeamID = iPadOSTeamID
	}

	if err := svc.ds.SaveABMToken(ctx, token); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating token teams in db")
	}

	return token, nil
}

func (svc *Service) RenewABMToken(ctx context.Context, token io.Reader, tokenID uint) (*fleet.ABMToken, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleBM{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	oldTok, err := svc.ds.GetABMTokenByID(ctx, tokenID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get ABM token by id")
	}

	encryptedToken, decryptedToken, err := svc.decryptUploadedABMToken(ctx, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypting ABM token for renewal")
	}

	if err := apple_mdm.SetDecryptedABMTokenMetadata(ctx, oldTok, decryptedToken, svc.depStorage, svc.ds, svc.logger, true); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "setting ABM token metadata")
	}

	oldTok.EncryptedToken = encryptedToken

	if err := svc.ds.SaveABMToken(ctx, oldTok); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "renewing ABM token")
	}

	return oldTok, nil
}

func (svc *Service) decryptUploadedABMToken(ctx context.Context, token io.Reader) (encryptedToken []byte, decryptedToken *client.OAuth1Tokens, err error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleBM{}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	pair, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetABMCert,
		fleet.MDMAssetABMKey,
	}, nil)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
				Message: "Please generate a keypair first.",
			}, "renewing ABM token")
		}

		return nil, nil, ctxerr.Wrap(ctx, err, "retrieving stored ABM assets")
	}

	encryptedToken, err = io.ReadAll(token)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "reading token bytes")
	}

	derCert, _ := pem.Decode(pair[fleet.MDMAssetABMCert].Value)
	if derCert == nil {
		return nil, nil, ctxerr.New(ctx, "ABM certificate in the database cannot be parsed")
	}

	cert, err := x509.ParseCertificate(derCert.Bytes)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "parsing ABM certificate")
	}

	decryptedToken, err = assets.DecryptRawABMToken(encryptedToken, cert, pair[fleet.MDMAssetABMKey].Value)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "Invalid token. Please provide a valid token from Apple Business Manager.",
			InternalErr: err,
		}, "validating ABM token")
	}
	return encryptedToken, decryptedToken, nil
}
