package mysql

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeams(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetSetDelete", testTeamsGetSetDelete},
		{"Users", testTeamsUsers},
		{"List", testTeamsList},
		{"Summary", testTeamsSummary},
		{"Search", testTeamsSearch},
		{"EnrollSecrets", testTeamsEnrollSecrets},
		{"TeamAgentOptions", testTeamsAgentOptions},
		{"DeleteIntegrationsFromTeams", testTeamsDeleteIntegrationsFromTeams},
		{"TeamsFeatures", testTeamsFeatures},
		{"TeamsMDMConfig", testTeamsMDMConfig},
		{"TestTeamsNameUnicode", testTeamsNameUnicode},
		{"TestTeamsNameEmoji", testTeamsNameEmoji},
		{"TestTeamsNameSort", testTeamsNameSort},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testTeamsGetSetDelete(t *testing.T, ds *Datastore) {
	createTests := []struct {
		name, description string
	}{
		{"foo_team", "foobar is the description"},
		{"bar_team", "were you hoping for more?"},
	}

	for _, tt := range createTests {
		t.Run(tt.name, func(t *testing.T) {
			team, err := ds.NewTeam(context.Background(), &fleet.Team{
				Name:        tt.name,
				Description: tt.description,
			})
			require.NoError(t, err)
			assert.NotZero(t, team.ID)

			team, err = ds.Team(context.Background(), team.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.name, team.Name)
			assert.Equal(t, tt.description, team.Description)

			team, err = ds.TeamByName(context.Background(), tt.name)
			require.NoError(t, err)
			assert.Equal(t, tt.name, team.Name)
			assert.Equal(t, tt.description, team.Description)

			p, err := ds.NewPack(context.Background(), &fleet.Pack{
				Name:    tt.name,
				TeamIDs: []uint{team.ID},
			})
			require.NoError(t, err)

			dummyMC := mobileconfig.Mobileconfig([]byte("DummyTestMobileconfigBytes"))
			dummyCP := fleet.MDMAppleConfigProfile{
				Name:         "DummyTestName",
				Identifier:   "DummyTestIdentifier",
				Mobileconfig: dummyMC,
				TeamID:       &team.ID,
			}
			cp, err := ds.NewMDMAppleConfigProfile(context.Background(), dummyCP)
			require.NoError(t, err)

			wcp, err := ds.NewMDMWindowsConfigProfile(context.Background(), fleet.MDMWindowsConfigProfile{
				Name:   "abc",
				TeamID: &team.ID,
				SyncML: []byte(`<Replace></Replace>`),
			})
			require.NoError(t, err)

			dec, err := ds.NewMDMAppleDeclaration(context.Background(), &fleet.MDMAppleDeclaration{
				Identifier: "decl-1",
				Name:       "decl-1",
				TeamID:     &team.ID,
			})
			require.NoError(t, err)

			err = ds.DeleteTeam(context.Background(), team.ID)
			require.NoError(t, err)

			newP, err := ds.Pack(context.Background(), p.ID)
			require.NoError(t, err)
			require.Empty(t, newP.Teams)

			_, err = ds.TeamByName(context.Background(), tt.name)
			require.Error(t, err)

			_, err = ds.GetMDMAppleConfigProfile(context.Background(), cp.ProfileUUID)
			var nfe fleet.NotFoundError
			require.ErrorAs(t, err, &nfe)

			_, err = ds.GetMDMWindowsConfigProfile(context.Background(), wcp.ProfileUUID)
			require.ErrorAs(t, err, &nfe)

			_, err = ds.GetMDMAppleConfigProfile(context.Background(), dec.DeclarationUUID)
			require.ErrorAs(t, err, &nfe)

			require.NoError(t, ds.DeletePack(context.Background(), newP.Name))
		})
	}
}

func testTeamsUsers(t *testing.T, ds *Datastore) {
	users := createTestUsers(t, ds)
	user1 := fleet.User{Name: users[0].Name, Email: users[0].Email, ID: users[0].ID}
	user2 := fleet.User{Name: users[1].Name, Email: users[1].Email, ID: users[1].ID}

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	team1, err = ds.Team(context.Background(), team1.ID)
	require.NoError(t, err)
	assert.Len(t, team1.Users, 0)

	team1Users := []fleet.TeamUser{
		{User: user1, Role: "maintainer"},
		{User: user2, Role: "observer"},
	}
	team1.Users = team1Users
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)

	team1, err = ds.Team(context.Background(), team1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, team1Users, team1.Users)
	// Ensure team 2 not effected
	team2, err = ds.Team(context.Background(), team2.ID)
	require.NoError(t, err)
	assert.Len(t, team2.Users, 0)

	team1Users = []fleet.TeamUser{
		{User: user2, Role: "maintainer"},
	}
	team1.Users = team1Users
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)
	team1, err = ds.Team(context.Background(), team1.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team1Users, team1.Users)

	team2Users := []fleet.TeamUser{
		{User: user2, Role: "observer"},
	}
	team2.Users = team2Users
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)
	team1, err = ds.Team(context.Background(), team1.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team1Users, team1.Users)
	team2, err = ds.SaveTeam(context.Background(), team2)
	require.NoError(t, err)
	team2, err = ds.Team(context.Background(), team2.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team2Users, team2.Users)
}

