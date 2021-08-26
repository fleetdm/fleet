package mock

import "github.com/fleetdm/fleet/v4/server/fleet"

//go:generate mockimpl -o datastore_mock.go "s *DataStore" "fleet.Datastore"
//go:generate mockimpl -o datastore_query_results.go "s *QueryResultStore" "fleet.QueryResultStore"

var _ fleet.Datastore = (*Store)(nil)

type Store struct {
	DataStore
}

func (m *Store) Drop() error                                     { return nil }
func (m *Store) MigrateTables() error                            { return nil }
func (m *Store) MigrateData() error                              { return nil }
func (m *Store) MigrationStatus() (fleet.MigrationStatus, error) { return 0, nil }
func (m *Store) Name() string                                    { return "mock" }

type mockTransaction struct{}

func (m *mockTransaction) Commit() error   { return nil }
func (m *mockTransaction) Rollback() error { return nil }

func (m *Store) Begin() (fleet.Transaction, error) { return &mockTransaction{}, nil }
