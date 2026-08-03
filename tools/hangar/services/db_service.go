package services

import (
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/db"
)

// DBService exposes the dev-MySQL backup directory. Mirrors db.rs.
type DBService struct{}

func (s *DBService) DBBackupsDir(repo string) string { return db.BackupsDir(repo) }
func (s *DBService) DBEnsureBackupsDir(repo string) (string, error) {
	return db.EnsureBackupsDir(repo)
}
func (s *DBService) DBListBackups(repo string) ([]db.BackupEntry, error) {
	return db.ListBackups(repo)
}
func (s *DBService) DBSaveBackupMeta(path string, branch, note *string) error {
	return db.SaveBackupMeta(path, branch, note, uint64(time.Now().UnixMilli()))
}
func (s *DBService) DBDeleteBackup(repo, path string) error { return db.DeleteBackup(repo, path) }
func (s *DBService) DBCheckBackupName(repo, rawName string) (db.BackupNameCheck, error) {
	return db.CheckBackupName(repo, rawName)
}
