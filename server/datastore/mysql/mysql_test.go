package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeColumn(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input  string
		output string
	}{
		{"foobar-column", "foobar-column"},
		{"foobar_column", "foobar_column"},
		{"foobar;column", "foobarcolumn"},
		{"foobar#", "foobar"},
		{"foobar*baz", "foobarbaz"},
	}

	for _, tt := range testCases {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.output, sanitizeColumn(tt.input))
		})
	}
}

func TestSearchLike(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inSQL     string
		inParams  []interface{}
		match     string
		columns   []string
		outSQL    string
		outParams []interface{}
	}{
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{},
			match:     "foobar",
			columns:   []string{"hostname"},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE AND (hostname LIKE ?)",
			outParams: []interface{}{"%foobar%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{3},
			match:     "foobar",
			columns:   []string{},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE",
			outParams: []interface{}{3},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{1},
			match:     "foobar",
			columns:   []string{"hostname"},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE AND (hostname LIKE ?)",
			outParams: []interface{}{1, "%foobar%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{1},
			match:     "foobar",
			columns:   []string{"hostname", "uuid"},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE AND (hostname LIKE ? OR uuid LIKE ?)",
			outParams: []interface{}{1, "%foobar%", "%foobar%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE TRUE",
			inParams:  []interface{}{1},
			match:     "foobar",
			columns:   []string{"hostname", "uuid"},
			outSQL:    "SELECT * FROM HOSTS WHERE TRUE AND (hostname LIKE ? OR uuid LIKE ?)",
			outParams: []interface{}{1, "%foobar%", "%foobar%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE 1=1",
			inParams:  []interface{}{1},
			match:     "forty_%",
			columns:   []string{"ipv4", "uuid"},
			outSQL:    "SELECT * FROM HOSTS WHERE 1=1 AND (ipv4 LIKE ? OR uuid LIKE ?)",
			outParams: []interface{}{1, "%forty\\_\\%%", "%forty\\_\\%%"},
		},
		{
			inSQL:     "SELECT * FROM HOSTS WHERE 1=1",
			inParams:  []interface{}{1},
			match:     "forty_%",
			columns:   []string{"ipv4", "uuid"},
			outSQL:    "SELECT * FROM HOSTS WHERE 1=1 AND (ipv4 LIKE ? OR uuid LIKE ?)",
			outParams: []interface{}{1, "%forty\\_\\%%", "%forty\\_\\%%"},
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			sql, params := searchLike(tt.inSQL, tt.inParams, tt.match, tt.columns...)
			assert.Equal(t, tt.outSQL, sql)
			assert.Equal(t, tt.outParams, params)
		})
	}
}
