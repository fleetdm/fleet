package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadQueriesFromYamlStrings(t *testing.T) {
	var testCases = []struct {
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
	var testCases = []struct{ queries []*Query }{
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

func TestValidateSQL(t *testing.T) {
	var testCases = []struct {
		sql       string
		shouldErr bool
	}{
		{"", false},
		{"foo", false},
		{"select 1", false},
		{"select * from time", false},
		{"select 1 from ATTACH", false},
		{"select * from attach where fast = 3", false},

		{"attach 'foo' as bar;", true},
		{"attach   (foo )as bar", true},
		{"attach('foo')AS bar", true},
		{"ATTACH 'foo' AS bar", true},
		{"ATTACH select name from where path = '/foobar' as bar", true},
		{`attach
   foo
    as
  bar`, true},
	}

	for _, tt := range testCases {
		t.Run(tt.sql, func(t *testing.T) {
			err := Query{Query: tt.sql}.ValidateSQL()
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
