package apple_mdm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mock"
	digicert_mock "github.com/fleetdm/fleet/v4/server/mock/digicert"
	ejbca_mock "github.com/fleetdm/fleet/v4/server/mock/ejbca"
	scep_mock "github.com/fleetdm/fleet/v4/server/mock/scep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noEJBCAService and noDigiCertService are for the tests that don't use the CA, which fail if it is called.
var (
	noEJBCAService    fleet.EJBCAService    = &ejbca_mock.Service{}
	noDigiCertService fleet.DigiCertService = &digicert_mock.Service{}
)

func TestPreprocessProfileContents(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	appCfg := &fleet.AppConfig{}
	appCfg.ServerSettings.ServerURL = "https://test.example.com"
	appCfg.MDM.EnabledAndConfigured = true
	ds := new(mock.Store)

	// No-op
	svc := scep.NewSCEPConfigService(logger, nil)
	digiCertService := digicert.NewService(digicert.WithLogger(logger))
	err := preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, noEJBCAService, logger, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	hostUUID := "host-1"
	cmdUUID := "cmd-1"
	var targets map[string]*fleet.CmdTarget
	populateTargets := func() {
		targets = map[string]*fleet.CmdTarget{
			"p1": {CmdUUID: cmdUUID, ProfileIdentifier: "com.add.profile", EnrollmentIDs: []string{hostUUID}},
		}
	}
	hostProfilesToInstallMap := make(map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload, 1)
	hostProfilesToInstallMap[fleet.HostProfileUUID{HostUUID: hostUUID, ProfileUUID: "p1"}] = &fleet.MDMAppleBulkUpsertHostProfilePayload{
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
	err = preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, nil)
	require.NoError(t, err)
	require.NotNil(t, updatedPayload)
	assert.Contains(t, updatedPayload.Detail, "Premium license")
	assert.Empty(t, targets)

	// Can't use NDES SCEP proxy without it being configured
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	updatedPayload = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, &fleet.GroupedCertificateAuthorities{})
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
	err = preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, nil)
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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	require.NotNil(t, updatedProfile)
	assert.Contains(t, updatedProfile.Detail, "FLEET_VAR_"+fleet.FleetVarNDESSCEPChallenge)
	assert.Contains(t, updatedProfile.Detail, "does not have sufficient permissions")
	assert.NotNil(t, updatedProfile.VariablesUpdatedAt)
	assert.Empty(t, targets)

	// NDES briefly unreachable. Nobody has to act and nothing was delivered, so the profile must be left queued.
	scepConfig.GetNDESSCEPChallengeFunc = func(ctx context.Context, proxy fleet.NDESSCEPProxyCA) (string, error) {
		assert.Equal(t, ndesPassword, proxy.Password)
		return "", errors.New("sending request: dial tcp 10.0.0.10:80: connect: connection refused")
	}
	updatedProfile = nil
	populateTargets()
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	assert.Nil(t, updatedProfile, "a self-clearing NDES failure must not be reported against the host")
	assert.Empty(t, targets, "the profile must not be sent without a challenge")

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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	require.NoError(t, err)
	require.NotNil(t, updatedProfile)
	assert.Contains(t, updatedProfile.Detail, "Unexpected number of hosts (0) for UUID")
	assert.Empty(t, targets)

	// multiple profiles, multiple hosts
	populateTargets = func() {
		targets = map[string]*fleet.CmdTarget{
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
		hostProfilesToInstallMap[fleet.HostProfileUUID{
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
	err = preprocessProfileContents(ctx, appCfg, ds, scepConfig, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
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

// TestPreprocessProfileContentsDigiCertUPNMultiHost is a regression test for
// https://github.com/fleetdm/fleet/issues/39324. When the same DigiCert CA is
// used for multiple hosts in a single batch, the CertificateUserPrincipalNames
// slice was shared via a shallow copy. In-place variable substitution for Host 1
// corrupted the cached CA entry, so Host 2 and later hosts received Host 1's
// substituted UPN instead of their own.
func TestPreprocessProfileContentsDigiCertUPNIsUniqueForMultipleHosts(t *testing.T) {
	ctx := context.Background()
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	logger := slog.New(slog.DiscardHandler)
	appCfg := &fleet.AppConfig{}
	appCfg.ServerSettings.ServerURL = "https://test.example.com"
	appCfg.MDM.EnabledAndConfigured = true
	ds := new(mock.Store)

	svc := scep.NewSCEPConfigService(logger, nil)

	const caName = "myCA"
	const host1UUID = "host-uuid-1"
	const host2UUID = "host-uuid-2"
	const host1Serial = "SERIAL-AAA"
	const host2Serial = "SERIAL-BBB"
	const cmdUUID = "cmd-uuid-1"

	// Track which UPNs and CNs were sent to GetCertificate per host.
	upnByHostUUID := make(map[string]string)
	cnByHostUUID := make(map[string]string)

	mockDigiCert := &digicert_mock.Service{}
	mockDigiCert.GetCertificateFunc = func(ctx context.Context, config fleet.DigiCertCA) (*fleet.DigiCertCertificate, error) {
		// The UPN in config should have been substituted with each host's own
		// hardware serial. Record it so we can assert correctness later.
		require.Len(t, config.CertificateUserPrincipalNames, 1)
		upn := config.CertificateUserPrincipalNames[0]
		// Determine which host this call is for by checking which serial is in the UPN.
		switch {
		case strings.Contains(upn, host1Serial):
			upnByHostUUID[host1UUID] = upn
			cnByHostUUID[host1UUID] = config.CertificateCommonName
		case strings.Contains(upn, host2Serial):
			upnByHostUUID[host2UUID] = upn
			cnByHostUUID[host2UUID] = config.CertificateCommonName
		default:
			t.Errorf("GetCertificate called with unexpected UPN %q", upn)
		}
		now := time.Now()
		return &fleet.DigiCertCertificate{
			PfxData:        []byte("fake-pfx"),
			Password:       "fake-password",
			NotValidBefore: now,
			NotValidAfter:  now.Add(365 * 24 * time.Hour),
			SerialNumber:   upn, // reuse upn as serial for easy tracing
		}, nil
	}

	// Both hosts share the same profile UUID but are separate enrollment IDs.
	targets := map[string]*fleet.CmdTarget{
		"p1": {
			CmdUUID:           cmdUUID,
			ProfileIdentifier: "com.apple.security.pkcs12",
			EnrollmentIDs:     []string{host1UUID, host2UUID},
		},
	}

	pending := fleet.MDMDeliveryPending
	hostProfilesToInstallMap := map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{HostUUID: host1UUID, ProfileUUID: "p1"}: {
			ProfileUUID:       "p1",
			ProfileIdentifier: "com.apple.security.pkcs12",
			HostUUID:          host1UUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &pending,
			CommandUUID:       cmdUUID,
			Scope:             fleet.PayloadScopeSystem,
		},
		{HostUUID: host2UUID, ProfileUUID: "p1"}: {
			ProfileUUID:       "p1",
			ProfileIdentifier: "com.apple.security.pkcs12",
			HostUUID:          host2UUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &pending,
			CommandUUID:       cmdUUID,
			Scope:             fleet.PayloadScopeSystem,
		},
	}

	// Profile contains both DigiCert fleet variables.
	profileContents := map[string]mobileconfig.Mobileconfig{
		"p1": []byte("$FLEET_VAR_" + string(fleet.FleetVarDigiCertPasswordPrefix) + caName + " $FLEET_VAR_" + string(fleet.FleetVarDigiCertDataPrefix) + caName),
	}

	// DigiCert CA whose UPN contains the hardware serial variable and CN the IdP username.
	groupedCAs := &fleet.GroupedCertificateAuthorities{
		DigiCert: []fleet.DigiCertCA{
			{
				Name:                          caName,
				URL:                           "https://digicert.example.com",
				APIToken:                      "api_token",
				ProfileID:                     "profile_id",
				CertificateCommonName:         "cn-" + fleet.FleetVarHostEndUserIDPUsername.WithPrefix(),
				CertificateUserPrincipalNames: []string{"$FLEET_VAR_" + string(fleet.FleetVarHostHardwareSerial) + "@example.com"},
				CertificateSeatID:             "seat_id",
			},
		},
	}

	// Mock datastore: return each host's own hardware serial when queried.
	hostsByUUID := map[string]*fleet.Host{
		host1UUID: {ID: 1, UUID: host1UUID, HardwareSerial: host1Serial, Platform: "darwin"},
		host2UUID: {ID: 2, UUID: host2UUID, HardwareSerial: host2Serial, Platform: "darwin"},
	}
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, _ fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		var hosts []*fleet.Host
		for _, uuid := range uuids {
			if h, ok := hostsByUUID[uuid]; ok {
				hosts = append(hosts, h)
			}
		}
		return hosts, nil
	}
	ds.HostIDsByIdentifierFunc = func(ctx context.Context, _ fleet.TeamFilter, identifiers []string) ([]uint, error) {
		return []uint{hostsByUUID[identifiers[0]].ID}, nil
	}
	ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		return &fleet.ScimUser{UserName: fmt.Sprintf("user%d@example.com", hostID)}, nil
	}
	ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
	}
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		return nil
	}
	ds.BulkUpsertMDMManagedCertificatesFunc = func(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
		return nil
	}

	err := preprocessProfileContents(ctx, appCfg, ds, svc, mockDigiCert, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, make(map[string]string), groupedCAs)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{host1UUID: "cn-user1@example.com", host2UUID: "cn-user2@example.com"}, cnByHostUUID)

	// Both hosts must have received GetCertificate calls with their own serial.
	require.True(t, mockDigiCert.GetCertificateFuncInvoked, "GetCertificate was never called")
	require.Contains(t, upnByHostUUID, host1UUID, "GetCertificate was not called for host 1")
	require.Contains(t, upnByHostUUID, host2UUID, "GetCertificate was not called for host 2")
	assert.Equal(t, host1Serial+"@example.com", upnByHostUUID[host1UUID], "host 1 UPN should contain its own serial")
	assert.Equal(t, host2Serial+"@example.com", upnByHostUUID[host2UUID], "host 2 UPN should contain its own serial")
}

