// based on github.com/kolide/launcher/pkg/osquery/tables
package dev_table_tooling

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
)

func Test_generate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		commandName    string
		expectedResult []map[string]string
	}{
		{
			name: "no command name",
		},
		{
			name:        "malware",
			commandName: "ransomware.exe",
		},
		{
			name:           "should always work happy path",
			commandName:    "echo",
			expectedResult: []map[string]string{{"name": "echo", "args": "hello", "output": "hello"}},
		},
	}

	table := Table{logger: log.NewNopLogger()}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			constraints := make(map[string][]string)
			constraints["name"] = []string{tt.commandName}

			got, _ := table.generate(context.Background(), tablehelpers.MockQueryContext(constraints))

			if len(tt.expectedResult) <= 0 {
				assert.Empty(t, got)
				return
			}

			// Test for expected results
			assert.Equal(t, tt.expectedResult[0]["name"], got[0]["name"])
			assert.Equal(t, tt.expectedResult[0]["args"], got[0]["args"])

			// To verify output, let's convert back to utf8
			decodedOutput, _ := base64.StdEncoding.DecodeString(got[0]["output"])
			scanner := bufio.NewScanner(bytes.NewReader(decodedOutput))
			for scanner.Scan() {
				// Scanner "normalizes" the output by removing platform-specific newline characters
				firstLine := scanner.Text()
				assert.Equal(t, tt.expectedResult[0]["output"], firstLine)
				break
			}
		})
	}
}
