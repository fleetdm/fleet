//go:build !darwin

package useraction

import "time"

func NewMDMMigrator(path string, frequency time.Duration, handler MDMMigratorHandler) MDMMigrator {
	return &NoopMDMMigrator{}
}

type NoopMDMMigrator struct{}

func (m *NoopMDMMigrator) CanRun() bool                       { return false }
func (m *NoopMDMMigrator) SetProps(MDMMigratorProps)          {}
func (m *NoopMDMMigrator) Show() error                        { return nil }
func (m *NoopMDMMigrator) ShowInterval() error                { return nil }
func (m *NoopMDMMigrator) Exit()                              {}
func (m *NoopMDMMigrator) MigrationInProgress() (bool, error) { return false, nil }
func (m *NoopMDMMigrator) MarkMigrationCompleted() error      { return nil }