func testTeamsList(t *testing.T, ds *Datastore) {
	users := createTestUsers(t, ds)
	user1 := fleet.User{Name: users[0].Name, Email: users[0].Email, ID: users[0].ID, GlobalRole: ptr.String(fleet.RoleAdmin)}
	user2 := fleet.User{Name: users[1].Name, Email: users[1].Email, ID: users[1].ID, GlobalRole: ptr.String(fleet.RoleObserver)}

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	teams, err := ds.ListTeams(context.Background(), fleet.TeamFilter{User: &user1}, fleet.ListOptions{})
	require.NoError(t, err)
	sort.Slice(teams, func(i, j int) bool { return teams[i].Name < teams[j].Name })

	assert.Equal(t, "team1", teams[0].Name)
	assert.Equal(t, 0, teams[0].HostCount)
	assert.Equal(t, 0, teams[0].UserCount)

	assert.Equal(t, "team2", teams[1].Name)
	assert.Equal(t, 0, teams[1].HostCount)
	assert.Equal(t, 0, teams[1].UserCount)

	host1 := test.NewHost(t, ds, "1", "1", "1", "1", time.Now())
	host2 := test.NewHost(t, ds, "2", "2", "2", "2", time.Now())
	host3 := test.NewHost(t, ds, "3", "3", "3", "3", time.Now())
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{host2.ID, host3.ID}))

	team1.Users = []fleet.TeamUser{
		{User: user1, Role: "maintainer"},
		{User: user2, Role: "observer"},
	}
	_, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)

	team2.Users = []fleet.TeamUser{
		{User: user1, Role: "maintainer"},
	}
	_, err = ds.SaveTeam(context.Background(), team2)
	require.NoError(t, err)

	teams, err = ds.ListTeams(context.Background(), fleet.TeamFilter{User: &user1}, fleet.ListOptions{})
	require.NoError(t, err)
	sort.Slice(teams, func(i, j int) bool { return teams[i].Name < teams[j].Name })

	assert.Equal(t, "team1", teams[0].Name)
	assert.Equal(t, 1, teams[0].HostCount)
	assert.Equal(t, 2, teams[0].UserCount)

	assert.Equal(t, "team2", teams[1].Name)
	assert.Equal(t, 2, teams[1].HostCount)
	assert.Equal(t, 1, teams[1].UserCount)

	// Test that ds.Teams returns the same data as ds.ListTeams
	// (except list of users).
	for _, t1 := range teams {
		t2, err := ds.Team(context.Background(), t1.ID)
		require.NoError(t, err)
		t2.Users = nil
		require.Equal(t, t1, t2)
	}
}

