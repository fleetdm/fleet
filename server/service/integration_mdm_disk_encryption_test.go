package service

// Disk encryption tests for the MDM suite.
//
// Belongs here: macOS FileVault, Windows BitLocker (including PINs and enforcement notifications),
// and Linux disk encryption settings at the global, fleet, and per-platform level; escrowing,
// retrieving, rotating, and archiving encryption keys; and disk encryption status, aggregates, and
// which hosts count toward them.
//
// Does not belong here: Apple recovery lock passwords (integration_mdm_recovery_lock_test.go) and
// configuration profiles in general (integration_mdm_profiles_*_test.go).

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/assets"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestMDMDiskEncryptionSettingBackwardsCompat() {
	t := s.T()

	// the deprecated flat toggle reads as the AND of the four per-platform
	// settings; a legacy-only write fans out to all of them
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": false }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	// a per-platform change wins over an unchanged flat toggle sent alongside
	// it: macOS enforcement turns on, the flat toggle still reads false (not
	// all platforms are on) and the FileVault profile exists
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "enable_disk_encryption": false, "macos_settings": {"enable_disk_encryption": true} }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption.Value)
	assert.False(t, acResp.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// changing the flat toggle and a per-platform setting to disagreeing
	// values in one request is a conflict
	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "enable_disk_encryption": true, "macos_settings": {"enable_disk_encryption": false} }
  }`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "conflicts with per-platform disk encryption settings")

	// changing the flat toggle fans out to every per-platform setting, winning
	// over per-platform values it does not change
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.True(t, acResp.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	assert.True(t, acResp.MDM.WindowsSettings.EnableDiskEncryption.Value)
	assert.True(t, acResp.MDM.LinuxSettings.EnableEscrowDiskEncryptionKey.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// turning the flat toggle off turns everything off and removes the profile
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "enable_disk_encryption": false }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.False(t, acResp.MDM.MacOSSettings.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// unrelated change doesn't affect the disk encryption setting
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "macos_settings": {"custom_settings": ["test.mobileconfig"]} }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	// Same tests, but for teams
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)

	checkTeamDiskEncryption := func(wantFlat, wantMacOSEnable bool) {
		var teamResp getTeamResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
		require.Equal(t, wantFlat, teamResp.Team.Config.MDM.EnableDiskEncryption)
		require.Equal(t, wantMacOSEnable, teamResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption.Value)
	}

	// after creation, disk encryption is off
	checkTeamDiskEncryption(false, false)

	// a per-platform change wins over an unchanged flat toggle sent alongside it
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
			MacOSSettings:        map[string]any{"enable_disk_encryption": true},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
	checkTeamDiskEncryption(false, true)
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// changing the flat toggle fans out over per-platform values it does not change
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
			MacOSSettings:        map[string]any{"enable_disk_encryption": true},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
	checkTeamDiskEncryption(false, true) // no change: same spec re-applied
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// changing the flat toggle and a per-platform setting to disagreeing
	// values in one request is a conflict
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
			MacOSSettings:        map[string]any{"enable_disk_encryption": false},
		},
	}}}
	res = s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "conflicts with per-platform disk encryption settings")

	// a legacy-only write fans out to all platforms
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
	checkTeamDiskEncryption(false, false)
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// unrelated change doesn't affect the disk encryption setting
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
			MacOSSettings:        map[string]any{"custom_settings": []any{"A", "B"}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
	checkTeamDiskEncryption(false, false)
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)
}

func (s *integrationMDMTestSuite) TestDiskEncryptionSharedSetting() {
	t := s.T()

	// create a team
	teamName := t.Name()
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc " + teamName,
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)

	setMDMEnabled := func(macMDM, windowsMDM bool) {
		appConf, err := s.ds.AppConfig(context.Background())
		s.Require().NoError(err)
		appConf.MDM.WindowsEnabledAndConfigured = windowsMDM
		appConf.MDM.EnabledAndConfigured = macMDM
		err = s.ds.SaveAppConfig(context.Background(), appConf)
		s.Require().NoError(err)
	}

	// before doing any modifications, grab the current values and make
	// sure they're set to the same ones on cleanup to not interfere with
	// other tests.
	origAppConf, err := s.ds.AppConfig(context.Background())
	s.Require().NoError(err)
	t.Cleanup(func() {
		err := s.ds.SaveAppConfig(context.Background(), origAppConf)
		s.Require().NoError(err)
	})

	checkConfigSetSucceeds := func() {
		res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK)
		errMsg := extractServerErrorText(res.Body)
		require.Empty(t, errMsg)

		// try to create a new team using specs
		teamSpecs := map[string]any{
			"specs": []any{
				map[string]any{
					"name": teamName + uuid.NewString(),
					"mdm": map[string]any{
						"enable_disk_encryption": true,
					},
				},
			},
		}
		res = s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
		errMsg = extractServerErrorText(res.Body)
		require.Empty(t, errMsg)

		// edit the existing team using specs
		teamSpecs = map[string]any{
			"specs": []any{
				map[string]any{
					"name": teamName,
					"mdm": map[string]any{
						"enable_disk_encryption": true,
					},
				},
			},
		}
		res = s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
		errMsg = extractServerErrorText(res.Body)
		require.Empty(t, errMsg)

		// always try to set the value to `false` so we start fresh
		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": false }
  }`), http.StatusOK)
		teamSpecs = map[string]any{
			"specs": []any{
				map[string]any{
					"name": teamName,
					"mdm": map[string]any{
						"enable_disk_encryption": false,
					},
				},
			},
		}
		s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
	}

	// MDM config succeeds because we have a private key baked into default suite config
	setMDMEnabled(false, false)
	checkConfigSetSucceeds()

	// enable windows mdm, no errors
	setMDMEnabled(false, true)
	checkConfigSetSucceeds()

	// enable mac mdm, no errors
	setMDMEnabled(true, true)
	checkConfigSetSucceeds()

	// only macos mdm enabled, no errors
	setMDMEnabled(true, false)
	checkConfigSetSucceeds()
}

func (s *integrationMDMTestSuite) TestEscrowBuddyBackwardsCompat() {
	t := s.T()
	ctx := context.Background()

	// create a host
	host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// install a filevault profile for that host
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// set the status as non-decryptable so a notification should be sent
	_, err := s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, "", "", new(false))
	require.NoError(t, err)

	// notification is false because the escrow buddy capability is not set
	orbitConfigResp := fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
	require.False(t, orbitConfigResp.Notifications.RotateDiskEncryptionKey)

	// send the request again, this time with the right header
	orbitConfigResp = fleet.OrbitGetConfigResponse{}
	res := s.DoRawWithHeaders("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, map[string]string{
		"Authorization":          fmt.Sprintf("Bearer %s", s.token),
		fleet.CapabilitiesHeader: string(fleet.CapabilityEscrowBuddy),
	})
	err = json.NewDecoder(res.Body).Decode(&orbitConfigResp)
	require.NoError(t, err)
	require.True(t, orbitConfigResp.Notifications.RotateDiskEncryptionKey)
}

func (s *integrationMDMTestSuite) TestMDMAppleHostDiskEncryption() {
	t := s.T()
	ctx := t.Context()

	// create a host
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// install a filevault profile for that host

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	fileVaultProf := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)
	hostCmdUUID := uuid.New().String()
	err = s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileUUID:       fileVaultProf.ProfileUUID,
			ProfileIdentifier: fileVaultProf.Identifier,
			HostUUID:          host.UUID,
			CommandUUID:       hostCmdUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryPending,
			Checksum:          []byte("csum"),
			Scope:             fleet.PayloadScopeSystem,
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err := s.ds.UpdateOrDeleteHostMDMAppleProfile(context.Background(), &fleet.HostMDMAppleProfile{
			HostUUID:      host.UUID,
			CommandUUID:   hostCmdUUID,
			ProfileUUID:   fileVaultProf.ProfileUUID,
			Status:        &fleet.MDMDeliveryVerifying,
			OperationType: fleet.MDMOperationTypeRemove,
		})
		require.NoError(t, err)
		// not an error if the profile does not exist
		_ = s.ds.DeleteMDMAppleConfigProfile(context.Background(), fileVaultProf.ProfileUUID)
	})

	// get that host - it should
	// report "enforcing" disk encryption
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Empty(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// report a profile install error
	err = s.ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		HostUUID:      host.UUID,
		CommandUUID:   hostCmdUUID,
		ProfileUUID:   fileVaultProf.ProfileUUID,
		Status:        &fleet.MDMDeliveryFailed,
		OperationType: fleet.MDMOperationTypeInstall,
		Detail:        "test error",
	})
	require.NoError(t, err)

	// get that host - it should report "failed" disk encryption and include the error message detail
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionFailed, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionFailed, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "test error", getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// report that the profile was installed and verified
	err = s.ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		HostUUID:      host.UUID,
		CommandUUID:   hostCmdUUID,
		ProfileUUID:   fileVaultProf.ProfileUUID,
		Status:        &fleet.MDMDeliveryVerified,
		OperationType: fleet.MDMOperationTypeInstall,
		Detail:        "",
	})
	require.NoError(t, err)

	// get that host - it has no encryption key at this point, so it should
	// report "action_required" disk encryption and "log_out" action.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.Equal(t, fleet.ActionRequiredRotateKey, *getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Empty(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// add an encryption key for the host
	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	}, nil)
	require.NoError(t, err)

	block, _ := pem.Decode(assets[fleet.MDMAssetCACert].Value)
	require.NotNil(t, block)

	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	recoveryKey := "AAA-BBB-CCC"
	encryptedKey, err := pkcs7.Encrypt([]byte(recoveryKey), []*x509.Certificate{parsed})
	require.NoError(t, err)
	base64EncryptedKey := base64.StdEncoding.EncodeToString(encryptedKey)

	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, base64EncryptedKey, "", nil)
	require.NoError(t, err)

	// get that host - it has an encryption key with unknown decryptability, so
	// it should report "verifying" disk encryption.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionVerifying, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionVerifying, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Empty(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// request with no token
	res := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusUnauthorized)
	res.Body.Close()

	// decryption status not processed yet, but we'll still return the key on request if we can decrypt it
	resp := getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue)

	// simulate the cron marking the decryption status as false
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, false, time.Now())
	require.NoError(t, err)

	// even if decryption status was set to false, we still try to decrypt when the key is requested
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue)

	// get that host - it has an encryption key that is un-decryptable, so it
	// should report "action_required" disk encryption and "rotate_key" action.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.Equal(t, fleet.ActionRequiredRotateKey, *getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Empty(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// no activities created so far
	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)
	count := 0
	for _, activity := range activities.Activities {
		if activity.Type != "read_host_disk_encryption_key" || activity.Details == nil {
			continue
		}
		// Scope to this host: the suite shares a database, so any other test that reads
		// an encryption key would otherwise be counted here.
		var details struct {
			HostID uint `json:"host_id"`
		}
		require.NoError(t, json.Unmarshal(*activity.Details, &details))
		if details.HostID == host.ID {
			count++
		}
	}
	require.Equal(t, 2, count) // two successful requests so far

	// decryptable key
	checkDecryptableKey := func(u fleet.User) {
		err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, true, time.Now())
		require.NoError(t, err)
		resp = getHostEncryptionKeyResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
		require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue)

		// use the admin token to get the activities
		currToken := s.token
		defer func() { s.token = currToken }()
		s.token = s.getTestAdminToken()
		s.lastActivityMatches(
			"read_host_disk_encryption_key",
			fmt.Sprintf(`{"host_display_name": "%s", "host_id": %d}`, host.DisplayName(), host.ID),
			0,
		)
	}

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          4827,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)

	// enable disk encryption on the team so the key is not deleted when the host is added
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: "team1_" + t.Name(),
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)

	// we're about to mess up with the token, make sure to set it to the
	// default value when the test ends
	currToken := s.token
	t.Cleanup(func() { s.token = currToken })

	// admins are able to see the host encryption key
	s.token = s.getTestAdminToken()
	checkDecryptableKey(s.users["admin1@example.com"])

	// get that host - it has an encryption key that is decryptable, so it
	// should report "verified" disk encryption.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionVerified, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionVerified, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Empty(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// maintainers are able to see the token
	u := s.users["user1@example.com"]
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// observers are able to see the token
	u = s.users["user2@example.com"]
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// add the host to a team
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
	require.NoError(t, err)

	// admins are still able to see the token
	s.token = s.getTestAdminToken()
	checkDecryptableKey(s.users["admin1@example.com"])

	// maintainers are still able to see the token
	u = s.users["user1@example.com"]
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// observers are still able to see the token
	u = s.users["user2@example.com"]
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// add a team member
	u = fleet.User{
		Name:       "test team user",
		Email:      "user1+team@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team,
				Role: fleet.RoleMaintainer,
			},
		},
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, &u)
	require.NoError(t, err)

	// members are able to see the token
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// create a separate team
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          4828,
		Name:        "team2_" + t.Name(),
		Description: "desc team2_" + t.Name(),
	})
	require.NoError(t, err)
	// add a team member
	u = fleet.User{
		Name:       "test team user",
		Email:      "user1+team2@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team2,
				Role: fleet.RoleMaintainer,
			},
		},
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, &u)
	require.NoError(t, err)

	// non-members aren't able to see the token
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusForbidden, &resp)

	// clear the current key and archived key still exists
	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, "", "", nil)
	require.NoError(t, err)

	// switch to the admin token to get the key
	s.token = s.getTestAdminToken()
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue) // old key is pulled from the archive

	detailsResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &detailsResp)
	require.False(t, detailsResp.Host.MDM.EncryptionKeyAvailable)
	require.NotNil(t, detailsResp.Host.MDM.EncryptionKeyArchived)
	require.True(t, *detailsResp.Host.MDM.EncryptionKeyArchived)
}

