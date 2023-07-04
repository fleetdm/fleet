// sntp_request allows querying SNTP servers to get the timestamp
// and clock offset from a NTP server (in millisecond precision).
package sntp_request

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/beevik/ntp"
	"github.com/osquery/osquery-go/plugin/table"
)

func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("server"),

		table.BigIntColumn("timestamp_ms"),
		table.BigIntColumn("clock_offset_ms"),
	}
}

func GenerateFunc(_ context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
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