func TestReplaceFleetVarInDigiCertItem(t *testing.T) {
	ds := new(mock.Store)
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		return []*fleet.Host{{ID: 1, UUID: "host", HardwareSerial: "serial", Platform: "darwin"}}, nil
	}
	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, idents []string) ([]uint, error) {
		return []uint{1}, nil
	}
	ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
		return []string{"o'brien@example.com"}, nil
	}
	ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		return &fleet.ScimUser{
			UserName:   "user@example.com",
			GivenName:  new("R&D"),
			FamilyName: new("User"),
			Department: new("Engineering & IT"),
			Groups:     []fleet.ScimUserGroup{{DisplayName: "Engineering"}, {DisplayName: "IT"}},
		}, nil
	}
	ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
		return []*fleet.HostDeviceMapping{{Email: "o'brien@example.com"}}, nil
	}

	tests := []struct {
		item     string
		expected string
	}{
		{fleet.FleetVarHostEndUserEmailIDP.WithBraces(), "o'brien@example.com"},
		{fleet.FleetVarHostHardwareSerial.WithBraces(), "serial"},
		{fleet.FleetVarHostUUID.WithBraces(), "host"},
		{fleet.FleetVarHostPlatform.WithBraces(), "macos"},
		{fleet.FleetVarHostEndUserIDPUsername.WithBraces(), "user@example.com"},
		{fleet.FleetVarHostEndUserIDPUsernameLocalPart.WithBraces(), "user"},
		{fleet.FleetVarHostEndUserIDPGroups.WithBraces(), "Engineering,IT"},
		{fleet.FleetVarHostEndUserIDPDepartment.WithBraces(), "Engineering & IT"},
		{fleet.FleetVarHostEndUserIDPFullname.WithBraces(), "R&D User"},
		// USERNAME is a prefix of USERNAME_LOCAL_PART
		{fleet.FleetVarHostEndUserIDPUsername.WithPrefix() + " " + fleet.FleetVarHostEndUserIDPUsernameLocalPart.WithPrefix(), "user@example.com user"},
	}
	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			item := "prefix-" + tt.item + "-suffix"
			ok, err := replaceFleetVarInCAItem(t.Context(), ds, &fleet.CmdTarget{CmdUUID: "command"}, fleet.Host{UUID: "host"}, make(map[string]string), &item, make(map[string]uint), nil, func(int) error { return nil }, digiCertVarHint)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, "prefix-"+tt.expected+"-suffix", item)
		})
	}

	// Failures mention the CA, since the variable lives in its fields and not in the profile.
	ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		return &fleet.ScimUser{UserName: "user@example.com"}, nil
	}
	ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
		return nil, nil
	}
	var failureDetail string
	ds.UpdateOrDeleteHostMDMAppleProfileFunc = func(ctx context.Context, profile *fleet.HostMDMAppleProfile) error {
		failureDetail = profile.Detail
		return nil
	}
	failures := []struct {
		fleetVar fleet.FleetVarName
		detail   string
	}{
		{fleet.FleetVarHostEndUserIDPDepartment, "Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT."},
		{fleet.FleetVarHostEndUserEmailIDP, "Fleet couldn't populate $FLEET_VAR_HOST_END_USER_EMAIL_IDP."},
	}
	for _, ca := range []struct{ name, hint string }{{"DigiCert", digiCertVarHint}, {"EJBCA", ejbcaVarHint}} {
		for _, tt := range failures {
			t.Run(ca.name+" failure "+string(tt.fleetVar), func(t *testing.T) {
				failureDetail = ""
				item := tt.fleetVar.WithBraces()
				ok, err := replaceFleetVarInCAItem(t.Context(), ds, &fleet.CmdTarget{CmdUUID: "command"}, fleet.Host{UUID: "host"}, make(map[string]string), &item, make(map[string]uint), nil, func(int) error { return nil }, ca.hint)
				require.NoError(t, err)
				require.False(t, ok)
				assert.Contains(t, failureDetail, tt.detail)
				assert.Contains(t, failureDetail, ca.name+" certificate authority")
			})
		}
	}
}