func (s *integrationMDMTestSuite) TestMDMAppleDiskEncryptionAfterTransferToFleetWithoutIt() {
	t := s.T()
	ctx := t.Context()

	srcTeamName := "fv_src_" + t.Name()
	s.Do("POST", "/api/latest/fleet/spec/fleets", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: srcTeamName,
		MDM:  fleet.TeamSpecMDM{EnableDiskEncryption: optjson.SetBool(true)},
	}}}, http.StatusOK)
	srcTeam, err := s.ds.TeamByName(ctx, srcTeamName)
	require.NoError(t, err)
	dstTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "fv_dst_" + t.Name()})
	require.NoError(t, err)

	token := "fv_transfer_token" //nolint:gosec // G101: test value, not a real credential
	host := createHostAndDeviceToken(t, s.ds, token)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&srcTeam.ID, []uint{host.ID})))

	// FileVault delivered and the key escrowed and decryptable
	fileVaultProf := s.assertConfigProfilesByIdentifier(&srcTeam.ID, mobileconfig.FleetFileVaultPayloadIdentifier, true)
	require.NoError(t, s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
		ProfileUUID:       fileVaultProf.ProfileUUID,
		ProfileIdentifier: fileVaultProf.Identifier,
		HostUUID:          host.UUID,
		CommandUUID:       uuid.New().String(),
		OperationType:     fleet.MDMOperationTypeInstall,
		Status:            &fleet.MDMDeliveryVerified,
		Checksum:          []byte("csum"),
		Scope:             fleet.PayloadScopeSystem,
	}}))
	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, "key", "", nil)
	require.NoError(t, err)
	require.NoError(t, s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, true, time.Now().Add(time.Minute)))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, fleet.DiskEncryptionVerified, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)

	// The transfer deletes the key, but the profile cron hasn't queued the
	// profile's removal yet: there is nothing for the end user to do.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{TeamID: &dstTeam.ID, HostIDs: []uint{host.ID}}, http.StatusOK, &addHostsToTeamResponse{})

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.False(t, getHostResp.Host.MDM.EncryptionKeyAvailable)
	require.Equal(t, fleet.DiskEncryptionRemovingEnforcement, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.Equal(t, fleet.DiskEncryptionRemovingEnforcement, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Nil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.ActionRequired)

	// the profile row reads "Removing enforcement" (remove + pending), not "Enforcing"
	requireFileVaultRowRemoving := func(profiles *[]fleet.HostMDMProfile) {
		require.NotNil(t, profiles)
		for _, p := range *profiles {
			if p.ProfileUUID == fileVaultProf.ProfileUUID {
				require.Equal(t, fleet.MDMOperationTypeRemove, p.OperationType)
				require.Equal(t, string(fleet.MDMDeliveryPending), *p.Status)
				return
			}
		}
		require.Fail(t, "FileVault profile row not found")
	}
	requireFileVaultRowRemoving(getHostResp.Host.MDM.Profiles)

	var getDeviceResp getDeviceHostResponse
	s.DoJSON("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK, &getDeviceResp)
	require.Equal(t, fleet.DiskEncryptionRemovingEnforcement, *getDeviceResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getDeviceResp.Host.MDM.MacOSSettings.ActionRequired)
	requireFileVaultRowRemoving(getDeviceResp.Host.MDM.Profiles)
}

func (s *integrationMDMTestSuite) TestMDMAppleHostDiskEncryptionWithDisabledEncryptionSetting() {
	t := s.T()
	ctx := context.Background()

	// Create a macOS host enrolled in Fleet's MDM. Fleet MDM enrollment is required
	// to escrow a disk encryption key (see the IsHostConnectedToFleetMDM check in
	// the darwin key ingestion).
	host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// Turn on disk encryption for the global team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "enable_disk_encryption": true } }`), http.StatusOK, &acResp)
	assert.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value)

	// Get the CA key pair for encryption/decryption
	cert, err := assets.CAKeyPair(ctx, s.ds)
	require.NoError(t, err)

	submitFileVaultKey := func(nodeKey, base64Key string, expectArchived bool) {
		// Encrypt the key with the CA cert
		encryptedKey, err := pkcs7.Encrypt([]byte(base64Key), []*x509.Certificate{cert.Leaf})
		require.NoError(t, err)
		encryptedBase64Key := base64.StdEncoding.EncodeToString(encryptedKey)

		// Submit the detail query results
		distributedReq := SubmitDistributedQueryResultsRequest{
			NodeKey: nodeKey,
			Results: map[string][]map[string]string{
				"fleet_detail_query_mdm_disk_encryption_key_file_darwin": {
					{
						"encrypted":     "1",
						"filevault_key": encryptedBase64Key,
					},
				},
			},
			Statuses: map[string]fleet.OsqueryStatus{
				"fleet_detail_query_mdm_disk_encryption_key_file_darwin": 0,
			},
		}
		distributedResp := submitDistributedQueryResultsResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

		// Key archived
		hdek, err := s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
		if expectArchived {
			require.NoError(t, err)
			require.NotEmpty(t, hdek.Base64Encrypted)
			// Decrypt and verify it matches what we sent
			decrypted, err := servermdm.DecryptBase64CMS(hdek.Base64Encrypted, cert.Leaf, cert.PrivateKey)
			require.NoError(t, err)
			require.Equal(t, base64Key, string(decrypted))
		}
	}

	activities := s.listActivities()
	escrowCountBefore := 0
	for _, activity := range activities {
		if activity.Type == "escrowed_disk_encryption_key" {
			escrowCountBefore++
		}
	}

	submitFileVaultKey(*host.NodeKey, "ABC-111-222", true)

	// Verify activity was created with correct details
	activities = s.listActivities()
	escrowKeyActivity := fleet.ActivityTypeEscrowedDiskEncryptionKey{}
	var seenEscrowKeyActivityID uint
	for _, activity := range activities {
		if activity.Type == escrowKeyActivity.ActivityName() {
			if activity.ID <= seenEscrowKeyActivityID {
				continue
			}
			err := json.Unmarshal(*activity.Details, &escrowKeyActivity)
			require.NoError(t, err)
			require.True(t, activity.FleetInitiated)
			seenEscrowKeyActivityID = activity.ID
		}
	}
	require.NotZero(t, seenEscrowKeyActivityID)
	require.Equal(t, escrowKeyActivity.HostID, host.ID)
	require.Equal(t, escrowKeyActivity.HostDisplayName, host.DisplayName())

	// Disable disk encryption
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "enable_disk_encryption": false } }`), http.StatusOK, &acResp)
	assert.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value)

	// Submit another key with disk encryption disabled
	lastSeenActivityID := seenEscrowKeyActivityID
	submitFileVaultKey(*host.NodeKey, "DEF-333-444", false)
	hdek, err := s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	decrypted, err := servermdm.DecryptBase64CMS(hdek.Base64Encrypted, cert.Leaf, cert.PrivateKey)
	require.NoError(t, err)
	require.Equal(t, "ABC-111-222", string(decrypted))

	// No activity
	activities = s.listActivities()
	for _, activity := range activities {
		if activity.Type == "escrowed_disk_encryption_key" {
			require.LessOrEqual(t, activity.ID, lastSeenActivityID)
		}
	}

	// Re-enable disk encryption
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "enable_disk_encryption": true } }`), http.StatusOK, &acResp)
	assert.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value)

	// Submit new key with disk encryption enabled
	submitFileVaultKey(*host.NodeKey, "GHI-555-666", true)

	// Verify activity was created with correct details
	activities = s.listActivities()
	escrowKeyActivity = fleet.ActivityTypeEscrowedDiskEncryptionKey{}
	var newActivityID uint
	for _, activity := range activities {
		if activity.Type == escrowKeyActivity.ActivityName() && activity.ID > lastSeenActivityID {
			err := json.Unmarshal(*activity.Details, &escrowKeyActivity)
			require.NoError(t, err)
			require.True(t, activity.FleetInitiated)
			newActivityID = activity.ID
		}
	}
	require.NotZero(t, newActivityID)
	require.Equal(t, escrowKeyActivity.HostID, host.ID)
	require.Equal(t, escrowKeyActivity.HostDisplayName, host.DisplayName())
}

func (s *integrationMDMTestSuite) TestWindowsMDMGetEncryptionKey() {
	t := s.T()
	ctx := context.Background()

	// create a host and enroll it in Fleet
	host := createOrbitEnrolledHost(t, "windows", "h1", s.ds)
	err := s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)
	// Without an enrollment row the server can't ask the host for a new key, so the error wouldn't promise one.
	require.NoError(t, s.ds.MDMWindowsInsertEnrolledDevice(ctx, &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            "disk-encryption-key-" + host.UUID,
		MDMHardwareID:          "disk-encryption-key-hw-" + host.UUID,
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          host.Hostname,
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.26100",
		MDMNotInOOBE:           true,
		HostUUID:               host.UUID,
	}))

	// request encryption key with no auth token
	res := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusUnauthorized)
	res.Body.Close()

	// no encryption key
	resp := getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusNotFound, &resp)

	// invalid host id
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID+999), nil, http.StatusNotFound, &resp)

	// add an encryption key for the host, the way the escrow endpoint stores it
	recoveryKey := "AAA-BBB-CCC"
	encryptedKey, err := servermdm.EncryptAndEncode(recoveryKey, s.fleetCfg.Server.PrivateKey)
	require.NoError(t, err)

	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, encryptedKey, "", new(true))
	require.NoError(t, err)

	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, host.ID, resp.HostID)
	require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeReadHostDiskEncryptionKey{}.ActivityName(),
		fmt.Sprintf(`{"host_display_name": "%s", "host_id": %d}`, host.DisplayName(), host.ID), 0)

	// Report a client error with no key. The error is recorded and the stored key is kept, because a failure on a host
	// that is still encrypted arrives when the admin most needs that key.
	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, "", "failed", nil)
	require.NoError(t, err)

	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue)

	detailsResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &detailsResp)
	require.True(t, detailsResp.Host.MDM.EncryptionKeyAvailable, "an error report must not take the key away")
	require.NotNil(t, detailsResp.Host.MDM.EncryptionKeyArchived)
	require.True(t, *detailsResp.Host.MDM.EncryptionKeyArchived)

	// A legacy key is encrypted to the WSTEP certificate and marked as such by the migration. It stays readable until the
	// hourly job moves it to the server private key.
	cert, _, _, err := s.fleetCfg.MDM.MicrosoftWSTEP()
	require.NoError(t, err)
	legacyRecoveryKey := "DDD-EEE-FFF"
	legacyKey, err := microsoft_mdm.Encrypt(legacyRecoveryKey, cert.Leaf)
	require.NoError(t, err)
	backdate := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_disk_encryption_keys SET base64_encrypted = ?, wstep_encrypted = 1, updated_at = ? WHERE host_id = ?`,
			legacyKey, backdate, host.ID)
		return err
	})
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, legacyRecoveryKey, resp.EncryptionKey.DecryptedValue)

	require.NoError(t, ReencryptWSTEPDiskEncryptionKeys(ctx, s.logger, s.ds, &s.fleetCfg))
	stored, err := s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	require.False(t, stored.WSTEPEncrypted)
	require.NotEqual(t, legacyKey, stored.Base64Encrypted)
	require.True(t, backdate.Equal(stored.UpdatedAt), "re-encrypting must not look like a new key")
	plain, err := servermdm.DecodeAndDecrypt(stored.Base64Encrypted, s.fleetCfg.Server.PrivateKey)
	require.NoError(t, err)
	require.Equal(t, legacyRecoveryKey, plain)
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, legacyRecoveryKey, resp.EncryptionKey.DecryptedValue)

	// A legacy key the configured certificate cannot read, as after the WSTEP pair was replaced: the job marks it, the
	// error no longer blames MDM, and the host is told to escrow a new key. The archive is cleared first, because the
	// read falls back to the newest readable archived key and would return the first one escrowed above.
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	deadKey, err := microsoft_mdm.Encrypt("GGG-HHH-III", selfSignedCertForKey(t, otherKey, 7))
	require.NoError(t, err)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		if _, err := q.ExecContext(ctx, `DELETE FROM host_disk_encryption_keys_archive WHERE host_id = ?`, host.ID); err != nil {
			return err
		}
		_, err := q.ExecContext(ctx, `UPDATE host_disk_encryption_keys SET base64_encrypted = ?, wstep_encrypted = 1, decryptable = 1 WHERE host_id = ?`,
			deadKey, host.ID)
		return err
	})
	require.NoError(t, ReencryptWSTEPDiskEncryptionKeys(ctx, s.logger, s.ds, &s.fleetCfg))
	stored, err = s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, new(false), stored.Decryptable)
	require.False(t, stored.WSTEPEncrypted, "an undecryptable key leaves the queue")
	// The host is only told a new key is coming while its fleet enforces Windows disk encryption.
	errResp := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusUnprocessableEntity)
	errBody, err := io.ReadAll(errResp.Body)
	require.NoError(t, err)
	errResp.Body.Close()
	require.Contains(t, string(errBody), "Couldn't decrypt the disk encryption key.")
	require.NotContains(t, string(errBody), "The host will escrow a new one")
	require.NotContains(t, string(errBody), "MDM has been turned off")
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_settings": { "enable_disk_encryption": true } } }`), http.StatusOK, &acResp)
	t.Cleanup(func() {
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_settings": { "enable_disk_encryption": false } } }`), http.StatusOK, &appConfigResponse{})
	})
	errResp = s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusUnprocessableEntity)
	errBody, err = io.ReadAll(errResp.Body)
	require.NoError(t, err)
	errResp.Body.Close()
	require.Contains(t, string(errBody), "The host will escrow a new one when it comes online.")

	// The host escrows a new key, stored the way the orbit endpoint stores it (TestHostDiskEncryptionKey covers the
	// endpoint itself), which clears the marker and makes the key readable again.
	newRecoveryKey := "JJJ-KKK-LLL"
	newKey, err := servermdm.EncryptAndEncode(newRecoveryKey, s.fleetCfg.Server.PrivateKey)
	require.NoError(t, err)
	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, newKey, "", new(true))
	require.NoError(t, err)
	stored, err = s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	require.False(t, stored.WSTEPEncrypted)
	require.Equal(t, new(true), stored.Decryptable)
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, newRecoveryKey, resp.EncryptionKey.DecryptedValue)
}

