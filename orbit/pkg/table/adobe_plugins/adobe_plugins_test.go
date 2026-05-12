package adobe_plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCEPPlugin(t *testing.T) {
	t.Parallel()

	tbl := &adobePluginsTable{logger: zerolog.Nop()}

	t.Run("valid manifest", func(t *testing.T) {
		t.Parallel()
		pluginPath := filepath.Join("testdata", "cep_plugin")
		sp := scanPath{extensionType: "CEP"}

		row := tbl.parseCEPPlugin(pluginPath, sp)
		require.NotNil(t, row)

		assert.Equal(t, pluginPath, row[colPath])
		assert.Equal(t, "cep_plugin", row[colName])
		assert.Equal(t, "2.1.0", row[colVersion])
		assert.Equal(t, "Test Vendor", row[colVendor])
		assert.Equal(t, "com.example.test.plugin", row[colBundleID])
		assert.Contains(t, row[colHostApplication], "Photoshop")
		assert.Contains(t, row[colHostApplication], "Illustrator")
		assert.Equal(t, "CEP", row[colExtensionType])
	})

	t.Run("missing manifest falls back to dir name", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		pluginPath := filepath.Join(dir, "no_manifest_plugin")
		require.NoError(t, os.MkdirAll(pluginPath, 0o755))

		sp := scanPath{extensionType: "CEP", user: "testuser"}
		row := tbl.parseCEPPlugin(pluginPath, sp)
		require.NotNil(t, row)

		assert.Equal(t, "no_manifest_plugin", row[colName])
		assert.Equal(t, "CEP", row[colExtensionType])
		assert.Equal(t, "testuser", row[colUser])
		assert.Empty(t, row[colVersion])
		assert.Empty(t, row[colBundleID])
	})

	t.Run("malformed manifest falls back to dir name", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		pluginPath := filepath.Join(dir, "bad_manifest")
		require.NoError(t, os.MkdirAll(filepath.Join(pluginPath, "CSXS"), 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginPath, "CSXS", "manifest.xml"),
			[]byte("not valid xml {{{"),
			0o644,
		))

		sp := scanPath{extensionType: "CEP"}
		row := tbl.parseCEPPlugin(pluginPath, sp)
		require.NotNil(t, row)

		assert.Equal(t, "bad_manifest", row[colName])
		assert.Equal(t, "CEP", row[colExtensionType])
		assert.Empty(t, row[colVersion])
	})
}

func TestParseUXPPlugin(t *testing.T) {
	t.Parallel()

	tbl := &adobePluginsTable{logger: zerolog.Nop()}

	t.Run("valid manifest", func(t *testing.T) {
		t.Parallel()
		pluginPath := filepath.Join("testdata", "uxp_plugin")
		sp := scanPath{extensionType: "UXP"}

		row := tbl.parseUXPPlugin(pluginPath, sp)
		require.NotNil(t, row)

		assert.Equal(t, pluginPath, row[colPath])
		assert.Equal(t, "Test UXP Plugin", row[colName])
		assert.Equal(t, "3.0.1", row[colVersion])
		assert.Equal(t, "UXP Test Vendor", row[colVendor])
		assert.Equal(t, "com.example.uxp.plugin", row[colBundleID])
		assert.Contains(t, row[colHostApplication], "Photoshop")
		assert.Contains(t, row[colHostApplication], "XD")
		assert.Equal(t, "UXP", row[colExtensionType])
	})

	t.Run("missing manifest falls back to dir name", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		pluginPath := filepath.Join(dir, "some_uxp_ext")
		require.NoError(t, os.MkdirAll(pluginPath, 0o755))

		sp := scanPath{extensionType: "UXP", user: "alice"}
		row := tbl.parseUXPPlugin(pluginPath, sp)
		require.NotNil(t, row)

		assert.Equal(t, "some_uxp_ext", row[colName])
		assert.Equal(t, "alice", row[colUser])
		assert.Empty(t, row[colVersion])
	})

	t.Run("manifest with id but no name uses id", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		pluginPath := filepath.Join(dir, "id_only")
		require.NoError(t, os.MkdirAll(pluginPath, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginPath, "manifest.json"),
			[]byte(`{"id": "com.vendor.idonly", "version": "1.0"}`),
			0o644,
		))

		sp := scanPath{extensionType: "UXP"}
		row := tbl.parseUXPPlugin(pluginPath, sp)
		require.NotNil(t, row)

		assert.Equal(t, "com.vendor.idonly", row[colName])
		assert.Equal(t, "com.vendor.idonly", row[colBundleID])
		assert.Equal(t, "1.0", row[colVersion])
	})
}

func TestParseNativePlugin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fileName     string
		expectedName string
	}{
		{"macOS plugin bundle", "MyPlugin.plugin", "MyPlugin"},
		{"Photoshop filter 8bf", "CoolFilter.8bf", "CoolFilter"},
		{"After Effects plugin", "Effect.aex", "Effect"},
		{"Windows DLL plugin", "Plugin.dll", "Plugin"},
		{"no extension", "SomePlugin", "SomePlugin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			pluginPath := filepath.Join(dir, tt.fileName)

			f, err := os.Create(pluginPath)
			require.NoError(t, err)
			f.Close()

			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Len(t, entries, 1)

			sp := scanPath{extensionType: "native", hostApp: "Photoshop"}
			row := parseNativePlugin(pluginPath, entries[0], sp)
			require.NotNil(t, row)

			assert.Equal(t, tt.expectedName, row[colName])
			assert.Equal(t, "Photoshop", row[colHostApplication])
			assert.Equal(t, "native", row[colExtensionType])
		})
	}
}

func TestResolveHostApps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		codes    []string
		expected string
	}{
		{"single known code", []string{"PHXS"}, "Photoshop"},
		{"multiple codes", []string{"PHXS", "ILST"}, "Photoshop, Illustrator"},
		{"deduplicates same app", []string{"PHXS", "PHSP"}, "Photoshop"},
		{"unknown code passes through", []string{"UNKNOWN"}, "UNKNOWN"},
		{"mixed known and unknown", []string{"PPRO", "CUSTOM"}, "Premiere Pro, CUSTOM"},
		{"empty list", nil, ""},
		{"case insensitive", []string{"phxs"}, "Photoshop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := resolveHostApps(tt.codes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScanEntry(t *testing.T) {
	t.Parallel()

	tbl := &adobePluginsTable{logger: zerolog.Nop()}

	t.Run("CEP skips non-directory entries", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		f, err := os.Create(filepath.Join(dir, "notadir.txt"))
		require.NoError(t, err)
		f.Close()

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)

		sp := scanPath{extensionType: "CEP"}
		row := tbl.scanEntry(filepath.Join(dir, "notadir.txt"), entries[0], sp)
		assert.Nil(t, row)
	})

	t.Run("native skips hidden files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		f, err := os.Create(filepath.Join(dir, ".DS_Store"))
		require.NoError(t, err)
		f.Close()

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)

		sp := scanPath{extensionType: "native", hostApp: "Photoshop"}
		row := tbl.scanEntry(filepath.Join(dir, ".DS_Store"), entries[0], sp)
		assert.Nil(t, row)
	})
}
