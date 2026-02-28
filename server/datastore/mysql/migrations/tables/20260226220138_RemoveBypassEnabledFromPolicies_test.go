package tables

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260226220138(t *testing.T) {
	// setOktaConfig sets or clears Okta conditional access config in app_config_json.
	setOktaConfig := func(t *testing.T, db *sqlx.DB, configured bool) {
		t.Helper()
		if !configured {
			return
		}
		oktaConfig := map[string]any{
			"okta_idp_id":                          "test-idp-id",
			"okta_assertion_consumer_service_url":  "https://example.com/acs",
			"okta_audience_uri":                    "https://example.com/audience",
			"okta_certificate":                     "test-certificate",
		}
		oktaJSON, err := json.Marshal(oktaConfig)
		require.NoError(t, err)
		_, err = db.Exec(
			`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.conditional_access', CAST(? AS JSON)) WHERE id = 1`,
			string(oktaJSON),
		)
		require.NoError(t, err)
	}

	// insertTeam inserts a team with a config that sets conditional_access_enabled.
	insertTeam := func(t *testing.T, db *sqlx.DB, name string, caEnabled bool) uint {
		t.Helper()
		config := map[string]any{
			"integrations": map[string]any{
				"conditional_access_enabled": caEnabled,
			},
		}
		configJSON, err := json.Marshal(config)
		require.NoError(t, err)
		id := execNoErrLastID(t, db, `INSERT INTO teams (name, config) VALUES (?, ?)`, name, string(configJSON))
		return uint(id) //nolint:gosec // dismiss G115
	}

	// insertPolicy inserts a policy with the given bypass/critical settings.
	insertPolicy := func(t *testing.T, db *sqlx.DB, name string, teamID *uint, bypassEnabled bool, critical bool) uint {
		t.Helper()
		id := execNoErrLastID(t, db,
			`INSERT INTO policies (name, description, query, team_id, critical, conditional_access_bypass_enabled, checksum)
			 VALUES (?, '', 'SELECT 1', ?, ?, ?, UNHEX(MD5(?)))`,
			name, teamID, critical, bypassEnabled, name,
		)
		return uint(id) //nolint:gosec // dismiss G115
	}

	// columnExists reports whether the given column exists on a table.
	columnExists := func(t *testing.T, db *sqlx.DB, table, column string) bool {
		t.Helper()
		var count int
		err := db.Get(&count, `
			SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
			table, column,
		)
		require.NoError(t, err)
		return count > 0
	}

	// policyIsCritical returns the critical field value for a policy by ID.
	policyIsCritical := func(t *testing.T, db *sqlx.DB, id uint) bool {
		t.Helper()
		var critical bool
		require.NoError(t, db.Get(&critical, `SELECT critical FROM policies WHERE id = ?`, id))
		return critical
	}

	t.Run("okta not configured", func(t *testing.T) {
		db := applyUpToPrev(t)

		// Okta NOT configured in app_config_json.
		setOktaConfig(t, db, false)

		teamID := insertTeam(t, db, "team-a", true)
		// Policy with bypass enabled — should NOT become critical since Okta is not configured.
		policyID := insertPolicy(t, db, "bypass-policy", &teamID, true, false)

		applyNext(t, db)

		assert.False(t, policyIsCritical(t, db, policyID))
		assert.False(t, columnExists(t, db, "policies", "conditional_access_bypass_enabled"))
	})

	t.Run("no ca enabled teams", func(t *testing.T) {
		db := applyUpToPrev(t)

		setOktaConfig(t, db, true)

		// Two teams, both with conditional access disabled.
		teamIDA := insertTeam(t, db, "team-a", false)
		teamIDB := insertTeam(t, db, "team-b", false)
		policyA := insertPolicy(t, db, "policy-a", &teamIDA, true, false)
		policyB := insertPolicy(t, db, "policy-b", &teamIDB, true, false)

		applyNext(t, db)

		assert.False(t, policyIsCritical(t, db, policyA))
		assert.False(t, policyIsCritical(t, db, policyB))
		assert.False(t, columnExists(t, db, "policies", "conditional_access_bypass_enabled"))
	})

	t.Run("one ca enabled team", func(t *testing.T) {
		db := applyUpToPrev(t)

		setOktaConfig(t, db, true)

		// Team A has conditional access enabled, team B does not.
		teamIDA := insertTeam(t, db, "team-a", true)
		teamIDB := insertTeam(t, db, "team-b", false)

		// Team A: bypass=true → should become critical
		policyABypass := insertPolicy(t, db, "team-a-bypass", &teamIDA, true, false)
		// Team A: bypass=false → should NOT become critical
		policyANoBypass := insertPolicy(t, db, "team-a-no-bypass", &teamIDA, false, false)
		// Team A: already critical, bypass=true → should remain critical
		policyAAlreadyCritical := insertPolicy(t, db, "team-a-already-critical", &teamIDA, true, true)
		// Team B: bypass=true → should NOT become critical (CA disabled for this team)
		policyBBypass := insertPolicy(t, db, "team-b-bypass", &teamIDB, true, false)
		// No-team policy: bypass=true → should NOT become critical (NULL not IN list)
		noTeamPolicy := insertPolicy(t, db, "no-team-bypass", nil, true, false)

		applyNext(t, db)

		assert.True(t, policyIsCritical(t, db, policyABypass))
		assert.False(t, policyIsCritical(t, db, policyANoBypass))
		assert.True(t, policyIsCritical(t, db, policyAAlreadyCritical))
		assert.False(t, policyIsCritical(t, db, policyBBypass))
		assert.False(t, policyIsCritical(t, db, noTeamPolicy))
		assert.False(t, columnExists(t, db, "policies", "conditional_access_bypass_enabled"))
	})

	t.Run("multiple ca enabled teams", func(t *testing.T) {
		db := applyUpToPrev(t)

		setOktaConfig(t, db, true)

		// Teams A and B have conditional access enabled, team C does not.
		teamIDA := insertTeam(t, db, "team-a", true)
		teamIDB := insertTeam(t, db, "team-b", true)
		teamIDC := insertTeam(t, db, "team-c", false)

		policyA := insertPolicy(t, db, "policy-a", &teamIDA, true, false)
		policyB := insertPolicy(t, db, "policy-b", &teamIDB, true, false)
		policyC := insertPolicy(t, db, "policy-c", &teamIDC, true, false)

		applyNext(t, db)

		assert.True(t, policyIsCritical(t, db, policyA))
		assert.True(t, policyIsCritical(t, db, policyB))
		assert.False(t, policyIsCritical(t, db, policyC))
		assert.False(t, columnExists(t, db, "policies", "conditional_access_bypass_enabled"))
	})

	t.Run("no teams exist", func(t *testing.T) {
		db := applyUpToPrev(t)

		setOktaConfig(t, db, true)
		// No teams inserted — migration should complete without errors.

		applyNext(t, db)

		assert.False(t, columnExists(t, db, "policies", "conditional_access_bypass_enabled"))
	})
}
