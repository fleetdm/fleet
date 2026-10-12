package service

// Miscellaneous tests for the MDM suite.
//
// Belongs here: MDM enabled/configured gating across endpoints, custom enrollment web URLs, orbit
// nudge settings, fleet labels used by MDM features, staged uploads, org logos served to devices,
// and the rule that Windows hosts can't turn MDM off.
//
// Does not belong here: anything with a topic file of its own — check the other
// integration_mdm_*_test.go files first.

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestOrgLogo() {
	t := s.T()

	// change org logo urls
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_logo_url": "http://test-image.com",
			"org_logo_url_light_background": "http://test-image-light.com"
		}
	}`), http.StatusOK, &acResp)

	// enroll a host
	token := "token_test_migration"
	host := createOrbitEnrolledHost(t, "darwin", "h", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	// check icon urls are correct
	getDesktopResp := fleetDesktopResponse{}
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
	require.NoError(t, res.Body.Close())
	require.NoError(t, getDesktopResp.Err)
	require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)                               //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
	require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
}

func (s *integrationMDMTestSuite) TestOrbitConfigNudgeSettings() {
	t := s.T()

	// ensure the config is empty before starting
	s.applyConfig([]byte(`
  mdm:
    macos_updates:
      deadline: ""
      minimum_version: ""
 `))

	var resp fleet.OrbitGetConfigResponse
	// missing orbit key
	s.DoJSON("POST", "/api/fleet/orbit/config", nil, http.StatusUnauthorized, &resp)

	// nudge config is empty if macos_updates is not set, and Windows MDM notifications are unset
	h := createOrbitEnrolledHost(t, "darwin", "h", s.ds)

	err := s.ds.UpdateHostOperatingSystem(context.Background(), h.ID, fleet.OperatingSystem{Platform: "darwin", Version: "12.0"})
	require.NoError(t, err)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.NudgeConfig)
	require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
	require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
	require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)

	// set macos_updates
	s.applyConfig([]byte(`
  mdm:
    macos_updates:
      deadline: 2022-01-04
      minimum_version: 14.6.1
 `))

	// still empty if MDM is turned off for the host
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.NudgeConfig)

	// turn on MDM features
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.SerialNumber = h.HardwareSerial
	mdmDevice.UUID = h.UUID

	err = mdmDevice.Enroll()
	require.NoError(t, err)

	// set os after enroll, as we now clear vitals on re-enrollment
	err = s.ds.UpdateHostOperatingSystem(context.Background(), h.ID, fleet.OperatingSystem{Platform: "darwin", Version: "12.0"})
	require.NoError(t, err)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err := fleet.NewNudgeConfig(fleet.AppleOSUpdateSettings{Deadline: optjson.SetString("2022-01-04"), MinimumVersion: optjson.SetString("14.6.1")})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, "2022-01-04 20:00:00 +0000 UTC", wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String())

	// create a team with an empty macos_updates config
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          4827,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)
	s.assertMacOSDeclarationsByName(&team.ID, servermdm.FleetMacOSUpdatesProfileName, false)

	// add the host to the team
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{h.ID}))
	require.NoError(t, err)

	// NudgeConfig should be empty
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.NudgeConfig)
	require.Equal(t, "2022-01-04 20:00:00 +0000 UTC", wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String())

	// modify the team config, add macos_updates config
	var tmResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.AppleOSUpdateSettings{
				Deadline:       optjson.SetString("1992-01-01"),
				MinimumVersion: optjson.SetString("13.6.9"),
			},
		},
	}, http.StatusOK, &tmResp)
	s.assertMacOSDeclarationsByName(&team.ID, servermdm.FleetMacOSUpdatesProfileName, true)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err = fleet.NewNudgeConfig(fleet.AppleOSUpdateSettings{Deadline: optjson.SetString("1992-01-01"), MinimumVersion: optjson.SetString("13.6.9")})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, "1992-01-01 20:00:00 +0000 UTC", wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String())

	// create a new host, still receives the global config
	h2 := createOrbitEnrolledHost(t, "darwin", "h2", s.ds)
	mdmDevice = mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.SerialNumber = h2.HardwareSerial
	mdmDevice.UUID = h2.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	err = s.ds.UpdateHostOperatingSystem(context.Background(), h2.ID, fleet.OperatingSystem{Platform: "darwin", Version: "12.0"})
	require.NoError(t, err)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h2.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err = fleet.NewNudgeConfig(fleet.AppleOSUpdateSettings{Deadline: optjson.SetString("2022-01-04"), MinimumVersion: optjson.SetString("14.6.1")})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, "2022-01-04 20:00:00 +0000 UTC", wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String())

	// host on macos > 14, shouldn't be receiving nudge configs
	h3 := createOrbitEnrolledHost(t, "darwin", "h3", s.ds)

	mdmDevice = mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.SerialNumber = h3.HardwareSerial
	mdmDevice.UUID = h3.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	err = s.ds.UpdateHostOperatingSystem(context.Background(), h3.ID, fleet.OperatingSystem{Platform: "darwin", Version: "14.1"})
	require.NoError(t, err)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h3.OrbitNodeKey)), http.StatusOK, &resp)
	require.Nil(t, resp.NudgeConfig)

	// host is available for nudge, but has not had details query run
	// yet, so we don't know the os version.
	h4 := createOrbitEnrolledHost(t, "darwin", "h4", s.ds)

	mdmDevice = mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.SerialNumber = h4.HardwareSerial
	mdmDevice.UUID = h4.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h4.OrbitNodeKey)), http.StatusOK, &resp)
	require.Nil(t, resp.NudgeConfig)
}

func (s *integrationMDMTestSuite) TestMDMEnabledAndConfigured() {
	t := s.T()
	ctx := context.Background()

	appConfig, err := s.ds.AppConfig(ctx)
	originalCopy := appConfig.Copy()
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, s.ds.SaveAppConfig(ctx, originalCopy))
	})

	checkAppConfig := func(t *testing.T, mdmEnabled, winEnabled bool) appConfigResponse {
		acResp := appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.True(t, acResp.AppConfig.MDM.AppleBMEnabledAndConfigured)
		require.Equal(t, mdmEnabled, acResp.AppConfig.MDM.EnabledAndConfigured)
		require.Equal(t, winEnabled, acResp.AppConfig.MDM.WindowsEnabledAndConfigured)
		return acResp
	}

	compareMacOSSetupValues := func(t *testing.T, got fleet.MacOSSetup, want fleet.MacOSSetup) {
		require.Equal(t, want.BootstrapPackage.Value, got.BootstrapPackage.Value)
		require.Equal(t, want.MacOSSetupAssistant.Value, got.MacOSSetupAssistant.Value)
		require.Equal(t, want.EnableEndUserAuthentication, got.EnableEndUserAuthentication)
	}

	insertBootstrapPackageAndSetupAssistant := func(t *testing.T, teamID *uint) {
		var tmID uint
		if teamID != nil {
			tmID = *teamID
		}

		// cleanup any residual bootstrap package
		_ = s.ds.DeleteMDMAppleBootstrapPackage(ctx, tmID)

		// add new bootstrap package
		require.NoError(t, s.ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{
			TeamID: tmID,
			Name:   "foo",
			Token:  uuid.New().String(),
			Bytes:  []byte("foo"),
			Sha256: []byte("foo-sha256"),
		}, nil))

		// add new setup assistant
		_, err := s.ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
			TeamID:  teamID,
			Name:    "bar",
			Profile: []byte("{}"),
		})
		require.NoError(t, err)
	}

	// TODO: Some global MDM config settings don't have MDMEnabledAndConfigured or
	// WindowsMDMEnabledAndConfigured validations currently. Either add validations
	// and test them or test absence of validation.
	t.Run("apply app config spec", func(t *testing.T) {
		t.Run("disk encryption", func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
			})

			acResp := checkAppConfig(t, true, true)
			require.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabled by default

			// initialize our test app config
			ac := appConfig.Copy()
			ac.AgentOptions = nil
			// the flat toggle is virtual (recomputed as the AND of the
			// per-platform settings on save), so keep the per-platform
			// settings in sync for the direct ds.SaveAppConfig calls below
			setACDiskEncryption := func(v bool) {
				ac.MDM.EnableDiskEncryption = optjson.SetBool(v)
				ac.MDM.MacOSSettings.EnableDiskEncryption = optjson.SetBool(v)
				ac.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey = optjson.SetBool(v)
				ac.MDM.WindowsSettings.EnableDiskEncryption = optjson.SetBool(v)
				ac.MDM.LinuxSettings.EnableEscrowDiskEncryptionKey = optjson.SetBool(v)
			}

			// enable disk encryption
			setACDiskEncryption(true)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, true)                           // both mac and windows mdm enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // enabled

			// directly set MDM.EnabledAndConfigured to false
			ac.MDM.EnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, true)                          // only windows mdm enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabling mdm doesn't change disk encryption

			// making an unrelated change should not cause validation error
			ac.OrgInfo.OrgName = "f1337"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                          // only windows mdm enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // no change
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// disabling disk encryption doesn't cause validation error
			setACDiskEncryption(false)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                           // only windows mdm enabled
			require.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabled
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// enabling disk encryption doesn't cause validation error
			setACDiskEncryption(true)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                          // only windows mdm enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // enabled

			// directly set MDM.WindowsEnabledAndConfigured to false
			ac.MDM.WindowsEnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, false)                         // both mac and windows mdm disabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabling mdm doesn't change disk encryption

			// disabling disk encryption doesn't cause validation error
			setACDiskEncryption(false)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                          // no MDM enabled
			require.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabled
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// enabling disk encryption doesn't cause validation error
			setACDiskEncryption(true)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                         // no MDM enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // enabled

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1338"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                         // both mac and windows mdm disabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // no change
			require.Equal(t, "f1338", acResp.AppConfig.OrgInfo.OrgName)
		})

		t.Run("macos setup", func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
			})

			acResp := checkAppConfig(t, true, true)
			compareMacOSSetupValues(t, fleet.MacOSSetup{}, acResp.AppConfig.MDM.MacOSSetup) // disabled by default

			// initialize our test app config
			ac := appConfig.Copy()
			ac.AgentOptions = nil
			ac.MDM.EndUserAuthentication = fleet.MDMEndUserAuthentication{
				EntityID:    "sso-provider",
				IDPName:     "sso-provider",
				MetadataURL: "https://sso-provider.example.com/metadata",
			}

			// add db records for bootstrap package and setup assistant
			insertBootstrapPackageAndSetupAssistant(t, nil)

			// enable MacOSSetup options
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString("foo"),
				EnableEndUserAuthentication: true,
				MacOSSetupAssistant:         optjson.SetString("bar"),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, true)                               // both mac and windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied

			// directly set MDM.EnabledAndConfigured to false
			ac.MDM.EnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, true)                              // only windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // still applied

			// making an unrelated change should not cause validation error
			ac.OrgInfo.OrgName = "f1337"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                              // only windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // still applied
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// disabling doesn't cause validation error
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString(""),
				EnableEndUserAuthentication: false,
				MacOSSetupAssistant:         optjson.SetString(""),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                              // only windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// bootstrap package and setup assistant were removed so reinsert records for next test
			insertBootstrapPackageAndSetupAssistant(t, nil)

			// enable MacOSSetup options fails because only Windows is enabled.
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString("foo"),
				EnableEndUserAuthentication: true,
				MacOSSetupAssistant:         optjson.SetString("bar"),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, true) // only windows enabled

			// directly set MDM.EnabledAndConfigured to true and windows to false
			ac.MDM.EnabledAndConfigured = true
			ac.MDM.WindowsEnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, true, false)                              // mac enabled, windows disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // directly applied

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1338"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                              // mac enabled, windows disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // no change
			require.Equal(t, "f1338", acResp.AppConfig.OrgInfo.OrgName)

			// disabling doesn't cause validation error
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString(""),
				EnableEndUserAuthentication: false,
				MacOSSetupAssistant:         optjson.SetString(""),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                              // only windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied

			// bootstrap package and setup assistant were removed so reinsert records for next test
			insertBootstrapPackageAndSetupAssistant(t, nil)

			// enable MacOSSetup options succeeds because only Windows is disabled
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString("foo"),
				EnableEndUserAuthentication: true,
				MacOSSetupAssistant:         optjson.SetString("bar"),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                              // only windows enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied

			// directly set MDM.EnabledAndConfigured to false
			ac.MDM.EnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, false)                             // both mac and windows mdm disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // still applied

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1339"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                             // both disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // no change
			require.Equal(t, "f1339", acResp.AppConfig.OrgInfo.OrgName)

			// setting macos setup empty values doesn't cause validation error when mdm is disabled
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString(""),
				EnableEndUserAuthentication: false,
				MacOSSetupAssistant:         optjson.SetString(""),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                             // both disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied

			// setting macos setup to non-empty values fails because mdm disabled
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString("foo"),
				EnableEndUserAuthentication: true,
				MacOSSetupAssistant:         optjson.SetString("bar"),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, false) // both disabled
		})

		t.Run("custom settings", func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
			})

			// initialize our test app config
			ac := appConfig.Copy()
			ac.AgentOptions = nil
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp := checkAppConfig(t, true, true)
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// add custom settings
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, true)                                                                                 // both mac and windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // applied
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // applied

			// directly set MDM.EnabledAndConfigured to false
			ac.MDM.EnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, true)                                                                                // only windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // still applied
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // still applied

			// making an unrelated change should not cause validation error
			ac.OrgInfo.OrgName = "f1337"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                                                                                // only windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // still applied
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // still applied
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// remove custom settings
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true) // only windows mdm enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// add custom macOS settings fails because only windows is enabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, true) // only windows enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// add custom Windows settings suceeds because only macOS is disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                                                                                // only windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // applied
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)                                                              // no change

			// cleanup Windows settings
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true) // only windows mdm enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// directly set MDM.EnabledAndConfigured to true and windows to false
			ac.MDM.EnabledAndConfigured = true
			ac.MDM.WindowsEnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, true, false)                                                                // mac enabled, windows disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings) // directly applied
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)                                      // still empty

			// add custom windows settings fails because only mac is enabled
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, true, false) // only mac enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)
			// set this value to empty again so we can test other assertions assuming we're not setting it
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1338"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                                                                // mac enabled, windows disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings) // no change
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)                                      // no change
			require.Equal(t, "f1338", acResp.AppConfig.OrgInfo.OrgName)

			// remove custom settings doesn't cause validation error
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false) // only mac enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)

			// add custom macOS settings suceeds because only Windows is disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                                                                // mac enabled, windows disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings) // applied
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)                                      // no change

			// temporarily enable and add custom settings for both platforms
			ac.MDM.EnabledAndConfigured = true
			ac.MDM.WindowsEnabledAndConfigured = true
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, true, true) // both mac and windows mdm enabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, true)                                                                                 // both mac and windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // applied
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // applied

			// directly set both configs to false
			ac.MDM.EnabledAndConfigured = false
			ac.MDM.WindowsEnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, false)                                                                               // both mac and windows mdm disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // no change
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // no change

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1339"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                                                                               // both disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // no change
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // no change
			require.Equal(t, "f1339", acResp.AppConfig.OrgInfo.OrgName)

			// setting the same values is ok even if mdm is disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                                                                               // both disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // no change
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // no change

			// setting different values fail even if mdm is disabled, and only some of the profiles have changed
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "oof"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, false) // both disabled
			// set the values back so we can compare them
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // no change
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // no change

			// setting empty values doesn't cause validation error when mdm is disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false) // both disabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// setting non-empty values fails because mdm disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, false) // both disabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)
		})
	})

	// TODO: Improve validations and related test coverage of team MDM config.
	// Some settings don't have MDMEnabledAndConfigured or WindowsMDMEnabledAndConfigured
	// validations currently. Either add vailidations and test them or test abscence
	// of validation. Also, the tests below only cover a limited set of permutations
	// compared to the app config tests above and should be expanded accordingly.
	t.Run("modify team", func(t *testing.T) {
		t.Cleanup(func() {
			require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
		})

		checkTeam := func(t *testing.T, team *fleet.Team, checkMDM *fleet.TeamPayloadMDM) teamResponse {
			var wantDiskEncryption bool
			var wantMacOSSetup fleet.MacOSSetup
			if checkMDM != nil {
				if checkMDM.MacOSSetup != nil {
					wantMacOSSetup = *checkMDM.MacOSSetup
					// bootstrap package always ignored by modify team endpoint so expect original value
					wantMacOSSetup.BootstrapPackage = team.Config.MDM.MacOSSetup.BootstrapPackage
					// setup assistant always ignored by modify team endpoint so expect original value
					wantMacOSSetup.MacOSSetupAssistant = team.Config.MDM.MacOSSetup.MacOSSetupAssistant
				}
				wantDiskEncryption = checkMDM.EnableDiskEncryption.Value
			}

			var resp teamResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &resp)
			require.Equal(t, team.Name, resp.Team.Name)
			require.Equal(t, wantDiskEncryption, resp.Team.Config.MDM.EnableDiskEncryption)
			require.Equal(t, wantMacOSSetup.BootstrapPackage.Value, resp.Team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
			require.Equal(t, wantMacOSSetup.MacOSSetupAssistant.Value, resp.Team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
			require.Equal(t, wantMacOSSetup.EnableEndUserAuthentication, resp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)

			return resp
		}

		// initialize our test app config
		ac := appConfig.Copy()
		ac.AgentOptions = nil
		ac.MDM.EnabledAndConfigured = false
		ac.MDM.WindowsEnabledAndConfigured = false
		require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
		checkAppConfig(t, false, false) // both mac and windows mdm disabled

		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{fleet.TeamPayload{
			Name: new("Ninjas"),
			MDM:  &fleet.TeamPayloadMDM{EnableDiskEncryption: optjson.SetBool(true)}, // mdm is ignored by the create team endpoint
		}}, http.StatusOK, &createTeamResp)
		team := createTeamResp.Team
		getTeamResp := checkTeam(t, team, nil) // newly created team has empty mdm config

		t.Cleanup(func() {
			require.NoError(t, s.ds.DeleteTeam(ctx, team.ID))
		})

		// TODO: Add cases for other team MDM config (e.g., macos settings, macos updates,
		// migration) and for other permutations of starting values (see app config tests above).
		cases := []struct {
			name           string
			mdm            *fleet.TeamPayloadMDM
			expectedStatus int
		}{
			{
				"mdm empty",
				&fleet.TeamPayloadMDM{},
				http.StatusOK,
			},
			{
				"mdm all zero values",
				&fleet.TeamPayloadMDM{
					EnableDiskEncryption: optjson.SetBool(false),
					MacOSSetup: &fleet.MacOSSetup{
						BootstrapPackage:            optjson.SetString(""),
						EnableEndUserAuthentication: false,
						MacOSSetupAssistant:         optjson.SetString(""),
					},
				},
				http.StatusOK,
			},
			{
				"bootstrap package",
				&fleet.TeamPayloadMDM{
					MacOSSetup: &fleet.MacOSSetup{
						BootstrapPackage: optjson.SetString("some-package"),
					},
				},
				// bootstrap package is always ignored by the modify team endpoint
				http.StatusOK,
			},
			{
				"setup assistant",
				&fleet.TeamPayloadMDM{
					MacOSSetup: &fleet.MacOSSetup{
						MacOSSetupAssistant: optjson.SetString("some-setup-assistant"),
					},
				},
				// setup assistant is always ignored by the modify team endpoint
				http.StatusOK,
			},
			{
				"enable disk encryption",
				&fleet.TeamPayloadMDM{
					EnableDiskEncryption: optjson.SetBool(true),
				},
				// disk encryption requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
			{
				"enable end user auth",
				&fleet.TeamPayloadMDM{
					MacOSSetup: &fleet.MacOSSetup{
						EnableEndUserAuthentication: true,
					},
				},
				// disk encryption requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
		}

		for _, c := range cases {
			// TODO: Add tests for other combinations of mac and windows mdm enabled/disabled
			t.Run(c.name, func(t *testing.T) {
				checkAppConfig(t, false, false) // both mac and windows mdm disabled

				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
					Name:        &team.Name,
					Description: new(c.name),
					MDM:         c.mdm,
				}, c.expectedStatus, &getTeamResp)

				if c.expectedStatus == http.StatusOK {
					getTeamResp = checkTeam(t, team, c.mdm)
					require.Equal(t, c.name, getTeamResp.Team.Description)
				} else {
					checkTeam(t, team, nil)
				}
			})
		}
	})

	// TODO: Improve validations and related test coverage of team MDM config.
	// Some settings don't have MDMEnabledAndConfigured or WindowsMDMEnabledAndConfigured
	// validations currently. Either add vailidations and test them or test abscence
	// of validation. Also, the tests below only cover a limited set of permutations
	// compared to the app config tests above and should be expanded accordingly.
	t.Run("edit team spec", func(t *testing.T) {
		t.Cleanup(func() {
			require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
		})

		checkTeam := func(t *testing.T, team *fleet.Team, checkMDM *fleet.TeamSpecMDM) teamResponse {
			// TODO - remove check of disk encryption from this function entirely?
			// var wantDiskEncryption bool
			var wantMacOSSetup fleet.MacOSSetup
			if checkMDM != nil {
				wantMacOSSetup = checkMDM.MacOSSetup
				// wantDiskEncryption = checkMDM.EnableDiskEncryption.Value
			}

			var resp teamResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &resp)
			require.Equal(t, team.Name, resp.Team.Name)
			// require.Equal(t, wantDiskEncryption, resp.Team.Config.MDM.EnableDiskEncryption)
			require.Equal(t, wantMacOSSetup.BootstrapPackage.Value, resp.Team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
			require.Equal(t, wantMacOSSetup.MacOSSetupAssistant.Value, resp.Team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
			require.Equal(t, wantMacOSSetup.EnableEndUserAuthentication, resp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)

			return resp
		}

		// initialize our test app config
		ac := appConfig.Copy()
		ac.AgentOptions = nil
		ac.MDM.EnabledAndConfigured = false
		ac.MDM.WindowsEnabledAndConfigured = false
		require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
		checkAppConfig(t, false, false) // both mac and windows mdm disabled

		// create a team from spec
		tmSpecReq := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "Pirates"}}}
		var tmSpecResp applyTeamSpecsResponse
		s.DoJSON("POST", "/api/latest/fleet/spec/teams", tmSpecReq, http.StatusOK, &tmSpecResp)
		teamID, ok := tmSpecResp.TeamIDsByName["Pirates"]
		require.True(t, ok)
		team := fleet.Team{ID: teamID, Name: "Pirates"}
		checkTeam(t, &team, nil) // newly created team has empty mdm config

		t.Cleanup(func() {
			require.NoError(t, s.ds.DeleteTeam(ctx, team.ID))
		})

		// TODO: Add cases for other team MDM config (e.g., macos settings, macos updates,
		// migration) and for other permutations of starting values (see app config tests above).
		cases := []struct {
			name           string
			mdm            *fleet.TeamSpecMDM
			expectedStatus int
		}{
			{
				"mdm empty",
				&fleet.TeamSpecMDM{},
				http.StatusOK,
			},
			{
				"mdm all zero values",
				&fleet.TeamSpecMDM{
					EnableDiskEncryption: optjson.SetBool(false),
					MacOSSetup: fleet.MacOSSetup{
						BootstrapPackage:            optjson.SetString(""),
						EnableEndUserAuthentication: false,
						MacOSSetupAssistant:         optjson.SetString(""),
					},
				},
				http.StatusOK,
			},
			{
				"bootstrap package",
				&fleet.TeamSpecMDM{
					MacOSSetup: fleet.MacOSSetup{
						BootstrapPackage: optjson.SetString("some-package"),
					},
				},
				// bootstrap package requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
			{
				"setup assistant",
				&fleet.TeamSpecMDM{
					MacOSSetup: fleet.MacOSSetup{
						MacOSSetupAssistant: optjson.SetString("some-setup-assistant"),
					},
				},
				// setup assistant requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
			{
				"enable disk encryption",
				&fleet.TeamSpecMDM{
					EnableDiskEncryption: optjson.SetBool(true),
				},
				// disk encryption does not require mdm enabled and configured
				http.StatusOK,
			},
			{
				"enable recovery lock password",
				&fleet.TeamSpecMDM{
					EnableRecoveryLockPassword: optjson.SetBool(true),
				},
				// recovery lock password requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
			// Ian - this test still passes, that is, returns 4xx – perhaps related to one of the endpoints we still need to update
			{
				"enable end user auth",
				&fleet.TeamSpecMDM{
					MacOSSetup: fleet.MacOSSetup{
						EnableEndUserAuthentication: true,
					},
				},
				// disk encryption requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
		}

		for _, c := range cases {
			// TODO: Add tests for other combinations of mac and windows mdm enabled/disabled
			t.Run(c.name, func(t *testing.T) {
				checkAppConfig(t, false, false) // both mac and windows mdm disabled

				tmSpecReq = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
					Name: team.Name,
					MDM:  *c.mdm,
				}}}
				s.DoJSON("POST", "/api/latest/fleet/spec/teams", tmSpecReq, c.expectedStatus, &tmSpecResp)

				if c.expectedStatus == http.StatusOK {
					checkTeam(t, &team, c.mdm)
				} else {
					checkTeam(t, &team, nil)
				}
			})
		}
	})
}

// ///////////////////////////////////////////////////////////////////////////
// Common helpers

func (s *integrationMDMTestSuite) TestCustomConfigurationWebURL() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)

	s.enableABM(t.Name())
	var lastSubmittedProfile *godep.Profile
	configurationWebURLShouldBeEmpty := true
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)

		switch r.URL.Path {
		case "/server/devices", "/devices/sync":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: "FAKE-1",
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
					{
						SerialNumber: "FAKE-2",
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			assert.NoError(t, err)
		case "/profile":
			lastSubmittedProfile = &godep.Profile{}
			rawProfile, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			err = json.Unmarshal(rawProfile, lastSubmittedProfile)
			assert.NoError(t, err)

			// check that the urls are not empty and equal
			assert.NotEmpty(t, lastSubmittedProfile.URL)
			if configurationWebURLShouldBeEmpty {
				assert.Empty(t, lastSubmittedProfile.ConfigurationWebURL)
			} else {
				assert.NotEmpty(t, lastSubmittedProfile.ConfigurationWebURL)
				assert.Equal(t, lastSubmittedProfile.URL, lastSubmittedProfile.ConfigurationWebURL)
			}
			err = encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			assert.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var prof profileAssignmentReq
			assert.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = map[string]string{
				prof.Devices[0]: string(fleet.DEPAssignProfileResponseSuccess),
			}
			encoder := json.NewEncoder(w)
			err = encoder.Encode(resp)
			assert.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		}
	}))

	// run once to ingest the devices
	s.runDEPSchedule()

	// disable first to make sure we start in the desired state
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"macos_setup": {
				"enable_end_user_authentication": false
			}
		}
	}`), http.StatusOK, &acResp)

	// configure end-user authentication globally
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "https://localhost:8080",
				"idp_name": "SimpleSAML",
				"metadata_url": "%s"
			},
			"macos_setup": {
				"enable_end_user_authentication": true
			}
		}
	}`, testSAMLIDPMetadataURL)), http.StatusOK, &acResp)

	// assign the DEP profile and assert that contains the right values for the URL
	configurationWebURLShouldBeEmpty = false
	s.runWorker()
	require.NotNil(t, lastSubmittedProfile)
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, acResp.ServerSettings.ServerURL+"/mdm/sso")

	// trying to set a custom configuration_web_url fails because end user authentication is enabled
	customSetupAsst := `{"configuration_web_url": "https://foo.example.com"}`
	var globalAsstResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(customSetupAsst),
	}, http.StatusUnprocessableEntity, &globalAsstResp)

	// disable end user authentication
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "",
				"idp_name": "",
				"metadata_url": ""
			},
			"macos_setup": {
				"enable_end_user_authentication": false
			}
		}
	}`), http.StatusOK, &acResp)

	// assign the DEP profile and assert that contains the right values for the URL
	configurationWebURLShouldBeEmpty = true
	s.runWorker()
	require.NotNil(t, lastSubmittedProfile)
	require.Contains(t, lastSubmittedProfile.URL, acResp.ServerSettings.ServerURL+"/api/mdm/apple/enroll?token=")

	// setting a custom configuration_web_url succeeds because user authentication is disabled
	globalAsstResp = createMDMAppleSetupAssistantResponse{}
	configurationWebURLShouldBeEmpty = false
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(customSetupAsst),
	}, http.StatusOK, &globalAsstResp)

	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
	require.NotNil(t, lastSubmittedProfile)
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, "https://foo.example.com")

	// try to enable end user auth again, it fails because configuration_web_url is set
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "https://localhost:8080",
				"idp_name": "SimpleSAML",
				"metadata_url": "%s"
			},
			"macos_setup": {
				"enable_end_user_authentication": true
			}
		}
	}`, testSAMLIDPMetadataURL)), http.StatusUnprocessableEntity, &acResp)

	// create a team via spec
	teamSpecs := map[string]any{
		"specs": []any{
			map[string]any{
				"name": t.Name(),
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_end_user_authentication": false,
					},
				},
			},
		},
	}
	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)
	teamID := applyResp.TeamIDsByName[t.Name()]

	// transfer a host to the team to ensure all ABM calls are made
	h, err := s.ds.HostByIdentifier(context.Background(), "FAKE-1")
	require.NoError(t, err)
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{TeamID: &teamID, HostIDs: []uint{h.ID}}, http.StatusOK, &addHostsToTeamResponse{})

	// re-set the global state to configure MDM SSO
	err = s.ds.DeleteMDMAppleSetupAssistant(context.Background(), nil)
	require.NoError(t, err)
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "https://localhost:8080",
				"idp_name": "SimpleSAML",
				"metadata_url": "%s"
			},
			"macos_setup": {
				"enable_end_user_authentication": true
			}
		}
	}`, testSAMLIDPMetadataURL)), http.StatusOK, &acResp)

	// enable end user auth
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": t.Name(),
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_end_user_authentication": true,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)
	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, acResp.ServerSettings.ServerURL+"/mdm/sso")

	// trying to set a custom configuration_web_url fails because end user authentication is enabled
	var tmAsstResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &teamID,
		Name:              t.Name(),
		EnrollmentProfile: json.RawMessage(customSetupAsst),
	}, http.StatusUnprocessableEntity, &tmAsstResp)

	// disable end user auth
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": t.Name(),
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_end_user_authentication": false,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// assign the DEP profile and assert that contains the right values for the URL
	configurationWebURLShouldBeEmpty = true
	s.runWorker()
	require.Contains(t, lastSubmittedProfile.URL, acResp.ServerSettings.ServerURL+"/api/mdm/apple/enroll?token=")

	// setting configuration_web_url succeeds because end user authentication is disabled
	tmAsstResp = createMDMAppleSetupAssistantResponse{}
	configurationWebURLShouldBeEmpty = false
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &teamID,
		Name:              t.Name(),
		EnrollmentProfile: json.RawMessage(customSetupAsst),
	}, http.StatusOK, &tmAsstResp)

	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, "https://foo.example.com")

	// try to enable end user auth again, it fails because configuration_web_url is set
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": t.Name(),
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_end_user_authentication": true,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, &applyResp)

	t.Cleanup(func() {
		acResp.MDM.MacOSSetup.EnableEndUserAuthentication = false
		acResp.MDM.MacOSSetup.LockEndUserInfo = optjson.SetBool(false)
		err := s.ds.SaveAppConfig(context.Background(), &acResp.AppConfig)
		require.NoError(t, err)
	})
}

