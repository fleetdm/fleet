//go:build darwin

package executable_hashes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func TestGenerateWithExactPath(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)

	// Create a macOS app bundle structure
	bundlePath := filepath.Join(dir, "Test.app")
	contentsDir := filepath.Join(bundlePath, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")
	require.NoError(t, os.MkdirAll(macosDir, 0o755))

	execName := "Test"
	// Create Info.plist with CFBundleExecutable key
	infoPlistPath := filepath.Join(contentsDir, "Info.plist")
	infoPlistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>%s</string>
</dict>
</plist>`, execName)
	require.NoError(t, os.WriteFile(infoPlistPath, []byte(infoPlistContent), 0o644))

	// Create the actual executable binary in Contents/MacOS/
	execPath := filepath.Join(macosDir, execName)
	content := []byte("test file content for hashing")
	require.NoError(t, os.WriteFile(execPath, content, 0o644))

	h := sha256.New()
	h.Write(content)
	expectedHash := hex.EncodeToString(h.Sum(nil))

	rows, err := Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{
					Expression: bundlePath,
					Operator:   table.OperatorEquals,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, bundlePath, rows[0][colPath])
	require.Equal(t, execPath, rows[0][colExecPath])
	require.Equal(t, expectedHash, rows[0][colExecHash])
}

func TestGenerateWithWildcard(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)

	testBundles := map[string]struct {
		executableName string
		content        []byte
	}{
		"Foo.app":      {"Foo", []byte("content of foo")},
		"Bar.app":      {"Bar", []byte("content of bar")},
		"Baz.service":  {"Baz", []byte("content of baz")},
		"Bonk.service": {"Bonk", []byte("content of bonk")},
	}

	expectedHashByBundlePath := make(map[string]string)
	expectedExecPathByBundlePath := make(map[string]string)

	// Create macOS app bundle structures
	for bundleName, bundleInfo := range testBundles {
		bundlePath := filepath.Join(dir, bundleName)
		contentsDir := filepath.Join(bundlePath, "Contents")
		macosDir := filepath.Join(contentsDir, "MacOS")
		require.NoError(t, os.MkdirAll(macosDir, 0o755))

		// Create Info.plist with CFBundleExecutable key
		infoPlistPath := filepath.Join(contentsDir, "Info.plist")
		infoPlistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>%s</string>
</dict>
</plist>`, bundleInfo.executableName)
		require.NoError(t, os.WriteFile(infoPlistPath, []byte(infoPlistContent), 0o644))

		// Create the actual executable in Contents/MacOS/
		execPath := filepath.Join(macosDir, bundleInfo.executableName)
		require.NoError(t, os.WriteFile(execPath, bundleInfo.content, 0o644))

		h := sha256.New()
		h.Write(bundleInfo.content)
		expectedHashByBundlePath[bundlePath] = hex.EncodeToString(h.Sum(nil))
		expectedExecPathByBundlePath[bundlePath] = execPath
	}

	rows, err := Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{
					Expression: filepath.Join(dir, "%.app"),
					Operator:   table.OperatorLike,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	serviceRows, err := Generate(context.Background(), table.QueryContext{
		Constraints: map[string]table.ConstraintList{
			colPath: {
				Constraints: []table.Constraint{{
					Expression: filepath.Join(dir, "%.service"),
					Operator:   table.OperatorLike,
				}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, serviceRows, 2)
	rows = append(rows, serviceRows...)

	got := make(map[string]fileInfo, 4)
	for _, row := range rows {
		got[row[colPath]] = fileInfo{
			Path:       row[colPath],
			ExecPath:   row[colExecPath],
			ExecSha256: row[colExecHash],
		}
	}

	for bundlePath, expectedHash := range expectedHashByBundlePath {
		require.Contains(t, got, bundlePath)
		info := got[bundlePath]
		require.Equal(t, bundlePath, info.Path)
		require.Equal(t, expectedExecPathByBundlePath[bundlePath], info.ExecPath)
		require.Equal(t, expectedHash, info.ExecSha256)
	}
}
