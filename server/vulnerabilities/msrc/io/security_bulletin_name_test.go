package msrc_io

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
}