func (s *integrationMDMTestSuite) TestHostsCantTurnMDMOff() {
	t := s.T()
	winHost, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	r := s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", winHost.ID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), fleet.CantTurnOffMDMForWindowsHostsMessage)
}

func (s *integrationMDMTestSuite) TestTeamLabelsTeamDeletion() {
	t := s.T()

	test.CreateInsertGlobalVPPToken(t, s.ds)

	t1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "t1",
	})
	require.NoError(t, err)
	t2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "t2",
	})
	require.NoError(t, err)

	newHost := func(name string, platform string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(t.Context(), &fleet.Host{
			OsqueryHostID: new(t.Name() + name),
			NodeKey:       new(t.Name() + name),
			UUID:          uuid.New().String(),
			Hostname:      fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:      platform,
			TeamID:        teamID,

			LabelUpdatedAt: time.Now().Add(-2 * time.Hour),
		})
		require.NoError(t, err)
		return h
	}
	// Create host on team t1.
	macOST1 := newHost("macOST1", "darwin", &t1.ID)
	// Create host on team t2.
	macOST2 := newHost("macOST2", "darwin", &t2.ID)
	// Create host on "No team".
	ubuntuHostNoTeam := newHost("linuxHostNoTeam", "ubuntu", nil)

	// Mock live queries.
	s.lq.On("QueriesForHost", macOST1.ID).Return(map[string]string{}, nil)
	s.lq.On("QueriesForHost", macOST2.ID).Return(map[string]string{}, nil)
	s.lq.On("QueriesForHost", ubuntuHostNoTeam.ID).Return(map[string]string{}, nil)

	// Create label on team t1.
	l1t1, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "l1t1",
		Query:               "SELECT t1;",
		TeamID:              &t1.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	// Create a label on team t2.
	l2t2, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "l2t2",
		Query:               "SELECT t2;",
		TeamID:              &t2.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	// Create global label.
	globalLabel, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "global",
		Query:               "SELECT global;",
		TeamID:              nil,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	// Set label membership for l2t2.
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithLabelResults(macOST2, map[uint]*bool{
		l2t2.ID:        new(true),
		globalLabel.ID: new(true),
	}), http.StatusOK, &distributedResp)

	// Set label membership for l1t1.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithLabelResults(macOST1, map[uint]*bool{
		l1t1.ID:        new(true),
		globalLabel.ID: new(true),
	}), http.StatusOK, &distributedResp)

	// Create an installer on t1 that references l1t1.
	payloadRubyTm1 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "install",
		Filename:         "ruby.deb",
		SelfService:      false,
		TeamID:           &t1.ID,
		LabelsIncludeAny: []string{l1t1.Name},
		Platform:         "linux",
	}
	s.uploadSoftwareInstaller(t, payloadRubyTm1, http.StatusOK, "")

	// Create a VPP app on t1 that references l1t1.
	_, err = s.ds.InsertVPPAppWithTeam(t.Context(), &fleet.VPPApp{
		Name:             "App123 " + t.Name(),
		BundleIdentifier: "bid_" + t.Name(),
		AdamID:           "adam_test_vpp2",
		Platform:         fleet.MacOSPlatform,
		ValidatedLabels: &fleet.LabelIdentsWithScope{
			LabelScope: fleet.LabelScopeIncludeAny,
			ByName: map[string]fleet.LabelIdent{
				l1t1.Name: {
					LabelID:   l1t1.ID,
					LabelName: l1t1.Name,
				},
			},
		},
	}, &t1.ID, nil)
	require.NoError(t, err)

	// Create an Apple configuration profile, a Windows profile and a declaration on t1 that references l1t1.
	s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{
		ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
			{DisplayName: "N1", Profile: mobileconfigForTestWithContent("N1", "I1", "com.test.profile.content", "test inner type", "test inner name"), LabelsIncludeAll: []string{l1t1.Name}},
			{DisplayName: "N2", Profile: syncMLForTest("./Foo/Bar"), LabelsIncludeAll: []string{l1t1.Name}},
			{DisplayName: "N3", Profile: declarationForTest("D1"), LabelsIncludeAll: []string{l1t1.Name}},
		},
	}, http.StatusNoContent, "team_id", fmt.Sprint(t1.ID))

	// Create in-house app in team t1 that references l1t1.
	payload := &fleet.UploadSoftwareInstallerPayload{
		TeamID:           &t1.ID,
		Filename:         "ipa_test2.ipa",
		Version:          "1.0.0",
		StorageID:        uuid.New().String(),
		SelfService:      true,
		LabelsIncludeAny: []string{l1t1.Name},
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

	// Create a policy in t1 that references l1t1.
	_, err = s.ds.NewTeamPolicy(t.Context(), t1.ID, nil, fleet.PolicyPayload{
		Name:             "p1t1",
		Query:            "SELECT 1;",
		Platform:         "darwin",
		LabelsIncludeAny: []string{l1t1.Name},
	})
	require.NoError(t, err)

	// Create a query in t1 that references l1t1.
	_, err = s.ds.NewQuery(
		t.Context(),
		&fleet.Query{
			Name:    "TestQueryTeamPolicy",
			Query:   "SELECT 2;",
			Saved:   true,
			Logging: fleet.LoggingSnapshot,
			TeamID:  &t1.ID,
			LabelsIncludeAny: []fleet.LabelIdent{
				{
					LabelID:   l1t1.ID,
					LabelName: l1t1.Name,
				},
			},
		},
	)
	require.NoError(t, err)

	// Delete team t1.
	delResp := deleteTeamResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", t1.ID), nil, http.StatusOK, &delResp)

	// Check label l1t1 on t1 has been removed.
	_, _, err = s.ds.Label(t.Context(), l1t1.ID, fleet.TeamFilter{})
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// Make sure l2t2 in t2 is unaffected.
	_, _, err = s.ds.Label(t.Context(), l2t2.ID, fleet.TeamFilter{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	require.NoError(t, err)

	// Make sure label membership for l1t1 is gone.
	hostLabels, err := s.ds.ListLabelsForHost(t.Context(), macOST1.ID)
	require.NoError(t, err)
	require.Len(t, hostLabels, 1)
	require.Equal(t, globalLabel.ID, hostLabels[0].ID)

	// Make sure label membership of macOST2 is unaffected.
	hostLabels, err = s.ds.ListLabelsForHost(t.Context(), macOST2.ID)
	require.NoError(t, err)
	require.Len(t, hostLabels, 2)
	sort.Slice(hostLabels, func(i, j int) bool {
		return hostLabels[i].ID < hostLabels[j].ID
	})
	require.Equal(t, l2t2.ID, hostLabels[0].ID)
	require.Equal(t, globalLabel.ID, hostLabels[1].ID)
}

func (s *integrationMDMTestSuite) TestTeamLabelsAssociationsCheck() {
	t := s.T()

	test.CreateInsertGlobalVPPToken(t, s.ds)

	t1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "t1",
	})
	require.NoError(t, err)
	t2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "t2",
	})
	require.NoError(t, err)

	// Create label on team t1.
	l1t1, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "l1t1",
		Query:               "SELECT t1;",
		TeamID:              &t1.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	// Create label on team t2.
	l2t2, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "l2t2",
		Query:               "SELECT t2;",
		TeamID:              &t2.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	// Create global label.
	globalLabel, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:   "global",
		Query:  "SELECT global;",
		TeamID: nil,
	})
	require.NoError(t, err)

	t.Run("1. policy labels assignment checks", func(t *testing.T) {
		// 1.A.1 Attempt to create global policy that references l1t1 (should fail).
		var gpResp fleet.GlobalPolicyResponse
		s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
			Name:             "All teams policy",
			Query:            "SELECT 1;",
			LabelsIncludeAny: []string{l1t1.Name, globalLabel.Name},
		}, http.StatusBadRequest, &gpResp)
		gpResp = fleet.GlobalPolicyResponse{}
		s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
			Name:             "All teams policy",
			Query:            "SELECT 1;",
			LabelsExcludeAny: []string{globalLabel.Name, l1t1.Name},
		}, http.StatusBadRequest, &gpResp)

		// 1.A.2 Attempt to create a global policy with global labels (should succeed).
		gpResp = fleet.GlobalPolicyResponse{}
		s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
			Name:             "All teams policy",
			Query:            "SELECT 1;",
			LabelsIncludeAny: []string{globalLabel.Name},
		}, http.StatusOK, &gpResp)
		globalPolicyID := gpResp.Policy.ID
		gpResp = fleet.GlobalPolicyResponse{}
		s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
			Name:             "All teams policy 2",
			Query:            "SELECT 1;",
			LabelsExcludeAny: []string{globalLabel.Name},
		}, http.StatusOK, &gpResp)

		// 1.A.3 Attempt to modify a global policy with team labels (should fail).
		mgpr := &fleet.ModifyGlobalPolicyRequest{
			Name:             new("newName1"),
			LabelsIncludeAny: []string{l1t1.Name},
		}
		patchPol1 := &fleet.ModifyGlobalPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicyID), mgpr, http.StatusBadRequest, patchPol1)
		mgpr = &fleet.ModifyGlobalPolicyRequest{
			Name:             new("newName1"),
			LabelsExcludeAny: []string{l1t1.Name},
		}
		patchPol1 = &fleet.ModifyGlobalPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicyID), mgpr, http.StatusBadRequest, patchPol1)

		// 1.A.4 Attempt to modify a global policy with global labels (should succeed).
		mgpr = &fleet.ModifyGlobalPolicyRequest{
			Name:             new("newName1"),
			LabelsIncludeAny: []string{globalLabel.Name},
		}
		patchPol1 = &fleet.ModifyGlobalPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicyID), mgpr, http.StatusOK, patchPol1)
		mgpr = &fleet.ModifyGlobalPolicyRequest{
			Name:             new("newName2"),
			LabelsIncludeAny: []string{},
			LabelsExcludeAny: []string{globalLabel.Name},
		}
		patchPol1 = &fleet.ModifyGlobalPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicyID), mgpr, http.StatusOK, patchPol1)

		// 1.B.1 Attempt to create a team policy that references l2t2 (should fail).
		tpResp := fleet.TeamPolicyResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", t1.ID), fleet.TeamPolicyRequest{
			Name:             "t1 policy",
			Query:            "SELECT 1;",
			LabelsIncludeAny: []string{globalLabel.Name, l2t2.Name},
		}, http.StatusBadRequest, &tpResp)
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", t1.ID), fleet.TeamPolicyRequest{
			Name:             "t1 policy exclude",
			Query:            "SELECT 1;",
			LabelsExcludeAny: []string{globalLabel.Name, l2t2.Name},
		}, http.StatusBadRequest, &tpResp)

		// 1.B.2 Attempt to create a team policy with a global label and same team label (should succeed).
		tpResp = fleet.TeamPolicyResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", t1.ID), fleet.TeamPolicyRequest{
			Name:             "t1 policy",
			Query:            "SELECT 1;",
			LabelsIncludeAny: []string{globalLabel.Name, l1t1.Name},
		}, http.StatusOK, &tpResp)
		teamPolicyID := tpResp.Policy.ID
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", t1.ID), fleet.TeamPolicyRequest{
			Name:             "t1 policy 2",
			Query:            "SELECT 1;",
			LabelsExcludeAny: []string{globalLabel.Name, l1t1.Name},
		}, http.StatusOK, &tpResp)

		// 1.B.3 Attempt to edit a team policy to reference l2t2 (should fail; label is outside team).
		mtplr := fleet.ModifyTeamPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, teamPolicyID), fleet.ModifyTeamPolicyRequest{
			LabelsIncludeAny: []string{l2t2.Name},
		}, http.StatusBadRequest, &mtplr)
		mtplr = fleet.ModifyTeamPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, teamPolicyID), fleet.ModifyTeamPolicyRequest{
			LabelsIncludeAny: []string{},
			LabelsExcludeAny: []string{l2t2.Name},
		}, http.StatusBadRequest, &mtplr)

		// 1.B.3 Attempt to edit a team policy to reference a team label on the same team (should succeed).
		mtplr = fleet.ModifyTeamPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, teamPolicyID), fleet.ModifyTeamPolicyRequest{
			LabelsIncludeAny: []string{l1t1.Name},
		}, http.StatusOK, &mtplr)
		mtplr = fleet.ModifyTeamPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, teamPolicyID), fleet.ModifyTeamPolicyRequest{
			LabelsIncludeAny: []string{},
			LabelsExcludeAny: []string{l1t1.Name, globalLabel.Name},
		}, http.StatusOK, &mtplr)

		// 1.C.1 Attempt to create a "No team" policy that references l1t1 (should fail).
		tpResp = fleet.TeamPolicyResponse{}
		s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", fleet.TeamPolicyRequest{
			Name:             "no team policy",
			Query:            "SELECT 1;",
			LabelsIncludeAny: []string{globalLabel.Name, l2t2.Name},
		}, http.StatusBadRequest, &tpResp)
		s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", fleet.TeamPolicyRequest{
			Name:             "no team policy exclude",
			Query:            "SELECT 1;",
			LabelsExcludeAny: []string{globalLabel.Name, l2t2.Name},
		}, http.StatusBadRequest, &tpResp)

		// 1.B.2 Attempt to create a "No team" policy with a global label (should succeed).
		tpResp = fleet.TeamPolicyResponse{}
		s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", fleet.TeamPolicyRequest{
			Name:             "no team policy",
			Query:            "SELECT 1;",
			LabelsIncludeAny: []string{globalLabel.Name},
		}, http.StatusOK, &tpResp)
		noTeamPolicyID := tpResp.Policy.ID
		s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", fleet.TeamPolicyRequest{
			Name:             "no team policy 2",
			Query:            "SELECT 1;",
			LabelsExcludeAny: []string{globalLabel.Name},
		}, http.StatusOK, &tpResp)

		// 1.B.3 Attempt to edit a "No team" policy with a team policy that references l2t2 (should fail).
		mtplr = fleet.ModifyTeamPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/0/policies/%d", noTeamPolicyID), fleet.ModifyTeamPolicyRequest{
			LabelsIncludeAny: []string{l2t2.Name},
		}, http.StatusBadRequest, &mtplr)
		mtplr = fleet.ModifyTeamPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/0/policies/%d", noTeamPolicyID), fleet.ModifyTeamPolicyRequest{
			LabelsIncludeAny: []string{},
			LabelsExcludeAny: []string{l2t2.Name},
		}, http.StatusBadRequest, &mtplr)

		// 1.B.3 Attempt to edit a team policy to reference a global label (should succeed).
		mtplr = fleet.ModifyTeamPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/0/policies/%d", noTeamPolicyID), fleet.ModifyTeamPolicyRequest{
			LabelsIncludeAny: []string{globalLabel.Name},
		}, http.StatusOK, &mtplr)
		mtplr = fleet.ModifyTeamPolicyResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/0/policies/%d", noTeamPolicyID), fleet.ModifyTeamPolicyRequest{
			LabelsIncludeAny: []string{},
			LabelsExcludeAny: []string{globalLabel.Name},
		}, http.StatusOK, &mtplr)
	})

	t.Run("2. query labels assignment checks", func(t *testing.T) {
		// 2.A.1 Attempt to create global query with team labels (should fail).
		var createQueryResp fleet.CreateQueryResponse
		reqQuery := &fleet.QueryPayload{
			Name:             new("All teams query"),
			Query:            new("SELECT 1;"),
			LabelsIncludeAny: []string{l1t1.Name},
		}
		s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusBadRequest, &createQueryResp)

		// 2.A.2 Attempt to create global query with global label (should succeed).
		createQueryResp = fleet.CreateQueryResponse{}
		reqQuery = &fleet.QueryPayload{
			Name:             new("All teams query"),
			Query:            new("SELECT 1;"),
			LabelsIncludeAny: []string{globalLabel.Name},
		}
		s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
		globalQueryID := createQueryResp.Query.ID

		// 2.A.3 Attempt to edit global query with team label (should fail).
		modifyQueryResp := fleet.ModifyQueryResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", globalQueryID), fleet.ModifyQueryRequest{
			LabelsIncludeAny: []string{l1t1.Name},
		}, http.StatusBadRequest, &modifyQueryResp)

		// 2.A.4 Attempt to edit global query with global label (should succeed).
		modifyQueryResp = fleet.ModifyQueryResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", globalQueryID), fleet.ModifyQueryRequest{
			LabelsIncludeAny: []string{globalLabel.Name},
		}, http.StatusOK, &modifyQueryResp)

		// 2.B.1 Attempt to create a team query with a label of another team (should fail).
		createQueryResp = fleet.CreateQueryResponse{}
		reqQuery = &fleet.QueryPayload{
			Name:             new("Team one query"),
			Query:            new("SELECT 1;"),
			LabelsIncludeAny: []string{l2t2.Name},

			TeamID: &t1.ID,
		}
		s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusBadRequest, &createQueryResp)

		// 2.B.2 Attempt to create a team query with a label of the same team (should succeed).
		createQueryResp = fleet.CreateQueryResponse{}
		reqQuery = &fleet.QueryPayload{
			Name:             new("Team one query"),
			Query:            new("SELECT 1;"),
			LabelsIncludeAny: []string{l1t1.Name, globalLabel.Name},

			TeamID: &t1.ID,
		}
		s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
		team1LabelID := createQueryResp.Query.ID

		// 2.A.3 Attempt to edit a team query with a label of another team (should fail).
		modifyQueryResp = fleet.ModifyQueryResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", team1LabelID), fleet.ModifyQueryRequest{
			LabelsIncludeAny: []string{l2t2.Name, globalLabel.Name},
		}, http.StatusBadRequest, &modifyQueryResp)

		// 2.A.4 Attempt to edit team query with a label of the same team (should succeed).
		modifyQueryResp = fleet.ModifyQueryResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", team1LabelID), fleet.ModifyQueryRequest{
			LabelsIncludeAny: []string{l1t1.Name},
		}, http.StatusOK, &modifyQueryResp)
	})

	t.Run("3. configuration profiles assignment check", func(t *testing.T) {
		// NOTE: Not testing the API endpoint POST /api/latest/fleet/mdm/profiles used by the UI
		// because of time constraints (we haven't yet implemented test utilities for multipart uploads).

		// Attempt to create a team profile with labels from another team (should fail).
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{
			ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
				{
					DisplayName: "N1", Profile: mobileconfigForTestWithContent("N1", "I1", "com.test.profile.content", "test inner type", "test inner name"),
					LabelsIncludeAll: []string{l2t2.Name},
				},
			},
		}, http.StatusBadRequest, "team_id", fmt.Sprint(t1.ID))
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{
			ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
				{
					DisplayName: "N2", Profile: syncMLForTest("./Foo/Bar"),
					LabelsIncludeAll: []string{l2t2.Name},
				},
			},
		}, http.StatusBadRequest, "team_id", fmt.Sprint(t1.ID))
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{
			ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
				{
					DisplayName: "N3", Profile: declarationForTest("D1"),
					LabelsIncludeAll: []string{l2t2.Name},
				},
			},
		}, http.StatusBadRequest, "team_id", fmt.Sprint(t1.ID))

		// Attempt to create a profile with a label on the same team (should succeed).
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{
			ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
				{
					DisplayName: "N3", Profile: declarationForTest("D1"),
					LabelsIncludeAll: []string{l1t1.Name},
				},
			},
		}, http.StatusNoContent, "team_id", fmt.Sprint(t1.ID))

		// Attempt to create a profile in "No team" with a label that belongs to a team (should fail).
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{
			ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
				{
					DisplayName: "N1", Profile: mobileconfigForTestWithContent("N1", "I1", "com.test.profile.content", "test inner type", "test inner name"),
					LabelsIncludeAll: []string{l1t1.Name},
				},
			},
		}, http.StatusBadRequest)

		// Attempt to create a profile in "No team" with a global label (should succeed).
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{
			ConfigurationProfiles: []fleet.BatchModifyMDMConfigProfilePayload{
				{
					DisplayName: "N1", Profile: mobileconfigForTestWithContent("N1", "I1", "com.test.profile.content", "test inner type", "test inner name"),
					LabelsIncludeAll: []string{globalLabel.Name},
				},
			},
		}, http.StatusNoContent)
	})

	t.Run("4. software installers assignment check", func(t *testing.T) {
		// Attempt to create a software installer with a label of another team (should fail).
		payloadRubyTm1 := &fleet.UploadSoftwareInstallerPayload{
			InstallScript:    "install",
			Filename:         "ruby.deb",
			SelfService:      false,
			TeamID:           &t1.ID,
			LabelsIncludeAny: []string{l2t2.Name},
			Platform:         "linux",
		}
		s.uploadSoftwareInstaller(t, payloadRubyTm1, http.StatusBadRequest, "")

		// Attempt to create a software installer with a label on the same team (should succeed).
		payloadRubyTm1 = &fleet.UploadSoftwareInstallerPayload{
			InstallScript:    "install",
			Filename:         "ruby.deb",
			SelfService:      false,
			TeamID:           &t1.ID,
			LabelsIncludeAny: []string{l1t1.Name},
			Platform:         "linux",
		}
		s.uploadSoftwareInstaller(t, payloadRubyTm1, http.StatusOK, "")
		swTitleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")

		// Attempt to edit a software installer with a label from another team (should fail).
		updatePayload := &fleet.UpdateSoftwareInstallerPayload{
			TitleID:          swTitleID,
			TeamID:           &t1.ID,
			LabelsIncludeAny: []string{l2t2.Name},
		}
		s.updateSoftwareInstaller(t, updatePayload, http.StatusBadRequest, "")

		// Attempt to edit a software installer with a label on the same team (should succeed).
		updatePayload = &fleet.UpdateSoftwareInstallerPayload{
			TitleID:          swTitleID,
			TeamID:           &t1.ID,
			LabelsIncludeAny: []string{l1t1.Name, globalLabel.Name},
		}
		s.updateSoftwareInstaller(t, updatePayload, http.StatusOK, "")

		// Attempt to create a software installer in "No team" with a team label (should fail).
		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallScript:    "install",
			Filename:         "dummy_installer.pkg",
			Title:            "DummyApp",
			TeamID:           nil,
			LabelsExcludeAny: []string{l1t1.Name},
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "")

		// Attempt to create a software installer in "No team" with a global label (should succeed).
		payload = &fleet.UploadSoftwareInstallerPayload{
			InstallScript:    "install",
			Filename:         "dummy_installer.pkg",
			Title:            "DummyApp",
			TeamID:           nil,
			LabelsExcludeAny: []string{globalLabel.Name},
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
		swTitleID = getSoftwareTitleID(t, s.ds, payload.Title, "apps")

		// Attempt to edit a software installer in "No team" with a team label (should fail).
		updatePayload = &fleet.UpdateSoftwareInstallerPayload{
			TitleID:          swTitleID,
			TeamID:           nil,
			LabelsIncludeAny: []string{l1t1.Name, globalLabel.Name},
		}
		s.updateSoftwareInstaller(t, updatePayload, http.StatusBadRequest, "")

		// Attempt to edit a software installer in "No team" with a global label (should succeed).
		updatePayload = &fleet.UpdateSoftwareInstallerPayload{
			TitleID:          swTitleID,
			TeamID:           nil,
			LabelsIncludeAny: []string{globalLabel.Name},
		}
		s.updateSoftwareInstaller(t, updatePayload, http.StatusOK, "")
	})

	t.Run("5. vpp apps assignment checks", func(t *testing.T) {
		// Set up VPP token
		orgName := "Fleet Device Management Inc."
		token := "mycooltoken"
		expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
		expDate := expTime.Format(fleet.VPPTimeFormat)
		tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
		dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
		var validToken uploadVPPTokenResponse
		s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

		// Get the token
		var resp getVPPTokensResponse
		s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
		require.NoError(t, resp.Err)

		// Associate team to the VPP token.
		var resPatchVPP patchVPPTokensTeamsResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{t1.ID}}, http.StatusOK, &resPatchVPP)

		// Attempt to add an app store app on a team with a label from another team (should fail).
		var addAppResp addAppStoreAppResponse
		addAppReq := &addAppStoreAppRequest{
			TeamID:           &t1.ID,
			AppStoreID:       "1",
			SelfService:      true,
			LabelsIncludeAny: []string{l2t2.Name},
		}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", addAppReq, http.StatusBadRequest, &addAppResp)

		// Attempt to add an app store app on a team with a label on the same team (should succeed).
		addAppReq = &addAppStoreAppRequest{
			TeamID:           &t1.ID,
			AppStoreID:       "1",
			SelfService:      true,
			LabelsIncludeAny: []string{l1t1.Name},
		}
		addAppResp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", addAppReq, http.StatusOK, &addAppResp)

		// Associate all teams token.
		resPatchVPP = patchVPPTokensTeamsResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

		// Attempt to add an app store app to "No team" with a team label (should fail).
		addAppResp = addAppStoreAppResponse{}
		addAppReq = &addAppStoreAppRequest{
			TeamID:           nil,
			AppStoreID:       "1",
			SelfService:      true,
			LabelsIncludeAny: []string{l2t2.Name},
		}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", addAppReq, http.StatusBadRequest, &addAppResp)

		// Attempt to add an app store app to "No team" with a global label (should succeed).
		addAppResp = addAppStoreAppResponse{}
		addAppReq = &addAppStoreAppRequest{
			TeamID:           nil,
			AppStoreID:       "1",
			SelfService:      true,
			LabelsIncludeAny: []string{globalLabel.Name},
		}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", addAppReq, http.StatusOK, &addAppResp)
	})

	t.Run("6. in-house apps assignment checks", func(t *testing.T) {
		// Attempt to create in-house app in team t1 that references l2t2 (should fail).
		payload := &fleet.UploadSoftwareInstallerPayload{
			TeamID:           &t1.ID,
			Filename:         "ipa_test2.ipa",
			Version:          "1.0.0",
			StorageID:        uuid.New().String(),
			SelfService:      true,
			LabelsIncludeAny: []string{l2t2.Name},
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusBadRequest, "")

		// Attempt to create in-house app in team t1 that references l1t1 (should succeed).
		payload = &fleet.UploadSoftwareInstallerPayload{
			TeamID:           &t1.ID,
			Filename:         "ipa_test2.ipa",
			Version:          "1.0.0",
			StorageID:        uuid.New().String(),
			SelfService:      true,
			LabelsIncludeAny: []string{l1t1.Name},
		}
		s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")

		// Not testing edit as this was tested in (4) above.
		// This is used to populate the in house apps tables.
	})

	// Make sure we can delete the teams given all the entities created.
	err = s.ds.DeleteTeam(t.Context(), t1.ID)
	require.NoError(t, err)
	err = s.ds.DeleteTeam(t.Context(), t2.ID)
	require.NoError(t, err)
}

