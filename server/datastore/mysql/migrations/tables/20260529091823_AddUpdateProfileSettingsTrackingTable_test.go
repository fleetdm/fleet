package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20260529091823(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert two declarations and two windows profiles, one with software update and one without.
	decl1 := `{"declaration": {"identifier": "com.apple.configuration.softwareupdate.enforcement.specific"}}`
	decl1UUID := uuid.NewString()
	decl2 := `{"declaration": {"identifier": "com.apple.configuration.someotherdeclaration"}}`
	decl2UUID := uuid.NewString()
	decl3UUID := uuid.NewString() // Fleet uploaded macOS
	decl4UUID := uuid.NewString() // Fleet uploaded iOS
	decl5UUID := uuid.NewString() // Fleet uploaded iPadOS
	if _, err := db.Exec(`INSERT INTO mdm_apple_declarations (declaration_uuid, identifier, name, raw_json) VALUES (?, "id-1", "id-1-name", ?), (?, "id-2", "id-2-name", ?), (?, "id-3", "Fleet macOS OS Updates", ?), (?, "id-4", "Fleet iOS OS Updates", ?), (?, "id-5", "Fleet iPadOS OS Updates", ?)`, decl1UUID, decl1, decl2UUID, decl2, decl3UUID, decl1, decl4UUID, decl1, decl5UUID, decl1); err != nil {
		t.Fatalf("insert apple declarations: %v", err)
	}

	profile1 := `<SyncML><Target><LocURI>./Vendor/MSFT/Policy/Config/Update/Install/</LocURI></Target></SyncML>`
	profile1UUID := uuid.NewString()
	profile2 := `<SyncML><Target><LocURI>./Vendor/MSFT/Policy/Config/SomeOtherSetting/</LocURI></Target></SyncML>`
	profile2UUID := uuid.NewString()
	profile3UUID := uuid.NewString() // Fleet uploaded
	if _, err := db.Exec(`INSERT INTO mdm_windows_configuration_profiles (profile_uuid, name, syncml) VALUES (?, "profile-1", ?), (?, "profile-2", ?), (?, "Windows OS Updates", ?)`, profile1UUID, profile1, profile2UUID, profile2, profile3UUID, profile1); err != nil {
		t.Fatalf("insert windows profiles: %v", err)
	}

	// Apply current migration.
	applyNext(t, db)

	// Now we check that decl1 and profile1 made it into the table, but not decl2 and profile2.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE apple_declaration_uuid = ?`, decl1UUID).Scan(&count); err != nil {
		t.Fatalf("query for decl1: %v", err)
	}
	require.Equal(t, 1, count, "expected decl1 to be in the update settings table, but it was not")

	// decl2
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE apple_declaration_uuid = ?`, decl2UUID).Scan(&count); err != nil {
		t.Fatalf("query for decl2: %v", err)
	}
	require.Equal(t, 0, count, "expected decl2 to not be in the update settings table, but it was")

	// decl3
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE apple_declaration_uuid = ?`, decl3UUID).Scan(&count); err != nil {
		t.Fatalf("query for decl3: %v", err)
	}
	require.Equal(t, 0, count, "expected decl3 to not be in the update settings table, but it was")

	// decl4
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE apple_declaration_uuid = ?`, decl4UUID).Scan(&count); err != nil {
		t.Fatalf("query for decl4: %v", err)
	}
	require.Equal(t, 0, count, "expected decl4 to not be in the update settings table, but it was")

	// decl5
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE apple_declaration_uuid = ?`, decl5UUID).Scan(&count); err != nil {
		t.Fatalf("query for decl5: %v", err)
	}
	require.Equal(t, 0, count, "expected decl5 to not be in the update settings table, but it was")

	// profile1
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE windows_profile_uuid = ?`, profile1UUID).Scan(&count); err != nil {
		t.Fatalf("query for profile1: %v", err)
	}
	require.Equal(t, 1, count, "expected profile1 to be in the update settings table, but it was not")

	// profile2
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE windows_profile_uuid = ?`, profile2UUID).Scan(&count); err != nil {
		t.Fatalf("query for profile2: %v", err)
	}
	require.Equal(t, 0, count, "expected profile2 to not be in the update settings table, but it was")

	// profile3
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE windows_profile_uuid = ?`, profile3UUID).Scan(&count); err != nil {
		t.Fatalf("query for profile3: %v", err)
	}
	require.Equal(t, 0, count, "expected profile3 to not be in the update settings table, but it was")

	// We test the constraint
	if _, err := db.Exec(`INSERT INTO mdm_configuration_profile_update_settings (apple_declaration_uuid, windows_profile_uuid) VALUES (?, ?)`, decl1UUID, profile1UUID); err == nil {
		t.Fatalf("expected error when inserting a row with both apple_declaration_uuid and windows_profile_uuid set, but got no error")
	}

	// And we check the foreign key cascade effects
	if _, err := db.Exec(`DELETE FROM mdm_apple_declarations WHERE declaration_uuid = ?`, decl1UUID); err != nil {
		t.Fatalf("delete decl1: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE apple_declaration_uuid = ?`, decl1UUID).Scan(&count); err != nil {
		t.Fatalf("query for decl1 after deletion: %v", err)
	}
	require.Equal(t, 0, count, "expected decl1 to be deleted from the update settings table after its deletion from the declarations table, but it was not")

	if _, err := db.Exec(`DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, profile1UUID); err != nil {
		t.Fatalf("delete profile1: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_update_settings WHERE windows_profile_uuid = ?`, profile1UUID).Scan(&count); err != nil {
		t.Fatalf("query for profile1 after deletion: %v", err)
	}
	require.Equal(t, 0, count, "expected profile1 to be deleted from the update settings table after its deletion from the windows profiles table, but it was not")
}