// TestBitLockerPINRequiresWindowsDiskEncryption covers the fleet write paths
// that previously persisted a PIN requirement without Windows disk encryption.
func (s *integrationMDMTestSuite) TestBitLockerPINRequiresWindowsDiskEncryption() {
	t := s.T()

	var createResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", &fleet.Team{Name: t.Name()}, http.StatusOK, &createResp)
	team := createResp.Team

	// PATCH the fleet: the PIN alone, with Windows disk encryption off
	res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{RequireBitLockerPIN: optjson.SetBool(true)},
	}, http.StatusUnprocessableEntity)
	assert.Contains(t, extractServerErrorText(res.Body), fleet.CantEnablePINRequiredIfDiskEncryptionEnabled)

	// creating a fleet from a spec with the PIN but no Windows disk encryption
	res = s.Do("POST", "/api/latest/fleet/spec/fleets", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: t.Name() + "-created",
		MDM:  fleet.TeamSpecMDM{RequireBitLockerPIN: optjson.SetBool(true)},
	}}}, http.StatusUnprocessableEntity)
	assert.Contains(t, extractServerErrorText(res.Body), fleet.CantEnablePINRequiredIfDiskEncryptionEnabled)

	// editing an existing fleet via spec, same story
	res = s.Do("POST", "/api/latest/fleet/spec/fleets", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM:  fleet.TeamSpecMDM{RequireBitLockerPIN: optjson.SetBool(true)},
	}}}, http.StatusUnprocessableEntity)
	assert.Contains(t, extractServerErrorText(res.Body), fleet.CantEnablePINRequiredIfDiskEncryptionEnabled)

	// the PIN together with Windows disk encryption is accepted
	s.Do("POST", "/api/latest/fleet/spec/fleets", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			RequireBitLockerPIN: optjson.SetBool(true),
			WindowsSettings:     fleet.WindowsSettings{EnableDiskEncryption: optjson.SetBool(true)},
		},
	}}}, http.StatusOK)
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
	assert.True(t, teamResp.Team.Config.MDM.WindowsSettings.RequireBitLockerPIN.Value)
	assert.True(t, teamResp.Team.Config.MDM.WindowsSettings.EnableDiskEncryption.Value)

	// turning encryption off while the PIN stays required is rejected, blaming
	// the encryption field
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			WindowsSettings: &fleet.TeamPayloadWindowsSettings{
				RequireBitLockerPIN: optjson.SetBool(true),
			},
			EnableDiskEncryption: optjson.SetBool(false),
		},
	}, http.StatusUnprocessableEntity)
	assert.Contains(t, extractServerErrorText(res.Body), fleet.CantDisableDiskEncryptionIfPINRequiredErrMsg)

	// the disk_encryption endpoint reports the same validation failure as a 422
	// rather than a server error
	res = s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{
		TeamID:          new(team.ID),
		WindowsSettings: &fleet.WindowsDiskEncryptionSettingsPayload{EnableDiskEncryption: new(false)},
	}, http.StatusUnprocessableEntity)
	assert.Contains(t, extractServerErrorText(res.Body), fleet.CantDisableDiskEncryptionIfPINRequiredErrMsg)
}

// TestBitLockerPINAliasConflict covers the deprecated windows_require_bitlocker_pin
// key disagreeing with its canonical windows_settings home across write paths.
func (s *integrationMDMTestSuite) TestBitLockerPINAliasConflict() {
	t := s.T()
	const conflictErr = "conflicts with mdm.windows_settings.require_bitlocker_pin"

	var createResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", &fleet.Team{Name: t.Name()}, http.StatusOK, &createResp)
	team := createResp.Team

	// global config: disagreeing values are rejected rather than silently
	// resolved to one of them
	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"mdm": {
	  "windows_require_bitlocker_pin": false,
	  "windows_settings": {"enable_disk_encryption": true, "require_bitlocker_pin": true}
	}}`), http.StatusUnprocessableEntity)
	assert.Contains(t, extractServerErrorText(res.Body), conflictErr)

	// agreeing values are accepted: a round-tripped document carries both
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"mdm": {
	  "windows_require_bitlocker_pin": true,
	  "windows_settings": {"enable_disk_encryption": true, "require_bitlocker_pin": true}
	}}`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.WindowsSettings.RequireBitLockerPIN.Value)
	assert.True(t, acResp.MDM.RequireBitLockerPIN.Value)

	// PATCH the fleet
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			RequireBitLockerPIN: optjson.SetBool(false),
			WindowsSettings: &fleet.TeamPayloadWindowsSettings{
				RequireBitLockerPIN: optjson.SetBool(true),
			},
		},
	}, http.StatusUnprocessableEntity)
	assert.Contains(t, extractServerErrorText(res.Body), conflictErr)

	// editing a fleet from a spec
	res = s.Do("POST", "/api/latest/fleet/spec/fleets", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			RequireBitLockerPIN: optjson.SetBool(false),
			WindowsSettings:     fleet.WindowsSettings{RequireBitLockerPIN: optjson.SetBool(true)},
		},
	}}}, http.StatusUnprocessableEntity)
	assert.Contains(t, extractServerErrorText(res.Body), conflictErr)

	// creating a fleet from a spec
	res = s.Do("POST", "/api/latest/fleet/spec/fleets", applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: t.Name() + "-created",
		MDM: fleet.TeamSpecMDM{
			RequireBitLockerPIN: optjson.SetBool(false),
			WindowsSettings:     fleet.WindowsSettings{RequireBitLockerPIN: optjson.SetBool(true)},
		},
	}}}, http.StatusUnprocessableEntity)
	assert.Contains(t, extractServerErrorText(res.Body), conflictErr)
}

func (s *integrationMDMTestSuite) TestDiskEncryptionEndpointPerPlatform() {
	t := s.T()

	// a single per-platform object changes only that platform
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{
		WindowsSettings: &fleet.WindowsDiskEncryptionSettingsPayload{EnableDiskEncryption: new(true)},
	}, http.StatusNoContent)
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.WindowsSettings.EnableDiskEncryption.Value)
	assert.False(t, acResp.MDM.MacOSSettings.EnableDiskEncryption.Value)
	assert.False(t, acResp.MDM.LinuxSettings.EnableEscrowDiskEncryptionKey.Value)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	// nil objects leave other platforms untouched
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{
		MacOSSettings: &fleet.MacOSDiskEncryptionSettingsPayload{EnableEscrowDiskEncryptionKey: new(true)},
	}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	assert.False(t, acResp.MDM.MacOSSettings.EnableDiskEncryption.Value)
	assert.True(t, acResp.MDM.WindowsSettings.EnableDiskEncryption.Value)

	// the deprecated flat toggle conflicts with a disagreeing per-platform value
	res := s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{
		EnableDiskEncryption: new(true),
		LinuxSettings:        &fleet.LinuxDiskEncryptionSettingsPayload{EnableEscrowDiskEncryptionKey: new(false)},
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "conflicts with per-platform disk encryption settings")

	// the flat toggle with agreeing per-platform values fans out to all four
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{
		EnableDiskEncryption: new(true),
		WindowsSettings:      &fleet.WindowsDiskEncryptionSettingsPayload{EnableDiskEncryption: new(true)},
	}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption.Value)
	assert.True(t, acResp.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	assert.True(t, acResp.MDM.LinuxSettings.EnableEscrowDiskEncryptionKey.Value)

	// the BitLocker PIN is settable via its canonical windows_settings home,
	// and the response mirrors the deprecated top-level key
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{
		WindowsSettings: &fleet.WindowsDiskEncryptionSettingsPayload{RequireBitLockerPIN: new(true)},
	}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.WindowsSettings.RequireBitLockerPIN.Value)
	assert.True(t, acResp.MDM.RequireBitLockerPIN.Value)

	// the deprecated PIN key conflicting with its canonical home is rejected
	res = s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{
		RequireBitLockerPIN: new(false),
		WindowsSettings:     &fleet.WindowsDiskEncryptionSettingsPayload{RequireBitLockerPIN: new(true)},
	}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "conflicts with windows_settings.require_bitlocker_pin")
}

