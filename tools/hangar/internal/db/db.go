// Package db manages the dev-MySQL backup directory inside the Fleet repo
// (<repo>/db-backups). Ported from src-tauri/src/db.rs. The directory gets
// its own .gitignore so dumps stay out of git without touching the repo's
// main .gitignore. Metadata (branch/note/timestamp) lives in a JSON sidecar
// next to each dump.
package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
)

const (
	backupsDirName = "db-backups"
	backupExt      = ".sql.gz"
)

// BackupEntry is one dump plus any sidecar metadata.
type BackupEntry struct {
	Name        string  `json:"name"`
	Path        string  `json:"path"`
	Size        uint64  `json:"size"`
	MtimeMS     uint64  `json:"mtime_ms"`
	Branch      *string `json:"branch"`
	Note        *string `json:"note"`
	CreatedAtMS *uint64 `json:"created_at_ms"`
}

type backupMeta struct {
	CreatedAtMS *uint64 `json:"created_at_ms"`
	Branch      *string `json:"branch"`
	Note        *string `json:"note"`
}

// BackupNameCheck is the result of validating a user-supplied backup name.
type BackupNameCheck struct {
	FinalName    string `json:"final_name"`
	Exists       bool   `json:"exists"`
	RelativePath string `json:"relative_path"`
}

func backupsDir(repo string) string { return filepath.Join(repo, backupsDirName) }

// metaPathFor appends ".json" to the full backup filename so list/delete
// don't have to re-parse anything (foo.sql.gz -> foo.sql.gz.json).
func metaPathFor(backupPath string) string { return backupPath + ".json" }

// BackupsDir returns the (unguaranteed) backups directory path.
func BackupsDir(repo string) string { return backupsDir(repo) }

// EnsureBackupsDir creates the backups dir (with its .gitignore) and returns it.
func EnsureBackupsDir(repo string) (string, error) {
	dir := backupsDir(repo)
	if err := ensureDirWithGitignore(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func ensureDirWithGitignore(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	gi := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gi); errors.Is(err, os.ErrNotExist) {
		body := "# Auto-created by Fleet Hangar.\n# Ignore all backup artifacts here.\n*\n!.gitignore\n"
		if err := os.WriteFile(gi, []byte(body), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", gi, err)
		}
	}
	return nil
}

func readMeta(path string) backupMeta {
	b, err := os.ReadFile(path)
	if err != nil {
		return backupMeta{}
	}
	var m backupMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return backupMeta{}
	}
	return m
}

func mtimeMS(path string) uint64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return uint64(fi.ModTime().UnixMilli())
}

// ListBackups returns all *.sql.gz dumps in the backups dir, newest first,
// with sidecar metadata attached. A missing dir yields an empty list.
func ListBackups(repo string) ([]BackupEntry, error) {
	dir := backupsDir(repo)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []BackupEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	out := make([]BackupEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, backupExt) {
			continue
		}
		path := filepath.Join(dir, name)
		var size uint64
		if fi, err := os.Stat(path); err == nil {
			size = uint64(fi.Size())
		}
		meta := readMeta(metaPathFor(path))
		out = append(out, BackupEntry{
			Name:        name,
			Path:        path,
			Size:        size,
			MtimeMS:     mtimeMS(path),
			Branch:      meta.Branch,
			Note:        meta.Note,
			CreatedAtMS: meta.CreatedAtMS,
		})
	}
	// Newest first.
	sort.Slice(out, func(i, j int) bool { return out[i].MtimeMS > out[j].MtimeMS })
	return out, nil
}

// SaveBackupMeta writes the sidecar for a dump. branch/note are trimmed;
// empty becomes nil. nowMS is stamped as created_at_ms.
func SaveBackupMeta(path string, branch, note *string, nowMS uint64) error {
	meta := backupMeta{
		CreatedAtMS: &nowMS,
		Branch:      trimToNil(branch),
		Note:        trimToNil(note),
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPathFor(path), b, 0o644)
}

func trimToNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// DeleteBackup removes a dump (and its sidecar) after verifying it lives
// under <repo>/db-backups and has the backup extension. The repo is passed
// explicitly so we can't be coerced into deleting outside the project.
func DeleteBackup(repo, path string) error {
	dir := backupsDir(repo)
	// Clean the frontend-supplied path first: HasPathPrefix compares raw
	// components, so without this a ".." traversal like
	// "<dir>/../db-backups-evil/x.sql.gz" would pass the prefix check (its
	// leading components match dir) yet resolve to a *.sql.gz outside dir.
	path = filepath.Clean(path)
	if !paths.HasPathPrefix(path, dir) {
		return fmt.Errorf("refusing to delete outside %s", dir)
	}
	if !strings.HasSuffix(filepath.Base(path), backupExt) {
		return fmt.Errorf("refusing to delete non-backup file: %s", path)
	}
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("deleting %s: %w", path, err)
		}
	}
	// Sidecar is best-effort for "missing", but a real I/O error surfaces.
	meta := metaPathFor(path)
	if _, err := os.Stat(meta); err == nil {
		if err := os.Remove(meta); err != nil {
			return fmt.Errorf("deleting %s: %w", meta, err)
		}
	}
	return nil
}

// CheckBackupName validates a user-supplied backup name and reports the
// final ".sql.gz" filename, whether it already exists, and its repo-relative
// path. Allowed characters: letters, digits, dot, underscore, dash.
func CheckBackupName(repo, rawName string) (BackupNameCheck, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(rawName), "/")
	stem := strings.TrimSuffix(trimmed, backupExt)
	if stem == "" {
		return BackupNameCheck{}, errors.New("backup name cannot be empty")
	}
	if strings.HasPrefix(stem, ".") {
		return BackupNameCheck{}, errors.New("backup name cannot start with a dot")
	}
	for _, r := range stem {
		if !isSafeNameRune(r) {
			return BackupNameCheck{}, errors.New("backup name may only contain letters, digits, dot, underscore, and dash")
		}
	}
	finalName := stem + backupExt
	full := filepath.Join(backupsDir(repo), finalName)
	_, statErr := os.Stat(full)
	return BackupNameCheck{
		FinalName:    finalName,
		Exists:       statErr == nil,
		RelativePath: backupsDirName + "/" + finalName,
	}, nil
}

func isSafeNameRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'
}
