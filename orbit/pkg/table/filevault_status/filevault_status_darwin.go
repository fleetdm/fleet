//go:build darwin
// +build darwin

// based on https://github.com/fleetdm/launcher/blob/main/pkg/osquery/tables/filevault

// based on github.com/kolide/launcher/pkg/osquery/tables
package filevault_status

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

const fdesetupPath = "/usr/bin/fdesetup"

type Table struct {
	logger log.Logger
}

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("status"),
	}

	t := &Table{
		logger: logger,
	}

	return table.NewPlugin("filevault_status", columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	output, err := tablehelpers.Exec(ctx, 10, []string{fdesetupPath}, []string{"status"}, false)
	if err != nil {
		level.Info(t.logger).Log("msg", "fdesetup failed", "err", err)

		// Don't error out if the binary isn't found
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("calling fdesetup: %w", err)
	}

	status := strings.TrimSuffix(string(output), "\n")

	results := []map[string]string{
		{
			"status": status,
		},
	}
	return results, nil
}