// TestTeamPatchPerPlatformDiskEncryption covers the per-platform disk encryption
// keys on PATCH /fleets/{id}. The bodies are raw JSON on purpose: the endpoint
// silently drops keys its payload struct doesn't carry, which a typed request
// would hide.
func (s *integrationMDMTestSuite) TestTeamPatchPerPlatformDiskEncryption() {
	t := s.T()

	tm, err := s.ds.NewTeam(t.Context(), &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	path := fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID)

	patch := func(body string, wantStatus int) fleet.TeamMDM {
		var resp teamResponse
		s.DoJSON("PATCH", path, json.RawMessage(body), wantStatus, &resp)
		return resp.Team.Config.MDM
	}

	// the deprecated flat toggle still fans out to all four
	mdm := patch(`{"mdm":{"enable_disk_encryption":true}}`, http.StatusOK)
	require.True(t, mdm.EnableDiskEncryption)
	require.True(t, mdm.MacOSSettings.EnableDiskEncryption.Value)
	require.True(t, mdm.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	require.True(t, mdm.WindowsSettings.EnableDiskEncryption.Value)
	require.True(t, mdm.LinuxSettings.EnableEscrowDiskEncryptionKey.Value)
	s.assertConfigProfilesByIdentifier(new(tm.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// turning macOS enforcement off while keeping escrow on: the per-platform
	// keys reach their own settings and leave the other platforms alone
	mdm = patch(`{"mdm":{
		"apple_settings":  {"enable_disk_encryption": false, "enable_escrow_disk_encryption_key": true},
		"windows_settings": {"enable_disk_encryption": true},
		"linux_settings":  {"enable_escrow_disk_encryption_key": true}
	}}`, http.StatusOK)
	require.False(t, mdm.MacOSSettings.EnableDiskEncryption.Value)
	require.True(t, mdm.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	require.True(t, mdm.WindowsSettings.EnableDiskEncryption.Value)
	require.True(t, mdm.LinuxSettings.EnableEscrowDiskEncryptionKey.Value)
	// the flat toggle is the AND of the four, so it now reads false
	require.False(t, mdm.EnableDiskEncryption)
	// only macOS changed, so only macOS gets an activity
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "platform": "macos"}`, tm.ID, tm.Name), 0)
	// escrow is still on, so the FileVault profile stays in place
	s.assertConfigProfilesByIdentifier(new(tm.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// the deprecated macos_settings name works the same as apple_settings
	mdm = patch(`{"mdm":{"macos_settings":{"enable_escrow_disk_encryption_key": false}}}`, http.StatusOK)
	require.False(t, mdm.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	// both halves of the macOS pair are now off, so the profile goes away
	s.assertConfigProfilesByIdentifier(new(tm.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// both names in the same request is a conflict
	res := s.Do("PATCH", path, json.RawMessage(`{"mdm":{
		"macos_settings": {"enable_disk_encryption": true},
		"apple_settings": {"enable_disk_encryption": true}
	}}`), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `Specify only one of "macos_settings" or "apple_settings"`)

	// a per-platform key omitted from the body keeps its stored value, and an
	// explicit null does too
	mdm = patch(`{"mdm":{"linux_settings":{"enable_escrow_disk_encryption_key": null}}}`, http.StatusOK)
	require.True(t, mdm.LinuxSettings.EnableEscrowDiskEncryptionKey.Value)
	require.True(t, mdm.WindowsSettings.EnableDiskEncryption.Value)

	// changing the flat toggle and a per-platform key to disagreeing values is a
	// conflict. Windows is on here, so setting it off is a real change, as is
	// the flat toggle going on
	res = s.Do("PATCH", path, json.RawMessage(`{"mdm":{
		"enable_disk_encryption": true,
		"windows_settings": {"enable_disk_encryption": false}
	}}`), http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "conflicts with per-platform disk encryption settings")

	// the GET-modify-PATCH round trip is not a conflict: only the flat toggle
	// changes, and the per-platform values the request carries match what is
	// already stored, so the toggle wins and fans out
	mdm = patch(`{"mdm":{
		"enable_disk_encryption": true,
		"apple_settings": {"enable_disk_encryption": false, "enable_escrow_disk_encryption_key": false},
		"windows_settings": {"enable_disk_encryption": true},
		"linux_settings": {"enable_escrow_disk_encryption_key": true}
	}}`, http.StatusOK)
	require.True(t, mdm.EnableDiskEncryption)
	require.True(t, mdm.MacOSSettings.EnableDiskEncryption.Value)
	require.True(t, mdm.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)

	// precedence the other way round: everything is on, so only the
	// per-platform key changes and it wins over the carried flat toggle
	mdm = patch(`{"mdm":{
		"enable_disk_encryption": true,
		"apple_settings": {"enable_disk_encryption": false}
	}}`, http.StatusOK)
	require.False(t, mdm.MacOSSettings.EnableDiskEncryption.Value)
	require.True(t, mdm.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	require.False(t, mdm.EnableDiskEncryption)

	// with Windows enforcement off, turning the PIN on is rejected
	patch(`{"mdm":{"windows_settings":{"enable_disk_encryption": false}}}`, http.StatusOK)
	res = s.Do("PATCH", path, json.RawMessage(`{"mdm":{"windows_settings":{"require_bitlocker_pin": true}}}`),
		http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.CantEnablePINRequiredIfDiskEncryptionEnabled)

	// the PIN alongside Windows enforcement is accepted, and turning enforcement
	// off afterwards is rejected while the PIN is still required
	mdm = patch(`{"mdm":{"windows_settings":{"enable_disk_encryption": true, "require_bitlocker_pin": true}}}`, http.StatusOK)
	require.True(t, mdm.WindowsSettings.RequireBitLockerPIN.Value)
	require.True(t, mdm.RequireBitLockerPIN)
	res = s.Do("PATCH", path, json.RawMessage(`{"mdm":{"windows_settings":{"enable_disk_encryption": false}}}`),
		http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.CantDisableDiskEncryptionIfPINRequiredErrMsg)

	// a request that changes nothing emits no activity
	before := s.lastActivityMatches("", "", 0)
	patch(`{"mdm":{"windows_settings":{"enable_disk_encryption": true}}}`, http.StatusOK)
	require.Equal(t, before, s.lastActivityMatches("", "", 0))

	// the GET response carries every per-platform key, so a GET-modify-PATCH
	// round trip can see them. The macOS object comes back under its new
	// apple_settings name only, though requests accept either name.
	res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", tm.ID), nil, http.StatusOK)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var raw struct {
		Fleet struct {
			MDM struct {
				EnableDiskEncryption *bool `json:"enable_disk_encryption"`
				AppleSettings        *struct {
					EnableDiskEncryption          *bool `json:"enable_disk_encryption"`
					EnableEscrowDiskEncryptionKey *bool `json:"enable_escrow_disk_encryption_key"`
				} `json:"apple_settings"`
				MacOSSettings *struct {
					EnableDiskEncryption *bool `json:"enable_disk_encryption"`
				} `json:"macos_settings"`
				WindowsSettings *struct {
					EnableDiskEncryption *bool `json:"enable_disk_encryption"`
					RequireBitLockerPIN  *bool `json:"require_bitlocker_pin"`
				} `json:"windows_settings"`
				LinuxSettings *struct {
					EnableEscrowDiskEncryptionKey *bool `json:"enable_escrow_disk_encryption_key"`
				} `json:"linux_settings"`
			} `json:"mdm"`
		} `json:"fleet"`
	}
	require.NoError(t, json.Unmarshal(body, &raw))
	gotMDM := raw.Fleet.MDM
	require.NotNil(t, gotMDM.EnableDiskEncryption)
	require.NotNil(t, gotMDM.AppleSettings)
	require.NotNil(t, gotMDM.AppleSettings.EnableDiskEncryption)
	require.NotNil(t, gotMDM.AppleSettings.EnableEscrowDiskEncryptionKey)
	require.NotNil(t, gotMDM.WindowsSettings)
	require.NotNil(t, gotMDM.WindowsSettings.EnableDiskEncryption)
	require.NotNil(t, gotMDM.WindowsSettings.RequireBitLockerPIN)
	require.NotNil(t, gotMDM.LinuxSettings)
	require.NotNil(t, gotMDM.LinuxSettings.EnableEscrowDiskEncryptionKey)
}

func (s *integrationMDMTestSuite) TestAppConfigMDMAppleDiskEncryption() {
	t := s.T()

	// set the macos disk encryption field
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	enabledDiskActID := s.lastActivityMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		`{"fleet_id": null, "fleet_name": null, "platform": "linux"}`, 0)

	// a legacy flat-toggle write emits one activity per platform
	var listActivities listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities,
		"order_key", "id", "order_direction", "desc", "per_page", "3")
	require.Len(t, listActivities.Activities, 3)
	gotPlatforms := make([]string, 0, 3)
	for _, act := range listActivities.Activities {
		require.Equal(t, fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(), act.Type)
		var details struct {
			Platform string `json:"platform"`
		}
		require.NoError(t, json.Unmarshal(*act.Details, &details))
		gotPlatforms = append(gotPlatforms, details.Platform)
	}
	require.ElementsMatch(t, []string{"macos", "windows", "linux"}, gotPlatforms)

	// will have generated the macos config profile
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// patch without specifying the macos disk encryption and an unrelated field,
	// should not alter it
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
                     "mdm": { "macos_settings": {"custom_settings": [{"path": "a"}]} }
		}`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "a"}}, acResp.MDM.MacOSSettings.CustomSettings)
	s.lastActivityMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		``, enabledDiskActID)

	// patch with false, would reset it but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
				"mdm": { "enable_disk_encryption": false }
		  }`), http.StatusOK, &acResp, "dry_run", "true")
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "a"}}, acResp.MDM.MacOSSettings.CustomSettings)
	s.lastActivityMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		``, enabledDiskActID)

	// patch with false, resets it
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	    "mdm": { "enable_disk_encryption": false, "macos_settings": { "custom_settings": [{"path":"b"}] } }
		  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "b"}}, acResp.MDM.MacOSSettings.CustomSettings)
	s.lastActivityMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		`{"fleet_id": null, "fleet_name": null, "platform": "linux"}`, 0)

	// will have deleted the macos config profile
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// use the MDM disk encryption endpoint to set it to true
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{EnableDiskEncryption: new(true)}, http.StatusNoContent)
	enabledDiskActID = s.lastActivityMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		`{"fleet_id": null, "fleet_name": null, "platform": "linux"}`, 0)

	// will have created the macos config profile
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "b"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// call update endpoint with no changes
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{}, http.StatusNoContent)
	s.lastActivityMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		``, enabledDiskActID)

	// the macos config profile still exists
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "b"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// mdm/apple/settings works for windows as well as it's being used by
	// clients (UI) this way
	appConf, err := s.ds.AppConfig(context.Background())
	s.Require().NoError(err)
	appConf.MDM.EnabledAndConfigured = false
	appConf.MDM.WindowsEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	s.Require().NoError(err)
	defer func() {
		appConf, err := s.ds.AppConfig(context.Background())
		s.Require().NoError(err)
		appConf.MDM.EnabledAndConfigured = true
		appConf.MDM.WindowsEnabledAndConfigured = true
		err = s.ds.SaveAppConfig(context.Background(), appConf)
		s.Require().NoError(err)
	}()

	// flip and verify the value
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{EnableDiskEncryption: new(false)}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{EnableDiskEncryption: new(true)}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
}

func (s *integrationMDMTestSuite) TestMDMAppleDiskEncryptionAggregate() {
	t := s.T()
	ctx := context.Background()

	// no hosts with any disk encryption status's
	expectedNoTeamDiskEncryptionSummary := fleet.MDMDiskEncryptionSummary{}
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary := fleet.MDMProfilesSummary{}
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// 10 new hosts
	var hosts []*fleet.Host
	for range 10 {
		h, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
		hosts = append(hosts, h)
	}

	// no team tests ====

	// a FileVault profile is only delivered while disk encryption is on;
	// with it off, a delivered one reports as awaiting removal
	ac, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	prevMacOSSettings := ac.MDM.MacOSSettings
	ac.MDM.MacOSSettings.EnableDiskEncryption = optjson.SetBool(true)
	ac.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey = optjson.SetBool(true)
	require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
	t.Cleanup(func() {
		ac, err := s.ds.AppConfig(context.Background())
		require.NoError(t, err)
		ac.MDM.MacOSSettings = prevMacOSSettings
		require.NoError(t, s.ds.SaveAppConfig(context.Background(), ac))
	})

	// new filevault profile with no team
	prof, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTest("filevault-1", mobileconfig.FleetFileVaultPayloadIdentifier), new(uint(0)))
	require.NoError(t, err)

	// generates a disk encryption aggregate value based on the arguments passed in
	generateAggregateValue := func(
		hosts []*fleet.Host,
		operationType fleet.MDMOperationType,
		status *fleet.MDMDeliveryStatus,
		decryptable bool,
	) {
		for _, host := range hosts {
			hostCmdUUID := uuid.New().String()
			err := s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
				{
					ProfileUUID:       prof.ProfileUUID,
					ProfileIdentifier: prof.Identifier,
					HostUUID:          host.UUID,
					CommandUUID:       hostCmdUUID,
					OperationType:     operationType,
					Status:            status,
					Checksum:          []byte("csum"),
					Scope:             fleet.PayloadScopeSystem,
				},
			})
			require.NoError(t, err)
			oneMinuteAfterThreshold := time.Now().Add(+1 * time.Minute)
			_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, "test-key", "", nil)
			require.NoError(t, err)
			err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, decryptable, oneMinuteAfterThreshold)
			require.NoError(t, err)
		}
	}

	// hosts 1,2 have disk encryption "applied" status
	generateAggregateValue(hosts[0:2], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, true) // nolint:nilaway // hosts is populated with 10 hosts above
	expectedNoTeamDiskEncryptionSummary.Verifying.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Verifying = 2
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// hosts 3,4 have disk encryption "action required" status
	generateAggregateValue(hosts[2:4], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, false)
	expectedNoTeamDiskEncryptionSummary.ActionRequired.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Pending = 2
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// hosts 5,6 have disk encryption "enforcing" status

	// host profiles status are `pending`
	generateAggregateValue(hosts[4:6], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryPending, true)
	expectedNoTeamDiskEncryptionSummary.Enforcing.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Pending = 4
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// host profiles status dont exist
	generateAggregateValue(hosts[4:6], fleet.MDMOperationTypeInstall, nil, true)
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)               // no change
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary) // no change

	// host profile is applied but decryptable key does not exist
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(
			context.Background(),
			"UPDATE host_disk_encryption_keys SET decryptable = NULL WHERE host_id IN (?, ?)",
			hosts[5].ID,
			hosts[6].ID,
		)
		require.NoError(t, err)
		return err
	})
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)               // no change
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary) // no change

	// hosts 7,8 have disk encryption "failed" status
	generateAggregateValue(hosts[6:8], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryFailed, true)
	expectedNoTeamDiskEncryptionSummary.Failed.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Failed = 2
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// hosts 9,10 have disk encryption "removing enforcement" status
	generateAggregateValue(hosts[8:10], fleet.MDMOperationTypeRemove, &fleet.MDMDeliveryPending, true)
	expectedNoTeamDiskEncryptionSummary.RemovingEnforcement.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Pending = 6
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// team tests ====

	// host 1,2 added to team 1
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team-1", Config: fleet.TeamConfig{MDM: fleet.TeamMDM{MacOSSettings: fleet.MacOSSettings{
		EnableDiskEncryption:          optjson.SetBool(true),
		EnableEscrowDiskEncryptionKey: optjson.SetBool(true),
	}}}})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{hosts[0].ID, hosts[1].ID}))
	require.NoError(t, err)

	// new filevault profile for team 1
	prof, err = fleet.NewMDMAppleConfigProfile(mobileconfigForTest("filevault-1", mobileconfig.FleetFileVaultPayloadIdentifier), new(uint(1)))
	require.NoError(t, err)
	prof.TeamID = &tm.ID
	require.NoError(t, err)

	// filtering by the "fleet_id" query param
	generateAggregateValue(hosts[0:2], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, true)

	var expectedTeamDiskEncryptionSummary fleet.MDMDiskEncryptionSummary
	expectedTeamDiskEncryptionSummary.Verifying.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, &tm.ID, expectedTeamDiskEncryptionSummary, true)

	expectedNoTeamDiskEncryptionSummary.Verifying.MacOS = 0 // now 0 because hosts 1,2 were added to team 1
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)

	expectedTeamProfilesSummary := fleet.MDMProfilesSummary{Verifying: 2}
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamProfilesSummary, &expectedTeamProfilesSummary)

	expectedNoTeamProfilesSummary = fleet.MDMProfilesSummary{
		Verifying: 0, // now 0 because hosts 1,2 were added to team 1
		Pending:   6,
		Failed:    2,
	}
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// verified status for host 1
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, s.ds, hosts[0], map[string]*fleet.HostMacOSProfile{prof.Identifier: {Identifier: prof.Identifier, DisplayName: prof.Name, InstallDate: time.Now()}}))
	// TODO: Why is there no change to the verification status of host 1 reflected in the summaries?
	s.checkMDMDiskEncryptionSummaries(t, &tm.ID, expectedTeamDiskEncryptionSummary, true)              // no change
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)               // no change
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamProfilesSummary, &expectedTeamProfilesSummary)  // no change
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary) // no change
}

func (s *integrationMDMTestSuite) TestTeamsMDMAppleDiskEncryption() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/fleets", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	// no macos config profile yet
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// apply with disk encryption
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
	lastDiskActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "platform": "linux"}`, team.ID, teamName), 0)

	// macos config profile created
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// retrieving the team returns the disk encryption setting
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	// apply with invalid disk encryption value should fail
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]any{"enable_disk_encryption": 123},
		},
	}}}
	res := s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `invalid value type at 'macos_settings.enable_disk_encryption': expected bool but got float64`)

	// a non-string profile description is rejected rather than dropped
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]any{"custom_settings": []any{
				map[string]any{"path": "a.mobileconfig", "description": 123},
			}},
		},
	}}}
	res = s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `invalid value type at 'macos_settings.custom_settings.description': expected string but got float64`)

	// apply an empty set of batch profiles to the team
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: nil},
		http.StatusUnprocessableEntity, "fleet_id", fmt.Sprint(team.ID), "fleet_name", team.Name)

	// the configuration profile is still there
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// apply without disk encryption settings specified and unrelated field,
	// should not replace existing disk encryption
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]any{
				"custom_settings": []map[string]any{
					{"path": "a"},
				},
			},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)
	require.Equal(t, []fleet.MDMProfileSpec{{Path: "a"}}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		``, lastDiskActID)

	// apply with false would clear the existing setting, but dry-run
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		``, lastDiskActID)

	// apply with false clears the existing setting
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]any{"enable_disk_encryption": false, "enable_escrow_disk_encryption_key": false},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.False(t, teamResp.Team.Config.MDM.EnableDiskEncryption)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "platform": "macos"}`, team.ID, teamName), 0)

	// macos config profile deleted
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// modify team's disk encryption via ModifyTeam endpoint
	var modResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}, http.StatusOK, &modResp)
	require.True(t, modResp.Team.Config.MDM.EnableDiskEncryption)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "platform": "macos"}`, team.ID, teamName), 0)

	// macos config profile created
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// modify team's disk encryption and description via ModifyTeam endpoint
	modResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), fleet.TeamPayload{
		Description: new("foobar"),
		MDM: &fleet.TeamPayloadMDM{
			EnableDiskEncryption: optjson.SetBool(false),
		},
	}, http.StatusOK, &modResp)
	require.False(t, modResp.Team.Config.MDM.EnableDiskEncryption)
	require.Equal(t, "foobar", modResp.Team.Description)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "platform": "linux"}`, team.ID, teamName), 0)

	// macos config profile deleted
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// use the MDM settings endpoint to set it to true
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{TeamID: new(team.ID), EnableDiskEncryption: new(true)}, http.StatusNoContent)
	lastDiskActID = s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "platform": "linux"}`, team.ID, teamName), 0)

	// macos config profile created
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	// use the MDM settings endpoint with no changes
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{TeamID: new(team.ID)}, http.StatusNoContent)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		``, lastDiskActID)

	// macos config profile still exists
	s.assertConfigProfilesByIdentifier(new(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	// use the MDM settings endpoint with an unknown team id
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{TeamID: new(uint(9999)), EnableDiskEncryption: new(true)}, http.StatusNotFound)

	// mdm/apple/settings works for windows as well as it's being used by
	// clients (UI) this way
	appConf, err := s.ds.AppConfig(context.Background())
	s.Require().NoError(err)
	appConf.MDM.EnabledAndConfigured = false
	appConf.MDM.WindowsEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	s.Require().NoError(err)
	defer func() {
		appConf, err := s.ds.AppConfig(context.Background())
		s.Require().NoError(err)
		appConf.MDM.EnabledAndConfigured = true
		appConf.MDM.WindowsEnabledAndConfigured = true
		err = s.ds.SaveAppConfig(context.Background(), appConf)
		s.Require().NoError(err)
	}()

	// flip and verify the value
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{TeamID: new(team.ID), EnableDiskEncryption: new(false)}, http.StatusNoContent)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.False(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{TeamID: new(team.ID), EnableDiskEncryption: new(true)}, http.StatusNoContent)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	// changing a single per-platform setting via a team spec generates one
	// activity for that platform only
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			LinuxSettings: fleet.LinuxSettings{
				EnableEscrowDiskEncryptionKey: optjson.SetBool(false),
			},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)
	linuxActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "platform": "linux"}`, team.ID, team.Name), 0)

	// a PIN-only change counts as a windows settings change
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{TeamID: new(team.ID), RequireBitLockerPIN: new(true)}, http.StatusNoContent)
	winActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedDiskEncryptionSettings{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "platform": "windows"}`, team.ID, team.Name), 0)
	require.Greater(t, winActID, linuxActID)
}

