package service

// macOS setup tests for the MDM suite.
//
// Belongs here: bootstrap packages and their status, the EULA, macOS setup settings and the setup
// assistant, and the managed local account.
//
// Does not belong here: setup experience scripts and software, and their release
// (integration_mdm_setup_experience_test.go), ADE profile assignment (integration_mdm_dep_test.go),
// and end user authentication (integration_mdm_sso_test.go).

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/bindata"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestBootstrapPackage() {
	t := s.T()

	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", name))
		require.NoError(t, err)
		return b
	}
	invalidPkg := read("invalid.tar.gz")
	unsignedPkg := read("unsigned.pkg")
	wrongTOCPkg := read("wrong-toc.pkg")
	signedPkg := read("signed.pkg")

	// dry run query param parsing test
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: signedPkg, Name: "pkg.pkg"}, http.StatusOK, "", true)
	// verify no bootstrap package was uploaded
	var metadataResp bootstrapPackageMetadataResponse
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/0/metadata", nil, http.StatusNotFound, &metadataResp)

	// empty bootstrap package
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{}, http.StatusBadRequest, "package multipart field is required", false)
	// no name
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: signedPkg}, http.StatusBadRequest, "package multipart field is required", false)
	// invalid
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: invalidPkg, Name: "invalid.tar.gz"}, http.StatusBadRequest, "invalid file type", false)
	// invalid names
	for _, char := range file.InvalidMacOSChars {
		s.uploadBootstrapPackage(
			&fleet.MDMAppleBootstrapPackage{
				Bytes: signedPkg,
				Name:  fmt.Sprintf("invalid_%c_name.pkg", char),
			}, http.StatusBadRequest, "", false,
		)
	}
	// unsigned
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: unsignedPkg, Name: "pkg.pkg"}, http.StatusBadRequest, "file is not signed", false)
	// wrong TOC
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: wrongTOCPkg, Name: "pkg.pkg"}, http.StatusBadRequest, "invalid package", false)
	// not a Distribution package
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: read("not-distribution-signed.pkg"), Name: "pkg.pkg"}, http.StatusBadRequest,
		fleet.BootstrapPkgNotDistributionErrMsg, false)
	// successfully upload a package
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: signedPkg, Name: "pkg.pkg", TeamID: 0}, http.StatusOK, "", false)
	// check the activity log
	s.lastActivityMatches(
		fleet.ActivityTypeAddedBootstrapPackage{}.ActivityName(),
		`{"bootstrap_package_name": "pkg.pkg", "team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)

	// get package metadata
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/0/metadata", nil, http.StatusOK, &metadataResp)
	require.Equal(t, "pkg.pkg", metadataResp.MDMAppleBootstrapPackage.Name)
	require.NotEmpty(t, metadataResp.MDMAppleBootstrapPackage.Sha256)
	require.NotEmpty(t, metadataResp.MDMAppleBootstrapPackage.Token)

	// download a package, wrong token
	var downloadResp downloadBootstrapPackageResponse
	s.DoJSON("GET", "/api/latest/fleet/bootstrap?token=bad", nil, http.StatusNotFound, &downloadResp)

	resp := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/bootstrap?token=%s", metadataResp.MDMAppleBootstrapPackage.Token), nil, http.StatusOK)
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, signedPkg, respBytes)

	// missing package
	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/1/metadata", nil, http.StatusNotFound, &metadataResp)

	// delete package
	var deleteResp deleteBootstrapPackageResponse
	s.DoJSON("DELETE", "/api/latest/fleet/bootstrap/0", nil, http.StatusOK, &deleteResp)
	// check the activity log
	s.lastActivityMatches(
		fleet.ActivityTypeDeletedBootstrapPackage{}.ActivityName(),
		`{"bootstrap_package_name": "pkg.pkg", "team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)

	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/0/metadata", nil, http.StatusNotFound, &metadataResp)
	// trying to delete again is a bad request
	s.DoJSON("DELETE", "/api/latest/fleet/bootstrap/0", nil, http.StatusNotFound, &deleteResp)
}

func (s *integrationMDMTestSuite) TestBootstrapPackageReplace() {
	t := s.T()
	ctx := t.Context()

	pkg, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	tmDetails := fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name)
	getMeta := func() fleet.MDMAppleBootstrapPackage {
		var resp bootstrapPackageMetadataResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/bootstrap/%d/metadata", tm.ID), nil, http.StatusOK, &resp)
		return *resp.MDMAppleBootstrapPackage
	}

	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "old.pkg", TeamID: tm.ID}, http.StatusOK, "", false)
	oldMeta := getMeta()
	require.Equal(t, "old.pkg", oldMeta.Name)

	// uploading over an existing package replaces it without a delete first
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "new.pkg", TeamID: tm.ID}, http.StatusOK, "", false)
	newMeta := getMeta()
	require.Equal(t, "new.pkg", newMeta.Name)
	require.NotEqual(t, oldMeta.Token, newMeta.Token)

	// the audit log reads as it did when the old package was deleted before the upload
	addedID := s.lastActivityMatches(fleet.ActivityTypeAddedBootstrapPackage{}.ActivityName(),
		`{"bootstrap_package_name": "new.pkg", `+tmDetails[1:], 0)
	deletedID := s.lastActivityOfTypeMatches(fleet.ActivityTypeDeletedBootstrapPackage{}.ActivityName(),
		`{"bootstrap_package_name": "old.pkg", `+tmDetails[1:], 0)
	require.Less(t, deletedID, addedID)

	// a dry run doesn't replace anything
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "dry.pkg", TeamID: tm.ID}, http.StatusOK, "", true)
	require.Equal(t, newMeta.Token, getMeta().Token)
}

