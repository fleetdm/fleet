//go:build darwin
// +build darwin

// Package location_services_permissions implements the table for getting macOS location services permissions
// Main usage is by a query that inputs a list of applications and verifies that the allowed apps are contained within this list.
package location_services_permissions

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("allowed_app_list"), // serves as an input
		table.TextColumn("contained"),        // Is the list of allowed apps contained within the input list
		table.TextColumn("total_allowed"),    // How many apps are allowed to use location services
	}
}

// PR REVIEWER: COMMENT TO DELETE IF NOT USED -->   var linePattern = regexp.MustCompile(`^(\w+)[\t\f\r ]+(\S[\S\t\f\r ]*)$`)

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	allowedAppList := ""
	if constraints, ok := queryContext.Constraints["allowed_app_list"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				allowed_app_list = constraint.Expression
			}
		}
	}

	output, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", "/var/db/locationd/clients.plist").CombinedOutput()
	if err != nil {
		return nil, err
	}

	result := parsePMSetOutput(output)

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return []map[string]string{{
		"getting": getting,

		"json_result": string(jsonResult),
	}}, nil
}

func parsePMSetOutput(output []byte) map[string]interface{} {
	scanner := bufio.NewScanner(bytes.NewReader(output))

	result := make(map[string]interface{})
	curKey := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line[0] != ' ' {
			curKey = strings.TrimSpace(line)
			result[curKey] = make(map[string]string)
			continue
		}
		line = strings.TrimSpace(line)
		loc := linePattern.FindStringSubmatch(line)
		if loc == nil {
			log.Debug().Str("line", line).Msg("failed to match line, ignoring")
			continue
		}
		if len(loc) != 3 {
			log.Debug().Str("line", line).Msg("invalid number of submatches")
			continue
		}
		m := result[curKey].(map[string]string)
		m[loc[1]] = loc[2]
	}

	return result
}
