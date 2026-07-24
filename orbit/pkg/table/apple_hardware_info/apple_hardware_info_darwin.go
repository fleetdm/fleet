//go:build darwin

package apple_hardware_info

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	osqclient "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

// Columns defines the schema for the apple_hardware_info table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("marketing_name"),
	}
}

// Generate queries system_info for the hardware model and maps it to its marketing name.
func Generate(ctx context.Context, _ table.QueryContext, socket string) ([]map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	c, err := osqclient.NewClient(socket, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	row, err := c.QueryRowContext(ctx, "SELECT hardware_model FROM system_info")
	if err != nil {
		return nil, err
	}

	model := row["hardware_model"]

	// Return an empty marketing_name when there's no mapping entry so a missing
	// mapping can be told apart from the raw model identifier.
	name := fleet.AppleHardwareModels[model]

	return []map[string]string{{
		"marketing_name": name,
	}}, nil
}
