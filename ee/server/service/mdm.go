package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/sso"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/micromdm/nanodep/godep"
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

	appCfg, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return nil, err
	}
	mdmServerURL, err := apple_mdm.ResolveAppleMDMURL(appCfg.ServerSettings.ServerURL)
	if err != nil {
		return nil, err
	}
	tok, err := svc.config.MDM.AppleBM()
	if err != nil {
		return nil, err
	}

	appleBM, err := getAppleBMAccountDetail(ctx, svc.depStorage, svc.ds, svc.logger)
	if err != nil {
		return nil, err
	}

	// fill the rest of the AppleBM fields
	appleBM.RenewDate = tok.AccessTokenExpiry
	appleBM.DefaultTeam = appCfg.MDM.AppleBMDefaultTeam
	appleBM.MDMServerURL = mdmServerURL

	return appleBM, nil
}

func getAppleBMAccountDetail(ctx context.Context, depStorage storage.AllStorage, ds fleet.Datastore, logger kitlog.Logger) (*fleet.AppleBM, error) {
	depClient := apple_mdm.NewDEPClient(depStorage, ds, logger)
	res, err := depClient.AccountDetail(ctx, apple_mdm.DEPName)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "apple GET /account request failed")
	}

	if res.AdminID == "" {
		// fallback to facilitator ID, as this is the same information but for
		// older versions of the Apple API.
		// https://github.com/fleetdm/fleet/issues/7515#issuecomment-1346579398
		res.AdminID = res.FacilitatorID
	}
	return &fleet.AppleBM{
		AppleID: res.AdminID,
		OrgName: res.OrgName,
	}, nil
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

	// TODO: save the pin (first return value) in the database
	err = svc.mdmAppleCommander.DeviceLock(ctx, []string{host.UUID}, uuid.New().String())
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

	// TODO: save the pin (first return value) in the database
	err = svc.mdmAppleCommander.EraseDevice(ctx, []string{host.UUID}, uuid.New().String())
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) MDMAppleEnableFileVaultAndEscrow(ctx context.Context, teamID *uint) error {
	cert, _, _, err := svc.config.MDM.AppleSCEP()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enabling FileVault")
	}

	var contents bytes.Buffer
	params := fileVaultProfileOptions{
		PayloadIdentifier:    mobileconfig.FleetFileVaultPayloadIdentifier,
		Base64DerCertificate: base64.StdEncoding.EncodeToString(cert.Leaf.Raw),
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
	if err := svc.ds.InsertMDMAppleBootstrapPackage(ctx, bp); err != nil {
		return err
	}

	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeAddedBootstrapPackage{BootstrapPackageName: name, TeamID: ptrTeamId, TeamName: ptrTeamName}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for upload bootstrap package")
	}

	return nil
}

func (svc *Service) GetMDMAppleBootstrapPackageBytes(ctx context.Context, token string) (*fleet.MDMAppleBootstrapPackage, error) {
	// skipauth: bootstrap packages are gated by token
	svc.authz.SkipAuthorization(ctx)

	pkg, err := svc.ds.GetMDMAppleBootstrapPackageBytes(ctx, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return pkg, nil
}

func (svc *Service) GetMDMAppleBootstrapPackageMetadata(ctx context.Context, teamID uint) (*fleet.MDMAppleBootstrapPackage, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleBootstrapPackage{TeamID: teamID}, fleet.ActionRead); err != nil {
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

	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedBootstrapPackage{BootstrapPackageName: meta.Name, TeamID: ptrTeamID, TeamName: ptrTeamName}); err != nil {
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

func (svc *Service) MDMAppleCreateEULA(ctx context.Context, name string, f io.ReadSeeker) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEULA{}, fleet.ActionWrite); err != nil {
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

	eula := &fleet.MDMAppleEULA{
		Name:  name,
		Token: uuid.New().String(),
		Bytes: bytes,
	}

	if err := svc.ds.MDMAppleInsertEULA(ctx, eula); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting EULA")
	}

	return nil
}

func (svc *Service) MDMAppleGetEULABytes(ctx context.Context, token string) (*fleet.MDMAppleEULA, error) {
	// skipauth: this resource is authorized using the token provided in the
	// request.
	svc.authz.SkipAuthorization(ctx)

	return svc.ds.MDMAppleGetEULABytes(ctx, token)
}

func (svc *Service) MDMAppleDeleteEULA(ctx context.Context, token string) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEULA{}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.MDMAppleDeleteEULA(ctx, token); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting EULA")
	}

	return nil
}

