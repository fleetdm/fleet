package mysql

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAndroidZeroTouch(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetNotFound", testZeroTouchGetNotFound},
		{"CreateAndGet", testZeroTouchCreateAndGet},
		{"CreateDuplicateTeam", testZeroTouchCreateDuplicateTeam},
		{"ConcurrentCreate", testZeroTouchConcurrentCreate},
		{"DeleteAll", testZeroTouchDeleteAll},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer testing_utils.TruncateTables(t, ds.primary, ds.logger, nil)
			c.fn(t, ds)
		})
	}
}

func testZeroTouchGetNotFound(t *testing.T, ds *Datastore) {
	_, err := ds.GetZeroTouchEnrollmentToken(testCtx(), nil)
	assert.True(t, fleet.IsNotFound(err))

	teamID := uint(999)
	_, err = ds.GetZeroTouchEnrollmentToken(testCtx(), &teamID)
	assert.True(t, fleet.IsNotFound(err))
}

func testZeroTouchCreateAndGet(t *testing.T, ds *Datastore) {
	expiresAt := time.Now().Add(100 * 365 * 24 * time.Hour).Truncate(time.Microsecond)

	// Create a token for unassigned (nil team)
	token := &android.ZeroTouchToken{
		TokenName:  "enterprises/LC00test/enrollmentTokens/abc123",
		TokenValue: "tokenvalue123",
		ExpiresAt:  expiresAt,
	}
	created, err := ds.CreateZeroTouchEnrollmentToken(testCtx(), token)
	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Nil(t, created.TeamID)

	// Get it back
	got, err := ds.GetZeroTouchEnrollmentToken(testCtx(), nil)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Nil(t, got.TeamID)
	assert.Equal(t, "enterprises/LC00test/enrollmentTokens/abc123", got.TokenName)
	assert.Equal(t, "tokenvalue123", got.TokenValue)
	assert.WithinDuration(t, expiresAt, got.ExpiresAt, time.Second)

	// Create a team, then a token for it
	team, err := ds.NewTeam(testCtx(), &fleet.Team{Name: "zt-test-team"})
	require.NoError(t, err)
	teamID := team.ID
	teamToken := &android.ZeroTouchToken{
		TeamID:     &teamID,
		TokenName:  "enterprises/LC00test/enrollmentTokens/def456",
		TokenValue: "tokenvalue456",
		ExpiresAt:  expiresAt,
	}
	createdTeam, err := ds.CreateZeroTouchEnrollmentToken(testCtx(), teamToken)
	require.NoError(t, err)
	assert.NotZero(t, createdTeam.ID)

	// Get the team token
	gotTeam, err := ds.GetZeroTouchEnrollmentToken(testCtx(), &teamID)
	require.NoError(t, err)
	assert.Equal(t, createdTeam.ID, gotTeam.ID)
	assert.Equal(t, &teamID, gotTeam.TeamID)
	assert.Equal(t, "tokenvalue456", gotTeam.TokenValue)

	// The unassigned token is still there
	gotUnassigned, err := ds.GetZeroTouchEnrollmentToken(testCtx(), nil)
	require.NoError(t, err)
	assert.Equal(t, created.ID, gotUnassigned.ID)
}

func testZeroTouchCreateDuplicateTeam(t *testing.T, ds *Datastore) {
	expiresAt := time.Now().Add(100 * 365 * 24 * time.Hour)

	team, err := ds.NewTeam(testCtx(), &fleet.Team{Name: "zt-dup-team"})
	require.NoError(t, err)
	teamID := team.ID

	first, err := ds.CreateZeroTouchEnrollmentToken(testCtx(), &android.ZeroTouchToken{
		TeamID:     &teamID,
		TokenName:  "enterprises/LC00test/enrollmentTokens/first",
		TokenValue: "first",
		ExpiresAt:  expiresAt,
	})
	require.NoError(t, err)

	// A duplicate insert returns the existing token, not an error.
	got, err := ds.CreateZeroTouchEnrollmentToken(testCtx(), &android.ZeroTouchToken{
		TeamID:     &teamID,
		TokenName:  "enterprises/LC00test/enrollmentTokens/second",
		TokenValue: "second",
		ExpiresAt:  expiresAt,
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, got.ID)
	assert.Equal(t, "first", got.TokenValue)
}

func testZeroTouchConcurrentCreate(t *testing.T, ds *Datastore) {
	expiresAt := time.Now().Add(100 * 365 * 24 * time.Hour)

	const n = 10
	type result struct {
		token *android.ZeroTouchToken
		err   error
	}
	results := make([]result, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			tok, err := ds.CreateZeroTouchEnrollmentToken(testCtx(), &android.ZeroTouchToken{
				TokenName:  fmt.Sprintf("enterprises/LC00test/enrollmentTokens/tok%d", i),
				TokenValue: fmt.Sprintf("value%d", i),
				ExpiresAt:  expiresAt,
			})
			results[i] = result{tok, err}
		}(i)
	}
	wg.Wait()

	// All calls must succeed and return the same winning token.
	var winnerID uint
	for _, r := range results {
		require.NoError(t, r.err)
		require.NotNil(t, r.token)
		if winnerID == 0 {
			winnerID = r.token.ID
		}
		assert.Equal(t, winnerID, r.token.ID, "all callers should get the same token")
	}
}

func testZeroTouchDeleteAll(t *testing.T, ds *Datastore) {
	expiresAt := time.Now().Add(100 * 365 * 24 * time.Hour)

	// Create unassigned token
	_, err := ds.CreateZeroTouchEnrollmentToken(testCtx(), &android.ZeroTouchToken{
		TokenName:  "enterprises/LC00test/enrollmentTokens/a",
		TokenValue: "a",
		ExpiresAt:  expiresAt,
	})
	require.NoError(t, err)

	// Create team token
	team, err := ds.NewTeam(testCtx(), &fleet.Team{Name: "zt-del-team"})
	require.NoError(t, err)
	teamID := team.ID
	_, err = ds.CreateZeroTouchEnrollmentToken(testCtx(), &android.ZeroTouchToken{
		TeamID:     &teamID,
		TokenName:  "enterprises/LC00test/enrollmentTokens/b",
		TokenValue: "b",
		ExpiresAt:  expiresAt,
	})
	require.NoError(t, err)

	// Delete all
	err = ds.DeleteZeroTouchEnrollmentTokens(testCtx())
	require.NoError(t, err)

	// Both should be gone
	_, err = ds.GetZeroTouchEnrollmentToken(testCtx(), nil)
	assert.True(t, fleet.IsNotFound(err))
	_, err = ds.GetZeroTouchEnrollmentToken(testCtx(), &teamID)
	assert.True(t, fleet.IsNotFound(err))

	// Deleting again should not error
	err = ds.DeleteZeroTouchEnrollmentTokens(testCtx())
	require.NoError(t, err)
}
