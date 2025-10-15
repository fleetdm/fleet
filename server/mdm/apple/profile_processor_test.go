package apple_mdm

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mock"
	scep_mock "github.com/fleetdm/fleet/v4/server/mock/scep"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreprocessProfileContents(t *testing.T) {
	ctx := context.Background()
	logger := kitlog.NewNopLogger()
	appCfg := &fleet.AppConfig{}
	appCfg.ServerSettings.ServerURL = "https://test.example.com"
	appCfg.MDM.EnabledAndConfigured = true
	ds := new(mock.Store)

	// No-op
	svc := scep.NewSCEPConfigService(logger, nil)
	digiCertService := digicert.NewService(digicert.WithLogger(logger))
	err := preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, logger, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	hostUUID := "host-1"
	cmdUUID := "cmd-1"
	var targets map[string]*CmdTarget
	populateTargets := func() {
		targets = map[string]*CmdTarget{
			"p1": {CmdUUID: cmdUUID, ProfileIdentifier: "com.add.profile", EnrollmentIDs: []string{hostUUID}},
		}
	}
	hostProfilesToInstallMap := make(map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload, 1)
	hostProfilesToInstallMap[HostProfileUUID{HostUUID: hostUUID, ProfileUUID: "p1"}] = &fleet.MDMAppleBulkUpsertHostProfilePayload{
		ProfileUUID:       "p1",
		ProfileIdentifier: "com.add.profile",
		HostUUID:          hostUUID,
		OperationType:     fleet.MDMOperationTypeInstall,
		Status:            &fleet.MDMDeliveryPending,
		CommandUUID:       cmdUUID,
		Scope:             fleet.PayloadScopeSystem,
	}
	userEnrollmentsToHostUUIDsMap := make(map[string]string)
	populateTargets()
	profileContents := map[string]mobileconfig.Mobileconfig{
		"p1": []byte("$FLEET_VAR_" + fleet.FleetVarNDESSCEPProxyURL),
	}

	var updatedPayload *fleet.MDMAppleBulkUpsertHostProfilePayload
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		require.Len(t, payload, 1)
		updatedPayload = payload[0]
		for _, p := range payload {
			require.NotNil(t, p.Status)
			assert.Equal(t, fleet.MDMDeliveryFailed, *p.Status)
			assert.Equal(t, cmdUUID, p.CommandUUID)
			assert.Equal(t, hostUUID, p.HostUUID)
			assert.Equal(t, fleet.MDMOperationTypeInstall, p.OperationType)
			assert.Equal(t, fleet.PayloadScopeSystem, p.Scope)
		}
		return nil
	}
	// Can't use NDES SCEP proxy with free tier
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierFree})
	err = preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, nil)
	require.NoError(t, err)
	require.NotNil(t, updatedPayload)
	assert.Contains(t, updatedPayload.Detail, "Premium license")
	assert.Empty(t, targets)

	// Can't use NDES SCEP proxy without it being configured
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	updatedPayload = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, &fleet.GroupedCertificateAuthorities{})
	require.NoError(t, err)
	require.NotNil(t, updatedPayload)
	assert.Contains(t, updatedPayload.Detail, "not configured")
	assert.NotNil(t, updatedPayload.VariablesUpdatedAt)
	assert.Empty(t, targets)

	// Unknown variable
	profileContents = map[string]mobileconfig.Mobileconfig{
		"p1": []byte("$FLEET_VAR_BOZO"),
	}
	updatedPayload = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, nil)
	require.NoError(t, err)
	require.NotNil(t, updatedPayload)
	assert.Contains(t, updatedPayload.Detail, "FLEET_VAR_BOZO")
	assert.Empty(t, targets)

	ndesPassword := "test-password"
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context,
		assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetNDESPassword: {Value: []byte(ndesPassword)},
		}, nil
	}

	ds.BulkUpsertMDMAppleHostProfilesFunc = nil
	var updatedProfile *fleet.HostMDMAppleProfile
	ds.UpdateOrDeleteHostMDMAppleProfileFunc = func(ctx context.Context, profile *fleet.HostMDMAppleProfile) error {
		updatedProfile = profile
		require.NotNil(t, updatedProfile.Status)
		assert.Equal(t, fleet.MDMDeliveryFailed, *updatedProfile.Status)
		assert.Equal(t, cmdUUID, updatedProfile.CommandUUID)
		assert.Equal(t, hostUUID, updatedProfile.HostUUID)
		assert.Equal(t, fleet.MDMOperationTypeInstall, updatedProfile.OperationType)
		return nil
	}
	ds.BulkUpsertMDMManagedCertificatesFunc = func(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
		assert.Empty(t, payload)
		return nil
	}

	adminUrl := "https://example.com"
	username := "admin"
	password := "test-password"
	groupedCAs := &fleet.GroupedCertificateAuthorities{
		NDESSCEP: &fleet.NDESSCEPProxyCA{
			URL:      "https://test-example.com",
			AdminURL: adminUrl,
			Username: username,
			Password: password,
		},
	}

	// Could not get NDES SCEP challenge
	profileContents = map[string]mobileconfig.Mobileconfig{
		"p1": []byte("$FLEET_VAR_" + fleet.FleetVarNDESSCEPChallenge),
	}
	scepConfig := &scep_mock.SCEPConfigService{}
	scepConfig.GetNDESSCEPChallengeFunc = func(ctx context.Context, proxy fleet.NDESSCEPProxyCA) (string, error) {
		assert.Equal(t, ndesPassword, proxy.Password)
		return "", scep.NewNDESInvalidError("NDES error")
	}
	updatedProfile = nil
	populateTargets()
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		assert.Empty(t, payload) // no profiles to update since FLEET VAR could not be populated
		return nil
	}
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	require.NotNil(t, updatedProfile)
	assert.Contains(t, updatedProfile.Detail, "FLEET_VAR_"+fleet.FleetVarNDESSCEPChallenge)
	assert.Contains(t, updatedProfile.Detail, "update credentials")
	assert.NotNil(t, updatedProfile.VariablesUpdatedAt)
	assert.Empty(t, targets)

	// Password cache full
	scepConfig.GetNDESSCEPChallengeFunc = func(ctx context.Context, proxy fleet.NDESSCEPProxyCA) (string, error) {
		assert.Equal(t, ndesPassword, proxy.Password)
		return "", scep.NewNDESPasswordCacheFullError("NDES error")
	}
	updatedProfile = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	require.NotNil(t, updatedProfile)
	assert.Contains(t, updatedProfile.Detail, "FLEET_VAR_"+fleet.FleetVarNDESSCEPChallenge)
	assert.Contains(t, updatedProfile.Detail, "cached passwords")
	assert.NotNil(t, updatedProfile.VariablesUpdatedAt)
	assert.Empty(t, targets)

	// Insufficient permissions
	scepConfig.GetNDESSCEPChallengeFunc = func(ctx context.Context, proxy fleet.NDESSCEPProxyCA) (string, error) {
		assert.Equal(t, ndesPassword, proxy.Password)
		return "", scep.NewNDESInsufficientPermissionsError("NDES error")
	}
	updatedProfile = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	require.NotNil(t, updatedProfile)
	assert.Contains(t, updatedProfile.Detail, "FLEET_VAR_"+fleet.FleetVarNDESSCEPChallenge)
	assert.Contains(t, updatedProfile.Detail, "does not have sufficient permissions")
	assert.NotNil(t, updatedProfile.VariablesUpdatedAt)
	assert.Empty(t, targets)

	// Other NDES challenge error
	scepConfig.GetNDESSCEPChallengeFunc = func(ctx context.Context, proxy fleet.NDESSCEPProxyCA) (string, error) {
		assert.Equal(t, ndesPassword, proxy.Password)
		return "", errors.New("NDES error")
	}
	updatedProfile = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	require.NotNil(t, updatedProfile)
	assert.Contains(t, updatedProfile.Detail, "FLEET_VAR_"+fleet.FleetVarNDESSCEPChallenge)
	assert.NotContains(t, updatedProfile.Detail, "cached passwords")
	assert.NotContains(t, updatedProfile.Detail, "update credentials")
	assert.NotNil(t, updatedProfile.VariablesUpdatedAt)
	assert.Empty(t, targets)

	// NDES challenge
	challenge := "ndes-challenge"
	scepConfig.GetNDESSCEPChallengeFunc = func(ctx context.Context, proxy fleet.NDESSCEPProxyCA) (string, error) {
		assert.Equal(t, ndesPassword, proxy.Password)
		return challenge, nil
	}
	updatedProfile = nil
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		for _, p := range payload {
			assert.NotEqual(t, cmdUUID, p.CommandUUID)
		}
		return nil
	}
	populateTargets()
	ds.BulkUpsertMDMManagedCertificatesFunc = func(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
		require.Len(t, payload, 1)
		assert.NotNil(t, payload[0].ChallengeRetrievedAt)
		return nil
	}
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	assert.Nil(t, updatedProfile)
	require.NotEmpty(t, targets)
	assert.Len(t, targets, 1)
	for profUUID, target := range targets {
		assert.NotEqual(t, profUUID, "p1") // new temporary UUID generated for specific host
		assert.NotEqual(t, cmdUUID, target.CmdUUID)
		assert.Equal(t, []string{hostUUID}, target.EnrollmentIDs)
		assert.Equal(t, challenge, string(profileContents[profUUID]))
	}

	// NDES SCEP proxy URL
	profileContents = map[string]mobileconfig.Mobileconfig{
		"p1": []byte("$FLEET_VAR_" + fleet.FleetVarNDESSCEPProxyURL),
	}
	expectedURL := "https://test.example.com" + SCEPProxyPath + url.QueryEscape(fmt.Sprintf("%s,%s,NDES", hostUUID, "p1"))
	updatedProfile = nil
	populateTargets()
	ds.BulkUpsertMDMManagedCertificatesFunc = func(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
		assert.Empty(t, payload)
		return nil
	}
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	assert.Nil(t, updatedProfile)
	require.NotEmpty(t, targets)
	assert.Len(t, targets, 1)
	for profUUID, target := range targets {
		assert.NotEqual(t, profUUID, "p1") // new temporary UUID generated for specific host
		assert.NotEqual(t, cmdUUID, target.CmdUUID)
		assert.Equal(t, []string{hostUUID}, target.EnrollmentIDs)
		assert.Equal(t, expectedURL, string(profileContents[profUUID]))
	}

	// No IdP email found
	ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
		return nil, nil
	}
	profileContents = map[string]mobileconfig.Mobileconfig{
		"p1": []byte("$FLEET_VAR_" + fleet.FleetVarHostEndUserEmailIDP),
	}
	updatedProfile = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	require.NotNil(t, updatedProfile)
	assert.Contains(t, updatedProfile.Detail, "FLEET_VAR_"+fleet.FleetVarHostEndUserEmailIDP)
	assert.Contains(t, updatedProfile.Detail, "no IdP email")
	assert.Empty(t, targets)

	// IdP email found
	email := "user@example.com"
	ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
		return []string{email}, nil
	}
	updatedProfile = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	assert.Nil(t, updatedProfile)
	require.NotEmpty(t, targets)
	assert.Len(t, targets, 1)
	for profUUID, target := range targets {
		assert.NotEqual(t, profUUID, "p1") // new temporary UUID generated for specific host
		assert.NotEqual(t, cmdUUID, target.CmdUUID)
		assert.Equal(t, []string{hostUUID}, target.EnrollmentIDs)
		assert.Equal(t, email, string(profileContents[profUUID]))
	}

	// Hardware serial
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, _ fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		assert.Equal(t, []string{hostUUID}, uuids)
		return []*fleet.Host{
			{HardwareSerial: "serial1"},
		}, nil
	}
	profileContents = map[string]mobileconfig.Mobileconfig{
		"p1": []byte("$FLEET_VAR_" + fleet.FleetVarHostHardwareSerial),
	}
	updatedProfile = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	assert.Nil(t, updatedProfile)
	require.NotEmpty(t, targets)
	assert.Len(t, targets, 1)
	for profUUID, target := range targets {
		assert.NotEqual(t, profUUID, "p1") // new temporary UUID generated for specific host
		assert.NotEqual(t, cmdUUID, target.CmdUUID)
		assert.Equal(t, []string{hostUUID}, target.EnrollmentIDs)
		assert.Equal(t, "serial1", string(profileContents[profUUID]))
	}

	// Hardware serial fail
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, _ fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		assert.Equal(t, []string{hostUUID}, uuids)
		return nil, nil
	}
	updatedProfile = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	require.NotNil(t, updatedProfile)
	assert.Contains(t, updatedProfile.Detail, "Unexpected number of hosts (0) for UUID")
	assert.Empty(t, targets)

	// multiple profiles, multiple hosts
	populateTargets = func() {
		targets = map[string]*CmdTarget{
			"p1": {CmdUUID: cmdUUID, ProfileIdentifier: "com.add.profile", EnrollmentIDs: []string{hostUUID, "host-2"}},  // fails
			"p2": {CmdUUID: cmdUUID, ProfileIdentifier: "com.add.profile2", EnrollmentIDs: []string{hostUUID, "host-3"}}, // works
			"p3": {CmdUUID: cmdUUID, ProfileIdentifier: "com.add.profile3", EnrollmentIDs: []string{hostUUID, "host-4"}}, // no variables
		}
	}
	populateTargets()
	groupedCAs.NDESSCEP = nil
	profileContents = map[string]mobileconfig.Mobileconfig{
		"p1": []byte("$FLEET_VAR_" + fleet.FleetVarNDESSCEPProxyURL),
		"p2": []byte("$FLEET_VAR_" + fleet.FleetVarHostEndUserEmailIDP),
		"p3": []byte("no variables"),
	}
	addProfileToInstall := func(hostUUID, profileUUID, profileIdentifier string) {
		hostProfilesToInstallMap[HostProfileUUID{
			HostUUID:    hostUUID,
			ProfileUUID: profileUUID,
		}] = &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileUUID:       profileUUID,
			ProfileIdentifier: profileIdentifier,
			HostUUID:          hostUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryPending,
			CommandUUID:       cmdUUID,
			Scope:             fleet.PayloadScopeSystem,
		}
	}
	addProfileToInstall(hostUUID, "p1", "com.add.profile")
	addProfileToInstall("host-2", "p1", "com.add.profile")
	addProfileToInstall(hostUUID, "p2", "com.add.profile2")
	addProfileToInstall("host-3", "p2", "com.add.profile2")
	addProfileToInstall(hostUUID, "p3", "com.add.profile3")
	addProfileToInstall("host-4", "p3", "com.add.profile3")
	expectedHostsToFail := []string{hostUUID, "host-2", "host-3"}
	ds.UpdateOrDeleteHostMDMAppleProfileFunc = func(ctx context.Context, profile *fleet.HostMDMAppleProfile) error {
		updatedProfile = profile
		require.NotNil(t, updatedProfile.Status)
		assert.Equal(t, fleet.MDMDeliveryFailed, *updatedProfile.Status)
		assert.NotEqual(t, cmdUUID, updatedProfile.CommandUUID)
		assert.Contains(t, expectedHostsToFail, updatedProfile.HostUUID)
		assert.Equal(t, fleet.MDMOperationTypeInstall, updatedProfile.OperationType)
		return nil
	}
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		for _, p := range payload {
			require.NotNil(t, p.Status)
			if fleet.MDMDeliveryFailed == *p.Status {
				assert.Equal(t, cmdUUID, p.CommandUUID)
			} else {
				assert.NotEqual(t, cmdUUID, p.CommandUUID)
			}
			assert.Equal(t, fleet.MDMOperationTypeInstall, p.OperationType)
		}
		return nil
	}
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	require.NotEmpty(t, targets)
	assert.Len(t, targets, 3)
	assert.Nil(t, targets["p1"])    // error
	assert.Nil(t, targets["p2"])    // renamed
	assert.NotNil(t, targets["p3"]) // normal, no variables
	for profUUID, target := range targets {
		assert.Contains(t, [][]string{{hostUUID}, {"host-3"}, {hostUUID, "host-4"}}, target.EnrollmentIDs)
		if profUUID == "p3" {
			assert.Equal(t, cmdUUID, target.CmdUUID)
		} else {
			assert.NotEqual(t, cmdUUID, target.CmdUUID)
		}
		assert.Contains(t, []string{email, "no variables"}, string(profileContents[profUUID]))
	}
}

