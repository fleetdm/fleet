package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260527215818(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	execNoErr(t, db,
		`INSERT INTO org_logo (mode, data) VALUES (?, ?)`,
		"light", []byte{0x89, 0x50, 0x4E, 0x47},
	)

	// Verify mode is the primary key (duplicate mode rejected).
	_, err := db.Exec(
		`INSERT INTO org_logo (mode, data) VALUES (?, ?)`,
		"light", []byte{0xFF, 0xD8, 0xFF},
	)
	require.Error(t, err)

	// Verify we can read the row back and uploaded_at was defaulted.
	var got struct {
		Mode       string    `db:"mode"`
		Data       []byte    `db:"data"`
		UploadedAt time.Time `db:"uploaded_at"`
	}
	require.NoError(t, db.Get(&got,
		`SELECT mode, data, uploaded_at FROM org_logo WHERE mode = ?`, "light"))
	require.Equal(t, "light", got.Mode)
	require.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, got.Data)
	require.WithinDuration(t, time.Now(), got.UploadedAt, time.Minute)
}
