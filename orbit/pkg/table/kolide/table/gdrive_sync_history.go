package table

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	fsutil "github.com/kolide/kit/fs"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func GDriveSyncHistoryInfo(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	g := &GDriveSyncHistory{
		client: client,
		logger: logger,
	}
	columns := []table.ColumnDefinition{
		table.TextColumn("inode"),
		table.TextColumn("filename"),
		table.TextColumn("mtime"),
		table.TextColumn("size"),
	}
	return table.NewPlugin("kolide_gdrive_sync_history", columns, g.generate)
}

type GDriveSyncHistory struct {
	client *osquery.ExtensionManagerClient
	logger log.Logger
}

// GDriveSyncHistoryGenerate will be called whenever the table is queried. It should return
// a full table scan.
func (g *GDriveSyncHistory) generateForPath(ctx context.Context, path string) ([]map[string]string, error) {
	dir, err := os.MkdirTemp("", "kolide_gdrive_sync_history")
	if err != nil {
		return nil, fmt.Errorf("creating kolide_gdrive_sync_history tmp dir: %w", err)
	}
	defer os.RemoveAll(dir) // clean up

	dst := filepath.Join(dir, "tmpfile")
	if err := fsutil.CopyFile(path, dst); err != nil {
		return nil, fmt.Errorf("copying sqlite db to tmp dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dst)
	if err != nil {
		return nil, fmt.Errorf("connecting to sqlite db: %w", err)
	}
	defer db.Close()

	db.Exec("PRAGMA journal_mode=WAL;")

	rows, err := db.Query("select distinct le.inode, le.filename, le.modified AS mtime, le.size from local_entry le, cloud_entry ce using (checksum) order by le.modified desc;")
	if err != nil {
		return nil, fmt.Errorf("query rows from gdrive sync history db: %w", err)
	}
	defer rows.Close()

	var results []map[string]string

	// loop through all the sqlite rows and add them as osquery rows in the results map
	for rows.Next() { // we initialize these variables for every row, that way we don't have data from the previous iteration
		var inode string
		var filename string
		var mtime string
		var size string
		if err := rows.Scan(&inode, &filename, &mtime, &size); err != nil {
			return nil, fmt.Errorf("scanning gdrive sync history db row: %w", err)
		}

		results = append(results, map[string]string{
			"inode":    inode,
			"filename": filename,
			"mtime":    mtime,
			"size":     size,
		})
	}
	return results, nil
}

// GDriveSyncHistoryGenerate will be called whenever the table is queried. It should return
// a full table scan.
func (g *GDriveSyncHistory) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	files, err := findFileInUserDirs("Library/Application Support/Google/Drive/user_default/snapshot.db", g.logger)
	if err != nil {
		return nil, fmt.Errorf("find gdrive sync history sqlite DBs: %w", err)
	}

	var results []map[string]string
	for _, file := range files {
		res, err := g.generateForPath(ctx, file.path)
		if err != nil {
			level.Info(g.logger).Log(
				"msg", "Generating gdrive history result",
				"path", file.path,
				"err", err,
			)
			continue
		}
		results = append(results, res...)
	}

	return results, nil
}
