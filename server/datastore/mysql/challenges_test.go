package mysql

import (
	"database/sql"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestChallenges(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ConsumeChallengeScope", testConsumeChallengeScope},
		{"ConsumeChallengeWithoutScope", testConsumeChallengeWithoutScope},
		{"ConsumeChallengeExpired", testConsumeChallengeExpired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testConsumeChallengeScope(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	scope := fleet.ChallengeScope{HostUUID: "host-a", ProfileUUID: "a-profile-1", CAName: "ca-1"}

	challenge, err := ds.NewChallenge(ctx, scope)
	require.NoError(t, err)

	for name, other := range map[string]fleet.ChallengeScope{
		"another host":    {HostUUID: "host-b", ProfileUUID: "a-profile-1", CAName: "ca-1"},
		"another profile": {HostUUID: "host-a", ProfileUUID: "a-profile-2", CAName: "ca-1"},
		"another CA":      {HostUUID: "host-a", ProfileUUID: "a-profile-1", CAName: "ca-2"},
		"no CA":           {HostUUID: "host-a", ProfileUUID: "a-profile-1"},
		"empty scope":     {},
	} {
		require.ErrorIs(t, ds.ConsumeChallenge(ctx, challenge, other), sql.ErrNoRows, name)
	}

	// The rejected attempts did not use the challenge up.
	require.NoError(t, ds.ConsumeChallenge(ctx, challenge, scope))
	require.ErrorIs(t, ds.ConsumeChallenge(ctx, challenge, scope), sql.ErrNoRows)

	require.ErrorIs(t, ds.ConsumeChallenge(ctx, "", scope), sql.ErrNoRows)
}

func testConsumeChallengeWithoutScope(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// A row written before challenges carried a scope.
	const challenge = "challenge-without-scope-00000000"
	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO challenges (challenge) VALUES (?)`, challenge)
	require.NoError(t, err)

	require.ErrorIs(t, ds.ConsumeChallenge(ctx, challenge, fleet.ChallengeScope{}), sql.ErrNoRows)
	require.ErrorIs(t, ds.ConsumeChallenge(ctx, challenge, fleet.ChallengeScope{HostUUID: "host-a", ProfileUUID: "a-profile-1"}),
		sql.ErrNoRows)
}

func testConsumeChallengeExpired(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	scope := fleet.ChallengeScope{HostUUID: "host-a", ProfileUUID: "a-profile-1", CAName: "ca-1"}

	challenge, err := ds.NewChallenge(ctx, scope)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE challenges SET created_at = ? WHERE challenge = ?`,
		time.Now().Add(-2*fleet.OneTimeChallengeTTL), challenge)
	require.NoError(t, err)

	require.ErrorIs(t, ds.ConsumeChallenge(ctx, challenge, scope), sql.ErrNoRows)
}
