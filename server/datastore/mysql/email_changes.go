package mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) PendingEmailChange(ctx context.Context, uid uint, newEmail, token string) error {
	sqlStatement := `
    INSERT INTO email_changes (
      user_id,
      token,
      new_email
    ) VALUES( ?, ?, ? )
  `
	_, err := ds.writer.ExecContext(ctx, sqlStatement, uid, token, newEmail)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting email change record")
	}

	return nil
}

// ConfirmPendingEmailChange finds email change record, updates user with new email,
// then deletes change record if everything succeeds.
func (ds *Datastore) ConfirmPendingEmailChange(ctx context.Context, id uint, token string) (newEmail string, err error) {
	changeRecord := struct {
		ID       uint
		UserID   uint `db:"user_id"`
		Token    string
		NewEmail string `db:"new_email"`
	}{}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, tx, &changeRecord, "SELECT * FROM email_changes WHERE token = ? AND user_id = ?", token, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return ctxerr.Wrap(ctx, notFound("email change with token"))
			}
			return ctxerr.Wrap(ctx, err, "email change")
		}

		query := `
    		UPDATE users SET
      			email = ?
    		WHERE id = ?
  `
		results, err := tx.ExecContext(ctx, query, changeRecord.NewEmail, changeRecord.UserID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating user's email")
		}

		rowsAffected, err := results.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "fetching affected rows updating user's email")
		}
		if rowsAffected == 0 {
			return ctxerr.Wrap(ctx, notFound("User").WithID(changeRecord.UserID))
		}

		_, err = tx.ExecContext(ctx, "DELETE FROM email_changes WHERE id = ?", changeRecord.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting email change")
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return changeRecord.NewEmail, nil
}
