package main

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/beevik/ntp"
	orbit_table "github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/osquery/osquery-go/plugin/table"
)

// sntpRequest allows querying SNTP servers to get the timestamp
// and clock offset from a NTP server (in millisecond precision).
type sntpRequest struct{}

var _ orbit_table.Extension = sntpRequest{}

// Name partially implements orbit_table.Extension.
func (o sntpRequest) Name() string {
	return "sntp_request"
}

// Columns partially implements orbit_table.Extension.
func (t sntpRequest) Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("server"),

		table.BigIntColumn("timestamp_ms"),
		table.BigIntColumn("clock_offset_ms"),
	}
}

// GenerateFunc partially implements orbit_table.Extension.
func (t sntpRequest) GenerateFunc(_ context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	server := ""
	if constraints, ok := queryContext.Constraints["server"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				server = constraint.Expression
			}
		}
	}
	if server == "" {
		return nil, errors.New("missing SNTP server column constraint; e.g. WHERE server = 'my.sntp.server'")
	}

	options := ntp.QueryOptions{
		Timeout: 30 * time.Second,
	}
	response, err := ntp.QueryWithOptions(server, options)
	if err != nil {
		return nil, err
	}
	return []map[string]string{{
		"server": server,

		"timestamp_ms":    strconv.FormatInt(response.Time.UnixMilli(), 10),
		"clock_offset_ms": strconv.FormatInt(response.ClockOffset.Milliseconds(), 10),
	}}, nil
}
