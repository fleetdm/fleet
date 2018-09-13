package mock

//go:generate mockimpl -o datastore_users.go "s *UserStore" "kolide.UserStore"
//go:generate mockimpl -o datastore_invites.go "s *InviteStore" "kolide.InviteStore"
//go:generate mockimpl -o datastore_appconfig.go "s *AppConfigStore" "kolide.AppConfigStore"
//go:generate mockimpl -o datastore_labels.go "s *LabelStore" "kolide.LabelStore"
//go:generate mockimpl -o datastore_options.go "s *OptionStore" "kolide.OptionStore"
//go:generate mockimpl -o datastore_packs.go "s *PackStore" "kolide.PackStore"
//go:generate mockimpl -o datastore_hosts.go "s *HostStore" "kolide.HostStore"
//go:generate mockimpl -o datastore_fim.go "s *FileIntegrityMonitoringStore" "kolide.FileIntegrityMonitoringStore"
//go:generate mockimpl -o datastore_osquery_options.go "s *OsqueryOptionsStore" "kolide.OsqueryOptionsStore"
//go:generate mockimpl -o datastore_scheduled_queries.go "s *ScheduledQueryStore" "kolide.ScheduledQueryStore"
//go:generate mockimpl -o datastore_queries.go "s *QueryStore" "kolide.QueryStore"
//go:generate mockimpl -o datastore_campaigns.go "s *CampaignStore" "kolide.CampaignStore"
//go:generate mockimpl -o datastore_sessions.go "s *SessionStore" "kolide.SessionStore"

import "github.com/kolide/fleet/server/kolide"

var _ kolide.Datastore = (*Store)(nil)

type Store struct {
	kolide.PasswordResetStore
	kolide.YARAStore
	kolide.TargetStore
	SessionStore
	CampaignStore
	ScheduledQueryStore
	OsqueryOptionsStore
	FileIntegrityMonitoringStore
	AppConfigStore
	HostStore
	InviteStore
	LabelStore
	OptionStore
	PackStore
	UserStore
	QueryStore
}

func (m *Store) Drop() error {
	return nil
}
func (m *Store) MigrateTables() error {
	return nil
}
func (m *Store) MigrateData() error {
	return nil
}
func (m *Store) MigrationStatus() (kolide.MigrationStatus, error) {
	return 0, nil
}
func (m *Store) Name() string {
	return "mock"
}

type mockTransaction struct{}

func (m *mockTransaction) Commit() error   { return nil }
func (m *mockTransaction) Rollback() error { return nil }

func (m *Store) Begin() (kolide.Transaction, error) {
	return &mockTransaction{}, nil
}