func (s *integrationMDMTestSuite) TestBootstrapPackageStatus() {
	t := s.T()

	abmOrgName := "abm_org"
	s.enableABM(abmOrgName)

	pkg, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)

	// upload a bootstrap package for "no team"
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "pkg.pkg", TeamID: 0}, http.StatusOK, "", false)

	// get package metadata
	var metadataResp bootstrapPackageMetadataResponse
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/0/metadata", nil, http.StatusOK, &metadataResp)
	globalBootstrapPackage := metadataResp.MDMAppleBootstrapPackage

	// create a team and upload a bootstrap package for that team.
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	// upload a bootstrap package for the team
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "pkg.pkg", TeamID: team.ID}, http.StatusOK, "", false)

	// get package metadata
	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/bootstrap/%d/metadata", team.ID), nil, http.StatusOK, &metadataResp)
	teamBootstrapPackage := metadataResp.MDMAppleBootstrapPackage

	type deviceWithResponse struct {
		bootstrapResponse string
		device            *mdmtest.TestAppleMDMClient
	}

	// Note: The responses specified here are not a 1:1 mapping of the possible responses specified
	// by Apple. Instead `enrollAndCheckBootstrapPackage` below uses them to simulate scenarios in
	// which a device may or may not send a response. For example, "Offline" means that no response
	// will be sent by the device, which should in turn be interpreted by Fleet as "Pending"). See
	// https://developer.apple.com/documentation/devicemanagement/installenterpriseapplicationresponse
	//
	// Below:
	// - Acknowledge means the device will enroll and acknowledge the request to install the bp
	// - Error means that the device will enroll and fail to install the bp
	// - Offline means that the device will enroll but won't acknowledge nor fail the bp request
	// - Pending means that the device won't enroll at all
	mdmEnrollInfo := mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}
	noTeamDevices := []deviceWithResponse{
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Offline", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Offline", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Pending", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Pending", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Skipped", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
	}

	teamDevices := []deviceWithResponse{
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Offline", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Pending", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Skipped", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
	}

	expectedSerialsByTeamAndStatus := make(map[uint]map[fleet.MDMBootstrapPackageStatus][]string)
	expectedSerialsByTeamAndStatus[0] = map[fleet.MDMBootstrapPackageStatus][]string{
		fleet.MDMBootstrapPackageInstalled: {noTeamDevices[0].device.SerialNumber, noTeamDevices[1].device.SerialNumber, noTeamDevices[2].device.SerialNumber},
		fleet.MDMBootstrapPackageFailed:    {noTeamDevices[3].device.SerialNumber},
		fleet.MDMBootstrapPackagePending:   {noTeamDevices[4].device.SerialNumber, noTeamDevices[5].device.SerialNumber, noTeamDevices[6].device.SerialNumber, noTeamDevices[7].device.SerialNumber},
	}
	expectedSerialsByTeamAndStatus[team.ID] = map[fleet.MDMBootstrapPackageStatus][]string{
		fleet.MDMBootstrapPackageInstalled: {teamDevices[0].device.SerialNumber, teamDevices[1].device.SerialNumber},
		fleet.MDMBootstrapPackageFailed:    {teamDevices[2].device.SerialNumber, teamDevices[3].device.SerialNumber, teamDevices[4].device.SerialNumber},
		fleet.MDMBootstrapPackagePending:   {teamDevices[5].device.SerialNumber, teamDevices[6].device.SerialNumber},
	}

	// for good measure, add a couple of manually enrolled hosts
	createHostThenEnrollMDM(s.ds, s.server.URL, t)
	createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create a non-macOS host
	winHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID: new("non-macos-host"),
		NodeKey:       new("non-macos-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.non.macos", t.Name()),
		Platform:      "windows",
		MDM:           fleet.MDMHostData{},
	})
	require.NoError(t, err)

	err = s.ds.SetOrUpdateMDMData(context.Background(), winHost.ID, false, true, s.server.URL+apple_mdm.MDMPath, true, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)

	// create a host that's not enrolled into MDM
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID: new("not-mdm-enrolled"),
		NodeKey:       new("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	mockRespDevices := noTeamDevices
	s.mockDEPResponse(abmOrgName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			assert.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			assert.NoError(t, err)
		case "/server/devices":
			err := encoder.Encode(godep.DeviceResponse{})
			assert.NoError(t, err)
		case "/devices/sync":
			depResp := []godep.Device{}
			for _, gd := range mockRespDevices {
				device := godep.Device{SerialNumber: gd.device.SerialNumber}
				if gd.bootstrapResponse == "Skipped" {
					device.MDMMigrationDeadline = new(time.Now())
				}
				depResp = append(depResp, device)
			}
			err := encoder.Encode(godep.DeviceResponse{Devices: depResp})
			assert.NoError(t, err)
		case "/profile/devices":
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// trigger a dep sync
	s.runDEPSchedule()

	var summaryResp getMDMAppleBootstrapPackageSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/summary", nil, http.StatusOK, &summaryResp)
	// We don't count the "skipped" device as it is pending migration and will not install the bootstrap package
	require.Equal(t, fleet.MDMAppleBootstrapPackageSummary{Pending: uint(len(noTeamDevices) - 1)}, summaryResp.MDMAppleBootstrapPackageSummary) //nolint:gosec // dismiss G115

	var lhr listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &lhr, "team_id", "0", "bootstrap_package", "pending")
	require.Len(t, lhr.Hosts, len(noTeamDevices)-1)

	// set the default bm assignment to `team`
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_business_manager": [{
			  "organization_name": %q,
			  "macos_team": %q,
			  "ios_team": %q,
			  "ipados_team": %q
			}]
		}
	}`, abmOrgName, team.Name, team.Name, team.Name)), http.StatusOK, &acResp)
	t.Cleanup(func() {
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"apple_business_manager": []
			}
		}`), http.StatusOK, &acResp)
	})

	// trigger a dep sync
	mockRespDevices = teamDevices
	s.runDEPSchedule()

	summaryResp = getMDMAppleBootstrapPackageSummaryResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/bootstrap/summary?team_id=%d", team.ID), nil, http.StatusOK, &summaryResp)
	require.Equal(t, fleet.MDMAppleBootstrapPackageSummary{Pending: uint(len(teamDevices) - 1)}, summaryResp.MDMAppleBootstrapPackageSummary) //nolint:gosec // dismiss G115

	mockErrorChain := []mdm.ErrorChain{
		{ErrorCode: 12021, ErrorDomain: "MCMDMErrorDomain", LocalizedDescription: "Unknown command", USEnglishDescription: "Unknown command"},
	}

	// devices send their responses
	enrollAndCheckBootstrapPackage := func(d *deviceWithResponse, bp *fleet.MDMAppleBootstrapPackage) {
		err := d.device.Enroll() // queues DEP post-enrollment worker job
		require.NoError(t, err)

		s.awaitRunAppleMDMWorkerSchedule()
		// process worker jobs
		s.runWorker()

		cmd, err := d.device.Idle()
		require.NoError(t, err)
		gotBootstrapPackage := false
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

			// if the command is to install the bootstrap package
			if manifest := fullCmd.Command.InstallEnterpriseApplication.Manifest; manifest != nil {
				gotBootstrapPackage = true
				require.Equal(t, "InstallEnterpriseApplication", cmd.Command.RequestType)
				require.NotNil(t, manifest)
				require.Equal(t, "software-package", manifest.ManifestItems[0].Assets[0].Kind)
				wantURL, err := bp.URL(s.server.URL)
				require.NoError(t, err)
				require.Equal(t, wantURL, manifest.ManifestItems[0].Assets[0].URL)

				// respond to the command accordingly
				switch d.bootstrapResponse {
				case "Acknowledge":
					cmd, err = d.device.Acknowledge(cmd.CommandUUID)
					require.NoError(t, err)
					continue
				case "Error":
					cmd, err = d.device.Err(cmd.CommandUUID, mockErrorChain)
					require.NoError(t, err)
					continue
				case "Offline":
					// host is offline, can't process any more commands
					cmd = nil
					continue
				}
			}
			cmd, err = d.device.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		require.Equal(t, d.bootstrapResponse != "Skipped", gotBootstrapPackage)
	}

	for _, d := range noTeamDevices {
		dd := d
		if dd.bootstrapResponse != "Pending" {
			enrollAndCheckBootstrapPackage(&dd, globalBootstrapPackage)
		}
	}

	for _, d := range teamDevices {
		dd := d
		if dd.bootstrapResponse != "Pending" {
			enrollAndCheckBootstrapPackage(&dd, teamBootstrapPackage)
		}
	}

	checkHostDetails := func(t *testing.T, hostID uint, hostUUID string, expectedStatus fleet.MDMBootstrapPackageStatus) {
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
		require.NotNil(t, hostResp.Host)
		require.NotNil(t, hostResp.Host.MDM.MacOSSetup)
		require.Equal(t, "pkg.pkg", hostResp.Host.MDM.MacOSSetup.BootstrapPackageName)
		require.Equal(t, expectedStatus, hostResp.Host.MDM.MacOSSetup.BootstrapPackageStatus)
		if expectedStatus == fleet.MDMBootstrapPackageFailed {
			require.Equal(t, hostResp.Host.MDM.MacOSSetup.Detail, apple_mdm.FmtErrorChain(mockErrorChain))
		} else {
			require.Empty(t, hostResp.Host.MDM.MacOSSetup.Detail)
		}
		require.Nil(t, hostResp.Host.MDM.MacOSSetup.Result)

		var hostByIdentifierResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", hostUUID), nil, http.StatusOK, &hostByIdentifierResp)
		require.NotNil(t, hostByIdentifierResp.Host)
		require.NotNil(t, hostByIdentifierResp.Host.MDM.MacOSSetup)
		require.Equal(t, expectedStatus, hostByIdentifierResp.Host.MDM.MacOSSetup.BootstrapPackageStatus)
		if expectedStatus == fleet.MDMBootstrapPackageFailed {
			require.Equal(t, hostResp.Host.MDM.MacOSSetup.Detail, apple_mdm.FmtErrorChain(mockErrorChain))
		} else {
			require.Empty(t, hostResp.Host.MDM.MacOSSetup.Detail)
		}
		require.Nil(t, hostResp.Host.MDM.MacOSSetup.Result)
	}

	checkHostAPIs := func(t *testing.T, status fleet.MDMBootstrapPackageStatus, teamID *uint) {
		var expectedSerials []string
		if teamID == nil {
			expectedSerials = expectedSerialsByTeamAndStatus[0][status]
		} else {
			expectedSerials = expectedSerialsByTeamAndStatus[*teamID][status]
		}

		listHostsPath := fmt.Sprintf("/api/latest/fleet/hosts?bootstrap_package=%s", status)
		if teamID != nil {
			listHostsPath += fmt.Sprintf("&team_id=%d", *teamID)
		}
		var listHostsResp listHostsResponse
		s.DoJSON("GET", listHostsPath, nil, http.StatusOK, &listHostsResp)
		require.NotNil(t, listHostsResp.Hosts)
		require.Len(t, listHostsResp.Hosts, len(expectedSerials))

		gotHostsBySerial := make(map[string]fleet.HostResponse)
		for _, h := range listHostsResp.Hosts {
			gotHostsBySerial[h.HardwareSerial] = h
		}
		require.Len(t, gotHostsBySerial, len(expectedSerials))

		for _, serial := range expectedSerials {
			require.Contains(t, gotHostsBySerial, serial)
			h := gotHostsBySerial[serial]

			// pending hosts don't have an UUID yet.
			if h.UUID != "" {
				checkHostDetails(t, h.ID, h.UUID, status)
			}
		}

		countPath := fmt.Sprintf("/api/latest/fleet/hosts/count?bootstrap_package=%s", status)
		if teamID != nil {
			countPath += fmt.Sprintf("&team_id=%d", *teamID)
		}
		var countResp countHostsResponse
		s.DoJSON("GET", countPath, nil, http.StatusOK, &countResp)
		require.Equal(t, len(expectedSerials), countResp.Count)
	}

	// check summary no team hosts
	summaryResp = getMDMAppleBootstrapPackageSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/summary", nil, http.StatusOK, &summaryResp)
	require.Equal(t, fleet.MDMAppleBootstrapPackageSummary{
		Installed: uint(3),
		Pending:   uint(4),
		Failed:    uint(1),
	}, summaryResp.MDMAppleBootstrapPackageSummary)

	checkHostAPIs(t, fleet.MDMBootstrapPackageInstalled, nil)
	checkHostAPIs(t, fleet.MDMBootstrapPackagePending, nil)
	checkHostAPIs(t, fleet.MDMBootstrapPackageFailed, nil)

	// check team summary
	summaryResp = getMDMAppleBootstrapPackageSummaryResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/bootstrap/summary?team_id=%d", team.ID), nil, http.StatusOK, &summaryResp)
	require.Equal(t, fleet.MDMAppleBootstrapPackageSummary{
		Installed: uint(2),
		Pending:   uint(2),
		Failed:    uint(3),
	}, summaryResp.MDMAppleBootstrapPackageSummary)

	checkHostAPIs(t, fleet.MDMBootstrapPackageInstalled, &team.ID)
	checkHostAPIs(t, fleet.MDMBootstrapPackagePending, &team.ID)
	checkHostAPIs(t, fleet.MDMBootstrapPackageFailed, &team.ID)
}

