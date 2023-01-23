package table

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	osquery "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

var chromeLocalStateDirs = map[string][]string{
	"windows": {"Appdata/Local/Google/Chrome/User Data"},
	"darwin":  {"Library/Application Support/Google/Chrome"},
}

// try the list of known linux paths if runtime.GOOS doesn't match 'darwin' or 'windows'
var chromeLocalStateDirDefault = []string{".config/google-chrome", ".config/chromium", "snap/chromium/current/.config/chromium"}

func ChromeUserProfiles(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	c := &chromeUserProfilesTable{
		client: client,
		logger: logger,
	}

	columns := []table.ColumnDefinition{
		table.TextColumn("username"),
		table.TextColumn("email"),
		table.TextColumn("name"),
		table.IntegerColumn("ephemeral"),
	}

	return table.NewPlugin("kolide_chrome_user_profiles", columns, c.generate)
}

type chromeUserProfilesTable struct {
	client *osquery.ExtensionManagerClient
	logger log.Logger
}

type chromeLocalState struct {
	Profile struct {
		InfoCache map[string]chromeProfileInfo `json:"info_cache"`
	} `json:"profile"`
}

type chromeProfileInfo struct {
	Name      string `json:"name"`
	Ephemeral bool   `json:"is_ephemeral"`
	Email     string `json:"user_name"`
}

func (c *chromeUserProfilesTable) generateForPath(ctx context.Context, fileInfo userFileInfo) ([]map[string]string, error) {
	var results []map[string]string
	data, err := os.ReadFile(fileInfo.path)
	if err != nil {
		return nil, fmt.Errorf("reading chrome local state file: %w", err)
	}
	var localState chromeLocalState
	if err := json.Unmarshal(data, &localState); err != nil {
		return nil, fmt.Errorf("unmarshalling chome local state: %w", err)
	}

	for _, profileInfo := range localState.Profile.InfoCache {
		results = append(results, map[string]string{
			"username":  fileInfo.user,
			"email":     profileInfo.Email,
			"name":      profileInfo.Name,
			"ephemeral": strconv.Itoa(btoi(profileInfo.Ephemeral)),
		})
	}

	return results, nil
}

func (c *chromeUserProfilesTable) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	osChromeLocalStateDirs, ok := chromeLocalStateDirs[runtime.GOOS]
	if !ok {
		osChromeLocalStateDirs = chromeLocalStateDirDefault
	}

	var results []map[string]string
	for _, localStateFilePath := range osChromeLocalStateDirs {
		userFiles, err := findFileInUserDirs(filepath.Join(localStateFilePath, "Local State"), c.logger)
		if err != nil {
			level.Info(c.logger).Log(
				"msg", "Finding chrome local state file",
				"path", localStateFilePath,
				"err", err,
			)
			continue
		}
		for _, file := range userFiles {
			res, err := c.generateForPath(ctx, file)
			if err != nil {
				level.Info(c.logger).Log(
					"msg", "Generating user profile result",
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
