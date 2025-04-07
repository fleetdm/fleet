package falcon_kernel_check

// based on github.com/kolide/launcher/pkg/osquery/tables
import (
	"context"
	"fmt"
	"regexp"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

const kernelCheckUtilPath = "/opt/CrowdStrike/falcon-kernel-check"

type Table struct {
	logger zerolog.Logger
	name   string
}

func TablePlugin(logger zerolog.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("kernel"),
		table.IntegerColumn("supported"),
		table.IntegerColumn("sensor_version"),
	}

	tableName := "falcon_kernel_check"
	t := &Table{
		name:   tableName,
		logger: logger.With().Str("table", tableName).Logger(),
	}

	return table.NewPlugin(tableName, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	output, err := tablehelpers.Exec(ctx, t.logger, 5, []string{kernelCheckUtilPath}, []string{}, false)
	if err != nil {
		t.logger.Info().Str("table", t.name).Err(err).Msg("exec failed")
		return nil, err
	}

	status, err := parseStatus(string(output))
	if err != nil {
		t.logger.Info().Str("table", t.name).Err(err).Msg("Error parsing exec status")
		return nil, err
	}

	results := []map[string]string{status}

	return results, nil
}

// Example falcon-kernel-check output:

// $ sudo /opt/CrowdStrike/falcon-kernel-check
// Host OS 5.13.0-51-generic #58~20.04.1-Ubuntu SMP Tue Jun 14 11:29:12 UTC 2022 is supported by Sensor version 14006.

// # Upgrade happens
// $ sudo /opt/CrowdStrike/falcon-kernel-check
// Host OS Linux 5.15.0-46-generic #49~20.04.1-Ubuntu SMP Thu Aug 4 19:15:44 UTC 2022 is not supported by Sensor version 14006.
//
// This regexp gets matches for the kernel string, supported status, and sensor version number
var kernelCheckRegexp = regexp.MustCompile(`^((?:Host OS (.*) (is supported|is not supported)))(?: by Sensor version (\d*))`)

func parseStatus(status string) (map[string]string, error) {
	matches := kernelCheckRegexp.FindAllStringSubmatch(status, -1)
	if len(matches) != 1 {
		return nil, fmt.Errorf("Failed to match output: %s", status)
	}
	if len(matches[0]) != 5 {
		return nil, fmt.Errorf("Got %d matches. Expected 5. Failed to match output: %s", len(matches[0]), status)
	}

	// matches[0][2] = kernel version string
	// matches[0][3] = (is supported|is not supported)
	// matches[0][4] = sensor version number
	supported := "0"
	if matches[0][3] == "is supported" {
		supported = "1"
	}

	data := make(map[string]string, 3)
	data["kernel"] = matches[0][2]
	data["supported"] = supported
	data["sensor_version"] = matches[0][4]

	return data, nil
}
