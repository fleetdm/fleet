// based on github.com/kolide/launcher/pkg/osquery/tables
package firefox_preferences

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

type Table struct {
	name   string
	logger log.Logger
}

const tableName = "firefox_preferences"

// For the first iteration of this table, we decided to do our own parsing with regex,
// leaving the JSON strings as-is.
//
// input  -> user_pref("app.normandy.foo", "{\"abc\":123}");
// output -> [user_pref("app.normandy.foo", "{"abc":123}"); app.normandy.foo {"abc":123}]
//
// Note that we do not capture the surrounding quotes for either groups.
//
// In the future, we may want to use go-mozpref:
// https://github.com/hansmi/go-mozpref
var re = regexp.MustCompile(`^user_pref\("([^,]+)",\s*"?(.*?)"?\);$`)

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		table.TextColumn("path"),
	)

	t := &Table{
		name:   tableName,
		logger: logger,
	}

	return table.NewPlugin(t.name, columns, t.generate)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	var results []map[string]string

	filePaths := tablehelpers.GetConstraints(queryContext, "path")

	if len(filePaths) == 0 {
		level.Info(t.logger).Log(
			"msg", fmt.Sprintf("no path provided to %s", tableName),
			"table", tableName,
		)
		return results, nil
	}

	for _, filePath := range filePaths {
		for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {
			flattenOpts := []dataflatten.FlattenOpts{
				dataflatten.WithQuery(strings.Split(dataQuery, "/")),
			}

			file, err := os.Open(filePath)
			if err != nil {
				level.Info(t.logger).Log(
					"msg", "failed to open file",
					"table", tableName,
					"path", filePath,
					"err", err,
				)
				continue
			}

			rawKeyVals := make(map[string]interface{})
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()

				// Given the line format: user_pref("app.normandy.first_run", false);
				// the return value should be a three element array, where the second
				// and third elements are the key and value, respectively.
				match := re.FindStringSubmatch(line)

				// If the match doesn't have a length of 3, the line is malformed in some way.
				// Skip it.
				if len(match) != 3 {
					continue
				}

				// The regex already stripped out the surrounding quotes, so now we're
				// left with escaped quotes that no longer make sense.
				// i.e. {\"249024122\":[1660860020218]}
				// Replace those with unescaped quotes.
				rawKeyVals[match[1]] = strings.ReplaceAll(match[2], "\\\"", "\"")
			}

			flatData, err := dataflatten.Flatten(rawKeyVals, flattenOpts...)
			if err != nil {
				level.Debug(t.logger).Log(
					"msg", "failed to flatten data for path",
					"table", tableName,
					"path", filePath,
					"err", err,
				)
				continue
			}

			rowData := map[string]string{"path": filePath}
			results = append(results, dataflattentable.ToMap(flatData, dataQuery, rowData)...)
		}
	}

	return results, nil
}
