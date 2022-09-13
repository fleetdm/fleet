package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

//go:generate mockimpl -o datastore_mock.go "s *DataStore" "fleet.Datastore"
//go:generate mockimpl -o datastore_installers.go "s *InstallerStore" "fleet.InstallerStore"

var _ fleet.Datastore = (*Store)(nil)

type Store struct {
	DataStore
}

func (m *Store) EnrollOrbit(ctx context.Context, hardwareUUID string, orbitNodeKey string) (*fleet.Host, error) {
	//TODO implement me
	return nil, nil
}

func (m *Store) Drop() error                             { return nil }
func (m *Store) MigrateTables(ctx context.Context) error { return nil }
func (m *Store) MigrateData(ctx context.Context) error   { return nil }
func (m *Store) MigrationStatus(ctx context.Context) (*fleet.MigrationStatus, error) {
	return &fleet.MigrationStatus{}, nil
}
func (m *Store) Name() string { return "mock" }
