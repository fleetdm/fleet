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

const (
	providerGitHub = "GitHub"
	providerGitLab = "GitLab"
)

func printNextSteps(w io.Writer, outputDir, provider string) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Next steps:")
	fmt.Fprintln(w, "")

	switch provider {
	case providerGitLab:
		fmt.Fprintln(w, "  1. Create a GitLab repository and push this directory to it:")
		fmt.Fprintf(w, "       cd %s\n", outputDir)
		fmt.Fprintln(w, "       glab repo create <name> --private")
		fmt.Fprintln(w, "       git remote add origin <repo-url>")
		fmt.Fprintln(w, "       git push -u origin main")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  2. Create a Fleet GitOps user and get an API token:")
		fmt.Fprintln(w, "       fleetctl user create --name GitOps --email gitops@example.com \\")
		fmt.Fprintln(w, "         --password <password> --global-role gitops --api-only")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  3. Add the required CI/CD variables to your GitLab project:")
		fmt.Fprintln(w, "       glab variable set FLEET_URL --repo <owner>/<repo>")
		fmt.Fprintln(w, "       glab variable set FLEET_API_TOKEN --repo <owner>/<repo> --mask")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "     Or add them in the GitLab UI under Settings > CI/CD > Variables.")
	case providerGitHub:
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
	default:
		fmt.Fprintln(w, "  1. Create a repository on GitLab or GitHub and push this directory to it.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  2. Create a Fleet GitOps user and get an API token:")
		fmt.Fprintln(w, "       fleetctl user create --name GitOps --email gitops@example.com \\")
		fmt.Fprintln(w, "         --password <password> --global-role gitops --api-only")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "  3. Add the required CI/CD variables (FLEET_URL and FLEET_API_TOKEN) to your GitLab or GitHub project.")
	}
}

