package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// NewChallenge generates a random, base64-encoded challenge and inserts it into the challenges
// table. It returns the generated challenge or an error if the insertion fails.
func (ds *Datastore) NewChallenge(ctx context.Context) (string, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	challenge := base64.URLEncoding.EncodeToString(key)
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO challenges (challenge) VALUES (?)`, challenge)
	if err != nil {
		return "", err
	}
	return challenge, nil
}

// ConsumeChallenge checks if a valid challenge exists in the challenges table
// and deletes it if it does. The error will include sql.ErrNoRows if the challenge
// is not found or is expired.
func (ds *Datastore) ConsumeChallenge(ctx context.Context, challenge string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return consumeChallengeTx(ctx, tx, challenge)
	})
}

func consumeChallengeTx(ctx context.Context, tx sqlx.ExtContext, challenge string) error {
	if challenge == "" {
		return ctxerr.Wrap(ctx, sql.ErrNoRows, "empty challenge") // no challenge provided, treat as invalid
	}

	// check if matching challenge exists and retrieve its creation time
	var createdAt time.Time
	if err := sqlx.GetContext(ctx, tx, &createdAt, `SELECT created_at FROM challenges WHERE challenge = ?`, challenge); err != nil {
		return ctxerr.Wrap(ctx, err, "check challenge existence")
	}

	// delete challenge regardless of validity
	result, err := tx.ExecContext(ctx, `DELETE FROM challenges WHERE challenge = ?`, challenge)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete challenge")
	}
	rowCt, _ := result.RowsAffected()
	if rowCt < 1 {
		// unlikely to happen since just checked existence and we're in a transaction, but error
		// for debugging purposes just in case
		return ctxerr.Wrap(ctx, sql.ErrNoRows, "expected challenge not found for deletion")
	}

	// check if challenge is still valid (not expired)
	if time.Since(createdAt) > fleet.OneTimeChallengeTTL {
		return ctxerr.Wrap(ctx, sql.ErrNoRows, "expired challenge")
	}

	return nil
}

// CleanupExpiredChallenges removes expired challenges from the challenges table.
func (ds *Datastore) CleanupExpiredChallenges(ctx context.Context) (int64, error) {
	res, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM challenges WHERE created_at < ?`, time.Now().Add(-fleet.OneTimeChallengeTTL))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "cleanup expired challenges")
	}
	rowCt, _ := res.RowsAffected()

	return rowCt, nil
}
