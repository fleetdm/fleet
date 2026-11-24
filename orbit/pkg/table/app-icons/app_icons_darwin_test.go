//go:build darwin

package appicons

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/tiff"
)

// The fixtures are 8x8 RGBA TIFFs with associated alpha: the top half is
// opaque red, the bottom-left quarter is fully transparent and the
// bottom-right quarter is 50% transparent blue.
func TestNormalizeIconTIFF(t *testing.T) {
	cases := []struct {
		file string
		// goDecodes reports whether golang.org/x/image/tiff can decode the
		// fixture as is.
		goDecodes bool
	}{
		// macOS 27 returns app icons as 16-bit float TIFF (SampleFormat=3).
		{file: "rgba_float16.tiff", goDecodes: false},
		{file: "rgba_float32.tiff", goDecodes: false},
		// macOS 26 and earlier return 8-bit unsigned TIFF.
		{file: "rgba_uint8.tiff", goDecodes: true},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", c.file))
			require.NoError(t, err)

			_, err = tiff.Decode(bytes.NewReader(raw))
			if c.goDecodes {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "unsupported feature: sample format")
			}

			normalized, err := normalizeIconTIFF(raw)
			require.NoError(t, err)

			img, err := tiff.Decode(bytes.NewReader(normalized))
			require.NoError(t, err)
			require.Equal(t, image.Rect(0, 0, 8, 8), img.Bounds())

			assertNRGBA(t, img, 1, 1, color.NRGBA{R: 255, A: 255})
			assertNRGBA(t, img, 6, 1, color.NRGBA{R: 255, A: 255})
			assertNRGBA(t, img, 1, 6, color.NRGBA{})
			assertNRGBA(t, img, 6, 6, color.NRGBA{B: 255, A: 128})
		})
	}
}

func TestNormalizeIconTIFFInvalid(t *testing.T) {
	_, err := normalizeIconTIFF(nil)
	require.Error(t, err)
	_, err = normalizeIconTIFF([]byte("not a tiff"))
	require.Error(t, err)
}

func TestGenerateAppIcons(t *testing.T) {
	for _, path := range []string{
		"/System/Applications/Calculator.app",
		"/System/Applications/Utilities/Terminal.app",
		// Generic document icon.
		"/etc/hosts",
		"/nonexistent/Foo.app",
	} {
		t.Run(path, func(t *testing.T) {
			rows, err := generateAppIcons(t.Context(), pathQuery(path))
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, path, rows[0]["path"])
			assert.NotEmpty(t, rows[0]["hash"])

			pngBytes, err := base64.StdEncoding.DecodeString(rows[0]["icon"])
			require.NoError(t, err)
			img, err := png.Decode(bytes.NewReader(pngBytes))
			require.NoError(t, err)
			assert.Equal(t, image.Rect(0, 0, 128, 128), img.Bounds())
			_, _, _, a := img.At(64, 64).RGBA()
			assert.NotZero(t, a, "icon center should not be transparent")
		})
	}
}

func TestGenerateAppIconsRequiresPath(t *testing.T) {
	_, err := generateAppIcons(t.Context(), table.QueryContext{})
	require.Error(t, err)
}

func pathQuery(path string) table.QueryContext {
	return table.QueryContext{Constraints: map[string]table.ConstraintList{
		"path": {Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: path}}},
	}}
}

func assertNRGBA(t *testing.T, img image.Image, x, y int, want color.NRGBA) {
	t.Helper()
	got := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	// Alpha must survive exactly (within rounding); color channels only
	// matter where the pixel is not fully transparent.
	assert.InDelta(t, want.A, got.A, 2, "alpha at (%d,%d): got %v", x, y, got)
	if want.A == 0 {
		return
	}
	assert.InDelta(t, want.R, got.R, 8, "red at (%d,%d): got %v", x, y, got)
	assert.InDelta(t, want.G, got.G, 8, "green at (%d,%d): got %v", x, y, got)
	assert.InDelta(t, want.B, got.B, 8, "blue at (%d,%d): got %v", x, y, got)
}
