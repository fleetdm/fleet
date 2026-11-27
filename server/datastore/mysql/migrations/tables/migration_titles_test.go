package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/goose"
	"github.com/stretchr/testify/assert"
)

func TestMigrationTitles(t *testing.T) {
	assert.NotEmpty(t, MigrationClient.Migrations)
	for _, m := range MigrationClient.Migrations {
		assert.NoError(t, goose.ValidateTitle(m.Title), "migration %d", m.Version)
	}
}
