package tables

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20230602111827, Down_20230602111827)
}

func Up_20230602111827(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	type mdmSolution struct {
		ID        uint   `db:"id"`
		ServerURL string `db:"server_url"`
		Name      string `db:"name"`
	}

	// first, find all the MDM solutions
	var mdmSolutions []mdmSolution
	err := txx.Select(
		&mdmSolutions,
		`SELECT id, server_url, name
                 FROM mobile_device_management_solutions
                 ORDER BY created_at DESC`,
	)
	if err != nil {
		return fmt.Errorf("selecting mobile_device_management_solutions: %w", err)
	}

	// find all the dupes
	uniqs := map[string]mdmSolution{}
	dupes := []uint{}
	for _, solution := range mdmSolutions {
		serverURL, err := url.Parse(solution.ServerURL)
		if err != nil {
			logger.Warn.Printf("unable to parse server_url %s, skipping\n", serverURL)
			continue
		}
		// strip any query parameters from the URL
		serverURL.RawQuery = ""
		cleanURL := serverURL.String()

		uniqSolution, ok := uniqs[cleanURL]
		if !ok {
			uniqs[cleanURL] = solution
			continue
		}

		dupes = append(dupes, solution.ID)

		// update host_mdm entries to point to the new solution
		_, err = txx.Exec(
			`UPDATE host_mdm SET server_url = ?, mdm_id = ?
                         WHERE mdm_id = ?`, cleanURL, uniqSolution.ID, solution.ID)
		if err != nil {
			return fmt.Errorf("updating host_mdm entries with new solution: %w", err)
		}
	}

	// delete all duplicated solutions
	if len(dupes) > 0 {
		stmt, args, err := sqlx.In(`DELETE FROM mobile_device_management_solutions WHERE id IN (?)`, dupes)
		if err != nil {
			return fmt.Errorf("building SQL IN statement: %w", err)
		}
		_, err = txx.Exec(stmt, args...)
		if err != nil {
			return fmt.Errorf("deleting duplicated MDM solutions: %w", err)
		}
	}

	// make sure all the new solutions have the right URL
	if len(uniqs) > 0 {
		inPart := ""
		args := []interface{}{}
		for serverURL, solution := range uniqs {
			inPart += "(?, ?, ?),"
			args = append(args, solution.ID, solution.Name, serverURL)

			// and the related host_mdm rows as well
			_, err = txx.Exec(`UPDATE host_mdm SET server_url = ? WHERE mdm_id = ?`, serverURL, solution.ID)
			if err != nil {
				return fmt.Errorf("updating host_mdm entries with new server_url: %w", err)
			}
		}
		stmt := `
	INSERT INTO mobile_device_management_solutions (id, name, server_url)
          VALUES %s
          ON DUPLICATE KEY UPDATE server_url = VALUES(server_url)
	`
		_, err = tx.Exec(
			fmt.Sprintf(stmt, strings.TrimSuffix(inPart, ",")),
			args...,
		)
		if err != nil {
			return fmt.Errorf("updating mobile_device_management_solutions server_url: %w", err)
		}
	}

	return nil
}

func Down_20230602111827(tx *sql.Tx) error {
	return nil
}