func (svc *Service) MDMAppleGetEULAMetadata(ctx context.Context) (*fleet.MDMAppleEULA, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEULA{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	eula, err := svc.ds.MDMAppleGetEULAMetadata(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting EULA metadata")
	}

	return eula, nil
}

func (svc *Service) SetOrUpdateMDMAppleSetupAssistant(ctx context.Context, asst *fleet.MDMAppleSetupAssistant) (*fleet.MDMAppleSetupAssistant, error) {
	if err := svc.authz.Authorize(ctx, asst, fleet.ActionWrite); err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(asst.Profile, &m); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "json unmarshal setup assistant profile")
	}

	deniedFields := map[string]string{
		"configuration_web_url":   `Couldn’t edit macos_setup_assistant. The automatic enrollment profile can’t include configuration_web_url. To require end user authentication, use the macos_setup.end_user_authentication option.`,
		"await_device_configured": `Couldn’t edit macos_setup_assistant. The automatic enrollment profile can’t include await_device_configured.`,
		"url":                     `Couldn’t edit macos_setup_assistant. The automatic enrollment profile can’t include url.`,
	}
	for k, msg := range deniedFields {
		if _, ok := m[k]; ok {
			return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("profile", msg))
		}
	}
	// TODO(mna): svc.depService.RegisterProfileWithAppleDEPServer()

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
		var teamName *string
		if newAsst.TeamID != nil {
			tm, err := svc.ds.Team(ctx, *newAsst.TeamID)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get team")
			}
			teamName = &tm.Name
		}
		if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeChangedMacosSetupAssistant{
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
		var teamName *string
		if teamID != nil {
			tm, err := svc.ds.Team(ctx, *teamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get team")
			}
			teamName = &tm.Name
		}
		if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeDeletedMacosSetupAssistant{
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

	serverURL := appConfig.ServerSettings.ServerURL
	authSettings := sso.Settings{
		Metadata:                    metadata,
		AssertionConsumerServiceURL: serverURL + svc.config.Server.URLPrefix + "/api/v1/fleet/mdm/sso/callback",
		SessionStore:                svc.ssoSessionStore,
		OriginalURL:                 "/api/v1/fleet/mdm/sso/callback",
	}

	idpURL, err := sso.CreateAuthorizationRequest(&authSettings, settings.EntityID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "InitiateSSO creating authorization")
	}

	return idpURL, nil

}

func (svc *Service) InitiateMDMAppleSSOCallback(ctx context.Context, auth fleet.Auth) ([]byte, error) {
	// skipauth: User context does not yet exist. Unauthenticated users may
	// hit the SSO callback.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithNoUser(ctx), level.Info)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get config for sso")
	}

	_, metadata, err := svc.ssoSessionStore.Fullfill(auth.RequestID())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate request in session")
	}

	err = sso.ValidateAudiences(
		*metadata,
		auth,
		appConfig.SSOSettings.EntityID,
		appConfig.ServerSettings.ServerURL,
		appConfig.ServerSettings.ServerURL+svc.config.Server.URLPrefix+"/api/v1/fleet/mdm/sso/callback",
	)

	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating sso response")
	}

	return apple_mdm.GenerateEnrollmentProfileMobileconfig(
		appConfig.OrgInfo.OrgName,
		appConfig.ServerSettings.ServerURL,
		svc.config.MDM.AppleSCEPChallenge,
		svc.mdmPushCertTopic,
	)
}

func (svc *Service) mdmAppleSyncDEPProfile(ctx context.Context) error {
	profiles, err := svc.ds.ListMDMAppleEnrollmentProfiles(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing profiles")
	}

	// Grab the first automatic enrollment profile we find, the current
	// behavior is that the last enrollment profile that was uploaded is
	// the one assigned to newly enrolled devices.
	//
	// TODO: this will change after #10995 where there can be a DEP profile
	// per team.
	var depProf *fleet.MDMAppleEnrollmentProfile
	for _, prof := range profiles {
		if prof.Type == "automatic" {
			depProf = prof
			break
		}
	}

	if depProf == nil {
		return svc.depService.CreateDefaultProfile(ctx)
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching app config")
	}

	enrollURL, err := apple_mdm.EnrollURL(depProf.Token, appCfg)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "generating enroll URL")
	}

	var jsonProf *godep.Profile
	if err := json.Unmarshal(*depProf.DEPProfile, &jsonProf); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshalling DEP profile")
	}

	return svc.depService.RegisterProfileWithAppleDEPServer(ctx, jsonProf, enrollURL)
}
