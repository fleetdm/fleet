//go:build darwin
// +build darwin

// based on github.com/kolide/launcher/pkg/osquery/tables
package apple_silicon_security_policy

import (
	"bytes"
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

const (
	bootPolicyUtilPath = "/usr/bin/bputil"
	bootPolicyUtilArgs = "--display-all-policies"
)

type Table struct {
	logger log.Logger
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns()

	tableName := "apple_silicon_security_policy"

	t := &Table{
		logger: log.With(logger, "table", tableName),
	}

	return table.NewPlugin(tableName, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	output, err := tablehelpers.Exec(ctx, t.logger, 30, []string{bootPolicyUtilPath}, []string{bootPolicyUtilArgs}, false)
	if err != nil {
		level.Info(t.logger).Log("msg", "bputil failed", "err", err)
		return nil, nil
	}

	if len(output) == 0 {
		level.Info(t.logger).Log("msg", "No bputil data to parse")
		return nil, nil
	}

	data := parseBootPoliciesOutput(bytes.NewReader(output))

	for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
		flattened, err := dataflatten.Flatten(data, dataflatten.WithLogger(t.logger), dataflatten.WithQuery(strings.Split(dataQuery, "/")))
		if err != nil {
			level.Info(t.logger).Log("msg", "Error flattening data", "err", err)
			return nil, nil
		}
		results = append(results, dataflattentable.ToMap(flattened, dataQuery, nil)...)
	}

	return results, nil
}
