package tables

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261116102943(t *testing.T) {
	db := applyUpToPrev(t)

	webhookConfig := func(policyIDs string) string {
		return fmt.Sprintf(`{"webhook_settings": {"failing_policies_webhook": {"enable_failing_policies_webhook": true, "policy_ids": %s}}}`, policyIDs)
	}
	newTeam := func(name string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO teams (name, config) VALUES (?, NULL)`, name)
	}
	newPolicy := func(name string, teamID any) int64 {
		return execNoErrLastID(t, db, `INSERT INTO policies (name, query, description, team_id, checksum) VALUES (?, 'SELECT 1', '', ?, UNHEX(MD5(?)))`,
			name, teamID, name)
	}

	teamA := newTeam("A")
	teamB := newTeam("B")
	teamEmpty := newTeam("empty")
	teamNull := newTeam("null list")
	teamNoConfig := newTeam("no config")

	globalListed := newPolicy("global listed", nil)
	globalUnlisted := newPolicy("global unlisted", nil)
	globalInTeamList := newPolicy("global in team A list", nil)
	teamAListed := newPolicy("team A listed", teamA)
	teamAUnlisted := newPolicy("team A unlisted", teamA)
	teamAInGlobalList := newPolicy("team A in global list", teamA)
	teamAInTeamBList := newPolicy("team A in team B list", teamA)
	teamBListed := newPolicy("team B listed", teamB)
	teamEmptyPolicy := newPolicy("team empty", teamEmpty)
	teamNullPolicy := newPolicy("team null", teamNull)
	teamNoConfigPolicy := newPolicy("team no config", teamNoConfig)
	noTeamListed := newPolicy("no team listed", 0)
	noTeamInTeamList := newPolicy("no team in team A list", 0)
	const deletedID = 999999

	execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_MERGE_PATCH(json_value, CAST(? AS JSON)) WHERE id = 1`,
		webhookConfig(fmt.Sprintf("[%d, %d, %d]", globalListed, teamAInGlobalList, deletedID)))
	execNoErr(t, db, `UPDATE teams SET config = ? WHERE id = ?`,
		webhookConfig(fmt.Sprintf("[%d, %d, %d]", teamAListed, globalInTeamList, noTeamInTeamList)), teamA)
	execNoErr(t, db, `UPDATE teams SET config = ? WHERE id = ?`, webhookConfig(fmt.Sprintf("[%d, %d]", teamBListed, teamAInTeamBList)), teamB)
	execNoErr(t, db, `UPDATE teams SET config = ? WHERE id = ?`, webhookConfig("[]"), teamEmpty)
	execNoErr(t, db, `UPDATE teams SET config = ? WHERE id = ?`, webhookConfig("null"), teamNull)
	execNoErr(t, db, `INSERT INTO default_team_config_json (id, json_value) VALUES (1, CAST(? AS JSON))
		ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`, webhookConfig(fmt.Sprintf("[%d]", noTeamListed)))

	applyNext(t, db)

	var enabled []int64
	require.NoError(t, sqlx.Select(db, &enabled, `SELECT id FROM policies WHERE webhooks_and_tickets_enabled = 1 ORDER BY id`))
	require.Equal(t, []int64{globalListed, teamAListed, teamBListed, noTeamListed}, enabled)

	var disabled []int64
	require.NoError(t, sqlx.Select(db, &disabled, `SELECT id FROM policies WHERE webhooks_and_tickets_enabled = 0 ORDER BY id`))
	require.Equal(t, []int64{
		globalUnlisted, globalInTeamList, teamAUnlisted, teamAInGlobalList, teamAInTeamBList,
		teamEmptyPolicy, teamNullPolicy, teamNoConfigPolicy, noTeamInTeamList,
	}, disabled)

	// New policies default to off.
	newID := newPolicy("created after", teamA)
	var flag bool
	require.NoError(t, sqlx.Get(db, &flag, `SELECT webhooks_and_tickets_enabled FROM policies WHERE id = ?`, newID))
	require.False(t, flag)
}