func (s *integrationMDMTestSuite) TestBitLockerEnforcementNotifications() {
	t := s.T()
	ctx := context.Background()
	windowsHost := createOrbitEnrolledHost(t, "windows", t.Name(), s.ds)

	checkNotification := func(want bool) {
		resp := fleet.OrbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *windowsHost.OrbitNodeKey)), http.StatusOK, &resp)
		require.Equal(t, want, resp.Notifications.EnforceBitLockerEncryption)
	}

	// notification is false by default
	checkNotification(false)

	// enroll the host into Fleet MDM
	_, _ = s.ds.MDMWindowsDeleteEnrolledDeviceOnReenrollment(t.Context(), enrollRequestHWDevID)
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *windowsHost.OrbitNodeKey)
	require.NoError(t, err)
	requestBytes, err := s.newSecurityTokenMsg(encodedBinToken, true, false)
	require.NoError(t, err)
	s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	// simulate osquery checking in and updating this info
	// TODO: should we automatically fill these fields on MDM enrollment?
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), windowsHost.ID, false, true, "https://example.com", true, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))

	// notification is still false
	checkNotification(false)

	// configure disk encryption for the global team. NOTE: since the
	// per-platform split, macos_settings.enable_disk_encryption is the macOS
	// enforcement setting, no longer an alias of the flat toggle; the orbit
	// notification gate reads the flat toggle (the AND of the per-platform
	// settings) until the per-platform behavior split ships (#51259), so this
	// test uses the flat toggle.
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "enable_disk_encryption": true } }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// host still doesn't get the notification because we don't have disk
	// encryption information yet.
	checkNotification(false)

	// host has disk encryption off, gets the notification
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), windowsHost.ID, false, nil))
	checkNotification(true)

	// host has disk encryption on, we don't have disk encryption info. Gets the notification
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), windowsHost.ID, true, nil))
	checkNotification(true)

	// host has disk encryption on, we don't know if the key is decriptable. Gets the notification
	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, windowsHost, "test-key", "", nil)
	require.NoError(t, err)
	checkNotification(true)

	// host has disk encryption on, the key is not decryptable by fleet. Gets the notification
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{windowsHost.ID}, false, time.Now())
	require.NoError(t, err)
	checkNotification(true)

	// host has disk encryption on, the disk was encrypted by fleet. Doesn't get the notification
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{windowsHost.ID}, true, time.Now())
	require.NoError(t, err)
	checkNotification(false)

	// create a new team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)
	// add the host to the team
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&tm.ID, []uint{windowsHost.ID}))
	require.NoError(t, err)

	// notification is false now since the team doesn't have disk encryption enabled
	checkNotification(false)

	// enable disk encryption on the team
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: tm.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)

	// host gets the notification
	checkNotification(true)

	// host has disk encryption off, gets the notification
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), windowsHost.ID, false, nil))
	checkNotification(true)

	// host has disk encryption on, we don't have disk encryption info. Gets the notification
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), windowsHost.ID, true, nil))
	checkNotification(true)

	// host has disk encryption on, we don't know if the key is decriptable. Gets the notification
	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, windowsHost, "test-key", "", nil)
	require.NoError(t, err)
	checkNotification(true)

	// host has disk encryption on, the key is not decryptable by fleet. Gets the notification
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{windowsHost.ID}, false, time.Now())
	require.NoError(t, err)
	checkNotification(true)

	// host has disk encryption on, the disk was encrypted by fleet. Doesn't get the notification
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{windowsHost.ID}, true, time.Now())
	require.NoError(t, err)
	checkNotification(false)
}

func (s *integrationMDMTestSuite) TestHostDiskEncryptionKey() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "windows", "h1", s.ds)

	// turn on disk encryption for the global team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "enable_disk_encryption": true } }`), http.StatusOK, &acResp)
	assert.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value)

	// try to call the endpoint while the host is not MDM-enrolled
	res := s.Do("POST", "/api/fleet/orbit/disk_encryption_key", fleet.OrbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey:  *host.OrbitNodeKey,
		EncryptionKey: []byte("WILL-FAIL"),
	}, http.StatusBadRequest)
	msg := extractServerErrorText(res.Body)
	require.Contains(t, msg, "host is not enrolled with fleet")

	// enroll it in fleet
	mdmDevice := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *host.OrbitNodeKey)
	err := mdmDevice.Enroll()
	require.NoError(t, err)
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)

	// set its encryption key
	s.Do("POST", "/api/fleet/orbit/disk_encryption_key", fleet.OrbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey:  *host.OrbitNodeKey,
		EncryptionKey: []byte("ABC"),
	}, http.StatusNoContent)

	hdek, err := s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, hdek.Decryptable)
	require.True(t, *hdek.Decryptable)

	// check a new activity entry is created ...
	activities := s.listActivities()
	escrowKeyActivity := fleet.ActivityTypeEscrowedDiskEncryptionKey{}
	var seenEscrowKeyActivityID uint
	for _, activity := range activities {
		if activity.Type == escrowKeyActivity.ActivityName() {
			err := json.Unmarshal(*activity.Details, &escrowKeyActivity)
			require.NoError(t, err)
			require.True(t, activity.FleetInitiated)

			seenEscrowKeyActivityID = activity.ID
		}
	}
	require.NotZero(t, seenEscrowKeyActivityID)
	require.Equal(t, escrowKeyActivity.HostID, host.ID)
	require.Equal(t, escrowKeyActivity.HostDisplayName, host.DisplayName())

	// mark it as non-server
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.DiskEncryptionEnabled) // the disk encryption status of the host is not set by the orbit request
	require.NotNil(t, hostResp.Host.MDM.OSSettings)
	require.NotNil(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *hostResp.Host.MDM.OSSettings.DiskEncryption.Status) // still pending because disk encryption status is not set
	require.Empty(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// the key is encrypted with the server private key, like Linux keys, never to the WSTEP certificate.
	require.False(t, hdek.WSTEPEncrypted)
	decrypted, err := servermdm.DecodeAndDecrypt(hdek.Base64Encrypted, s.fleetCfg.Server.PrivateKey)
	require.NoError(t, err)
	require.Equal(t, "ABC", decrypted)

	// set it with a client error
	s.Do("POST", "/api/fleet/orbit/disk_encryption_key", fleet.OrbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey: *host.OrbitNodeKey,
		ClientError:  "fail",
	}, http.StatusNoContent)

	// The error is recorded, but the stored key and its decryptable flag are kept.
	hdek, err = s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, hdek.Decryptable)
	require.True(t, *hdek.Decryptable)
	require.NotEmpty(t, hdek.Base64Encrypted)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.DiskEncryptionEnabled) // the disk encryption status of the host is not set by the orbit request
	require.NotNil(t, hostResp.Host.MDM.OSSettings)
	require.NotNil(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionFailed, *hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "fail", hostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// set a different key
	s.Do("POST", "/api/fleet/orbit/disk_encryption_key", fleet.OrbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey:  *host.OrbitNodeKey,
		EncryptionKey: []byte("DEF"),
	}, http.StatusNoContent)

	hdek, err = s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, hdek.Decryptable)
	require.True(t, *hdek.Decryptable)

	// escrowing a new encryption key should create a new acitivy entry
	activities = s.listActivities()
	escrowKeyActivity = fleet.ActivityTypeEscrowedDiskEncryptionKey{}
	newActivityCount := 0
	for _, activity := range activities {
		if activity.Type == escrowKeyActivity.ActivityName() && activity.ID > seenEscrowKeyActivityID {
			newActivityCount++
			err := json.Unmarshal(*activity.Details, &escrowKeyActivity)
			require.NoError(t, err)
			require.True(t, activity.FleetInitiated)

			seenEscrowKeyActivityID = activity.ID
		}
	}
	require.Equal(t, 1, newActivityCount, "Expected exactly one new escrow activity")
	require.Equal(t, escrowKeyActivity.HostID, host.ID)
	require.Equal(t, escrowKeyActivity.HostDisplayName, host.DisplayName())

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.DiskEncryptionEnabled) // the disk encryption status of the host is not set by the orbit request
	require.NotNil(t, hostResp.Host.MDM.OSSettings)
	require.NotNil(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *hostResp.Host.MDM.OSSettings.DiskEncryption.Status) // still pending because disk encryption status is not set
	require.Empty(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	decrypted, err = servermdm.DecodeAndDecrypt(hdek.Base64Encrypted, s.fleetCfg.Server.PrivateKey)
	require.NoError(t, err)
	require.Equal(t, "DEF", decrypted)

	// report host disks as encrypted
	err = s.ds.SetOrUpdateHostDisksEncryption(ctx, host.ID, true, nil)
	require.NoError(t, err)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.True(t, *hostResp.Host.DiskEncryptionEnabled)
	require.NotNil(t, hostResp.Host.MDM.OSSettings)
	require.NotNil(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionVerified, *hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Empty(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// Test that disk encryption keys are not archived when disk encryption is disabled
	// This can happen when another mdm provider enforces disk encryption and when
	// fleet runs it will attempt to escrow the key.
	t.Run("key not archived when disk encryption disabled", func(t *testing.T) {
		activities := s.listActivities()
		escrowCountBefore := 0
		for _, activity := range activities {
			if activity.Type == "escrowed_disk_encryption_key" {
				escrowCountBefore++
			}
		}

		// Disable disk encryption globally
		acResp := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "enable_disk_encryption": false } }`), http.StatusOK, &acResp)
		assert.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value)

		s.Do("POST", "/api/fleet/orbit/disk_encryption_key", fleet.OrbitPostDiskEncryptionKeyRequest{
			OrbitNodeKey:  *host.OrbitNodeKey,
			EncryptionKey: []byte("NEW-KEY"),
		}, http.StatusNoContent)

		// Key shouldn't be archived
		hdekAfter, err := s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
		require.NoError(t, err)
		decryptedAfter, err := servermdm.DecodeAndDecrypt(hdekAfter.Base64Encrypted, s.fleetCfg.Server.PrivateKey)
		require.NoError(t, err)
		require.Equal(t, "DEF", decryptedAfter)

		// No activity
		activities = s.listActivities()
		escrowCountAfter := 0
		for _, activity := range activities {
			if activity.Type == "escrowed_disk_encryption_key" {
				escrowCountAfter++
			}
		}
		require.Equal(t, escrowCountBefore, escrowCountAfter, "No new escrow activity should be created when disk encryption is disabled")

		// Enable disk encryption
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "enable_disk_encryption": true } }`), http.StatusOK, &acResp)
		assert.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value)

		s.Do("POST", "/api/fleet/orbit/disk_encryption_key", fleet.OrbitPostDiskEncryptionKeyRequest{
			OrbitNodeKey:  *host.OrbitNodeKey,
			EncryptionKey: []byte("NEW-KEY-2"),
		}, http.StatusNoContent)

		// Key should be archived
		hdekFinal, err := s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
		require.NoError(t, err)
		decryptedFinal, err := servermdm.DecodeAndDecrypt(hdekFinal.Base64Encrypted, s.fleetCfg.Server.PrivateKey)
		require.NoError(t, err)
		require.Equal(t, "NEW-KEY-2", decryptedFinal)

		// New escrow activity
		activities = s.listActivities()
		newEscrowCount := 0
		escrowKeyActivity = fleet.ActivityTypeEscrowedDiskEncryptionKey{}
		for _, activity := range activities {
			if activity.Type == escrowKeyActivity.ActivityName() && activity.ID > seenEscrowKeyActivityID {
				newEscrowCount++
				err := json.Unmarshal(*activity.Details, &escrowKeyActivity)
				require.NoError(t, err)
				require.True(t, activity.FleetInitiated)

				seenEscrowKeyActivityID = activity.ID
			}
		}
		require.Equal(t, 1, newEscrowCount)
		require.NotZero(t, seenEscrowKeyActivityID)
		require.Equal(t, escrowKeyActivity.HostID, host.ID)
		require.Equal(t, escrowKeyActivity.HostDisplayName, host.DisplayName())
	})
}

func (s *integrationMDMTestSuite) checkMDMDiskEncryptionSummaries(t *testing.T, teamID *uint, expectedSummary fleet.MDMDiskEncryptionSummary, checkFileVaultSummary bool) {
	var queryParams []string
	if teamID != nil {
		queryParams = append(queryParams, "fleet_id", fmt.Sprintf("%d", *teamID))
	}

	if checkFileVaultSummary {
		var fileVault getMDMAppleFileVaultSummaryResponse
		s.DoJSON("GET", "/api/v1/fleet/mdm/apple/filevault/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &fileVault, queryParams...)
		require.Equal(t, expectedSummary.Failed.MacOS, fileVault.Failed)
		require.Equal(t, expectedSummary.Enforcing.MacOS, fileVault.Enforcing)
		require.Equal(t, expectedSummary.ActionRequired.MacOS, fileVault.ActionRequired)
		require.Equal(t, expectedSummary.Verifying.MacOS, fileVault.Verifying)
		require.Equal(t, expectedSummary.Verified.MacOS, fileVault.Verified)
		require.Equal(t, expectedSummary.RemovingEnforcement.MacOS, fileVault.RemovingEnforcement)
	}

	var combined getMDMDiskEncryptionSummaryResponse
	s.DoJSON("GET", "/api/v1/fleet/disk_encryption", getMDMProfilesSummaryRequest{}, http.StatusOK, &combined, queryParams...)
	require.Equal(t, expectedSummary.Failed, combined.Failed)
	require.Equal(t, expectedSummary.Enforcing, combined.Enforcing)
	require.Equal(t, expectedSummary.ActionRequired, combined.ActionRequired)
	require.Equal(t, expectedSummary.Verifying, combined.Verifying)
	require.Equal(t, expectedSummary.Verified, combined.Verified)
	require.Equal(t, expectedSummary.RemovingEnforcement, combined.RemovingEnforcement)
}

func (s *integrationMDMTestSuite) TestMDMDiskEncryptionIssue16636() {
	// see https://github.com/fleetdm/fleet/issues/16636

	t := s.T()

	// send an empty patch object to ensure it's ok if not provided
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{}`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// set it to true
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// call PATCH with a null value, leaves it untouched
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": null }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// send an empty patch object to ensure it doesn't alter the value
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{}`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// setting it to false works
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": false }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// send a null value again
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": null }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// confirm via GET
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)
}

func (s *integrationMDMTestSuite) TestIsServerBitlockerStatus() {
	t := s.T()
	ctx := context.Background()

	// create a server host that is not enrolled in MDM
	host := createOrbitEnrolledHost(t, "windows", "server-host", s.ds)
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, true, false, "", false, "", "", fleet.PersonalEnrollmentTypeNone))

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	var hr getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hr)
	require.Nil(t, hr.Host.MDM.OSSettings.DiskEncryption.Status)

	// create a non-server host that is enrolled in MDM
	host2 := createOrbitEnrolledHost(t, "windows", "non-server-host", s.ds)
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host2.ID, false, true, "http://example.com", false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))

	hr = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host2.ID), nil, http.StatusOK, &hr)
	require.NotNil(t, hr.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *hr.Host.MDM.OSSettings.DiskEncryption.Status)
}

