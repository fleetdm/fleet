package tables

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20230328091816, Down_20230328091816)
}

func fixupSoftware(tx *sql.Tx, collation string) error {
	rows, err := tx.Query(`
         SELECT
           COUNT(*) as total,
           CONCAT('[', GROUP_CONCAT(id SEPARATOR ','), ']') as ids
         FROM software
         GROUP BY ` +
		fmt.Sprintf("`version` COLLATE %s,", collation) +
		fmt.Sprintf("`release` COLLATE %s,", collation) +
		fmt.Sprintf(`name      COLLATE %s,
		source    COLLATE %s,
		vendor    COLLATE %s,
		arch      COLLATE %s
		HAVING total
		COLLATE %s`, collation, collation, collation, collation, collation))

	if err != nil {
		return err
	}

	defer rows.Close()
	var idGroups [][]uint
	for rows.Next() {
		var rawIDs json.RawMessage
		var total int
		if err := rows.Scan(&total, &rawIDs); err != nil {
			return err
		}
		var ids []uint
		if err := json.Unmarshal(rawIDs, &ids); err != nil {
			return err
		}
		idGroups = append(idGroups, ids)
	}

	for _, ids := range idGroups {
		for i := 1; i < len(ids); i++ {
			if _, err := tx.Exec("DELETE FROM software_cve WHERE software_id = ?", ids[i]); err != nil {
				return err
			}
			if _, err := tx.Exec("DELETE FROM software_host_counts WHERE software_id = ?", ids[i]); err != nil {
				return err
			}
			if _, err := tx.Exec("UPDATE host_software SET software_id = ? WHERE software_id = ?", ids[0], ids[i]); err != nil {
				return err
			}
			if _, err := tx.Exec("DELETE FROM software WHERE id = ?", ids[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func fixupHostUsers(tx *sql.Tx, collation string) error {
	rows, err := tx.Query(fmt.Sprintf(`
         SELECT
           COUNT(*) as total,
           CONCAT('[', GROUP_CONCAT(JSON_OBJECT('username', username, 'host_id', host_id, 'uid', uid) SEPARATOR ","), ']') as ids
         FROM host_users
         GROUP BY
           host_id,
           uid,
           username COLLATE %s
         HAVING total > 1
         COLLATE %s`, collation, collation))
	if err != nil {
		return err
	}

	type hostUser struct {
		Username string
		HostID   uint `json:"host_id"`
		UID      uint
	}

	defer rows.Close()
	var keyGroups [][]hostUser
	for rows.Next() {
		var raw json.RawMessage
		var total int
		if err := rows.Scan(&total, &raw); err != nil {
			return err
		}

		var hu []hostUser
		if err := json.Unmarshal(raw, &hu); err != nil {
			return err
		}
		keyGroups = append(keyGroups, hu)
	}

	for _, keys := range keyGroups {
		for i := 1; i < len(keys); i++ {
			if _, err := tx.Exec("DELETE FROM host_users WHERE host_id = ? AND uid = ? AND username = ?", keys[i].HostID, keys[i].UID, keys[i].Username); err != nil {
				return err
			}
		}
	}
	return nil
}

func fixupOS(tx *sql.Tx, collation string) error {
	rows, err := tx.Query(fmt.Sprintf(`
         SELECT
           COUNT(*) as total,
           CONCAT('[', GROUP_CONCAT(JSON_OBJECT('name', name, 'version', version, 'arch', arch, 'kernel_version', kernel_version, 'platform', platform) SEPARATOR ","), ']') as ids
         FROM operating_systems
         GROUP BY `+
		fmt.Sprintf("`version` COLLATE %s,", collation)+
		`name      COLLATE %s,
           arch    COLLATE %s,
           kernel_version COLLATE %s,
           platform    COLLATE %s
         HAVING total > 1
         COLLATE %s`, collation, collation, collation, collation, collation))
	if err != nil {
		return err
	}

	type os struct {
		Name          string
		Version       string
		Arch          string
		KernelVersion string `json:"kernel_version"`
		Platform      string
	}

	defer rows.Close()
	var keyGroups [][]os
	for rows.Next() {
		var raw json.RawMessage
		var total int
		if err := rows.Scan(&total, &raw); err != nil {
			return err
		}

		var o []os
		if err := json.Unmarshal(raw, &o); err != nil {
			return err
		}
		keyGroups = append(keyGroups, o)
	}

	for _, keys := range keyGroups {
		for i := 1; i < len(keys); i++ {
			if _, err := tx.Exec("DELETE FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? AND platform = ?", keys[i].Name, keys[i].Version, keys[i].Arch, keys[i].KernelVersion, keys[i].Platform); err != nil {
				return err
			}
		}
	}
	return nil
}

// changeCollation changes the default collation set of the database and all
// table to the provided collation
//
// This is based on the changeCharacterSet function that's included in this
// module and part of the 20170306075207_UseUTF8MB migration.
func changeCollation(tx *sql.Tx, charset string, collation string) (err error) {
	if err := fixupSoftware(tx, collation); err != nil {
		return err
	}

	if err := fixupHostUsers(tx, collation); err != nil {
		return err
	}

	if err := fixupOS(tx, collation); err != nil {
		return err
	}

	_, err = tx.Exec(fmt.Sprintf("ALTER DATABASE DEFAULT CHARACTER SET `%s` COLLATE `%s`", charset, collation))
	if err != nil {
		return fmt.Errorf("alter database: %w", err)
	}

	txx := sqlx.Tx{Tx: tx}
	var names []string
	err = txx.Select(&names, `
          SELECT table_name
          FROM information_schema.TABLES AS T, information_schema.COLLATION_CHARACTER_SET_APPLICABILITY AS C
          WHERE C.collation_name = T.table_collation
          AND T.table_schema = (SELECT database())
          AND (C.CHARACTER_SET_NAME != ? OR C.COLLATION_NAME != ?)
	  -- exclude tables that have columns with specific collations
	  AND table_name NOT IN ('hosts', 'enroll_secrets')`, charset, collation)
	if err != nil {
		return fmt.Errorf("selecting tables: %w", err)
	}

	// disable foreign checks before changing the collations, otherwise the
	// migration might fail. These are re-enabled after we're done.
	defer func() {
		if _, execErr := tx.Exec("SET FOREIGN_KEY_CHECKS = 1"); execErr != nil {
			err = fmt.Errorf("re-enabling foreign key checks: %w", err)
		}
	}()
	if _, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return fmt.Errorf("disabling foreign key checks: %w", err)
	}
	for _, name := range names {
		_, err = tx.Exec(fmt.Sprintf("ALTER TABLE `%s` CONVERT TO CHARACTER SET `%s` COLLATE `%s`", name, charset, collation))
		if err != nil {
			return fmt.Errorf("alter table %s: %w", name, err)
		}
	}

	// enroll secrets was intentionally excluded because it contains only
	// one "text" column, and that column contains a specific collation.
	//
	// note the use of DEFAULT below to indicate that new columns will
	// contain the desired collation unless explicitly stated.
	_, err = tx.Exec(fmt.Sprintf("ALTER TABLE enroll_secrets DEFAULT CHARACTER SET `%s` COLLATE `%s`", charset, collation))
	if err != nil {
		return fmt.Errorf("alter table enroll_secrets: %w", err)
	}

	// `hosts` was intentionally excluded, change the collation of all
	// "text" columns except for `node_key` and `orbit_node_key`
	//
	// note the use of DEFAULT below to indicate that new columns will
	// contain the desired collation unless explicitly stated.
	tmpl := template.Must(template.New("").Parse(`
	        ALTER TABLE hosts DEFAULT CHARACTER SET {{ .Cs }} COLLATE {{ .Co }},
		MODIFY osquery_host_id   varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} DEFAULT NULL,
		MODIFY hostname          varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY uuid              varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY platform          varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY osquery_version   varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY os_version        varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY build             varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY platform_like     varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY code_name         varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY cpu_type          varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY cpu_subtype       varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY cpu_brand         varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY hardware_vendor   varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY hardware_model    varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY hardware_version  varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY hardware_serial   varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY computer_name     varchar(255) CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY primary_ip        varchar(45)  CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY primary_mac       varchar(17)  CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT '',
		MODIFY public_ip         varchar(45)  CHARACTER SET {{ .Cs }} COLLATE {{ .Co }} NOT NULL DEFAULT ''`))
	var stmt bytes.Buffer
	if err := tmpl.Execute(&stmt, map[string]string{"Cs": charset, "Co": collation}); err != nil {
		return fmt.Errorf("executing template to alter hosts: %w", err)
	}
	if _, err = tx.Exec(stmt.String()); err != nil {
		return fmt.Errorf("alter table hosts: %w", err)
	}
	return err
}

func Up_20230328091816(tx *sql.Tx) error {
	// while newer versions of MySQL default to utf8mb4_0900_ai_ci, we
	// still need to support 5.7, so we choose utf8mb4_unicode_ci for more
	// details on the rationale, see:
	// https://github.com/fleetdm/fleet/pull/10515#discussion_r1137611693
	return changeCollation(tx, "utf8mb4", "utf8mb4_unicode_ci")
}

func Down_20230328091816(tx *sql.Tx) error {
	return nil
}
