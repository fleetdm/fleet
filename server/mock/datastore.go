package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

//go:generate go run ./mockimpl/impl.go -o datastore_mock.go "s *DataStore" "fleet.Datastore"
//go:generate go run ./mockimpl/impl.go -o datastore_installers.go "s *InstallerStore" "fleet.InstallerStore"
//go:generate go run ./mockimpl/impl.go -o nanodep/storage.go "s *Storage" "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage.AllDEPStorage"
//go:generate go run ./mockimpl/impl.go -o mdm/datastore_mdm_mock.go "fs *MDMAppleStore" "fleet.MDMAppleStore"
//go:generate go run ./mockimpl/impl.go -o scep/depot.go "d *Depot" "depot.Depot"
//go:generate go run ./mockimpl/impl.go -o mdm/bootstrap_package_store.go "s *MDMBootstrapPackageStore" "fleet.MDMBootstrapPackageStore"
//go:generate go run ./mockimpl/impl.go -o software/software_installer_store.go "s *SoftwareInstallerStore" "fleet.SoftwareInstallerStore"

var _ fleet.Datastore = (*Store)(nil)

type Store struct {
	DataStore
}

func (m *Store) EnrollOrbit(ctx context.Context, isMDMEnabled bool, orbitHostInfo fleet.OrbitHostInfo, orbitNodeKey string, teamID *uint) (*fleet.Host, error) {
	return nil, nil
}

func (m *Store) LoadHostByOrbitNodeKey(ctx context.Context, orbitNodeKey string) (*fleet.Host, error) {
	return nil, nil
}

func (m *Store) Drop() error                             { return nil }
func (m *Store) MigrateTables(ctx context.Context) error { return nil }
func (m *Store) MigrateData(ctx context.Context) error   { return nil }
func (m *Store) MigrationStatus(ctx context.Context) (*fleet.MigrationStatus, error) {
	return &fleet.MigrationStatus{}, nil
}
func (m *Store) Name() string { return "mock" }
