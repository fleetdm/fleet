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
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	kitlog "github.com/go-kit/kit/log"
	"github.com/google/uuid"
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

func (svc *Service) DeleteMDMAppleBootstrapPackage(ctx context.Context, teamID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleBootstrapPackage{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	var ptrTeamName *string
	var ptrTeamId *uint
	if teamID >= 1 {
		tm, err := svc.teamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get team name for delete bootstrap package activity details")
		}
		ptrTeamName = &tm.Name
		ptrTeamId = &teamID
	}

	meta, err := svc.ds.GetMDMAppleBootstrapPackageMeta(ctx, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching bootstrap package metadata")
	}

	if err := svc.ds.DeleteMDMAppleBootstrapPackage(ctx, teamID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting bootstrap package")
	}

	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedBootstrapPackage{BootstrapPackageName: meta.Name, TeamID: ptrTeamId, TeamName: ptrTeamName}); err != nil {
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
