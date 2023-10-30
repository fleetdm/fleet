//go:build !windows
// +build !windows

package gsettings

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestGsettingsValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		expected []map[string]string
	}{
		{
			filename: "blank.txt",
			expected: []map[string]string{},
		},
		{
			filename: "simple.txt",
			expected: []map[string]string{
				{
					"username": "tester",
					"key":      "access-key",
					"value":    "''",
					"schema":   "org.gnome.rhythmbox.plugins.webremote",
				},
				{
					"username": "tester",
					"key":      "foo-bar",
					"value":    "2",
					"schema":   "org.gnome.rhythmbox.plugins.webremote",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		table := GsettingsValues{
			logger: log.NewNopLogger(),
			getBytes: func(ctx context.Context, username string, buf *bytes.Buffer) error {
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
			})

			results, err := table.generate(ctx, qCon)
			require.NoError(t, err, "generating results from %s", tt.filename)
			require.ElementsMatch(t, tt.expected, results)
		})
	}
}

func TestPerUser(t *testing.T) {
	t.Parallel()
	if os.Getenv("RUN_GSETTINGS_PER_USER_TEST") != "true" {
		// all these tests will only pass if run as root, and with specific
		// gnome desktop settings set. The setup to make this test run in CI
		// is... complex
		t.Skip("skipping - proper setup not detected")
	}

	tests := []struct {
		usernames   []string
		keyNames    []string
		schemaNames []string
		expected    map[string]string
		unexpected  map[string]string
	}{
		{
			usernames:   []string{"blaed"},
			keyNames:    []string{"idle-delay"},
			schemaNames: []string{"org.gnome.desktop.session"},
			expected: map[string]string{
				"username": "blaed",
				"key":      "idle-delay",
				"value":    "uint32 240", //  TODO: should parse out the uint32...
				"schema":   "org.gnome.desktop.session",
			},
			unexpected: map[string]string{
				"username": "blaed",
				"key":      "idle-delay",
				"value":    "uint32 300", // the default/global value
				"schema":   "org.gnome.desktop.session",
			},
		},
		{
			usernames:   []string{"kids"},
			keyNames:    []string{"idle-delay"},
			schemaNames: []string{"org.gnome.desktop.session"},
			expected: map[string]string{
				"username": "kids",
				"key":      "idle-delay",
				"value":    "uint32 600",
				"schema":   "org.gnome.desktop.session",
			},
			unexpected: map[string]string{
				"username": "kids",
				"key":      "idle-delay",
				"value":    "uint32 300", // the default/global value
				"schema":   "org.gnome.desktop.session",
			},
		},
	}

	for _, tt := range tests {
		table := GsettingsValues{
			logger:   log.NewNopLogger(),
			getBytes: execGsettings,
		}
		mockQC := tablehelpers.MockQueryContext(map[string][]string{
			"username": tt.usernames,
			"schema":   tt.schemaNames,
			"key":      tt.keyNames,
		})

		rows, err := table.generate(context.TODO(), mockQC)
		require.NoError(t, err, "generating results")
		require.Contains(t, rows, tt.expected, "generated rows should contain the expected result")
		require.NotContains(t, rows, tt.unexpected, "generated rows should not contain the unexpected result")
	}
}

func TestListKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		expected []string
	}{
		{
			filename: "listkeys.txt",
			expected: []string{
				"nmines",
				"window-width",
				"ysize",
				"use-question-marks",
				"use-autoflag",
				"use-animations",
				"mode",
				"xsize",
				"theme",
				"window-height",
				"window-is-maximized",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		table := GsettingsMetadata{
			logger: log.NewNopLogger(),
			cmdRunner: func(ctx context.Context, args []string, tmpdir string, buf *bytes.Buffer) error {
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

			results, err := table.listKeys(ctx, "org.gnome.Mines", "faketmpdir")
			require.NoError(t, err, "generating results from %s", tt.filename)
			require.ElementsMatch(t, tt.expected, results)
		})
	}
}

func TestGetType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "type i",
			expected: "int32",
		},
		{
			input:    "range i 4 100",
			expected: "int32 (4 to 100)",
		},
		{
			input: `enum
'artists-albums'
'genres-artists'
'genres-artists-albums'
`,
			expected: "enum: [ 'artists-albums','genres-artists','genres-artists-albums' ]",
		},
		{
			input:    "type as",
			expected: "array of string",
		},
	}

	for _, tt := range tests {
		tt := tt
		table := GsettingsMetadata{
			logger: log.NewNopLogger(),
			cmdRunner: func(ctx context.Context, args []string, tmpdir string, buf *bytes.Buffer) error {
				_, err := buf.WriteString(tt.input)
				require.NoError(t, err)

				return nil
			},
		}
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()

			result, err := table.getType(ctx, "key", "schema", "fake-tmp-dir")
			require.NoError(t, err, "getting type", tt.expected)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "b",
			expected: "bool",
		},
		{
			input:    "n",
			expected: "int16",
		},
		{
			input:    "q",
			expected: "uint16",
		},
		{
			input:    "u",
			expected: "uint32",
		},
		{
			input:    "x",
			expected: "int64",
		},
		{
			input:    "t",
			expected: "uint64",
		},
		{
			input:    "d",
			expected: "double",
		},
		{
			input:    "s",
			expected: "string",
		},
		{
			input:    "as",
			expected: "array of string",
		},
		{
			input:    "ax",
			expected: "array of int64",
		},
		{
			input:    "at",
			expected: "array of uint64",
		},
		{
			input:    "(ss)", // tuples currently unsupported
			expected: "other",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			result := convertType(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
