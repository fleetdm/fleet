//go:build darwin
// +build darwin

// Package ioreg provides a table wrapper around the `ioreg` macOS
// command.
//
// As the returned data is a complex nested plist, this uses the
// dataflatten tooling. (See
// https://github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten)
// based on github.com/kolide/launcher/pkg/osquery/tables
package ioreg

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

const ioregPath = "/usr/sbin/ioreg"

const allowedCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type Table struct {
	logger    log.Logger
	tableName string
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		// ioreg input options. These match the ioreg
		// command line. See the ioreg man page.
		table.TextColumn("c"),
		table.IntegerColumn("d"),
		table.TextColumn("k"),
		table.TextColumn("n"),
		table.TextColumn("p"),
		table.IntegerColumn("r"), // boolean
	)

	t := &Table{
		logger:    logger,
		tableName: "ioreg",
	}

	return table.NewPlugin(t.tableName, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	gcOpts := []tablehelpers.GetConstraintOpts{
		tablehelpers.WithDefaults(""),
		tablehelpers.WithAllowedCharacters(allowedCharacters),
		tablehelpers.WithLogger(t.logger),
	}

	for _, ioC := range tablehelpers.GetConstraints(queryContext, "c", gcOpts...) {
		// We always need "-a", it's the "archive" output
		ioregArgs := []string{"-a"}

		if ioC != "" {
			ioregArgs = append(ioregArgs, "-c", ioC)
		}

		for _, ioD := range tablehelpers.GetConstraints(queryContext, "d", gcOpts...) {
			if ioD != "" {
				ioregArgs = append(ioregArgs, "-d", ioD)
			}

			for _, ioK := range tablehelpers.GetConstraints(queryContext, "k", gcOpts...) {
				if ioK != "" {
					ioregArgs = append(ioregArgs, "-k", ioK)
				}
				for _, ioN := range tablehelpers.GetConstraints(queryContext, "n", gcOpts...) {
					if ioN != "" {
						ioregArgs = append(ioregArgs, "-n", ioN)
					}

					for _, ioP := range tablehelpers.GetConstraints(queryContext, "p", gcOpts...) {
						if ioP != "" {
							ioregArgs = append(ioregArgs, "-p", ioP)
						}

						for _, ioR := range tablehelpers.GetConstraints(queryContext, "r", gcOpts...) {
							switch ioR {
							case "", "0":
								// do nothing
							case "1":
								ioregArgs = append(ioregArgs, "-r")
							default:
								level.Info(t.logger).Log("msg", "r should be blank, 0, or 1")
								continue
							}

							for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
								// Finally, an inner loop

								ioregOutput, err := tablehelpers.Exec(ctx, 30, []string{ioregPath}, ioregArgs, false)
								if err != nil {
									level.Info(t.logger).Log("msg", "ioreg failed", "err", err)
									continue
								}

								flatData, err := t.flattenOutput(dataQuery, ioregOutput)
								if err != nil {
									level.Info(t.logger).Log("msg", "flatten failed", "err", err)
									continue
								}

								rowData := map[string]string{
									"c": ioC,
									"d": ioD,
									"k": ioK,
									"n": ioN,
									"p": ioP,
									"r": ioR,
								}

								results = append(results, dataflattentable.ToMap(flatData, dataQuery, rowData)...)
							}
						}
					}
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
