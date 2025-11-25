package versionfilters

import (
	"testing"

	"github.com/google/go-github/v37/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdobeAcrobatVersionFilter(t *testing.T) {
	t.Run("filters out 2020 from mixed version list", func(t *testing.T) {
		input := []*github.RepositoryContent{
			{Name: github.String("2020")},
			{Name: github.String("25.001.20844")},
			{Name: github.String("24.040.20160")},
		}

		result := AdobeAcrobatVersionFilter(input)

		require.Len(t, result, 2)
		assert.Equal(t, "25.001.20844", result[0].GetName())
		assert.Equal(t, "24.040.20160", result[1].GetName())
	})

	t.Run("preserves all versions when 2020 is not present", func(t *testing.T) {
		input := []*github.RepositoryContent{
			{Name: github.String("26.002.12345")},
			{Name: github.String("25.001.20844")},
			{Name: github.String("24.040.20160")},
		}

		result := AdobeAcrobatVersionFilter(input)

		require.Len(t, result, 3)
		assert.Equal(t, "26.002.12345", result[0].GetName())
		assert.Equal(t, "25.001.20844", result[1].GetName())
		assert.Equal(t, "24.040.20160", result[2].GetName())
	})

	t.Run("returns empty slice when input contains only 2020", func(t *testing.T) {
		input := []*github.RepositoryContent{
			{Name: github.String("2020")},
		}

		result := AdobeAcrobatVersionFilter(input)

		assert.Empty(t, result)
	})

	t.Run("returns empty slice for empty input", func(t *testing.T) {
		input := []*github.RepositoryContent{}

		result := AdobeAcrobatVersionFilter(input)

		assert.Empty(t, result)
	})

	t.Run("handles nil input gracefully", func(t *testing.T) {
		result := AdobeAcrobatVersionFilter(nil)

		assert.Empty(t, result)
	})

	t.Run("preserves versions that contain 2020 as substring", func(t *testing.T) {
		// Future-proofing: ensure we only filter exact "2020" match
		input := []*github.RepositoryContent{
			{Name: github.String("2020")},
			{Name: github.String("2020.1.0")}, // hypothetical future version
			{Name: github.String("25.001.20844")},
		}

		result := AdobeAcrobatVersionFilter(input)

		require.Len(t, result, 2)
		assert.Equal(t, "2020.1.0", result[0].GetName())
		assert.Equal(t, "25.001.20844", result[1].GetName())
	})
}

func TestApplyFilters(t *testing.T) {
	t.Run("applies filter for registered package", func(t *testing.T) {
		input := []*github.RepositoryContent{
			{Name: github.String("2020")},
			{Name: github.String("25.001.20844")},
		}

		result := ApplyFilters("Adobe.Acrobat.Pro", input)

		require.Len(t, result, 1)
		assert.Equal(t, "25.001.20844", result[0].GetName())
	})

	t.Run("returns original contents for unregistered package", func(t *testing.T) {
		input := []*github.RepositoryContent{
			{Name: github.String("1.0.0")},
			{Name: github.String("2.0.0")},
		}

		result := ApplyFilters("Some.Other.Package", input)

		require.Len(t, result, 2)
		assert.Equal(t, "1.0.0", result[0].GetName())
		assert.Equal(t, "2.0.0", result[1].GetName())
	})

	t.Run("handles empty package identifier", func(t *testing.T) {
		input := []*github.RepositoryContent{
			{Name: github.String("1.0.0")},
		}

		result := ApplyFilters("", input)

		assert.Equal(t, input, result)
	})
}
