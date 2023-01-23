package table

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	fsutil "github.com/kolide/kit/fs"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

var onepasswordDataFiles = map[string][]string{
	"windows": {"AppData/Local/1password/data/1Password10.sqlite"},
	"darwin": {
		"Library/Application Support/1Password 4/Data/B5.sqlite",
		"Library/Group Containers/2BUA8C4S2C.com.agilebits/Library/Application Support/1Password/Data/B5.sqlite",
		"Library/Containers/2BUA8C4S2C.com.agilebits.onepassword-osx-helper/Data/Library/Data/B5.sqlite",
	},
}

func OnePasswordAccounts(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("username"),
		table.TextColumn("user_email"),
		table.TextColumn("team_name"),
		table.TextColumn("server"),
		table.TextColumn("user_first_name"),
		table.TextColumn("user_last_name"),
		table.TextColumn("account_type"),
	}

	o := &onePasswordAccountsTable{
		client: client,
		logger: logger,
	}

	return table.NewPlugin("kolide_onepassword_accounts", columns, o.generate)
}

type onePasswordAccountsTable struct {
	client *osquery.ExtensionManagerClient
	logger log.Logger
}

// generate the onepassword account info results given the path to a
// onepassword sqlite DB
func (o *onePasswordAccountsTable) generateForPath(ctx context.Context, fileInfo userFileInfo) ([]map[string]string, error) {
	dir, err := os.MkdirTemp("", "kolide_onepassword_accounts")
	if err != nil {
		return nil, fmt.Errorf("creating kolide_onepassword_accounts tmp dir: %w", err)
	}
	defer os.RemoveAll(dir) // clean up

	dst := filepath.Join(dir, "tmpfile")
	if err := fsutil.CopyFile(fileInfo.path, dst); err != nil {
		return nil, fmt.Errorf("copying sqlite db to tmp dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dst)
	if err != nil {
		return nil, fmt.Errorf("connecting to sqlite db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT user_email, team_name, server, user_first_name, user_last_name, account_type FROM accounts")
	if err != nil {
		return nil, fmt.Errorf("query rows from onepassword account configuration db: %w", err)
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var email, team, server, firstName, lastName, accountType string
		if err := rows.Scan(&email, &team, &server, &firstName, &lastName, &accountType); err != nil {
			return nil, fmt.Errorf("scanning onepassword account configuration db row: %w", err)
		}
		results = append(results, map[string]string{
			"user_email":      email,
			"username":        fileInfo.user,
			"team_name":       team,
			"server":          server,
			"user_first_name": firstName,
			"user_last_name":  lastName,
			"account_type":    accountType,
		})
	}
	return results, nil
}

func (o *onePasswordAccountsTable) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string
	osDataFiles, ok := onepasswordDataFiles[runtime.GOOS]
	if !ok {
		return results, errors.New("No onepasswordDataFiles for this platform")
	}

	for _, dataFilePath := range osDataFiles {
		files, err := findFileInUserDirs(dataFilePath, o.logger)
		if err != nil {
			level.Info(o.logger).Log(
				"msg", "Find 1password sqlite DBs",
				"path", dataFilePath,
				"err", err,
			)
			continue
		}

		for _, file := range files {
			res, err := o.generateForPath(ctx, file)
			if err != nil {
				level.Info(o.logger).Log(
					"msg", "Generating onepassword result",
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
