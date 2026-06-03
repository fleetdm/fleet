package tables

import (
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons, not security
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
)

func init() {
	MigrationClient.AddMigration(Up_20240221112844, Down_20240221112844)
}

// policiesChecksum20240221112844 reproduces the value previously computed by the
// SQL expression unhex(md5(concat_ws(char(0), coalesce(team_id, ”), name))).
// MySQL 9.6/9.7 removed MD5(), so it is computed in Go.
func policiesChecksum20240221112844(teamID *uint, name string) []byte {
	var teamStr string
	if teamID != nil {
		teamStr = strconv.FormatUint(uint64(*teamID), 10)
	}
	sum := md5.Sum([]byte(teamStr + "\x00" + name)) // nolint:gosec
	return sum[:]
}

func Up_20240221112844(tx *sql.Tx) error {
	// The bug was that the checksum was not updating, so we recompute it for
	// existing rows. The checksum was previously recomputed with the SQL MD5()
	// function (removed in MySQL 9.6/9.7); it is now computed in Go.
	isDuplicate := func(err error) bool {
		var driverErr *mysql.MySQLError
		return errors.As(err, &driverErr) && driverErr.Number == mysqlerr.ER_DUP_ENTRY
	}

	type policyRow struct {
		id     uint64
		teamID *uint
		name   string
	}
	rows, err := tx.Query("SELECT id, team_id, name FROM policies")
	if err != nil {
		return fmt.Errorf("failed to query policies table: %w", err)
	}
	defer rows.Close()
	var policies []policyRow
	for rows.Next() {
		var r policyRow
		if err := rows.Scan(&r.id, &r.teamID, &r.name); err != nil {
			return fmt.Errorf("failed to scan policies table: %w", err)
		}
		policies = append(policies, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed during row iteration of policies table: %w", err)
	}

	for _, p := range policies {
		_, err = tx.Exec(`UPDATE policies SET checksum = ? WHERE id = ?`, policiesChecksum20240221112844(p.teamID, p.name), p.id)
		if isDuplicate(err) {
			// Since there is a duplicate name somewhere in the table (should be
			// rare), disambiguate by appending an integer suffix to the name.
			for i := 2; i < 10000; i++ {
				newName := fmt.Sprintf("%s%d", p.name, i)
				_, err = tx.Exec(`UPDATE policies SET name = ?, checksum = ? WHERE id = ?`,
					newName, policiesChecksum20240221112844(p.teamID, newName), p.id)
				if isDuplicate(err) {
					continue
				} else if err != nil {
					// We do not update this row -- it can be updated next time the policy is modified. This should be very rare.
					// Lack of update can happen if duplicate name is 255 characters, or all of the nearly 10000 names we tried are already taken.
					logger.Warn.Printf("failed to update policy id %d", p.id)
				}
				break
			}
		} else if err != nil {
			return fmt.Errorf("failed to update policies table to fill the checksum column on id %d: %w", p.id, err)
		}
	}
	return nil
}

func Down_20240221112844(tx *sql.Tx) error {
	return nil
}