func testTeamsSummary(t *testing.T, ds *Datastore) {
	_, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "ts1"})
	require.NoError(t, err)
	_, err = ds.NewTeam(context.Background(), &fleet.Team{Name: "ts2"})
	require.NoError(t, err)

	teams, err := ds.TeamsSummary(context.Background())
	require.NoError(t, err)
	sort.Slice(teams, func(i, j int) bool { return teams[i].Name < teams[j].Name })

	assert.Equal(t, "ts1", teams[0].Name)
	assert.Equal(t, "ts2", teams[1].Name)
}

func testTeamsSearch(t *testing.T, ds *Datastore) {
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team3, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "foobar"})
	require.NoError(t, err)
	team4, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "floobar"})
	require.NoError(t, err)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	filter := fleet.TeamFilter{User: user}

	teams, err := ds.SearchTeams(context.Background(), filter, "")
	require.NoError(t, err)
	assert.Len(t, teams, 4)

	teams, err = ds.SearchTeams(context.Background(), filter, "", team1.ID, team2.ID, team3.ID)
	require.NoError(t, err)
	assert.Len(t, teams, 1)
	assert.Equal(t, team4.Name, teams[0].Name)

	teams, err = ds.SearchTeams(context.Background(), filter, "oo", team1.ID, team2.ID, team3.ID)
	require.NoError(t, err)
	assert.Len(t, teams, 1)
	assert.Equal(t, team4.Name, teams[0].Name)

	teams, err = ds.SearchTeams(context.Background(), filter, "oo")
	require.NoError(t, err)
	assert.Len(t, teams, 2)

	teams, err = ds.SearchTeams(context.Background(), filter, "none")
	require.NoError(t, err)
	assert.Len(t, teams, 0)
}

func testTeamsEnrollSecrets(t *testing.T, ds *Datastore) {
	secrets := []*fleet.EnrollSecret{{Secret: "secret1"}, {Secret: "secret2"}}
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:    "team1",
		Secrets: secrets,
	})
	require.NoError(t, err)

	enrollSecrets, err := ds.TeamEnrollSecrets(context.Background(), team1.ID)
	require.NoError(t, err)

	var justSecrets []*fleet.EnrollSecret
	for _, secret := range enrollSecrets {
		require.NotNil(t, secret.TeamID)
		assert.Equal(t, team1.ID, *secret.TeamID)
		justSecrets = append(justSecrets, &fleet.EnrollSecret{Secret: secret.Secret})
	}
	test.ElementsMatchSkipTimestampsID(t, secrets, justSecrets)
}

func testTeamsAgentOptions(t *testing.T, ds *Datastore) {
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	teamAgentOptions1, err := ds.TeamAgentOptions(context.Background(), team1.ID)
	require.NoError(t, err)
	require.Nil(t, teamAgentOptions1)

	agentOptions := json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team2",
		Config: fleet.TeamConfig{
			AgentOptions: &agentOptions,
		},
	})
	require.NoError(t, err)

	teamAgentOptions2, err := ds.TeamAgentOptions(context.Background(), team2.ID)
	require.NoError(t, err)
	require.JSONEq(t, string(agentOptions), string(*teamAgentOptions2))
}

