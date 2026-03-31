// Package startertemplates provides the embedded GitOps template files used by
// both `fleetctl new` (to scaffold a repo on disk) and the server setup flow
// (to apply the starter library to a new Fleet instance).
package startertemplates

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"
)

//go:embed all:templates
var templateFS embed.FS

const templateRoot = "templates"

// RenderTemplate renders a single template file's content with the given variables.
// Templates use <%= and %> as delimiters.
func RenderTemplate(content []byte, vars map[string]string) ([]byte, error) {
	tmpl, err := template.New("").Delims("<%=", "%>").Parse(string(content))
	if err != nil {
		return nil, err
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// TemplateVars returns the template variable map for the given org name,
// handling YAML escaping.
func TemplateVars(orgName string) (map[string]string, error) {
	yamlOrgName, err := yaml.Marshal(orgName)
	if err != nil {
		return nil, fmt.Errorf("marshaling org name: %w", err)
	}
	return map[string]string{
		"org_name": strings.TrimSpace(string(yamlOrgName)),
	}, nil
}

// RenderToTempDir renders all templates to a temporary directory and returns
// the path. The caller is responsible for cleaning up (os.RemoveAll).
func RenderToTempDir(orgName string) (string, error) {
	vars, err := TemplateVars(orgName)
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "fleet-starter-*")
	if err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}

	if err := renderToDir(tempDir, vars); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}
	return tempDir, nil
}

// RenderToDir renders all templates into the given output directory.
func RenderToDir(outputDir, orgName string) error {
	vars, err := TemplateVars(orgName)
	if err != nil {
		return err
	}
	return renderToDir(outputDir, vars)
}

func renderToDir(outputDir string, vars map[string]string) error {
	return fs.WalkDir(templateFS, templateRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(templateRoot, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// Strip .template. from filenames (e.g. foo.template.yml -> foo.yml).
		relPath = strings.Replace(relPath, ".template.", ".", 1)
		outPath := filepath.Join(outputDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(outPath, 0o755)
		}

		content, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", path, err)
		}

		content, err = RenderTemplate(content, vars)
		if err != nil {
			return fmt.Errorf("rendering template %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}

		return os.WriteFile(outPath, content, 0o644)
	})
}
