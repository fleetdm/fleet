# Zero to GitOps: set up Fleet GitOps and make changes with AI

This guide takes you from nothing to a Git repository that keeps your Fleet instance in sync. When you're done, every change to Fleet goes through a pull request. CI checks it with a dry run, a person reviews and merges it, and CI applies it. Then you'll make your first change with an AI agent, which writes the YAML and opens the pull request for you. The steps cover GitHub and GitLab, from macOS, Linux, or Windows.

## Prerequisites

- A Fleet instance and its URL, for example `https://fleet.example.com`.
- A Fleet user with the global admin role.
- Fleet Premium for the GitOps role and GitOps mode. On Fleet Free, give the GitOps user the global admin role instead, and skip the GitOps mode step.
- A GitHub or GitLab account where you can create a private repository and add CI secrets or variables.
- [Git](https://git-scm.com/downloads) installed on your computer.
- For the AI step: an AI coding agent. This guide uses [Claude Code](https://fleetdm.com/guides/use-claude-code-with-fleet). The repository also works with Codex, Cursor, GitHub Copilot, Gemini CLI, and Kilo Code.

> **Warning:** GitOps is declarative. The YAML in your repository is the whole configuration, so anything it doesn't list can be removed when CI applies it. For example, fleets that aren't defined in `fleets/` are deleted, and so are labels that a file doesn't list. If your Fleet instance already has configuration you want to keep, follow [Existing Fleet instance: export your configuration](#existing-fleet-instance-export-your-configuration), and run the [dry run](#dry-run-before-you-apply) before you push.

## Install fleetctl

`fleetctl` is Fleet's command-line tool. You use it to create the repository files and check them before CI applies them.

**macOS and Linux:** Run the install script:

```bash
curl -sSL https://fleetdm.com/resources/install-fleetctl.sh | bash
```

The script installs `fleetctl` in `~/.fleetctl`. If it tells you to add that folder to your `PATH`, run the `export` command it prints, and add the same line to your shell profile.

**Windows:** Download and run the fleetctl installer from [fleetdm.com/download](https://fleetdm.com/download). The installer adds `fleetctl` to your system `PATH`.

> **Note:** On Windows, close and reopen PowerShell, Windows Terminal, and VS Code after you install. Windows only applies the new `PATH` to programs you open after the installer finishes.

Confirm the install:

```bash
fleetctl --version
```

## Log in to Fleet

Point `fleetctl` at your Fleet instance, then log in with your admin account:

```bash
fleetctl config set --address https://fleet.example.com
fleetctl login
```

If your Fleet account uses single sign-on (SSO), `fleetctl login` can't use your password. Instead, go to **My account** in the Fleet UI, select **Get API token**, and set the token:

```bash
fleetctl config set --token <your-api-token>
```

## Create a GitOps user

CI needs its own API-only user with a token that doesn't expire. Create one with the GitOps role:

```bash
fleetctl user create --name GitOps --email gitops@example.com --global-role gitops --api-only
```

`fleetctl` shows the new user's API token once. Copy it somewhere safe. You'll add it to your Git provider as `FLEET_API_TOKEN`.

On Fleet Free, use `--global-role admin` instead. The GitOps role is only available in Fleet Premium.

## Create the repository files

Pick the path that matches your Fleet instance.

### New Fleet instance: start from a template

Run `fleetctl new` and enter your organization's name when it asks:

```bash
fleetctl new
```

This creates an `it-and-security` folder with:

- `default.yml` for organization-wide settings, and `fleets/` with one file per fleet. The template starts with two fleets: **💻 Workstations** and **📱🔐 Personal mobile devices**.
- `platforms/` folders for policies, reports, scripts, software, and configuration profiles, per platform.
- CI for GitHub (`.github/`) and GitLab (`.gitlab-ci.yml`).
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and a `fleet-gitops` skill, which tell AI agents how the repository works and how to validate changes.

### Existing Fleet instance: export your configuration

`fleetctl generate-gitops` writes your current Fleet configuration out as GitOps YAML. Export into its own folder, then copy in the CI and AI files from the template.

1. Export your configuration:

   ```bash
   fleetctl generate-gitops --dir it-and-security
   ```

2. Create the template in a separate folder:

   ```bash
   fleetctl new --dir starter
   ```

3. Copy the CI and AI files into your export:

   ```bash
   cp -R starter/.github starter/.gitlab-ci.yml starter/.agents starter/.claude starter/AGENTS.md starter/CLAUDE.md starter/GEMINI.md starter/.gitignore it-and-security/
   ```

4. Delete the `starter` folder.

> **Warning:** Don't run `fleetctl generate-gitops --force` inside a `fleetctl new` folder. It overwrites files but doesn't delete the template's example fleets, so your first sync would create **💻 Workstations** and **📱🔐 Personal mobile devices** as new fleets.

`generate-gitops` doesn't write secrets, such as enroll secrets or integration API keys, into the files. It leaves a comment in their place and prints the list of files and keys to fix. Replace each one with an environment variable reference, for example `$ENROLL_SECRET`, and add that variable to your Git provider alongside `FLEET_URL` and `FLEET_API_TOKEN`. See [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl) for its options and known gaps.

## Dry run before you apply

Check what GitOps would change before CI changes anything. From the `it-and-security` folder, set `FLEET_URL` and any other variables your files reference, then run a dry run:

```bash
cd it-and-security
export FLEET_URL=https://fleet.example.com
fleetctl gitops -f default.yml $(for f in fleets/*.yml; do echo -f "$f"; done) --delete-other-fleets --dry-run
```

On Windows, run the same commands in Git Bash, which comes with Git for Windows.

Read the output. Lines that start with `[-] would've deleted` or `[!] would've deleted` are things the first sync would remove. If something you need is on that list, add it to the YAML and run the dry run again. Continue when the output only shows changes you expect.

## Push the repository

1. Create an empty, private repository on [GitHub](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-new-repository) or [GitLab](https://docs.gitlab.com/user/project/). Don't add a README or other files.

2. Add `FLEET_URL` and `FLEET_API_TOKEN`, plus any variables from the export step:
   - **GitHub:** Add them as [repository secrets](https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions#creating-secrets-for-a-repository).
   - **GitLab:** Add them as [CI/CD variables](https://docs.gitlab.com/ci/variables/#define-a-cicd-variable-in-the-ui), and mark them as masked.

   Set `FLEET_URL` to your Fleet URL, starting with `https://` and with no trailing slash.

3. From the `it-and-security` folder, commit and push. Replace the remote URL with your repository's URL:

   ```bash
   git init -b main
   git add .
   git commit -m "Add Fleet GitOps configuration"
   git remote add origin https://github.com/your-org/it-and-security.git
   git push -u origin main
   ```

The push to `main` is your first sync. CI runs a dry run, and if it passes, applies the configuration to Fleet.

> **Note:** Add the secrets before you push. CI runs as soon as `main` receives a commit, and fails if `FLEET_URL` or `FLEET_API_TOKEN` is missing.

## Confirm CI is syncing

**GitHub:** Go to your repository's **Actions** tab and open the **Apply latest configuration to Fleet** run. If GitHub asks you to enable workflows, enable them, select the workflow, and click **Run workflow**.

The GitHub workflow runs when:

- You push to `main`. It applies the configuration.
- Someone opens or updates a pull request against `main`. It runs a dry run only, so reviewers see the result before merging.
- Every night at 6:00 AM UTC. It re-applies `main`, which undoes changes made outside Git.
- You start it from the **Actions** tab.

**GitLab:** Go to **Build > Pipelines** and open the latest pipeline. To keep Fleet in sync between commits, add a nightly [pipeline schedule](https://docs.gitlab.com/ci/pipelines/schedules/) on `main`. The repository's `README.md` has the exact settings.

## Turn on GitOps mode

The nightly sync overwrites changes made in the Fleet UI. GitOps mode stops people from making those changes in the first place. It's available in Fleet Premium.

1. In Fleet, go to **Settings > Integrations > Change management**.
2. Turn on GitOps mode.
3. In **Git repository URL**, enter your repository's URL, then save.

The Fleet UI now blocks edits to settings that GitOps manages and links to your repository. If you want to keep managing labels, software, or enroll secrets in the UI, add an exception for them on the same page. See [GitOps mode](https://fleetdm.com/guides/gitops-mode) for what stays editable.

## Make your first change with AI

From here on, you describe a change, an AI agent writes it, and your pull request process decides whether it ships. The agent never applies anything itself. The repository's `AGENTS.md` tells it to leave applying to CI after a person merges.

This example uses Claude Code. Other agents follow the same steps, because they read the same `AGENTS.md` and skill.

1. Open the repository in Claude Code. See [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet) to sign in and choose the folder.
2. Ask for a change in plain language. For example: "Create a branch and add a policy that checks FileVault is on for macOS hosts."
3. Claude Code uses the `fleet-gitops` skill to write the policy file under `platforms/macos/policies/`. It checks the query against Fleet's osquery schema and runs the skill's offline validator.
4. Review the change. Read the new file, and ask the agent to explain anything you don't follow.
5. Ask the agent to commit the change and open a pull request, or do it yourself.
6. Wait for the dry run on the pull request. It shows exactly what the change does to Fleet, or the error that stopped it.
7. Merge the pull request when the dry run passes and the change looks right. CI applies it to Fleet.

> **Note:** Don't give the agent your `FLEET_API_TOKEN`. It doesn't need one to write and validate YAML, and CI already has the token to apply merged changes.

To make review required, protect `main`. Require a pull request with at least one approval, and require the dry run check to pass before merging. See GitHub's [branch protection rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/managing-a-branch-protection-rule) or GitLab's [protected branches](https://docs.gitlab.com/user/project/repository/branches/protected/).

## Verify

1. On GitHub, the **Apply latest configuration to Fleet** run for your merge shows a green check. On GitLab, the pipeline for `main` passed.
2. In Fleet, go to **Policies** and select the **💻 Workstations** fleet. The FileVault policy is listed.
3. With GitOps mode on, open any policy in the Fleet UI. Editing is turned off, and the UI points to your repository.

## Troubleshoot

**`fleetctl` isn't recognized on Windows**

Close and reopen your terminal and code editor so they pick up the new `PATH`. If it still isn't found, run `C:\Program Files\FleetDM\fleetctl\fleetctl.exe --version` to confirm the install, then reinstall from [fleetdm.com/download](https://fleetdm.com/download).

**`Windows clients must configure rootca (secure) or tls-skip-verify (insecure)`**

Some `fleetctl` releases on Windows don't use the Windows certificate store. Point `fleetctl` at a PEM file containing your Fleet server's root certificate authority, and include your proxy's CA if your network inspects TLS:

```bash
fleetctl config set --rootca C:\path\to\ca-bundle.pem
```

Avoid `--tls-skip-verify`. It turns off certificate checks.

**CI fails before it reaches the dry run**

Check that `FLEET_URL` and `FLEET_API_TOKEN` are set, and that `FLEET_URL` has no trailing slash. A `401` or `403` error means the token is wrong or the GitOps user's role can't make the change.

**The dry run fails on a key or value**

The error names the file and key. Check it against the [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files), or paste the error into your AI agent and ask it to fix the file.

**The first sync fails after `generate-gitops`**

Look for placeholders that `generate-gitops` left in place of secrets or settings it can't export yet. Replace each one, then run the dry run again.

## Further reading

- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet)
- [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl)
- [Set up GitOps for staging and production](https://fleetdm.com/guides/set-up-gitops-for-staging-and-production)
- [Revert a bad change in GitOps](https://fleetdm.com/guides/revert-a-bad-change-in-gitops)
- [fleetctl](https://fleetdm.com/guides/fleetctl)

<meta name="articleTitle" value="Zero to GitOps: set up Fleet GitOps and make changes with AI">
<meta name="authorFullName" value="Mitch Francese">
<meta name="authorGitHubUsername" value="tux234">
<meta name="publishedOn" value="2026-10-21">
<meta name="category" value="guides">
<meta name="description" value="Go from nothing to a Git repository that keeps Fleet in sync, then make your first change with an AI agent through a pull request.">