func testTeamsDeleteIntegrationsFromTeams(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	urla, urlb, urlc, urld, urle, urlf, urlg := "http://a.com", "http://b.com", "http://c.com", "http://d.com", "http://e.com", "http://f.com", "http://g.com"

	// create some teams
	team1, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
		Config: fleet.TeamConfig{
			Integrations: fleet.TeamIntegrations{
				Jira: []*fleet.TeamJiraIntegration{
					{URL: urla, ProjectKey: "A"},
					{URL: urlb, ProjectKey: "B"},
				},
				Zendesk: []*fleet.TeamZendeskIntegration{
					{URL: urlc, GroupID: 1},
					{URL: urld, GroupID: 2},
				},
			},
		},
	})
	require.NoError(t, err)

	team2, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "team2",
		Config: fleet.TeamConfig{
			Integrations: fleet.TeamIntegrations{
				Jira: []*fleet.TeamJiraIntegration{
					{URL: urla, ProjectKey: "A"},
					{URL: urle, ProjectKey: "E"},
				},
				Zendesk: []*fleet.TeamZendeskIntegration{
					{URL: urlc, GroupID: 1},
					{URL: urlf, GroupID: 3},
				},
			},
		},
	})
	require.NoError(t, err)

	team3, err := ds.NewTeam(ctx, &fleet.Team{
		Name: "team3",
		Config: fleet.TeamConfig{
			Integrations: fleet.TeamIntegrations{
				Jira: []*fleet.TeamJiraIntegration{
					{URL: urle, ProjectKey: "E"},
				},
				Zendesk: []*fleet.TeamZendeskIntegration{
					{URL: urlf, GroupID: 3},
				},
			},
		},
	})
	require.NoError(t, err)

	assertIntgURLs := func(wantTm1, wantTm2, wantTm3 []string) {
		// assert that the integrations' URLs of each team corresponds to the
		// expected values
		expected := [][]string{wantTm1, wantTm2, wantTm3}
		for i, id := range []uint{team1.ID, team2.ID, team3.ID} {
			tm, err := ds.Team(ctx, id)
			require.NoError(t, err)

			var urls []string
			for _, j := range tm.Config.Integrations.Jira {
				urls = append(urls, j.URL)
			}
			for _, z := range tm.Config.Integrations.Zendesk {
				urls = append(urls, z.URL)
			}

			want := expected[i]
			require.ElementsMatch(t, want, urls)
		}
	}

	// delete nothing
	err = ds.DeleteIntegrationsFromTeams(context.Background(), fleet.Integrations{})
	require.NoError(t, err)
	assertIntgURLs([]string{urla, urlb, urlc, urld}, []string{urla, urle, urlc, urlf}, []string{urle, urlf})

	// delete a, b, c (in the url) so that team1 and team2 are impacted
	err = ds.DeleteIntegrationsFromTeams(context.Background(), fleet.Integrations{
		Jira: []*fleet.JiraIntegration{
			{URL: urla, ProjectKey: "A"},
			{URL: urlb, ProjectKey: "B"},
		},
		Zendesk: []*fleet.ZendeskIntegration{
			{URL: urlc, GroupID: 1},
		},
	})
	require.NoError(t, err)
	assertIntgURLs([]string{urld}, []string{urle, urlf}, []string{urle, urlf})

	// delete g, no team is impacted
	err = ds.DeleteIntegrationsFromTeams(context.Background(), fleet.Integrations{
		Jira: []*fleet.JiraIntegration{
			{URL: urlg, ProjectKey: "G"},
		},
	})
	require.NoError(t, err)
	assertIntgURLs([]string{urld}, []string{urle, urlf}, []string{urle, urlf})
}

func testTeamsFeatures(t *testing.T, ds *Datastore) {
	defaultFeatures := fleet.Features{}
	defaultFeatures.ApplyDefaultsForNewInstalls()
	ctx := context.Background()

	t.Run("NULL config in the database", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team_null_config"})
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err = tx.ExecContext(
				ctx,
				"UPDATE teams SET config = NULL WHERE id = ?",
				team.ID,
			)
			return err
		})

		features, err := ds.TeamFeatures(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, &defaultFeatures, features)

		// retrieving a team also returns a team with the default
		// features
		team, err = ds.Team(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, defaultFeatures, team.Config.Features)

		team, err = ds.TeamByName(ctx, team.Name)
		require.NoError(t, err)
		assert.Equal(t, defaultFeatures, team.Config.Features)
	})

	t.Run("NULL config.features in the database", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team_null_config_features"})
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err = tx.ExecContext(
				ctx,
				"UPDATE teams SET config = '{}' WHERE id = ?",
				team.ID,
			)
			return err
		})

		features, err := ds.TeamFeatures(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, &defaultFeatures, features)

		// retrieving a team also returns a team with the default
		// features
		team, err = ds.Team(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, defaultFeatures, team.Config.Features)

		team, err = ds.TeamByName(ctx, team.Name)
		require.NoError(t, err)
		assert.Equal(t, defaultFeatures, team.Config.Features)
	})

	t.Run("saves and retrieves configs", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{
			Name: "team1",
			Config: fleet.TeamConfig{
				Features: fleet.Features{
					EnableHostUsers:         false,
					EnableSoftwareInventory: false,
					AdditionalQueries:       nil,
				},
			},
		})
		require.NoError(t, err)
		features, err := ds.TeamFeatures(ctx, team.ID)
		require.NoError(t, err)

		assert.Equal(t, &fleet.Features{
			EnableHostUsers:         false,
			EnableSoftwareInventory: false,
			AdditionalQueries:       nil,
		}, features)
	})
}

