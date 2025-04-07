package tables

import (
	"crypto/md5" //nolint:gosec
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20240829165605(t *testing.T) {
	createTokenAndHash := func() (string, []byte) {
		tok := uuid.NewString()
		h := md5.New() //nolint:gosec
		_, _ = h.Write([]byte(tok))
		md5Checksum := h.Sum(nil)
		return tok, md5Checksum
	}

	type job struct {
		ID        uint             `json:"id" db:"id"`
		CreatedAt time.Time        `json:"created_at" db:"created_at"`
		UpdatedAt *time.Time       `json:"updated_at" db:"updated_at"`
		Name      string           `json:"name" db:"name"`
		Args      *json.RawMessage `json:"args" db:"args"`
		State     string           `json:"state" db:"state"`
		Retries   int              `json:"retries" db:"retries"`
		Error     string           `json:"error" db:"error"`
		NotBefore time.Time        `json:"not_before" db:"not_before"`
	}

	type vppToken struct {
		ID               uint      `db:"id"`
		OrganizationName string    `db:"organization_name"`
		Location         string    `db:"location"`
		RenewAt          time.Time `db:"renew_at"`
		Token            []byte    `db:"token"`
		TeamID           *uint     `db:"team_id"`
		NullTeamType     string    `db:"null_team_type"`
	}

	type jobArgs struct {
		Task string `json:"task"`
	}

	t.Run("NoExistingToken", func(t *testing.T) {
		db := applyUpToPrev(t)

		// create a vpp app
		adamID := "abcdEFGH"
		execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, ?)`, adamID, "darwin")
		execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform) VALUES (?, ?)`, adamID, "darwin")

		// create a host with a VPP install request
		hostID := insertHost(t, db, nil)
		execNoErr(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, command_uuid, platform) VALUES (?, ?, ?, ?)`, hostID, adamID, uuid.NewString(), "darwin")

		// there is no pending job of that type at the moment
		var jobs []*job
		err := db.Select(&jobs, `SELECT id, name, args, state, retries, error, not_before FROM jobs WHERE name = 'db_migration'`)
		require.NoError(t, err)
		require.Empty(t, jobs)

		// Apply current migration.
		applyNext(t, db)

		var exists int
		// no VPP token in the old storage
		err = db.Get(&exists, `SELECT 1 FROM mdm_config_assets WHERE name = 'vpp_token'`)
		require.ErrorIs(t, err, sql.ErrNoRows)
		// no VPP token in the new storage
		err = db.Get(&exists, `SELECT 1 FROM vpp_tokens`)
		require.ErrorIs(t, err, sql.ErrNoRows)

		// the existing host install request is not linked
		var hostTokenID *uint
		err = db.Get(&hostTokenID, `SELECT vpp_token_id FROM host_vpp_software_installs WHERE host_id = ?`, hostID)
		require.NoError(t, err)
		require.Nil(t, hostTokenID)

		// there is still no pending job of that type
		err = db.Select(&jobs, `SELECT id, name, args, state, retries, error, not_before FROM jobs WHERE name = 'db_migration'`)
		require.NoError(t, err)
		require.Empty(t, jobs)
	})

	t.Run("ExistingTokenAndInstallRequest", func(t *testing.T) {
		db := applyUpToPrev(t)

		// create an existing VPP token
		existingToken, md5Checksum := createTokenAndHash()
		execNoErr(t, db, `INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES ('vpp_token', ?, ?)`, existingToken, md5Checksum)

		// create a vpp app
		adamID := "abcdEFGH"
		execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, ?)`, adamID, "darwin")
		execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform) VALUES (?, ?)`, adamID, "darwin")

		// create a host with a VPP install request
		hostID := insertHost(t, db, nil)
		execNoErr(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, command_uuid, platform) VALUES (?, ?, ?, ?)`, hostID, adamID, uuid.NewString(), "darwin")

		// there is no pending job of that type at the moment
		var jobs []*job
		err := db.Select(&jobs, `SELECT id, name, args, state, retries, error, not_before FROM jobs WHERE name = 'db_migration'`)
		require.NoError(t, err)
		require.Empty(t, jobs)

		// Apply current migration.
		applyNext(t, db)

		// VPP token is now soft-deleted in the old storage
		var assetDeletedUUID string
		err = db.Get(&assetDeletedUUID, `SELECT deletion_uuid FROM mdm_config_assets WHERE name = 'vpp_token'`)
		require.NoError(t, err)
		require.NotEmpty(t, assetDeletedUUID)

		// VPP token is stored in the new storage, with the expected config
		var storedToken vppToken
		err = db.Get(&storedToken, `
SELECT
	id, organization_name, location, renew_at, token, team_id, null_team_type
FROM
	vpp_tokens
LIMIT 1`)
		require.NoError(t, err)

		// we don't have those fields during DB migration
		require.NotZero(t, storedToken.ID)
		require.Empty(t, storedToken.OrganizationName)
		require.Empty(t, storedToken.Location)
		require.Equal(t, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), storedToken.RenewAt)
		// token matches
		require.Equal(t, existingToken, string(storedToken.Token))
		require.Nil(t, storedToken.TeamID)
		require.Equal(t, "allteams", storedToken.NullTeamType)

		// the existing host install request is linked to the token
		var hostTokenID *uint
		err = db.Get(&hostTokenID, `SELECT vpp_token_id FROM host_vpp_software_installs WHERE host_id = ?`, hostID)
		require.NoError(t, err)
		require.NotNil(t, hostTokenID)
		require.EqualValues(t, storedToken.ID, *hostTokenID)

		// the job was enqueued to finish migrating the token
		err = db.Select(&jobs, `SELECT id, name, args, state, retries, error, not_before FROM jobs WHERE name = 'db_migration'`)
		require.NoError(t, err)
		require.Len(t, jobs, 1)

		require.Equal(t, "db_migration", jobs[0].Name)
		require.Equal(t, 0, jobs[0].Retries)
		require.LessOrEqual(t, jobs[0].NotBefore, time.Now().UTC())
		require.NotNil(t, jobs[0].Args)

		var args jobArgs
		err = json.Unmarshal(*jobs[0].Args, &args)
		require.NoError(t, err)
		require.Equal(t, "migrate_vpp_token", args.Task)
	})
}
