package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260803182251, Down_20260803182251)
}

func Up_20260803182251(tx *sql.Tx) error {
	// scim_group_group stores direct parent -> child SCIM group edges. Microsoft
	// Entra ID provisions nested groups by sending group-type members (e.g. a
	// PATCH that adds "group-62" as a member of "group-61") rather than flattening
	// them into user members. This table records those edges so that a user's
	// effective (transitive) group membership can be resolved by walking the graph.
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS scim_group_group (
	    parent_group_id INT UNSIGNED NOT NULL,
	    child_group_id INT UNSIGNED NOT NULL,
	    created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
	    PRIMARY KEY (parent_group_id, child_group_id),
	    KEY idx_scim_group_group_child (child_group_id),
	    CONSTRAINT fk_scim_group_group_parent FOREIGN KEY (parent_group_id) REFERENCES scim_groups (id) ON DELETE CASCADE,
	    CONSTRAINT fk_scim_group_group_child FOREIGN KEY (child_group_id) REFERENCES scim_groups (id) ON DELETE CASCADE
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)
	if err != nil {
		return fmt.Errorf("failed to create scim_group_group table: %s", err)
	}

	return nil
}

func Down_20260803182251(tx *sql.Tx) error {
	return nil
}
