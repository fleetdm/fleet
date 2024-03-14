package tables

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20240205095928(t *testing.T) {
	db := applyUpToPrev(t)

	threeDayAgo := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Second)

	// create a Windows and an Apple profile
	idA, idW := "a"+uuid.New().String(), "w"+uuid.New().String()
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, name, identifier, mobileconfig, checksum, created_at, updated_at) VALUES (?, 0, 'A', 'A', '<plist></plist>', '0', ?, ?)`, idA, threeDayAgo, threeDayAgo)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, created_at, updated_at) VALUES (?, 0, 'W', '<Replace>W</Replace>', ?, ?)`, idW, threeDayAgo, threeDayAgo)

	// Apply current migration.
	applyNext(t, db)

	// updated_at is now called uploaded_at and its value has not changed
	var prof struct {
		ProfileUUID string     `db:"profile_uuid"`
		CreatedAt   time.Time  `db:"created_at"`
		UploadedAt  *time.Time `db:"uploaded_at"`
	}
	err := sqlx.Get(db, &prof, `SELECT profile_uuid, created_at, uploaded_at FROM mdm_apple_configuration_profiles WHERE profile_uuid = ?`, idA)
	require.NoError(t, err)
	require.Equal(t, idA, prof.ProfileUUID)
	require.NotNil(t, prof.UploadedAt)
	require.Equal(t, threeDayAgo, *prof.UploadedAt)
	require.Equal(t, threeDayAgo, prof.CreatedAt)

	err = sqlx.Get(db, &prof, `SELECT profile_uuid, created_at, uploaded_at FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, idW)
	require.NoError(t, err)
	require.Equal(t, idW, prof.ProfileUUID)
	require.NotNil(t, prof.UploadedAt)
	require.Equal(t, threeDayAgo, *prof.UploadedAt)
	require.Equal(t, threeDayAgo, prof.CreatedAt)

	secondsAgo := time.Now().UTC().Add(-2 * time.Second).Truncate(time.Second)

	// creating new profiles without an explicit uploaded_at results in
	// a NULL value (defaulting to current timestamp is removed)
	idA2, idW2 := "a"+uuid.New().String(), "w"+uuid.New().String()
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, name, identifier, mobileconfig, checksum) VALUES (?, 0, 'A2', 'A2', '<plist></plist>', '0')`, idA2)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, 0, 'W2', '<Replace>W2</Replace>')`, idW2)

	err = sqlx.Get(db, &prof, `SELECT profile_uuid, created_at, uploaded_at FROM mdm_apple_configuration_profiles WHERE profile_uuid = ?`, idA2)
	require.NoError(t, err)
	require.Equal(t, idA2, prof.ProfileUUID)
	require.Nil(t, prof.UploadedAt)
	require.True(t, prof.CreatedAt.After(secondsAgo))

	err = sqlx.Get(db, &prof, `SELECT profile_uuid, created_at, uploaded_at FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, idW2)
	require.NoError(t, err)
	require.Equal(t, idW2, prof.ProfileUUID)
	require.Nil(t, prof.UploadedAt)
	require.True(t, prof.CreatedAt.After(secondsAgo))
}
