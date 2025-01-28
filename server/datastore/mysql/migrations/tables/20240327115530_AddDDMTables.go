package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240327115530, Down_20240327115530)
}

func Up_20240327115530(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE mdm_apple_declarations (
    -- declaration_uuid is used as the primary key of the declaration
    declaration_uuid varchar(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

    -- team_id references the team that owns this declaration
    team_id int(10) unsigned NOT NULL DEFAULT '0',

    -- identifier is the "Identifier" field in the declaration, surfaced for convenience.
    identifier varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

    -- name is the name of the declaration
    name varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

    -- raw_json contains a JSON blob with the declaration contents
    raw_json json NOT NULL,

    -- checksum is an MD5 checksum of the declaration, in binary form
    checksum binary(16) NOT NULL,

    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    uploaded_at timestamp NULL DEFAULT NULL,

    PRIMARY KEY (declaration_uuid),
    UNIQUE KEY idx_mdm_apple_declaration_team_identifier (team_id, identifier),
    UNIQUE KEY idx_mdm_apple_declaration_team_name (team_id, name)
)
    `)
	if err != nil {
		return fmt.Errorf("creating mdm_apple_declarations table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE mdm_declaration_labels (
    -- id is used as the primary key of this table
    id int(10) unsigned NOT NULL AUTO_INCREMENT,

    -- apple_declaration_uuid references a declaration
    apple_declaration_uuid varchar(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',


    -- label name is stored here because we need to list the labels in the UI
    -- even if it has been deleted from the labels table.
    label_name varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

    -- label id is nullable in case it gets deleted from the labels table.
    -- A row in this table with label_id = null indicates the "broken" state
    label_id int(10) unsigned DEFAULT NULL,

    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    uploaded_at timestamp NULL DEFAULT NULL,

    PRIMARY KEY (id),
    UNIQUE KEY idx_mdm_declaration_labels_label_name (apple_declaration_uuid, label_name),
    KEY label_id (label_id),
    CONSTRAINT mdm_declaration_labels_ibfk_1 FOREIGN KEY (apple_declaration_uuid) REFERENCES mdm_apple_declarations (declaration_uuid) ON DELETE CASCADE,
    CONSTRAINT mdm_declaration_labels_ibfk_3 FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE SET NULL
)
    `)
	if err != nil {
		return fmt.Errorf("creating mdm_declaration_labels table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE mdm_apple_declaration_activation_references (
    -- declaration_uuid is the declaration that contains the references
    declaration_uuid varchar(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

    -- reference is the declaration_uuid of another declaration
    reference varchar(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

    PRIMARY KEY (declaration_uuid, reference),
    CONSTRAINT FOREIGN KEY (declaration_uuid) REFERENCES mdm_apple_declarations (declaration_uuid) ON UPDATE CASCADE,
    CONSTRAINT FOREIGN KEY (reference) REFERENCES mdm_apple_declarations (declaration_uuid) ON UPDATE CASCADE
)
    `)
	if err != nil {
		return fmt.Errorf("creating mdm_apple_declaration_activation_references table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE host_mdm_apple_declarations (
    -- host_uuid references a host in the hosts table
    host_uuid varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

    -- status represents the status of the declaration in the host
    status varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,

    -- operation_type is used to signal if the declaration is being added, removed, etc
    operation_type varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,

    -- detail contains any messages or errors from the protocol or Fleet
    detail text COLLATE utf8mb4_unicode_ci,

    -- checksum of the currently implemented declaration
    checksum binary(16) NOT NULL,

    -- declaration_uuid references the declaration assigned to the host's team
    declaration_uuid varchar(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

   -- declaration_identifier is the identifier of the declaration
    declaration_identifier varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

    -- declaration_name is the name of the declaration
    declaration_name varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

    PRIMARY KEY (host_uuid, declaration_uuid),
    KEY status (status),
    KEY operation_type (operation_type),
    CONSTRAINT host_mdm_apple_declarations_ibfk_1 FOREIGN KEY (status) REFERENCES mdm_delivery_status (status) ON UPDATE CASCADE,
    CONSTRAINT host_mdm_apple_declarations_ibfk_2 FOREIGN KEY (operation_type) REFERENCES mdm_operation_types (operation_type) ON UPDATE CASCADE
)
    `)
	if err != nil {
		return fmt.Errorf("creatign host_mdm_apple_declarations table %w", err)
	}

	return nil
}

func Down_20240327115530(tx *sql.Tx) error {
	return nil
}
