package service

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	mdmcrypto "github.com/fleetdm/fleet/v4/server/mdm/crypto"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
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

	identifier, _, err := svc.validateAppleDDMAsset(ctx, data)
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

	actTeamID, actTeamName, err := svc.assetActivityTeam(ctx, teamID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "resolving team for asset activity")
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeCreatedDeclarationAsset{
		AssetName: name,
		TeamID:    actTeamID,
		TeamName:  actTeamName,
	}); err != nil {
		return "", ctxerr.Wrap(ctx, err, "logging activity for created declaration asset")
	}

	return assetUUID, nil
}

// assetActivityTeam resolves the team id/name pointers to include in a DDM
// asset activity. Both are nil for the "no team" case (team 0 or nil).
func (svc *Service) assetActivityTeam(ctx context.Context, teamID *uint) (*uint, *string, error) {
	if teamID == nil || *teamID == 0 {
		return nil, nil, nil
	}
	tm, err := svc.ds.TeamLite(ctx, *teamID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "loading team for asset activity")
	}
	return teamID, &tm.Name, nil
}

func (svc *Service) validateAppleDDMAsset(ctx context.Context, data []byte) (identifier, assetType string, err error) {
	var rawAsset fleet.RawDDMAsset
	if err := json.Unmarshal(data, &rawAsset); err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "unmarshaling asset data")
	}

	if rawAsset.Identifier == "" {
		return "", "", &fleet.BadRequestError{Message: "Asset must contain a non-empty identifier"}
	}

	if !strings.HasPrefix(rawAsset.Type, "com.apple.asset.") {
		return "", "", &fleet.BadRequestError{Message: "Asset type must be a valid Apple asset type beginning with 'com.apple.asset.'"}
	}

	// Check if Identifier uses a FLEET_SECRET, fail if so.
	if strings.Contains(rawAsset.Identifier, "FLEET_SECRET") {
		return "", "", &fleet.BadRequestError{Message: "Asset identifier must not contain a $FLEET_SECRET"}
	}

	expanded, _, err := svc.ds.ExpandEmbeddedSecretsAndUpdatedAt(ctx, string(data))
	if err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "expanding embedded secrets and updated_at")
	}

	if err := json.Unmarshal([]byte(expanded), &rawAsset); err != nil {
		return "", "", ctxerr.Wrap(ctx, err, "unmarshaling asset data")
	}

	// We disallow authentication, as we force MDM auth when serving the assets.
	if rawAsset.Payload.Authentication != nil {
		return "", "", &fleet.BadRequestError{Message: "Asset payload must not contain an authentication key"}
	}

	if rawAsset.Payload.Reference.DataURL == "" {
		return "", "", &fleet.BadRequestError{Message: "Asset payload must contain a non-empty reference data URL"}
	}

	if _, err := url.ParseRequestURI(rawAsset.Payload.Reference.DataURL); err != nil {
		return "", "", &fleet.BadRequestError{Message: fmt.Sprintf("Invalid payload data URL: %v", err)}
	}

	return rawAsset.Identifier, rawAsset.Type, nil
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
		if fleet.IsForeignKey(err) {
			return &fleet.BadRequestError{Message: "Couldn't delete. A configuration profile is linked to this asset. Please delete the profile and try again."}
		}
		return ctxerr.Wrap(ctx, err, "deleting Apple DDM asset")
	}

	actTeamID, actTeamName, err := svc.assetActivityTeam(ctx, asset.TeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "resolving team for asset activity")
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedDeclarationAsset{
		AssetName: asset.Name,
		TeamID:    actTeamID,
		TeamName:  actTeamName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for deleted declaration asset")
	}

	return nil
}

