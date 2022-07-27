package main

import (
	"context"

	"github.com/fleetdm/fleet/v4/orbit/pkg/build"
	orbit_table "github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/osquery/osquery-go/plugin/table"
)

// orbitInfoExtension implements an extension table that provides info about Orbit.
type orbitInfoExtension struct {
	deviceAuthToken string
}

var _ orbit_table.Extension = orbitInfoExtension{}

// Name partially implements orbit_table.Extension.
func (o orbitInfoExtension) Name() string {
	return "orbit_info"
}

// Columns partially implements orbit_table.Extension.
func (o orbitInfoExtension) Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("version"),
		table.TextColumn("device_auth_token"),
	}
}

// GenerateFunc partially implements orbit_table.Extension.
func (o orbitInfoExtension) GenerateFunc(_ context.Context, qctx table.QueryContext) ([]map[string]string, error) {
	v := build.Version
	if v == "" {
		v = "unknown"
	}

	// get the server-approved token from the WHERE clause, and update the orbit token
	// if it is different.
	if whereClause := qctx.Constraints["device_auth_token"]; whereClause.Affinity == table.ColumnTypeText &&
		len(whereClause.Constraints) == 1 && whereClause.Constraints[0].Operator == table.OperatorEquals {
		if newToken := whereClause.Constraints[0].Expression; newToken != "" && newToken != o.deviceAuthToken {
			// TODO(mna): update local file with the new token
			// TODO(mna): this needs to be mutex-protected, the extension might run concurrently
			o.deviceAuthToken = newToken
		}
	}
	return []map[string]string{
		{
			"version":           v,
			"device_auth_token": o.deviceAuthToken,
		},
	}, nil
}