func TestPreprocessProfileContentsEndUserIDP(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	appCfg := &fleet.AppConfig{}
	appCfg.ServerSettings.ServerURL = "https://test.example.com"
	appCfg.MDM.EnabledAndConfigured = true
	ds := new(mock.Store)

	svc := scep.NewSCEPConfigService(logger, nil)
	digiCertService := digicert.NewService(digicert.WithLogger(logger))

	hostUUID := "host-1"
	cmdUUID := "cmd-1"
	var targets map[string]*fleet.CmdTarget
	// this is a func to re-create it each time because calling the preprocess function modifies this
	populateTargets := func() {
		targets = map[string]*fleet.CmdTarget{
			"p1": {CmdUUID: cmdUUID, ProfileIdentifier: "com.add.profile", EnrollmentIDs: []string{hostUUID}},
		}
	}
	hostProfilesToInstallMap := map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload{
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
	ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
						Groups:   []fleet.ScimUserGroup{{DisplayName: "a"}, {DisplayName: "b"}},
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
						Groups:   []fleet.ScimUserGroup{{DisplayName: "a"}, {DisplayName: "b"}},
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "idp@example.com",
						Groups:   []fleet.ScimUserGroup{{DisplayName: "a"}, {DisplayName: "b"}},
					}, nil
				}
				ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
					return []*fleet.HostDeviceMapping{
						{
							Email: "other@example.com", Source: fleet.DeviceMappingGoogleChromeProfiles,
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
					}, nil
				}
				ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
					return []*fleet.HostDeviceMapping{
						{
							Email: "other@example.com", Source: fleet.DeviceMappingGoogleChromeProfiles,
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{}, nil
				}
				ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
					return []*fleet.HostDeviceMapping{
						{
							Email: "other@example.com", Source: fleet.DeviceMappingGoogleChromeProfiles,
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{}, nil
				}
				ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
					return []*fleet.HostDeviceMapping{
						{
							Email: "other@example.com", Source: fleet.DeviceMappingGoogleChromeProfiles,
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There are no IdP groups for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_GROUPS.")
			},
		},
		{
			desc:           "department, no idp user",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPDepartment),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There is no IdP department for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT.")
			},
		},
		{
			desc:           "groups with user groups, user has no groups",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPGroups),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There are no IdP groups for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_GROUPS.")
			},
		},
		{
			desc:           "profile with department, user has department",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPDepartment),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName:   "user1@example.com",
						Department: ptr.String("Engineering"),
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Len(t, targets, 0) // target is not present
				assert.Contains(t, updatedProfile.Detail, "There is no IdP department for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT.")
			},
		},
		{
			desc:           "profile with full name, user has full name",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPFullname),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName:   "user1@example.com",
						GivenName:  ptr.String("First"),
						FamilyName: ptr.String("Last"),
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName:  "user1@example.com",
						GivenName: ptr.String("First"),
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName:   "user1@example.com",
						FamilyName: ptr.String("Last"),
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
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Contains(t, updatedProfile.Detail, fmt.Sprintf("There is no IdP full name for this host. Fleet couldn't populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPFullname))
				assert.Len(t, targets, 0)
			},
		},
		{
			desc:           "profile with external ID, user has external ID",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPExternalID),
			expectedStatus: fleet.MDMDeliveryPending,
			setup: func() {
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName:   "user1@example.com",
						ExternalID: new("00u1abcd2EFGH3ijk4x7"),
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, updatedPayload.Detail)
				assert.Len(t, targets, 1)
				require.Equal(t, "00u1abcd2EFGH3ijk4x7", output)
			},
		},
		{
			desc:           "profile with external ID, user has no external ID",
			profileContent: "$FLEET_VAR_" + string(fleet.FleetVarHostEndUserIDPExternalID),
			expectedStatus: fleet.MDMDeliveryFailed,
			setup: func() {
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					require.EqualValues(t, 1, hostID)
					return &fleet.ScimUser{
						UserName: "user1@example.com",
					}, nil
				}
			},
			assert: func(output string) {
				assert.Empty(t, targets)
				assert.Contains(t, updatedProfile.Detail, "There is no IdP external ID for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID.")
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

			err := preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, noEJBCAService, logger, targets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, nil)
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

