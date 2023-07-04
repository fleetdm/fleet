//go:build darwin
// +build darwin

package sudo_info

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("json_result"),
	}
}

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	out, err := exec.Command("/usr/bin/sudo", "-V").Output()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}
	result := parseSudoVOutput(out)

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return []map[string]string{{
		"json_result": string(jsonResult),
	}}, nil
}

func parseSudoVOutput(output []byte) map[string]interface{} {
	scanner := bufio.NewScanner(bytes.NewReader(output))

	result := make(map[string]interface{})
	curKey := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			if result[curKey] == nil {
				result[curKey] = []string{}
			}
			result[curKey] = append(result[curKey].([]string), strings.TrimSpace(line))
			continue
		}
		colonIndex := strings.IndexByte(line, ':')
		if colonIndex == -1 {
			curKey = line
			result[line] = nil
			continue
		}
		if colonIndex == len(line)-1 {
			curKey = line[:len(line)-1]
			continue
		}
		result[line[0:colonIndex]] = line[colonIndex+2:] // +2 because of space character after colon
	}

	return result
}