func (s *integrationMDMTestSuite) TestEULA() {
	t := s.T()
	pdfBytes := []byte("%PDF-1.pdf-contents")
	pdfName := "eula.pdf"
	pdfHash := sha256.Sum256(pdfBytes)

	// trying to get metadata about an EULA that hasn't been uploaded yet is an error
	metadataResp := getMDMEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata", nil, http.StatusNotFound, &metadataResp)

	// trying to upload a file that is not a PDF fails
	s.uploadEULA(&fleet.MDMEULA{Bytes: []byte("should-fail"), Name: "should-fail.pdf"}, http.StatusBadRequest, "The file must be a PDF (.pdf).")
	// trying to upload an empty file fails
	s.uploadEULA(&fleet.MDMEULA{Bytes: []byte{}, Name: "should-fail.pdf"}, http.StatusBadRequest, "The file must be a PDF (.pdf).")

	// file larger than the max EULA size should be rejected by the request size limit
	largePDF := append(
		[]byte("%PDF-1.7\n"),
		bytes.Repeat([]byte("A"), int(fleet.MaxEULASize)+1-len("%PDF-1.7\n"))...,
	)

	s.uploadEULA(&fleet.MDMEULA{Bytes: largePDF, Name: "oversize.pdf"}, http.StatusRequestEntityTooLarge, "Request exceeds the max size limit")

	// Test with MaxRequestBodySize set to unlimited (-1) - the endpoint still enforces the EULA request body limit
	oldLimit := platform_http.MaxRequestBodySize
	platform_http.MaxRequestBodySize = -1
	largePDFUnlimited := append(
		[]byte("%PDF-1.7\n"),
		bytes.Repeat([]byte("A"), int(fleet.MaxEULASize)+1-len("%PDF-1.7\n"))...,
	)
	s.uploadEULA(&fleet.MDMEULA{Bytes: largePDFUnlimited, Name: "oversize_unlimited.pdf"}, http.StatusRequestEntityTooLarge, "Request exceeds the max size limit")
	platform_http.MaxRequestBodySize = oldLimit

	// Test with MaxRequestBodySize larger than MaxEULASize - should reject at MaxRequestBodySize
	platform_http.MaxRequestBodySize = 50 * units.MiB
	largePDFLarge := append(
		[]byte("%PDF-1.7\n"),
		bytes.Repeat([]byte("A"), int(50*units.MiB)+1-len("%PDF-1.7\n"))...,
	)
	s.uploadEULA(&fleet.MDMEULA{Bytes: largePDFLarge, Name: "oversize_large.pdf"}, http.StatusRequestEntityTooLarge, "Request exceeds the max size limit")
	platform_http.MaxRequestBodySize = oldLimit

	// admin is able to upload a new EULA
	// a path in the uploaded name is dropped, since generate-gitops writes the
	// stored name to disk
	s.uploadEULA(&fleet.MDMEULA{Bytes: pdfBytes, Name: `..\..\` + pdfName}, http.StatusOK, "")
	s.lastActivityMatches(fleet.ActivityTypeAddedEndUserAgreement{}.ActivityName(), `{"platform":"darwin"}`, 0)

	// get EULA metadata
	metadataResp = getMDMEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata", nil, http.StatusOK, &metadataResp)
	require.NotEmpty(t, metadataResp.MDMEULA.Token)
	require.NotEmpty(t, metadataResp.MDMEULA.CreatedAt)
	require.Equal(t, pdfName, metadataResp.MDMEULA.Name)
	require.Equal(t, pdfHash[:], metadataResp.MDMEULA.Sha256)
	eulaToken := metadataResp.Token

	// download EULA
	resp := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", eulaToken), nil, http.StatusOK)
	require.EqualValues(t, len(pdfBytes), resp.ContentLength)
	require.Equal(t, "application/pdf", resp.Header.Get("content-type"))
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, pdfBytes, respBytes)

	// try to download EULA with a bad token
	var downloadResp downloadBootstrapPackageResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/bad-token", nil, http.StatusNotFound, &downloadResp)

	// trying to upload any EULA without deleting the previous one first results in an error
	s.uploadEULA(&fleet.MDMEULA{Bytes: pdfBytes, Name: "should-fail.pdf"}, http.StatusConflict, "")

	// delete EULA
	var deleteResp deleteMDMEULAResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", eulaToken), nil, http.StatusOK, &deleteResp)
	s.lastActivityMatches(fleet.ActivityTypeDeletedEndUserAgreement{}.ActivityName(), `{"platform":"darwin"}`, 0)
	metadataResp = getMDMEULAMetadataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", eulaToken), nil, http.StatusNotFound, &metadataResp)
	// trying to delete again is a bad request
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", eulaToken), nil, http.StatusNotFound, &deleteResp)

	// a name with nothing usable left after sanitizing is stored under the default name
	s.uploadEULA(&fleet.MDMEULA{Bytes: pdfBytes, Name: ".."}, http.StatusOK, "")
	metadataResp = getMDMEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata", nil, http.StatusOK, &metadataResp)
	require.Equal(t, fleet.MDMEULADefaultDarwinFileName, metadataResp.Name)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", metadataResp.Token), nil, http.StatusOK, &deleteResp)
}

func (s *integrationMDMTestSuite) TestWindowsEULA() {
	t := s.T()
	clearEULAs := func() {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), "DELETE FROM eulas")
			return err
		})
	}
	// Earlier tests can leave a macOS EULA behind and this test uploads its own, while tests after it expect no
	// agreement, so start clean and clean up even on failure.
	clearEULAs()
	t.Cleanup(clearEULAs)

	// The terms page needs the bundled templates, which require `-tags full`.
	defer func() {
		if panicVal := recover(); panicVal != nil {
			if msg, ok := panicVal.(string); ok && strings.Contains(msg, "Assets may not be used when running Fleet as a library") {
				t.Skip("skipping, test will fail due to assets not built (requires '-tags full')")
			}
			panic(panicVal)
		}
	}()
	_, _ = bindata.Asset("check if assets are build")

	mdBytes := []byte("# Acme terms\n\nBy continuing you accept the **Acme** device policy.\n\n[Read more](https://acme.example/policy)\n")
	mdName := "terms.md"
	mdHash := sha256.Sum256(mdBytes)

	tosURL := microsoft_mdm.MDE2TOSPath + "?api-version=1.0&redirect_uri=ms-appx-web%3a%2f%2fMicrosoft.AAD.BrokerPlugin&client-request-id=f2cf3127-1e80-4d73-965d-42a3b84bdb40"
	tosPage := func() string {
		resp := s.DoRaw("GET", tosURL, nil, http.StatusOK)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(b)
	}

	// nothing uploaded yet: metadata is a 404 and the device sees Fleet's default terms
	var metadataResp getMDMEULAMetadataResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata?platform=windows", nil, http.StatusNotFound, &metadataResp)
	require.Contains(t, tosPage(), "Fleet is open-source software")

	// validation: markdown only, non-empty, UTF-8, within the size limit
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: []byte("%PDF-1.7"), Name: "terms.pdf"}, http.StatusBadRequest, "must be a markdown (.md) file")
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: []byte("  \n\t"), Name: "terms.md"}, http.StatusBadRequest, "The file is empty")
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: []byte{0xff, 0xfe, 0x41}, Name: "terms.md"}, http.StatusBadRequest, "UTF-8")
	// an HTML block would be dropped with its text, and a page with nothing to read is useless
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: []byte("# Terms\n\n<div>\nClause 4 applies.\n</div>\n"), Name: "terms.md"},
		http.StatusBadRequest, "contains HTML")
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: []byte("<!-- draft -->\n"), Name: "terms.md"}, http.StatusBadRequest, "no text to show")
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: bytes.Repeat([]byte("Clause.\n"), 70_000), Name: "oversize.md"},
		http.StatusBadRequest, "512 KB or smaller")
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: bytes.Repeat([]byte("A"), 9000), Name: "terms.md"},
		http.StatusBadRequest, "line longer than 8 KB")

	// a dry run validates without storing anything, and a mistyped one doesn't save
	s.uploadEULAFile("windows", "?dry_run=treu", &fleet.MDMEULA{Bytes: mdBytes, Name: mdName}, http.StatusBadRequest, "failed to decode dry_run")
	s.uploadEULAFile("windows", "?dry_run=true", &fleet.MDMEULA{Bytes: []byte("<!-- draft -->\n"), Name: mdName}, http.StatusBadRequest, "no text to show")
	s.uploadEULAFile("windows", "?dry_run=true", &fleet.MDMEULA{Bytes: mdBytes, Name: mdName}, http.StatusOK, "")
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata?platform=windows", nil, http.StatusNotFound, &metadataResp)

	// upload; a path in the uploaded name is dropped, since generate-gitops
	// writes the stored name to disk
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: mdBytes, Name: `..\..\` + mdName}, http.StatusOK, "")
	s.lastActivityMatches(fleet.ActivityTypeAddedEndUserAgreement{}.ActivityName(), `{"platform":"windows"}`, 0)

	// metadata
	metadataResp = getMDMEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata?platform=windows", nil, http.StatusOK, &metadataResp)
	require.NotEmpty(t, metadataResp.Token)
	require.NotEmpty(t, metadataResp.CreatedAt)
	require.Equal(t, mdName, metadataResp.Name)
	require.Equal(t, mdHash[:], metadataResp.Sha256)
	// metadata reports the platform, since the endpoints serve both
	require.Equal(t, fleet.MDMEULAPlatformWindows, metadataResp.Platform)
	winToken := metadataResp.Token

	// download, as an attachment and never as a renderable page; like the macOS PDF, the token is enough
	resp := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", winToken), nil, http.StatusOK)
	require.EqualValues(t, len(mdBytes), resp.ContentLength)
	require.Equal(t, "text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment")
	require.Contains(t, resp.Header.Get("Content-Disposition"), mdName)
	require.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, mdBytes, got)

	// the device now sees the rendered agreement in place of the default terms,
	// with the link flattened to its text
	page := tosPage()
	require.Contains(t, page, `<div class="eula-custom"><h1>Acme terms</h1>`)
	require.Contains(t, page, "<strong>Acme</strong>")
	require.Contains(t, page, "Read more")
	require.NotContains(t, page, "acme.example")
	require.NotContains(t, page, "Fleet is open-source software")
	require.Contains(t, page, "IsAccepted=true")

	// replacing the agreement shows the new one on the next request, not the
	// previously rendered page
	var replaceResp deleteMDMEULAResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", winToken), nil, http.StatusOK, &replaceResp)
	// It has an HTML table with merged cells, as pandoc writes it from a Word document.
	revised := "# Revised terms\n\n<table style=\"width:84%;\">\n<tr><th colspan=\"2\">Coverage</th></tr>\n" +
		"<tr><td>Laptop</td><td><u>Yes</u></td></tr>\n</table>\n"
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: []byte(revised), Name: mdName}, http.StatusOK, "")
	page = tosPage()
	require.Contains(t, page, "<h1>Revised terms</h1>")
	require.Contains(t, page, `<th colspan="2">Coverage</th>`)
	require.Contains(t, page, "<td><u>Yes</u></td>")
	require.NotContains(t, page, "width:84%")
	require.NotContains(t, page, "Acme terms")
	metadataResp = getMDMEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata?platform=windows", nil, http.StatusOK, &metadataResp)
	winToken = metadataResp.Token

	// both agreements share the endpoints: the token picks the agreement, and the platform must be one of the two
	pdfBytes := []byte("%PDF-1.pdf-contents")
	s.uploadEULAFile("linux", "", &fleet.MDMEULA{Bytes: pdfBytes, Name: "eula.pdf"},
		http.StatusUnprocessableEntity, `must be "darwin" or "windows"`)
	// the platform can also come in the query string, as on metadata; the form and the query can't disagree
	s.uploadEULAFile("", "?platform=windows&dry_run=true", &fleet.MDMEULA{Bytes: mdBytes, Name: mdName}, http.StatusOK, "")
	s.uploadEULAFile("windows", "?platform=windows&dry_run=true", &fleet.MDMEULA{Bytes: mdBytes, Name: mdName}, http.StatusOK, "")
	s.uploadEULAFile("windows", "?platform=darwin&dry_run=true", &fleet.MDMEULA{Bytes: mdBytes, Name: mdName},
		http.StatusUnprocessableEntity, "different platforms")
	s.uploadEULAFile("darwin", "", &fleet.MDMEULA{Bytes: pdfBytes, Name: "eula.pdf"}, http.StatusOK, "")
	var macMeta getMDMEULAMetadataResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata", nil, http.StatusOK, &macMeta)
	require.Equal(t, fleet.MDMEULAPlatformDarwin, macMeta.Platform)
	macToken := macMeta.Token
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata?platform=darwin", nil, http.StatusOK, &macMeta)
	require.Equal(t, macToken, macMeta.Token)
	macEULAPath := fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", macToken)
	winEULAPath := fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", winToken)
	resp = s.DoRawNoAuth("GET", macEULAPath, nil, http.StatusOK)
	require.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
	resp = s.DoRawNoAuth("GET", winEULAPath, nil, http.StatusOK)
	require.Equal(t, "text/markdown; charset=utf-8", resp.Header.Get("Content-Type"))
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata?platform=linux", nil, http.StatusUnprocessableEntity, &macMeta)

	// downloading needs only the token, deleting still needs a login
	s.DoRawNoAuth("DELETE", winEULAPath, nil, http.StatusUnauthorized)

	// each agreement still needs its own platform's MDM, even with another platform's on
	appCfg, err := s.ds.AppConfig(t.Context())
	require.NoError(t, err)
	setMDM := func(ctx context.Context, apple, windows bool) {
		appCfg.MDM.EnabledAndConfigured = apple
		appCfg.MDM.WindowsEnabledAndConfigured = windows
		require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))
	}
	// t.Context() is canceled before cleanups run.
	t.Cleanup(func() { setMDM(context.Background(), true, true) })
	requireMDMOff := func(res *http.Response, msg string) {
		require.Contains(t, extractServerErrorText(res.Body), msg)
	}

	setMDM(t.Context(), true, false)
	requireMDMOff(s.DoRawNoAuth("GET", winEULAPath, nil, http.StatusBadRequest), fleet.WindowsMDMNotConfiguredMessage)
	requireMDMOff(s.Do("DELETE", winEULAPath, nil, http.StatusBadRequest), fleet.WindowsMDMNotConfiguredMessage)
	s.uploadWindowsEULA(&fleet.MDMEULA{Bytes: mdBytes, Name: mdName}, http.StatusBadRequest, fleet.WindowsMDMNotConfiguredMessage)
	s.DoRawNoAuth("GET", macEULAPath, nil, http.StatusOK)

	setMDM(t.Context(), false, true)
	requireMDMOff(s.DoRawNoAuth("GET", macEULAPath, nil, http.StatusBadRequest), fleet.MDMNotConfiguredMessage)
	requireMDMOff(s.Do("DELETE", macEULAPath, nil, http.StatusBadRequest), fleet.MDMNotConfiguredMessage)
	s.DoRawNoAuth("GET", winEULAPath, nil, http.StatusOK)
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata?platform=windows", nil, http.StatusOK, &metadataResp)
	setMDM(t.Context(), true, true)

	var deleteResp deleteMDMEULAResponse
	var macDelete deleteMDMEULAResponse

	// a dry run delete keeps the agreement, and still reports a token that matches nothing
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s?dry_run=true", winToken), nil, http.StatusOK, &deleteResp)
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata?platform=windows", nil, http.StatusOK, &metadataResp)
	s.DoJSON("DELETE", "/api/latest/fleet/setup_experience/eula/bad-token?dry_run=true", nil, http.StatusNotFound, &deleteResp)

	// delete the Windows agreement; macOS keeps its EULA and the device is back on the default terms
	s.DoJSON("DELETE", winEULAPath, nil, http.StatusOK, &deleteResp)
	s.lastActivityMatches(fleet.ActivityTypeDeletedEndUserAgreement{}.ActivityName(), `{"platform":"windows"}`, 0)
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata?platform=windows", nil, http.StatusNotFound, &metadataResp)
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata", nil, http.StatusOK, &macMeta)
	require.Contains(t, tosPage(), "Fleet is open-source software")

	// leave the suite as it was found
	s.DoJSON("DELETE", macEULAPath, nil, http.StatusOK, &macDelete)
}

func (s *integrationMDMTestSuite) TestMDMMacOSSetup() {
	t := s.T()

	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			assert.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			assert.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// setup test data
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "https://localhost:8080",
				"idp_name": "SimpleSAML",
				"metadata_url": "%s"
		      }
		}
	}`, testSAMLIDPMetadataURL)), http.StatusOK, &acResp)
	require.NotEmpty(t, acResp.MDM.EndUserAuthentication)

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	endUserAuthCases := []struct {
		raw            string
		expectedEUA    bool // expected end user authentication value
		expectedLEUI   bool // expected lock end user info value
		expectedStatus int  // expected HTTP status code from the API response
	}{
		{
			raw:            `"mdm": {}`,
			expectedEUA:    false,
			expectedLEUI:   false,
			expectedStatus: http.StatusOK,
		},
		{
			raw: `"mdm": {
				"macos_setup": {}
			}`,
			expectedEUA:    false,
			expectedLEUI:   false,
			expectedStatus: http.StatusOK,
		},
		{
			raw: `"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}`,
			expectedEUA:    true,
			expectedLEUI:   true, // enabling EUA auto-enables LockEndUserInfo
			expectedStatus: http.StatusOK,
		},
		{
			raw: `"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": true, "lock_end_user_info": false
				}
			}`,
			expectedEUA:    true,
			expectedLEUI:   false,
			expectedStatus: http.StatusOK,
		},
		{
			raw: `"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": true, "lock_end_user_info": true
				}
			}`,
			expectedEUA:    true,
			expectedLEUI:   true,
			expectedStatus: http.StatusOK,
		},
		{
			raw: `"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": false
				}
			}`,
			expectedEUA:    false,
			expectedLEUI:   false, // EUA->false clears LEUI as well
			expectedStatus: http.StatusOK,
		},
		{
			raw: `"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": false, "lock_end_user_info": true
				}
			}`,
			expectedEUA:    false,
			expectedLEUI:   false, // Returns an error because LEUI cannot be true if EUA is false
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}

	writeTmpJSON := func(t *testing.T, v any) string {
		tmpFile, err := os.CreateTemp(t.TempDir(), "*.json")
		require.NoError(t, err)
		err = json.NewEncoder(tmpFile).Encode(v)
		require.NoError(t, err)
		return tmpFile.Name()
	}

	mustReadFile := func(t *testing.T, path string) string {
		b, err := os.ReadFile(path)
		require.NoError(t, err)
		return string(b)
	}

	asstOk := writeTmpJSON(t, map[string]any{"ok": true})
	asstURL := writeTmpJSON(t, map[string]any{"url": "https://example.com"})
	asstAwait := writeTmpJSON(t, map[string]any{"await_device_configured": true})
	asstsByName := map[string]string{
		asstOk:    mustReadFile(t, asstOk),
		asstURL:   mustReadFile(t, asstURL),
		asstAwait: mustReadFile(t, asstAwait),
	}

	enableReleaseDeviceCases := []struct {
		enableRelease     *bool
		setupAssistant    string
		expectedRelease   bool
		expectedAssistant string
		expectedStatus    int
	}{
		{
			enableRelease:     nil,
			setupAssistant:    "",
			expectedRelease:   false,
			expectedAssistant: "",
			expectedStatus:    http.StatusOK,
		},
		{
			enableRelease:     new(true),
			setupAssistant:    "",
			expectedRelease:   true,
			expectedAssistant: "",
			expectedStatus:    http.StatusOK,
		},
		{
			enableRelease:     new(false),
			setupAssistant:    "",
			expectedRelease:   false,
			expectedAssistant: "",
			expectedStatus:    http.StatusOK,
		},
		{
			enableRelease:     new(false),
			setupAssistant:    asstURL,
			expectedRelease:   false,
			expectedAssistant: "",
			expectedStatus:    http.StatusUnprocessableEntity,
		},
		{
			enableRelease:     new(true),
			setupAssistant:    asstAwait,
			expectedRelease:   false,
			expectedAssistant: "",
			expectedStatus:    http.StatusUnprocessableEntity,
		},
	}

	t.Run("UpdateAppConfig", func(t *testing.T) {
		acResp := appConfigResponse{}
		path := "/api/latest/fleet/config"
		fmtJSON := func(s string) json.RawMessage {
			return json.RawMessage(fmt.Sprintf(`{
				%s
			}`, s))
		}

		// get the initial appconfig; enable end user authentication and release
		// device default is false
		s.DoJSON("GET", path, nil, http.StatusOK, &acResp)
		require.False(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.False(t, acResp.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

		for i, c := range endUserAuthCases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				acResp = appConfigResponse{}
				s.DoJSON("PATCH", path, fmtJSON(c.raw), c.expectedStatus, &acResp)
				if c.expectedStatus == http.StatusOK {
					require.Equal(t, c.expectedEUA, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
					require.Equal(t, c.expectedLEUI, acResp.MDM.MacOSSetup.LockEndUserInfo.Value)
				}

				acResp = appConfigResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &acResp)
				require.Equal(t, c.expectedEUA, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
				require.Equal(t, c.expectedLEUI, acResp.MDM.MacOSSetup.LockEndUserInfo.Value)
			})
		}

		for i, c := range enableReleaseDeviceCases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				macSetup := map[string]any{}
				if c.enableRelease != nil {
					macSetup["enable_release_device_manually"] = *c.enableRelease
				}
				if c.setupAssistant != "" {
					macSetup["macos_setup_assistant"] = c.setupAssistant
				}

				uploadSucceeded := true
				if c.setupAssistant != "" {
					s.Do("POST", "/api/v1/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
						Name:              c.setupAssistant,
						EnrollmentProfile: json.RawMessage(asstsByName[c.setupAssistant]),
					}, c.expectedStatus)
					if c.expectedStatus >= 300 {
						uploadSucceeded = false
					}
				}

				if uploadSucceeded {
					acResp = appConfigResponse{}
					s.DoJSON("PATCH", path,
						json.RawMessage(jsonMustMarshal(t, map[string]any{"mdm": map[string]any{"macos_setup": macSetup}})),
						c.expectedStatus, &acResp)
					require.Equal(t, c.expectedRelease, acResp.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
					require.Equal(t, c.expectedAssistant, acResp.MDM.MacOSSetup.MacOSSetupAssistant.Value)
				}

				acResp = appConfigResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &acResp)
				require.Equal(t, c.expectedRelease, acResp.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
				require.Equal(t, c.expectedAssistant, acResp.MDM.MacOSSetup.MacOSSetupAssistant.Value)
			})
		}
	})

	t.Run("UpdateTeamConfig", func(t *testing.T) {
		path := fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID)
		fmtJSON := `{
			"name": %q,
			%s
		}`

		// get the initial team config; enable end user authentication and release
		// device default is false
		teamResp := teamResponse{}
		s.DoJSON("GET", path, nil, http.StatusOK, &teamResp)
		require.False(t, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.False(t, teamResp.Team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

		for i, c := range endUserAuthCases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				teamResp = teamResponse{}
				s.DoJSON("PATCH", path, json.RawMessage(fmt.Sprintf(fmtJSON, tm.Name, c.raw)), c.expectedStatus, &teamResp)
				if c.expectedStatus == http.StatusOK {
					require.Equal(t, c.expectedEUA, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
					require.Equal(t, c.expectedLEUI, teamResp.Team.Config.MDM.MacOSSetup.LockEndUserInfo.Value)
				}

				teamResp = teamResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &teamResp)
				require.Equal(t, c.expectedEUA, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
				require.Equal(t, c.expectedLEUI, teamResp.Team.Config.MDM.MacOSSetup.LockEndUserInfo.Value)
			})
		}

		for i, c := range enableReleaseDeviceCases {
			expectedPatchStatus := c.expectedStatus
			if expectedPatchStatus == http.StatusOK {
				expectedPatchStatus = http.StatusNoContent
			}

			t.Run(strconv.Itoa(i), func(t *testing.T) {
				if c.setupAssistant != "" {
					s.Do("POST", "/api/v1/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
						TeamID:            &tm.ID,
						Name:              c.setupAssistant,
						EnrollmentProfile: json.RawMessage(asstsByName[c.setupAssistant]),
					}, c.expectedStatus)
					uploadSucceeded := c.expectedStatus < 300

					if uploadSucceeded {
						// use the apply team specs to set both the setup assistant and the
						// enable release at once
						macSetup := fleet.MacOSSetup{
							MacOSSetupAssistant: optjson.SetString(c.setupAssistant),
						}
						if c.enableRelease != nil {
							macSetup.EnableReleaseDeviceManually = optjson.SetBool(*c.enableRelease)
						}
						teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
							Name: tm.Name,
							MDM:  fleet.TeamSpecMDM{MacOSSetup: macSetup},
						}}}
						s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
					}
				} else {
					// no setup assistant, use the PATCH /setup_experience endpoint
					payload := map[string]any{
						"team_id": tm.ID,
					}
					if c.enableRelease != nil {
						payload["enable_release_device_manually"] = *c.enableRelease
					}
					s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), expectedPatchStatus)
				}

				teamResp = teamResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &teamResp)
				require.Equal(t, c.expectedRelease, teamResp.Team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
				require.Equal(t, c.expectedAssistant, teamResp.Team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
			})
		}
	})

	t.Run("TestMDMAppleSetupEndpoint", func(t *testing.T) {
		t.Run("TestNoTeam", func(t *testing.T) {
			var acResp appConfigResponse
			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: new(uint(0)), EnableEndUserAuthentication: new(true)}, http.StatusNoContent)
			acResp = appConfigResponse{}
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			require.True(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			lastActivityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, 0)

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: new(uint(0)), EnableEndUserAuthentication: new(true)}, http.StatusNoContent)
			acResp = appConfigResponse{}
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			require.True(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				``, lastActivityID) // no new activity

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: new(uint(0)), LockEndUserInfo: new(true)}, http.StatusNoContent)
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			require.True(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			require.True(t, acResp.MDM.MacOSSetup.LockEndUserInfo.Value)

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: new(uint(0)), EnableEndUserAuthentication: new(false)}, http.StatusNoContent)
			acResp = appConfigResponse{}
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			require.False(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			require.False(t, acResp.MDM.MacOSSetup.LockEndUserInfo.Value)
			require.Greater(t, s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledMacosSetupEndUserAuth{}.ActivityName(),
				`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, 0), lastActivityID)

			// Can't enable Lock End User Info when EUA is disabled
			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: new(uint(0)), LockEndUserInfo: new(true)}, http.StatusUnprocessableEntity)
		})

		t.Run("TestTeam", func(t *testing.T) {
			tmConfigPath := fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID)
			expectedActivityDetail := fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name)
			var tmResp teamResponse
			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: new(true)}, http.StatusNoContent)
			tmResp = teamResponse{}
			s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
			require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			lastActivityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				expectedActivityDetail, 0)

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: new(true)}, http.StatusNoContent)
			tmResp = teamResponse{}
			s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
			require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				``, lastActivityID) // no new activity

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, LockEndUserInfo: new(true)}, http.StatusNoContent)
			s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
			require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			require.True(t, tmResp.Team.Config.MDM.MacOSSetup.LockEndUserInfo.Value)

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: new(false)}, http.StatusNoContent)
			tmResp = teamResponse{}
			s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
			require.False(t, tmResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			require.False(t, tmResp.Team.Config.MDM.MacOSSetup.LockEndUserInfo.Value)
			require.Greater(t, s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledMacosSetupEndUserAuth{}.ActivityName(),
				expectedActivityDetail, 0), lastActivityID)

			// Can't enable Lock End User Info when EUA is disabled
			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, LockEndUserInfo: new(true)}, http.StatusUnprocessableEntity)
		})
	})

	t.Run("ValidateEnableEndUserAuthentication", func(t *testing.T) {
		// ensure the test is setup correctly
		var acResp appConfigResponse
		var errResp validationErrResp
		var teamResp teamResponse
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
		require.NotEmpty(t, acResp.MDM.EndUserAuthentication)

		// can't clear IdP settings while end user authentication is enabled (global)
		errResp = validationErrResp{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "",
					"idp_name": "",
					"metadata_url": ""
				}
			}
		}`), http.StatusUnprocessableEntity, &errResp)
		require.Len(t, errResp.Errors, 1)
		require.Equal(t, "End user authentication is enabled. Please disable end user authentication in Controls > Setup experience and try again", errResp.Errors[0].Reason)

		//  disable end user authentication before clearing IdP settings
		acResp = appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": false
				}
			}
		}`), http.StatusOK, &acResp)
		require.False(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
		acResp = appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "",
					"idp_name": "",
					"metadata_url": ""
				}
			}
		}`), http.StatusOK, &acResp)
		require.True(t, acResp.MDM.EndUserAuthentication.IsEmpty())

		// can't clear IdP settings while end user authentication is enabled on a team
		// 1. configure IdP globally
		acResp = appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "https://localhost:8080",
					"idp_name": "SimpleSAML",
					"metadata_url": "%s"
				}
			}
		}`, testSAMLIDPMetadataURL)), http.StatusOK, &acResp)
		require.NotEmpty(t, acResp.MDM.EndUserAuthentication)
		require.False(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)

		// 2. enable EUA on a team
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), json.RawMessage(fmt.Sprintf(`{
			"name": %q,
			"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}
		}`, tm.Name)), http.StatusOK, &teamResp)
		require.True(t, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)

		// 3. clearing IdP while team EUA is enabled should fail
		errResp = validationErrResp{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "",
					"idp_name": "",
					"metadata_url": ""
				}
			}
		}`), http.StatusUnprocessableEntity, &errResp)
		require.Len(t, errResp.Errors, 1)
		require.Equal(t, "End user authentication is enabled. Please disable end user authentication in Controls > Setup experience and try again", errResp.Errors[0].Reason)

		// 4. disable team EUA, then clear IdP
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), json.RawMessage(fmt.Sprintf(`{
			"name": %q,
			"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": false
				}
			}
		}`, tm.Name)), http.StatusOK, &teamResp)
		acResp = appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "",
					"idp_name": "",
					"metadata_url": ""
				}
			}
		}`), http.StatusOK, &acResp)
		require.True(t, acResp.MDM.EndUserAuthentication.IsEmpty())

		// can't enable end user authentication without a configured IdP
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "",
					"idp_name": "",
					"metadata_url": ""
				},
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}
		}`), http.StatusUnprocessableEntity, &acResp)

		// can't use setup endpoint to enable end user authentication on no team without a configured IdP
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: new(uint(0)), EnableEndUserAuthentication: new(true)}, http.StatusUnprocessableEntity)

		// can't enable end user authentication on team config without a configured IdP already on app config
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), json.RawMessage(fmt.Sprintf(`{
			"name": %q,
			"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}
		}`, tm.Name)), http.StatusUnprocessableEntity, &teamResp)

		// can't use setup endpoint to enable end user authentication on team without a configured IdP
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: new(true)}, http.StatusUnprocessableEntity)

		// ensure IdP is empty for the rest of the tests
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "",
					"idp_name": "",
					"metadata_url": ""
				}
			}
		}`), http.StatusOK, &acResp)
		require.Empty(t, acResp.MDM.EndUserAuthentication)
	})
}

func (s *integrationMDMTestSuite) TestMacosSetupAssistant() {
	ctx := context.Background()
	t := s.T()

	const defaultProf = `{
	"profile_name": "%s",
	"allow_pairing": true,
	"is_mdm_removable": true,
	"org_magic": "1",
	"language": "en",
	"region": "US",
	"skip_setup_items": [
		"Accessibility",
		"Appearance",
		"AppleID",
		"AppStore",
		"Biometric",
		"Diagnostics",
		"FileVault",
		"iCloudDiagnostics",
		"iCloudStorage",
		"Location",
		"Payment",
		"Privacy",
		"Restore",
		"ScreenTime",
		"Siri",
		"TermsOfAddress",
		"TOS",
		"UnlockWithWatch"
	]
}
`

	// Associate the token with the team
	s.enableABM(t.Name())
	// start a server that will mock the Apple DEP API
	profileCalls := 0
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, "foo")))
		case "/profile":
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			profileCalls++
			var prof godep.Profile
			assert.NoError(t, json.Unmarshal(body, &prof))
			switch {
			case len(prof.ProfileName) > 125:
				w.WriteHeader(http.StatusBadRequest)
				assert.NoError(t, encoder.Encode(map[string]any{"error": "CONFIG_NAME_INVALID"}))
			case prof.ProfileName == "":
				w.WriteHeader(http.StatusBadRequest)
				assert.NoError(t, encoder.Encode(map[string]any{"error": "CONFIG_NAME_REQUIRED"}))
			case len(prof.ConfigurationWebURL) > 125:
				w.WriteHeader(http.StatusBadRequest)
				assert.NoError(t, encoder.Encode(map[string]any{"error": "CONFIG_URL_INVALID"}))
			case len(prof.Department) > 125:
				w.WriteHeader(http.StatusBadRequest)
				assert.NoError(t, encoder.Encode(map[string]any{"error": "DEPARTMENT_INVALID"}))
			case len(prof.SupportPhoneNumber) > 50:
				w.WriteHeader(http.StatusBadRequest)
				assert.NoError(t, encoder.Encode(map[string]any{"error": "SUPPORT_PHONE_INVALID"}))
			case len(prof.SupportEmailAddress) > 250:
				w.WriteHeader(http.StatusBadRequest)
				assert.NoError(t, encoder.Encode(map[string]any{"error": "SUPPORT_EMAIL_INVALID"}))
			case len(prof.OrgMagic) > 256:
				w.WriteHeader(http.StatusBadRequest)
				assert.NoError(t, encoder.Encode(map[string]any{"error": "MAGIC_INVALID"}))
			case !prof.IsMDMRemovable && !prof.IsSupervised:
				w.WriteHeader(http.StatusBadRequest)
				assert.NoError(t, encoder.Encode(map[string]any{"error": "FLAGS_INVALID"}))
			default:
				w.WriteHeader(http.StatusOK)
				assert.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"}))
			}
		}
	}))

	// get for no team returns 404
	var getResp getMDMAppleSetupAssistantResponse
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNotFound, &getResp)
	// get for non-existing team returns 404
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNotFound, &getResp, "team_id", "123")

	// Profile name too long
	var createResp createMDMAppleSetupAssistantResponse
	r := s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_too_long",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "%s"}`, strings.Repeat("a", 126))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "CONFIG_NAME_INVALID")

	// Profile name missing
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(`{"profile_name": ""}`),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "CONFIG_NAME_REQUIRED")

	// Config URL invalid
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "%s"}`, strings.Repeat("a", 126))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "CONFIG_URL_INVALID")

	// Department invalid
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "%s"}`, strings.Repeat("a", 126))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "DEPARTMENT_INVALID")

	// Invalid support phone
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "foo", "support_phone_number": "%s"}`, strings.Repeat("1", 51))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "SUPPORT_PHONE_INVALID")

	// Invalid support email
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "foo", "support_phone_number": "555-123-4567", "support_email_address": "%s"}`, strings.Repeat("1", 251))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "SUPPORT_EMAIL_INVALID")

	// Invalid magic
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "foo", "support_phone_number": "555-123-4567", "support_email_address": "support@example.com", "org_magic": "%s"}`, strings.Repeat("1", 257))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "MAGIC_INVALID")

	// Invalid flag combo
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "foo", "support_phone_number": "555-123-4567", "support_email_address": "support@example.com", "org_magic": "1", "is_mdm_removable": false, "is_supervised": false}`),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "FLAGS_INVALID")

	// create a setup assistant for no team
	noTeamProf := fmt.Sprintf(defaultProf, "no-team")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &createResp)
	noTeamAsst := createResp.MDMAppleSetupAssistant
	require.Nil(t, noTeamAsst.TeamID)
	require.NotZero(t, noTeamAsst.UploadedAt)
	require.Equal(t, "no-team", noTeamAsst.Name)
	require.JSONEq(t, noTeamProf, string(noTeamAsst.Profile))
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team", "team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, 0)

	// create a team and a setup assistant for that team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)
	tmProf := fmt.Sprintf(defaultProf, "team1")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team1",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	tmAsst := createResp.MDMAppleSetupAssistant
	require.NotNil(t, tmAsst.TeamID)
	require.Equal(t, tm.ID, *tmAsst.TeamID)
	require.NotZero(t, tmAsst.UploadedAt)
	require.Equal(t, "team1", tmAsst.Name)
	require.JSONEq(t, tmProf, string(tmAsst.Profile))
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team1", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name), 0)

	// update no-team
	noTeamProf = fmt.Sprintf(defaultProf, "no-team2")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team2",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &createResp)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team2", "team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, 0)

	// update team
	tmProf = fmt.Sprintf(defaultProf, "team2")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team2",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	lastChangedActID := s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name), 0)

	// sleep a second so the uploaded-at timestamp would change if there were
	// changes, then update again no team/team but without any change, doesn't
	// create a changed activity.
	time.Sleep(time.Second)

	// no change to no-team
	priorProfileCalls := profileCalls
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team2",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &createResp)
	// the last activity is that of the team (i.e. no new activity was created for no-team)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name), lastChangedActID)
	// No new requests to DEP were made since nothing changed
	require.Equal(t, priorProfileCalls, profileCalls)

	// no change to team
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team2",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name), lastChangedActID)
	// No new requests to DEP were made since nothing changed
	require.Equal(t, priorProfileCalls, profileCalls)

	// update team with only a setup assistant JSON change, should detect it
	// and create a new activity (name is the same)
	tmProf = fmt.Sprintf(defaultProf, "update")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team2",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	latestChangedActID := s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name), 0)
	require.Greater(t, latestChangedActID, lastChangedActID)
	// Should revalidate with Apple now since there was a change
	require.Greater(t, profileCalls, priorProfileCalls)

	// get no team
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusOK, &getResp)
	require.Nil(t, getResp.TeamID)
	require.NotZero(t, getResp.UploadedAt)
	require.Equal(t, "no-team2", getResp.Name)
	require.JSONEq(t, noTeamProf, string(getResp.Profile))

	// get team
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusOK, &getResp, "team_id", fmt.Sprint(tm.ID))
	require.NotNil(t, getResp.TeamID)
	require.Equal(t, tm.ID, *getResp.TeamID)
	require.NotZero(t, getResp.UploadedAt)
	require.Equal(t, "team2", getResp.Name)
	require.JSONEq(t, tmProf, string(getResp.Profile))

	// try to set the url
	tmProf = `{"url": "https://example.com"}`
	res := s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team5",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `The automatic enrollment profile can't include url.`)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name), latestChangedActID)

	// try to set a non-object json value
	tmProf = `true`
	res = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team6",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusInternalServerError) // TODO: that should be a 4xx error, see #4406
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `cannot unmarshal bool into Go value of type map[string]interface`)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm.ID, tm.Name, tm.ID, tm.Name), latestChangedActID)

	// delete the no-team setup assistant
	s.Do("DELETE", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNoContent)
	latestChangedActID = s.lastActivityMatches(fleet.ActivityTypeDeletedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team2", "team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, 0)

	// get for no team returns 404
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNotFound, &getResp)

	// delete the team (not the assistant), this also deletes the assistant
	err = s.ds.DeleteTeam(ctx, tm.ID)
	require.NoError(t, err)

	// get for team returns 404
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNotFound, &getResp, "team_id", fmt.Sprint(tm.ID))

	// no deleted activity was created for the team as the whole team was deleted
	// (a deleted team activity would exist if that was done via the API and not
	// directly with the datastore)
	s.lastActivityMatches(fleet.ActivityTypeDeletedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team2", "team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`, latestChangedActID)

	// create another team and a setup assistant for that team
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name() + "2",
		Description: "desc2",
	})
	require.NoError(t, err)
	tm2Prof := fmt.Sprintf(defaultProf, "teamB")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm2.ID,
		Name:              "teamB",
		EnrollmentProfile: json.RawMessage(tm2Prof),
	}, http.StatusOK, &createResp)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "teamB", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm2.ID, tm2.Name, tm2.ID, tm2.Name), 0)

	// delete that team's setup assistant
	s.Do("DELETE", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNoContent, "team_id", fmt.Sprint(tm2.ID))
	s.lastActivityMatches(fleet.ActivityTypeDeletedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "teamB", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm2.ID, tm2.Name, tm2.ID, tm2.Name), 0)

	// Try with a team that has no relevant ABM tokens. Even with a second, unrelated
	// token present (so there is no single obvious token to use), validation succeeds
	// using any token.
	teamNoABM, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name() + "no_abm",
		Description: "no abm",
	})
	require.NoError(t, err)
	otherOrg := t.Name() + "some_other_org"
	s.enableABM(otherOrg)
	s.mockDEPResponse(otherOrg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, "foo")))
		case "/profile":
			w.WriteHeader(http.StatusOK)
			assert.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"}))
		}
	}))
	require.NoError(t, err)
	var noABMResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &teamNoABM.ID,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(defaultProf, "no_abm")),
	}, http.StatusOK, &noABMResp)

	// With every token invalid or terms-expired, adding a setup assistant fails with
	// an actionable message instead of a cryptic Apple error.
	allToks, err := s.ds.ListABMTokens(ctx)
	require.NoError(t, err)
	for _, tok := range allToks {
		_, err = s.ds.SetABMTokenTermsExpiredForOrgName(ctx, tok.OrganizationName, true)
		require.NoError(t, err)
	}
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &teamNoABM.ID,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(defaultProf, "no_usable_tokens")),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "All Apple Business Manager (ABM) tokens are invalid or have expired terms")

	// With no ABM token at all, adding a setup assistant is rejected with a clear message.
	allToks, err = s.ds.ListABMTokens(ctx)
	require.NoError(t, err)
	for _, tok := range allToks {
		require.NoError(t, s.ds.DeleteABMToken(ctx, tok.ID))
	}
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &teamNoABM.ID,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(defaultProf, "no_abm_at_all")),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "No Apple Business Manager (ABM) token found")
}

func (s *integrationMDMTestSuite) TestManagedLocalAccount() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := t.Context()
	abmOrgName := t.Name()
	s.enableABM(abmOrgName)

	// Create a team used by enrollment subtests
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: t.Name() + "team"}, http.StatusOK, &createTeamResp)
	team := createTeamResp.Team

	// Enable managed local account on the team
	s.Do("PATCH", "/api/latest/fleet/setup_experience",
		fleet.MDMAppleSetupPayload{TeamID: &team.ID, EnableManagedLocalAccount: new(true)}, http.StatusNoContent)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledManagedLocalAccount{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q, "platform": "darwin"}`, team.ID, team.Name, team.ID, team.Name), 0)

	// Assign ABM org to the team
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_business_manager": [{
				"organization_name": %q,
				"macos_team": %q,
				"ios_team": "",
				"ipados_team": ""
			}]
		}
	}`, abmOrgName, team.Name)), http.StatusOK, &acResp)

	// Mock DEP response with devices
	serials := []string{"MANAGED_ACCT_1", "MANAGED_ACCT_2", "MANAGED_ACCT_3", "MANAGED_ACCT_4"}
	syncCount := 0
	s.mockDEPResponse(abmOrgName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			_, _ = w.Write([]byte(`{"profile_uuid": "abc"}`))
		case "/devices/sync":
			var devices []godep.Device
			for i := 0; i <= syncCount && i < len(serials); i++ {
				devices = append(devices, godep.Device{SerialNumber: serials[i]})
			}
			syncCount++
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{Devices: devices})
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	t.Run("Enrollment flow", func(t *testing.T) {
		// DEP-enroll the first host and run the post-enrollment worker
		s.runDEPSchedule()
		depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
		mdmDevice := mdmtest.NewTestMDMClientAppleDEPFromDevice(s.server.URL, depURLToken, serials[0], "MacBookPro16,1")
		require.NoError(t, mdmDevice.Enroll())
		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()

		// Verify managed account row exists with status = NULL (pending)
		var status *string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &status,
				"SELECT status FROM host_managed_local_account_passwords WHERE host_uuid = ?", mdmDevice.UUID)
		})
		require.Nil(t, status)

		host, err := s.ds.HostByIdentifier(ctx, serials[0])
		require.NoError(t, err)

		// Get host API should show managed account as pending. The password is
		// available even before the device acks AccountConfiguration: under the
		// new (rotation) semantics, password_available is decoupled from the
		// command lifecycle and tracks "we have a current encrypted_password and
		// status != failed".
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		require.NotNil(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.Status)
		require.Equal(t, "pending", *hostResp.Host.MDM.OSSettings.ManagedLocalAccount.Status)
		require.True(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.PasswordAvailable)

		// Device acknowledges AccountConfiguration command → status becomes verified
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		var foundAccountConfig bool
		for cmd != nil {
			if cmd.Command.RequestType == fleet.AccountConfigurationCmdName {
				foundAccountConfig = true
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		require.True(t, foundAccountConfig, "AccountConfiguration command should have been queued")

		// Managed account row should be status = verified
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &status,
				"SELECT status FROM host_managed_local_account_passwords WHERE host_uuid = ?", mdmDevice.UUID)
		})
		require.NotNil(t, status)
		require.Equal(t, string(fleet.MDMDeliveryVerified), *status)

		// Get host API should show managed account as verified with password available
		hostResp = getHostResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		require.NotNil(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.Status)
		require.Equal(t, "verified", *hostResp.Host.MDM.OSSettings.ManagedLocalAccount.Status)
		require.True(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.PasswordAvailable)

		// Activity for creating a local account was created
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedManagedLocalAccount{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

		// After the AccountConfiguration ack, the host should have been
		// flagged for refetch so we can capture _fleetadmin's UUID on the
		// next osquery detail cycle, and account_uuid should still be NULL.
		hostAfterAck, err := s.ds.Host(ctx, host.ID)
		require.NoError(t, err)
		require.True(t, hostAfterAck.RefetchRequested, "host should have RefetchRequested set after AccountConfiguration ack")
		var accountUUIDAfterAck *string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &accountUUIDAfterAck,
				"SELECT account_uuid FROM host_managed_local_account_passwords WHERE host_uuid = ?", mdmDevice.UUID)
		})
		require.Nil(t, accountUUIDAfterAck)

		// Enroll the host with osquery so we can simulate a detail-query write.
		var enrollSecretResp applyEnrollSecretSpecResponse
		s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
			Spec: &fleet.EnrollSecretSpec{
				Secrets: []*fleet.EnrollSecret{{Secret: t.Name(), TeamID: &team.ID}},
			},
		}, http.StatusOK, &enrollSecretResp)

		enrollJSON, err := json.Marshal(&contract.EnrollOsqueryAgentRequest{
			EnrollSecret:   t.Name(),
			HostIdentifier: mdmDevice.UUID,
		})
		require.NoError(t, err)
		var osqueryEnrollResp contract.EnrollOsqueryAgentResponse
		hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", enrollJSON, http.StatusOK)
		require.NoError(t, json.NewDecoder(hres.Body).Decode(&osqueryEnrollResp))
		require.NoError(t, hres.Body.Close())
		require.NotEmpty(t, osqueryEnrollResp.NodeKey)

		// Simulate the osquery fleet_detail_query_users write, including the
		// _fleetadmin row. This should capture account_uuid.
		submitUsersResult := func() {
			distributedReq := SubmitDistributedQueryResultsRequest{
				NodeKey: osqueryEnrollResp.NodeKey,
				Results: map[string][]map[string]string{
					"fleet_detail_query_users": {
						{"uid": "501", "username": "alice", "type": "standard", "groupname": "staff", "shell": "/bin/zsh", "uuid": "alice-osquery-uuid"},
						{"uid": "502", "username": "_fleetadmin", "type": "standard", "groupname": "staff", "shell": "/bin/zsh", "uuid": "fleetadmin-osquery-uuid"},
					},
				},
				Statuses: map[string]fleet.OsqueryStatus{
					"fleet_detail_query_users": 0,
				},
			}
			distributedResp := submitDistributedQueryResultsResponse{}
			s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
		}
		submitUsersResult()

		var accountUUIDAfterIngest *string
		var updatedAtAfterIngest time.Time
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			row := q.QueryRowxContext(ctx,
				"SELECT account_uuid, updated_at FROM host_managed_local_account_passwords WHERE host_uuid = ?", mdmDevice.UUID)
			return row.Scan(&accountUUIDAfterIngest, &updatedAtAfterIngest)
		})
		require.NotNil(t, accountUUIDAfterIngest)
		require.Equal(t, "fleetadmin-osquery-uuid", *accountUUIDAfterIngest)

		// _fleetadmin is filtered from host_users even though the query
		// returns it, so only alice is persisted.
		hostUsers, err := s.ds.ListHostUsers(ctx, host.ID)
		require.NoError(t, err)
		require.Len(t, hostUsers, 1)
		require.Equal(t, "alice", hostUsers[0].Username)

		// Re-submit the same payload. account_uuid is already set, so the
		// conditional UPDATE is a no-op. updated_at must not change, proving
		// no writer traffic hit the row a second time.
		submitUsersResult()
		var accountUUIDAfterRepeat *string
		var updatedAtAfterRepeat time.Time
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			row := q.QueryRowxContext(ctx,
				"SELECT account_uuid, updated_at FROM host_managed_local_account_passwords WHERE host_uuid = ?", mdmDevice.UUID)
			return row.Scan(&accountUUIDAfterRepeat, &updatedAtAfterRepeat)
		})
		require.NotNil(t, accountUUIDAfterRepeat)
		require.Equal(t, *accountUUIDAfterIngest, *accountUUIDAfterRepeat)
		require.True(t, updatedAtAfterRepeat.Equal(updatedAtAfterIngest),
			"updated_at must not change on repeat ingest (expected %v, got %v)", updatedAtAfterIngest, updatedAtAfterRepeat)

		// Read the managed account password
		var pwdResp getHostManagedAccountPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password", host.ID), nil, http.StatusOK, &pwdResp)
		require.NotNil(t, pwdResp.ManagedLocalAccount)
		require.NotEmpty(t, pwdResp.ManagedLocalAccount.Password)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeViewedManagedLocalAccount{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

		// Disable managed local account
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: &team.ID, EnableManagedLocalAccount: new(false)}, http.StatusNoContent)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledManagedLocalAccount{}.ActivityName(),
			fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q, "platform": "darwin"}`, team.ID, team.Name, team.ID, team.Name), 0)

		// Existing host's password is still readable
		pwdResp = getHostManagedAccountPasswordResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password", host.ID), nil, http.StatusOK, &pwdResp)
		require.NotNil(t, pwdResp.ManagedLocalAccount)
		require.NotEmpty(t, pwdResp.ManagedLocalAccount.Password)

		// Second host enrolled after disable — should NOT get a managed account
		s.runDEPSchedule()
		mdmDevice2 := mdmtest.NewTestMDMClientAppleDEPFromDevice(s.server.URL, depURLToken, serials[1], "MacBookPro16,1")
		require.NoError(t, mdmDevice2.Enroll())
		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()

		cmd, err = mdmDevice2.Idle()
		require.NoError(t, err)
		for cmd != nil {
			cmd, err = mdmDevice2.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}

		host2, err := s.ds.HostByIdentifier(ctx, serials[1])
		require.NoError(t, err)
		var pwdResp2 getHostManagedAccountPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password", host2.ID), nil, http.StatusBadRequest, &pwdResp2)
	})

	t.Run("Failed enrollment", func(t *testing.T) {
		// Re-enable managed local account on the team
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: &team.ID, EnableManagedLocalAccount: new(true)}, http.StatusNoContent)

		// DEP-enroll a third host
		s.runDEPSchedule()
		depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
		mdmDevice3 := mdmtest.NewTestMDMClientAppleDEPFromDevice(s.server.URL, depURLToken, serials[2], "MacBookPro16,1")
		require.NoError(t, mdmDevice3.Enroll())
		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()

		// Device sends error response to AccountConfiguration command
		cmd, err := mdmDevice3.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType == fleet.AccountConfigurationCmdName {
				cmd, err = mdmDevice3.Err(cmd.CommandUUID, []mdm.ErrorChain{
					// TODO(JK): Not sure if this error is correct, need to test failure on a real device
					{ErrorCode: 12064, ErrorDomain: "MCMDMErrorDomain", LocalizedDescription: "Failed to create the admin account.", USEnglishDescription: "Failed to create the admin account."},
				})
			} else {
				cmd, err = mdmDevice3.Acknowledge(cmd.CommandUUID)
			}
			require.NoError(t, err)
		}

		// Verify status is "failed"
		var status *string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &status,
				"SELECT status FROM host_managed_local_account_passwords WHERE host_uuid = ?", mdmDevice3.UUID)
		})
		require.NotNil(t, status)
		require.Equal(t, string(fleet.MDMDeliveryFailed), *status)

		// Get host API should show managed account as failed with no password available
		host3, err := s.ds.HostByIdentifier(ctx, serials[2])
		require.NoError(t, err)
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host3.ID), nil, http.StatusOK, &hostResp)
		require.NotNil(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.Status)
		require.Equal(t, "failed", *hostResp.Host.MDM.OSSettings.ManagedLocalAccount.Status)
		require.False(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.PasswordAvailable)
	})

	t.Run("Setup experience global config", func(t *testing.T) {
		// Enable via PATCH /setup_experience
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: new(uint), EnableManagedLocalAccount: new(true)}, http.StatusNoContent)

		var acResp appConfigResponse
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.True(t, acResp.MDM.MacOSSetup.EnableManagedLocalAccount.Valid)
		require.True(t, acResp.MDM.MacOSSetup.EnableManagedLocalAccount.Value)
		lastActivityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledManagedLocalAccount{}.ActivityName(),
			`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null, "platform": "darwin"}`, 0)

		// Patching same value again should not create a new activity
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: new(uint), EnableManagedLocalAccount: new(true)}, http.StatusNoContent)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledManagedLocalAccount{}.ActivityName(),
			``, lastActivityID)

		// Disable
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: new(uint), EnableManagedLocalAccount: new(false)}, http.StatusNoContent)

		acResp = appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.True(t, acResp.MDM.MacOSSetup.EnableManagedLocalAccount.Valid)
		require.False(t, acResp.MDM.MacOSSetup.EnableManagedLocalAccount.Value)
		require.Greater(t, s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledManagedLocalAccount{}.ActivityName(),
			`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null, "platform": "darwin"}`, 0), lastActivityID)
	})

	t.Run("Rotation flow", func(t *testing.T) {
		// The integration test scaffold doesn't register the rotation cron, so
		// we drive it directly the same way the recovery-lock subtests do.
		runRotationCron := func(t *testing.T) {
			t.Helper()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			require.NoError(t, apple_mdm.SendManagedLocalAccountRotationCommands(
				t.Context(), s.ds, s.mdmCommander, logger, s.fleetSvc.NewActivity,
			))
		}

		// Re-enable managed local account on the team for this subtest.
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: &team.ID, EnableManagedLocalAccount: new(true)}, http.StatusNoContent)

		// DEP-enroll a fresh host (serial #4) so we can drive the rotation
		// state machine end-to-end without disturbing earlier subtests' hosts.
		s.runDEPSchedule()
		depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
		mdmDevice := mdmtest.NewTestMDMClientAppleDEPFromDevice(s.server.URL, depURLToken, serials[3], "MacBookPro16,1")
		require.NoError(t, mdmDevice.Enroll())
		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()

		// Ack AccountConfiguration so status flips to verified.
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}

		host, err := s.ds.HostByIdentifier(ctx, serials[3])
		require.NoError(t, err)

		// Capture _fleetadmin's account_uuid so we exercise the immediate (not deferred) path.
		require.NoError(t, s.ds.SetManagedLocalAccountUUID(ctx, mdmDevice.UUID, "AAAAAAAA-BBBB-CCCC-DDDD-FFFFFFFFFFFF"))

		// View the password — first view sets auto_rotate_at and flips status to pending.
		var pwdResp getHostManagedAccountPasswordResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password", host.ID), nil, http.StatusOK, &pwdResp)
		require.NotNil(t, pwdResp.ManagedLocalAccount)
		originalPassword := pwdResp.ManagedLocalAccount.Password
		require.NotEmpty(t, originalPassword)

		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		require.NotNil(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.AutoRotateAt)
		firstRotateAt := *hostResp.Host.MDM.OSSettings.ManagedLocalAccount.AutoRotateAt
		require.Equal(t, "pending", *hostResp.Host.MDM.OSSettings.ManagedLocalAccount.Status)
		require.True(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.PasswordAvailable)

		// Second view inside the window must NOT extend the timer.
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password", host.ID), nil, http.StatusOK, &pwdResp)
		hostResp = getHostResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		require.NotNil(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.AutoRotateAt)
		require.True(t, firstRotateAt.Equal(*hostResp.Host.MDM.OSSettings.ManagedLocalAccount.AutoRotateAt),
			"second view inside the window must not extend auto_rotate_at")

		// Manual rotation (UUID present) → 204, pending_rotation true.
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password/rotate", host.ID), nil, http.StatusNoContent)

		hostResp = getHostResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		require.True(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.PendingRotation)
		// Once the rotation command has been sent, auto_rotate_at must be cleared
		// — the API should not keep returning the stale view-driven deadline.
		require.Nil(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.AutoRotateAt)

		// Activity logged with the calling user (admin) — NOT Fleet.
		rotatedName := fleet.ActivityTypeRotatedManagedLocalAccountPassword{}.ActivityName()
		manualRotationActivityID := s.lastActivityOfTypeMatches(rotatedName,
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)
		require.NotZero(t, manualRotationActivityID)

		// Rotating again while one is already in flight is rejected with 400 — the
		// previous behavior was an idempotent 204, but per spec this should now
		// surface a clear error to the caller.
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password/rotate", host.ID), nil, http.StatusBadRequest)
		// And the activity count must not have grown — the rejected call must
		// not log a duplicate triggered-rotation activity.
		require.Equal(t, manualRotationActivityID, s.lastActivityOfTypeMatches(rotatedName, "", 0),
			"rejected rotate-while-pending must not emit a new activity")

		// Drain commands and ack the SetAutoAdminPassword. Verify state transitions to verified.
		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		var foundSetAutoAdmin bool
		for cmd != nil {
			if cmd.Command.RequestType == fleet.SetAutoAdminPasswordCmdName {
				foundSetAutoAdmin = true
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		require.True(t, foundSetAutoAdmin, "SetAutoAdminPassword command should have been queued")

		hostResp = getHostResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		require.Equal(t, "verified", *hostResp.Host.MDM.OSSettings.ManagedLocalAccount.Status)
		require.False(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.PendingRotation)
		require.Nil(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.AutoRotateAt)

		// Password actually changed.
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password", host.ID), nil, http.StatusOK, &pwdResp)
		require.NotEqual(t, originalPassword, pwdResp.ManagedLocalAccount.Password)
		afterFirstRotation := pwdResp.ManagedLocalAccount.Password

		// No new rotation activity was emitted by the ack handler — the most
		// recent rotation activity must still be the one logged at click time.
		require.Equal(t, manualRotationActivityID, s.lastActivityOfTypeMatches(rotatedName, "", 0),
			"ack handler must not log a duplicate rotation activity")

		// Auto rotation: backdate auto_rotate_at into the past, trigger the cron,
		// ack — assert the activity is logged with FleetInitiated=true.
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password", host.ID), nil, http.StatusOK, &pwdResp)
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE host_managed_local_account_passwords SET auto_rotate_at = NOW(6) - INTERVAL 1 MINUTE WHERE host_uuid = ?`,
				mdmDevice.UUID)
			return err
		})
		runRotationCron(t)

		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		foundSetAutoAdmin = false
		for cmd != nil {
			if cmd.Command.RequestType == fleet.SetAutoAdminPasswordCmdName {
				foundSetAutoAdmin = true
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		require.True(t, foundSetAutoAdmin, "auto-rotation should have queued a SetAutoAdminPassword command")

		// Auto-rotation activity exists and is FleetInitiated.
		var fleetActivities listActivitiesResponse
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &fleetActivities,
			"order_key", "id", "order_direction", "desc", "per_page", "20")
		rotateActivityName := fleet.ActivityTypeRotatedManagedLocalAccountPassword{}.ActivityName()
		var sawFleetRotation bool
		for _, a := range fleetActivities.Activities {
			if a.Type != rotateActivityName || !a.FleetInitiated || a.Details == nil {
				continue
			}
			var d fleet.ActivityTypeRotatedManagedLocalAccountPassword
			if err := json.Unmarshal(*a.Details, &d); err != nil {
				continue
			}
			if d.HostID == host.ID {
				sawFleetRotation = true
				break
			}
		}
		require.True(t, sawFleetRotation, "expected an auto-rotation activity with FleetInitiated=true for this host")

		// Password changed again.
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password", host.ID), nil, http.StatusOK, &pwdResp)
		require.NotEqual(t, afterFirstRotation, pwdResp.ManagedLocalAccount.Password)

		// Deferred rotation: clear account_uuid, click rotate, verify deferred state
		// without a command in the queue.
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE host_managed_local_account_passwords SET account_uuid = NULL WHERE host_uuid = ?`,
				mdmDevice.UUID)
			return err
		})

		preDeferredID := s.lastActivityOfTypeMatches(rotatedName, "", 0)
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password/rotate", host.ID), nil, http.StatusNoContent)
		// Activity logged at click time with user actor (FleetInitiated=false).
		afterDeferredID := s.lastActivityOfTypeMatches(rotatedName, "", 0)
		require.Greater(t, afterDeferredID, preDeferredID, "deferred manual rotate should log an activity at click time")

		// No SetAutoAdminPassword in the queue (UUID was missing).
		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			require.NotEqual(t, fleet.SetAutoAdminPasswordCmdName, cmd.Command.RequestType,
				"deferred rotation must not have queued a SetAutoAdminPassword command yet")
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}

		// Restore account_uuid and run the cron — the row should now be picked up,
		// the command enqueued, and after ack NO new activity should be added
		// (the deferred manual click already logged it).
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE host_managed_local_account_passwords SET account_uuid = ? WHERE host_uuid = ?`,
				"AAAAAAAA-BBBB-CCCC-DDDD-FFFFFFFFFFFF", mdmDevice.UUID)
			return err
		})
		runRotationCron(t)

		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		foundSetAutoAdmin = false
		for cmd != nil {
			if cmd.Command.RequestType == fleet.SetAutoAdminPasswordCmdName {
				foundSetAutoAdmin = true
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		require.True(t, foundSetAutoAdmin, "cron should fulfill the deferred rotation once UUID lands")

		// Activity count must NOT have grown after the cron-driven completion.
		require.Equal(t, afterDeferredID, s.lastActivityOfTypeMatches(rotatedName, "", 0),
			"deferred rotation must not be re-logged by the cron")

		// Failure path: trigger another manual rotation, then NACK the
		// SetAutoAdminPassword command. Assert the failed-to-rotate activity
		// is logged and the row transitions to status=failed.
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/managed_account_password/rotate", host.ID), nil, http.StatusNoContent)

		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		var nackedSetAutoAdmin bool
		for cmd != nil {
			if cmd.Command.RequestType == fleet.SetAutoAdminPasswordCmdName {
				nackedSetAutoAdmin = true
				cmd, err = mdmDevice.Err(cmd.CommandUUID, []mdm.ErrorChain{{ErrorCode: 12345, ErrorDomain: "test"}})
			} else {
				cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			}
			require.NoError(t, err)
		}
		require.True(t, nackedSetAutoAdmin, "device should have received a SetAutoAdminPassword command to NACK")

		// failed-to-rotate activity logged with the host's display name. No actor —
		// the activity is intentionally attributed to Fleet at ack time.
		failedName := fleet.ActivityTypeFailedToRotateManagedLocalAccountPassword{}.ActivityName()
		failedActivityID := s.lastActivityOfTypeMatches(failedName,
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)
		require.NotZero(t, failedActivityID, "device error should produce a failed-to-rotate activity")

		hostResp = getHostResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		require.Equal(t, "failed", *hostResp.Host.MDM.OSSettings.ManagedLocalAccount.Status)
		require.False(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.PendingRotation)
		require.Nil(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.AutoRotateAt)
		require.False(t, hostResp.Host.MDM.OSSettings.ManagedLocalAccount.PasswordAvailable)
	})

	t.Run("Setup experience team config", func(t *testing.T) {
		// Create a team
		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: t.Name() + "team"}, http.StatusOK, &createTeamResp)
		tm := createTeamResp.Team
		tmConfigPath := fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID)
		expectedDetail := fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q, "platform": "darwin"}`, tm.ID, tm.Name, tm.ID, tm.Name)

		// Enable via PATCH /setup_experience
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableManagedLocalAccount: new(true)}, http.StatusNoContent)

		var tmResp teamResponse
		s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
		require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Valid)
		require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Value)
		lastActivityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledManagedLocalAccount{}.ActivityName(),
			expectedDetail, 0)

		// Patching same value again should not create a new activity
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableManagedLocalAccount: new(true)}, http.StatusNoContent)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledManagedLocalAccount{}.ActivityName(),
			``, lastActivityID)

		// Disable
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableManagedLocalAccount: new(false)}, http.StatusNoContent)

		tmResp = teamResponse{}
		s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
		require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Valid)
		require.False(t, tmResp.Team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Value)
		require.Greater(t, s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledManagedLocalAccount{}.ActivityName(),
			expectedDetail, 0), lastActivityID)
	})

	t.Run("Setup experience team config via Update fleet endpoint", func(t *testing.T) {
		// Create a team
		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: t.Name() + "team"}, http.StatusOK, &createTeamResp)
		tm := createTeamResp.Team
		tmConfigPath := fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID)
		expectedDetail := fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q, "platform": "darwin"}`, tm.ID, tm.Name, tm.ID, tm.Name)

		// Enable via PATCH /teams/:id
		var tmResp teamResponse
		s.DoJSON("PATCH", tmConfigPath, fleet.TeamPayload{
			MDM: &fleet.TeamPayloadMDM{MacOSSetup: &fleet.MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(true)}},
		}, http.StatusOK, &tmResp)
		require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Valid)
		require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Value)
		lastActivityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledManagedLocalAccount{}.ActivityName(),
			expectedDetail, 0)

		// Patching same value again should not create a new activity
		s.DoJSON("PATCH", tmConfigPath, fleet.TeamPayload{
			MDM: &fleet.TeamPayloadMDM{MacOSSetup: &fleet.MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(true)}},
		}, http.StatusOK, &tmResp)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledManagedLocalAccount{}.ActivityName(),
			``, lastActivityID)

		// Disable
		s.DoJSON("PATCH", tmConfigPath, fleet.TeamPayload{
			MDM: &fleet.TeamPayloadMDM{MacOSSetup: &fleet.MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(false)}},
		}, http.StatusOK, &tmResp)
		require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Valid)
		require.False(t, tmResp.Team.Config.MDM.MacOSSetup.EnableManagedLocalAccount.Value)
		s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledManagedLocalAccount{}.ActivityName(), expectedDetail, 0)
	})
}

func (s *integrationMDMTestSuite) uploadWindowsEULA(eula *fleet.MDMEULA, expectedStatus int, wantErr string) {
	s.uploadEULAFile("windows", "", eula, expectedStatus, wantErr)
}