func (svc *Service) BatchSetAppleDDMAssets(ctx context.Context, teamID *uint, teamName string, assets []fleet.MDMAppleDDMAssetBatchPayload, dryRun bool) error {
	var tmName *string
	if teamName != "" {
		tmName = &teamName
	}
	if teamID != nil && tmName != nil {
		svc.authz.SkipAuthorization(ctx)
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("team_name", "cannot specify both team_id and team_name"))
	}

	var resolvedTeamName string
	if teamID != nil || tmName != nil {
		tm, err := svc.teamByIDOrName(ctx, teamID, tmName)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "resolving team for assets batch")
		}
		if tm == nil {
			return ctxerr.Wrap(ctx, common_mysql.NotFound("Team"))
		}
		teamID = &tm.ID
		resolvedTeamName = tm.Name
	}

	if err := svc.authz.Authorize(ctx, &fleet.DDMAssetAuthz{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	// Secrets may not be available during a dry run (e.g. GitOps), so skip
	// validating assets that reference them, mirroring the profiles batch path.
	if dryRun {
		withoutSecrets := make([]fleet.MDMAppleDDMAssetBatchPayload, 0, len(assets))
		for _, a := range assets {
			if len(fleet.ContainsPrefixVars(string(a.Contents), fleet.ServerSecretPrefix)) == 0 {
				withoutSecrets = append(withoutSecrets, a)
			}
		}
		assets = withoutSecrets
	}

	toSet := make([]*fleet.MDMAppleDDMAssetToSet, 0, len(assets))
	seenNames := make(map[string]struct{}, len(assets))
	seenIdentifiers := make(map[string]struct{}, len(assets))
	for _, a := range assets {
		identifier, assetType, err := svc.validateAppleDDMAsset(ctx, a.Contents)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "validating asset %q", a.Name)
		}
		if _, ok := seenNames[a.Name]; ok {
			return &fleet.BadRequestError{Message: fmt.Sprintf("Couldn't apply. The asset name %q is used more than once.", a.Name)}
		}
		if _, ok := seenIdentifiers[identifier]; ok {
			return &fleet.BadRequestError{Message: fmt.Sprintf("Couldn't apply. The asset identifier %q is used more than once.", identifier)}
		}
		seenNames[a.Name] = struct{}{}
		seenIdentifiers[identifier] = struct{}{}
		toSet = append(toSet, &fleet.MDMAppleDDMAssetToSet{
			Name:       a.Name,
			Identifier: identifier,
			Type:       assetType,
			Data:       a.Contents,
		})
	}

	if dryRun {
		return nil
	}

	changes, err := svc.ds.BatchSetAppleDDMAssets(ctx, teamID, toSet)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "batch setting apple ddm assets")
	}

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID != nil && *teamID > 0 {
		actTeamID = teamID
		actTeamName = &resolvedTeamName
	}
	for _, name := range changes.Created {
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeCreatedDeclarationAsset{
			AssetName: name,
			TeamID:    actTeamID,
			TeamName:  actTeamName,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "logging activity for created declaration asset")
		}
	}
	for _, name := range changes.Edited {
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeEditedDeclarationAsset{
			AssetName: name,
			TeamID:    actTeamID,
			TeamName:  actTeamName,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "logging activity for edited declaration asset")
		}
	}
	for _, name := range changes.Deleted {
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedDeclarationAsset{
			AssetName: name,
			TeamID:    actTeamID,
			TeamName:  actTeamName,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "logging activity for deleted declaration asset")
		}
	}

	return nil
}

