package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOperationType(t *testing.T) {
	cases := []struct {
		input    string
		expected OperationType
	}{
		{"equals", Equals},
		{"not equal", NotEqual},
		{"case insensitive equals", CaseInsensitiveEquals},
		{"case insensitive not equal", CaseInsensitiveNotEqual},
		{"greater than", GreaterThan},
		{"less than", LessThan},
		{"greater than or equal", GreaterThanOrEqual},
		{"less than or equal", LessThanOrEqual},
		{"bitwise and", BitwiseAnd},
		{"bitwise or", BitwiseOr},
		{"pattern match", PatternMatch},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, NewOperationType(c.input))
	}
}
