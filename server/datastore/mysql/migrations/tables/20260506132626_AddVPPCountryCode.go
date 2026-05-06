package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260506132626, Down_20260506132626)
}

func Up_20260506132626(tx *sql.Tx) error {
	// country_code is the App Store storefront country (lowercase ISO 3166-1
	// alpha-2 such as "us", "de", "fr") associated with the row.
	//
	// On vpp_tokens, it identifies the country/storefront tied to the Apple
	// Business Manager account that owns the token. The value is fetched from
	// Apple's /client/config endpoint when the token is uploaded.
	//
	// On vpp_apps, it identifies the storefront the app was first added from
	// in Fleet ("anchored" storefront). All future metadata fetches for that
	// app come from this storefront, regardless of which team triggers the
	// fetch.
	//
	// Both columns are nullable so existing rows survive the migration; values
	// are then populated lazily by the API/service layer.
	if _, err := tx.Exec(`ALTER TABLE vpp_tokens ADD COLUMN country_code VARCHAR(4) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL`); err != nil {
		return fmt.Errorf("add country_code to vpp_tokens: %w", err)
	}

	if _, err := tx.Exec(`ALTER TABLE vpp_apps ADD COLUMN country_code VARCHAR(4) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL`); err != nil {
		return fmt.Errorf("add country_code to vpp_apps: %w", err)
	}

	return nil
}

func Down_20260506132626(tx *sql.Tx) error {
	return nil
}
