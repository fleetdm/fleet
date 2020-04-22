package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeColumn(t *testing.T) {
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
			assert.Equal(t, tt.output, sanitizeColumn(tt.input))
		})
	}
}