func (s *integrationMDMTestSuite) TestStagedUpload() {
	t := s.T()
	if s.stagedUploadStore == nil {
		t.Skip("set S3_STORAGE_TEST to run staged upload tests")
	}

	presign := func(target fleet.StagedUploadTarget, size int64, wantStatus int, wantErr string) *fleet.StagedUpload {
		var resp createStagedUploadResponse
		res := s.Do("POST", "/api/latest/fleet/staged_upload", createStagedUploadRequest{Target: target, Size: size}, wantStatus)
		if wantStatus != http.StatusOK {
			require.Contains(t, extractServerErrorText(res.Body), wantErr)
			return nil
		}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))
		require.NotEmpty(t, resp.UploadID)
		require.WithinDuration(t, time.Now().Add(fleet.StagedUploadURLExpiry), resp.ExpiresAt, time.Minute)
		return resp.StagedUpload
	}
	stage := func(target fleet.StagedUploadTarget, content []byte) string {
		up := presign(target, int64(len(content)), http.StatusOK, "")
		req, err := http.NewRequest(http.MethodPut, up.URL, bytes.NewReader(content))
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		return up.UploadID
	}
	requireGone := func(uploadID string, gone bool) {
		_, _, err := s.stagedUploadStore.Get(t.Context(), uploadID)
		require.Equal(t, gone, fleet.IsNotFound(err), err)
	}
	finalizeBootstrap := func(fields map[string]string, pkg []byte, dryRun bool, wantStatus int, wantErr string) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		for k, v := range fields {
			require.NoError(t, w.WriteField(k, v))
		}
		if pkg != nil {
			fw, err := w.CreateFormFile("package", "pkg.pkg")
			require.NoError(t, err)
			_, err = fw.Write(pkg)
			require.NoError(t, err)
		}
		require.NoError(t, w.Close())
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/bootstrap", b.Bytes(), wantStatus, map[string]string{
			"Content-Type":  w.FormDataContentType(),
			"Authorization": "Bearer " + s.token,
		}, "dry_run", strconv.FormatBool(dryRun))
		if wantErr != "" {
			require.Contains(t, extractServerErrorText(res.Body), wantErr)
		}
	}
	readTestdata := func(elem ...string) []byte {
		b, err := os.ReadFile(filepath.Join(append([]string{"testdata"}, elem...)...))
		require.NoError(t, err)
		return b
	}

	// presign errors
	var cfgResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &cfgResp)
	maxSize := cfgResp.MaxSoftwarePackageSize
	presign(fleet.StagedUploadTargetSoftwarePackage, maxSize+1, http.StatusBadRequest, "The maximum file size is")
	presign(fleet.StagedUploadTargetSoftwarePackage, 0, http.StatusBadRequest, "size is required")
	presign("nope", 1, http.StatusBadRequest, "Unsupported upload target")

	observer := &fleet.User{Name: "Staged Observer", Email: "staged-observer@example.com", GlobalRole: new(fleet.RoleObserver)}
	require.NoError(t, observer.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(t.Context(), observer)
	require.NoError(t, err)
	adminToken := s.token
	s.setTokenForTest(t, observer.Email, test.GoodPassword)
	presign(fleet.StagedUploadTargetSoftwarePackage, 1, http.StatusForbidden, "")
	presign(fleet.StagedUploadTargetBootstrapPackage, 1, http.StatusForbidden, "")
	s.token = adminToken

	// bootstrap package
	signedPkg := readTestdata("bootstrap-packages", "signed.pkg")
	finalizeBootstrap(map[string]string{"upload_id": uuid.NewString()}, signedPkg, false, http.StatusBadRequest, "only one of package or upload_id")
	finalizeBootstrap(map[string]string{"upload_id": uuid.NewString()}, nil, false, http.StatusBadRequest, "filename multipart field is required")
	finalizeBootstrap(map[string]string{"upload_id": uuid.NewString(), "filename": "bad:name.pkg"}, nil, false, http.StatusBadRequest, "invalid characters")
	finalizeBootstrap(map[string]string{"upload_id": "../bootstrap-packages/x", "filename": "pkg.pkg"}, nil, false, http.StatusBadRequest, "Invalid upload_id")
	finalizeBootstrap(map[string]string{"upload_id": uuid.NewString(), "filename": "pkg.pkg"}, nil, false, http.StatusBadRequest, "Upload not found")

	unsignedID := stage(fleet.StagedUploadTargetBootstrapPackage, readTestdata("bootstrap-packages", "unsigned.pkg"))
	finalizeBootstrap(map[string]string{"upload_id": unsignedID, "filename": "pkg.pkg"}, nil, false, http.StatusBadRequest, "file is not signed")
	requireGone(unsignedID, false)

	uploadID := stage(fleet.StagedUploadTargetBootstrapPackage, signedPkg)
	finalizeBootstrap(map[string]string{"upload_id": uploadID, "filename": "staged.pkg"}, nil, true, http.StatusOK, "")
	s.Do("GET", "/api/latest/fleet/bootstrap/0/metadata", nil, http.StatusNotFound)
	requireGone(uploadID, true)

	uploadID = stage(fleet.StagedUploadTargetBootstrapPackage, signedPkg)
	finalizeBootstrap(map[string]string{"upload_id": uploadID, "filename": "staged.pkg"}, nil, false, http.StatusOK, "")
	requireGone(uploadID, true)
	var metaResp bootstrapPackageMetadataResponse
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/0/metadata", nil, http.StatusOK, &metaResp)
	require.Equal(t, "staged.pkg", metaResp.MDMAppleBootstrapPackage.Name)
	wantSha := sha256.Sum256(signedPkg)
	require.Equal(t, wantSha[:], metaResp.MDMAppleBootstrapPackage.Sha256)
	s.lastActivityMatches(fleet.ActivityTypeAddedBootstrapPackage{}.ActivityName(),
		`{"bootstrap_package_name": "staged.pkg", "team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, 0)
	s.Do("DELETE", "/api/latest/fleet/bootstrap/0", nil, http.StatusOK)

	// software package add and edit
	uploadID = stage(fleet.StagedUploadTargetSoftwarePackage, readTestdata("software-installers", "ruby.deb"))
	rubyFile, err := fleet.NewKeepFileReader(filepath.Join("testdata", "software-installers", "ruby.deb"))
	require.NoError(t, err)
	defer rubyFile.Close()
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID: 1, TeamID: new(uint(0)), StagedUploadID: uploadID, Filename: "ruby.deb", InstallerFile: rubyFile,
	}, http.StatusBadRequest, "only one of software or upload_id")
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		StagedUploadID: uploadID, Filename: "ruby.deb", InstallScript: "install", TeamID: new(uint(0)),
	}, http.StatusOK, "")
	requireGone(uploadID, true)
	titleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")
	var titleResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp, "fleet_id", "0")
	require.Equal(t, "ruby.deb", titleResp.SoftwareTitle.SoftwarePackage.Name)

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "fleet_id", "0")

	// edit replaces the package with the staged bytes
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{Filename: "script.sh", TeamID: new(uint(0))}, http.StatusOK, "")
	var scriptTitleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &scriptTitleID, `SELECT title_id FROM software_installers WHERE filename = 'script.sh'`)
	})
	newScript := []byte("#!/bin/sh\necho staged\n")
	uploadID = stage(fleet.StagedUploadTargetSoftwarePackage, newScript)
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID: scriptTitleID, TeamID: new(uint(0)), StagedUploadID: uploadID, Filename: "script.sh",
	}, http.StatusOK, "")
	requireGone(uploadID, true)
	var storageID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &storageID, `SELECT storage_id FROM software_installers WHERE title_id = ?`, scriptTitleID)
	})
	newSha := sha256.Sum256(newScript)
	require.Equal(t, fmt.Sprintf("%x", newSha), storageID)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", scriptTitleID), nil, http.StatusNoContent, "fleet_id", "0")

	// in-house apps take the same staged add and edit paths
	uploadID = stage(fleet.StagedUploadTargetSoftwarePackage, readTestdata("software-installers", "ipa_test.ipa"))
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		StagedUploadID: uploadID, Filename: "ipa_test.ipa", TeamID: new(uint(0)),
	}, http.StatusOK, "")
	requireGone(uploadID, true)
	var ipaTitleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &ipaTitleID, `SELECT title_id FROM in_house_apps WHERE filename = 'ipa_test.ipa'`)
	})
	// Same app, different bytes: copy the archive and add an entry.
	ipa := readTestdata("software-installers", "ipa_test.ipa")
	zr, err := zip.NewReader(bytes.NewReader(ipa), int64(len(ipa)))
	require.NoError(t, err)
	var ipaV2 bytes.Buffer
	zw := zip.NewWriter(&ipaV2)
	for _, f := range zr.File {
		require.NoError(t, zw.Copy(f))
	}
	_, err = zw.Create("extra.txt")
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	uploadID = stage(fleet.StagedUploadTargetSoftwarePackage, ipaV2.Bytes())
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID: ipaTitleID, TeamID: new(uint(0)), StagedUploadID: uploadID, Filename: "ipa_test_v2.ipa",
	}, http.StatusOK, "")
	requireGone(uploadID, true)
	var ipaFilename, ipaStorageID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return q.QueryRowxContext(t.Context(), `SELECT filename, storage_id FROM in_house_apps WHERE title_id = ?`, ipaTitleID).Scan(&ipaFilename, &ipaStorageID)
	})
	ipaV2Sha := sha256.Sum256(ipaV2.Bytes())
	require.Equal(t, "ipa_test_v2.ipa", ipaFilename)
	require.Equal(t, fmt.Sprintf("%x", ipaV2Sha), ipaStorageID)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", ipaTitleID), nil, http.StatusNoContent, "fleet_id", "0")
}
