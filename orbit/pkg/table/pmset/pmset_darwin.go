//go:build darwin
// +build darwin

// Package pmset implements the table for getting macOS power settings
// with the `pmset -g` command
package pmset

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("getting"), // pmset -g (aka GETTING option)
		table.TextColumn("json_result"),
	}
}

var linePattern = regexp.MustCompile(`^(\w+)[\t\f\r ]+(\S[\S\t\f\r ]*)$`)

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	getting := ""
	if constraints, ok := queryContext.Constraints["getting"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				getting = constraint.Expression
			}
		}
	}

	output, err := exec.CommandContext(ctx, "/usr/bin/pmset", "-g", getting).CombinedOutput()
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