// Test for #24265
func (s *integrationMDMTestSuite) TestNonMDWindowsHostsIgnoredInDiskEncryptionStats() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	// get the All hosts label ID
	ls, err := s.ds.LabelIDsByName(ctx, []string{"All Hosts"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, ls, 1)
	allHostsLblID := ls["All Hosts"]

	// create a couple Windows non-MDM-enrolled hosts
	winHost1 := createOrbitEnrolledHost(t, "windows", "h1", s.ds)
	winHost2 := createOrbitEnrolledHost(t, "windows", "h2", s.ds)
	err = s.ds.SetOrUpdateMDMData(ctx, winHost1.ID, false, false, "", false, "", "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)
	err = s.ds.SetOrUpdateMDMData(ctx, winHost2.ID, false, false, "", false, "", "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)
	err = s.ds.AddLabelsToHost(ctx, winHost1.ID, []uint{allHostsLblID})
	require.NoError(t, err)
	err = s.ds.AddLabelsToHost(ctx, winHost2.ID, []uint{allHostsLblID})
	require.NoError(t, err)

	// first a quick stats check without disk encryption
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{EnableDiskEncryption: new(false)}, http.StatusNoContent)
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{}, nil)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{}, false)

	// enable disk encryption
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{EnableDiskEncryption: new(true)}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// all profiles counts are expected to be 0 (no host is MDM-enrolled in Fleet)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{}, nil)

	// filter hosts with any OS Settings status, should have none
	statuses := []fleet.OSSettingsStatus{
		fleet.OSSettingsVerified, fleet.OSSettingsVerifying,
		fleet.OSSettingsPending, fleet.OSSettingsFailed,
	}
	for _, status := range statuses {
		s.checkListHostsFilter(t, allHostsLblID, "os_settings", string(status))
	}

	// all disk encryption counts are expected to be 0 (no host is MDM-enrolled in Fleet)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{}, false)

	// filter hosts with any Disk Encryption status, should have none
	diskStatuses := []fleet.DiskEncryptionStatus{
		fleet.DiskEncryptionVerified, fleet.DiskEncryptionVerifying,
		fleet.DiskEncryptionActionRequired, fleet.DiskEncryptionEnforcing,
		fleet.DiskEncryptionFailed, fleet.DiskEncryptionRemovingEnforcement,
	}
	for _, status := range diskStatuses {
		s.checkListHostsFilter(t, allHostsLblID, "os_settings_disk_encryption", string(status))
	}

	// enroll a Windows host in Fleet MDM
	enrollWindowsHostInMDMViaOrbit(t, winHost1, s.ds, s.server.URL)

	// stats should now count this host
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{Pending: 1}, nil)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{Enforcing: fleet.MDMPlatformsCounts{Windows: 1}}, false)

	// filters should return this host
	s.checkListHostsFilter(t, allHostsLblID, "os_settings", string(fleet.OSSettingsPending), winHost1.ID)
	s.checkListHostsFilter(t, allHostsLblID, "os_settings_disk_encryption", string(fleet.DiskEncryptionEnforcing), winHost1.ID)

	// enroll the other Windows host in a third-party MDM
	err = s.ds.SetOrUpdateMDMData(ctx, winHost2.ID, false, true, "https://simplemdm.com", true, fleet.WellKnownMDMSimpleMDM, "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)

	// stats should NOT count winHost2 (not in Fleet MDM)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{Pending: 1}, nil)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{Enforcing: fleet.MDMPlatformsCounts{Windows: 1}}, false)

	// filters should NOT return this host
	s.checkListHostsFilter(t, allHostsLblID, "os_settings", string(fleet.OSSettingsPending), winHost1.ID)
	s.checkListHostsFilter(t, allHostsLblID, "os_settings_disk_encryption", string(fleet.DiskEncryptionEnforcing), winHost1.ID)
}

// Test for #24266
func (s *integrationMDMTestSuite) TestLinuxHostsIgnoredInOSSettingsStats() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	// get the All hosts label ID
	ls, err := s.ds.LabelIDsByName(ctx, []string{"All Hosts"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, ls, 1)
	allHostsLblID := ls["All Hosts"]

	// create a couple Linux hosts that would support disk encryption if it was enabled
	linuxHost1 := createOrbitEnrolledHost(t, "ubuntu", "h1", s.ds)
	linuxHost2 := createOrbitEnrolledHost(t, "rhel", "h2", s.ds)
	linuxHost2.OSVersion = "Fedora 41"
	err = s.ds.UpdateHost(ctx, linuxHost2)
	require.NoError(t, err)
	linuxHost3 := createOrbitEnrolledHost(t, "arch", "h3", s.ds)

	err = s.ds.AddLabelsToHost(ctx, linuxHost1.ID, []uint{allHostsLblID})
	require.NoError(t, err)
	err = s.ds.AddLabelsToHost(ctx, linuxHost2.ID, []uint{allHostsLblID})
	require.NoError(t, err)
	err = s.ds.AddLabelsToHost(ctx, linuxHost3.ID, []uint{allHostsLblID})
	require.NoError(t, err)

	// ensure disk encryption is disabled
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{EnableDiskEncryption: new(false)}, http.StatusNoContent)
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	// all profiles counts are expected to be 0
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{}, nil)

	// filter hosts with any OS Settings status, should have none
	statuses := []fleet.OSSettingsStatus{
		fleet.OSSettingsVerified, fleet.OSSettingsVerifying,
		fleet.OSSettingsPending, fleet.OSSettingsFailed,
	}
	for _, status := range statuses {
		s.checkListHostsFilter(t, allHostsLblID, "os_settings", string(status))
	}

	// all disk encryption counts are expected to be 0 (not enabled)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{}, false)

	// filter hosts with any Disk Encryption status, should have none
	diskStatuses := []fleet.DiskEncryptionStatus{
		fleet.DiskEncryptionVerified, fleet.DiskEncryptionVerifying,
		fleet.DiskEncryptionActionRequired, fleet.DiskEncryptionEnforcing,
		fleet.DiskEncryptionFailed, fleet.DiskEncryptionRemovingEnforcement,
	}
	for _, status := range diskStatuses {
		s.checkListHostsFilter(t, allHostsLblID, "os_settings_disk_encryption", string(status))
	}

	// now turn disk encryption on
	s.Do("POST", "/api/latest/fleet/disk_encryption", updateDiskEncryptionRequest{EnableDiskEncryption: new(true)}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// the 2 LUKS-supported linux hosts should show up
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{Pending: 2}, nil)
	s.checkListHostsFilter(t, allHostsLblID, "os_settings", string(fleet.OSSettingsPending), linuxHost1.ID, linuxHost2.ID)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{ActionRequired: fleet.MDMPlatformsCounts{Linux: 2}}, false)
	s.checkListHostsFilter(t, allHostsLblID, "os_settings_disk_encryption", string(fleet.DiskEncryptionActionRequired), linuxHost1.ID, linuxHost2.ID)
}

// verifies that ListHosts, CountHosts and ListHostsInLabel all return the
// expected host IDs when run with the provided filter.
func (s *withServer) checkListHostsFilter(t *testing.T, allHostsLabelID uint, filter, value string, expectIDs ...uint) {
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes, filter, value)
	require.Len(t, listHostsRes.Hosts, len(expectIDs), value)
	require.ElementsMatch(t, expectIDs, pluckHostIDsFromHostResponse(listHostsRes.Hosts), value)

	countResp := countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, filter, value)
	require.Equal(t, len(expectIDs), countResp.Count, value)

	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", allHostsLabelID), nil, http.StatusOK, &listHostsRes, filter, value)
	require.Len(t, listHostsRes.Hosts, len(expectIDs), value)
	require.ElementsMatch(t, expectIDs, pluckHostIDsFromHostResponse(listHostsRes.Hosts), value)

	// The other hosts-filter-related endpoints (delete by filter, transfer by
	// filter, hosts report) all use either ListHosts or ListsHostsInLabel.
}

// fileVaultStatusHelpers returns helpers to drive the host's FileVault profile
// status and assert the derived disk encryption status through the API.
func (s *integrationMDMTestSuite) fileVaultStatusHelpers(ctx context.Context, host *fleet.Host) (
	setProfileStatus func(fleet.MDMDeliveryStatus),
	checkHost func(want fleet.DiskEncryptionStatus, wantAction *fleet.ActionRequiredState),
	checkHostListFilter func(status fleet.DiskEncryptionStatus, wantIncluded bool),
) {
	t := s.T()

	fileVaultProf := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)
	hostCmdUUID := uuid.New().String()
	setProfileStatus = func(status fleet.MDMDeliveryStatus) {
		require.NoError(t, s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
			ProfileUUID:       fileVaultProf.ProfileUUID,
			ProfileIdentifier: fileVaultProf.Identifier,
			HostUUID:          host.UUID,
			CommandUUID:       hostCmdUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &status,
			Checksum:          []byte("csum"),
			Scope:             fleet.PayloadScopeSystem,
		}}))
	}
	t.Cleanup(func() {
		require.NoError(t, s.ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
			HostUUID:      host.UUID,
			CommandUUID:   hostCmdUUID,
			ProfileUUID:   fileVaultProf.ProfileUUID,
			Status:        &fleet.MDMDeliveryVerifying,
			OperationType: fleet.MDMOperationTypeRemove,
		}))
		_ = s.ds.DeleteMDMAppleConfigProfile(ctx, fileVaultProf.ProfileUUID)
	})

	checkHost = func(want fleet.DiskEncryptionStatus, wantAction *fleet.ActionRequiredState) {
		var getHostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
		require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
		require.Equal(t, want, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
		if wantAction == nil {
			require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
		} else {
			require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
			require.Equal(t, *wantAction, *getHostResp.Host.MDM.MacOSSettings.ActionRequired)
		}
		require.NotNil(t, getHostResp.Host.MDM.OSSettings)
		require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
		require.Equal(t, want, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	}

	checkHostListFilter = func(status fleet.DiskEncryptionStatus, wantIncluded bool) {
		for _, param := range []string{"os_settings_disk_encryption", "macos_settings_disk_encryption"} {
			var listResp listHostsResponse
			s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, param, string(status))
			var found bool
			for _, h := range listResp.Hosts {
				if h.ID == host.ID {
					found = true
				}
			}
			require.Equal(t, wantIncluded, found, "%s=%s", param, status)
		}
	}
	return setProfileStatus, checkHost, checkHostListFilter
}

func (s *integrationMDMTestSuite) submitDarwinDiskEncryptionResults(nodeKey string, encrypted bool) {
	var rows []map[string]string
	if encrypted {
		rows = []map[string]string{{"1": "1"}}
	}
	distributedReq := SubmitDistributedQueryResultsRequest{
		NodeKey:  nodeKey,
		Results:  map[string][]map[string]string{"fleet_detail_query_disk_encryption_darwin": rows},
		Statuses: map[string]fleet.OsqueryStatus{"fleet_detail_query_disk_encryption_darwin": 0},
	}
	var distributedResp submitDistributedQueryResultsResponse
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
}

func (s *integrationMDMTestSuite) submitDarwinFileVaultKey(ctx context.Context, nodeKey, recoveryKey string) {
	t := s.T()
	cert, err := assets.CAKeyPair(ctx, s.ds)
	require.NoError(t, err)
	encryptedKey, err := pkcs7.Encrypt([]byte(recoveryKey), []*x509.Certificate{cert.Leaf})
	require.NoError(t, err)

	distributedReq := SubmitDistributedQueryResultsRequest{
		NodeKey: nodeKey,
		Results: map[string][]map[string]string{
			"fleet_detail_query_mdm_disk_encryption_key_file_darwin": {{
				"encrypted":     "1",
				"filevault_key": base64.StdEncoding.EncodeToString(encryptedKey),
			}},
		},
		Statuses: map[string]fleet.OsqueryStatus{"fleet_detail_query_mdm_disk_encryption_key_file_darwin": 0},
	}
	var distributedResp submitDistributedQueryResultsResponse
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
}

