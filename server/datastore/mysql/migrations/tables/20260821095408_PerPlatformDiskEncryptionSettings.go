package tables

import (
	"database/sql"
	"encoding/json"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260821095408, Down_20260821095408)
}

// Up_20260821095408 splits the single mdm.enable_disk_encryption toggle into
// four per-platform settings, initializing all of them from the old value so
// upgraded configs behave identically:
//   - mdm.macos_settings.enable_disk_encryption
//   - mdm.macos_settings.enable_escrow_disk_encryption_key
//   - mdm.windows_settings.enable_disk_encryption
//   - mdm.linux_settings.enable_escrow_disk_encryption_key
//
// An absent or false old value writes explicit false values (declarative
// readers must not treat absence as unset). The rewrite is done on generic
// JSON maps so no other key is altered, and it deliberately does NOT touch
// mdm_apple_configuration_profiles: the both-on state renders the exact same
// FileVault profile, so no profile bytes, checksums, or uploaded_at change and
// nothing is re-sent to hosts.
func Up_20260821095408(tx *sql.Tx) error {
	// app config
	var raw []byte
	err := tx.QueryRow(`SELECT json_value FROM app_config_json LIMIT 1`).Scan(&raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// fresh install, nothing to rewrite
	case err != nil:
		return errors.Wrap(err, "select app_config_json")
	default:
		updated, err := fanOutDiskEncryptionSetting(raw)
		if err != nil {
			return errors.Wrap(err, "rewrite app_config_json disk encryption settings")
		}
		if _, err := tx.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1`, updated); err != nil {
			return errors.Wrap(err, "update app_config_json")
		}
	}

	// team configs
	rows, err := tx.Query(`SELECT id, config FROM teams WHERE config IS NOT NULL`)
	if err != nil {
		return errors.Wrap(err, "select teams config")
	}
	defer rows.Close()

	type teamUpdate struct {
		id     uint
		config []byte
	}
	var updates []teamUpdate
	for rows.Next() {
		var id uint
		var config []byte
		if err := rows.Scan(&id, &config); err != nil {
			return errors.Wrap(err, "scan team config")
		}
		updated, err := fanOutDiskEncryptionSetting(config)
		if err != nil {
			return errors.Wrapf(err, "rewrite team %d disk encryption settings", id)
		}
		updates = append(updates, teamUpdate{id: id, config: updated})
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "iterate teams")
	}

	for _, u := range updates {
		if _, err := tx.Exec(`UPDATE teams SET config = ? WHERE id = ?`, u.config, u.id); err != nil {
			return errors.Wrapf(err, "update team %d config", u.id)
		}
	}

	return nil
}

// fanOutDiskEncryptionSetting rewrites one config JSON blob (app config or
// team config): the old flat mdm.enable_disk_encryption value (absent = false)
// is written to all four per-platform settings. Existing values under the
// per-platform keys are overwritten — the flat toggle was the source of truth
// before this migration (the macos_settings key was a write-through alias of
// it, so they never legitimately diverged).
func fanOutDiskEncryptionSetting(raw []byte) ([]byte, error) {
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, errors.Wrap(err, "unmarshal config")
	}
	if config == nil {
		config = map[string]any{}
	}

	mdm, ok := config["mdm"].(map[string]any)
	if !ok {
		mdm = map[string]any{}
		config["mdm"] = mdm
	}
	enabled, _ := mdm["enable_disk_encryption"].(bool) // absent or non-bool means false

	for _, k := range []string{"macos_settings", "windows_settings", "linux_settings"} {
		if _, ok := mdm[k].(map[string]any); !ok {
			mdm[k] = map[string]any{}
		}
	}
	mdm["macos_settings"].(map[string]any)["enable_disk_encryption"] = enabled
	mdm["macos_settings"].(map[string]any)["enable_escrow_disk_encryption_key"] = enabled
	mdm["windows_settings"].(map[string]any)["enable_disk_encryption"] = enabled
	mdm["linux_settings"].(map[string]any)["enable_escrow_disk_encryption_key"] = enabled

	updated, err := json.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "marshal config")
	}
	return updated, nil
}

func Down_20260821095408(tx *sql.Tx) error {
	return nil
}
