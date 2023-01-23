package secureboot

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/efi"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

type Table struct {
	logger log.Logger
}

func TablePlugin(_client *osquery.ExtensionManagerClient, logger log.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.IntegerColumn("secure_boot"),
		table.IntegerColumn("setup_mode"),
	}

	t := &Table{
		logger: logger,
	}

	return table.NewPlugin("kolide_secureboot", columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	sb, err := efi.ReadSecureBoot()
	if err != nil {
		level.Info(t.logger).Log("msg", "Unable to read secureboot", "err", err)
		return nil, fmt.Errorf("Reading secure_boot from efi: %w", err)
	}

	sm, err := efi.ReadSetupMode()
	if err != nil {
		level.Info(t.logger).Log("msg", "Unable to read setupmode", "err", err)
		return nil, fmt.Errorf("Reading setup_mode from efi: %w", err)
	}

	row := map[string]string{
		"secure_boot": boolToIntString(sb),
		"setup_mode":  boolToIntString(sm),
	}

	return []map[string]string{row}, nil
}

func boolToIntString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
