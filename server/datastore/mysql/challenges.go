package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
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
	fmt.Println("New challenge created:", challenge)
	return challenge, nil
}

// ConsumeChallenge checks if a valid challenge exists in the challenges table
// and deletes it if it does. The error will include sql.ErrNoRows if the challenge
// is not found or is expired.
func (ds *Datastore) ConsumeChallenge(ctx context.Context, challenge string) error {
	if challenge == "" {
		// no challenge provided, treat as invalid
		return ctxerr.Wrap(ctx, sql.ErrNoRows, "consume challenge called with empty challenge")
	}
	// use transaction to ensure atomicity of the challenge check and deletion
	var valid bool
	// msg will hold the reason for invalidation if applicable because any transaction err means
	// we want to retry/rollback, rather when we want to return a validation error to the caller
	var msg string
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// check if matching challenge exists and retrieve its creation time
		var createdAt time.Time
		if err := sqlx.GetContext(ctx, tx, &createdAt, `SELECT created_at FROM challenges WHERE challenge = ?`, challenge); err != nil {
			if err == sql.ErrNoRows {
				// invalid, challenge not found
				msg = "challenge not found"
				return nil
			}
			// some other error, return it
			return ctxerr.Wrap(ctx, err, "get challenge")
		}
		// delete challenge regardless of validity
		r, err := tx.ExecContext(ctx, `DELETE FROM challenges WHERE challenge = ?`, challenge)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete challenge")
		}
		if rowCt, _ := r.RowsAffected(); rowCt < 1 {
			// unlikely to happen since just checked existence and we're in a transaction,
			// but we'll treat as invalid so we log as error for debugging purposes just in case
			msg = "challenge not found for deletion"
			return nil
		}
		// check expiry
		if time.Since(createdAt) <= fleet.OneTimeChallengeTTL {
			valid = true
		} else {
			msg = "challenge expired"
		}
		return nil
	})

	switch {
	case err != nil:
		// if we encountered an error during the transaction, return it
		return ctxerr.Wrap(ctx, err, "consume challenge transaction")
	case valid:
		// challenge consumed successfully
		return nil
	default:
		// challenge was invalid or expired, treat as not found
		return ctxerr.Wrap(ctx, sql.ErrNoRows, msg)
	}
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
