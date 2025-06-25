package tables

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250625155654(t *testing.T) {
	db := applyUpToPrev(t)

	eulaBytes := []byte("test eula content")

	hash := sha256.New()
	_, _ = hash.Write(eulaBytes)
	sha256 := hash.Sum(nil)

	execNoErr(t, db,
		`INSERT INTO eulas (id, bytes, token, name) VALUES (?, ?, ?, ?)`,
		1, eulaBytes, "test-token", "test-name",
	)

	// Apply current migration.
	applyNext(t, db)

	var got []byte
	err := db.Get(&got, `SELECT sha256 FROM eulas WHERE id = ?`, 1)
	require.NoError(t, err)

	require.True(t, bytes.Equal(got, sha256))
}