func TestPreprocessProfileContentsPSSORegistrationToken(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	appCfg := &fleet.AppConfig{}
	appCfg.ServerSettings.ServerURL = "https://test.example.com"
	appCfg.MDM.EnabledAndConfigured = true
	svc := scep.NewSCEPConfigService(logger, nil)
	digiCertService := digicert.NewService(digicert.WithLogger(logger))

	const hostUUID = "host-1"
	const cmdUUID = "cmd-1"
	placeholder := "$" + fleet.HostSecretPrefix + fleet.HostSecretPSSODeviceRegistrationToken

	newHostProfilesMap := func() map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload {
		m := make(map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload, 1)
		m[fleet.HostProfileUUID{HostUUID: hostUUID, ProfileUUID: "p1"}] = &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileUUID:       "p1",
			ProfileIdentifier: "com.fleetdm.platformsso",
			HostUUID:          hostUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryPending,
			CommandUUID:       cmdUUID,
			Scope:             fleet.PayloadScopeSystem,
		}
		return m
	}

	t.Run("only variable short-circuits the per-host split", func(t *testing.T) {
		ds := new(mock.Store)
		bulkUpsertCalled := false
		ds.BulkUpsertMDMAppleHostProfilesFunc = func(_ context.Context, _ []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
			bulkUpsertCalled = true
			return nil
		}

		targets := map[string]*fleet.CmdTarget{
			"p1": {CmdUUID: cmdUUID, ProfileIdentifier: "com.fleetdm.platformsso", EnrollmentIDs: []string{hostUUID}},
		}
		profileContents := map[string]mobileconfig.Mobileconfig{
			"p1": []byte("<string>" + fleet.FleetVarPSSODeviceRegistrationToken.WithPrefix() + "</string>"),
		}

		err := preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, noEJBCAService, logger, targets, profileContents, newHostProfilesMap(), make(map[string]string), &fleet.GroupedCertificateAuthorities{})
		require.NoError(t, err)

		// The original target is preserved (no per-host fan-out, no new temp UUIDs).
		require.Len(t, targets, 1)
		require.Contains(t, targets, "p1")
		require.Equal(t, cmdUUID, targets["p1"].CmdUUID)
		require.False(t, bulkUpsertCalled, "host profiles should not be re-keyed for the short-circuited profile")

		// The variable resolved to the host-secret placeholder on the shared content.
		require.Equal(t, "<string>"+placeholder+"</string>", string(profileContents["p1"]))
		require.NotContains(t, string(profileContents["p1"]), "FLEET_VAR_")
	})

	t.Run("multiple hosts share placeholder without per-host fan-out", func(t *testing.T) {
		const host2UUID = "host-2"
		const host3UUID = "host-3"

		ds := new(mock.Store)
		bulkUpsertCalled := false
		ds.BulkUpsertMDMAppleHostProfilesFunc = func(_ context.Context, _ []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
			bulkUpsertCalled = true
			return nil
		}

		enrollmentIDs := []string{hostUUID, host2UUID, host3UUID}
		targets := map[string]*fleet.CmdTarget{
			"p1": {CmdUUID: cmdUUID, ProfileIdentifier: "com.fleetdm.platformsso", EnrollmentIDs: enrollmentIDs},
		}
		profileContents := map[string]mobileconfig.Mobileconfig{
			"p1": []byte("<string>" + fleet.FleetVarPSSODeviceRegistrationToken.WithPrefix() + "</string>"),
		}

		multiHostProfilesMap := make(map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload)
		for _, hUUID := range enrollmentIDs {
			multiHostProfilesMap[fleet.HostProfileUUID{HostUUID: hUUID, ProfileUUID: "p1"}] = &fleet.MDMAppleBulkUpsertHostProfilePayload{
				ProfileUUID:       "p1",
				ProfileIdentifier: "com.fleetdm.platformsso",
				HostUUID:          hUUID,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				CommandUUID:       cmdUUID,
				Scope:             fleet.PayloadScopeSystem,
			}
		}

		err := preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, noEJBCAService, logger, targets, profileContents, multiHostProfilesMap, make(map[string]string), &fleet.GroupedCertificateAuthorities{})
		require.NoError(t, err)

		// Still one target — no per-host fan-out regardless of host count.
		require.Len(t, targets, 1)
		require.Contains(t, targets, "p1")
		require.Equal(t, cmdUUID, targets["p1"].CmdUUID)
		// All enrollment IDs remain in the single shared target.
		require.ElementsMatch(t, enrollmentIDs, targets["p1"].EnrollmentIDs)
		require.False(t, bulkUpsertCalled, "host profiles should not be re-keyed for the short-circuited profile")

		// Single shared profile content with the placeholder — no per-host copies.
		require.Equal(t, "<string>"+placeholder+"</string>", string(profileContents["p1"]))
		require.NotContains(t, string(profileContents["p1"]), "FLEET_VAR_")
	})

	t.Run("mixed with a host-specific variable still fans out and keeps the placeholder", func(t *testing.T) {
		ds := new(mock.Store)
		ds.BulkUpsertMDMAppleHostProfilesFunc = func(_ context.Context, _ []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
			return nil
		}

		targets := map[string]*fleet.CmdTarget{
			"p1": {CmdUUID: cmdUUID, ProfileIdentifier: "com.fleetdm.platformsso", EnrollmentIDs: []string{hostUUID}},
		}
		profileContents := map[string]mobileconfig.Mobileconfig{
			"p1": []byte("<string>" + fleet.FleetVarPSSODeviceRegistrationToken.WithPrefix() + "</string><string>" + fleet.FleetVarHostUUID.WithPrefix() + "</string>"),
		}

		err := preprocessProfileContents(ctx, appCfg, ds, svc, digiCertService, noEJBCAService, logger, targets, profileContents, newHostProfilesMap(), make(map[string]string), &fleet.GroupedCertificateAuthorities{})
		require.NoError(t, err)

		// The original target is replaced by a per-host target.
		require.NotContains(t, targets, "p1")
		require.Len(t, targets, 1)
		var tempUUID string
		for k := range targets {
			tempUUID = k
		}
		hostContents := string(profileContents[tempUUID])
		require.Contains(t, hostContents, placeholder)
		require.Contains(t, hostContents, hostUUID)
		require.NotContains(t, hostContents, "FLEET_VAR_")
	})
}

