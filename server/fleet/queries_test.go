package fleet

import (
	"database/sql"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSnapshot(t *testing.T) {
	testCases := []struct {
		query    *Query
		expected *bool
	}{
		{
			query:    nil,
			expected: nil,
		},
		{
			query:    &Query{Logging: "snapshot"},
			expected: ptr.Bool(true),
		},
		{
			query:    &Query{Logging: "differential"},
			expected: nil,
		},
		{
			query:    &Query{Logging: "differential_ignore_removals"},
			expected: nil,
		},
	}
	for _, tCase := range testCases {
		require.Equal(t, tCase.expected, tCase.query.GetSnapshot())
	}
}

func TestGetRemoved(t *testing.T) {
	testCases := []struct {
		query    *Query
		expected *bool
	}{
		{
			query:    nil,
			expected: nil,
		},
		{
			query:    &Query{Logging: "snapshot"},
			expected: nil,
		},
		{
			query:    &Query{Logging: "differential"},
			expected: ptr.Bool(true),
		},
		{
			query:    &Query{Logging: "differential_ignore_removals"},
			expected: ptr.Bool(false),
		},
	}
	for i, tCase := range testCases {
		require.Equal(t, tCase.expected, tCase.query.GetRemoved(), i)
	}
}

func TestTeamIDStr(t *testing.T) {
	testCases := []struct {
		query    *Query
		expected string
	}{
		{
			query:    nil,
			expected: "",
		},
		{
			query:    &Query{},
			expected: "",
		},
		{
			query:    &Query{TeamID: ptr.Uint(10)},
			expected: "10",
		},
	}

	for _, tCase := range testCases {
		require.Equal(t, tCase.expected, tCase.query.TeamIDStr())
	}
}

func TestLoadQueriesFromYamlStrings(t *testing.T) {
	testCases := []struct {
		yaml      string
		queries   []*Query
		shouldErr bool
	}{
		{"notyaml", []*Query{}, true},
		{"", []*Query{}, false},
		{"---", []*Query{}, false},
		{
			`
---
apiVersion: k8s.fleet.com/v1alpha1
kind: OsqueryQuery
spec:
  name: osquery_version
  description: The osquery version info
  query: select * from osquery_info
  support:
    osquery: 2.9.0
---
apiVersion: k8s.fleet.com/v1alpha1
kind: OsqueryQuery
spec:
  name: osquery_schedule
  description: Report performance stats for each file in the query schedule.
  query: select name, interval, executions, output_size, wall_time, (user_time
---

apiVersion: k8s.fleet.com/v1alpha1
kind: OsqueryQuery
spec:
  name: foobar
  description: froobing
  query: select fizz from frog

`,
			[]*Query{
				{
					Name:        "osquery_version",
					Description: "The osquery version info",
					Query:       "select * from osquery_info",
				},
				{
					Name:        "osquery_schedule",
					Description: "Report performance stats for each file in the query schedule.",
					Query:       "select name, interval, executions, output_size, wall_time, (user_time",
				},
				{
					Name:        "foobar",
					Description: "froobing",
					Query:       "select fizz from frog",
				},
			},
			false,
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			queries, err := LoadQueriesFromYaml(tt.yaml)
			if tt.shouldErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
				assert.Equal(t, tt.queries, queries)
			}
		})
	}
}

func TestRoundtripQueriesYaml(t *testing.T) {
	testCases := []struct{ queries []*Query }{
		{[]*Query{{Name: "froob", Description: "bing", Query: "blong"}}},
		{
			[]*Query{
				{Name: "froob", Description: "bing", Query: "blong"},
				{Name: "mant", Description: "smump", Query: "tmit"},
				{Name: "gorm", Description: "", Query: "blirz"},
				{Name: "blob", Description: "shmoo", Query: "smarle"},
			},
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			yml, err := WriteQueriesToYaml(tt.queries)
			require.Nil(t, err)
			queries, err := LoadQueriesFromYaml(yml)
			require.Nil(t, err)
			assert.Equal(t, tt.queries, queries)
		})
	}
}

