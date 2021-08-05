package mock

import "github.com/fleetdm/fleet/v4/server/fleet"

//go:generate mockimpl -o datastore_activities.go "s *ActivitiesStore" "fleet.ActivitiesStore"
//go:generate mockimpl -o datastore_appconfig.go "s *AppConfigStore" "fleet.AppConfigStore"
//go:generate mockimpl -o datastore_campaigns.go "s *CampaignStore" "fleet.CampaignStore"
//go:generate mockimpl -o datastore_carves.go "s *CarveStore" "fleet.CarveStore"
//go:generate mockimpl -o datastore_hosts.go "s *HostStore" "fleet.HostStore"
//go:generate mockimpl -o datastore_invites.go "s *InviteStore" "fleet.InviteStore"
//go:generate mockimpl -o datastore_labels.go "s *LabelStore" "fleet.LabelStore"
//go:generate mockimpl -o datastore_packs.go "s *PackStore" "fleet.PackStore"
//go:generate mockimpl -o datastore_queries.go "s *QueryStore" "fleet.QueryStore"
//go:generate mockimpl -o datastore_query_results.go "s *QueryResultStore" "fleet.QueryResultStore"
//go:generate mockimpl -o datastore_scheduled_queries.go "s *ScheduledQueryStore" "fleet.ScheduledQueryStore"
//go:generate mockimpl -o datastore_sessions.go "s *SessionStore" "fleet.SessionStore"
//go:generate mockimpl -o datastore_software.go "s *SoftwareStore" "fleet.SoftwareStore"
//go:generate mockimpl -o datastore_statistics.go "s *StatisticsStore" "fleet.StatisticsStore"
//go:generate mockimpl -o datastore_targets.go "s *TargetStore" "fleet.TargetStore"
//go:generate mockimpl -o datastore_teams.go "s *TeamStore" "fleet.TeamStore"
//go:generate mockimpl -o datastore_users.go "s *UserStore" "fleet.UserStore"

var _ fleet.Datastore = (*Store)(nil)

type Store struct {
	fleet.PasswordResetStore
	TeamStore
	TargetStore
	SessionStore
	CampaignStore
	ScheduledQueryStore
	AppConfigStore
	HostStore
	InviteStore
	LabelStore
	PackStore
	UserStore
	QueryStore
	QueryResultStore
	CarveStore
	SoftwareStore
	ActivitiesStore
	StatisticsStore
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
func (m *Store) MigrationStatus() (fleet.MigrationStatus, error) {
	return 0, nil
}
func (m *Store) Name() string {
	return "mock"
}

type mockTransaction struct{}

func (m *mockTransaction) Commit() error   { return nil }
func (m *mockTransaction) Rollback() error { return nil }

func (m *Store) Begin() (fleet.Transaction, error) {
	return &mockTransaction{}, nil
}
