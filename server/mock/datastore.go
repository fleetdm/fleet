package mock

//go:generate mockimpl -o datastore_users.go "s *UserStore" "kolide.UserStore"
//go:generate mockimpl -o datastore_invites.go "s *InviteStore" "kolide.InviteStore"
//go:generate mockimpl -o datastore_appconfig.go "s *AppConfigStore" "kolide.AppConfigStore"
//go:generate mockimpl -o datastore_licenses.go "s *LicenseStore" "kolide.LicenseStore"
//go:generate mockimpl -o datastore_labels.go "s *LabelStore" "kolide.LabelStore"

import "github.com/kolide/kolide/server/kolide"

var _ kolide.Datastore = (*Store)(nil)

type Store struct {
	kolide.HostStore
	kolide.PackStore
	kolide.CampaignStore
	kolide.SessionStore
	kolide.PasswordResetStore
	kolide.QueryStore
	kolide.OptionStore
	kolide.ScheduledQueryStore
	kolide.DecoratorStore
	kolide.FileIntegrityMonitoringStore
	kolide.YARAStore
	LicenseStore
	InviteStore
	UserStore
	AppConfigStore
	LabelStore
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
