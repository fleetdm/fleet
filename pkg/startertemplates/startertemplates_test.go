package startertemplates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplate(t *testing.T) {
	t.Run("replaces known vars", func(t *testing.T) {
		vars := map[string]string{"name": "Fleet", "version": "4.83.0"}
		result, err := RenderTemplate([]byte(`app: <%= .name %> v<%= .version %>`), vars)
		require.NoError(t, err)
		assert.Equal(t, "app: Fleet v4.83.0", string(result))
	})

	t.Run("no vars in content", func(t *testing.T) {
		result, err := RenderTemplate([]byte("plain text"), map[string]string{"x": "y"})
		require.NoError(t, err)
		assert.Equal(t, "plain text", string(result))
	})
}

func TestTemplateVars(t *testing.T) {
	t.Run("simple name", func(t *testing.T) {
		vars, err := TemplateVars("My Org")
		require.NoError(t, err)
		assert.Equal(t, "My Org", vars["org_name"])
	})

	t.Run("name with special YAML chars", func(t *testing.T) {
		vars, err := TemplateVars("Ops: IT & Security")
		require.NoError(t, err)
		// Should be YAML-quoted to be safe.
		assert.Contains(t, vars["org_name"], "Ops: IT & Security")
	})
}

func TestRenderToTempDir(t *testing.T) {
	dir, err := RenderToTempDir("Test Org")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	t.Run("creates default.yml", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(dir, "default.yml"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "Test Org")
		assert.NotContains(t, string(content), "<%=")
	})

	t.Run("default.yml is valid YAML", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(dir, "default.yml"))
		require.NoError(t, err)
		var parsed map[string]any
		require.NoError(t, yaml.Unmarshal(content, &parsed))
		orgSettings, _ := parsed["org_settings"].(map[string]any)
		orgInfo, _ := orgSettings["org_info"].(map[string]any)
		assert.Equal(t, "Test Org", orgInfo["org_name"])
	})

	t.Run("creates fleet files", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, "fleets", "workstations.yml"))
		assert.NoError(t, err)
		_, err = os.Stat(filepath.Join(dir, "fleets", "personal-mobile-devices.yml"))
		assert.NoError(t, err)
	})

	t.Run("creates label files", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, "labels", "apple-silicon-macos-hosts.yml"))
		assert.NoError(t, err)
	})

	t.Run("strips .template. from filenames", func(t *testing.T) {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			assert.NotContains(t, info.Name(), ".template.")
			return nil
		})
		require.NoError(t, err)
	})
}

func TestRenderToDir(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "output")
	err := RenderToDir(outDir, "My Corp")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(outDir, "default.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "My Corp")
}
