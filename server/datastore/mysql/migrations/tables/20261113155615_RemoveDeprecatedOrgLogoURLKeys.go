package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261113155615, Down_20261113155615)
}

// orgLogoURLKeyRemovals are the deprecated org_info logo URL keys and the mode-aware keys that replace them.
var orgLogoURLKeyRemovals = []struct{ Old, New string }{
	{Old: "org_logo_url", New: "org_logo_url_dark_mode"},
	{Old: "org_logo_url_light_background", New: "org_logo_url_light_mode"},
}

// Up_20261113155615 drops the deprecated org logo URL keys from the stored config. Fleet kept them mirrored to the new keys
// on every write, but a config last saved before the new keys existed only has the old one, so its value moves to the new
// key when that one is missing or empty.
func Up_20261113155615(tx *sql.Tx) error {
	for _, r := range orgLogoURLKeyRemovals {
		oldPath := "$.org_info." + r.Old
		newPath := "$.org_info." + r.New
		//nolint:gosec // built from orgLogoURLKeyRemovals, no user input
		stmt := fmt.Sprintf(`UPDATE app_config_json SET json_value = JSON_REMOVE(
				IF(COALESCE(JSON_UNQUOTE(JSON_EXTRACT(json_value, '%[2]s')), '') = '',
					JSON_SET(json_value, '%[2]s', JSON_EXTRACT(json_value, '%[1]s')),
					json_value),
				'%[1]s')
			WHERE JSON_CONTAINS_PATH(json_value, 'one', '%[1]s')`,
			oldPath, newPath)
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("remove %s from app_config_json: %w", oldPath, err)
		}
	}
	return nil
}

func Down_20261113155615(tx *sql.Tx) error {
	return nil
}
