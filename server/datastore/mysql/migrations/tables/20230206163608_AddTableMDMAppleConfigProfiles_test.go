package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230206163608(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	stmt := `
INSERT INTO
    mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig)
VALUES (?, ?, ?, ?)`

	mcBytes := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>TestPayloadName</string>
	<key>PayloadIdentifier</key>
	<string>TestPayloadIdentifier</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>TestPayloadUUID</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`)

	_, err := db.Exec(stmt, 0, "TestPayloadIdentifier", "TestPayloadName", mcBytes)
	require.NoError(t, err)

	var (
		identifier   string
		mobileconfig []byte
	)
	err = db.QueryRow(`SELECT identifier, mobileconfig FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?`, "TestPayloadName", 0).Scan(&identifier, &mobileconfig)
	require.NoError(t, err)
	require.Equal(t, "TestPayloadIdentifier", identifier)
	require.Equal(t, mcBytes, mobileconfig)
}
