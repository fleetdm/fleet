package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

//go:generate mockimpl -o datastore_mock.go "s *DataStore" "fleet.Datastore"
//go:generate mockimpl -o datastore_query_results.go "s *QueryResultStore" "fleet.QueryResultStore"

var _ fleet.Datastore = (*Store)(nil)

type Store struct {
	DataStore
}

func (m *Store) Drop() error                             { return nil }
func (m *Store) MigrateTables(ctx context.Context) error { return nil }
func (m *Store) MigrateData(ctx context.Context) error   { return nil }
func (m *Store) MigrationStatus(ctx context.Context) (*fleet.MigrationStatus, error) {
	return &fleet.MigrationStatus{}, nil
}
func (m *Store) Name() string { return "mock" }
