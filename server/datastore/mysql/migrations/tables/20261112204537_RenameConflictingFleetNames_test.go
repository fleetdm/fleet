package tables

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

func TestUp_20261112204537(t *testing.T) {
	// Each name is inserted in order, so its index is its id order.
	insertFleets := func(t *testing.T, db *sqlx.DB, names []string) []int64 {
		ids := make([]int64, 0, len(names))
		for _, name := range names {
			ids = append(ids, execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES (?)`, name))
		}
		return ids
	}
	fleetNames := func(t *testing.T, db *sqlx.DB) []string {
		names := []string{}
		require.NoError(t, db.Select(&names, `SELECT name FROM teams ORDER BY id`))
		return names
	}
	mustJSON := func(t *testing.T, v any) string {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return string(b)
	}
	captureLog := func(t *testing.T) *bytes.Buffer {
		var buf bytes.Buffer
		original := logger.Info.Writer()
		logger.Info.SetOutput(&buf)
		t.Cleanup(func() { logger.Info.SetOutput(original) })
		return &buf
	}

	kanji253 := strings.Repeat("界", 253)

	renameCases := []struct {
		name   string
		before []string
		after  []string
	}{
		{
			name:   "name key and no letter or number",
			before: []string{"Workstations", "work-stations", "Workstations!", "🚀", "!!!"},
			after:  []string{"Workstations", "work-stations 2", "Workstations! 3", "🚀 1", "!!! 2"},
		},
		{
			name:   "ignored characters",
			before: []string{"Workstations", "Work stations", "WORK_STATIONS", "Workstations 💻", "Wörkstations", "Ｗorkstations"},
			after:  []string{"Workstations", "Work stations 2", "WORK_STATIONS 3", "Workstations 💻 4", "Wörkstations 5", "Ｗorkstations 6"},
		},
		{
			name:   "number taken by earlier fleet",
			before: []string{"Workstations", "Workstations 2", "workstations", "WORKSTATIONS"},
			after:  []string{"Workstations", "Workstations 2", "workstations 3", "WORKSTATIONS 4"},
		},
		{
			name:   "number taken by later fleet",
			before: []string{"Workstations", "workstations", "Workstations 2", "Workstations-3"},
			after:  []string{"Workstations", "workstations 4", "Workstations 2", "Workstations-3"},
		},
		{
			name:   "number taken under collation",
			before: []string{"Team 𝐀", "Team 𝐁", "Team 𝐂 2"},
			after:  []string{"Team 𝐀", "Team 𝐁 3", "Team 𝐂 2"},
		},
		{
			name:   "no letter or number skips numbers",
			before: []string{"1", "🚀", "---", "🚀🚀🚀", "「」", "・・・", "2"},
			after:  []string{"1", "🚀 3", "--- 4", "🚀🚀🚀 5", "「」 6", "・・・ 7", "2"},
		},
		{
			name:   "collation duplicates",
			before: []string{"ABC", "abc", "Renee", "Reneé", "ＡＢＣ"},
			after:  []string{"ABC", "abc 2", "Renee", "Reneé 2", "ＡＢＣ 3"},
		},
		{
			name:   "collation only matches",
			before: []string{"Team 𝐀", "Team 𝐁", "はは", "ハハ", "パパ"},
			after:  []string{"Team 𝐀", "Team 𝐁 2", "はは", "ハハ 2", "パパ 3"},
		},
		{
			name:   "full case folding",
			before: []string{"straße", "strasse", "Strasse!"},
			after:  []string{"straße", "strasse 2", "Strasse! 3"},
		},
		{
			name:   "non Latin names",
			before: []string{"開発チーム", "開発・チーム", "開発チーム。", "開発", "Ingeniería", "2026"},
			after:  []string{"開発チーム", "開発・チーム 2", "開発チーム。 3", "開発", "Ingeniería", "2026"},
		},
		{
			// "Team 𝐁" only matches "Team 𝐀" under the collation and "Team B!" only matches "Team 𝐁" by name key, so
			// once "Team 𝐁" is renamed nothing matches "Team B!".
			name:   "matches do not chain",
			before: []string{"Team 𝐀", "Team 𝐁", "Team B!"},
			after:  []string{"Team 𝐀", "Team 𝐁 2", "Team B!"},
		},
		{
			// "Team 𝐀!" matched "Team 𝐁!" under the collation before the migration, but not the name it was given.
			name:   "collation match was renamed",
			before: []string{"Team B", "Team 𝐁!", "Team 𝐀!"},
			after:  []string{"Team B", "Team 𝐁! 2", "Team 𝐀!"},
		},
		{
			name:   "name shortened to fit",
			before: []string{kanji253, kanji253 + "!", kanji253 + "?!", strings.Repeat("x", 255), strings.Repeat("X", 255)},
			after:  []string{kanji253, kanji253 + " 2", kanji253 + " 3", strings.Repeat("x", 255), strings.Repeat("X", 253) + " 2"},
		},
		{
			name:   "whitespace not trimmed",
			before: []string{" Lab", "Kiosks ", "Lab", "  Lab  "},
			after:  []string{" Lab", "Kiosks ", "Lab 2", "  Lab   3"},
		},
		{
			name:   "nothing to rename",
			before: []string{"Workstations", "Workstations 2", "Servers", "サーバー", "Fleet 🚀"},
			after:  []string{"Workstations", "Workstations 2", "Servers", "サーバー", "Fleet 🚀"},
		},
		{
			name:  "no fleets",
			after: []string{},
		},
	}
	// Subtest names become database names, so they are short and have no punctuation.
	for _, tc := range renameCases {
		t.Run(tc.name, func(t *testing.T) {
			db := applyUpToPrev(t)
			ids := insertFleets(t, db, tc.before)
			var appConfigBefore string
			require.NoError(t, db.Get(&appConfigBefore, `SELECT CONCAT(json_value, updated_at) FROM app_config_json WHERE id = 1`))
			logged := captureLog(t)

			applyNext(t, db)

			after := fleetNames(t, db)
			require.Equal(t, tc.after, after)

			var wantLog strings.Builder
			for i, name := range after {
				assert.LessOrEqual(t, utf8.RuneCountInString(name), 255)
				assert.NotEmpty(t, fleetNameKey20261112204537(name))
				if name != tc.before[i] {
					fmt.Fprintf(&wantLog, "Renamed fleet ID %d from %q to %q to follow the Fleet 5 fleet name rules.\n", ids[i], tc.before[i], name)
				}
			}
			// The logger prefixes each line with "INFO: " and the time.
			var gotLog strings.Builder
			for line := range strings.Lines(logged.String()) {
				_, msg, _ := strings.Cut(line, "Renamed")
				gotLog.WriteString("Renamed" + msg)
			}
			assert.Equal(t, wantLog.String(), gotLog.String())

			// No fleet matches another by name key or under the collation.
			keys := make(map[string]string)
			for _, name := range after {
				key := fleetNameKey20261112204537(name)
				assert.NotContains(t, keys, key, "%q and %q have the same name key", keys[key], name)
				keys[key] = name
			}
			var collationMatches int
			require.NoError(t, db.Get(&collationMatches, `SELECT COUNT(*) FROM teams a JOIN teams b ON a.name = b.name AND a.id < b.id`))
			assert.Zero(t, collationMatches)

			var appConfigAfter string
			require.NoError(t, db.Get(&appConfigAfter, `SELECT CONCAT(json_value, updated_at) FROM app_config_json WHERE id = 1`))
			assert.Equal(t, appConfigBefore, appConfigAfter, "app config has no fleet name copies to rewrite")
		})
	}

	t.Run("only the name changes", func(t *testing.T) {
		db := applyUpToPrev(t)

		execNoErr(t, db, `INSERT INTO users (id, name, email, password, salt) VALUES (1, 'Admin', 'admin@example.com', 'x', 'x')`)
		execNoErr(t, db, `INSERT INTO teams (id, name, description, config, filename, created_at) VALUES
			(1, 'Workstations', 'kept', '{"features": {"enable_host_users": true}}', 'fleets/workstations.yml', '2025-01-02 03:04:05'),
			(2, 'work-stations', 'renamed', '{"features": {"enable_host_users": false}}', 'fleets/work-stations.yml', '2025-02-03 04:05:06'),
			(3, '🚀', 'rocket', NULL, NULL, '2025-03-04 05:06:07')`)
		execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform, team_id) VALUES
			('h1', 'k1', 'host1', 'u1', 'darwin', 1), ('h2', 'k2', 'host2', 'u2', 'darwin', 2), ('h3', 'k3', 'host3', 'u3', 'windows', 3)`)
		execNoErr(t, db, `INSERT INTO enroll_secrets (secret, team_id) VALUES ('s1', 1), ('s2', 2), ('s3', 3)`)
		execNoErr(t, db, `INSERT INTO user_teams (user_id, team_id, role) VALUES (1, 1, 'admin'), (1, 2, 'maintainer'), (1, 3, 'observer')`)
		execNoErr(t, db, `INSERT INTO abm_tokens
			(organization_name, apple_id, renew_at, token, enrollment_url_token, macos_default_team_id, ios_default_team_id, ipados_default_team_id, byod_default_team_id)
			VALUES ('Org', 'abm@example.com', '2030-01-01', 'x', REPEAT('e', 33), 2, 3, 1, NULL)`)
		execNoErr(t, db, `INSERT INTO vpp_tokens (id, organization_name, location, renew_at, token) VALUES (1, 'Org', 'Loc', '2030-01-01', 'x')`)
		execNoErr(t, db, `INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (1, 2, 'none'), (1, 3, 'none')`)
		// The Windows automatic enrollment default fleet is stored by ID.
		execNoErr(t, db, `UPDATE app_config_json
			SET json_value = JSON_SET(json_value, '$.mdm.windows_automatic_enrollment', JSON_OBJECT('default_fleet_id', 2))`)

		type row map[string]any
		snapshot := func(query string) []row {
			rows, err := db.Queryx(query)
			require.NoError(t, err)
			defer rows.Close()
			var out []row
			for rows.Next() {
				r := make(row)
				require.NoError(t, rows.MapScan(r))
				out = append(out, r)
			}
			require.NoError(t, rows.Err())
			return out
		}
		unchanged := []string{
			`SELECT id, description, config, filename, created_at FROM teams ORDER BY id`,
			`SELECT hostname, team_id FROM hosts ORDER BY id`,
			`SELECT secret, team_id FROM enroll_secrets ORDER BY secret`,
			`SELECT user_id, team_id, role FROM user_teams ORDER BY team_id`,
			`SELECT organization_name, macos_default_team_id, ios_default_team_id, ipados_default_team_id, byod_default_team_id, updated_at FROM abm_tokens`,
			`SELECT vpp_token_id, team_id, null_team_type FROM vpp_token_teams ORDER BY team_id`,
			`SELECT json_value, updated_at FROM app_config_json`,
			`SELECT COUNT(*) FROM activity_past`,
		}
		before := make([][]row, 0, len(unchanged))
		for _, q := range unchanged {
			before = append(before, snapshot(q))
		}

		applyNext(t, db)

		require.Equal(t, []string{"Workstations", "work-stations 2", "🚀 1"}, fleetNames(t, db))
		for i, q := range unchanged {
			assert.Equal(t, before[i], snapshot(q), q)
		}
	})

	// The stored app config keys are being renamed in Fleet 5 by migrations whose order relative to this one isn't
	// fixed, so each case uses one spelling for every key.
	spellings := []struct {
		name           string
		abmList        string
		abmFleetSuffix string
		vppFleets      string
	}{
		{name: "Fleet 4 keys", abmList: "apple_business_manager", abmFleetSuffix: "_team", vppFleets: "teams"},
		{name: "Fleet 5 list key", abmList: "apple_business", abmFleetSuffix: "_team", vppFleets: "teams"},
		{name: "Fleet 5 keys", abmList: "apple_business", abmFleetSuffix: "_fleet", vppFleets: "fleets"},
	}
	for _, sp := range spellings {
		t.Run("app config "+sp.name, func(t *testing.T) {
			db := applyUpToPrev(t)

			// Fleet names are stored composed. The copy in app config is what the admin typed, so it can be decomposed.
			decomposed := norm.NFD.String("Café")
			require.NotEqual(t, "Café", decomposed)
			ids := insertFleets(t, db, []string{"Workstations", "work-stations", "🚀", "Servers", "Kiosks", "Cafe", "Café"})
			workstations, workStations, rocket, servers, kiosks, cafeAccent := ids[0], ids[1], ids[2], ids[3], ids[4], ids[6]

			insertABM := func(org string, macOS, iOS, iPadOS, byod any) {
				execNoErr(t, db, `INSERT INTO abm_tokens
					(organization_name, apple_id, renew_at, token, enrollment_url_token, macos_default_team_id, ios_default_team_id, ipados_default_team_id, byod_default_team_id)
					VALUES (?, 'abm@example.com', '2030-01-01', 'x', RPAD(?, 33, 'e'), ?, ?, ?, ?)`, org, org, macOS, iOS, iPadOS, byod)
			}
			insertABM("Renamed everywhere", workStations, workStations, rocket, cafeAccent)
			insertABM("Mixed", workstations, workStations, nil, servers)
			insertABM("Stale copies", kiosks, nil, nil, nil)
			insertABM("Not in app config", workStations, nil, nil, nil)

			insertVPP := func(location string, fleetIDs ...int64) {
				tokenID := execNoErrLastID(t, db,
					`INSERT INTO vpp_tokens (organization_name, location, renew_at, token) VALUES ('Org', ?, '2030-01-01', 'x')`, location)
				for _, id := range fleetIDs {
					execNoErr(t, db, `INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, ?, 'none')`, tokenID, id)
				}
				if len(fleetIDs) == 0 {
					execNoErr(t, db, `INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, NULL, 'allteams')`, tokenID)
				}
			}
			insertVPP("Renamed", workStations, rocket, servers, cafeAccent)
			insertVPP("All fleets")
			insertVPP("Stale copies", kiosks)
			insertVPP("Not in app config", workStations)

			abmEntry := func(org string, isDefault bool, macOS, iOS, iPadOS, byod string) map[string]any {
				return map[string]any{
					"organization_name":          org,
					"default":                    isDefault,
					"macos" + sp.abmFleetSuffix:  macOS,
					"ios" + sp.abmFleetSuffix:    iOS,
					"ipados" + sp.abmFleetSuffix: iPadOS,
					"byod" + sp.abmFleetSuffix:   byod,
				}
			}
			vppEntry := func(location string, fleets ...string) map[string]any {
				return map[string]any{"location": location, sp.vppFleets: fleets}
			}
			config := func(mdm map[string]any) string {
				var merged string
				require.NoError(t, db.Get(&merged, `SELECT JSON_MERGE_PATCH(json_value, ?) FROM app_config_json WHERE id = 1`,
					mustJSON(t, map[string]any{"mdm": mdm})))
				return merged
			}

			before := config(map[string]any{
				"apple_bm_default_team": "work-stations",
				sp.abmList: []any{
					abmEntry("Renamed everywhere", true, "work-stations", "work-stations", "🚀", decomposed),
					// Like a rename through the API, every copy on a token assigned to the fleet is updated, even for
					// a platform the fleet isn't the default for (iPadOS here).
					abmEntry("Mixed", false, "Workstations", "work-stations", "work-stations", "Servers"),
					// These name renamed fleets, but the token isn't assigned to them.
					abmEntry("Stale copies", false, "Kiosks", "work-stations", "🚀", decomposed),
					abmEntry("No token", false, "work-stations", "", "", ""),
				},
				"volume_purchasing_program": []any{
					vppEntry("Renamed", "work-stations", "🚀", "Servers", decomposed),
					vppEntry("All fleets", "All teams"),
					vppEntry("Stale copies", "Kiosks", "work-stations", "🚀"),
					vppEntry("No token", "work-stations"),
				},
			})
			execNoErr(t, db, `UPDATE app_config_json SET json_value = ? WHERE id = 1`, before)

			applyNext(t, db)

			require.Equal(t, []string{"Workstations", "work-stations 2", "🚀 1", "Servers", "Kiosks", "Cafe", "Café 2"}, fleetNames(t, db))

			var got string
			require.NoError(t, db.Get(&got, `SELECT json_value FROM app_config_json WHERE id = 1`))
			execNoErr(t, db, `UPDATE app_config_json SET json_value = ? WHERE id = 1`, before)
			want := config(map[string]any{
				"apple_bm_default_team": "work-stations 2",
				sp.abmList: []any{
					abmEntry("Renamed everywhere", true, "work-stations 2", "work-stations 2", "🚀 1", "Café 2"),
					abmEntry("Mixed", false, "Workstations", "work-stations 2", "work-stations 2", "Servers"),
					abmEntry("Stale copies", false, "Kiosks", "work-stations", "🚀", decomposed),
					abmEntry("No token", false, "work-stations", "", "", ""),
				},
				"volume_purchasing_program": []any{
					vppEntry("Renamed", "work-stations 2", "🚀 1", "Servers", "Café 2"),
					vppEntry("All fleets", "All teams"),
					vppEntry("Stale copies", "Kiosks", "work-stations", "🚀"),
					vppEntry("No token", "work-stations"),
				},
			})
			assert.JSONEq(t, want, got)
		})
	}

	// A key rename that is half applied leaves lists and name keys in both spellings.
	t.Run("app config both spellings", func(t *testing.T) {
		db := applyUpToPrev(t)

		ids := insertFleets(t, db, []string{"Workstations", "workstations"})
		execNoErr(t, db, `INSERT INTO abm_tokens
			(organization_name, apple_id, renew_at, token, enrollment_url_token, macos_default_team_id, ios_default_team_id)
			VALUES ('Org', 'abm@example.com', '2030-01-01', 'x', REPEAT('e', 33), ?, ?)`, ids[1], ids[0])
		tokenID := execNoErrLastID(t, db,
			`INSERT INTO vpp_tokens (organization_name, location, renew_at, token) VALUES ('Org', 'Loc', '2030-01-01', 'x')`)
		execNoErr(t, db, `INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, ?, 'none')`, tokenID, ids[1])

		config := func(renamed string) string {
			var merged string
			require.NoError(t, db.Get(&merged, `SELECT JSON_MERGE_PATCH(json_value, ?) FROM app_config_json WHERE id = 1`,
				mustJSON(t, map[string]any{"mdm": map[string]any{
					"apple_business_manager": []any{
						map[string]any{"organization_name": "Org", "macos_team": renamed, "ios_team": "Workstations", "ipados_team": "", "byod_fleet": renamed},
					},
					"apple_business": []any{
						map[string]any{"organization_name": "Org", "macos_fleet": renamed, "ios_fleet": "Workstations", "ipados_team": renamed},
						"not an object",
					},
					"volume_purchasing_program": []any{
						map[string]any{"location": "Loc", "teams": []string{renamed}, "fleets": []string{"Workstations", renamed}},
						map[string]any{"location": "Loc", "teams": nil},
						map[string]any{"location": "Loc", "fleets": "not a list"},
						map[string]any{"location": "Loc", "fleets": []any{1, nil, renamed}},
					},
				}})))
			return merged
		}
		before, want := config("workstations"), config("workstations 2")
		execNoErr(t, db, `UPDATE app_config_json SET json_value = ? WHERE id = 1`, before)

		applyNext(t, db)

		require.Equal(t, []string{"Workstations", "workstations 2"}, fleetNames(t, db))
		var got string
		require.NoError(t, db.Get(&got, `SELECT json_value FROM app_config_json WHERE id = 1`))
		assert.JSONEq(t, want, got)
	})

	t.Run("app config null lists or no row", func(t *testing.T) {
		for _, deleteRow := range []bool{false, true} {
			db := applyUpToPrev(t)
			insertFleets(t, db, []string{"Workstations", "workstations"})
			if deleteRow {
				execNoErr(t, db, `DELETE FROM app_config_json`)
			}
			var before []string
			require.NoError(t, db.Select(&before, `SELECT CONCAT(json_value, updated_at) FROM app_config_json`))

			applyNext(t, db)

			require.Equal(t, []string{"Workstations", "workstations 2"}, fleetNames(t, db))
			var after []string
			require.NoError(t, db.Select(&after, `SELECT CONCAT(json_value, updated_at) FROM app_config_json`))
			assert.Equal(t, before, after)
		}
	})
}

func TestFleetNameKey20261112204537(t *testing.T) {
	for name, want := range map[string]string{
		"Workstations":    "workstations",
		"Work-stations":   "workstations",
		"WORK_STATIONS":   "workstations",
		"Workstations 💻":  "workstations",
		"Workstations 2":  "workstations2",
		"straße":          "strasse",
		"ＡＢＣ":             "abc",
		"Reneé":           "renee",
		"開発・チーム":          "開発チーム",
		"🚀":               "",
		"!!!":             "",
		"「」":              "",
		" \t":             "",
		"Ⅻ":               "xii",
		"½":               "12",
		"Ångström":        "angstrom",
		"ǅ":               "dz",
		"ﬁre":             "fire",
		"İstanbul":        "istanbul",
		"ΣΊΣΥΦΟΣ σίσυφος": "σισυφοσσισυφοσ",
	} {
		assert.Equal(t, want, fleetNameKey20261112204537(name), name)
	}
}