func newCommand() *cli.Command {
	var (
		orgName   string
		outputDir string
		force     bool
		extra     bool
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
			&cli.BoolFlag{
				Name:        "extra",
				Usage:       "[Experimental] Interactively set up git repo, GitHub/GitLab remote, and Fleet GitOps user",
				Destination: &extra,
				Hidden:      true,
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

			if !extra {
				printNextSteps(c.App.Writer, outputDir, "")
				return nil
			}

			// --extra: interactively set up git, GitHub remote, and Fleet GitOps user.

			// Check if the directory already had a git repo before we started.
			hadExistingRepo := false
			if _, err := os.Stat(filepath.Join(outputDir, ".git")); err == nil {
				hadExistingRepo = true
			}

			stdout := c.App.Writer
			stderr := c.App.ErrWriter

			if _, err := exec.LookPath("git"); err == nil {
				gitInitArgs := []string{"init", outputDir}
				// Initialize git repo if necessary.
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

			// Skip remote setup if this was a pre-existing repo.
			if hadExistingRepo {
				printNextSteps(stdout, outputDir, "")
			}

			// Ask which provider the user wants to push to.
			providerSel := promptui.Select{
				Label: "Where will you be hosting your GitOps repository",
				Items: []string{providerGitHub, providerGitLab},
			}
			_, provider, err := providerSel.Run()
			if err != nil {
				return fmt.Errorf("selection cancelled")
			}

			// Determine the CLI tool for this provider.
			var cliTool string
			switch provider {
			case providerGitHub:
				cliTool = "gh"
			case providerGitLab:
				cliTool = "glab"
			}

			if _, err := exec.LookPath(cliTool); err != nil {
				printNextSteps(stdout, outputDir, provider)
				return nil
			}

			setupPrompt := promptui.Prompt{
				Label:     fmt.Sprintf("The %s CLI (%s) is installed. Set up your Fleet GitOps repo on %s", provider, cliTool, provider),
				IsConfirm: true,
			}
			if _, err := setupPrompt.Run(); err != nil {
				printNextSteps(stdout, outputDir, provider)
				return nil
			}

			// Prompt for the repo owner/group and name.
			repoOwner, err := promptRepoOwner(provider)
			if err != nil {
				return err
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

			// Prompt for Fleet server URL.
			fleetURLPrompt := promptui.Prompt{
				Label: "Fleet server URL (e.g. https://fleet.example.com)",
			}
			fleetURL, err := fleetURLPrompt.Run()
			if err != nil || fleetURL == "" {
				return fmt.Errorf("Fleet server URL is required")
			}

			// Get the Fleet API token.
			apiToken, err := promptAPIToken(stdout, fleetURL)
			if err != nil {
				return err
			}

			// Create the repo and configure the remote.
			if err := createRepoAndRemote(provider, repoName, outputDir); err != nil {
				return err
			}

			// Set secrets/variables on the repo.
			if err := setRepoSecrets(provider, repoName, fleetURL, apiToken); err != nil {
				return err
			}

			// Push to remote.
			if out, err := exec.Command("git", "-C", outputDir, "push", "-u", "origin", "main").CombinedOutput(); err != nil {
				return fmt.Errorf("pushing to remote: %s", strings.TrimSpace(string(out)))
			}

			repoURL := fmt.Sprintf("https://github.com/%s", repoName)
			if provider == providerGitLab {
				repoURL = fmt.Sprintf("https://gitlab.com/%s", repoName)
			}
			fmt.Fprintf(stdout, "\nYour Fleet GitOps repo is ready at %s\n", repoURL)

			return nil
		},
	}
}

// promptRepoOwner asks the user to select or enter a repo owner/group using
// the appropriate CLI tool for the given provider.
func promptRepoOwner(provider string) (string, error) {
	switch provider {
	case providerGitHub:
		// Try to fetch accounts from gh.
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

		var accounts []string
		if personalLogin != "" {
			accounts = append(accounts, personalLogin)
		}
		accounts = append(accounts, orgs...)

		if len(accounts) == 0 {
			p := promptui.Prompt{Label: "GitHub owner (user or org)"}
			result, err := p.Run()
			if err != nil || result == "" {
				return "", fmt.Errorf("repository owner is required")
			}
			return result, nil
		}

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
			return "", fmt.Errorf("selection cancelled")
		}
		return accounts[idx], nil

	case providerGitLab:
		// glab doesn't have an org list equivalent, so just prompt.
		p := promptui.Prompt{Label: "GitLab group or username"}
		result, err := p.Run()
		if err != nil || result == "" {
			return "", fmt.Errorf("repository owner is required")
		}
		return result, nil

	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

// promptAPIToken either collects an existing API token or creates a new GitOps
// user via fleetctl and returns the token.
func promptAPIToken(stdout io.Writer, fleetURL string) (string, error) {
	tokenSel := promptui.Select{
		Label: "Do you have a Fleet API token for GitOps",
		Items: []string{
			"Yes, I have an API token",
			"No, create a new GitOps user for me",
		},
	}
	tokenChoice, _, err := tokenSel.Run()
	if err != nil {
		return "", fmt.Errorf("selection cancelled")
	}

	fleetctlPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding fleetctl path: %w", err)
	}

	if tokenChoice == 0 {
		tokenPrompt := promptui.Prompt{
			Label: "Fleet API token",
			Mask:  '*',
		}
		apiToken, err := tokenPrompt.Run()
		if err != nil || apiToken == "" {
			return "", fmt.Errorf("API token is required")
		}
		return apiToken, nil
	}

	// Need to create a GitOps user. First ensure fleetctl is configured
	// and logged in to the right server.
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".fleet", "config")

	needsLogin := true
	if addr, err := getConfigValue(configPath, "default", "address"); err == nil {
		if addrStr, ok := addr.(string); ok && strings.TrimRight(addrStr, "/") == strings.TrimRight(fleetURL, "/") {
			if tok, err := getConfigValue(configPath, "default", "token"); err == nil {
				if tokStr, ok := tok.(string); ok && tokStr != "" {
					needsLogin = false
				}
			}
		}
	}

	if needsLogin {
		if err := setConfigValue(configPath, "default", "address", strings.TrimRight(fleetURL, "/")); err != nil {
			return "", fmt.Errorf("setting Fleet address in config: %w", err)
		}

		fmt.Fprintln(stdout, "\nLog in to Fleet so we can create the GitOps user.")
		loginCmd := exec.Command(fleetctlPath, "login")
		loginCmd.Stdin = os.Stdin
		loginCmd.Stdout = os.Stdout
		loginCmd.Stderr = os.Stderr
		if err := loginCmd.Run(); err != nil {
			return "", fmt.Errorf("fleetctl login failed: %w", err)
		}
	}

	gitopsNamePrompt := promptui.Prompt{
		Label:   "GitOps user name",
		Default: "GitOps",
	}
	gitopsName, err := gitopsNamePrompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed: %w", err)
	}

	gitopsEmailPrompt := promptui.Prompt{
		Label: "GitOps user email",
	}
	gitopsEmail, err := gitopsEmailPrompt.Run()
	if err != nil || gitopsEmail == "" {
		return "", fmt.Errorf("GitOps user email is required")
	}

	gitopsPasswordPrompt := promptui.Prompt{
		Label: "GitOps user password",
		Mask:  '*',
	}
	gitopsPassword, err := gitopsPasswordPrompt.Run()
	if err != nil || gitopsPassword == "" {
		return "", fmt.Errorf("GitOps user password is required")
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
		return "", fmt.Errorf("creating GitOps user: %s", strings.TrimSpace(string(out)))
	}

	matches := apiTokenPattern.FindSubmatch(out)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find API token in fleetctl output: %s", strings.TrimSpace(string(out)))
	}
	return string(matches[1]), nil
}

