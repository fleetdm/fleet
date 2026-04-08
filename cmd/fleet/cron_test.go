package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mock"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	"github.com/fleetdm/fleet/v4/server/test"
)

func TestNewAppleMDMProfileManagerWithoutConfig(t *testing.T) {
	ctx := context.Background()
	mdmStorage := &mdmmock.MDMAppleStore{}
	ds := new(mock.Store)
	kv := new(mock.AdvancedKVStore)
	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, nil)
	logger := slog.New(slog.DiscardHandler)

	sch, err := newAppleMDMProfileManagerSchedule(ctx, "foo", ds, cmdr, kv, logger, 0)
	require.NotNil(t, sch)
	require.NoError(t, err)
}

func TestNewWindowsMDMProfileManagerWithoutConfig(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	logger := slog.New(slog.DiscardHandler)

	sch, err := newWindowsMDMProfileManagerSchedule(ctx, "foo", ds, logger)
	require.NotNil(t, sch)
	require.NoError(t, err)
}

func TestMigrateABMTokenDuringDEPCronJob(t *testing.T) {
	ctx := context.Background()
	ds := mysql.CreateMySQLDS(t)

	depStorage, err := ds.NewMDMAppleDEPStorage()
	require.NoError(t, err)
	// to avoid issues with syncer, use that constant as org name for now
	const tokenOrgName = "fleet"

	// insert an ABM token as if it had been migrated by the DB migration script
	tok := mysql.SetTestABMAssets(t, ds, "")
	// tok, err := ds.InsertABMToken(ctx, &fleet.ABMToken{EncryptedToken: abmToken, RenewAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)})
	// require.NoError(t, err)
	require.Empty(t, tok.OrganizationName)

	// start a server that will mock the Apple DEP API
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, tokenOrgName)))
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"})
			require.NoError(t, err)
		case "/server/devices":
			err := encoder.Encode(godep.DeviceResponse{Devices: nil})
			require.NoError(t, err)
		case "/devices/sync":
			err := encoder.Encode(godep.DeviceResponse{Devices: nil})
			require.NoError(t, err)
		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	err = depStorage.StoreConfig(ctx, tokenOrgName, &nanodep_client.Config{BaseURL: srv.URL})
	require.NoError(t, err)
	err = depStorage.StoreConfig(ctx, apple_mdm.UnsavedABMTokenOrgName, &nanodep_client.Config{BaseURL: srv.URL})
	require.NoError(t, err)

	logger := slog.New(slog.DiscardHandler)
	syncFn := appleMDMDEPSyncerJob(ds, depStorage, logger)
	err = syncFn(ctx)
	require.NoError(t, err)

	// token has been updated with its org name/apple id
	tok, err = ds.GetABMTokenByOrgName(ctx, tokenOrgName)
	require.NoError(t, err)
	require.Equal(t, tokenOrgName, tok.OrganizationName)
	require.Equal(t, "admin123", tok.AppleID)
	require.Nil(t, tok.MacOSDefaultTeamID)
	require.Nil(t, tok.IOSDefaultTeamID)
	require.Nil(t, tok.IPadOSDefaultTeamID)

	// empty-name token does not exist anymore
	_, err = ds.GetABMTokenByOrgName(ctx, "")
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// the default profile was created
	defProf, err := ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
	require.NoError(t, err)
	require.NotNil(t, defProf)
	require.NotEmpty(t, defProf.Token)

	// no profile UUID was assigned for no-team (because there are no hosts right now)
	_, _, err = ds.GetMDMAppleDefaultSetupAssistant(ctx, nil, "")
	require.Error(t, err)
	require.ErrorAs(t, err, &nfe)

	// no teams, so no team-specific custom setup assistants
	teams, err := ds.ListTeams(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, teams)

	// no no-team custom setup assistant
	_, err = ds.GetMDMAppleSetupAssistant(ctx, nil)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// no host got created
	hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Empty(t, hosts)
}

func TestCleanupStaleOSVVulnerabilities(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)
	ds := mysql.CreateMySQLDS(t)

	// Create test software
	host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	software := []fleet.Software{
		{Name: "pkg1", Version: "1.0", Source: "apps"},
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 1)

	// Insert vulnerabilities from both sources
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: host.Software[0].ID,
		CVE:        "CVE-2024-0001",
	}, fleet.UbuntuOSVSource)
	require.NoError(t, err)

	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: host.Software[0].ID,
		CVE:        "CVE-2024-0002",
	}, fleet.UbuntuOVALSource)
	require.NoError(t, err)

	t.Run("OSV enabled - does not delete OSV vulnerabilities", func(t *testing.T) {
		cleanupStaleOSVVulnerabilities(ctx, ds, logger, true)

		// Both vulnerabilities should still exist
		osvVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOSVSource)
		require.NoError(t, err)
		require.Len(t, osvVulns[host.ID], 1)

		ovalVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.Len(t, ovalVulns[host.ID], 1)
	})

	t.Run("OSV disabled - deletes only UbuntuOSVSource vulnerabilities", func(t *testing.T) {
		cleanupStaleOSVVulnerabilities(ctx, ds, logger, false)

		// OSV vulnerabilities should be deleted
		osvVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOSVSource)
		require.NoError(t, err)
		require.Empty(t, osvVulns[host.ID])

		// OVAL vulnerability should remain
		ovalVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.Len(t, ovalVulns[host.ID], 1)
		require.Equal(t, "CVE-2024-0002", ovalVulns[host.ID][0].CVE)
	})
}

func TestCleanupStaleOVALVulnerabilities(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)
	ds := mysql.CreateMySQLDS(t)

	// Create test software
	host := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
	software := []fleet.Software{
		{Name: "pkg2", Version: "2.0", Source: "apps"},
	}
	_, err := ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))
	require.Len(t, host.Software, 1)

	// Insert vulnerabilities from multiple sources
	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: host.Software[0].ID,
		CVE:        "CVE-2024-0003",
	}, fleet.UbuntuOSVSource)
	require.NoError(t, err)

	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: host.Software[0].ID,
		CVE:        "CVE-2024-0004",
	}, fleet.UbuntuOVALSource)
	require.NoError(t, err)

	_, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
		SoftwareID: host.Software[0].ID,
		CVE:        "CVE-2024-0005",
	}, fleet.RHELOVALSource)
	require.NoError(t, err)

	t.Run("deletes only UbuntuOVALSource vulnerabilities, preserves others", func(t *testing.T) {
		cleanupStaleOVALVulnerabilities(ctx, ds, logger)

		// Ubuntu OVAL vulnerabilities should be deleted
		ovalVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOVALSource)
		require.NoError(t, err)
		require.Empty(t, ovalVulns[host.ID])

		// OSV vulnerabilities should remain
		osvVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.UbuntuOSVSource)
		require.NoError(t, err)
		require.Len(t, osvVulns[host.ID], 1)
		require.Equal(t, "CVE-2024-0003", osvVulns[host.ID][0].CVE)

		// RHEL OVAL vulnerabilities should remain
		rhelVulns, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.RHELOVALSource)
		require.NoError(t, err)
		require.Len(t, rhelVulns[host.ID], 1)
		require.Equal(t, "CVE-2024-0005", rhelVulns[host.ID][0].CVE)
	})
}