func testTeamsMDMConfig(t *testing.T, ds *Datastore) {
	defaultMDM := fleet.TeamMDM{}
	ctx := context.Background()

	t.Run("NULL config in the database", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team_null_config"})
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err = tx.ExecContext(
				ctx,
				"UPDATE teams SET config = NULL WHERE id = ?",
				team.ID,
			)
			return err
		})

		mdm, err := ds.TeamMDMConfig(ctx, team.ID)
		require.NoError(t, err)
		assert.Nil(t, mdm)

		// retrieving a team also returns a team with the default
		// settings
		team, err = ds.Team(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, defaultMDM, team.Config.MDM)

		team, err = ds.TeamByName(ctx, team.Name)
		require.NoError(t, err)
		assert.Equal(t, defaultMDM, team.Config.MDM)
	})

	t.Run("NULL config.mdm in the database", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team_null_config_mdm"})
		require.NoError(t, err)
		ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			_, err = tx.ExecContext(
				ctx,
				"UPDATE teams SET config = '{}' WHERE id = ?",
				team.ID,
			)
			return err
		})

		mdm, err := ds.TeamMDMConfig(ctx, team.ID)
		require.NoError(t, err)
		assert.Nil(t, mdm)

		// retrieving a team also returns a team with the default
		// settings
		team, err = ds.Team(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, defaultMDM, team.Config.MDM)

		team, err = ds.TeamByName(ctx, team.Name)
		require.NoError(t, err)
		assert.Equal(t, defaultMDM, team.Config.MDM)
	})

	t.Run("saves and retrieves configs", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{
			Name: "team1",
			Config: fleet.TeamConfig{
				MDM: fleet.TeamMDM{
					MacOSUpdates: fleet.AppleOSUpdateSettings{
						MinimumVersion: optjson.SetString("10.15.0"),
						Deadline:       optjson.SetString("2025-10-01"),
					},
					IOSUpdates: fleet.AppleOSUpdateSettings{
						MinimumVersion: optjson.SetString("11.11.11"),
						Deadline:       optjson.SetString("2024-04-04"),
					},
					IPadOSUpdates: fleet.AppleOSUpdateSettings{
						MinimumVersion: optjson.SetString("12.12.12"),
						Deadline:       optjson.SetString("2023-03-03"),
					},
					WindowsUpdates: fleet.WindowsUpdates{
						DeadlineDays:    optjson.SetInt(7),
						GracePeriodDays: optjson.SetInt(3),
					},
					MacOSSetup: fleet.MacOSSetup{
						BootstrapPackage:    optjson.SetString("bootstrap"),
						MacOSSetupAssistant: optjson.SetString("assistant"),
					},
					WindowsSettings: fleet.WindowsSettings{
						CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}),
					},
				},
			},
		})
		require.NoError(t, err)
		mdm, err := ds.TeamMDMConfig(ctx, team.ID)
		require.NoError(t, err)

		assert.Equal(t, &fleet.TeamMDM{
			MacOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("10.15.0"),
				Deadline:       optjson.SetString("2025-10-01"),
			},
			IOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("11.11.11"),
				Deadline:       optjson.SetString("2024-04-04"),
			},
			IPadOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("12.12.12"),
				Deadline:       optjson.SetString("2023-03-03"),
			},
			WindowsUpdates: fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(7),
				GracePeriodDays: optjson.SetInt(3),
			},
			MacOSSetup: fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString("bootstrap"),
				MacOSSetupAssistant:         optjson.SetString("assistant"),
				EnableReleaseDeviceManually: optjson.SetBool(false),
				Script:                      optjson.String{Set: true},
				Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
			},
			WindowsSettings: fleet.WindowsSettings{
				CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}),
			},
		}, mdm)
	})
}