func (s *integrationMDMTestSuite) TestRotateFileVaultKey() {
	t := s.T()
	ctx := t.Context()

	// Escrow is the only setting rotation needs.
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": false, "enable_escrow_disk_encryption_key": true } }
	}`), http.StatusOK)
	t.Cleanup(func() {
		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": { "macos_settings": { "enable_escrow_disk_encryption_key": false } }
		}`), http.StatusOK)
	})

	caKeyPair, err := assets.CAKeyPair(ctx, s.ds)
	require.NoError(t, err)
	encrypt := func(plain string) []byte {
		b, err := pkcs7.Encrypt([]byte(plain), []*x509.Certificate{caKeyPair.Leaf})
		require.NoError(t, err)
		return b
	}
	storedKey := func(hostID uint) *fleet.HostDiskEncryptionKey {
		key, err := s.ds.GetHostDiskEncryptionKey(ctx, hostID)
		require.NoError(t, err)
		return key
	}
	getKey := func(hostID uint) getHostEncryptionKeyResponse {
		var resp getHostEncryptionKeyResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", hostID), nil, http.StatusOK, &resp)
		return resp
	}
	countRows := func(query string, args ...any) int {
		var n int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n, query, args...)
		})
		return n
	}
	archiveRows := func(hostID uint) int {
		return countRows(`SELECT COUNT(*) FROM host_disk_encryption_keys_archive WHERE host_id = ?`, hostID)
	}
	hostActivities := func(hostID uint, activityType string) int {
		// not every host activity is linked in activity_host_past, so match on details
		return countRows(`
			SELECT COUNT(*) FROM activity_past
			WHERE activity_type = ? AND JSON_EXTRACT(details, '$.host_id') = ?`, activityType, hostID)
	}
	rotateURL := func(hostID uint) string {
		return fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key/rotate", hostID)
	}
	// the device may have enrollment commands queued ahead of the rotation
	nextRotateCommand := func(t *testing.T, mdmClient *mdmtest.TestAppleMDMClient) *mdm.Command {
		cmd, err := mdmClient.Idle()
		for ; err == nil && cmd != nil; cmd, err = mdmClient.Acknowledge(cmd.CommandUUID) {
			if cmd.Command.RequestType == fleet.RotateFileVaultKeyCmdName {
				return cmd
			}
		}
		require.NoError(t, err)
		return nil
	}
	newEscrowedHost := func(t *testing.T, recoveryKey string) (*fleet.Host, *mdmtest.TestAppleMDMClient) {
		host, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)
		s.submitDarwinFileVaultKey(ctx, *host.NodeKey, recoveryKey)
		require.NoError(t, s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, true, time.Now()))
		require.Nil(t, nextRotateCommand(t, mdmClient))
		return host, mdmClient
	}

	t.Run("host acknowledges", func(t *testing.T) {
		host, mdmClient := newEscrowedHost(t, "AAA-111-222")
		before := getKey(host.ID)
		require.Equal(t, "AAA-111-222", before.EncryptionKey.DecryptedValue)
		require.False(t, before.EncryptionKey.RotationPending)
		archivedBefore := archiveRows(host.ID)
		escrowedBefore := hostActivities(host.ID, fleet.ActivityTypeEscrowedDiskEncryptionKey{}.ActivityName())
		readBefore := hostActivities(host.ID, fleet.ActivityTypeReadHostDiskEncryptionKey{}.ActivityName())

		s.Do("POST", rotateURL(host.ID), nil, http.StatusOK)
		s.lastActivityMatches(fleet.ActivityTypeRotatedDiskEncryptionKey{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)
		require.Equal(t, readBefore, hostActivities(host.ID, fleet.ActivityTypeReadHostDiskEncryptionKey{}.ActivityName()),
			"rotating does not read the key on the user's behalf")

		marker := storedKey(host.ID).RotationCommandUUID
		require.NotNil(t, marker)
		var storedCmd string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &storedCmd, `SELECT command FROM nano_commands WHERE command_uuid = ?`, *marker)
		})
		require.Contains(t, storedCmd, "$FLEET_HOST_SECRET_FILEVAULT_KEY")
		require.NotContains(t, storedCmd, "AAA-111-222")

		pending := getKey(host.ID)
		require.True(t, pending.EncryptionKey.RotationPending)
		require.Equal(t, "AAA-111-222", pending.EncryptionKey.DecryptedValue)
		require.True(t, before.EncryptionKey.UpdatedAt.Equal(pending.EncryptionKey.UpdatedAt))

		s.Do("POST", rotateURL(host.ID), nil, http.StatusConflict)

		cmd := nextRotateCommand(t, mdmClient)
		require.NotNil(t, cmd)
		require.Equal(t, *marker, cmd.CommandUUID)
		require.Contains(t, string(cmd.Raw), "AAA-111-222", "the device receives the current key")
		require.NotContains(t, string(cmd.Raw), "FLEET_HOST_SECRET")

		_, err := mdmClient.AcknowledgeRotateFileVaultKey(cmd.CommandUUID, encrypt("NEW-333-444"))
		require.NoError(t, err)

		after := getKey(host.ID)
		require.Equal(t, "NEW-333-444", after.EncryptionKey.DecryptedValue)
		require.False(t, after.EncryptionKey.RotationPending)
		require.True(t, after.EncryptionKey.UpdatedAt.After(before.EncryptionKey.UpdatedAt))
		require.Equal(t, new(true), storedKey(host.ID).Decryptable, "decryptable without waiting for the cron")
		require.Equal(t, archivedBefore+1, archiveRows(host.ID))
		require.Equal(t, escrowedBefore+1, hostActivities(host.ID, fleet.ActivityTypeEscrowedDiskEncryptionKey{}.ActivityName()))

		// osquery then reports FileVaultPRK.dat: the same key in a different envelope
		s.submitDarwinFileVaultKey(ctx, *host.NodeKey, "NEW-333-444")
		ingested := storedKey(host.ID)
		require.Equal(t, new(true), ingested.Decryptable)
		require.True(t, after.EncryptionKey.UpdatedAt.Equal(ingested.UpdatedAt))
		require.Equal(t, archivedBefore+1, archiveRows(host.ID))
		require.Equal(t, escrowedBefore+1, hostActivities(host.ID, fleet.ActivityTypeEscrowedDiskEncryptionKey{}.ActivityName()))

		// a report of a different key follows the normal path
		s.submitDarwinFileVaultKey(ctx, *host.NodeKey, "OTHER-555")
		require.Equal(t, archivedBefore+2, archiveRows(host.ID), "a different key still follows the normal path")
		require.Nil(t, storedKey(host.ID).Decryptable)

		s.Do("POST", rotateURL(host.ID), nil, http.StatusUnprocessableEntity)
	})

	t.Run("osquery reports the new key before the acknowledgement", func(t *testing.T) {
		host, mdmClient := newEscrowedHost(t, "AAA-111-222")
		s.Do("POST", rotateURL(host.ID), nil, http.StatusOK)
		cmd := nextRotateCommand(t, mdmClient)
		require.NotNil(t, cmd)

		s.submitDarwinFileVaultKey(ctx, *host.NodeKey, "NEW-333-444")
		reported := storedKey(host.ID)
		require.Nil(t, reported.Decryptable)
		archived := archiveRows(host.ID)
		escrowed := hostActivities(host.ID, fleet.ActivityTypeEscrowedDiskEncryptionKey{}.ActivityName())

		_, err := mdmClient.AcknowledgeRotateFileVaultKey(cmd.CommandUUID, encrypt("NEW-333-444"))
		require.NoError(t, err)

		acked := storedKey(host.ID)
		require.Equal(t, reported.Base64Encrypted, acked.Base64Encrypted, "the reported copy is kept")
		require.Equal(t, new(true), acked.Decryptable)
		require.Nil(t, acked.RotationCommandUUID)
		require.Equal(t, archived, archiveRows(host.ID))
		require.Equal(t, escrowed, hostActivities(host.ID, fleet.ActivityTypeEscrowedDiskEncryptionKey{}.ActivityName()))
		require.Equal(t, "NEW-333-444", getKey(host.ID).EncryptionKey.DecryptedValue)
	})

	t.Run("host returns an error", func(t *testing.T) {
		host, mdmClient := newEscrowedHost(t, "AAA-111-222")
		s.Do("POST", rotateURL(host.ID), nil, http.StatusOK)
		cmd := nextRotateCommand(t, mdmClient)
		require.NotNil(t, cmd)

		_, err := mdmClient.Err(cmd.CommandUUID, []mdm.ErrorChain{{
			ErrorCode: 12, ErrorDomain: "MDMErrorDomain", USEnglishDescription: "The password is incorrect.",
		}})
		require.NoError(t, err)

		key := storedKey(host.ID)
		require.Nil(t, key.RotationCommandUUID)
		require.Equal(t, new(false), key.Decryptable)
		s.lastActivityMatches(fleet.ActivityTypeFailedToRotateDiskEncryptionKey{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "detail": "MDMErrorDomain (12): The password is incorrect."}`,
				host.ID, host.DisplayName()), 0)

		// the stored key is no longer usable, so there is nothing to rotate with
		s.Do("POST", rotateURL(host.ID), nil, http.StatusUnprocessableEntity)
	})

	t.Run("acknowledgement without a usable key", func(t *testing.T) {
		host, mdmClient := newEscrowedHost(t, "AAA-111-222")
		s.Do("POST", rotateURL(host.ID), nil, http.StatusOK)
		cmd := nextRotateCommand(t, mdmClient)
		require.NotNil(t, cmd)

		_, err := mdmClient.AcknowledgeRotateFileVaultKey(cmd.CommandUUID, []byte("not cms"))
		require.NoError(t, err)

		key := storedKey(host.ID)
		require.Nil(t, key.RotationCommandUUID)
		require.Equal(t, new(false), key.Decryptable)
		s.lastActivityMatches(fleet.ActivityTypeFailedToRotateDiskEncryptionKey{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "detail": "Fleet couldn't decrypt the new key returned by the host."}`,
				host.ID, host.DisplayName()), 0)
	})

	t.Run("stored key changes before delivery", func(t *testing.T) {
		host, mdmClient := newEscrowedHost(t, "AAA-111-222")
		s.Do("POST", rotateURL(host.ID), nil, http.StatusOK)
		replaced, err := s.ds.ReplaceHostDiskEncryptionKeyBlob(ctx, host.ID, storedKey(host.ID).Base64Encrypted,
			base64.StdEncoding.EncodeToString([]byte("not cms")))
		require.NoError(t, err)
		require.True(t, replaced)

		require.Nil(t, nextRotateCommand(t, mdmClient), "a command that can't be expanded is not delivered")
		key := storedKey(host.ID)
		require.Nil(t, key.RotationCommandUUID)
		require.Equal(t, new(false), key.Decryptable)
		var listActivities listActivitiesResponse
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities, "order_key", "id", "order_direction", "desc", "per_page", "1")
		require.Len(t, listActivities.Activities, 1)
		require.Equal(t, fleet.ActivityTypeFailedToRotateDiskEncryptionKey{}.ActivityName(), listActivities.Activities[0].Type)
		require.Contains(t, string(*listActivities.Activities[0].Details), "failed to expand host secrets")
	})

	t.Run("transfer to a fleet without escrow while pending", func(t *testing.T) {
		host, mdmClient := newEscrowedHost(t, "AAA-111-222")
		s.Do("POST", rotateURL(host.ID), nil, http.StatusOK)
		rotation := storedKey(host.ID).RotationCommandUUID
		require.NotNil(t, rotation)
		followUp := uuid.NewString()
		require.NoError(t, s.mdmCommander.EnqueueCommand(ctx, []string{host.UUID}, fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>ProfileList</string>
	</dict>
	<key>CommandUUID</key>
	<string>%s</string>
</dict>
</plist>`, followUp)))

		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{host.ID}},
			http.StatusOK, &addHostsToTeamResponse{})
		_, err = s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
		require.True(t, fleet.IsNotFound(err), "the destination fleet doesn't escrow, so the key is dropped")

		// The rotation can't be expanded any more. It must fail rather than block
		// the commands queued behind it.
		var delivered []string
		cmd, err := mdmClient.Idle()
		for i := 0; err == nil && i < 10; i++ {
			if cmd != nil {
				delivered = append(delivered, cmd.CommandUUID)
				cmd, err = mdmClient.Acknowledge(cmd.CommandUUID)
			} else {
				cmd, err = mdmClient.Idle()
			}
		}
		require.NoError(t, err)
		require.Contains(t, delivered, followUp)
		require.NotContains(t, delivered, *rotation)

		var status string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &status, `SELECT status FROM nano_command_results WHERE id = ? AND command_uuid = ?`, host.UUID, *rotation)
		})
		require.Equal(t, fleet.MDMAppleStatusError, status)
	})

	t.Run("a marker whose command is gone is replaced after the grace period", func(t *testing.T) {
		host, _ := newEscrowedHost(t, "AAA-111-222")
		s.Do("POST", rotateURL(host.ID), nil, http.StatusOK)
		first := storedKey(host.ID).RotationCommandUUID
		require.NotNil(t, first)
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM nano_enrollment_queue WHERE command_uuid = ?`, *first)
			return err
		})

		// indistinguishable from a request whose command isn't enqueued yet
		require.True(t, getKey(host.ID).EncryptionKey.RotationPending)
		s.Do("POST", rotateURL(host.ID), nil, http.StatusConflict)

		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				UPDATE host_disk_encryption_keys
				SET rotation_requested_at = NOW(6) - INTERVAL 2 MINUTE, updated_at = updated_at
				WHERE host_id = ?`, host.ID)
			return err
		})
		// GET and rotate agree that the marker is stale
		require.False(t, getKey(host.ID).EncryptionKey.RotationPending)
		s.Do("POST", rotateURL(host.ID), nil, http.StatusOK)
		require.True(t, getKey(host.ID).EncryptionKey.RotationPending)
		second := storedKey(host.ID).RotationCommandUUID
		require.NotNil(t, second)
		require.NotEqual(t, *first, *second)
	})

	t.Run("re-enrollment clears a pending rotation", func(t *testing.T) {
		host, mdmClient := newEscrowedHost(t, "AAA-111-222")
		s.Do("POST", rotateURL(host.ID), nil, http.StatusOK)
		require.NotNil(t, storedKey(host.ID).RotationCommandUUID)

		require.NoError(t, mdmClient.Authenticate())
		key, err := s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
		if !fleet.IsNotFound(err) {
			require.NoError(t, err)
			require.Nil(t, key.RotationCommandUUID)
		}
	})

	t.Run("ineligible hosts", func(t *testing.T) {
		s.Do("POST", rotateURL(0), nil, http.StatusNotFound)

		windowsHost := createOrbitEnrolledHost(t, "windows", t.Name(), s.ds)
		s.Do("POST", rotateURL(windowsHost.ID), nil, http.StatusBadRequest)

		host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
		s.Do("POST", rotateURL(host.ID), nil, http.StatusUnprocessableEntity)
	})
}

// Escrow without enforcement: status follows the escrowed key, not the disk
// state — action required (turn on encryption while the disk is off, rotate
// key once it is on) → verifying → verified.
func (s *integrationMDMTestSuite) TestMDMAppleHostDiskEncryptionEscrowOnly() {
	t := s.T()
	ctx := context.Background()

	host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": false, "enable_escrow_disk_encryption_key": true } }
	}`), http.StatusOK, &acResp)
	require.False(t, acResp.MDM.MacOSSettings.EnableDiskEncryption.Value)
	require.True(t, acResp.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	t.Cleanup(func() {
		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": { "macos_settings": { "enable_escrow_disk_encryption_key": false } }
		}`), http.StatusOK)
	})

	setProfileStatus, checkHost, checkHostListFilter := s.fileVaultStatusHelpers(ctx, host)
	rotateKey := fleet.ActionRequiredRotateKey
	turnOnEncryption := fleet.ActionRequiredTurnOnEncryption

	setProfileStatus(fleet.MDMDeliveryPending)
	checkHost(fleet.DiskEncryptionEnforcing, nil)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{Enforcing: fleet.MDMPlatformsCounts{MacOS: 1}}, true)

	// profile installed, nothing enforces FileVault and the disk is still off
	setProfileStatus(fleet.MDMDeliveryVerified)
	s.submitDarwinDiskEncryptionResults(*host.NodeKey, false)
	checkHost(fleet.DiskEncryptionActionRequired, &turnOnEncryption)
	checkHostListFilter(fleet.DiskEncryptionActionRequired, true)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{ActionRequired: fleet.MDMPlatformsCounts{MacOS: 1}}, true)

	// disk encrypted by a third party, no key escrowed yet
	s.submitDarwinDiskEncryptionResults(*host.NodeKey, true)
	checkHost(fleet.DiskEncryptionActionRequired, &rotateKey)
	checkHostListFilter(fleet.DiskEncryptionActionRequired, true)
	checkHostListFilter(fleet.DiskEncryptionVerified, false)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{ActionRequired: fleet.MDMPlatformsCounts{MacOS: 1}}, true)

	// Escrow Buddy rotated the key and osquery reported it
	s.submitDarwinFileVaultKey(ctx, *host.NodeKey, "ABC-111-222")
	checkHost(fleet.DiskEncryptionVerifying, nil)
	checkHostListFilter(fleet.DiskEncryptionVerifying, true)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{Verifying: fleet.MDMPlatformsCounts{MacOS: 1}}, true)

	// the cron decrypted the key
	require.NoError(t, s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, true, time.Now()))
	checkHost(fleet.DiskEncryptionVerified, nil)
	checkHostListFilter(fleet.DiskEncryptionVerified, true)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{Verified: fleet.MDMPlatformsCounts{MacOS: 1}}, true)

	// undecryptable key: back to action required despite the encrypted disk
	require.NoError(t, s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, false, time.Now()))
	checkHost(fleet.DiskEncryptionActionRequired, &rotateKey)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{ActionRequired: fleet.MDMPlatformsCounts{MacOS: 1}}, true)
}

// Enforcement without escrow: status follows the reported disk state —
// enforcing → verifying (unknown) → action required (log out) → verified.
func (s *integrationMDMTestSuite) TestMDMAppleHostDiskEncryptionEnforceOnly() {
	t := s.T()
	ctx := context.Background()

	host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": true, "enable_escrow_disk_encryption_key": false } }
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption.Value)
	require.False(t, acResp.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey.Value)
	t.Cleanup(func() {
		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": { "macos_settings": { "enable_disk_encryption": false } }
		}`), http.StatusOK)
	})

	setProfileStatus, checkHost, checkHostListFilter := s.fileVaultStatusHelpers(ctx, host)
	logOut := fleet.ActionRequiredLogOut

	setProfileStatus(fleet.MDMDeliveryPending)
	checkHost(fleet.DiskEncryptionEnforcing, nil)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{Enforcing: fleet.MDMPlatformsCounts{MacOS: 1}}, true)

	// profile installed, osquery hasn't reported the disk state yet
	setProfileStatus(fleet.MDMDeliveryVerified)
	checkHost(fleet.DiskEncryptionVerifying, nil)
	checkHostListFilter(fleet.DiskEncryptionVerifying, true)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{Verifying: fleet.MDMPlatformsCounts{MacOS: 1}}, true)

	// FileVault is deferred until the user logs out, so the disk is still unencrypted
	s.submitDarwinDiskEncryptionResults(*host.NodeKey, false)
	checkHost(fleet.DiskEncryptionActionRequired, &logOut)
	checkHostListFilter(fleet.DiskEncryptionActionRequired, true)
	checkHostListFilter(fleet.DiskEncryptionVerified, false)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{ActionRequired: fleet.MDMPlatformsCounts{MacOS: 1}}, true)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{Pending: 1}, &fleet.MDMProfilesSummary{Pending: 1})

	// the user logged out and FileVault turned on
	s.submitDarwinDiskEncryptionResults(*host.NodeKey, true)
	checkHost(fleet.DiskEncryptionVerified, nil)
	checkHostListFilter(fleet.DiskEncryptionVerified, true)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{Verified: fleet.MDMPlatformsCounts{MacOS: 1}}, true)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{Verified: 1}, &fleet.MDMProfilesSummary{Verified: 1})

	// a key, in any state, doesn't affect enforce-only derivation
	s.submitDarwinFileVaultKey(ctx, *host.NodeKey, "ABC-111-222")
	require.NoError(t, s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, false, time.Now()))
	checkHost(fleet.DiskEncryptionVerified, nil)
	s.checkMDMDiskEncryptionSummaries(t, nil, fleet.MDMDiskEncryptionSummary{Verified: fleet.MDMPlatformsCounts{MacOS: 1}}, true)
}

