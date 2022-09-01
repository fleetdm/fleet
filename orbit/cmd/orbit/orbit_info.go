package main

import (
	"context"

	"github.com/fleetdm/fleet/v4/orbit/pkg/build"
	orbit_table "github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/fleetdm/fleet/v4/orbit/pkg/token"
	"github.com/osquery/osquery-go/plugin/table"
)

// orbitInfoExtension implements an extension table that provides info about Orbit.
type orbitInfoExtension struct {
	trw *token.ReadWriter
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

	var err error
	var token string
	if o.trw != nil {
		if token, err = o.trw.Read(); err != nil {
			return nil, err
		}
	}

	return []map[string]string{
		{
			"version":           v,
			"device_auth_token": token,
		},
	}, nil
}
