package oval_parsed

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectStateEvrStringEval(t *testing.T) {
	cases := []struct {
		val      string
		other    string
		cmp      func(string, string) int
		expected bool
	}{
		{"equals|1.1", "1.1", func(s1, s2 string) int { return 0 }, true},
		{"equals|1.1", "1.0", func(s1, s2 string) int { return 1 }, false},
		{"not equal|1.1", "2.1", func(s1, s2 string) int { return -1 }, true},
		{"not equal|1.1", "1.1", func(s1, s2 string) int { return 0 }, false},
		{"greater than|1.1", "2.1", func(s1, s2 string) int { return -1 }, false},
		{"greater than|1.1", "1.1", func(s1, s2 string) int { return 0 }, false},
		{"greater than|1.2", "1.1", func(s1, s2 string) int { return 1 }, true},
		{"greater than or equal|1.1", "2.1", func(s1, s2 string) int { return -1 }, false},
		{"greater than or equal|1.1", "1.1", func(s1, s2 string) int { return 0 }, true},
		{"greater than or equal|1.2", "1.1", func(s1, s2 string) int { return 1 }, true},
		{"less than|1.1", "2.1", func(s1, s2 string) int { return -1 }, true},
		{"less than|1.1", "1.1", func(s1, s2 string) int { return 0 }, false},
		{"less than|1.2", "1.1", func(s1, s2 string) int { return 1 }, false},
		{"less than or equal|1.1", "2.1", func(s1, s2 string) int { return -1 }, true},
		{"less than or equal|1.1", "1.1", func(s1, s2 string) int { return 0 }, true},
		{"less than or equal|1.2", "1.1", func(s1, s2 string) int { return 1 }, false},
	}

	for _, c := range cases {
		require.Equal(t, c.expected, ObjectStateEvrString(c.val).Eval(c.other, c.cmp))
	}
}
