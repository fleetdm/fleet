package tables

import "testing"

func TestUp_20260817110708(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a single policy, and assert new columns are null and row still exists.
	_, err := db.Exec(`INSERT INTO policies (name, description, query, resolution, platforms, checksum) VALUES (?, ?, ?, ?, ?, ?)`,
		"test policy", "test description", "fake-query", "test resolution", "darwin,windows", "fake")
	if err != nil {
		t.Fatalf("failed to insert policy: %v", err)
	}

	// Apply current migration.
	applyNext(t, db)

	// Assert on first row that new columns are null and row still exists.
	var resendAppleProfileUUID, resendWindowsProfileUUID *string
	err = db.QueryRow(`SELECT resend_apple_profile_uuid, resend_windows_profile_uuid FROM policies WHERE name = ?`, "test policy").
		Scan(&resendAppleProfileUUID, &resendWindowsProfileUUID)
	if err != nil {
		t.Fatalf("failed to query policy: %v", err)
	}
	if resendAppleProfileUUID != nil {
		t.Errorf("expected resend_apple_profile_uuid to be null, got %v", *resendAppleProfileUUID)
	}
	if resendWindowsProfileUUID != nil {
		t.Errorf("expected resend_windows_profile_uuid to be null, got %v", *resendWindowsProfileUUID)
	}

	// Insert a row with a FK failure for Apple
	_, err = db.Exec(`INSERT INTO policies (name, description, query, resolution, platforms, checksum, resend_apple_profile_uuid) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"test policy 2", "test description 2", "fake-query-2", "test resolution 2", "darwin", "fake-2", "non-existent-uuid")
	if err == nil {
		t.Fatalf("expected FK constraint violation for resend_apple_profile_uuid, but insert succeeded")
	}

	// Insert a row with a FK failure for Windows
	_, err = db.Exec(`INSERT INTO policies (name, description, query, resolution, platforms, checksum, resend_windows_profile_uuid) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"test policy 3", "test description 3", "fake-query-3", "test resolution 3", "windows", "fake-3", "non-existent-uuid")
	if err == nil {
		t.Fatalf("expected FK constraint violation for resend_windows_profile_uuid, but insert succeeded")
	}

	// Add valid rows for Apple and Windows to ensure they can be inserted correctly.
	// First, insert valid profiles into the respective tables.
	_, err = db.Exec(`INSERT INTO mdm_apple_configuration_profiles (profile_uuid, name, identifier, mobileconfig, checksum) VALUES (?, ?, ?, ?, ?)`, "valid-apple-uuid", "Valid Apple Profile", "com.example.profile", "fake-mobileconfig", "fake-checksum")
	if err != nil {
		t.Fatalf("failed to insert valid Apple profile: %v", err)
	}
	_, err = db.Exec(`INSERT INTO mdm_windows_configuration_profiles (profile_uuid, name, syncml) VALUES (?, ?, ?)`, "valid-windows-uuid", "Valid Windows Profile", "fake-syncml")
	if err != nil {
		t.Fatalf("failed to insert valid Windows profile: %v", err)
	}

	// Insert a row with both columns set (should fail due to check constraint)
	_, err = db.Exec(`INSERT INTO policies (name, description, query, resolution, platforms, checksum, resend_apple_profile_uuid, resend_windows_profile_uuid) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"test policy 4", "test description 4", "fake-query-4", "test resolution 4", "darwin,windows", "fake-4", "valid-apple-uuid", "valid-windows-uuid")
	if err == nil {
		t.Fatalf("expected check constraint violation for both resend_apple_profile_uuid and resend_windows_profile_uuid being set, but insert succeeded")
	}

	// Insert with valid apple config profile
	_, err = db.Exec(`INSERT INTO policies (name, description, query, resolution, platforms, checksum, resend_apple_profile_uuid) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"test policy 2", "test description 2", "fake-query-2", "test resolution 2", "darwin", "fake-2", "valid-apple-uuid")
	if err != nil {
		t.Fatalf("failed to insert policy with valid Apple profile: %v", err)
	}

	// Insert with valid windows config profile
	_, err = db.Exec(`INSERT INTO policies (name, description, query, resolution, platforms, checksum, resend_windows_profile_uuid) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"test policy 3", "test description 3", "fake-query-3", "test resolution 3", "windows", "fake-3", "valid-windows-uuid")
	if err != nil {
		t.Fatalf("failed to insert policy with valid Windows profile: %v", err)
	}
}
