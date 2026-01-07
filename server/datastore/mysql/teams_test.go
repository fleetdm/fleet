package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
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
		{"TeamIDsWithSetupExperienceIdPEnabled", testTeamIDsWithSetupExperienceIdPEnabled},
		{"DefaultTeamConfig", testDefaultTeamConfig},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testTeamsGetSetDelete(t *testing.T, ds *Datastore) {
	user, err := ds.NewUser(t.Context(), &fleet.User{
		Name:       "Foobar",
		Email:      "foobar@example.com",
		Password:   []byte("insecure"),
		GlobalRole: ptr.String(fleet.RoleAdmin),
	})
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, ds)

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

			teamLite, err := ds.TeamLite(context.Background(), team.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.name, teamLite.Name)
			assert.Equal(t, tt.description, teamLite.Description)

			team, err = ds.TeamWithExtras(context.Background(), team.ID)
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
			cp, err := ds.NewMDMAppleConfigProfile(context.Background(), dummyCP, nil)
			require.NoError(t, err)

			wcp, err := ds.NewMDMWindowsConfigProfile(context.Background(), fleet.MDMWindowsConfigProfile{
				Name:   "abc",
				TeamID: &team.ID,
				SyncML: []byte(`<Replace></Replace>`),
			}, nil)
			require.NoError(t, err)

			dec, err := ds.NewMDMAppleDeclaration(context.Background(), &fleet.MDMAppleDeclaration{
				Identifier: "decl-1",
				Name:       "decl-1",
				TeamID:     &team.ID,
				RawJSON:    json.RawMessage(`{"Type": "com.apple.configuration.test", "Identifier": "decl-1"}`),
			})
			require.NoError(t, err)

			teamLabel, err := ds.NewLabel(t.Context(), &fleet.Label{
				Name:   "Team label",
				Query:  "SELECT 1;",
				TeamID: &team.ID,
			})
			require.NoError(t, err)

			// Create host on the team.
			host, err := ds.NewHost(t.Context(), &fleet.Host{
				OsqueryHostID:  ptr.String(fmt.Sprintf("foobar-%d", team.ID)),
				NodeKey:        ptr.String(fmt.Sprintf("foobar-%d", team.ID)),
				UUID:           fmt.Sprintf("foobar-%d", team.ID),
				Hostname:       fmt.Sprintf("foobar-%d.local", team.ID),
				HardwareSerial: fmt.Sprintf("darwin-serial-%d", team.ID),
				Platform:       "darwin",
			})
			require.NoError(t, err)

			// Add a bunch of data into tables with team IDs that should be cleared when team is deleted
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				res, err := q.ExecContext(context.Background(), fmt.Sprintf(`INSERT INTO software_titles (name, source) VALUES ('MyCoolApp_%s', 'apps')`, tt.name))
				if err != nil {
					return err
				}
				titleID, _ := res.LastInsertId()
				_, err = q.ExecContext(
					context.Background(),
					`INSERT INTO
					      mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum)
				     VALUES (?, ?, ?, ?, ?, ?)`,
					fmt.Sprintf("uuid_%s", tt.name),
					0,
					fmt.Sprintf("TestPayloadIdentifier_%s", tt.name),
					fmt.Sprintf("TestPayloadName_%s", tt.name),
					`<?xml version="1.0"`,
					[]byte("test"),
				)
				if err != nil {
					return err
				}
				_, err = q.ExecContext(context.Background(), `
								  INSERT INTO
								      mdm_windows_configuration_profiles (team_id, name, syncml, profile_uuid)
								  VALUES (?, ?, ?, ?)`, 0, fmt.Sprintf("TestPayloadName_%s", tt.name), `<?xml version="1.0"`, fmt.Sprintf("uuid_%s", tt.name))
				if err != nil {
					return err
				}
				_, err = q.ExecContext(context.Background(), `
								  INSERT INTO
								      mdm_android_configuration_profiles (profile_uuid, team_id, name, raw_json)
								  VALUES (?, ?, ?, ?)`, fmt.Sprintf("uuid_%s", tt.name), 0, fmt.Sprintf("TestPayloadName_%s", tt.name), `{"foo": "bar"}`)
				if err != nil {
					return err
				}
				_, err = q.ExecContext(
					context.Background(),
					"INSERT INTO software_title_display_names (team_id, software_title_id, display_name) VALUES (?, ?, ?)",
					team.ID,
					titleID,
					"delete_test",
				)
				if err != nil {
					return err
				}
				_, err = q.ExecContext(
					context.Background(),
					"INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename) VALUES (?, ?, ?, ?)",
					team.ID,
					titleID,
					"delete_test",
					"delete_test.png",
				)
				if err != nil {
					return err
				}

				// Insert the team label on a in-house application.
				res, err = q.ExecContext(context.Background(), fmt.Sprintf(`INSERT INTO software_titles (name, source) VALUES ('ipa_test-%s', 'ipados_apps')`, tt.name))
				if err != nil {
					return err
				}
				inHouseTitleID, _ := res.LastInsertId()
				res, err = q.ExecContext(context.Background(), `INSERT INTO in_house_apps
					(title_id, team_id, global_or_team_id, filename, version, platform, storage_id)
					VALUES (?, ?, ?, 'ipa_test.ipa', '1.0', 'ipados', '1dbbaf76f371ecb4c3dcdcfb53b8915b09ffe6c812586105e1ef1d421eb6fd6b')`,
					inHouseTitleID, team.ID, team.ID)
				if err != nil {
					return err
				}
				inHouseAppID, _ := res.LastInsertId()
				_, err = q.ExecContext(
					context.Background(),
					"INSERT INTO in_house_app_labels (in_house_app_id, label_id) VALUES (?, ?)",
					inHouseAppID,
					teamLabel.ID,
				)
				if err != nil {
					return err
				}

				// Insert the team label on a custom software installer.
				installer, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
				require.NoError(t, err)
				_, _, err = ds.MatchOrCreateSoftwareInstaller(t.Context(), &fleet.UploadSoftwareInstallerPayload{
					InstallScript: "install zoo",
					InstallerFile: installer,
					StorageID:     uuid.NewString(),
					Filename:      "zoo.pkg",
					Title:         "zoo",
					Source:        "apps",
					Version:       "0.0.1",
					UserID:        user.ID,
					TeamID:        &team.ID,
					ValidatedLabels: &fleet.LabelIdentsWithScope{
						LabelScope: fleet.LabelScopeIncludeAny,
						ByName: map[string]fleet.LabelIdent{teamLabel.Name: {
							LabelID:   teamLabel.ID,
							LabelName: teamLabel.Name,
						}},
					},
				})
				require.NoError(t, err)

				// Insert the team label on a VPP app.
				vppApp := &fleet.VPPApp{
					Name: "vpp_app_labels_1" + t.Name(),
					VPPAppTeam: fleet.VPPAppTeam{
						VPPAppID: fleet.VPPAppID{AdamID: "5", Platform: fleet.MacOSPlatform},
						ValidatedLabels: &fleet.LabelIdentsWithScope{
							LabelScope: fleet.LabelScopeIncludeAny,
							ByName: map[string]fleet.LabelIdent{
								teamLabel.Name: {
									LabelID:   teamLabel.ID,
									LabelName: teamLabel.Name,
								},
							},
						},
					},
					BundleIdentifier: "b5",
				}
				_, err = ds.InsertVPPAppWithTeam(t.Context(), vppApp, &team.ID)
				require.NoError(t, err)

				// Record a label membership on the team label.
				err = ds.RecordLabelQueryExecutions(t.Context(), host, map[uint]*bool{teamLabel.ID: ptr.Bool(true)}, time.Now(), false)
				require.NoError(t, err)
				hostLabels, err := ds.ListLabelsForHost(t.Context(), host.ID)
				require.NoError(t, err)
				require.Len(t, hostLabels, 1)
				require.Equal(t, teamLabel.ID, hostLabels[0].ID)

				return nil
			})

			err = ds.DeleteTeam(context.Background(), team.ID)
			require.NoError(t, err)

			// Check that related tables were cleared.
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				for _, table := range teamRefs {
					var count int
					err := sqlx.GetContext(context.Background(), q, &count, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE team_id = ?`, table), team.ID)
					if err != nil {
						return err
					}
					assert.Zerof(t, count, "table: %s", table)
				}
				return nil
			})

			// Check that team-label related tables were cleared.
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				for _, table := range teamLabelsRefs {
					var count int
					err := sqlx.GetContext(context.Background(), q, &count, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE label_id = ?`, table), teamLabel.ID)
					if err != nil {
						return err
					}
					assert.Zerof(t, count, "table: %s", table)
				}
				return nil
			})

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

			// Check team label is gone.
			labels, err := ds.LabelsByName(context.Background(), []string{teamLabel.Name}, fleet.TeamFilter{})
			require.NoError(t, err)
			require.Empty(t, labels)

			hostLabels, err := ds.ListLabelsForHost(t.Context(), host.ID)
			require.NoError(t, err)
			require.Empty(t, hostLabels)
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

	team1, err = ds.TeamWithExtras(context.Background(), team1.ID)
	require.NoError(t, err)
	assert.Len(t, team1.Users, 0)

	team1Users := []fleet.TeamUser{
		{User: user1, Role: "maintainer"},
		{User: user2, Role: "observer"},
	}
	team1.Users = team1Users
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)

	team1, err = ds.TeamWithExtras(context.Background(), team1.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, team1Users, team1.Users)
	// Ensure team 2 not effected
	team2, err = ds.TeamWithExtras(context.Background(), team2.ID)
	require.NoError(t, err)
	assert.Len(t, team2.Users, 0)

	team1Users = []fleet.TeamUser{
		{User: user2, Role: "maintainer"},
	}
	team1.Users = team1Users
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)
	team1, err = ds.TeamWithExtras(context.Background(), team1.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team1Users, team1.Users)

	team2Users := []fleet.TeamUser{
		{User: user2, Role: "observer"},
	}
	team2.Users = team2Users
	team1, err = ds.SaveTeam(context.Background(), team1)
	require.NoError(t, err)
	team1, err = ds.TeamWithExtras(context.Background(), team1.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, team1Users, team1.Users)
	team2, err = ds.SaveTeam(context.Background(), team2)
	require.NoError(t, err)
	team2, err = ds.TeamWithExtras(context.Background(), team2.ID)
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
	require.NoError(t, ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host1.ID})))
	require.NoError(t, ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team2.ID, []uint{host2.ID, host3.ID})))

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

	// Test that ds.TeamsWithExtras returns the same data as ds.ListTeams
	// (except list of users).
	for _, t1 := range teams {
		t2, err := ds.TeamWithExtras(context.Background(), t1.ID)
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
			tm, err := ds.TeamLite(ctx, id)
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
		team, err = ds.TeamWithExtras(ctx, team.ID)
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
		team, err = ds.TeamWithExtras(ctx, team.ID)
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
		team, err = ds.TeamWithExtras(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, defaultMDM, team.Config.MDM)

		teamLite, err := ds.TeamLite(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, defaultMDM, teamLite.Config.MDM)

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
		team, err = ds.TeamWithExtras(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, defaultMDM, team.Config.MDM)

		teamLite, err := ds.TeamLite(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, defaultMDM, teamLite.Config.MDM)

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
						ManualAgentInstall:  optjson.SetBool(true),
					},
					WindowsSettings: fleet.WindowsSettings{
						CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}),
					},
					AndroidSettings: fleet.AndroidSettings{
						CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "qux"}}),
						Certificates:   optjson.Slice[fleet.CertificateTemplateSpec]{Set: true, Value: []fleet.CertificateTemplateSpec{}},
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
				UpdateNewHosts: optjson.Bool{Set: true},
			},
			IOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("11.11.11"),
				Deadline:       optjson.SetString("2024-04-04"),
				UpdateNewHosts: optjson.Bool{Set: true},
			},
			IPadOSUpdates: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("12.12.12"),
				Deadline:       optjson.SetString("2023-03-03"),
				UpdateNewHosts: optjson.Bool{Set: true},
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
				ManualAgentInstall:          optjson.SetBool(true),
			},
			WindowsSettings: fleet.WindowsSettings{
				CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}),
			},
			AndroidSettings: fleet.AndroidSettings{
				CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "qux"}}),
				Certificates:   optjson.Slice[fleet.CertificateTemplateSpec]{Set: true, Value: []fleet.CertificateTemplateSpec{}},
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