func TestPreprocessProfileContentsEJBCA(t *testing.T) {
	t.Parallel()
	hostsByUUID := map[string]*fleet.Host{
		"host-1": {ID: 1, UUID: "host-1", HardwareSerial: "SERIAL-AAA", Platform: "darwin"},
		"host-2": {ID: 2, UUID: "host-2", HardwareSerial: "SERIAL-BBB", Platform: "darwin"},
	}
	ejbcaCA := func(name string) fleet.EJBCA {
		return fleet.EJBCA{
			Name:                          name,
			CertificateAuthorityName:      "ManagementCA",
			CertificateProfileName:        "certProfile",
			EndEntityProfileName:          "eeProfile",
			CertificateCommonName:         "cn-" + fleet.FleetVarHostHardwareSerial.WithPrefix(),
			CertificateUserPrincipalNames: []string{fleet.FleetVarHostHardwareSerial.WithPrefix() + "@example.com"},
		}
	}
	// profileFor is a profile that gets a certificate from each of the EJBCA CAs.
	profileFor := func(caNames ...string) string {
		var profile strings.Builder
		for _, name := range caNames {
			fmt.Fprintf(&profile, "<string>$FLEET_VAR_EJBCA_PASSWORD_%[1]s</string><data>${FLEET_VAR_EJBCA_DATA_%[1]s}</data>", name)
		}
		return profile.String()
	}
	type result struct {
		enrolls  []fleet.EJBCA                  // what the EJBCA service was asked to enroll, in order
		contents map[string]string              // the profile each host got, by host UUID
		managed  []*fleet.MDMManagedCertificate // the records saved
		failures map[string]string              // the failure detail the profile of each host got, by host UUID
		profiles map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload
		retried  map[string][]string // the profiles that were queued to be retried, by host UUID
	}
	// opts is for what a test needs besides EJBCA CAs: a failing enrollment, DigiCert CAs with their service, and how many times the
	// profile of each host has been retried already.
	type opts struct {
		enroll      func(fleet.EJBCA) error
		digiCert    fleet.DigiCertService
		digiCertCAs []fleet.DigiCertCA
		retries     uint
	}
	run := func(t *testing.T, ctx context.Context, profile string, cas []fleet.EJBCA, o opts) result {
		logger := slog.New(slog.DiscardHandler)
		appCfg := &fleet.AppConfig{}
		appCfg.MDM.EnabledAndConfigured = true
		res := result{contents: map[string]string{}, failures: map[string]string{}, retried: map[string][]string{}}

		ds := new(mock.Store)
		ds.ListHostsLiteByUUIDsFunc = func(_ context.Context, _ fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
			return []*fleet.Host{hostsByUUID[uuids[0]]}, nil
		}
		// No host has an IdP email, which is what the test of a variable that can't be filled in needs.
		ds.GetHostEmailsFunc = func(context.Context, string, string) ([]string, error) { return nil, nil }
		ds.BulkUpsertMDMAppleHostProfilesFunc = func(context.Context, []*fleet.MDMAppleBulkUpsertHostProfilePayload) error { return nil }
		ds.BulkUpsertMDMManagedCertificatesFunc = func(_ context.Context, payload []*fleet.MDMManagedCertificate) error {
			res.managed = append(res.managed, payload...)
			return nil
		}
		ds.UpdateOrDeleteHostMDMAppleProfileFunc = func(_ context.Context, p *fleet.HostMDMAppleProfile) error {
			res.failures[p.HostUUID] = p.Detail
			return nil
		}
		ds.GetHostMDMProfilesRetryCountsFunc = func(context.Context, *fleet.Host) ([]fleet.HostMDMProfileRetryCount, error) {
			return []fleet.HostMDMProfileRetryCount{{ProfileIdentifier: "com.apple.security.pkcs12", Retries: o.retries}}, nil
		}
		ds.UpdateHostMDMProfilesVerificationFunc = func(_ context.Context, host *fleet.Host, _, _, toRetry []string) error {
			assert.Equal(t, "darwin", host.Platform)
			res.retried[host.UUID] = toRetry
			return nil
		}

		ejbcaMock := &ejbca_mock.Service{}
		ejbcaMock.GetCertificateFunc = func(_ context.Context, config fleet.EJBCA) (*fleet.EJBCACertificate, error) {
			res.enrolls = append(res.enrolls, config)
			if o.enroll != nil {
				if err := o.enroll(config); err != nil {
					return nil, err
				}
			}
			serial := strings.TrimSuffix(config.CertificateUserPrincipalNames[0], "@example.com")
			now := time.Now()
			return &fleet.EJBCACertificate{
				PfxData: []byte("pfx-" + config.Name + "-" + serial), Password: "password-" + config.Name + "-" + serial,
				NotValidBefore: now, NotValidAfter: now.Add(time.Hour), SerialNumber: "abc",
			}, nil
		}

		res.profiles = map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload{}
		for uuid := range hostsByUUID {
			res.profiles[fleet.HostProfileUUID{HostUUID: uuid, ProfileUUID: "p1"}] = &fleet.MDMAppleBulkUpsertHostProfilePayload{
				ProfileUUID: "p1", ProfileIdentifier: "com.apple.security.pkcs12", HostUUID: uuid, OperationType: fleet.MDMOperationTypeInstall,
				Status: &fleet.MDMDeliveryPending, CommandUUID: "cmd-1", Scope: fleet.PayloadScopeSystem,
			}
		}
		targets := map[string]*fleet.CmdTarget{
			"p1": {CmdUUID: "cmd-1", ProfileIdentifier: "com.apple.security.pkcs12", EnrollmentIDs: []string{"host-1", "host-2"}},
		}
		profileContents := map[string]mobileconfig.Mobileconfig{"p1": mobileconfig.Mobileconfig(profile)}
		digiCertService := noDigiCertService
		if o.digiCert != nil {
			digiCertService = o.digiCert
		}

		err := preprocessProfileContents(ctx, appCfg, ds, scep.NewSCEPConfigService(logger, nil), digiCertService, ejbcaMock, logger, targets,
			profileContents, res.profiles, map[string]string{}, &fleet.GroupedCertificateAuthorities{EJBCA: cas, DigiCert: o.digiCertCAs})
		require.NoError(t, err)
		for profUUID, target := range targets {
			if profUUID != "p1" {
				res.contents[target.EnrollmentIDs[0]] = string(profileContents[profUUID])
			}
		}
		return res
	}
	premium := func() context.Context {
		return license.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	}

	t.Run("expands the CA fields for each host and records the certificates", func(t *testing.T) {
		// The username of the first CA defaults to its common name, and the second CA has its own template.
		withUsername := ejbcaCA("usernameCA")
		withUsername.UsernameTemplate = fleet.FleetVarHostUUID.WithPrefix()
		res := run(t, premium(), profileFor("defaultCA", "usernameCA"), []fleet.EJBCA{ejbcaCA("defaultCA"), withUsername}, opts{})

		require.Len(t, res.enrolls, 4)
		for _, config := range res.enrolls {
			// Each host's values are the ones of its own serial, and nothing is left from the other host.
			serial := strings.TrimSuffix(config.CertificateUserPrincipalNames[0], "@example.com")
			assert.Equal(t, "cn-"+serial, config.CertificateCommonName)
			if config.Name == "defaultCA" {
				assert.Equal(t, "cn-"+serial, config.UsernameTemplate)
			} else {
				assert.Contains(t, []string{"host-1", "host-2"}, config.UsernameTemplate)
			}
		}

		// The profile of each host has the certificate and password of each CA, and no variable is left.
		require.Len(t, res.contents, 2)
		for host, contents := range res.contents {
			for _, name := range []string{"defaultCA", "usernameCA"} {
				entry := name + "-" + hostsByUUID[host].HardwareSerial
				assert.Contains(t, contents, base64.StdEncoding.EncodeToString([]byte("pfx-"+entry)))
				assert.Contains(t, contents, "password-"+entry)
			}
			assert.NotContains(t, contents, "FLEET_VAR_")
		}

		require.Len(t, res.managed, 4)
		for _, m := range res.managed {
			assert.Equal(t, fleet.CAConfigEJBCA, m.Type)
			assert.Equal(t, "p1", m.ProfileUUID)
			assert.Equal(t, "abc", *m.Serial)
		}
		assert.Empty(t, res.failures)
	})

	t.Run("a failed enrollment fails the profile of that host with the reason", func(t *testing.T) {
		res := run(t, premium(), profileFor("myCA"), []fleet.EJBCA{ejbcaCA("myCA")}, opts{enroll: func(fleet.EJBCA) error { return errors.New("boom") }})
		assert.Equal(t, map[string]string{
			"host-1": "Couldn't get certificate from EJBCA for myCA. boom",
			"host-2": "Couldn't get certificate from EJBCA for myCA. boom",
		}, res.failures)
		assert.Empty(t, res.managed)
		assert.Empty(t, res.contents)
	})

	t.Run("a CA that is unavailable retries the profile until the retries are used up", func(t *testing.T) {
		enroll := func(fleet.EJBCA) error {
			return fleet.CertificateAuthorityTransientError{Message: "EJBCA is unavailable"}
		}

		res := run(t, premium(), profileFor("myCA"), []fleet.EJBCA{ejbcaCA("myCA")}, opts{enroll: enroll, retries: mdm.MaxAppleProfileRetries - 1})
		assert.Equal(t, map[string][]string{"host-1": {"com.apple.security.pkcs12"}, "host-2": {"com.apple.security.pkcs12"}}, res.retried)
		assert.Empty(t, res.failures)

		res = run(t, premium(), profileFor("myCA"), []fleet.EJBCA{ejbcaCA("myCA")}, opts{enroll: enroll, retries: mdm.MaxAppleProfileRetries})
		assert.Empty(t, res.retried)
		assert.Equal(t, map[string]string{
			"host-1": "Couldn't get certificate from EJBCA for myCA. EJBCA is unavailable",
			"host-2": "Couldn't get certificate from EJBCA for myCA. EJBCA is unavailable",
		}, res.failures)
	})

	t.Run("a variable that can't be filled in says it is in the CA", func(t *testing.T) {
		ca := ejbcaCA("myCA")
		ca.CertificateCommonName = fleet.FleetVarHostEndUserEmailIDP.WithPrefix()
		res := run(t, premium(), profileFor("myCA"), []fleet.EJBCA{ca}, opts{})
		assert.Empty(t, res.enrolls)
		require.Len(t, res.failures, 2)
		for _, detail := range res.failures {
			assert.Contains(t, detail, "There is no IdP email for this host")
			assert.Contains(t, detail, "EJBCA certificate authority")
		}
	})

	for _, tt := range []struct {
		name    string
		ctx     context.Context
		cas     []fleet.EJBCA
		message string
	}{
		{"a CA that doesn't exist", premium(), nil, "Fleet couldn't populate $EJBCA_PASSWORD_myCA because myCA certificate authority doesn't exist."},
		{
			"EJBCA needs Fleet Premium", license.NewContext(t.Context(), &fleet.LicenseInfo{Tier: fleet.TierFree}),
			[]fleet.EJBCA{ejbcaCA("myCA")}, "EJBCA integration requires a Fleet Premium license.",
		},
	} {
		t.Run(tt.name+" fails the profile", func(t *testing.T) {
			res := run(t, tt.ctx, profileFor("myCA"), tt.cas, opts{})
			assert.Empty(t, res.enrolls)
			require.Len(t, res.profiles, 2)
			for _, p := range res.profiles {
				assert.Equal(t, fleet.MDMDeliveryFailed, *p.Status)
				assert.Equal(t, tt.message, p.Detail)
			}
		})
	}

	t.Run("a failure later in the profile doesn't record the certificates issued before it", func(t *testing.T) {
		digiCert := &digicert_mock.Service{}
		digiCert.GetCertificateFunc = func(context.Context, fleet.DigiCertCA) (*fleet.DigiCertCertificate, error) {
			return &fleet.DigiCertCertificate{PfxData: []byte("pfx"), Password: "password", SerialNumber: "abc"}, nil
		}
		// The variables are processed longest first, not in the order of the profile, so the DigiCert variable, which is longer, is issued
		// before EJBCA fails.
		profile := profileFor("myCA") + "<string>$FLEET_VAR_DIGICERT_PASSWORD_aLongerDigiCertCA</string>" +
			"<data>${FLEET_VAR_DIGICERT_DATA_aLongerDigiCertCA}</data>"
		res := run(t, premium(), profile, []fleet.EJBCA{ejbcaCA("myCA")}, opts{
			enroll:      func(fleet.EJBCA) error { return errors.New("boom") },
			digiCert:    digiCert,
			digiCertCAs: []fleet.DigiCertCA{{Name: "aLongerDigiCertCA", CertificateCommonName: "cn", CertificateSeatID: "seat"}},
		})
		require.True(t, digiCert.GetCertificateFuncInvoked, "DigiCert has to be issued before the failure for this test to mean anything")
		assert.Len(t, res.failures, 2)
		assert.Empty(t, res.managed)
		assert.Empty(t, res.contents)
	})
}
