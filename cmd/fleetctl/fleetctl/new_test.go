package fleetctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestNewCreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	output, err := runNewCommand(t, "--org-name", "ACME Corp", "--dir", outDir)
	require.NoError(t, err, output)

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
}

func TestNewTemplateStripping(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	_, err := runNewCommand(t, "--org-name", "Test", "--dir", outDir)
	require.NoError(t, err)

	// .template. should be stripped from output filenames.
	err = filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		assert.NotContains(t, info.Name(), ".template.", "file %s should not contain .template.", path)
		return nil
	})
	require.NoError(t, err)
}

func TestNewOrgNameTemplating(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	_, err := runNewCommand(t, "--org-name", "ACME Corp", "--dir", outDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(outDir, "default.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "ACME Corp")
	assert.NotContains(t, string(content), "<%= org_name %>")
}

func TestNewFleetctlVersionTemplating(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	_, err := runNewCommand(t, "--org-name", "Test", "--dir", outDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(outDir, ".github/fleet-gitops/action.yml"))
	require.NoError(t, err)
	assert.NotContains(t, string(content), "<%= FleetctlVersion %>")
	// The version should be whatever resolveFleetctlVersion returns for the test app version.
	expected := resolveFleetctlVersion(testAppVersion)
	assert.Contains(t, string(content), expected)
}

func TestNewYAMLEscaping(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	_, err := runNewCommand(t, "--org-name", `Acme "Corp" \ Inc`, "--dir", outDir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(outDir, "default.yml"))
	require.NoError(t, err)
	// Quotes and backslashes should be escaped.
	assert.Contains(t, string(content), `Acme \"Corp\" \\ Inc`)
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
	dir := t.TempDir()

	t.Run("only control characters", func(t *testing.T) {
		outDir := filepath.Join(dir, "ctrl-only")
		_, err := runNewCommand(t, "--org-name", "\x01\x02\x03", "--dir", outDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "organization name is required")
	})

	t.Run("org name too long", func(t *testing.T) {
		outDir := filepath.Join(dir, "long")
		longName := strings.Repeat("a", 256)
		_, err := runNewCommand(t, "--org-name", longName, "--dir", outDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "255 characters")
	})

	t.Run("org name at max length", func(t *testing.T) {
		outDir := filepath.Join(dir, "maxlen")
		maxName := strings.Repeat("a", 255)
		_, err := runNewCommand(t, "--org-name", maxName, "--dir", outDir)
		require.NoError(t, err)
	})

	t.Run("org name with control characters", func(t *testing.T) {
		outDir := filepath.Join(dir, "ctrl")
		_, err := runNewCommand(t, "--org-name", "ACME\x00Corp", "--dir", outDir)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(outDir, "default.yml"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "ACMECorp")
		assert.NotContains(t, string(content), "\x00")
	})

	t.Run("org name only whitespace", func(t *testing.T) {
		outDir := filepath.Join(dir, "ws")
		_, err := runNewCommand(t, "--org-name", "   ", "--dir", outDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "organization name is required")
	})
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
		result := renderTemplate([]byte(`app: <%= name %> v<%= version %>`), vars)
		assert.Equal(t, "app: Fleet v4.83.0", string(result))
	})

	t.Run("leaves unknown vars", func(t *testing.T) {
		result := renderTemplate([]byte(`<%= unknown %>`), vars)
		assert.Equal(t, "<%= unknown %>", string(result))
	})

	t.Run("handles no vars", func(t *testing.T) {
		result := renderTemplate([]byte("no vars here"), vars)
		assert.Equal(t, "no vars here", string(result))
	})
}

func TestResolveFleetctlVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"fleet-v tag", "fleet-v4.74.0", "4.74.0"},
		{"plain semver", "4.74.0", "4.74.0"},
		{"semver with rc suffix", "4.74.0-rc.2503171200", "4.74.0"},
		{"semver with build metadata", "4.74.0+2503171200", "4.74.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveFleetctlVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveFleetctlVersionUnknown(t *testing.T) {
	// "unknown" doesn't match any pattern; falls back to npm or "latest".
	result := resolveFleetctlVersion("unknown")
	assert.NotEmpty(t, result)
}