// createRepoAndRemote creates a remote repository and configures the git
// remote in the output directory.
func createRepoAndRemote(provider, repoName, outputDir string) error {
	switch provider {
	case providerGitHub:
		if out, err := exec.Command("gh", "repo", "create", repoName, "--private", "--source", outputDir, "--remote", "origin").CombinedOutput(); err != nil {
			outStr := strings.TrimSpace(string(out))
			if !strings.Contains(strings.ToLower(outStr), "already exists") {
				return fmt.Errorf("creating GitHub repository: %s", outStr)
			}
			existsPrompt := promptui.Prompt{
				Label:     fmt.Sprintf("Repository %s already exists. Use it anyway", repoName),
				IsConfirm: true,
			}
			if _, err := existsPrompt.Run(); err != nil {
				return fmt.Errorf("aborted")
			}
			remoteURL := fmt.Sprintf("https://github.com/%s.git", repoName)
			if err := setGitRemote(outputDir, remoteURL); err != nil {
				return err
			}
		}

	case providerGitLab:
		if out, err := exec.Command("glab", "repo", "create", repoName, "--private", "--defaultBranch", "main").CombinedOutput(); err != nil {
			outStr := strings.TrimSpace(string(out))
			if !strings.Contains(strings.ToLower(outStr), "already exists") {
				return fmt.Errorf("creating GitLab repository: %s", outStr)
			}
			existsPrompt := promptui.Prompt{
				Label:     fmt.Sprintf("Repository %s already exists. Use it anyway", repoName),
				IsConfirm: true,
			}
			if _, err := existsPrompt.Run(); err != nil {
				return fmt.Errorf("aborted")
			}
		}
		remoteURL := fmt.Sprintf("https://gitlab.com/%s.git", repoName)
		if err := setGitRemote(outputDir, remoteURL); err != nil {
			return err
		}
	}
	return nil
}

// setGitRemote adds or updates the "origin" remote for the given directory.
func setGitRemote(outputDir, remoteURL string) error {
	if out, err := exec.Command("git", "-C", outputDir, "remote", "add", "origin", remoteURL).CombinedOutput(); err != nil {
		if strings.Contains(string(out), "already exists") {
			if out, err := exec.Command("git", "-C", outputDir, "remote", "set-url", "origin", remoteURL).CombinedOutput(); err != nil {
				return fmt.Errorf("setting remote URL: %s", strings.TrimSpace(string(out)))
			}
		} else {
			return fmt.Errorf("adding remote: %s", strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// setRepoSecrets sets the Fleet URL and API token as secrets (GitHub) or
// CI/CD variables (GitLab) on the remote repository.
func setRepoSecrets(provider, repoName, fleetURL, apiToken string) error {
	secrets := []struct{ name, value string }{
		{"FLEET_URL", fleetURL},
		{"FLEET_API_TOKEN", apiToken},
	}

	switch provider {
	case providerGitHub:
		for _, s := range secrets {
			cmd := exec.Command("gh", "secret", "set", s.name, "--repo", repoName)
			cmd.Stdin = strings.NewReader(s.value)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("setting secret %s: %s", s.name, strings.TrimSpace(string(out)))
			}
		}
	case providerGitLab:
		for _, s := range secrets {
			args := []string{"variable", "set", s.name, "--repo", repoName, "--value", s.value}
			if s.name == "FLEET_API_TOKEN" {
				args = append(args, "--mask")
			}
			if out, err := exec.Command("glab", args...).CombinedOutput(); err != nil {
				return fmt.Errorf("setting variable %s: %s", s.name, strings.TrimSpace(string(out)))
			}
		}
	}
	return nil
}
