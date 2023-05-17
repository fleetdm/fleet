//go:build !darwin

package useraction

import "time"

func NewMDMMigrator(path string, frequency time.Duration, handler MDMMigratorHandler) MDMMigrator {
	return &NoopMDMMigrator{}
}

type NoopMDMMigrator struct{}

func (m *NoopMDMMigrator) CanRun() bool { return false }
func (m *NoopMDMMigrator) Start()       { return nil }
func (m *NoopMDMMigrator) Complete()    { return nil }
func (m *NoopMDMMigrator) Exit()        {}
