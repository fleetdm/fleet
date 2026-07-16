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

// --- central per-server backups (app-data), addressed by directory ---

// DBServerBackupsDir returns the central backups dir for a server id.
func (s *DBService) DBServerBackupsDir(serverID string) (string, error) {
	return db.ServerBackupsDir(serverID)
}
func (s *DBService) DBEnsureDir(dir string) (string, error) { return db.EnsureDir(dir) }
func (s *DBService) DBListBackupsInDir(dir string) ([]db.BackupEntry, error) {
	return db.ListBackupsInDir(dir)
}
func (s *DBService) DBDeleteBackupInDir(dir, path string) error {
	return db.DeleteBackupInDir(dir, path)
}
func (s *DBService) DBCheckBackupNameInDir(dir, rawName string) (db.BackupNameCheck, error) {
	return db.CheckBackupNameInDir(dir, rawName)
}
