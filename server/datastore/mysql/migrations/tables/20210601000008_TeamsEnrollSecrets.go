package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210601000008, Down_20210601000008)
}

func Up_20210601000008(tx *sql.Tx) error {
	// Add team_id
	sql := `
		ALTER TABLE enroll_secrets
		ADD COLUMN team_id INT UNSIGNED,
		ADD FOREIGN KEY fk_enroll_secrets_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add team_id to enroll_secrets")
	}

	// Remove "active" as a concept from enroll secrets
	sql = `
		DELETE FROM enroll_secrets
		WHERE NOT active
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "remove inactive secrets")
	}

	// ********* TEST ONLY BEGIN *********
	// This will make an enroll secrets test fail because it should end up with one unexpected secret
	// if _, err := tx.Exec(
	//	`INSERT INTO enroll_secrets (secret, name) VALUES ('aaaa', '1'), ('aaaa', '2'), ('aaaa', '3')`); err != nil {
	//	return errors.Wrap(err, "add red hat label")
	// }
	// ********* TEST ONLY ENDS  *********

	//nolint
	rows, err := tx.Query(`SELECT secret, count(secret) FROM enroll_secrets GROUP BY secret HAVING count(secret) > 1`)
	if err != nil {
		return errors.Wrap(err, "remove duplicate secrets")
	}
	type sec struct {
		secret string
		count  int
	}
	var secretsToReduce []sec
	for rows.Next() {
		var secret string
		var c int

		if err = rows.Scan(&secret, &c); err != nil {
			return errors.Wrap(err, "scanning duplicated secrets")
		}
		secretsToReduce = append(secretsToReduce, sec{secret: secret, count: c})
	}
	for _, s := range secretsToReduce {
		// Remove duplicate secrets
		if _, err := tx.Exec(
			`DELETE FROM enroll_secrets WHERE secret = ? LIMIT ?`,
			s.secret, s.count-1,
		); err != nil {
			return errors.Wrap(err, "remove inactive secrets")
		}
	}

	sql = `
		ALTER TABLE enroll_secrets
		DROP COLUMN active,
		DROP COLUMN name,
		ADD PRIMARY KEY (secret)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "alter enroll_secrets")
	}

	sql = `
		ALTER TABLE hosts
		DROP COLUMN enroll_secret_name
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "alter hosts")
	}

	return nil
}

func Down_20210601000008(tx *sql.Tx) error {
	return nil
}
