//go:build darwin
// +build darwin

// Package profiles provides a table wrapper around the various
// profiles options.
//
// As the returned data is a complex nested plist, this uses the
// dataflatten tooling. (See
// https://godoc.org/github.com/kolide/launcher/pkg/dataflatten)

package profiles

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	profilesPath          = "/usr/bin/profiles"
	userAllowedCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	typeAllowedCharacters = "abcdefghijklmnopqrstuvwxyz"
)

var allowedCommands = []string{"show", "list", "status"} // Consider "sync" but that's a write comand

type Table struct {
	client    *osquery.ExtensionManagerClient
	logger    log.Logger
	tableName string
}

func TablePlugin(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	// profiles options. See `man profiles`. These may not be needed,
	// we use `show -all` as the default, and it probably covers
	// everything.
	columns := dataflattentable.Columns(
		table.TextColumn("user"),
		table.TextColumn("command"),
		table.TextColumn("type"),
	)

	t := &Table{
		client:    client,
		logger:    logger,
		tableName: "kolide_profiles",
	}

	return table.NewPlugin(t.tableName, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	for _, command := range tablehelpers.GetConstraints(queryContext, "command", tablehelpers.WithAllowedValues(allowedCommands), tablehelpers.WithDefaults("show")) {
		for _, profileType := range tablehelpers.GetConstraints(queryContext, "type", tablehelpers.WithAllowedCharacters(typeAllowedCharacters), tablehelpers.WithDefaults("")) {
			for _, user := range tablehelpers.GetConstraints(queryContext, "user", tablehelpers.WithAllowedCharacters(userAllowedCharacters), tablehelpers.WithDefaults("_all")) {
				for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {

					// apple documents `-output stdout-xml` as sending the
					// output to stdout, in xml. This, however, does not work
					// for some subset of the profiles command. I've reported it
					// to apple (feedback FB8962811), and while it may someday
					// be fixed, we need to support it where it is.
					dir, err := os.MkdirTemp("", "kolide_profiles")
					if err != nil {
						return nil, fmt.Errorf("creating kolide_profiles tmp dir: %w", err)
					}
					defer os.RemoveAll(dir)

					outputFile := filepath.Join(dir, "output.xml")

					profileArgs := []string{command, "-output", outputFile}

					if profileType != "" {
						profileArgs = append(profileArgs, "-type", profileType)
					}

					// setup the command line. This table overloads the `user`
					// column so one can select either:
					//   * All profiles merged, using the special value `_all` (this is the default)
					//   * The device profiles, using the special value `_device`
					//   * a user specific one, using the username
					switch {
					case user == "" || user == "_all":
						profileArgs = append(profileArgs, "-all")
					case user == "_device":
						break
					case user != "":
						profileArgs = append(profileArgs, "-user", user)
					default:
						return nil, fmt.Errorf("Unknown user argument: %s", user)
					}

					output, err := tablehelpers.Exec(ctx, t.logger, 30, []string{profilesPath}, profileArgs)
					if err != nil {
						level.Info(t.logger).Log("msg", "ioreg exec failed", "err", err)
						continue
					}

					if bytes.Contains(output, []byte("requires root privileges")) {
						level.Info(t.logger).Log("ioreg requires root privileges")
						continue
					}

					flattenOpts := []dataflatten.FlattenOpts{
						dataflatten.WithLogger(t.logger),
						dataflatten.WithQuery(strings.Split(dataQuery, "/")),
					}

					flatData, err := dataflatten.PlistFile(outputFile, flattenOpts...)
					if err != nil {
						level.Info(t.logger).Log("msg", "flatten failed", "err", err)
						continue
					}

					rowData := map[string]string{
						"command": command,
						"type":    profileType,
						"user":    user,
					}

					results = append(results, dataflattentable.ToMap(flatData, dataQuery, rowData)...)

				}
			}
		}
	}
	return results, nil
}

func (t *Table) flattenOutput(dataQuery string, systemOutput []byte) ([]dataflatten.Row, error) {
	flattenOpts := []dataflatten.FlattenOpts{
		dataflatten.WithLogger(t.logger),
		dataflatten.WithQuery(strings.Split(dataQuery, "/")),
	}

	return dataflatten.Plist(systemOutput, flattenOpts...)
}
