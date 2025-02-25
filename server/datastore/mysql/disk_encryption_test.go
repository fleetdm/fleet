package mysql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiskEncryption(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestCleanupDiskEncryptionKeysOnTeamChange", testCleanupDiskEncryptionKeysOnTeamChange},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testCleanupDiskEncryptionKeysOnTeamChange(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// No-op test
	assert.NoError(t, ds.CleanupDiskEncryptionKeysOnTeamChange(ctx, []uint{1, 2, 3}, nil))
}
