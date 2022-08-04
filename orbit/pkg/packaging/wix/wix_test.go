package wix

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWindowsJoin(t *testing.T) {
	cases := []struct {
		in  []string
		out string
	}{
		{[]string{"one\\two", "three"}, "one\\two\\three"},
		{[]string{"one/two/three", "four.txt"}, "one\\two\\three\\four.txt"},
		{[]string{"one", "two", "three"}, "one\\two\\three"},
		{[]string{"one/two/three", "four/five.txt"}, "one\\two\\three\\four\\five.txt"},
	}

	for _, c := range cases {
		require.Equal(t, c.out, windowsJoin(c.in...))
	}
}
