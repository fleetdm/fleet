package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20240703154849(t *testing.T) {
	db := applyUpToPrev(t)

	// create an MDM profile and an MDM declaration
	profStmt := `
INSERT INTO
    mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, profile_uuid, checksum)
VALUES (?, ?, ?, ?, ?, ?)`

	profUUID := uuid.NewString()
	mcBytes := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
</dict>
</plist>
`)

	_, err := db.Exec(profStmt, 0, "TestPayloadIdentifier", "TestPayloadName", mcBytes, profUUID, "ABCD")
	require.NoError(t, err)

	declStmt := `
INSERT INTO
	mdm_apple_declarations (declaration_uuid, team_id, identifier, name, raw_json, checksum)
VALUES (?, ?, ?, ?, ?, ?)`

	declUUID := uuid.NewString()
	_, err = db.Exec(declStmt, declUUID, 0, "TestDecl", "TestDecl", `{}`, "abcd")
	require.NoError(t, err)

	// create a couple labels
	idlA := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LA', 'select 1')`)
	idlB := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LB', 'select 1')`)

	// finally we can create the MDM profile label and MDM declaration label entries
	profLblID := execNoErrLastID(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`,
		profUUID, "LA", idlA)
	declLblID := execNoErrLastID(t, db, `INSERT INTO mdm_declaration_labels (apple_declaration_uuid, label_name, label_id) VALUES (?, ?, ?)`,
		declUUID, "LB", idlB)

	// Apply current migration.
	applyNext(t, db)

	// check that the "exclude" flag is false in the DB (set it to true to verify
	// that it did scan from the DB)
	exclude := true
	err = db.Get(&exclude, "SELECT exclude FROM mdm_configuration_profile_labels WHERE id = ?", profLblID)
	require.NoError(t, err)
	require.False(t, exclude)

	exclude = true
	err = db.Get(&exclude, "SELECT exclude FROM mdm_declaration_labels WHERE id = ?", declLblID)
	require.NoError(t, err)
	require.False(t, exclude)
}
