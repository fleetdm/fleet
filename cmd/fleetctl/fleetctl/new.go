package fleetctl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/startertemplates"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
)

func printNextSteps(w io.Writer, outputDir string) {
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
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "For more information, see the README.md file at %s\n", filepath.Join(outputDir, "README.md"))
	fmt.Fprintln(w, "")
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
					return errors.New("organization name is required")
				}
				if len(name) > 255 {
					return errors.New("organization name must be 255 characters or fewer")
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

			if err := startertemplates.RenderToDir(outputDir, orgName); err != nil {
				return fmt.Errorf("creating GitOps directory structure: %w", err)
			}

			fmt.Fprintf(c.App.Writer, "Created new Fleet GitOps repository at %s\n", outputDir)
			fmt.Fprintf(c.App.Writer, "Organization name: %s\n", orgName)

			printNextSteps(c.App.Writer, outputDir)

			return nil
		},
	}
}
