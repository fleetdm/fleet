package file_test

import (
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	// Setup
	originalPath := filepath.Join(tmp, "original")
	dstPath := filepath.Join(tmp, "copy")
	expectedContents := []byte("foo")
	expectedMode := fs.FileMode(0644)
	require.NoError(t, os.WriteFile(originalPath, expectedContents, os.ModePerm)) //nolint:gosec // allow write file with 0o777
	require.NoError(t, os.WriteFile(dstPath, []byte("this should be overwritten"), expectedMode))

	// Test
	require.NoError(t, file.Copy(originalPath, dstPath, expectedMode))

	contents, err := os.ReadFile(originalPath)
	require.NoError(t, err)
	assert.Equal(t, expectedContents, contents)

	contents, err = os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, expectedContents, contents)

	info, err := os.Stat(dstPath)
	require.NoError(t, err)
	assert.Equal(t, expectedMode, info.Mode())

	// Copy of nonexistent path fails
	require.Error(t, file.Copy(filepath.Join(tmp, "notexist"), dstPath, os.ModePerm))

	// Copy to nonexistent directory
	require.Error(t, file.Copy(originalPath, filepath.Join("tmp", "notexist", "foo"), os.ModePerm))
}

func TestCopyWithPerms(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	// Setup
	originalPath := filepath.Join(tmp, "original")
	dstPath := filepath.Join(tmp, "copy")
	expectedContents := []byte("foo")
	expectedMode := fs.FileMode(0755)
	require.NoError(t, os.WriteFile(originalPath, expectedContents, expectedMode))

	// Test
	require.NoError(t, file.CopyWithPerms(originalPath, dstPath))

	contents, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, expectedContents, contents)

	info, err := os.Stat(dstPath)
	require.NoError(t, err)
	assert.Equal(t, expectedMode, info.Mode())
}

func TestExists(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	// Setup
	path := filepath.Join(tmp, "file")
	require.NoError(t, os.WriteFile(path, []byte(""), os.ModePerm)) //nolint:gosec // allow write file with 0o777
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "dir", "nested"), os.ModePerm))

	// Test
	exists, err := file.Exists(path)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = file.Exists(filepath.Join(tmp, "notexist"))
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = file.Exists(filepath.Join(tmp, "dir"))
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestExtractInstallerMetadata tests the ExtractInstallerMetadata function. It
// calls the function for every file under testdata/installers and checks that
// it returns the expected metadata by comparing it to the software name,
// version and hash in the filename.
//
// The filename should have the following format:
//
//	<software_name>$<version>$<sha256hash>$<bundle_identifier>[$<anything>].<extension>
//
// That is, it breaks the file name at the dollar sign and the first part is
// the expected name, the second is the expected version, the third is the
// hex-encoded hash and the fourth is the bundle identifier. Note that by
// default, files in testdata/installers are NOT included in git, so the test
// files must be added manually (for size and licenses considerations). Why the
// dollar sign? Because dots, dashes and underlines are more likely to be part
// of the name or version.
func TestExtractInstallerMetadata(t *testing.T) {
	dents, err := os.ReadDir(filepath.Join("testdata", "installers"))
	if err != nil {
		t.Fatal(err)
	}

	for _, dent := range dents {
		if !dent.Type().IsRegular() || strings.HasPrefix(dent.Name(), ".") {
			continue
		}
		t.Run(dent.Name(), func(t *testing.T) {
			parts := strings.Split(strings.TrimSuffix(dent.Name(), filepath.Ext(dent.Name())), "$")
			if len(parts) < 4 {
				t.Fatalf("invalid filename, expected at least 4 sections, got %d: %s", len(parts), dent.Name())
			}
			wantName, wantVersion, wantHash, wantBundleIdentifier := parts[0], parts[1], parts[2], parts[3]
			wantExtension := strings.TrimPrefix(filepath.Ext(dent.Name()), ".")

			tfr, err := fleet.NewKeepFileReader(filepath.Join("testdata", "installers", dent.Name()))
			require.NoError(t, err)
			defer tfr.Close()

			meta, err := file.ExtractInstallerMetadata(tfr)
			require.NoError(t, err)
			assert.Equal(t, wantName, meta.Name)
			assert.Equal(t, wantVersion, meta.Version)
			assert.Equal(t, wantHash, hex.EncodeToString(meta.SHASum))
			assert.Equal(t, wantExtension, meta.Extension)
			assert.Equal(t, wantBundleIdentifier, meta.BundleIdentifier)
		})
	}
}

func TestDos2UnixNewlines(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No newlines",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "Single Windows newline",
			input:    "Hello\r\nWorld",
			expected: "Hello\nWorld",
		},
		{
			name:     "Multiple Windows newlines",
			input:    "Hello\r\nWorld\r\nTest",
			expected: "Hello\nWorld\nTest",
		},
		{
			name:     "Mixed newlines",
			input:    "Hello\r\nWorld\nTest",
			expected: "Hello\nWorld\nTest",
		},
		{
			name:     "All unix",
			input:    "Hello\nWorld\nTest",
			expected: "Hello\nWorld\nTest",
		},
	}

	// Execute each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := file.Dos2UnixNewlines(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractFilenameFromURLPath(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"http://example.com", ""},
		{"http://example.com/", ""},
		{"http://example.com?foo=bar", ""},
		{"http://example.com/foo.pkg", "foo.pkg"},
		{"http://example.com/foo.exe", "foo.exe"},
		{"http://example.com/foo.pkg?bar=baz", "foo.pkg"},
		{"http://example.com/foo.bar.pkg", "foo.bar.pkg"},
		{"http://example.com/foo", "foo.pkg"},
		{"http://example.com/foo/bar/baz", "baz.pkg"},
		{"http://example.com/foo?bar=baz", "foo.pkg"},
	}

	for _, c := range cases {
		got := file.ExtractFilenameFromURLPath(c.in, "pkg")
		require.Equalf(t, c.out, got, "for URL %s", c.in)
	}
}
