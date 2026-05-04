package fleetctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

const testAppVersion = "fleet-v4.83.0"

// runNewCommand runs the "new" command with the given args and returns stdout and any error.
func runNewCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := &strings.Builder{}
	app := &cli.App{
		Name:      "fleetctl",
		Version:   testAppVersion,
		Writer:    buf,
		ErrWriter: buf,
		Commands:  []*cli.Command{newCommand()},
	}
	cliArgs := append([]string{"fleetctl", "new"}, args...)
	err := app.Run(cliArgs)
	return buf.String(), err
}

func TestNewBasicFileStructure(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	output, err := runNewCommand(t, "--org-name", `Acme "Corp" \ Inc`, "--dir", outDir)
	require.NoError(t, err, output)

	t.Run("has expected files", func(t *testing.T) {
		// Spot-check key files exist.
		expectedFiles := []string{
			"default.yml",
			".gitignore",
			".github/workflows/workflow.yml",
			".github/fleet-gitops/action.yml",
			".gitlab-ci.yml",
			"README.md",
			"fleets/workstations.yml",
			"labels/apple-silicon-macos-hosts.yml",
			"platforms/macos/policies/all-software-updates-installed.yml",
		}
		for _, f := range expectedFiles {
			path := filepath.Join(outDir, f)
			_, err := os.Stat(path)
			assert.NoError(t, err, "expected file %s to exist", f)
		}
	})

	t.Run("strips .template. from output filenames", func(t *testing.T) {
		// .template. should be stripped from output filenames.
		err = filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			assert.NotContains(t, info.Name(), ".template.", "file %s should not contain .template.", path)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("replaces and escapes org_name template var", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(outDir, "default.yml"))
		require.NoError(t, err)
		assert.Contains(t, string(content), `Acme "Corp" \ Inc`)
		assert.NotContains(t, string(content), "<%=")

		// Verify the output is valid YAML that round-trips correctly.
		var parsed map[string]any
		require.NoError(t, yaml.Unmarshal(content, &parsed))
		orgSettings, _ := parsed["org_settings"].(map[string]any)
		orgInfo, _ := orgSettings["org_info"].(map[string]any)
		assert.Equal(t, `Acme "Corp" \ Inc`, orgInfo["org_name"])
	})
}

func TestNewOrgNameYAMLQuoting(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	// A colon followed by a space is special in YAML, so yaml.Marshal
	// wraps the value in quotes to produce valid output.
	_, err := runNewCommand(t, "--org-name", "Ops: IT & Security", "--dir", outDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(outDir, "default.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), `'Ops: IT & Security'`)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(content, &parsed))
	orgSettings, _ := parsed["org_settings"].(map[string]any)
	orgInfo, _ := orgSettings["org_info"].(map[string]any)
	assert.Equal(t, "Ops: IT & Security", orgInfo["org_name"])
}

func TestNewTemplateStripping(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	_, err := runNewCommand(t, "--org-name", "Test", "--dir", outDir)
	require.NoError(t, err)
}

func TestNewDirFlag(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "custom-dir")

	output, err := runNewCommand(t, "--org-name", "Test", "--dir", outDir)
	require.NoError(t, err)
	assert.Contains(t, output, "custom-dir")

	_, err = os.Stat(filepath.Join(outDir, "default.yml"))
	assert.NoError(t, err)
}

func TestNewExistingDirWithoutForce(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "existing")
	require.NoError(t, os.Mkdir(outDir, 0o755))

	_, err := runNewCommand(t, "--org-name", "Test", "--dir", outDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.Contains(t, err.Error(), "--force")
	// Verify no default.yml was created.
	_, err = os.Stat(filepath.Join(outDir, "default.yml"))
	assert.Error(t, err)
}

func TestNewExistingDirWithForce(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "existing")
	require.NoError(t, os.Mkdir(outDir, 0o755))

	_, err := runNewCommand(t, "--org-name", "Test", "--dir", outDir, "--force")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(outDir, "default.yml"))
	assert.NoError(t, err)
}

func TestNewOrgNameValidation(t *testing.T) {
	tests := []struct {
		name        string
		orgName     string
		wantErr     string // empty means no error expected
		checkOutput func(t *testing.T, outDir string)
	}{
		{
			name:    "only control characters",
			orgName: "\x01\x02\x03",
			wantErr: "organization name is required",
		},
		{
			name:    "too long",
			orgName: strings.Repeat("a", 256),
			wantErr: "255 characters",
		},
		{
			name:    "at max length",
			orgName: strings.Repeat("a", 255),
		},
		{
			name:    "control characters stripped",
			orgName: "ACME\x00Corp",
			checkOutput: func(t *testing.T, outDir string) {
				content, err := os.ReadFile(filepath.Join(outDir, "default.yml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), "ACMECorp")
				assert.NotContains(t, string(content), "\x00")
			},
		},
		{
			name:    "only whitespace",
			orgName: "   ",
			wantErr: "organization name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := filepath.Join(t.TempDir(), "out")
			_, err := runNewCommand(t, "--org-name", tt.orgName, "--dir", outDir)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tt.checkOutput != nil {
				tt.checkOutput(t, outDir)
			}
		})
	}
}

func TestNewOutputMessages(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	output, err := runNewCommand(t, "--org-name", "Test Org", "--dir", outDir)
	require.NoError(t, err)
	assert.Contains(t, output, "Created new Fleet GitOps repository")
	assert.Contains(t, output, "Organization name: Test Org")
	assert.Contains(t, output, "Next steps:")
}

func TestRenderTemplate(t *testing.T) {
	vars := map[string]string{
		"name":    "Fleet",
		"version": "4.83.0",
	}

	t.Run("replaces known vars", func(t *testing.T) {
		result, err := renderTemplate([]byte(`app: <%= .name %> v<%= .version %>`), vars)
		require.NoError(t, err)
		assert.Equal(t, "app: Fleet v4.83.0", string(result))
	})

	t.Run("unknown vars produce no value", func(t *testing.T) {
		result, err := renderTemplate([]byte(`<%= .unknown %>`), vars)
		require.NoError(t, err)
		assert.Equal(t, "<no value>", string(result))
	})

	t.Run("handles no vars", func(t *testing.T) {
		result, err := renderTemplate([]byte("no vars here"), vars)
		require.NoError(t, err)
		assert.Equal(t, "no vars here", string(result))
	})
}
