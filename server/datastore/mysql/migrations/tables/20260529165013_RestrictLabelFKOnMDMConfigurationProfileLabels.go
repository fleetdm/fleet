package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260529165013, Down_20260529165013)
}

func Up_20260529165013(tx *sql.Tx) error {
	// mdm_configuration_profile_labels
	cpConstraints, err := constraintsForTable(tx, "mdm_configuration_profile_labels", map[string]struct{}{"labels": {}})
	if err != nil {
		return err
	}
	if len(cpConstraints) != 1 {
		return errors.New("mdm_configuration_profile_labels foreign key to labels not found")
	}
	if _, err := tx.Exec(fmt.Sprintf(`
		ALTER TABLE mdm_configuration_profile_labels
		DROP FOREIGN KEY %s,
		ADD CONSTRAINT mdm_configuration_profile_labels_ibfk_label FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE RESTRICT
	`, cpConstraints[0])); err != nil {
		return fmt.Errorf("altering mdm_configuration_profile_labels label_id foreign key: %w", err)
	}

	// mdm_declaration_labels
	declConstraints, err := constraintsForTable(tx, "mdm_declaration_labels", map[string]struct{}{"labels": {}})
	if err != nil {
		return err
	}
	if len(declConstraints) != 1 {
		return errors.New("mdm_declaration_labels foreign key to labels not found")
	}
	if _, err := tx.Exec(fmt.Sprintf(`
		ALTER TABLE mdm_declaration_labels
		DROP FOREIGN KEY %s,
		ADD CONSTRAINT mdm_declaration_labels_ibfk_label FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE RESTRICT
	`, declConstraints[0])); err != nil {
		return fmt.Errorf("altering mdm_declaration_labels label_id foreign key: %w", err)
	}

	return nil
}

func Down_20260529165013(tx *sql.Tx) error {
	return nil
}
