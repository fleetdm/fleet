package fleet

import (
	"testing"

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