func testTeamsNameUnicode(t *testing.T, ds *Datastore) {
	var equivalentNames []string
	item, _ := strconv.Unquote(`"\uAC00"`) // 가
	equivalentNames = append(equivalentNames, item)
	item, _ = strconv.Unquote(`"\u1100\u1161"`) // ᄀ + ᅡ
	equivalentNames = append(equivalentNames, item)

	// Save team
	team, err := ds.NewTeam(context.Background(), &fleet.Team{Name: equivalentNames[0]})
	require.NoError(t, err)
	assert.Equal(t, equivalentNames[0], team.Name)

	// Try to create team with equivalent name
	_, err = ds.NewTeam(context.Background(), &fleet.Team{Name: equivalentNames[1]})
	assert.True(t, IsDuplicate(err), err)

	// Try to update a different team with equivalent name -- not allowed
	teamEmoji, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "💻"})
	require.NoError(t, err)
	_, err = ds.SaveTeam(context.Background(), &fleet.Team{ID: teamEmoji.ID, Name: equivalentNames[1]})
	assert.True(t, IsDuplicate(err), err)

	// Try to find team with equivalent name
	teamFilter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
	results, err := ds.ListTeams(context.Background(), teamFilter, fleet.ListOptions{MatchQuery: equivalentNames[1]})
	assert.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, equivalentNames[0], results[0].Name)

	results, err = ds.SearchTeams(context.Background(), teamFilter, equivalentNames[1])
	assert.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, equivalentNames[0], results[0].Name)

	result, err := ds.TeamByName(context.Background(), equivalentNames[1])
	assert.NoError(t, err)
	assert.Equal(t, equivalentNames[0], result.Name)
}

func testTeamsNameEmoji(t *testing.T, ds *Datastore) {
	// Try to save teams with emojis
	emoji0 := "🔥"
	_, err := ds.NewTeam(context.Background(), &fleet.Team{Name: emoji0})
	require.NoError(t, err)
	emoji1 := "💻"
	teamEmoji, err := ds.NewTeam(context.Background(), &fleet.Team{Name: emoji1})
	require.NoError(t, err)
	assert.Equal(t, emoji1, teamEmoji.Name)

	// Try to find team with emoji0
	teamFilter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
	results, err := ds.ListTeams(context.Background(), teamFilter, fleet.ListOptions{MatchQuery: emoji0})
	assert.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, emoji0, results[0].Name)

	// Try to find team with emoji1
	results, err = ds.SearchTeams(context.Background(), teamFilter, emoji1)
	assert.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, emoji1, results[0].Name)
}

// Ensure case-insensitive sort order for ames
func testTeamsNameSort(t *testing.T, ds *Datastore) {
	var teams [3]*fleet.Team
	var err error
	// Save teams
	teams[1], err = ds.NewTeam(context.Background(), &fleet.Team{Name: "В"})
	require.NoError(t, err)
	teams[2], err = ds.NewTeam(context.Background(), &fleet.Team{Name: "о"})
	require.NoError(t, err)
	teams[0], err = ds.NewTeam(context.Background(), &fleet.Team{Name: "а"})
	require.NoError(t, err)

	teamFilter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
	results, err := ds.ListTeams(context.Background(), teamFilter, fleet.ListOptions{OrderKey: "name"})
	assert.NoError(t, err)
	require.Len(t, teams, 3)
	for i, item := range teams {
		assert.Equal(t, item.Name, results[i].Name)
	}
}