func TestVerifyQueryPlatforms(t *testing.T) {
	testCases := []struct {
		name           string
		platformString string
		shouldErr      bool
	}{
		{"empty platform string okay", "", false},
		{"platform string 'darwin' okay", "darwin", false},
		{"platform string 'linux' okay", "linux", false},
		{"platform string 'windows' okay", "windows", false},
		{"platform string 'darwin,linux,windows' okay", "darwin,linux,windows", false},
		{"platform string 'foo' invalid – not a supported platform", "foo", true},
		{"platform string 'charles,darwin,linux,windows' invalid – 'charles' not a supported platform", "charles,darwin,linux,windows", true},
		{"platform string 'darwin windows' invalid – missing comma delimiter", "darwin windows", true},
		{"platform string 'charles darwin' invalid – 'charles' not supported and missing comma delimiter", "charles darwin", true},
		{"platform string ';inux' invalid – ';inux' not a supported platform", ";inux", true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyQueryPlatforms(tt.platformString)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMapQueryReportResultRows(t *testing.T) {
	macOSUSBDevicesLastFetched := time.Now()
	ubuntuUSBDevicesLastFetched := time.Now().Add(-1 * time.Hour)
	macOSOsqueryInfoLastFetched := time.Now().Add(-2 * time.Hour)
	for _, tc := range []struct {
		name       string
		rows       []*ScheduledQueryResultRow
		expected   []HostQueryResultRow
		shouldFail bool
	}{
		{
			name: "USB devices query with results from a macOS and Linux host",
			rows: []*ScheduledQueryResultRow{
				{
					HostID:      1,
					Hostname:    sql.NullString{String: "macOS host", Valid: true},
					LastFetched: macOSUSBDevicesLastFetched,
					Data: ptr.RawMessage([]byte(`{
						"class": "9",
						"model": "AppleUSBVHCIBCE Root Hub Simulation",
						"model_id": "8000",
						"protocol": "",
						"removable": "0",
						"serial": "0",
						"subclass": "255",
						"usb_address": "",
						"usb_port": "",
						"vendor": "Apple Inc.",
						"vendor_id": "05bc",
						"version": "0.0"
					}`)),
				},
				{
					HostID:      1,
					Hostname:    sql.NullString{String: "macOS host", Valid: true},
					LastFetched: macOSUSBDevicesLastFetched,
					Data: ptr.RawMessage([]byte(`{
						"class": "9",
						"model": "AppleUSBXHCI Root Hub Simulation",
						"model_id": "8007",
						"protocol": "",
						"removable": "0",
						"serial": "0",
						"subclass": "255",
						"usb_address": "",
						"usb_port": "",
						"vendor": "Apple Inc.",
						"vendor_id": "05ac",
						"version": "0.0"
					}`)),
				},
				{
					HostID:      2,
					Hostname:    sql.NullString{String: "ubuntu host", Valid: true},
					LastFetched: ubuntuUSBDevicesLastFetched,
					Data: ptr.RawMessage([]byte(`{
						"class": "9",
						"model": "1.1 root hub",
						"model_id": "0001",
						"protocol": "0",
						"removable": "-1",
						"serial": "0000:02:00.0",
						"subclass": "0",
						"usb_address": "1",
						"usb_port": "1",
						"vendor": "Linux Foundation",
						"vendor_id": "1d6b",
						"version": "0602"
					}`)),
				},
			},
			expected: []HostQueryResultRow{
				{
					HostID:      1,
					Hostname:    "macOS host",
					LastFetched: macOSUSBDevicesLastFetched,
					Columns: map[string]string{
						"class":       "9",
						"model":       "AppleUSBVHCIBCE Root Hub Simulation",
						"model_id":    "8000",
						"protocol":    "",
						"removable":   "0",
						"serial":      "0",
						"subclass":    "255",
						"usb_address": "",
						"usb_port":    "",
						"vendor":      "Apple Inc.",
						"vendor_id":   "05bc",
						"version":     "0.0",
					},
				},
				{
					HostID:      1,
					Hostname:    "macOS host",
					LastFetched: macOSUSBDevicesLastFetched,
					Columns: map[string]string{
						"class":       "9",
						"model":       "AppleUSBXHCI Root Hub Simulation",
						"model_id":    "8007",
						"protocol":    "",
						"removable":   "0",
						"serial":      "0",
						"subclass":    "255",
						"usb_address": "",
						"usb_port":    "",
						"vendor":      "Apple Inc.",
						"vendor_id":   "05ac",
						"version":     "0.0",
					},
				},
				{
					HostID:      2,
					Hostname:    "ubuntu host",
					LastFetched: ubuntuUSBDevicesLastFetched,
					Columns: map[string]string{
						"class":       "9",
						"model":       "1.1 root hub",
						"model_id":    "0001",
						"protocol":    "0",
						"removable":   "-1",
						"serial":      "0000:02:00.0",
						"subclass":    "0",
						"usb_address": "1",
						"usb_port":    "1",
						"vendor":      "Linux Foundation",
						"vendor_id":   "1d6b",
						"version":     "0602",
					},
				},
			},
			shouldFail: false,
		},
		{
			name: "macOS osquery_info result",
			rows: []*ScheduledQueryResultRow{
				{
					HostID:      1,
					Hostname:    sql.NullString{String: "macOS host", Valid: true},
					LastFetched: macOSOsqueryInfoLastFetched,
					Data: ptr.RawMessage([]byte(`{
						"build_distro": "10.14",
						"build_platform": "darwin",
						"config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
						"config_valid": "1",
						"extensions": "active",
						"instance_id": "7f02ff0f-f8a7-4ba9-a1d2-66836b154f4a",
						"pid": "96730",
						"platform_mask": "21",
						"start_time": "1696421866",
						"uuid": "589966AE-074A-503B-B17B-54B05684A120",
						"version": "5.9.1",
						"watcher": "96729"
					}`)),
				},
			},
			expected: []HostQueryResultRow{
				{
					HostID:      1,
					Hostname:    "macOS host",
					LastFetched: macOSOsqueryInfoLastFetched,
					Columns: map[string]string{
						"build_distro":   "10.14",
						"build_platform": "darwin",
						"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
						"config_valid":   "1",
						"extensions":     "active",
						"instance_id":    "7f02ff0f-f8a7-4ba9-a1d2-66836b154f4a",
						"pid":            "96730",
						"platform_mask":  "21",
						"start_time":     "1696421866",
						"uuid":           "589966AE-074A-503B-B17B-54B05684A120",
						"version":        "5.9.1",
						"watcher":        "96729",
					},
				},
			},
			shouldFail: false,
		},
		{
			name: "invalid JSON result",
			rows: []*ScheduledQueryResultRow{
				{
					HostID:      3,
					Hostname:    sql.NullString{String: "bar", Valid: true},
					LastFetched: time.Now(),
					Data:        ptr.RawMessage([]byte(`invalid JSON`)),
				},
			},
			shouldFail: true,
		},
		{
			name: "invalid item value type",
			rows: []*ScheduledQueryResultRow{
				{
					HostID:      3,
					Hostname:    sql.NullString{String: "bar", Valid: true},
					LastFetched: time.Now(),
					Data:        ptr.RawMessage([]byte(`{"foobar": 1}`)),
				},
			},
			shouldFail: true,
		},
	} {
		results, err := MapQueryReportResultsToRows(tc.rows)
		if !tc.shouldFail {
			require.NoError(t, err)
			require.Equal(t, tc.expected, results)
		} else {
			require.Error(t, err)
		}
	}
}
