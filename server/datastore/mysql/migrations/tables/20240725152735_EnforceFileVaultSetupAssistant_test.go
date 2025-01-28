package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

func TestUp_20240725152735(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `
INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum, profile_uuid)
VALUES (?, ?, ?, ?, UNHEX(MD5(mobileconfig)), UUID())
	`

	profileBytes := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>Defer</key>
			<true/>
			<key>Enable</key>
			<string>On</string>
			<key>PayloadDisplayName</key>
			<string>FileVault 2</string>
			<key>PayloadIdentifier</key>
			<string>com.apple.MCX.FileVault2.3548D750-6357-4910-8DEA-D80ADCE2C787</string>
			<key>PayloadType</key>
			<string>com.apple.MCX.FileVault2</string>
			<key>PayloadUUID</key>
			<string>3548D750-6357-4910-8DEA-D80ADCE2C787</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>ShowRecoveryKey</key>
			<false/>
			<key>DeferForceAtUserLoginMaxBypassAttempts</key>
			<integer>1</integer>
		</dict>
		<dict>
			<key>EncryptCertPayloadUUID</key>
			<string>A326B71F-EB80-41A5-A8CD-A6F932544281</string>
			<key>Location</key>
			<string>Fleet</string>
			<key>PayloadDisplayName</key>
			<string>FileVault Recovery Key Escrow</string>
			<key>PayloadIdentifier</key>
			<string>com.apple.security.FDERecoveryKeyEscrow.3690D771-DCB8-4D5D-97D6-209A138DF03E</string>
			<key>PayloadType</key>
			<string>com.apple.security.FDERecoveryKeyEscrow</string>
			<key>PayloadUUID</key>
			<string>3C329F2B-3D47-4141-A2B5-5C52A2FD74F8</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
		<dict>
			<key>PayloadCertificateFileName</key>
			<string>Fleet certificate</string>
			<key>PayloadContent</key>
			<data>dGVzdAo=</data>
			<key>PayloadDisplayName</key>
			<string>Certificate Root</string>
			<key>PayloadIdentifier</key>
			<string>com.apple.security.root.A326B71F-EB80-41A5-A8CD-A6F932544281</string>
			<key>PayloadType</key>
			<string>com.apple.security.pkcs1</string>
			<key>PayloadUUID</key>
			<string>A326B71F-EB80-41A5-A8CD-A6F932544281</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
		<dict>
			<key>dontAllowFDEDisable</key>
			<true/>
			<key>PayloadIdentifier</key>
			<string>com.apple.MCX.62024f29-105E-497A-A724-1D5BA4D9E854</string>
			<key>PayloadType</key>
			<string>com.apple.MCX</string>
			<key>PayloadUUID</key>
			<string>62024f29-105E-497A-A724-1D5BA4D9E854</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>Disk encryption</string>
	<key>PayloadIdentifier</key>
	<string>com.fleetdm.fleet.mdm.filevault</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>74FEAC88-B614-468E-A4B4-B4B0C93B5D52</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`)

	// add a global FV profile
	r, err := db.Exec(insertStmt, 0, mobileconfig.FleetFileVaultPayloadIdentifier, "Disk encryption", profileBytes)
	require.NoError(t, err)
	globalProfileID, _ := r.LastInsertId()

	// create a team
	r, err = db.Exec(`INSERT INTO teams (name) VALUES (?)`, "Test Team")
	require.NoError(t, err)
	teamID, _ := r.LastInsertId()

	// add the FV profile to the team
	r, err = db.Exec(insertStmt, teamID, mobileconfig.FleetFileVaultPayloadIdentifier, "Disk encryption", profileBytes)
	require.NoError(t, err)
	teamProfileID, _ := r.LastInsertId()

	var (
		identifier string
		gotConfig  []byte
	)

	checkProfsStmt := "SELECT identifier, mobileconfig FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?"
	err = db.QueryRow(checkProfsStmt, "Disk encryption", 0).Scan(&identifier, &gotConfig)
	require.NoError(t, err)
	require.Equal(t, mobileconfig.FleetFileVaultPayloadIdentifier, identifier)
	require.Equal(t, profileBytes, gotConfig)

	err = db.QueryRow(checkProfsStmt, "Disk encryption", teamID).Scan(&identifier, &gotConfig)
	require.NoError(t, err)
	require.Equal(t, mobileconfig.FleetFileVaultPayloadIdentifier, identifier)
	require.Equal(t, profileBytes, gotConfig)

	// Apply current migration.
	applyNext(t, db)

	verifyNewPayload := func(profileID int64) {
		var mc []byte
		stmt := "SELECT mobileconfig FROM mdm_apple_configuration_profiles WHERE profile_id = ?"
		err = db.QueryRow(stmt, profileID).Scan(&mc)
		require.NoError(t, err)

		// unmarshal only the fields we want to test
		var payload struct {
			PayloadContent []map[string]interface{}
		}
		_, err = plist.Unmarshal(mc, &payload)
		require.NoError(t, err)
		require.Len(t, payload.PayloadContent, 4)

		// find the right payload
		var found map[string]interface{}
		for _, p := range payload.PayloadContent {
			if p["PayloadType"] == "com.apple.MCX.FileVault2" {
				found = p
				break
			}
		}

		require.NotNil(t, found)
		require.EqualValues(t, true, found["ForceEnableInSetupAssistant"])
	}

	verifyNewPayload(globalProfileID)
	verifyNewPayload(teamProfileID)
}
