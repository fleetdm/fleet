package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransformQuery(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{"foobar", "foobar*"},
		{"tim tom", "tim tom*"},
		{"f%5", "f%5*"},
		{"f-o-o-b-a-r", "f o o b a r*"},
		{"f-o+o-b--+a-r+", "f o o b a r*"},
		{"gandalf@the_white.com", "gandalf the_white.com*"},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.out, transformQuery(tt.in))
		})
	}
}

func TestQueryMinLength(t *testing.T) {
	testCases := []struct {
		in  string
		out bool
	}{
		{"a b c d", false},
		{"foobar", true},
		{"a foo fim b", true},
		{"a fo fi b*", false},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.out, queryMinLength(tt.in))
		})
	}
}
