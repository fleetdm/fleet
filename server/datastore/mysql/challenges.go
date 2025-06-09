package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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

// HasChallenge checks if a valid challenge exists in the challenges table
// and deletes it if it does. If the challenge does not exist or is not valid (i.e. expired),
// an error is returned.
func (ds *Datastore) HasChallenge(ctx context.Context, challenge string) (bool, error) {
	if challenge == "" {
		return false, ctxerr.New(ctx, "challenge cannot be empty")
	}

	var valid bool
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// check if matching challenge exists
		var createdAt time.Time
		err := sqlx.GetContext(ctx, tx, &createdAt, `SELECT created_at FROM challenges WHERE challenge = ?`, challenge)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil // invalid, challenge does not exist
		case err != nil:
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
			return ctxerr.New(ctx, "expected challenge not found for deletion")
		}
		// check if challenge is still valid (not expired)
		if time.Since(createdAt) <= 1*time.Hour {
			valid = true
		}
		return nil
	})
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "nas challenge")
	}

	return valid, nil
}

// CleanupExpiredChallenges removes expired challenges from the challenges table.
func (ds *Datastore) CleanupExpiredChallenges(ctx context.Context) (int64, error) {
	res, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM challenges WHERE created_at < ?`, time.Now().Add(-1*time.Hour))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "cleanup expired challenges")
	}
	rowCt, _ := res.RowsAffected()

	return rowCt, nil
}
