package fleetctl

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

func newCommand() *cli.Command {
	var orgName string
	return &cli.Command{
		Name:      "new",
		Usage:     "Create a new Fleet GitOps repository structure",
		UsageText: "fleetctl new [options] [dir_name]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "org-name",
				Usage:       "The name of your organization",
				Destination: &orgName,
			},
		},
		Action: func(c *cli.Context) error {
			outputDir := c.Args().First()
			if outputDir == "" {
				outputDir = "it-and-security"
			}
			if _, err := os.Stat(outputDir); err == nil {
				return fmt.Errorf("%s already exists; please remove it or run from a different directory", outputDir)
			}

			if orgName == "" {
				fmt.Print(`Enter your organization name (default: "My organization"): `)
				reader := bufio.NewReader(os.Stdin)
				line, err := reader.ReadString('\n')
				if err != nil {
					orgName = "My organization"
				} else {
					orgName = strings.TrimSpace(line)
				}
				if orgName == "" {
					orgName = "My organization"
				}
			}

			vars := map[string]string{
				"org_name": orgName,
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

			fmt.Fprintf(c.App.Writer, "Created %s/ — Fleet GitOps directory structure\n", outputDir)
			fmt.Fprintf(c.App.Writer, "Organization name: %s\n", orgName)

			return nil
		},
	}
}
