package table

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	fsutil "github.com/kolide/kit/fs"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

var profileDirs = map[string][]string{
	"windows": {"Appdata/Local/Google/Chrome/User Data"},
	"darwin":  {"Library/Application Support/Google/Chrome"},
}
var profileDirsDefault = []string{".config/google-chrome", ".config/chromium", "snap/chromium/current/.config/chromium"}

func ChromeLoginDataEmails(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	c := &ChromeLoginDataEmailsTable{
		client: client,
		logger: logger,
	}
	columns := []table.ColumnDefinition{
		table.TextColumn("username"),
		table.TextColumn("email"),
		table.BigIntColumn("count"),
	}
	return table.NewPlugin("kolide_chrome_login_data_emails", columns, c.generate)
}

type ChromeLoginDataEmailsTable struct {
	client *osquery.ExtensionManagerClient
	logger log.Logger
}

func (c *ChromeLoginDataEmailsTable) generateForPath(ctx context.Context, file userFileInfo) ([]map[string]string, error) {
	dir, err := os.MkdirTemp("", "kolide_chrome_login_data_emails")
	if err != nil {
		return nil, fmt.Errorf("creating kolide_chrome_login_data_emails tmp dir: %w", err)
	}
	defer os.RemoveAll(dir) // clean up

	dst := filepath.Join(dir, "tmpfile")
	if err := fsutil.CopyFile(file.path, dst); err != nil {
		return nil, fmt.Errorf("copying sqlite file to tmp dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dst)
	if err != nil {
		return nil, fmt.Errorf("connecting to sqlite db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT username_value, count(*) AS count FROM logins GROUP BY lower(username_value)")
	if err != nil {
		return nil, fmt.Errorf("query rows from chrome login keychain db: %w", err)
	}
	defer rows.Close()

	var results []map[string]string

	// loop through all the sqlite rows and add them as osquery rows in the results map
	for rows.Next() { // we initialize these variables for every row, that way we don't have data from the previous iteration
		var username_value string
		var username_count string
		if err := rows.Scan(&username_value, &username_count); err != nil {
			return nil, fmt.Errorf("scanning chrome login keychain db row: %w", err)
		}
		// append anything that could be an email
		if !strings.Contains(username_value, "@") {
			continue
		}
		results = append(results, map[string]string{
			"username": file.user,
			"email":    username_value,
			"count":    username_count,
		})
	}
	return results, nil
}

func (c *ChromeLoginDataEmailsTable) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string
	osProfileDirs, ok := profileDirs[runtime.GOOS]
	if !ok {
		osProfileDirs = profileDirsDefault
	}

	for _, profileDir := range osProfileDirs {
		files, err := findFileInUserDirs(filepath.Join(profileDir, "*/Login Data"), c.logger)
		if err != nil {
			level.Info(c.logger).Log(
				"msg", "Find chrome login data sqlite DBs",
				"path", profileDir,
				"err", err,
			)
			continue
		}

		for _, file := range files {
			res, err := c.generateForPath(ctx, file)
			if err != nil {
				level.Info(c.logger).Log(
					"msg", "Generating chrome keychain result",
					"path", file.path,
					"err", err,
				)
				continue
			}
			results = append(results, res...)
		}
	}

	return results, nil
}
