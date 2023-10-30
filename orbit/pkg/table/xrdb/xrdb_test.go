//go:build linux
// +build linux

package xrdb

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/kolide/launcher/pkg/osquery/tables/tablehelpers"
	"github.com/stretchr/testify/require"
)

func TestXrdbParse(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		filename string
		expected []map[string]string
	}{
		{
			filename: "blank.txt",
			expected: []map[string]string{},
		},
		{
			filename: "results.txt",
			expected: []map[string]string{
				{
					"username": "tester",
					"key":      "*customization",
					"value":    "-color",
					"display":  ":0",
				},

				{
					"username": "tester",
					"key":      "Xft.dpi",
					"value":    "96",
					"display":  ":0",
				},

				{
					"username": "tester",
					"key":      "Xft.antialias",
					"value":    "1",
					"display":  ":0",
				},
				{
					"username": "tester",
					"key":      "Xft.hinting",
					"value":    "1",
					"display":  ":0",
				},
				{
					"username": "tester",
					"key":      "Xft.hintstyle",
					"value":    "hintslight",
					"display":  ":0",
				},
				{
					"username": "tester",
					"key":      "Xft.rgba",
					"value":    "rgb",
					"display":  ":0",
				},
				{
					"username": "tester",
					"key":      "Xcursor.size",
					"value":    "24",
					"display":  ":0",
				},
				{
					"username": "tester",
					"key":      "Xcursor.theme",
					"value":    "Yaru",
					"display":  ":0",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		table := XRDBSettings{
			logger: log.NewNopLogger(),
			getBytes: func(ctx context.Context, display, username string, buf *bytes.Buffer) error {
				f, err := os.Open(filepath.Join("testdata", tt.filename))
				require.NoError(t, err, "opening file %s", tt.filename)
				_, err = buf.ReadFrom(f)
				require.NoError(t, err, "read file %s", tt.filename)

				return nil
			},
		}
		t.Run(tt.filename, func(t *testing.T) {
			t.Parallel()
			ctx := context.TODO()
			qCon := tablehelpers.MockQueryContext(map[string][]string{
				"username": {"tester"},
				"display":  {":0"},
			})

			results, err := table.generate(ctx, qCon)
			require.NoError(t, err, "generating results from %s", tt.filename)
			require.ElementsMatch(t, tt.expected, results)
		})
	}
}
