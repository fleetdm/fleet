package mysql

import (
	"database/sql"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestChallenges(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NewAndConsume", testChallengeNewAndConsume},
		{"ConsumeMissing", testChallengeConsumeMissing},
		{"ConsumeWithinTTL", testChallengeConsumeWithinTTL},
		{"ConsumeExpired", testChallengeConsumeExpired},
		{"CleanupRespectsTTL", testChallengeCleanupRespectsTTL},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testChallengeNewAndConsume(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	challenge, err := ds.NewChallenge(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, challenge)

	err = ds.ConsumeChallenge(ctx, challenge)
	require.NoError(t, err)

	// Second consume must fail — challenge is one-time.
	err = ds.ConsumeChallenge(ctx, challenge)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func testChallengeConsumeMissing(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	err := ds.ConsumeChallenge(ctx, "")
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	err = ds.ConsumeChallenge(ctx, "never-issued")
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// testChallengeConsumeWithinTTL backdates a challenge to just within OneTimeChallengeTTL and
// confirms it's still consumable. Regression coverage for issue #44111: devices may take
// hours/days to process the InstallProfile push before sending the SCEP request.
func testChallengeConsumeWithinTTL(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	challenge, err := ds.NewChallenge(ctx)
	require.NoError(t, err)

	// Backdate to 1 minute inside the TTL.
	backdated := time.Now().Add(-fleet.OneTimeChallengeTTL).Add(1 * time.Minute)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE challenges SET created_at = ? WHERE challenge = ?`, backdated, challenge)
		return err
	})

	err = ds.ConsumeChallenge(ctx, challenge)
	require.NoError(t, err)
}

// testChallengeConsumeExpired backdates a challenge past OneTimeChallengeTTL and confirms it's
// rejected as expired (returns sql.ErrNoRows wrapped with "challenge expired").
func testChallengeConsumeExpired(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	challenge, err := ds.NewChallenge(ctx)
	require.NoError(t, err)

	// Backdate past the TTL.
	expired := time.Now().Add(-fleet.OneTimeChallengeTTL).Add(-1 * time.Minute)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE challenges SET created_at = ? WHERE challenge = ?`, expired, challenge)
		return err
	})

	err = ds.ConsumeChallenge(ctx, challenge)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// testChallengeCleanupRespectsTTL verifies CleanupExpiredChallenges deletes only challenges
// older than OneTimeChallengeTTL and leaves still-valid challenges in place.
func testChallengeCleanupRespectsTTL(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	freshChallenge, err := ds.NewChallenge(ctx)
	require.NoError(t, err)
	expiredChallenge, err := ds.NewChallenge(ctx)
	require.NoError(t, err)

	expired := time.Now().Add(-fleet.OneTimeChallengeTTL).Add(-1 * time.Minute)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE challenges SET created_at = ? WHERE challenge = ?`, expired, expiredChallenge)
		return err
	})

	deleted, err := ds.CleanupExpiredChallenges(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)

	// Fresh challenge survives and is still consumable.
	err = ds.ConsumeChallenge(ctx, freshChallenge)
	require.NoError(t, err)

	// Expired challenge is gone.
	err = ds.ConsumeChallenge(ctx, expiredChallenge)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
