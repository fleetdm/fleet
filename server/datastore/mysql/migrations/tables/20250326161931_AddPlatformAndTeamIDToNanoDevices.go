package tables

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20250326161931, Down_20250326161931)
}

func Up_20250326161931(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE nano_devices
	ADD COLUMN platform VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	ADD COLUMN enroll_team_id INT UNSIGNED DEFAULT NULL,
	ADD CONSTRAINT fk_nano_devices_team_id FOREIGN KEY (enroll_team_id) REFERENCES teams (id) ON DELETE SET NULL
`)
	if err != nil {
		return fmt.Errorf("failed to alter nano_devices: %w", err)
	}

	// parse the authenticate field of existing nano_devices that don't have a
	// corresponding host entry (deleted hosts) and set the platform value to
	// ios/ipados so that they can be recreated when they checkin.
	const deletedDevicesStmt = `
SELECT
	d.id,
	d.authenticate
FROM
	nano_devices d
	JOIN nano_enrollments e ON d.id = e.device_id
	LEFT OUTER JOIN hosts h ON h.uuid = d.id
WHERE
	e.type = 'Device' AND
	e.enabled = 1 AND
	h.id IS NULL
`

	var deletedDevices []struct {
		ID           string `db:"id"`
		Authenticate string `db:"authenticate"`
	}
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	if err := txx.Select(&deletedDevices, deletedDevicesStmt); err != nil {
		return fmt.Errorf("selecting existing deleted devices to update the platform: %w", err)
	}

	for _, d := range deletedDevices {
		msg, err := mdm.DecodeCheckin([]byte(d.Authenticate))
		if err != nil {
			// ignore invalid authenticate messages, we won't be able to re-create those devices
			continue
		}

		authMsg, ok := msg.(*mdm.Authenticate)
		if !ok {
			// ignore invalid authenticate messages, we won't be able to re-create those devices
			continue
		}

		platform := "darwin"
		iPhone := strings.HasPrefix(authMsg.ProductName, "iPhone")
		iPad := strings.HasPrefix(authMsg.ProductName, "iPad")
		if iPhone {
			platform = "ios"
		} else if iPad {
			platform = "ipados"
		}

		_, err = tx.Exec(`UPDATE nano_devices SET platform = ? WHERE id = ?`, platform, d.ID)
		if err != nil {
			return fmt.Errorf("failed to update nano_devices: %w", err)
		}
	}
	return nil
}

func Down_20250326161931(tx *sql.Tx) error {
	return nil
}
