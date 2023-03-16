package tables

import (
	"bytes"
	"database/sql"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"
)

func init() {
	MigrationClient.AddMigration(Up_20230315104937, Down_20230315104937)
}

// changeCollation changes the default collation set of the database and all
// table to the provided collation
//
// This is based on the changeCharacterSet function that's included in this
// module and part of the 20170306075207_UseUTF8MB migration.
func changeCollation(tx *sql.Tx, charset string, collation string) (err error) {
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
	// note the use of DEFAULT above to indicate that new columns will
	// contain the desired collation unless explicitly stated.
	_, err = tx.Exec(fmt.Sprintf("ALTER TABLE enroll_secrets DEFAULT CHARACTER SET `%s` COLLATE `%s`", charset, collation))
	if err != nil {
		return fmt.Errorf("alter table enroll_secrets: %w", err)
	}

	// `hosts` was intentionally excluded, change the collation of all
	// "text" columns except for `node_key` and `orbit_node_key`
	//
	// note the use of DEFAULT above to indicate that new columns will
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

func Up_20230315104937(tx *sql.Tx) error {
	// while newer versions of MySQL default to
	// utf8mb4_0900_ai_ci, we still need to support 5.7, which
	// defaults to utf8mb4_general_ci
	return changeCollation(tx, "utf8mb4", "utf8mb4_general_ci")
}

func Down_20230315104937(tx *sql.Tx) error {
	return nil
}
