//go:build darwin
// +build darwin

package remotectl

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed test-data/single_device_dumpstate.txt
var single_device_dumpstate string

//go:embed test-data/multiple_devices_dumpstate.txt
var multiple_devices_dumpstate string

//go:embed test-data/malformed_dumpstate_at_top_level.txt
var malformed_dumpstate_at_top_level string

//go:embed test-data/malformed_dumpstate_in_properties.txt
var malformed_dumpstate_in_properties string

func TestParse(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name                string
		input               []byte
		expectedDeviceCount int
		expectedValueCount  int
		expectedErr         bool
	}{
		{
			name:                "empty input",
			input:               []byte("\n"),
			expectedDeviceCount: 0,
			expectedValueCount:  0,
			expectedErr:         false,
		},
		{
			name:                "dumpstate with single device in output",
			input:               []byte(single_device_dumpstate),
			expectedDeviceCount: 1,
			expectedValueCount:  54,
			expectedErr:         false,
		},
		{
			name:                "dumpstate with multiple devices in output",
			input:               []byte(multiple_devices_dumpstate),
			expectedDeviceCount: 3,
			expectedValueCount:  188,
			expectedErr:         false,
		},
		{
			name:                "malformed dumpstate output -- malformed top-level property",
			input:               []byte(malformed_dumpstate_at_top_level),
			expectedDeviceCount: 0,
			expectedValueCount:  0,
			expectedErr:         true,
		},
		{
			name:                "malformed dumpstate output -- malformed item in Properties dict",
			input:               []byte(malformed_dumpstate_in_properties),
			expectedDeviceCount: 0,
			expectedValueCount:  0,
			expectedErr:         true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			result, err := p.Parse(bytes.NewReader(tt.input))
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)

			resultMap := result.(map[string]map[string]interface{})

			// Count the number of devices and the total number of data in the result, to confirm we extracted all the information we meant to
			actualDeviceCount := 0
			actualValueCount := 0
			for deviceName, deviceValues := range resultMap {
				if deviceName != "" && len(deviceValues) != 0 {
					actualDeviceCount += 1
				}
				assert.True(t, bytes.Contains(tt.input, []byte(deviceName)))
				// Confirm that we stripped "Found" from the front of the device name
				assert.False(t, strings.HasPrefix(deviceName, "Found"), fmt.Sprintf("device name not extracted correctly: got %s", deviceName))

				for topLevelKey, topLevelValue := range deviceValues {
					if topLevelKey == "Properties" {
						properties := topLevelValue.(map[string]interface{})
						for propertyKey, propertyValue := range properties {
							actualValueCount += 1
							validateKeyValueInCommandOutput(t, propertyKey, propertyValue.(string), tt.input)
						}

						continue
					}

					if topLevelKey == "Heartbeat" {
						for _, heartbeat := range topLevelValue.([]string) {
							actualValueCount += 1
							assert.True(t, bytes.Contains(tt.input, []byte(heartbeat)))
						}

						continue
					}

					if topLevelKey == "Services" || topLevelKey == "Local Services" {
						for _, service := range topLevelValue.([]map[string]interface{}) {
							for serviceKey, serviceValue := range service {
								if serviceKey == "Name" {
									actualValueCount += 1
									assert.True(t, bytes.Contains(tt.input, []byte(serviceValue.(string))))

									continue
								}

								if serviceKey == "Properties" {
									for servicePropertyKey, servicePropertyValue := range serviceValue.(map[string]interface{}) {
										actualValueCount += 1
										validateKeyValueInCommandOutput(t, servicePropertyKey, servicePropertyValue.(string), tt.input)
									}

									continue
								}

								actualValueCount += 1
								validateKeyValueInCommandOutput(t, serviceKey, serviceValue.(string), tt.input)
							}
						}

						continue
					}

					actualValueCount += 1
					validateKeyValueInCommandOutput(t, topLevelKey, topLevelValue.(string), tt.input)
				}
			}

			assert.Equal(t, tt.expectedDeviceCount, actualDeviceCount)
			assert.Equal(t, tt.expectedValueCount, actualValueCount)
		})
	}
}

func validateKeyValueInCommandOutput(t *testing.T, key, val string, commandOutput []byte) {
	// First, confirm that the key and value both exists in the original output
	assert.True(t, bytes.Contains(commandOutput, []byte(key)))
	assert.True(t, bytes.Contains(commandOutput, []byte(val)))

	// Validate that the key and value were associated with each other
	regexFmt := `\Q%s\E.+\Q%s\E` // match key, then any delimiter, then value, on one line
	re := regexp.MustCompile(fmt.Sprintf(regexFmt, key, val))
	assert.True(t, re.Match(commandOutput), fmt.Sprintf("expected to see %s : %s in original command output but did not", key, val))
}
