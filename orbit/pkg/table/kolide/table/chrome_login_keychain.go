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

// DEPRECATED use kolide_chrome_login_data_emails
func ChromeLoginKeychainInfo(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	c := &ChromeLoginKeychain{
		client: client,
		logger: logger,
	}
	columns := []table.ColumnDefinition{
		table.TextColumn("origin_url"),
		table.TextColumn("action_url"),
		table.TextColumn("username_value"),
	}
	return table.NewPlugin("kolide_chrome_login_keychain", columns, c.generate)
}

type ChromeLoginKeychain struct {
	client *osquery.ExtensionManagerClient
	logger log.Logger
}

func (c *ChromeLoginKeychain) generateForPath(ctx context.Context, path string) ([]map[string]string, error) {
	dir, err := os.MkdirTemp("", "kolide_chrome_login_keychain")
	if err != nil {
		return nil, fmt.Errorf("creating kolide_chrome_login_keychain tmp dir: %w", err)
	}
	defer os.RemoveAll(dir) // clean up

	dst := filepath.Join(dir, "tmpfile")
	if err := fsutil.CopyFile(path, dst); err != nil {
		return nil, fmt.Errorf("copying db to tmp dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dst)
	if err != nil {
		return nil, fmt.Errorf("connecting to sqlite db: %w", err)
	}
	defer db.Close()

	db.Exec("PRAGMA journal_mode=WAL;")

	rows, err := db.Query("SELECT origin_url, action_url, username_value FROM logins")
	if err != nil {
		return nil, fmt.Errorf("query rows from chrome login keychain db: %w", err)
	}
	defer rows.Close()

	var results []map[string]string

	// loop through all the sqlite rows and add them as osquery rows in the results map
	for rows.Next() { // we initialize these variables for every row, that way we don't have data from the previous iteration
		var origin_url string
		var action_url string
		var username_value string
		if err := rows.Scan(&origin_url, &action_url, &username_value); err != nil {
			return nil, fmt.Errorf("scanning chrome login keychain db row: %w", err)
		}

		results = append(results, map[string]string{
			"origin_url":     origin_url,
			"action_url":     action_url,
			"username_value": username_value,
		})
	}
	return results, nil
}

func (c *ChromeLoginKeychain) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	files, err := findFileInUserDirs("Library/Application Support/Google/Chrome/*/Login Data", c.logger)
	if err != nil {
		return nil, fmt.Errorf("find chrome login data sqlite DBs: %w", err)
	}

	var results []map[string]string
	for _, file := range files {
		res, err := c.generateForPath(ctx, file.path)
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

	return results, nil
}
