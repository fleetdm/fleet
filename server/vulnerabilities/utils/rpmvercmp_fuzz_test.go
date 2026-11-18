package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzRpmvercmp(f *testing.F) {
	for _, versions := range [][3]string{
		{"", "0", "1"},
		{"1.0", "1.0-1", "1.0-2"},
		{"1.0~rc1", "1.0", "1.1"},
		{"0:1.0-1", "1:1.0-1", "2:1.0-1"},
		{"1.0001", "1.1", "1.2"},
		{"6.0-4.el6.x86_64", "6.0-5.el6.x86_64", "6.1-1.el6.x86_64"},
	} {
		f.Add(versions[0], versions[1], versions[2])
	}
	f.Fuzz(func(t *testing.T, a, b, c string) {
		if len(a)+len(b)+len(c) > 1024 {
			t.Skip("versions exceed fuzz input budget")
		}
		require.Zero(t, Rpmvercmp(a, a), "comparison must be reflexive")
		ab, ba := Rpmvercmp(a, b), Rpmvercmp(b, a)
		require.Equal(t, -ab, ba, "comparison must be antisymmetric")
		bc := Rpmvercmp(b, c)
		if ab <= 0 && bc <= 0 {
			require.LessOrEqual(t, Rpmvercmp(a, c), 0, "comparison must be transitive")
		}
	})
}
