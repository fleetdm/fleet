package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20260702013100(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert two abm_tokens
	if _, err := db.Exec(`INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token) VALUES ('token1', 'apple1', NOW(), 'token1'), ('token2', 'apple2', NOW(), 'token2')`); err != nil {
		t.Fatalf("inserting abm_tokens: %v", err)
	}

	// Apply current migration.
	applyNext(t, db)

	// Verify the new columns exist and are populated with expected values.
	rows, err := db.Query(`SELECT id, byod_default_team_id, enrollment_url_token FROM abm_tokens`)
	if err != nil {
		t.Fatalf("selecting from abm_tokens: %v", err)
	}
	defer rows.Close()

	type abmToken struct {
		ID                 uint
		ByodDefaultTeamID  *uint
		EnrollmentURLToken []byte
	}
	var tokens []abmToken
	for rows.Next() {
		var token abmToken
		if err := rows.Scan(&token.ID, &token.ByodDefaultTeamID, &token.EnrollmentURLToken); err != nil {
			t.Fatalf("scanning abm_tokens: %v", err)
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating abm_tokens: %v", err)
	}

	if len(tokens) != 2 {
		t.Fatalf("expected 2 abm_tokens, got %d", len(tokens))
	}
	var abmTokenID uint
	var uniqueTestValue []byte
	for _, token := range tokens {
		if token.ByodDefaultTeamID != nil {
			t.Errorf("expected byod_default_team_id to be NULL, got %d", *token.ByodDefaultTeamID)
		}
		if len(token.EnrollmentURLToken) == 0 {
			t.Error("expected enrollment_url_token to be populated, got empty string")
		}
		if uniqueTestValue == nil {
			uniqueTestValue = token.EnrollmentURLToken
		}
		if abmTokenID == 0 {
			abmTokenID = token.ID
		}
	}

	// Try to insert a row without enrollment_url_token, which should fail due to the NOT NULL constraint.
	if _, err := db.Exec(`INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token) VALUES ('token3', 'apple3', NOW(), 'token3')`); err == nil {
		t.Error("expected error when inserting abm_token without enrollment_url_token, got none")
	}

	// Try to insert an empty enrollment URL token, which should fail due to the CHECK constraint.
	if _, err := db.Exec(`INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token, enrollment_url_token) VALUES ('token3', 'apple3', NOW(), 'token3', '')`); err == nil {
		t.Error("expected error when inserting empty enrollment_url_token, got none")
	}

	// Try to insert a duplicate enrollment URL token, which should fail due to the UNIQUE constraint.
	if _, err := db.Exec(`INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token, enrollment_url_token) VALUES ('token4', 'apple4', NOW(), 'token4', ?)`, uniqueTestValue); err == nil {
		t.Error("expected error when inserting duplicate enrollment_url_token, got none")
	}

	// Try to insert default_team_id that doesn't exist, which should fail due to the FOREIGN KEY constraint.
	badTok, err := fleet.GenerateRandom32ByteEntropyURLSafeToken()
	require.NoError(t, err)
	if _, err := db.Exec(`INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token, enrollment_url_token, byod_default_team_id) VALUES ('token5', 'apple5', NOW(), 'token5', ?, 9999)`, badTok); err == nil {
		t.Error("expected error when inserting non-existent byod_default_team_id, got none")
	}

	// Insert a valid team and then insert an abm_token with that team as the byod_default_team_id, which should succeed.
	res, err := db.Exec(`INSERT INTO teams (name) VALUES ('Test Team')`)
	if err != nil {
		t.Fatalf("inserting team: %v", err)
	}
	teamID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("getting last insert id for team: %v", err)
	}
	token, err := fleet.GenerateRandom32ByteEntropyURLSafeToken()
	require.NoError(t, err)
	if _, err := db.Exec(`INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token, byod_default_team_id, enrollment_url_token) VALUES ('token6', 'apple6', NOW(), 'token6', ?, ?)`, teamID, token); err != nil {
		t.Fatalf("inserting abm_token with valid byod_default_team_id: %v", err)
	}

	// Insert a row into mdm_adue_enrollment_challenges with missing challenge
	if _, err := db.Exec(`INSERT INTO mdm_adue_enrollment_challenges (idp_account_uuid, abm_token_id, expires_at) VALUES (?, ?, DATE_ADD(NOW(), INTERVAL 1 HOUR))`, "uuid1", abmTokenID); err == nil {
		t.Error("expected error when inserting mdm_adue_enrollment_challenges without challenge, got none")
	}

	// Insert a row into mdm_adue_enrollment_challenges with missing idp account UUID
	if _, err := db.Exec(`INSERT INTO mdm_adue_enrollment_challenges (challenge, abm_token_id, expires_at) VALUES ('challenge1', ?, DATE_ADD(NOW(), INTERVAL 1 HOUR))`, abmTokenID); err == nil {
		t.Error("expected error when inserting mdm_adue_enrollment_challenges without idp_account_uuid, got none")
	}

	// Insert a row into mdm_adue_enrollment_challenges with missing expires_at
	if _, err := db.Exec(`INSERT INTO mdm_adue_enrollment_challenges (challenge, idp_account_uuid, abm_token_id) VALUES ('challenge1', 'uuid1', ?)`, abmTokenID); err == nil {
		t.Error("expected error when inserting mdm_adue_enrollment_challenges without expires_at, got none")
	}

	// Insert a row with bad idp_account_uuid reference
	if _, err := db.Exec(`INSERT INTO mdm_adue_enrollment_challenges (challenge, idp_account_uuid, abm_token_id, expires_at) VALUES ('challenge1', 'invalid-uuid', ?, DATE_ADD(NOW(), INTERVAL 1 HOUR))`, abmTokenID); err == nil {
		t.Error("expected error when inserting mdm_adue_enrollment_challenges with invalid idp_account_uuid, got none")
	}

	// Insert mdm_idp_accounts row
	_, err = db.Exec(`INSERT INTO mdm_idp_accounts (uuid, username) VALUES ('uuid1', 'user1')`)
	if err != nil {
		t.Fatalf("inserting mdm_idp_accounts row: %v", err)
	}

	// Insert a row with bad abm_tokens reference
	if _, err := db.Exec(`INSERT INTO mdm_adue_enrollment_challenges (challenge, idp_account_uuid, abm_token_id, expires_at) VALUES ('challenge1', 'uuid1', 9999, DATE_ADD(NOW(), INTERVAL 1 HOUR))`); err == nil {
		t.Error("expected error when inserting mdm_adue_enrollment_challenges with invalid abm_token_id, got none")
	}

	// Insert a valid row into mdm_adue_enrollment_challenges, which should succeed.
	if _, err := db.Exec(`INSERT INTO mdm_adue_enrollment_challenges (challenge, idp_account_uuid, abm_token_id, expires_at) VALUES ('challenge1', 'uuid1', ?, DATE_ADD(NOW(), INTERVAL 1 HOUR))`, abmTokenID); err != nil {
		t.Fatalf("inserting valid row into mdm_adue_enrollment_challenges: %v", err)
	}

	// Try a new row with the same challenge to fail uniqueness
	if _, err := db.Exec(`INSERT INTO mdm_adue_enrollment_challenges (challenge, idp_account_uuid, abm_token_id, expires_at) VALUES ('challenge1', 'uuid1', ?, DATE_ADD(NOW(), INTERVAL 1 HOUR))`, abmTokenID); err == nil {
		t.Error("expected error when inserting duplicate challenge into mdm_adue_enrollment_challenges, got none")
	}

	// Delete mdm_idp_accounts and check FK delete cascade
	if _, err := db.Exec(`DELETE FROM mdm_idp_accounts WHERE uuid = 'uuid1'`); err != nil {
		t.Fatalf("deleting from mdm_idp_accounts: %v", err)
	}
	row := db.QueryRow(`SELECT COUNT(*) FROM mdm_adue_enrollment_challenges WHERE idp_account_uuid = 'uuid1'`)
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("counting mdm_adue_enrollment_challenges after deleting mdm_idp_accounts: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 mdm_adue_enrollment_challenges after deleting mdm_idp_accounts, got %d", count)
	}

	// Insert the idp account again
	if _, err = db.Exec(`INSERT INTO mdm_idp_accounts (uuid, username) VALUES ('uuid1', 'user1')`); err != nil {
		t.Fatalf("inserting mdm_idp_accounts row: %v", err)
	}

	// Insert the row again and delete abm_tokens and check FK delete cascade
	if _, err := db.Exec(`INSERT INTO mdm_adue_enrollment_challenges (challenge, idp_account_uuid, abm_token_id, expires_at) VALUES ('challenge1', 'uuid1', ?, DATE_ADD(NOW(), INTERVAL 1 HOUR))`, abmTokenID); err != nil {
		t.Fatalf("inserting valid row into mdm_adue_enrollment_challenges: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM abm_tokens WHERE id = ?`, abmTokenID); err != nil {
		t.Fatalf("deleting from abm_tokens: %v", err)
	}
	row = db.QueryRow(`SELECT COUNT(*) FROM mdm_adue_enrollment_challenges WHERE abm_token_id = ?`, abmTokenID)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("counting mdm_adue_enrollment_challenges after deleting abm_tokens: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 mdm_adue_enrollment_challenges after deleting abm_tokens, got %d", count)
	}

	// Insert the abm_token again and check that byod_default_team_id is set to NULL on team deletion
	token, err = fleet.GenerateRandom32ByteEntropyURLSafeToken()
	require.NoError(t, err)
	res, err = db.Exec(`INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token, byod_default_team_id, enrollment_url_token) VALUES ('token7', 'apple7', NOW(), 'token7', ?, ?)`, teamID, token)
	if err != nil {
		t.Fatalf("inserting abm_token with valid byod_default_team_id: %v", err)
	}
	insertedABMTokenID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("getting last insert id for abm_token: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM teams WHERE id = ?`, teamID); err != nil {
		t.Fatalf("deleting from teams: %v", err)
	}
	row = db.QueryRow(`SELECT byod_default_team_id FROM abm_tokens WHERE id = ?`, insertedABMTokenID)
	var byodDefaultTeamID *uint
	if err := row.Scan(&byodDefaultTeamID); err != nil {
		t.Fatalf("scanning byod_default_team_id after deleting team: %v", err)
	}
	if byodDefaultTeamID != nil {
		t.Errorf("expected byod_default_team_id to be NULL after deleting team, got %d", *byodDefaultTeamID)
	}
}
