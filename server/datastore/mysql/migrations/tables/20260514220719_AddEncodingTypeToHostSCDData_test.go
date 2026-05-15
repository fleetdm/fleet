package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260514220719(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a pre-migration row representing a dense host_bitmap. After the
	// migration this row must still be readable, with encoding_type defaulting
	// to 0 (dense).
	denseBytes := []byte{0x82, 0x05} // bits 1, 7, 8, 10 set: hosts {1, 7, 8, 10}
	validFrom := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, valid_from)
		VALUES (?, ?, ?, ?)`,
		"cve", "CVE-2026-0001", denseBytes, validFrom,
	)
	require.NoError(t, err)

	applyNext(t, db)

	// Pre-existing row reads back with encoding_type = 0 via DEFAULT and
	// unchanged host_bitmap bytes.
	var encoding int
	var bitmap []byte
	err = db.QueryRow(`
		SELECT encoding_type, host_bitmap FROM host_scd_data
		WHERE dataset = ? AND entity_id = ?`,
		"cve", "CVE-2026-0001",
	).Scan(&encoding, &bitmap)
	require.NoError(t, err)
	assert.Equal(t, 0, encoding, "legacy row should default to encoding_type=0 (dense)")
	assert.Equal(t, denseBytes, bitmap, "INSTANT ALTER must not rewrite row data")

	// New rows may be written with encoding_type = 1 (roaring).
	roaringBytes := []byte{0x3A, 0x30, 0x00, 0x00} // arbitrary stand-in; library serializes its own format
	_, err = db.Exec(`
		INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, encoding_type, valid_from)
		VALUES (?, ?, ?, ?, ?)`,
		"cve", "CVE-2026-0002", roaringBytes, 1, validFrom,
	)
	require.NoError(t, err)

	err = db.QueryRow(`
		SELECT encoding_type, host_bitmap FROM host_scd_data
		WHERE dataset = ? AND entity_id = ?`,
		"cve", "CVE-2026-0002",
	).Scan(&encoding, &bitmap)
	require.NoError(t, err)
	assert.Equal(t, 1, encoding)
	assert.Equal(t, roaringBytes, bitmap)
}
