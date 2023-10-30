package repcli

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test-data/repcli_linux.txt
var repcli_linux_status []byte

//go:embed test-data/repcli_darwin.txt
var repcli_mac_status []byte

func TestParse(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name     string
		input    []byte
		expected map[string]any
	}{
		{
			name:     "empty input",
			expected: resultMap{},
		},
		{
			name: "unexpected format input",
			input: []byte(`
Test: topLevelValue
Nested:
	Sub: Section
	Double Sub:
		second Level: value
		Triple Nested Flag Test:
			Deepest Flag: deepflag1
			Deepest Flag: deepflag2
			Deepest Lone Value: lone Value
You Should Not See this erroneous L1n3
			`),
			expected: resultMap{
				"nested": resultMap{
					"sub": "Section",
					"double_sub": resultMap{
						"second_level": "value",
						"triple_nested_flag_test": resultMap{
							"deepest_flag":       []string{"deepflag1", "deepflag2"},
							"deepest_lone_value": "lone Value",
						},
					},
				},
				"test": "topLevelValue",
			},
		},
		{
			name:  "repcli linux status",
			input: repcli_linux_status,
			expected: resultMap{
				"cloud_status": resultMap{
					"proxy":          "No",
					"registered":     "Yes",
					"server_address": "https://dev-prod06.example.com",
				},
				"general_info": resultMap{
					"devicehash":     "test6b7v9Xo5bX50okW5KABCD+wHxb/YZeSzrZACKo0=",
					"deviceid":       "123453928",
					"quarantine":     "No",
					"sensor_version": "2.14.0.1234321",
				},
				"rules_status": resultMap{
					"policy_name":      "LinuxDefaultPolicy",
					"policy_timestamp": "02/20/2023",
				},
				"sensor_status": resultMap{
					"details": resultMap{
						"liveresponse": []string{
							"NoSession",
							"Enabled",
							"NoKillSwitch",
						},
					},
					"state": "Enabled",
				},
			},
		},
		{
			name:  "repcli mac status",
			input: repcli_mac_status,
			expected: resultMap{
				"cloud_status": resultMap{
					"mdm_device_id":      "99999999-4C8C-45A0-B3EA-053672776382",
					"next_check-in":      "Now",
					"next_cloud_upgrade": "None",
					"platform_type":      "CLIENT_ARM64",
					"private_logging":    "Disabled",
					"registered":         "Yes",
					"server_address":     "https://dev-prod05.example.com",
				},
				"enforcement_status": resultMap{
					"execution_blocks":     "0",
					"network_restrictions": "0",
				},
				"full_disk_access_configurations": resultMap{
					"osquery":          "Unknown",
					"repmgr":           "Not Configured",
					"system_extension": "Unknown",
					"uninstall_helper": "Unknown",
					"uninstall_ui":     "Unknown",
				},
				"general_info": resultMap{
					"background_scan":    "Complete",
					"fips_mode":          "Disabled",
					"kernel_file_filter": "Connected",
					"kernel_type":        "System Extension",
					"last_reset":         "not set",
					"sensor_restarts":    "1911",
					"sensor_version":     "3.7.2.81",
					"system_extension":   "Running",
				},
				"proxy_settings": resultMap{
					"proxy_configured": "No",
				},
				"queues": resultMap{
					"livequeries": resultMap{
						"completed":   "0",
						"outstanding": "0",
						"peak":        "2",
					},
					"pscevents_batch_upload": resultMap{
						"failed":               "0",
						"mean_data_rate_(b/s)": "7583",
						"pending":              "0",
						"uploaded":             "1727",
					},
					"reputation_expedited": resultMap{
						"last_completed_id": "50",
						"last_queue_id":     "50",
						"max_outstanding":   "2",
						"outstanding":       "0",
						"total_queued":      "50",
					},
					"reputation_resubmit": resultMap{
						"max_outstanding": "0",
						"outstanding":     "0",
						"total_queued":    "0",
					},
					"reputation_slow": resultMap{
						"demand":   "0",
						"ready":    "128",
						"resubmit": "0",
						"stale":    "715",
					},
				},
				"rules_status": resultMap{
					"active_policies": resultMap{
						"dc_allow_external_devices_revision[1]":         "Enabled(Manifest)",
						"device_control_reporting_policy_revision[5]":   "Enabled(Manifest)",
						"eedr_reporting_revision[18]":                   "Enabled(Manifest)",
						"sensor_telemetry_reporting_policy_revision[3]": "Enabled(Built-in)",
					},
					"endpoint_standard_product": "Enabled",
					"enterprise_edr_product":    "Enabled",
					"policy_name":               "Workstations",
					"policy_timestamp":          "08/22/2023 15:19:53",
				},
				"sensor_state": resultMap{
					"boot_count": "103",
					"details": resultMap{
						"fulldiskaccess": "NotEnabled",
						"liveresponse": []string{
							"NoSession",
							"NoKillSwitch",
							"Enabled",
						},
					},
					"first_boot_after_os_upgrade": "No",
					"service_uptime":              "155110500 ms",
					"service_waketime":            "37860000 ms",
					"state":                       "Enabled",
					"svcstable":                   "Yes",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := New()
			result, err := p.Parse(bytes.NewReader(tt.input))
			require.NoError(t, err, "unexpected error parsing input")

			require.Equal(t, tt.expected, result)
		})
	}
}