func (svc *Service) ReleaseABDevices(ctx context.Context, hostIDs []uint) ([]*fleet.ABReleaseDeviceResponse, error) {
	user := authz.UserFromContext(ctx)
	if user == nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewAuthRequiredError("user not found in context")
	}

	if !user.IsAnyAdmin() {
		return nil, authz.ForbiddenWithInternal("release AB devices requires an admin role", user, nil, fleet.ActionWrite)
	}

	if len(hostIDs) > 32_000 {
		svc.authz.SkipAuthorization(ctx)
		// Arbitrary limit, Apple does not document what a fair limit is.
		// Mainly to avoid querying more than 65k if that should ever happen in one MySQL statement, and break with too many statements.
		return nil, &fleet.BadRequestError{Message: "Too many host IDs provided. Maximum is 32,000."}
	}

	// First look up all hosts teamID's and serials.
	liteHosts, err := svc.ds.ListHostsLiteByIDs(ctx, hostIDs)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "listing hosts by ids")
	}

	// This is only really used for logging the display name in the activity
	hostIDToLiteHost := make(map[uint]*fleet.Host, len(liteHosts))

	// hostID -> response map
	response := make(map[uint]*fleet.ABReleaseDeviceResponse, len(hostIDs))

	setSuccessResponse := func(hostID uint) {
		if response[hostID] != nil {
			return // no-op, to avoid overwriting previous status, shouldn't really happen though.
		}
		response[hostID] = &fleet.ABReleaseDeviceResponse{
			HostID: hostID,
			Status: string(fleet.ABReleaseDeviceStatusSuccess),
		}
	}

	setErrorResponse := func(hostID uint, status fleet.ABReleaseDeviceStatus, errMsg string) {
		if response[hostID] != nil {
			return // no-op, to avoid overwriting previous status, shouldn't really happen though.
		}
		response[hostID] = &fleet.ABReleaseDeviceResponse{
			HostID: hostID,
			Status: string(status),
			Error:  errMsg,
		}
	}

	// We iterate over all hosts, to build a serial lookup map, and a deduped teamID list for authorization.
	unseenHostIDs := make(map[uint]struct{}, len(hostIDs))
	for _, id := range hostIDs {
		unseenHostIDs[id] = struct{}{}
	}

	serialToHostID := make(map[string]uint, len(liteHosts))
	teamIDs := make(map[uint]struct{}, len(liteHosts))
	for _, h := range liteHosts {
		hostIDToLiteHost[h.ID] = h
		delete(unseenHostIDs, h.ID)

		if h.TeamID != nil {
			teamIDs[*h.TeamID] = struct{}{}
		} else {
			teamIDs[0] = struct{}{}
		}

		if !fleet.IsApplePlatform(h.FleetPlatform()) {
			setErrorResponse(h.ID, fleet.ABReleaseDeviceStatusError, "This is not an eligible Apple host.")
			continue
		}

		if h.HardwareSerial == "" {
			setErrorResponse(h.ID, fleet.ABReleaseDeviceStatusError, "Host has no hardware serial.")
			continue
		}

		serialToHostID[h.HardwareSerial] = h.ID
	}

	for hostID := range unseenHostIDs {
		setErrorResponse(hostID, fleet.ABReleaseDeviceStatusError, "Host not found.")
	}

	if len(teamIDs) == 0 {
		// Only queried non-existent hosts, only global admin can see not founds.
		if err := svc.authz.Authorize(ctx, &fleet.ABReleaseDeviceAuthz{}, fleet.ActionWrite); err != nil {
			return nil, err
		}
	}

	// authz check on all teams from gathered hostID's
	for teamID := range teamIDs {
		tid := teamID
		if err := svc.authz.Authorize(ctx, &fleet.ABReleaseDeviceAuthz{TeamID: &tid}, fleet.ActionWrite); err != nil {
			return nil, err
		}
	}

	if len(serialToHostID) == 0 {
		sliceResponse := slices.Collect(maps.Values(response))
		slices.SortFunc(sliceResponse, func(a, b *fleet.ABReleaseDeviceResponse) int {
			return cmp.Compare(a.HostID, b.HostID)
		})
		return sliceResponse, nil
	}

	validHostIDs := slices.Collect(maps.Values(serialToHostID))
	depAssignments, err := svc.ds.GetHostDEPAssignmentsByHostIDs(ctx, validHostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host DEP assignments by host IDs")
	}

	// build another unseen map, to report which devices aren't in AB.
	unseenHostIDs = make(map[uint]struct{}, len(validHostIDs))
	for _, id := range validHostIDs {
		unseenHostIDs[id] = struct{}{}
	}

	// overwrite serialToHostID so we only have valid DEP assigned hosts in the serial list.
	serialToHostID = make(map[string]uint, len(depAssignments))
	// get a list of deduped token ID's
	dedupedTokenIDs := make(map[uint][]string, len(depAssignments))
	for _, assignment := range depAssignments {
		delete(unseenHostIDs, assignment.HostID)
		if assignment.HardwareSerial == "" {
			// Should not happen, but if query diverts then we safeguard.
			setErrorResponse(assignment.HostID, fleet.ABReleaseDeviceStatusError, "Host has no hardware serial.")
			continue
		}
		if assignment.ABMTokenID != nil {
			dedupedTokenIDs[*assignment.ABMTokenID] = append(dedupedTokenIDs[*assignment.ABMTokenID], assignment.HardwareSerial)
			serialToHostID[assignment.HardwareSerial] = assignment.HostID
		} else {
			// Should not happen, but if query diverts then we safeguard.
			setErrorResponse(assignment.HostID, fleet.ABReleaseDeviceStatusError, "Host has no associated ABM token.")
		}
	}

	for hostID := range unseenHostIDs {
		setErrorResponse(hostID, fleet.ABReleaseDeviceStatusError, "This host was not found in Apple Business.")
	}

	// We list all here and filter by deduped list, the returned list is so small anyways.
	tokens, err := svc.ds.ListABMTokens(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing ABM tokens")
	}

	depClient := apple_mdm.NewDEPClient(svc.depStorage, svc.ds, svc.logger)

	// Iterate over deduped token ID's and call the disown devices, for all serials associated with that token.
	for tokenID, serials := range dedupedTokenIDs {
		var token *fleet.ABMToken
		for _, t := range tokens {
			if t.ID == tokenID {
				token = t
				break
			}
		}
		if token == nil {
			for _, serial := range serials {
				setErrorResponse(serialToHostID[serial], fleet.ABReleaseDeviceStatusError, "ABM token not found.")
			}
			continue
		}

		svc.logger.DebugContext(ctx, "Releasing AB devices", "token_id", tokenID, "organization_name", token.OrganizationName, "serials", serials)
		disownResp, err := depClient.DisownDevices(ctx, token.OrganizationName, serials...)
		if err != nil {
			if depAuthErr, ok := errors.AsType[*client.AuthError](err); ok {
				svc.logger.ErrorContext(ctx, "Release AB devices failed with DEP auth error", "token_id", tokenID, "organization_name", token.OrganizationName, "error", depAuthErr)

				for _, serial := range serials {
					setErrorResponse(serialToHostID[serial], fleet.ABReleaseDeviceStatusError, fmt.Sprintf("Couldn't release host from Apple Business. Apple rejected this request. Confirm that “Allow this MDM server to release devices” is enabled in Apple Business. Learn More: %s", "https://fleetdm.com/learn-more-about/release-devices"))
				}
				continue
			}

			// Other generic HTTP/network/JSON errors.
			svc.logger.ErrorContext(ctx, "Failed to release AB devices", "token_id", tokenID, "organization_name", token.OrganizationName, "error", err)
			for _, serial := range serials {
				setErrorResponse(serialToHostID[serial], fleet.ABReleaseDeviceStatusError, "Couldn't release host from Apple Business.")
			}
			continue
		}

		releasedSerials := make([]string, 0, len(disownResp.Devices))
		for _, serial := range serials {
			hostID := serialToHostID[serial]

			status, ok := disownResp.Devices[serial]
			if !ok {
				svc.logger.ErrorContext(ctx, "No status returned for serial from DEP disown devices", "token_id", tokenID, "organization_name", token.OrganizationName, "serial", serial)
				setErrorResponse(hostID, fleet.ABReleaseDeviceStatusError, "Couldn't release host from Apple Business.")
			}

			if strings.EqualFold(string(status), string(fleet.ABReleaseDeviceStatusSuccess)) {
				setSuccessResponse(hostID)
				if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeReleasedDeviceFromAB{
					HostID:          hostID,
					HostSerial:      serial,
					HostDisplayName: hostIDToLiteHost[hostID].DisplayName(),
				}); err != nil {
					svc.logger.ErrorContext(ctx, "Failed to log activity for released device from AB", "host_id", hostID, "serial", serial, "error", err)
				}
				releasedSerials = append(releasedSerials, serial)
			} else {
				svc.logger.ErrorContext(ctx, "Got non success status from DEP disown devices", "token_id", tokenID, "organization_name", token.OrganizationName, "serial", serial, "status", status)
				setErrorResponse(hostID, fleet.ABReleaseDeviceStatusError, fmt.Sprintf("Error releasing device: %s", status))
			}

		}

		if err := svc.ds.DeleteHostDEPAssignments(ctx, token.ID, releasedSerials); err != nil {
			// We only log the error, but continue to try the remaining tokens.
			svc.logger.ErrorContext(ctx, "Failed to delete host DEP assignments after releasing devices", "token_id", tokenID, "organization_name", token.OrganizationName, "serials", releasedSerials, "error", err)
		}
	}

	sliceResponse := slices.Collect(maps.Values(response))
	slices.SortFunc(sliceResponse, func(a, b *fleet.ABReleaseDeviceResponse) int {
		return cmp.Compare(a.HostID, b.HostID)
	})

	return sliceResponse, nil
}
