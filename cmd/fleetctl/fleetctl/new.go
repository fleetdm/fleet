package fleetctl

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
)

//go:embed all:templates/new
var newTemplateFS embed.FS

var ejsTagPattern = regexp.MustCompile(`<%=\s*(\w+)\s*%>`)

func renderTemplate(content []byte, vars map[string]string) []byte {
	return ejsTagPattern.ReplaceAllFunc(content, func(match []byte) []byte {
		submatch := ejsTagPattern.FindSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		key := string(submatch[1])
		if val, ok := vars[key]; ok {
			return []byte(val)
		}
		return match
	})
}

// fleetVersionPattern matches "fleet-vX.Y.Z" and captures "X.Y.Z".
var fleetVersionPattern = regexp.MustCompile(`^fleet-v(\d+\.\d+\.\d+)$`)

// semverPattern matches a plain "X.Y.Z" (possibly with pre-release suffix).
var semverPattern = regexp.MustCompile(`^(\d+\.\d+\.\d+)`)

// resolveFleetctlVersion returns a semver version string for fleetctl.
// It tries to extract it from the app version (e.g. "fleet-v4.74.0" -> "4.74.0"),
// falling back to the latest published version on npm.
func resolveFleetctlVersion(appVersion string) string {
	// Try "fleet-vX.Y.Z" format.
	if m := fleetVersionPattern.FindStringSubmatch(appVersion); len(m) == 2 {
		return m[1]
	}
	// Try plain semver (possibly with pre-release/build suffix).
	if m := semverPattern.FindStringSubmatch(appVersion); len(m) == 2 {
		return m[1]
	}
	// Fall back to latest published version from npm.
	if out, err := exec.Command("npm", "view", "fleetctl", "version").CombinedOutput(); err == nil {
		if v := strings.TrimSpace(string(out)); semverPattern.MatchString(v) {
			return v
		}
	}
	return "latest"
}

func printNextSteps(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Next steps:")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  1. Create a repository on GitHub or GitLab and push this directory to it.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  2. Create a Fleet GitOps user and get an API token:")
	fmt.Fprintln(w, "       fleetctl user create --name GitOps --email gitops@example.com \\")
	fmt.Fprintln(w, "         --password <password> --global-role gitops --api-only")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  3. Add FLEET_URL and FLEET_API_TOKEN as secrets (GitHub) or CI/CD variables (GitLab).")
}

func newCommand() *cli.Command {
	var (
		orgName   string
		outputDir string
		force     bool
	)
	return &cli.Command{
		Name:      "new",
		Usage:     "Create a new Fleet GitOps repository structure",
		UsageText: "fleetctl new [options]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "org-name",
				Usage:       "The name of your organization",
				Destination: &orgName,
			},
			&cli.StringFlag{
				Name:        "dir",
				Aliases:     []string{"d"},
				Usage:       "Output directory path",
				Value:       "it-and-security",
				Destination: &outputDir,
			},
			&cli.BoolFlag{
				Name:        "force",
				Aliases:     []string{"f"},
				Usage:       "Write files into an existing directory",
				Destination: &force,
			},
		},
		Action: func(c *cli.Context) error {
			if !force {
				if _, err := os.Stat(outputDir); err == nil {
					return fmt.Errorf("%s already exists; use --force to write into an existing directory", outputDir)
				}
			}

			cleanOrgName := func(name string) string {
				return strings.Map(func(r rune) rune {
					if r < 0x20 || r == 0x7f {
						return -1 // strip control characters
					}
					return r
				}, strings.TrimSpace(name))
			}
			validateOrgName := func(name string) error {
				name = cleanOrgName(name)
				if name == "" {
					return fmt.Errorf("organization name is required")
				}
				if len(name) > 255 {
					return fmt.Errorf("organization name must be 255 characters or fewer")
				}
				return nil
			}
			if orgName == "" {
				prompt := promptui.Prompt{
					Label:    "Organization name",
					Default:  "My organization",
					Validate: func(s string) error { return validateOrgName(s) },
				}
				result, err := prompt.Run()
				if err != nil {
					return fmt.Errorf("prompt failed: %w", err)
				}
				orgName = cleanOrgName(result)
			} else {
				orgName = cleanOrgName(orgName)
				if err := validateOrgName(orgName); err != nil {
					return err
				}
			}

			// Escape for safe inclusion in double-quoted YAML strings.
			yamlOrgName := strings.ReplaceAll(orgName, `\`, `\\`)
			yamlOrgName = strings.ReplaceAll(yamlOrgName, `"`, `\"`)

			vars := map[string]string{
				"org_name":        yamlOrgName,
				"FleetctlVersion": resolveFleetctlVersion(c.App.Version),
			}

			templateRoot := "templates/new"

			err := fs.WalkDir(newTemplateFS, templateRoot, func(path string, d fs.DirEntry, err error) error {
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

				// Strip .template from filenames (e.g. foo.template.yml -> foo.yml).
				relPath = strings.Replace(relPath, ".template.", ".", 1)
				outPath := filepath.Join(outputDir, relPath)

				if d.IsDir() {
					return os.MkdirAll(outPath, 0o755)
				}

				content, err := newTemplateFS.ReadFile(path)
				if err != nil {
					return fmt.Errorf("reading template %s: %w", path, err)
				}

				content = renderTemplate(content, vars)

				if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
					return err
				}

				return os.WriteFile(outPath, content, 0o644)
			})
			if err != nil {
				return fmt.Errorf("creating GitOps directory structure: %w", err)
			}

			fmt.Fprintf(c.App.Writer, "Created new Fleet GitOps repository at %s\n", outputDir)
			fmt.Fprintf(c.App.Writer, "Organization name: %s\n", orgName)

			printNextSteps(c.App.Writer)

			return nil
		},
	}
}
