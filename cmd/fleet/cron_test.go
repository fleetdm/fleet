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

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
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
	ds := mysqltest.CreateMySQLDS(t)

	depStorage, err := ds.NewMDMAppleDEPStorage()
	require.NoError(t, err)
	// to avoid issues with syncer, use that constant as org name for now
	const tokenOrgName = "fleet"

	// insert an ABM token as if it had been migrated by the DB migration script
	tok := mysqltest.SetTestABMAssets(t, ds, "")
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
	ds := mysqltest.CreateMySQLDS(t)

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
	ds := mysqltest.CreateMySQLDS(t)

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

func TestBuildChartScopeResolver(t *testing.T) {
	historicalData := func(uptime, vulns bool) fleet.HistoricalDataSettings {
		return fleet.HistoricalDataSettings{Uptime: uptime, Vulnerabilities: vulns}
	}
	makeAppCfg := func(uptime, vulns bool) *fleet.AppConfig {
		return &fleet.AppConfig{
			Features: fleet.Features{HistoricalData: historicalData(uptime, vulns)},
		}
	}
	makeTeam := func(id uint, uptime, vulns bool) *fleet.Team {
		return &fleet.Team{
			ID: id,
			Config: fleet.TeamConfig{
				Features: fleet.Features{HistoricalData: historicalData(uptime, vulns)},
			},
		}
	}

	t.Run("global off → skip", func(t *testing.T) {
		scope := buildChartScopeResolver(makeAppCfg(false, true), nil, nil)
		skip, disabled := scope("uptime")
		require.True(t, skip)
		require.Nil(t, disabled)
	})

	t.Run("global on, no teams → no scoping", func(t *testing.T) {
		scope := buildChartScopeResolver(makeAppCfg(true, true), nil, nil)
		skip, disabled := scope("uptime")
		require.False(t, skip)
		require.Nil(t, disabled)
	})

	t.Run("global on, all teams on → empty disabled list", func(t *testing.T) {
		scope := buildChartScopeResolver(
			makeAppCfg(true, true),
			[]*fleet.Team{makeTeam(1, true, true), makeTeam(2, true, true)},
			nil,
		)
		skip, disabled := scope("uptime")
		require.False(t, skip)
		require.Empty(t, disabled)
	})

	t.Run("global on, mixed teams → disabled team IDs only", func(t *testing.T) {
		scope := buildChartScopeResolver(
			makeAppCfg(true, true),
			[]*fleet.Team{
				makeTeam(1, true, true),
				makeTeam(2, false, true), // uptime disabled on team 2
				makeTeam(3, true, true),
				makeTeam(4, false, true), // uptime disabled on team 4
			},
			nil,
		)
		skip, disabled := scope("uptime")
		require.False(t, skip)
		require.ElementsMatch(t, []uint{2, 4}, disabled)
	})

	t.Run("per-dataset isolation: uptime and cve resolve independently", func(t *testing.T) {
		scope := buildChartScopeResolver(
			makeAppCfg(true, true),
			[]*fleet.Team{
				makeTeam(1, true, false), // cve only on team 1
				makeTeam(2, false, true), // uptime only on team 2
				makeTeam(3, true, true),
			},
			nil,
		)
		skipU, disabledU := scope("uptime")
		require.False(t, skipU)
		require.ElementsMatch(t, []uint{2}, disabledU)

		skipC, disabledC := scope("cve")
		require.False(t, skipC)
		require.ElementsMatch(t, []uint{1}, disabledC)
	})

	t.Run("global cve off, teams ignored", func(t *testing.T) {
		scope := buildChartScopeResolver(
			makeAppCfg(true, false),
			[]*fleet.Team{makeTeam(1, true, true)},
			nil,
		)
		skip, disabled := scope("cve")
		require.True(t, skip)
		require.Nil(t, disabled)
	})

	t.Run("unknown dataset name falls through to no-scope", func(t *testing.T) {
		scope := buildChartScopeResolver(makeAppCfg(true, true), nil, nil)
		skip, disabled := scope("unknown_dataset")
		require.False(t, skip)
		require.Nil(t, disabled)
	})
}

func TestHostVitalsLabelMembershipCronIDP(t *testing.T) {
	ctx := context.Background()
	ds := mysqltest.CreateMySQLDS(t)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "idp-cron-team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "idp-cron-team2"})
	require.NoError(t, err)

	// host0 -> team1, host1 -> team2, host2 -> global (no team).
	hosts := make([]*fleet.Host, 3)
	teamIDs := []*uint{&team1.ID, &team2.ID, nil}
	for i := range 3 {
		h, err := ds.NewHost(ctx, &fleet.Host{
			OsqueryHostID:  new(fmt.Sprintf("idp-cron-%d", i)),
			NodeKey:        new(fmt.Sprintf("idp-cron-%d", i)),
			UUID:           fmt.Sprintf("idp-cron-uuid%d", i),
			Hostname:       fmt.Sprintf("idp-cron-host%d.local", i),
			HardwareSerial: fmt.Sprintf("idp-cron-hwd%d", i),
			Platform:       "darwin",
			TeamID:         teamIDs[i],
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	// All three SCIM users are in the same "Engineering" IdP group.
	scimUserIDs := make([]uint, 3)
	for i := range 3 {
		id, err := ds.CreateScimUser(ctx, &fleet.ScimUser{
			UserName: fmt.Sprintf("idp-cron-user%d", i),
			Active:   new(true),
		})
		require.NoError(t, err)
		scimUserIDs[i] = id
		hostID, scimUserID := hosts[i].ID, id
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				"INSERT INTO host_scim_user (host_id, scim_user_id) VALUES (?, ?)",
				hostID, scimUserID)
			return err
		})
	}
	_, err = ds.CreateScimGroup(ctx, &fleet.ScimGroup{DisplayName: "Engineering", ScimUsers: scimUserIDs})
	require.NoError(t, err)

	criteria, err := json.Marshal(&fleet.HostVitalCriteria{
		Vital: new("end_user_idp_group"),
		Value: new("Engineering"),
	})
	require.NoError(t, err)

	// Create a global and a team1-scoped IdP host vitals label.
	globalLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "idp-cron-global",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeHostVitals,
		HostVitalsCriteria:  new(json.RawMessage(criteria)),
	})
	require.NoError(t, err)
	team1Label, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "idp-cron-team1",
		TeamID:              &team1.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeHostVitals,
		HostVitalsCriteria:  new(json.RawMessage(criteria)),
	})
	require.NoError(t, err)

	// Run the actual cron.
	require.NoError(t, cronHostVitalsLabelMembership(ctx, ds))

	filter := fleet.TeamFilter{User: test.UserAdmin}

	globalHosts, err := ds.ListHostsInLabel(ctx, filter, globalLabel.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	gotGlobal := make([]uint, 0, len(globalHosts))
	for _, h := range globalHosts {
		gotGlobal = append(gotGlobal, h.ID)
	}
	require.ElementsMatch(t, []uint{hosts[0].ID, hosts[1].ID, hosts[2].ID}, gotGlobal)

	team1Hosts, err := ds.ListHostsInLabel(ctx, filter, team1Label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	gotTeam1 := make([]uint, 0, len(team1Hosts))
	for _, h := range team1Hosts {
		gotTeam1 = append(gotTeam1, h.ID)
	}
	require.ElementsMatch(t, []uint{hosts[0].ID}, gotTeam1)
}