func TestPreprocessProfileContentsEndUserIDP(t *testing.T) {
	ctx := context.Background()
	logger := kitlog.NewNopLogger()
	appCfg := &fleet.AppConfig{}
	appCfg.ServerSettings.ServerURL = "https://test.example.com"
	appCfg.MDM.EnabledAndConfigured = true
	ds := new(mock.Store)

	svc := scep.NewSCEPConfigService(logger, nil)
	digiCertService := digicert.NewService(digicert.WithLogger(logger))

	hostUUID := "host-1"
	cmdUUID := "cmd-1"
	var targets map[string]*CmdTarget
	// this is a func to re-create it each time because calling the preprocess function modifies this
	populateTargets := func() {
		targets = map[string]*CmdTarget{
			"p1": {CmdUUID: cmdUUID, ProfileIdentifier: "com.add.profile", EnrollmentIDs: []string{hostUUID}},
		}
	}
	hostProfilesToInstallMap := map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{HostUUID: hostUUID, ProfileUUID: "p1"}: {
			ProfileUUID:       "p1",
			ProfileIdentifier: "com.add.profile",
			HostUUID:          hostUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryPending,
			CommandUUID:       cmdUUID,
		},
	}

	userEnrollmentsToHostUUIDsMap := make(map[string]string)

	var updatedPayload *fleet.MDMAppleBulkUpsertHostProfilePayload
	var expectedStatus fleet.MDMDeliveryStatus
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		require.Len(t, payload, 1)
		updatedPayload = payload[0]
		require.NotNil(t, updatedPayload.Status)
		assert.Equal(t, expectedStatus, *updatedPayload.Status)
		// cmdUUID was replaced by a new unique command on success
		assert.NotEqual(t, cmdUUID, updatedPayload.CommandUUID)
		assert.Equal(t, hostUUID, updatedPayload.HostUUID)
		assert.Equal(t, fleet.MDMOperationTypeInstall, updatedPayload.OperationType)
		return nil
	}
	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, idents []string) ([]uint, error) {
		require.Len(t, idents, 1)
		require.Equal(t, hostUUID, idents[0])
		return []uint{1}, nil
	}
	var updatedProfile *fleet.HostMDMAppleProfile
	ds.UpdateOrDeleteHostMDMAppleProfileFunc = func(ctx context.Context, profile *fleet.HostMDMAppleProfile) error {
		updatedProfile = profile
		require.NotNil(t, profile.Status)
		assert.Equal(t, expectedStatus, *profile.Status)
		return nil
	}
	ds.GetAllCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) ([]*fleet.CertificateAuthority, error) {
		return []*fleet.CertificateAuthority{}, nil
	}

	cases := []struct {
		desc           string
		profileContent string
		expectedStatus fleet.MDMDeliveryStatus
		setup          func()
		assert         func(output string)
	}{
		{
			desc:           "username only scim",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPUsername),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "user1@example.com", output)
			},
		},
		{
			desc:           "username local part only scim",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPUsernameLocalPart),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "user1", output)
			},
		},
		{
			desc:           "groups only scim",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPGroups),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
						IdpGroups:   []string{"a", "b"},
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "a,b", output)
			},
		},
		{
			desc:           "multiple times username only scim",
			profileContent: strings.Repeat("${FLEET_VAR_"+string(fleet.FleetVarHostEndUserIDPUsername)+"}", 3),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "user1@example.comuser1@example.comuser1@example.com", output)
			},
		},
		{
			desc:           "all 3 vars with scim",
			profileContent: "${FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPUsername) + "}${FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPUsernameLocalPart) + "}${FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPGroups) + "}",
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
						IdpGroups:   []string{"a", "b"},
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "user1@example.comuser1a,b", output)
			},
		},
		{
			desc:           "username no scim, with idp",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPUsername),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "idp@example.com",
						OtherEmails: []fleet.HostDeviceMapping{
							{
								Email: "other@example.com", Source: fleet.DeviceMappingGoogleChromeProfiles,
							},
						},
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "idp@example.com", output)
			},
		},
		{
			desc:           "username scim and idp",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPUsername),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
						OtherEmails: []fleet.HostDeviceMapping{
							{
								Email: "other@example.com", Source: fleet.DeviceMappingGoogleChromeProfiles,
							},
						},
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "user1@example.com", output)
			},
		},
		{
			desc:           "username, no idp user",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPUsername),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						OtherEmails: []fleet.HostDeviceMapping{
							{
								Email: "other@example.com", Source: fleet.DeviceMappingGoogleChromeProfiles,
							},
						},
					}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There is no IdP username for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_USERNAME.")
			},
		},
		{
			desc:           "username local part, no idp user",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPUsernameLocalPart),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						OtherEmails: []fleet.HostDeviceMapping{
							{
								Email: "other@example.com", Source: fleet.DeviceMappingGoogleChromeProfiles,
							},
						},
					}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There is no IdP username for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART.")
			},
		},
		{
			desc:           "groups, no idp user",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPGroups),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There is no IdP groups for this host. Fleet couldn’t populate $FLEET_VAR_HOST_END_USER_IDP_GROUPS.")
			},
		},
		{
			desc:           "department, no idp user",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPDepartment),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There is no IdP department for this host. Fleet couldn’t populate $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT.")
			},
		},
		{
			desc:           "groups with user groups, user has no groups",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPGroups),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There is no IdP groups for this host. Fleet couldn’t populate $FLEET_VAR_HOST_END_USER_IDP_GROUPS.")
			},
		},
		{
			desc:           "profile with department, user has department",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPDepartment),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
						Department:  "Engineering",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "Engineering", output)
			},
		},
		{
			desc:           "profile with department, user has no department",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPDepartment),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There is no IdP department for this host. Fleet couldn’t populate $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT.")
			},
		},
		{
			desc:           "profile with full name, user has full name",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPFullname),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
						IdpFullName: "First Last",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "First Last", output)
			},
		},
		{
			desc:           "profile with full name, user only has given name",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPFullname),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
						IdpFullName: "First",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "First", output)
			},
		},
		{
			desc:           "profile with full name, user only has family name",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPFullname),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
						IdpFullName: "Last",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail) // no error detail
				assert.Len(t, targets, 1)              // target is still present
				require.Equal(t, "Last", output)
			},
		},
		{
			desc:           "profile with full name, user has no full name value",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPFullname),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.GetEndUsersFunc = func(ctx context.Context, hostID uint) (*fleet.HostEndUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.HostEndUser{
						IdpUserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Contains(t, updatedProfile.Detail, fmt.Sprintf("There is no IdP full name for this host. Fleet couldn’t populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPFullname))
				assert.Len(t, targets, 0)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.setup()

			profileContents := map[string]mobileconfig.Mobileconfig{
				"p1": []byte(c.profileContent),
			}
			populateTargets()
			expectedStatus = c.expectedStatus
			updatedPayload = nil
			updatedProfile = nil

			err := preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, nil)
			require.NoError(t, err)
			var output string
			if expectedStatus == fleet.MDMDeliveryFailed {
				require.Nil(t, updatedPayload)
				require.NotNil(t, updatedProfile)
			} else {
				require.NotNil(t, updatedPayload)
				require.Nil(t, updatedProfile)
				output = string(profileContents[updatedPayload.CommandUUID])
			}

			c.assert(output)
		})
	}
}