// The archived-key lookup can fall back to matching on hardware_serial, which is
// agent-reported and carries no tenancy. That fallback is opt-in via a query
// parameter and restricted to globally-scoped users; this exercises the public
// endpoint so the parameter cannot silently stop reaching the service.
func (s *integrationMDMTestSuite) TestHostEncryptionKeyArchivedSerialFallback() {
	t := s.T()
	ctx := context.Background()

	const serial = "ARCHIVEFALLBACK1"
	recoveryKey := "AAA-BBB-CCC"

	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert}, nil)
	require.NoError(t, err)
	block, _ := pem.Decode(assets[fleet.MDMAssetCACert].Value)
	require.NotNil(t, block)
	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	encryptFor := func(key string) string {
		enc, err := pkcs7.Encrypt([]byte(key), []*x509.Certificate{parsed})
		require.NoError(t, err)
		return base64.StdEncoding.EncodeToString(enc)
	}

	victimTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_victim"})
	require.NoError(t, err)
	claimantTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_claimant"})
	require.NoError(t, err)

	newHost := func(suffix string, teamID uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new(t.Name() + suffix),
			UUID:            t.Name() + suffix,
			Hostname:        t.Name() + suffix,
			HardwareSerial:  serial,
			Platform:        "darwin",
			TeamID:          &teamID,
		})
		require.NoError(t, err)
		return h
	}

	// Every escrow of a changed key appends to the archive, and the fallback takes the
	// newest row for the serial - so the second key here is the one it should reach.
	victim := newHost("_victim", victimTeam.ID)
	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, victim, encryptFor("DDD-EEE-FFF"), "", nil)
	require.NoError(t, err)
	_, err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, victim, encryptFor(recoveryKey), "", nil)
	require.NoError(t, err)

	// Same serial, different fleet, and no key of its own: the shape a spoofed
	// enrollment produces.
	claimant := newHost("_claimant", claimantTeam.ID)

	claimantObserver := &fleet.User{
		Name:       "claimant observer",
		Email:      t.Name() + "_observer@example.com",
		GlobalRole: nil,
		Teams:      []fleet.UserTeam{{Team: *claimantTeam, Role: fleet.RoleObserver}},
	}
	require.NoError(t, claimantObserver.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, claimantObserver)
	require.NoError(t, err)

	keyURL := fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", claimant.ID)

	t.Run("fleet-scoped user is denied the fallback", func(t *testing.T) {
		s.setTokenForTest(t, claimantObserver.Email, test.GoodPassword)
		defer func() { s.token = s.getTestAdminToken() }()

		// Asking for the fallback explicitly must not reach the other fleet's key, and
		// must be indistinguishable from the key simply not existing.
		s.Do("GET", keyURL+"?allow_serial_lookup=true", nil, http.StatusNotFound)
		s.Do("GET", keyURL, nil, http.StatusNotFound)
	})

	t.Run("global user only gets it when opting in", func(t *testing.T) {
		// Omitting the parameter must keep the old, host-id-only behaviour: the
		// parameter is optional, so this also pins that it stays optional.
		s.Do("GET", keyURL, nil, http.StatusNotFound)

		var resp getHostEncryptionKeyResponse
		s.DoJSON("GET", keyURL+"?allow_serial_lookup=true", nil, http.StatusOK, &resp)
		require.NotNil(t, resp.EncryptionKey)
		require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue)
	})
}

// TestBitLockerPINHandoff drives the whole end-user PIN flow: the My device page submits a PIN, the agent is told to
// collect it on its next config poll, collects it exactly once, reports success, and the host comes out the other side
// with its PIN recorded, an activity written, and the notification gone.
func (s *integrationMDMTestSuite) TestBitLockerPINHandoff() {
	t := s.T()
	ctx := t.Context()

	// The Windows enrollment row is what carries both the capability and the pending flag.
	host, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	deviceToken := "bitlocker-pin-handoff-" + uuid.NewString()
	require.NoError(t, s.ds.SetOrUpdateDeviceAuthToken(ctx, host.ID, deviceToken))

	// Require a PIN, then put the host in the one state where only the end user can act: encrypted, key escrowed and
	// decryptable, protection on, and no PIN yet.
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config",
		json.RawMessage(`{ "mdm": { "enable_disk_encryption": true, "windows_require_bitlocker_pin": true } }`),
		http.StatusOK, &acResp)
	protectionOn := fleet.BitLockerProtectionStatusOn
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(ctx, host.ID, true, &protectionOn))
	_, err := s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host, "test-key", "", new(true))
	require.NoError(t, err)
	// The volume has a TPM protector but no PIN, the only state in which Windows offers PIN setup.
	require.NoError(t, s.ds.SetOrUpdateHostDiskBitLockerProtectors(ctx, host.ID, true, false))

	orbitConfig := func(caps string) fleet.OrbitGetConfigResponse {
		headers := map[string]string{}
		if caps != "" {
			headers[fleet.CapabilitiesHeader] = caps
		}
		res := s.DoRawWithHeaders("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, headers)
		var resp fleet.OrbitGetConfigResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))
		return resp
	}
	deviceHost := func() getDeviceHostResponse {
		var resp getDeviceHostResponse
		s.DoJSON("GET", "/api/latest/fleet/device/"+deviceToken, nil, http.StatusOK, &resp)
		return resp
	}
	submitPIN := func(pin string, wantStatus int) {
		body, err := json.Marshal(map[string]string{"pin": pin})
		require.NoError(t, err)
		s.Do("POST", "/api/latest/fleet/device/"+deviceToken+"/disk_encryption_pin", json.RawMessage(body), wantStatus)
	}
	collectPIN := func(wantStatus int) fleet.OrbitGetDiskEncryptionPINDetailsResponse {
		var resp fleet.OrbitGetDiskEncryptionPINDetailsResponse
		s.DoJSON("POST", "/api/fleet/orbit/disk_encryption_pin/details",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), wantStatus, &resp)
		return resp
	}
	reportOutcome := func(requestUUID string, outcome fleet.BitLockerPINRequestStatus, clientError string, wantStatus int) {
		body, err := json.Marshal(map[string]string{
			"orbit_node_key": *host.OrbitNodeKey, "request_uuid": requestUUID, "outcome": string(outcome), "client_error": clientError,
		})
		require.NoError(t, err)
		s.Do("POST", "/api/fleet/orbit/disk_encryption_pin/result", json.RawMessage(body), wantStatus)
	}

	// An agent that has not advertised the capability is treated as unable to apply a PIN, so the page keeps the old
	// instructions modal and a submission is refused.
	orbitConfig("")
	resp := deviceHost()
	require.NotNil(t, resp.Host.MDM.OSSettings)
	require.NotNil(t, resp.Host.MDM.OSSettings.DiskEncryption.ActionRequired)
	require.Equal(t, fleet.ActionRequiredCreatePIN, *resp.Host.MDM.OSSettings.DiskEncryption.ActionRequired)
	require.NotNil(t, resp.Host.MDM.OSSettings.DiskEncryption.FleetdCanSetPIN)
	require.False(t, *resp.Host.MDM.OSSettings.DiskEncryption.FleetdCanSetPIN)
	submitPIN("123456", http.StatusBadRequest)

	// The agent advertises that it can apply a PIN; the server persists that on the enrollment row.
	caps := fleet.CapabilityMap{fleet.CapabilityWindowsBitLockerPIN: struct{}{}}
	capsHeader := caps.String()
	require.False(t, orbitConfig(capsHeader).Notifications.BitLockerPINRequestPending)
	resp = deviceHost()
	require.NotNil(t, resp.Host.MDM.OSSettings.DiskEncryption.FleetdCanSetPIN)
	require.True(t, *resp.Host.MDM.OSSettings.DiskEncryption.FleetdCanSetPIN)

	// A PIN that Windows would reject is refused with a validation error before anything is stored.
	submitPIN("12ab", http.StatusUnprocessableEntity)
	require.False(t, orbitConfig(capsHeader).Notifications.BitLockerPINRequestPending)

	// The end user submits a valid PIN, and the agent's next poll is told to collect it.
	submitPIN("123456", http.StatusNoContent)
	resp = deviceHost()
	require.NotNil(t, resp.Host.MDM.OSSettings.DiskEncryption.PINRequest)
	require.Equal(t, fleet.BitLockerPINRequestPending, resp.Host.MDM.OSSettings.DiskEncryption.PINRequest.Status)
	require.True(t, orbitConfig(capsHeader).Notifications.BitLockerPINRequestPending)

	// Collecting returns the PIN in the clear, exactly once.
	pinResp := collectPIN(http.StatusOK)
	require.Equal(t, "123456", pinResp.PIN)
	firstRequestUUID := pinResp.RequestUUID

	// An outcome that names no collected submission is refused.
	reportOutcome("not-a-real-request", fleet.BitLockerPINRequestSet, "", http.StatusNotFound)

	// A replayed collect gets nothing, and the notification is already gone.
	collectPIN(http.StatusNotFound)
	require.False(t, orbitConfig(capsHeader).Notifications.BitLockerPINRequestPending)

	// A failure is reported back to the waiting page.
	reportOutcome(firstRequestUUID, fleet.BitLockerPINRequestFailed, "PIN already set", http.StatusNoContent)
	resp = deviceHost()
	require.NotNil(t, resp.Host.MDM.OSSettings.DiskEncryption.PINRequest)
	require.Equal(t, fleet.BitLockerPINRequestFailed, resp.Host.MDM.OSSettings.DiskEncryption.PINRequest.Status)
	require.Equal(t, "PIN already set", resp.Host.MDM.OSSettings.DiskEncryption.PINRequest.Error)
	require.NotNil(t, resp.Host.MDM.OSSettings.DiskEncryption.ActionRequired)

	// The end user retries with an enhanced PIN, the agent collects it, and this time it works.
	submitPIN(`Fl"eet 2026!`, http.StatusNoContent)
	require.True(t, orbitConfig(capsHeader).Notifications.BitLockerPINRequestPending)
	pinResp = collectPIN(http.StatusOK)
	require.Equal(t, `Fl"eet 2026!`, pinResp.PIN)
	require.NotEqual(t, firstRequestUUID, pinResp.RequestUUID, "each submission gets its own id")

	// A late outcome for the superseded submission must not be recorded against this one.
	reportOutcome(firstRequestUUID, fleet.BitLockerPINRequestSet, "", http.StatusNotFound)
	reportOutcome(pinResp.RequestUUID, fleet.BitLockerPINRequestSet, "", http.StatusNoContent)

	// The PIN is recorded, so the end user's banner clears without them pressing Refetch.
	resp = deviceHost()
	require.Nil(t, resp.Host.MDM.OSSettings.DiskEncryption.ActionRequired)
	require.NotNil(t, resp.Host.MDM.OSSettings.DiskEncryption.PINRequest)
	require.Equal(t, fleet.BitLockerPINRequestSet, resp.Host.MDM.OSSettings.DiskEncryption.PINRequest.Status)
	require.False(t, orbitConfig(capsHeader).Notifications.BitLockerPINRequestPending)

	// Success asks the host to refetch so osquery confirms the protector list, and is recorded for the admin with no
	// actor, because the end user chose the PIN rather than Fleet applying one of its own.
	refetched, err := s.ds.Host(ctx, host.ID)
	require.NoError(t, err)
	require.True(t, refetched.RefetchRequested)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeCreatedDiskEncryptionPIN{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()),
		0,
	)

	// The PIN fields are Windows-only.
	macHost := createOrbitEnrolledHost(t, "darwin", t.Name()+"-mac", s.ds)
	macToken := "bitlocker-pin-handoff-mac-" + uuid.NewString()
	require.NoError(t, s.ds.SetOrUpdateDeviceAuthToken(ctx, macHost.ID, macToken))
	macRes := s.DoRaw("GET", "/api/latest/fleet/device/"+macToken, nil, http.StatusOK)
	macBody, err := io.ReadAll(macRes.Body)
	require.NoError(t, err)
	require.NoError(t, macRes.Body.Close())
	require.Contains(t, string(macBody), `"os_settings"`, "precondition: os_settings is present, so the checks below mean something")
	require.NotContains(t, string(macBody), "fleetd_can_set_pin")
	require.NotContains(t, string(macBody), "pin_request")
}
