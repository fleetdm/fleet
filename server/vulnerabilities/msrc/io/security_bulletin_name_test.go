package io

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSecurityBulletinName(t *testing.T) {
	t.Run("#date", func(t *testing.T) {
		sut := NewSecurityBulletinName("Windows_10-2022_09_10.json")
		result, err := sut.date()
		require.NoError(t, err)
		require.Equal(t, 2022, result.Year())
		require.Equal(t, time.Month(9), result.Month())
		require.Equal(t, 10, result.Day())
	})

	t.Run("#ProductName", func(t *testing.T) {
		a := NewSecurityBulletinName("Windows_10-2022_09_10.json")
		require.Equal(t, "Windows 10", a.ProductName())
	})

	t.Run("#Before", func(t *testing.T) {
		a := NewSecurityBulletinName("Windows_10-2022_09_10.json")
		b := NewSecurityBulletinName("Windows_10-2022_10_10.json")
		c := NewSecurityBulletinName("Windows_10-2022_10_10.json")
		require.True(t, a.Before(b))
		require.False(t, b.Before(a))
		require.False(t, b.Before(c))
		require.False(t, c.Before(b))
	})
}
