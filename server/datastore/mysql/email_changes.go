package mysql

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (ds *Datastore) PendingEmailChange(uid uint, newEmail, token string) error {
	sqlStatement := `
    INSERT INTO email_changes (
      user_id,
      token,
      new_email
    ) VALUES( ?, ?, ? )
  `
	_, err := ds.db.Exec(sqlStatement, uid, token, newEmail)
	if err != nil {
		return errors.Wrap(err, "inserting email change record")
	}

	return nil
}

// ConfirmPendingEmailChange finds email change record, updates user with new email,
// then deletes change record if everything succeeds.
func (ds *Datastore) ConfirmPendingEmailChange(token string) (newEmail string, err error) {
	var (
		tx      *sqlx.Tx
		success bool // indicates all db operations success if true
	)
	changeRecord := struct {
		ID       uint
		UserID   uint `db:"user_id"`
		Token    string
		NewEmail string `db:"new_email"`
	}{}
	err = ds.db.Get(&changeRecord, "SELECT * FROM email_changes WHERE token = ?", token)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", notFound("email change with token")
		}
		return "", errors.Wrap(err, "email change")
	}

	tx, err = ds.db.Beginx()
	if err != nil {
		return "", errors.Wrap(err, "begin transaction to change email")
	}
	defer func() {
		if success {
			if err = tx.Commit(); err == nil {
				return // success
			}
			err = errors.Wrap(err, "commit transaction for email change")
			tx.Rollback()
		}
	}()

	query := `
    UPDATE users SET
      email = ?
    WHERE id = ?
  `
	_, err = tx.Exec(query, changeRecord.NewEmail, changeRecord.UserID)
	if err != nil {
		return "", errors.Wrap(err, "updating user's email")
	}
	_, err = tx.Exec("DELETE FROM email_changes WHERE id = ?", changeRecord.ID)
	if err != nil {
		return "", errors.Wrap(err, "deleting email change")
	}
	success = true // cause things to be committed in defer func
	return changeRecord.NewEmail, nil
}
