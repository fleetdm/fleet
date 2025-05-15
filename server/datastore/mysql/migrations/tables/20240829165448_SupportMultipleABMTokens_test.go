package tables

import (
	"crypto/md5" //nolint:gosec
	"database/sql"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20240829165448(t *testing.T) {
	createTokenAndHash := func() (string, []byte) {
		tok := uuid.NewString()
		h := md5.New() //nolint:gosec
		_, _ = h.Write([]byte(tok))
		md5Checksum := h.Sum(nil)
		return tok, md5Checksum
	}

	type abmToken struct {
		ID                  uint      `db:"id"`
		OrganizationName    string    `db:"organization_name"`
		AppleID             string    `db:"apple_id"`
		TermsExpired        bool      `db:"terms_expired"`
		RenewAt             time.Time `db:"renew_at"`
		Token               []byte    `db:"token"`
		MacOSDefaultTeamID  *uint     `db:"macos_default_team_id"`
		IOSDefaultTeamID    *uint     `db:"ios_default_team_id"`
		IPadOSDefaultTeamID *uint     `db:"ipados_default_team_id"`
	}

	t.Run("NoExistingToken", func(t *testing.T) {
		db := applyUpToPrev(t)

		// create a host with a DEP assignment (should not exist when there is no
		// ABM, but maybe ABM was setup then removed)
		hostID := insertHost(t, db, nil)
		execNoErr(t, db, `INSERT INTO host_dep_assignments (host_id) VALUES (?)`, hostID)

		// Apply current migration.
		applyNext(t, db)

		var exists int
		// no ABM token in the old storage
		err := db.Get(&exists, `SELECT 1 FROM mdm_config_assets WHERE name = 'abm_token'`)
		require.ErrorIs(t, err, sql.ErrNoRows)
		// no ABM token in the new storage
		err = db.Get(&exists, `SELECT 1 FROM abm_tokens`)
		require.ErrorIs(t, err, sql.ErrNoRows)

		// the existing host DEP assignment is still not linked to any token
		var hostTokenID *uint
		err = db.Get(&hostTokenID, `SELECT abm_token_id FROM host_dep_assignments WHERE host_id = ?`, hostID)
		require.NoError(t, err)
		require.Nil(t, hostTokenID)
	})

	t.Run("ExistingTokenWithTeamTermsFalse", func(t *testing.T) {
		db := applyUpToPrev(t)

		// create an existing ABM token
		existingToken, md5Checksum := createTokenAndHash()
		execNoErr(t, db, `INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES ('abm_token', ?, ?)`, existingToken, md5Checksum)

		// set a config for ABM
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm', JSON_OBJECT('apple_bm_default_team', 'team1'))`)

		// create the corresponding team
		tmID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES (?)`, "team1")

		// create a host with a DEP assignment
		hostID := insertHost(t, db, ptr.Uint(uint(tmID))) //nolint:gosec // dismiss G115
		execNoErr(t, db, `INSERT INTO host_dep_assignments (host_id) VALUES (?)`, hostID)

		// Apply current migration.
		applyNext(t, db)

		// ABM token is now soft-deleted in the old storage
		var assetDeletedUUID string
		err := db.Get(&assetDeletedUUID, `SELECT deletion_uuid FROM mdm_config_assets WHERE name = 'abm_token'`)
		require.NoError(t, err)
		require.NotEmpty(t, assetDeletedUUID)

		// ABM token is stored in the new storage, with the expected config
		var storedToken abmToken
		err = db.Get(&storedToken, `
SELECT
	id, organization_name, apple_id, terms_expired, renew_at, token, macos_default_team_id, ios_default_team_id, ipados_default_team_id
FROM
	abm_tokens
LIMIT 1`)
		require.NoError(t, err)

		// we don't have those fields during DB migration
		require.Empty(t, storedToken.OrganizationName)
		require.Empty(t, storedToken.AppleID)
		require.Equal(t, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), storedToken.RenewAt)
		// terms were not set as expired in appconfig
		require.False(t, storedToken.TermsExpired)
		// token matches
		require.Equal(t, existingToken, string(storedToken.Token))
		// all platform default teams are set to the configured team
		require.NotNil(t, storedToken.MacOSDefaultTeamID)
		require.EqualValues(t, tmID, *storedToken.MacOSDefaultTeamID)
		require.NotNil(t, storedToken.IOSDefaultTeamID)
		require.EqualValues(t, tmID, *storedToken.IOSDefaultTeamID)
		require.NotNil(t, storedToken.IPadOSDefaultTeamID)
		require.EqualValues(t, tmID, *storedToken.IPadOSDefaultTeamID)

		// the existing host DEP assignment is linked to the token
		var hostTokenID *uint
		err = db.Get(&hostTokenID, `SELECT abm_token_id FROM host_dep_assignments WHERE host_id = ?`, hostID)
		require.NoError(t, err)
		require.NotNil(t, hostTokenID)
		require.EqualValues(t, storedToken.ID, *hostTokenID)
	})

	t.Run("ExistingTokenWithInvalidTeamTermsTrue", func(t *testing.T) {
		db := applyUpToPrev(t)

		// create an existing ABM token
		existingToken, md5Checksum := createTokenAndHash()
		execNoErr(t, db, `INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES ('abm_token', ?, ?)`, existingToken, md5Checksum)

		// set a config for ABM
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm', JSON_OBJECT('apple_bm_default_team', 'no-such-team', 'apple_bm_terms_expired', true))`)

		// create a team, but not one matching the default team name
		tmID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES (?)`, "team1")

		// create a host with a DEP assignment
		hostID := insertHost(t, db, ptr.Uint(uint(tmID))) //nolint:gosec // dismiss G115
		execNoErr(t, db, `INSERT INTO host_dep_assignments (host_id) VALUES (?)`, hostID)

		// Apply current migration.
		applyNext(t, db)

		// ABM token is now soft-deleted in the old storage
		var assetDeletedUUID string
		err := db.Get(&assetDeletedUUID, `SELECT deletion_uuid FROM mdm_config_assets WHERE name = 'abm_token'`)
		require.NoError(t, err)
		require.NotEmpty(t, assetDeletedUUID)

		// ABM token is stored in the new storage, with the expected config
		var storedToken abmToken
		err = db.Get(&storedToken, `
SELECT
	id, organization_name, apple_id, terms_expired, renew_at, token, macos_default_team_id, ios_default_team_id, ipados_default_team_id
FROM
	abm_tokens
LIMIT 1`)
		require.NoError(t, err)

		// we don't have those fields during DB migration
		require.Empty(t, storedToken.OrganizationName)
		require.Empty(t, storedToken.AppleID)
		require.Equal(t, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), storedToken.RenewAt)
		// terms were set as expired in appconfig
		require.True(t, storedToken.TermsExpired)
		// token matches
		require.Equal(t, existingToken, string(storedToken.Token))
		// all platform default teams are set to nil as the team did not exist
		require.Nil(t, storedToken.MacOSDefaultTeamID)
		require.Nil(t, storedToken.IOSDefaultTeamID)
		require.Nil(t, storedToken.IPadOSDefaultTeamID)

		// the existing host DEP assignment is linked to the token
		var hostTokenID *uint
		err = db.Get(&hostTokenID, `SELECT abm_token_id FROM host_dep_assignments WHERE host_id = ?`, hostID)
		require.NoError(t, err)
		require.NotNil(t, hostTokenID)
		require.EqualValues(t, storedToken.ID, *hostTokenID)
	})

	t.Run("ExistingTokenNoMDMConfig", func(t *testing.T) {
		db := applyUpToPrev(t)

		// create an existing ABM token
		existingToken, md5Checksum := createTokenAndHash()
		execNoErr(t, db, `INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES ('abm_token', ?, ?)`, existingToken, md5Checksum)

		// app config does not have the MDM object
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_REMOVE(json_value, '$.mdm')`)

		// create a host with a DEP assignment
		hostID := insertHost(t, db, nil)
		execNoErr(t, db, `INSERT INTO host_dep_assignments (host_id) VALUES (?)`, hostID)

		// Apply current migration.
		applyNext(t, db)

		// ABM token is now soft-deleted in the old storage
		var assetDeletedUUID string
		err := db.Get(&assetDeletedUUID, `SELECT deletion_uuid FROM mdm_config_assets WHERE name = 'abm_token'`)
		require.NoError(t, err)
		require.NotEmpty(t, assetDeletedUUID)

		// ABM token is stored in the new storage, with the default config
		var storedToken abmToken
		err = db.Get(&storedToken, `
SELECT
	id, organization_name, apple_id, terms_expired, renew_at, token, macos_default_team_id, ios_default_team_id, ipados_default_team_id
FROM
	abm_tokens
LIMIT 1`)
		require.NoError(t, err)

		// we don't have those fields during DB migration
		require.Empty(t, storedToken.OrganizationName)
		require.Empty(t, storedToken.AppleID)
		require.Equal(t, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), storedToken.RenewAt)
		require.False(t, storedToken.TermsExpired)
		// token matches
		require.Equal(t, existingToken, string(storedToken.Token))
		// all platform default teams are set to nil
		require.Nil(t, storedToken.MacOSDefaultTeamID)
		require.Nil(t, storedToken.IOSDefaultTeamID)
		require.Nil(t, storedToken.IPadOSDefaultTeamID)

		// the existing host DEP assignment is linked to the token
		var hostTokenID *uint
		err = db.Get(&hostTokenID, `SELECT abm_token_id FROM host_dep_assignments WHERE host_id = ?`, hostID)
		require.NoError(t, err)
		require.NotNil(t, hostTokenID)
		require.EqualValues(t, storedToken.ID, *hostTokenID)
	})

	t.Run("ExistingTokenCorruptedJSONConfig", func(t *testing.T) {
		db := applyUpToPrev(t)

		// create an existing ABM token
		existingToken, md5Checksum := createTokenAndHash()
		execNoErr(t, db, `INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES ('abm_token', ?, ?)`, existingToken, md5Checksum)

		// set a corrupted JSON config for ABM
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm', JSON_OBJECT('apple_bm_default_team', 123, 'apple_bm_terms_expired', 'abc'))`)

		// Apply current migration.
		applyNext(t, db)

		// ABM token is now soft-deleted in the old storage
		var assetDeletedUUID string
		err := db.Get(&assetDeletedUUID, `SELECT deletion_uuid FROM mdm_config_assets WHERE name = 'abm_token'`)
		require.NoError(t, err)
		require.NotEmpty(t, assetDeletedUUID)

		// ABM token is stored in the new storage, with the default config (as the existing one was invalid)
		var storedToken abmToken
		err = db.Get(&storedToken, `
SELECT
	id, organization_name, apple_id, terms_expired, renew_at, token, macos_default_team_id, ios_default_team_id, ipados_default_team_id
FROM
	abm_tokens
LIMIT 1`)
		require.NoError(t, err)

		// we don't have those fields during DB migration
		require.Empty(t, storedToken.OrganizationName)
		require.Empty(t, storedToken.AppleID)
		require.Equal(t, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), storedToken.RenewAt)
		require.False(t, storedToken.TermsExpired)
		// token matches
		require.Equal(t, existingToken, string(storedToken.Token))
		// all platform default teams are set to nil
		require.Nil(t, storedToken.MacOSDefaultTeamID)
		require.Nil(t, storedToken.IOSDefaultTeamID)
		require.Nil(t, storedToken.IPadOSDefaultTeamID)
	})
}
