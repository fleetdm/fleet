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

var apiTokenPattern = regexp.MustCompile(`The API token for your new user is:\s*(\S+)`)

func printNextSteps(w io.Writer, outputDir string) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Next steps:")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  1. Create a GitHub repository and push this directory to it:")
	fmt.Fprintf(w, "       cd %s\n", outputDir)
	fmt.Fprintln(w, "       gh repo create <owner>/<repo> --private --source=. --remote=origin --push")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  2. Create a Fleet GitOps user and get an API token:")
	fmt.Fprintln(w, "       fleetctl user create --name GitOps --email gitops@example.com \\")
	fmt.Fprintln(w, "         --password <password> --global-role gitops --api-only")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  3. Add the required secrets to your GitHub repository:")
	fmt.Fprintln(w, "       gh secret set FLEET_URL --repo <owner>/<repo>")
	fmt.Fprintln(w, "       gh secret set FLEET_API_TOKEN --repo <owner>/<repo>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "     Or add them in the GitHub UI under Settings > Secrets and variables > Actions.")
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

			if orgName == "" {
				prompt := promptui.Prompt{
					Label:   "Organization name",
					Default: "My organization",
				}
				result, err := prompt.Run()
				if err != nil {
					return fmt.Errorf("prompt failed: %w", err)
				}
				orgName = result
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

			fmt.Fprintf(c.App.Writer, "Created new Fleet GitOps repository at %s\n", outputDir)
			fmt.Fprintf(c.App.Writer, "Organization name: %s\n", orgName)

			// Check if the directory already had a git repo before we started.
			hadExistingRepo := false
			if _, err := os.Stat(filepath.Join(outputDir, ".git")); err == nil {
				hadExistingRepo = true
			}

			stdout := c.App.Writer
			stderr := c.App.ErrWriter

			if _, err := exec.LookPath("git"); err == nil {
				gitInitArgs := []string{"init", outputDir}
				if !hadExistingRepo {
					gitInitArgs = []string{"init", "-b", "main", outputDir}
				}
				if out, err := exec.Command("git", gitInitArgs...).CombinedOutput(); err != nil {
					fmt.Fprintf(stderr, "Error running git: %s\n", strings.TrimSpace(string(out)))
				} else {
					// Stage all files and create initial commit.
					for _, args := range [][]string{
						{"-C", outputDir, "add", "."},
						{"-C", outputDir, "commit", "-m", "Created Fleet GitOps structure"},
					} {
						if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
							fmt.Fprintf(stderr, "Error running git: %s\n", strings.TrimSpace(string(out)))
							break
						}
					}
				}
			} else {
				fmt.Fprintln(stderr, "Git not found in PATH; skipping git init.\nMake sure git is installed, and initialize the repository later by running 'git init' in the output directory.")
			}

			// Skip GitHub setup if this was a pre-existing repo.
			if hadExistingRepo {
				return nil
			}

			_, ghErr := exec.LookPath("gh")
			if ghErr != nil {
				// gh not installed — print manual next steps.
				printNextSteps(stdout, outputDir)
				return nil
			}

			ghSetupPrompt := promptui.Prompt{
				Label:     "The GitHub CLI (gh) is installed. Set up your Fleet GitOps repo on GitHub",
				IsConfirm: true,
			}
			if _, err := ghSetupPrompt.Run(); err != nil {
				printNextSteps(stdout, outputDir)
				return nil
			}

			// Build the list of GitHub accounts (personal + orgs) the user can push to.
			personalLogin := ""
			if out, err := exec.Command("gh", "api", "user", "--jq", ".login").CombinedOutput(); err == nil {
				personalLogin = strings.TrimSpace(string(out))
			}
			var orgs []string
			if out, err := exec.Command("gh", "org", "list").CombinedOutput(); err == nil {
				for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						orgs = append(orgs, line)
					}
				}
			}

			// Build the account list for selection.
			var accounts []string
			if personalLogin != "" {
				accounts = append(accounts, personalLogin)
			}
			accounts = append(accounts, orgs...)

			var repoOwner string
			if len(accounts) == 0 {
				ownerPrompt := promptui.Prompt{
					Label: "GitHub owner (user or org)",
				}
				result, err := ownerPrompt.Run()
				if err != nil || result == "" {
					return fmt.Errorf("repository owner is required")
				}
				repoOwner = result
			} else {
				// Label personal account for clarity.
				items := make([]string, len(accounts))
				for i, acct := range accounts {
					if i == 0 && personalLogin != "" {
						items[i] = acct + " (personal)"
					} else {
						items[i] = acct
					}
				}
				sel := promptui.Select{
					Label: "Where would you like to create the repository",
					Items: items,
				}
				idx, _, err := sel.Run()
				if err != nil {
					return fmt.Errorf("selection cancelled")
				}
				repoOwner = accounts[idx]
			}

			repoNamePrompt := promptui.Prompt{
				Label:   "Repository name",
				Default: "fleet-gitops",
			}
			repoShortName, err := repoNamePrompt.Run()
			if err != nil {
				return fmt.Errorf("prompt failed: %w", err)
			}
			repoName := repoOwner + "/" + repoShortName

			fleetURLPrompt := promptui.Prompt{
				Label: "Fleet server URL (e.g. https://fleet.example.com)",
			}
			fleetURL, err := fleetURLPrompt.Run()
			if err != nil || fleetURL == "" {
				return fmt.Errorf("Fleet server URL is required")
			}

			gitopsNamePrompt := promptui.Prompt{
				Label:   "GitOps user name",
				Default: "GitOps",
			}
			gitopsName, err := gitopsNamePrompt.Run()
			if err != nil {
				return fmt.Errorf("prompt failed: %w", err)
			}

			gitopsEmailPrompt := promptui.Prompt{
				Label: "GitOps user email",
			}
			gitopsEmail, err := gitopsEmailPrompt.Run()
			if err != nil || gitopsEmail == "" {
				return fmt.Errorf("GitOps user email is required")
			}

			gitopsPasswordPrompt := promptui.Prompt{
				Label: "GitOps user password",
				Mask:  '*',
			}
			gitopsPassword, err := gitopsPasswordPrompt.Run()
			if err != nil || gitopsPassword == "" {
				return fmt.Errorf("GitOps user password is required")
			}

			// Create the GitOps user via fleetctl.
			fleetctlPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("finding fleetctl path: %w", err)
			}

			out, err := exec.Command(
				fleetctlPath, "user", "create",
				"--name", gitopsName,
				"--email", gitopsEmail,
				"--password", gitopsPassword,
				"--global-role", "gitops",
				"--api-only",
			).CombinedOutput()
			if err != nil {
				return fmt.Errorf("creating GitOps user: %s", strings.TrimSpace(string(out)))
			}

			// Parse the API token from the output.
			matches := apiTokenPattern.FindSubmatch(out)
			if len(matches) < 2 {
				return fmt.Errorf("could not find API token in fleetctl output: %s", strings.TrimSpace(string(out)))
			}
			apiToken := string(matches[1])

			// Create the GitHub repo.
			if out, err := exec.Command("gh", "repo", "create", repoName, "--private", "--source", outputDir, "--remote", "origin").CombinedOutput(); err != nil {
				outStr := strings.TrimSpace(string(out))
				if !strings.Contains(strings.ToLower(outStr), "already exists") {
					return fmt.Errorf("creating GitHub repository: %s", outStr)
				}
				// Repo already exists — offer to use it.
				existsPrompt := promptui.Prompt{
					Label:     fmt.Sprintf("Repository %s already exists. Use it anyway", repoName),
					IsConfirm: true,
				}
				if _, err := existsPrompt.Run(); err != nil {
					return fmt.Errorf("aborted")
				}
				// Add the remote manually since gh didn't do it.
				if out, err := exec.Command("git", "-C", outputDir, "remote", "add", "origin", fmt.Sprintf("https://github.com/%s.git", repoName)).CombinedOutput(); err != nil {
					// If remote already exists, update it instead.
					if strings.Contains(string(out), "already exists") {
						if out, err := exec.Command("git", "-C", outputDir, "remote", "set-url", "origin", fmt.Sprintf("https://github.com/%s.git", repoName)).CombinedOutput(); err != nil {
							return fmt.Errorf("setting remote URL: %s", strings.TrimSpace(string(out)))
						}
					} else {
						return fmt.Errorf("adding remote: %s", strings.TrimSpace(string(out)))
					}
				}
			}

			// Set repository secrets.
			for _, secret := range []struct{ name, value string }{
				{"FLEET_URL", fleetURL},
				{"FLEET_API_TOKEN", apiToken},
			} {
				cmd := exec.Command("gh", "secret", "set", secret.name, "--repo", repoName)
				cmd.Stdin = strings.NewReader(secret.value)
				if out, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("setting secret %s: %s", secret.name, strings.TrimSpace(string(out)))
				}
			}

			// Push to GitHub.
			if out, err := exec.Command("git", "-C", outputDir, "push", "-u", "origin", "main").CombinedOutput(); err != nil {
				return fmt.Errorf("pushing to GitHub: %s", strings.TrimSpace(string(out)))
			}

			fmt.Fprintf(stdout, "\nYour Fleet GitOps repo is ready at https://github.com/%s\n", repoName)

			return nil
		},
	}
}
