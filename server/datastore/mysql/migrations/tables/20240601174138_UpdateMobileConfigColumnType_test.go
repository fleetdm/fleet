package tables

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240601174138(t *testing.T) {
	db := applyUpToPrev(t)

	// Create a 1mb long string. This will fail at first, but will work with new column type.
	var b strings.Builder
	b.Grow(1000000)
	for i := 0; i < 1000000; i++ {
		b.WriteByte('a')
	}
	s := b.String()

	stmt := `INSERT INTO mdm_apple_configuration_profiles (profile_id, team_id, identifier, name, checksum, mobileconfig) VALUES (?,?,?,?,?,?)`

	_, err := db.Exec(stmt, 1, 0, "foo", "foo", "foo", s)
	require.ErrorContains(t, err, "Data too long")

	applyNext(t, db)

	_, err = db.Exec(stmt, 1, 0, "foo", "foo", "foo", s)
	require.NoError(t, err)
}
