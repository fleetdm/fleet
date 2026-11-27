package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDpkgvercmp(t *testing.T) {
	const (
		LESS    = -1
		EQUAL   = 0
		GREATER = 1
	)

	cases := []struct {
		a        string
		expected int
		b        string
	}{
		{"1.0", EQUAL, "1.0"},
		{"1.0", EQUAL, "0:1.0"},
		{"1.01", EQUAL, "1.1"},
		{"1:1.0", GREATER, "2.0"},
		{"1:9.6p1-3ubuntu13.5", GREATER, "9.6p1-3ubuntu14"},
		{"1.0", LESS, "1.0-1"},
		{"1.0-1-1", GREATER, "1.0-1"},
		{"1.0~rc1", LESS, "1.0"},
		{"1.0~~", LESS, "1.0~"},
		{"1.0~rc1-1", LESS, "1.0-1"},
		{"1.0", LESS, "1.0a"},
		{"1.0a", LESS, "1.0+"},
		{"1.2.3+dfsg-1", GREATER, "1.2.3-1"},
		{"1.0-2ubuntu7", LESS, "1.0-2ubuntu17"},
		{"1.0-2ubuntu9", LESS, "1.0-2ubuntu10"},
		{"10.5.0-4ubuntu2.1cross1", GREATER, "10.5.0-4ubuntu2cross1"},
		{"0.17+2.4-3ubuntu1", LESS, "0.17+2.4-3ubuntu1.1"},
		{"1.1.0-2build2", LESS, "1.1.0-2ubuntu0.1"},
		{"3.0.13-0ubuntu3.5", LESS, "3.0.13-0ubuntu3.5+esm1"},
	}

	for _, c := range cases {
		t.Run(c.a+" vs "+c.b, func(t *testing.T) {
			require.Equal(t, c.expected, Dpkgvercmp(c.a, c.b))
			require.Equal(t, -c.expected, Dpkgvercmp(c.b, c.a))
		})
	}
}

func FuzzDpkgvercmp(f *testing.F) {
	for _, versions := range [][3]string{
		{"", "0", "1"},
		{"1.0~rc1", "1.0", "1.0-1"},
		{"0:1.0-1", "1:1.0-1", "2:1.0-1"},
		{"1.0-2ubuntu7", "1.0-2ubuntu17", "1.0-2ubuntu17.1"},
		{"1.0a", "1.0+", "1.0.1"},
	} {
		f.Add(versions[0], versions[1], versions[2])
	}
	f.Fuzz(func(t *testing.T, a, b, c string) {
		if len(a)+len(b)+len(c) > 1024 {
			t.Skip("versions exceed fuzz input budget")
		}
		require.Zero(t, Dpkgvercmp(a, a), "comparison must be reflexive")
		ab, ba := Dpkgvercmp(a, b), Dpkgvercmp(b, a)
		require.Equal(t, -ab, ba, "comparison must be antisymmetric")
		bc := Dpkgvercmp(b, c)
		if ab <= 0 && bc <= 0 {
			require.LessOrEqual(t, Dpkgvercmp(a, c), 0, "comparison must be transitive")
		}
	})
}
