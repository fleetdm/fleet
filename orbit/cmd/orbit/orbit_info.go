package main

import (
	"context"

	"github.com/fleetdm/fleet/v4/orbit/pkg/build"
	"github.com/fleetdm/fleet/v4/orbit/pkg/fstoken"
	orbit_table "github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/osquery/osquery-go/plugin/table"
)

// orbitInfoExtension implements an extension table that provides info about Orbit.
type orbitInfoExtension struct {
	fstoken fstoken.Token
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
func (o orbitInfoExtension) GenerateFunc(_ context.Context, _ table.QueryContext) ([]map[string]string, error) {
	v := build.Version
	if v == "" {
		v = "unknown"
	}
	token, err := o.fstoken.Get()
	if err != nil {
		return nil, err
	}
	return []map[string]string{
		{
			"version":           v,
			"device_auth_token": token,
		},
	}, nil
}
