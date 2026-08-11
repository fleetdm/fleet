package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260811124908(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	var label struct {
		Description         string `db:"description"`
		Platform            string `db:"platform"`
		LabelType           uint   `db:"label_type"`
		LabelMembershipType uint   `db:"label_membership_type"`
	}
	require.NoError(t, db.Get(&label, `
		SELECT description, platform, label_type, label_membership_type
		FROM labels WHERE name = 'tvOS'`))

	require.Equal(t, "All tvOS hosts", label.Description)
	// Builtin labels must carry no platform; see Up_20251015103800.
	require.Empty(t, label.Platform)
	require.EqualValues(t, 1, label.LabelType)           // builtin
	require.EqualValues(t, 1, label.LabelMembershipType) // manual
}