func testTeamIDsWithSetupExperienceIdPEnabled(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// no team exists (well, except "no team" but not enabled there)
	teamIDs, err := ds.TeamIDsWithSetupExperienceIdPEnabled(ctx)
	require.NoError(t, err)
	require.Empty(t, teamIDs)

	// create a couple teams
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "A"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "B"})
	require.NoError(t, err)

	// still no team with IdP enabled
	teamIDs, err = ds.TeamIDsWithSetupExperienceIdPEnabled(ctx)
	require.NoError(t, err)
	require.Empty(t, teamIDs)
	_, _ = team1, team2

	// enable it for "No team"
	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	ac.MDM.MacOSSetup.EnableEndUserAuthentication = true
	err = ds.SaveAppConfig(ctx, ac)
	require.NoError(t, err)

	teamIDs, err = ds.TeamIDsWithSetupExperienceIdPEnabled(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{0}, teamIDs)

	// enable it for team1
	team1.Config.MDM.MacOSSetup.EnableEndUserAuthentication = true
	_, err = ds.SaveTeam(ctx, team1)
	require.NoError(t, err)

	teamIDs, err = ds.TeamIDsWithSetupExperienceIdPEnabled(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{0, team1.ID}, teamIDs)

	// enable it for team2
	team2.Config.MDM.MacOSSetup.EnableEndUserAuthentication = true
	_, err = ds.SaveTeam(ctx, team2)
	require.NoError(t, err)

	teamIDs, err = ds.TeamIDsWithSetupExperienceIdPEnabled(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{0, team1.ID, team2.ID}, teamIDs)

	// disable it for team1
	team1.Config.MDM.MacOSSetup.EnableEndUserAuthentication = false
	_, err = ds.SaveTeam(ctx, team1)
	require.NoError(t, err)

	teamIDs, err = ds.TeamIDsWithSetupExperienceIdPEnabled(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{0, team2.ID}, teamIDs)

	// delete team2
	err = ds.DeleteTeam(ctx, team2.ID)
	require.NoError(t, err)

	teamIDs, err = ds.TeamIDsWithSetupExperienceIdPEnabled(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint{0}, teamIDs)
}

