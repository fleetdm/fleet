//go:build darwin
// +build darwin

// Package pwpolicy provides a table wrapper around the `pwpolicy` macOS
// command.
//
// As the returned data is a complex nested plist, this uses the
// dataflatten tooling. (See
// https://godoc.org/github.com/kolide/launcher/pkg/dataflatten)

package pwpolicy

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	pwpolicyPath = "/usr/bin/pwpolicy"
	pwpolicyCmd  = "getaccountpolicies"
)

type Table struct {
	client    *osquery.ExtensionManagerClient
	logger    log.Logger
	tableName string
	execCC    func(context.Context, string, ...string) *exec.Cmd
}

func TablePlugin(client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		table.TextColumn("username"),
	)

	t := &Table{
		client:    client,
		logger:    logger,
		tableName: "kolide_pwpolicy",
		execCC:    exec.CommandContext,
	}

	return table.NewPlugin(t.tableName, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	for _, pwpolicyUsername := range tablehelpers.GetConstraints(queryContext, "username", tablehelpers.WithDefaults("")) {
		pwpolicyArgs := []string{pwpolicyCmd}

		if pwpolicyUsername != "" {
			pwpolicyArgs = append(pwpolicyArgs, "-u", pwpolicyUsername)
		}

		for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
			pwPolicyOutput, err := t.execPwpolicy(ctx, pwpolicyArgs)
			if err != nil {
				level.Info(t.logger).Log("msg", "pwpolicy failed", "err", err)
				continue
			}

			flattenOpts := []dataflatten.FlattenOpts{
				dataflatten.WithLogger(t.logger),
				dataflatten.WithQuery(strings.Split(dataQuery, "/")),
			}

			flatData, err := dataflatten.Plist(pwPolicyOutput, flattenOpts...)
			if err != nil {
				level.Info(t.logger).Log("msg", "flatten failed", "err", err)
				continue
			}

			rowData := map[string]string{
				"username": pwpolicyUsername,
			}

			results = append(results, dataflattentable.ToMap(flatData, dataQuery, rowData)...)
		}
	}

	return results, nil
}

func (t *Table) execPwpolicy(ctx context.Context, args []string) ([]byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := t.execCC(ctx, pwpolicyPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	level.Debug(t.logger).Log("msg", "calling pwpolicy", "args", cmd.Args)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("calling pwpolicy. Got: %s: %w", string(stderr.Bytes()), err)
	}

	// Remove first line of output because it always contains non-plist content
	outputBytes := bytes.SplitAfterN(stdout.Bytes(), []byte("\n"), 2)[1]

	return outputBytes, nil
}
