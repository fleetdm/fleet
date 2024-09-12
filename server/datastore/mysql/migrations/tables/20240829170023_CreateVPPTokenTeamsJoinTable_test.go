package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240829170023(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec("INSERT INTO teams (name) VALUES (?)", "team1")
	require.NoError(t, err)

	_, err = db.Exec(`
	INSERT INTO vpp_tokens (
		organization_name,
		location,
		renew_at,
		token,
		team_id,
		null_team_type
	) VALUES
		(?, ?, ?, ?, ?, ?),
		(?, ?, ?, ?, ?, ?),
		(?, ?, ?, ?, ?, ?)
	`,
		"org1", "loc1", "2030-01-01 10:10:10", "blob1", 1, "none",
		"org2", "loc2", "2030-02-01 10:10:10", "blob2", nil, "noteam",
		"org3", "loc3", "2030-03-01 10:10:10", "blob3", nil, "allteams",
	)
	require.NoError(t, err)

	var count []int
	err = db.Select(&count, "SELECT COUNT(*) FROM vpp_tokens")
	require.NoError(t, err)
	require.Equal(t, 3, count[0])

	// Apply current migration.
	applyNext(t, db)

	var sel []selresult

	err = db.Select(&sel, `
	SELECT
		v.organization_name,
		v.token,
		j.vpp_token_id,
		j.team_id,
		j.null_team_type
	FROM
		vpp_tokens v
	LEFT OUTER JOIN
		vpp_token_teams j
	ON
		v.id = j.vpp_token_id
	`)
	require.NoError(t, err)
	require.Len(t, sel, 3)

	expected := []selresult{
		{
			// Make assumptions about autoincrement IDs
			TokenID:  1,
			Org:      "org1",
			Token:    "blob1",
			TeamID:   ptr.Int(1),
			NullTeam: "none",
		},
		{
			TokenID:  2,
			Org:      "org2",
			Token:    "blob2",
			TeamID:   nil,
			NullTeam: "noteam",
		},
		{
			TokenID:  3,
			Org:      "org3",
			Token:    "blob3",
			TeamID:   nil,
			NullTeam: "allteams",
		},
	}

	for _, exp := range expected {
		actual := find(t, sel, exp.TokenID)
		assert.Equal(t, exp.Org, actual.Org)
		assert.Equal(t, exp.Token, actual.Token)
		if exp.TeamID == nil {
			assert.Nil(t, actual.TeamID)
		} else {
			assert.Equal(t, *exp.TeamID, *actual.TeamID)
		}
		assert.Equal(t, exp.NullTeam, actual.NullTeam)
	}
}

type selresult struct {
	TokenID  int    `db:"vpp_token_id"`
	Org      string `db:"organization_name"`
	Token    string `db:"token"`
	TeamID   *int   `db:"team_id"`
	NullTeam string `db:"null_team_type"`
}

func find(t *testing.T, arr []selresult, tokenID int) selresult {
	for _, thing := range arr {
		if thing.TokenID == tokenID {
			return thing
		}
	}

	t.Errorf("failed to find result with tokenID %d", tokenID)
	return selresult{}
}
