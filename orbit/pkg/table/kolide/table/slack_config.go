package table

import (
	"context"
	"encoding/json"
	"errors"
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

var slackConfigDirs = map[string][]string{
	"windows": {
		"AppData/Roaming/Slack",
		"AppData/Local/Packages/*.Slack*/LocalCache/Roaming/Slack",
	},
	"darwin": {
		"Library/Application Support/Slack",
		"Library/Containers/com.tinyspeck.slackmacgap/Data/Library/Application Support/Slack",
	},
}

// try the list of known linux paths if runtime.GOOS doesn't match 'darwin' or 'windows'
var slackConfigDirDefault = []string{".config/Slack"}

func SlackConfig(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("team_id"),
		table.TextColumn("team_name"),
		table.TextColumn("team_url"),
		table.IntegerColumn("logged_in"),
		table.TextColumn("user_handle"),
		table.TextColumn("user_id"),
	}

	t := &SlackConfigTable{
		client: client,
		logger: logger,
	}

	return table.NewPlugin("kolide_slack_config", columns, t.generate)
}

type SlackConfigTable struct {
	client *osquery.ExtensionManagerClient
	logger log.Logger
}

type slackTeamsFile map[string]struct {
	LoggedIn   bool   `json:"hasValidSession"`
	TeamID     string `json:"team_id"`
	TeamName   string `json:"team_name"`
	TeamUrl    string `json:"team_url"`
	UserHandle string `json:"name"`
	UserID     string `json:"user_id"`
}

func (t *SlackConfigTable) generateForPath(ctx context.Context, file userFileInfo) ([]map[string]string, error) {
	var results []map[string]string
	data, err := os.ReadFile(file.path)
	if err != nil {
		return results, fmt.Errorf("Reading slack teams file: %w", err)
	}
	var slackTeamConfigs slackTeamsFile
	if err := json.Unmarshal(data, &slackTeamConfigs); err != nil {
		return results, fmt.Errorf("unmarshalling slack teams: %w", err)
	}
	for _, teamConfig := range slackTeamConfigs {
		results = append(results, map[string]string{
			"team_id":     teamConfig.TeamID,
			"team_name":   teamConfig.TeamName,
			"team_url":    teamConfig.TeamUrl,
			"logged_in":   strconv.Itoa(btoi(teamConfig.LoggedIn)),
			"user_handle": teamConfig.UserHandle,
			"user_id":     teamConfig.UserID,
		})
	}

	return results, nil
}

func (t *SlackConfigTable) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string
	// Prevent this table from being used to easily enumerate a user's slack teams
	q, ok := queryContext.Constraints["team_id"]
	if ok && len(q.Constraints) == 0 {
		return results, errors.New("The kolide_slack_config table requires that you specify a constraint WHERE team_id =")
	}
	if ok { // If we have a constraint on team_id limit it to the = operator
		for _, constraint := range q.Constraints {
			if constraint.Operator != table.OperatorEquals {
				return results, errors.New("The kolide_slack_config table only accepts = constraints on the team_id column")
			}
		}
	}
	osProfileDirs, ok := slackConfigDirs[runtime.GOOS]
	if !ok {
		osProfileDirs = slackConfigDirDefault
	}
	for _, profileDir := range osProfileDirs {
		files, err := findFileInUserDirs(filepath.Join(profileDir, "storage/slack-teams"), t.logger)
		if err != nil {
			level.Info(t.logger).Log(
				"msg", "Finding slack teams json",
				"path", profileDir,
				"err", err,
			)
			continue
		}
		for _, file := range files {
			res, err := t.generateForPath(ctx, file)
			if err != nil {
				level.Info(t.logger).Log(
					"msg", "Generating slack team result",
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