func testDefaultTeamConfig(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Test getting default config (should be initialized by migration)
	config, err := ds.DefaultTeamConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Default config should have webhook disabled
	assert.False(t, config.WebhookSettings.FailingPoliciesWebhook.Enable)
	assert.Empty(t, config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	assert.Empty(t, config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)

	// Test saving a new config
	newConfig := &fleet.TeamConfig{
		WebhookSettings: fleet.TeamWebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:         true,
				DestinationURL: "https://example.com/webhook",
				PolicyIDs:      []uint{1, 2, 3},
				HostBatchSize:  100,
			},
		},
		Features: fleet.Features{
			EnableHostUsers:         true,
			EnableSoftwareInventory: true,
		},
	}

	err = ds.SaveDefaultTeamConfig(ctx, newConfig)
	require.NoError(t, err)

	// Verify the config was saved
	savedConfig, err := ds.DefaultTeamConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, savedConfig)

	assert.True(t, savedConfig.WebhookSettings.FailingPoliciesWebhook.Enable)
	assert.Equal(t, "https://example.com/webhook", savedConfig.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	assert.Equal(t, []uint{1, 2, 3}, savedConfig.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	assert.Equal(t, 100, savedConfig.WebhookSettings.FailingPoliciesWebhook.HostBatchSize)

	// Test updating existing config
	updatedConfig := &fleet.TeamConfig{
		WebhookSettings: fleet.TeamWebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:         false,
				DestinationURL: "https://updated.com/webhook",
				PolicyIDs:      []uint{4, 5},
				HostBatchSize:  50,
			},
		},
	}

	err = ds.SaveDefaultTeamConfig(ctx, updatedConfig)
	require.NoError(t, err)

	// Verify the update
	finalConfig, err := ds.DefaultTeamConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, finalConfig)

	assert.False(t, finalConfig.WebhookSettings.FailingPoliciesWebhook.Enable)
	assert.Equal(t, "https://updated.com/webhook", finalConfig.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	assert.Equal(t, []uint{4, 5}, finalConfig.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	assert.Equal(t, 50, finalConfig.WebhookSettings.FailingPoliciesWebhook.HostBatchSize)
}
