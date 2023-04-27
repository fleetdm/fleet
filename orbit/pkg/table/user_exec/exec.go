// build +darwin
// Package user_exec provides a generic way to run osquery as a user on macOS.
// Some built-in osquery tables (e.g. screenlock) will only provide data for the
// current user, which generally means "root" in fleet. This often isn't useful,
// since root should never be used in the gui on macOS.
package user_exec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// ExecOsqueryLaunchctl runs osquery under launchctl, in a user context.
func ExecOsqueryLaunchctl(ctx context.Context, timeoutSeconds int, username string, osqueryPath string, query string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	targetUser, err := user.Lookup(username)
	if err != nil {
		return nil, fmt.Errorf("looking up username %s: %w", username, err)
	}

	cmd := exec.CommandContext(ctx,
		"launchctl",
		"asuser",
		targetUser.Uid,
		osqueryPath,
		"--config_path", "/dev/null",
		"--disable_events",
		"--disable_database",
		"--disable_audit",
		"--ephemeral",
		"-S",
		"--json",
		query,
	)

	// On almost all macOS systems the root directory will be read only and we will get an
	// error if it tries to write. However, I cannot find any reason launchctl asuser or osquery would want to
	// write to the current directory
	cmd.Dir = "/"

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	cmd.Stdout, cmd.Stderr = stdout, stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running osquery. Got: '%s': %w", string(stderr.Bytes()), err)
	}

	return stdout.Bytes(), nil
}

func ExecOsqueryLaunchctlParsed(ctx context.Context, timeoutSeconds int, username string, osqueryPath string, query string) ([]map[string]string, error) {
	outBytes, err := ExecOsqueryLaunchctl(ctx, timeoutSeconds, username, osqueryPath, query)
	if err != nil {
		return nil, err
	}

	var osqueryResults []map[string]string

	if err := json.Unmarshal(outBytes, &osqueryResults); err != nil {
		log.Info().Err(err).Msg("error unmarshalling json")
		return nil, fmt.Errorf("unmarshalling json: %w", err)
	}

	return osqueryResults, nil
}

const (
	allowedUsernameCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-. "
)

// Table provides a table generator that will
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

type Table struct {
	osqueryd  string
	query     string
	tablename string
}

func TablePlugin(
	tablename string, osqueryd string, osqueryQuery string, columns []table.ColumnDefinition,
) *table.Plugin {
	columns = append(columns, table.TextColumn("user"))

	t := &Table{
		osqueryd:  osqueryd,
		query:     osqueryQuery,
		tablename: tablename,
	}

	return table.NewPlugin(t.tablename, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	users := validQueryUsers(queryContext)

	if len(users) == 0 {
		return nil, fmt.Errorf("The %s table requires a user", t.tablename)
	}

	for _, user := range users {
		osqueryResults, err := ExecOsqueryLaunchctlParsed(ctx, 5, user, t.osqueryd, t.query)
		if err != nil {
			log.Info().Err(err).Msgf("Failed to run osquery as user %s", user)
		}

		for _, row := range osqueryResults {
			row["user"] = user
			results = append(results, row)
		}
	}
	return results, nil
}

func validQueryUsers(queryContext table.QueryContext) []string {
	q, ok := queryContext.Constraints["user"]
	if !ok || len(q.Constraints) == 0 {
		return []string{}
	}

	users := []string{}

Outer:
	for _, c := range q.Constraints {
		if c.Operator != table.OperatorEquals {
			continue
		}
		for _, char := range c.Expression {
			if !strings.ContainsRune(allowedUsernameCharacters, char) {
				log.Info().Msgf("Attempted to use invalid username %s", c.Expression)
				continue Outer
			}
		}
		users = append(users, c.Expression)

	}

	return users
}
