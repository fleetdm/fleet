package io

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSecurityBulletinName(t *testing.T) {
	t.Run("MacOfficeRelNotesFileName", func(t *testing.T) {
		now := time.Now()
		result := MacOfficeRelNotesFileName(now)
		require.Contains(t, result, "macoffice")
		require.Contains(t, result, strconv.Itoa(now.Year()))
		require.Contains(t, result, strconv.Itoa(int(now.Month())))
		require.Contains(t, result, strconv.Itoa(now.Day()))
	})

	t.Run("MSRCFileName", func(t *testing.T) {
		now := time.Now()
		result := MSRCFileName("Windows 2020", now)
		require.Contains(t, result, "Windows_2020")
		require.Contains(t, result, strconv.Itoa(now.Year()))
		require.Contains(t, result, strconv.Itoa(int(now.Month())))
		require.Contains(t, result, strconv.Itoa(now.Day()))
	})

	t.Run("String", func(t *testing.T) {
		sut, err := NewMSRCMetadata("Windows_10-2022_09_10.json")
		require.NoError(t, err)
		require.Equal(t, "Windows_10-2022_09_10.json", sut.String())
	})

	t.Run("validates timestamp on filename at construction time", func(t *testing.T) {
		_, err := NewMSRCMetadata("Windows_10-2022.json")
		require.Error(t, err)

		_, err = NewMacOfficeRelNotesMetadata("Windows_10-2022.json")
		require.Error(t, err)
	})

	t.Run("#date", func(t *testing.T) {
		sut, err := NewMSRCMetadata("Windows_10-2022_09_10.json")
		require.NoError(t, err)
		result, err := sut.date()
		require.NoError(t, err)
		require.Equal(t, 2022, result.Year())
		require.Equal(t, time.Month(9), result.Month())
		require.Equal(t, 10, result.Day())
	})

	t.Run("#ProductName", func(t *testing.T) {
		sut, err := NewMSRCMetadata("Windows_10-2022_09_10.json")
		require.NoError(t, err)
		require.Equal(t, "Windows 10", sut.ProductName())
	})

	t.Run("Product name not included in filename", func(t *testing.T) {
		sut := MetadataFileName{prefix: "", filename: "2022_09_10.json"}
		require.Equal(t, "", sut.ProductName())
	})

	t.Run("Date not included in filename", func(t *testing.T) {
		sut := MetadataFileName{prefix: "", filename: "Windows_10.json"}
		_, err := sut.date()
		require.Error(t, err)
	})

	t.Run("#Before", func(t *testing.T) {
		a, err := NewMSRCMetadata("Windows_10-2022_09_10.json")
		require.NoError(t, err)
		b, err := NewMSRCMetadata("Windows_10-2022_10_10.json")
		require.NoError(t, err)
		c, err := NewMSRCMetadata("Windows_10-2022_10_10.json")
		require.NoError(t, err)
		require.True(t, a.Before(b))
		require.False(t, b.Before(a))
		require.False(t, b.Before(c))
		require.False(t, c.Before(b))

		t.Run("when a is empty", func(t *testing.T) {
			a := MetadataFileName{}
			b, err := NewMSRCMetadata("Windows_10-2022_09_10.json")
			require.NoError(t, err)
			require.True(t, a.Before(b))
		})

		t.Run("when b is empty", func(t *testing.T) {
			a, err := NewMSRCMetadata("Windows_10-2022_09_10.json")
			b := MetadataFileName{}
			require.NoError(t, err)
			require.False(t, a.Before(b))
		})
	})
}
