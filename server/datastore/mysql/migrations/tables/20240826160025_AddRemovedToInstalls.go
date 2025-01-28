package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20240826160025, Down_20240826160025)
}

func Up_20240826160025(tx *sql.Tx) error {
	if !columnExists(tx, "host_software_installs", "removed") {
		if _, err := tx.Exec("ALTER TABLE host_software_installs ADD COLUMN removed TINYINT NOT NULL DEFAULT 0"); err != nil {
			return fmt.Errorf("failed to add removed to host_software_installs: %w", err)
		}
	}

	if !columnExists(tx, "host_vpp_software_installs", "removed") {
		if _, err := tx.Exec("ALTER TABLE host_vpp_software_installs ADD COLUMN removed TINYINT NOT NULL DEFAULT 0"); err != nil {
			return fmt.Errorf("failed to add removed to host_vpp_software_installs: %w", err)
		}
	}

	// Mark software installs as removed if the software is no longer installed on the host.
	// Note that some software never shows up in software table after being installed (because software detail query can't find it).
	// So, we will only mark software as removed if it shows up in the software table (for another host).
	getPackagesRemovedStmt := `
	SELECT DISTINCT hsi.id
	FROM host_software_installs hsi
	INNER JOIN software_installers si ON hsi.software_installer_id = si.id
	INNER JOIN software_titles st ON si.title_id = st.id
	-- software is installed on some host
	INNER JOIN software s ON s.title_id = st.id
	INNER JOIN hosts h ON hsi.host_id = h.id
	WHERE NOT EXISTS (
		-- software is not installed on this specific host
		SELECT 1 FROM host_software hs
		INNER JOIN software s2 ON hs.software_id = s2.id AND s2.title_id = st.id
		WHERE hs.host_id = hsi.host_id
	) AND (
		-- software status is: Installed
		(hsi.post_install_script_exit_code IS NOT NULL AND hsi.post_install_script_exit_code = 0) OR
		(hsi.post_install_script_exit_code IS NULL AND hsi.install_script_exit_code IS NOT NULL AND hsi.install_script_exit_code = 0)
	) AND
		-- software was refetched after it was installed
		hsi.updated_at < h.detail_updated_at
`
	var ids []uint
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	if err := txx.Select(&ids, getPackagesRemovedStmt); err != nil {
		return fmt.Errorf("failed to find host_software_installs to remove: %w", err)
	}
	if len(ids) > 0 {
		stmt, args, err := sqlx.In("UPDATE host_software_installs SET removed = 1	WHERE id IN (?)", ids)
		if err != nil {
			return fmt.Errorf("failed to expand slice value for host_software_installs: %w", err)
		}
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("failed to mark host_software_installs as removed: %w", err)
		}
	}

	// Mark VPP installs as removed if the software is no longer installed on the host.
	getVppRemovedStmt := `
	SELECT DISTINCT hvsi.id
	FROM host_vpp_software_installs hvsi
	INNER JOIN vpp_apps vap ON hvsi.adam_id = vap.adam_id AND hvsi.platform = vap.platform
	INNER JOIN software_titles st ON vap.title_id = st.id
	INNER JOIN hosts h ON hvsi.host_id = h.id
	WHERE NOT EXISTS (
		-- software is not installed on this specific host
		SELECT 1 FROM host_software hs
		INNER JOIN software s2 ON hs.software_id = s2.id AND s2.title_id = st.id
		WHERE hs.host_id = hvsi.host_id
	) AND
		-- software was refetched after it was installed
		hvsi.updated_at < h.detail_updated_at
`

	var vppIDs []uint
	if err := txx.Select(&vppIDs, getVppRemovedStmt); err != nil {
		return fmt.Errorf("failed to find host_vpp_software_installs to remove: %w", err)
	}
	if len(vppIDs) > 0 {
		stmt, args, err := sqlx.In("UPDATE host_vpp_software_installs SET removed = 1	WHERE id IN (?)", vppIDs)
		if err != nil {
			return fmt.Errorf("failed to expand slice value for host_vpp_software_installs: %w", err)
		}
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("failed to mark host_vpp_software_installs as removed: %w", err)
		}
	}

	return nil
}

func Down_20240826160025(_ *sql.Tx) error {
	return nil
}
