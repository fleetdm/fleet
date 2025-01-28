//go:build !darwin

package useraction

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/migration"
	"github.com/fleetdm/fleet/v4/server/service"
)

func NewMDMMigrator(path string, frequency time.Duration, handler MDMMigratorHandler, mrw *migration.ReadWriter, fleetURL string, showCh chan struct{}) MDMMigrator {
	return &NoopMDMMigrator{}
}

func StartMDMMigrationOfflineWatcher(ctx context.Context, client *service.DeviceClient, swiftDialogPath string, swiftDialogCh chan struct{}, fileWatcher migration.FileWatcher) MDMOfflineWatcher {
	return &NoopOfflineWatcher{}
}

type NoopOfflineWatcher struct{}

func (o *NoopOfflineWatcher) ShowIfOffline(ctx context.Context) bool { return false }

type NoopMDMMigrator struct{}

func (m *NoopMDMMigrator) CanRun() bool                         { return false }
func (m *NoopMDMMigrator) SetProps(MDMMigratorProps)            {}
func (m *NoopMDMMigrator) Show() error                          { return nil }
func (m *NoopMDMMigrator) ShowInterval() error                  { return nil }
func (m *NoopMDMMigrator) Exit()                                {}
func (m *NoopMDMMigrator) MigrationInProgress() (string, error) { return "", nil }
func (m *NoopMDMMigrator) MarkMigrationCompleted() error        { return nil }
