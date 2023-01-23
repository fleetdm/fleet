//go:build darwin
// +build darwin

// Package osquery_exec_table provides a table generator that will
// call osquery in a user context.
//
// This is necessary because some macOS tables need to run in user
// context. Running this in root context returns no
// results. Furthermore, these cannot run in sudo. Sudo sets the
// effective uid, but instead we need a bunch of keychain context.
//
// Resulting data is odd. If a user is logged in, even inactive,
// correct data is returned. If a user has not ever configured these
// settings, the default values are returned. If the user has
// configured these settings, _and_ the user is not logged in, no data
// is returned.

package osquery_user_exec_table

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	allowedUsernameCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-. "
)

type Table struct {
	client    *osquery.ExtensionManagerClient
	logger    log.Logger
	osqueryd  string
	query     string
	tablename string
}

func TablePlugin(
	client *osquery.ExtensionManagerClient, logger log.Logger,
	tablename string, osqueryd string, osqueryQuery string, columns []table.ColumnDefinition,
) *table.Plugin {
	columns = append(columns, table.TextColumn("user"))

	t := &Table{
		client:    client,
		logger:    logger,
		osqueryd:  osqueryd,
		query:     osqueryQuery,
		tablename: tablename,
	}

	return table.NewPlugin(t.tablename, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	users := tablehelpers.GetConstraints(queryContext, "user",
		tablehelpers.WithAllowedCharacters(allowedUsernameCharacters),
	)

	if len(users) == 0 {
		return nil, fmt.Errorf("The %s table requires a user", t.tablename)
	}

	for _, user := range users {
		osqueryResults, err := tablehelpers.ExecOsqueryLaunchctlParsed(ctx, t.logger, 5, user, t.osqueryd, t.query)
		if err != nil {
			continue
		}

		for _, row := range osqueryResults {
			row["user"] = user
			results = append(results, row)
		}
	}
	return results, nil
}
