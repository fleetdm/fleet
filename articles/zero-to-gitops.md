# Zero to GitOps: set up Fleet GitOps and make changes with AI

Managing Fleet in the UI works well until a second or third admin joins and nobody can answer "who changed this, and why?" GitOps moves your Fleet configuration into a Git repository. Every change becomes a pull request that a teammate reviews, CI checks, and a merge deploys. That same repository is what lets an AI agent do real work for you: it can read your configuration, write a change, and hand it back as a pull request.

This guide starts from an empty folder and ends with a repository that keeps Fleet in sync, plus a first change written by an AI agent. It covers GitHub and GitLab, from macOS, Linux, or Windows.

## How the pieces fit together

It helps to see the loop you're building before you set anything up:

1. Your Fleet configuration lives as YAML files in a Git repository: organization-wide settings in `default.yml`, and one file per fleet in `fleets/`.
2. A person, or an AI agent, changes those files on a branch and opens a pull request.
3. CI runs `fleetctl gitops --dry-run` against your Fleet instance and reports what the change would do, or the error that would stop it.
4. A person reviews the change and the dry run, then merges.
5. CI applies the merged configuration to Fleet. It also re-applies it every night, so changes made outside Git don't stick.

Fleet's GitOps configuration is declarative. The repository describes the whole configuration, so Fleet creates what the files define and removes or resets what they don't. That's what makes the repository trustworthy as the source of truth. It's also why an existing Fleet instance starts differently from a new one, as you'll see below.

AI fits in at step 2. An agent can propose changes, but it can't merge or apply them, so a person approves everything that reaches a device. For the longer argument, read [Why AI-powered device management requires GitOps](https://fleetdm.com/articles/why-ai-powered-device-management-requires-gitops).

## Prerequisites

- A Fleet instance, and a user with the global admin role.
- Fleet Premium for the GitOps role and GitOps mode. On Fleet Free, the GitOps user needs the global admin role, and GitOps mode isn't available.
- A GitHub or GitLab account where you can create a private repository and add secrets (GitHub) or CI/CD variables (GitLab).
- [Git](https://git-scm.com/downloads) installed on your computer.
- An AI coding agent for the last section. This guide uses [Claude Code](https://fleetdm.com/guides/use-claude-code-with-fleet). The repository also works with Codex, Cursor, GitHub Copilot, Gemini CLI, and Kilo Code.

## Install fleetctl

`fleetctl` is Fleet's command-line tool. On your computer, you only need it to create the repository. CI installs its own copy every time it runs, matched to your Fleet server's version.

On macOS or Linux, run the install script:

```sh
curl -sSL https://fleetdm.com/resources/install-fleetctl.sh | bash
```

The script installs `fleetctl` in `~/.fleetctl`. If that folder isn't on your `PATH` yet, the script prints the `export` line to add to your shell profile.

On Windows, download the installer from [fleetdm.com/download](https://fleetdm.com/download). It installs `fleetctl` in `C:\Program Files\FleetDM\fleetctl` and adds that folder to your system `PATH`.

> **Note:** Windows only gives the updated `PATH` to programs started after the installer finishes. If `fleetctl` isn't recognized, close and reopen PowerShell, Windows Terminal, or VS Code.

Run `fleetctl --version` to confirm the install.

## Create the repository

How you create the repository depends on whether your Fleet instance is new, or already has configuration you want to keep.

### Start a new Fleet instance from the template

Run `fleetctl new` and enter your organization's name when it asks:

```sh
fleetctl new
```

`fleetctl new` works offline, so you don't need to log in to Fleet first. It creates an `it-and-security` folder with everything the loop needs:

- `default.yml`, plus two starter fleets in `fleets/`: **💻 Workstations** and **📱🔐 Personal mobile devices**. Keep them, rename them, or replace them with your own.
- `platforms/`, with a folder per platform for policies, reports, scripts, software, and configuration profiles. The fleet files already pull in everything in these folders, so a new policy in `platforms/macos/policies/` applies to Workstations without extra wiring.
- The CI workflow for GitHub Actions in `.github/`, and the equivalent pipeline for GitLab in `.gitlab-ci.yml`.
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and a `fleet-gitops` skill. These tell AI agents how the repository is laid out, which rules `fleetctl gitops` enforces, and how to validate a change before proposing it.

### Bring an existing Fleet instance into Git

If you've been configuring Fleet in the UI, start from what's already there. Otherwise, the first sync would replace your configuration with the template's. `fleetctl generate-gitops` exports your current configuration as GitOps YAML. It needs a global admin login:

```sh
fleetctl config set --address https://fleet.example.com
fleetctl login
fleetctl generate-gitops --dir it-and-security
```

If you sign in to Fleet with single sign-on (SSO), `fleetctl login` can't use your password. Instead, go to **My account** in the Fleet UI, select **Get API token**, and run `fleetctl config set --token <your-token>`.

The export doesn't include CI or the AI files. Create the template in a separate folder and copy those files across:

```sh
fleetctl new --dir starter
cp -R starter/.github starter/.gitlab-ci.yml starter/.agents starter/.claude starter/AGENTS.md starter/CLAUDE.md starter/GEMINI.md starter/.gitignore it-and-security/
rm -rf starter
```

> **Warning:** Don't run `generate-gitops --force` inside a `fleetctl new` folder. It overwrites the files it writes but leaves the template's starter fleets in place, so your first sync would create **💻 Workstations** and **📱🔐 Personal mobile devices** alongside your real fleets.

`generate-gitops` keeps secrets, like enroll secrets and integration API keys, out of the files. It leaves a comment where each one belongs and lists them when it finishes. Replace each comment with a variable reference, such as `$ENROLL_SECRET`, and add the real value to your Git provider in the next section.

Read through the export before you push. Fleet treats it as your complete configuration, so anything missing from it is removed on the first sync. Settings that `generate-gitops` can't export yet come out as placeholders that fail the dry run, which stops CI before it changes anything. [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl) lists those gaps and the export's other options.

## Give CI access to Fleet

CI talks to Fleet with an API token. Give it a dedicated API-only user rather than your own account. API-only users can't sign in to the Fleet UI, and their tokens don't expire, so CI keeps working when a person's session times out.

To create the user in the Fleet UI:

1. Go to **Settings > Users**.
2. Select **Add user > API-only user**.
3. Name it something like "GitOps", and give it the global **GitOps** role. On Fleet Free, choose **Admin**.
4. Save, and copy the API token Fleet shows you.

You can also create the user from the command line, after `fleetctl login`. On Fleet Free, use `--global-role admin`:

```sh
fleetctl user create --name GitOps --api-only --global-role gitops
```

Next, create an empty, private repository on GitHub or GitLab, without a README. Give its CI two values:

- `FLEET_URL`: your Fleet URL, such as `https://fleet.example.com`, with no trailing slash.
- `FLEET_API_TOKEN`: the token you just copied.

On GitHub, add them as repository secrets in **Settings > Secrets and variables > Actions**. See [GitHub's docs on secrets](https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions#creating-secrets-for-a-repository). On GitLab, add them as masked CI/CD variables in **Settings > CI/CD > Variables**. See [GitLab's docs on CI/CD variables](https://docs.gitlab.com/ci/variables/#define-a-cicd-variable-in-the-ui). If `generate-gitops` asked you to replace any secrets, add those variables now too.

Add the secrets before you push. CI starts as soon as the default branch receives its first commit, and it can't reach Fleet without them.

## Push and watch the first sync

From the `it-and-security` folder, commit everything and push it to your new repository:

```sh
git init -b main
git add .
git commit -m "Add Fleet GitOps configuration"
git remote add origin https://github.com/your-org/it-and-security.git
git push -u origin main
```

On GitHub, open the repository's **Actions** tab and select the **Apply latest configuration to Fleet** run. On GitLab, go to **Build > Pipelines**. Each run does the same three things:

1. Asks your Fleet server for its version, and installs the matching `fleetctl`.
2. Runs `fleetctl gitops --dry-run` with `default.yml` and every file in `fleets/`. If the dry run fails, the run stops and Fleet doesn't change.
3. Applies the configuration.

The dry run's output lists everything the run creates, changes, or deletes. It's worth reading on your first sync, and on any pull request where the change surprises you.

From now on, the GitHub workflow runs on every push to `main`, on every pull request (dry run only), and every night at 6:00 AM UTC. The nightly run puts Fleet back in line with `main` if someone changed something in the UI. GitLab runs the same job on merge requests (dry run only) and on pushes to the default branch. To sync nightly on GitLab, add a [pipeline schedule](https://docs.gitlab.com/ci/pipelines/schedules/) for your default branch. The repository's `README.md` has suggested settings.

## Turn on GitOps mode

The nightly sync quietly undoes changes made in the UI, which confuses anyone who doesn't know Fleet is managed from Git. GitOps mode makes it explicit. It turns off editing in the Fleet UI for everything GitOps manages, and points people to your repository instead. It's available in Fleet Premium.

1. Go to **Settings > Integrations > Change management**.
2. Select **Enable GitOps mode**.
3. Enter your repository's URL in **Git repository URL**, then save.

If you want to keep managing labels, software, or enroll secrets in the UI, add an exception for them on the same page. See [GitOps mode](https://fleetdm.com/guides/gitops-mode) for what stays editable.

## Make your first change with AI

This is where the repository pays off. Instead of looking up YAML keys and osquery tables, you describe what you want, and an agent writes it as a pull request.

The agent works from the same files a person would. `AGENTS.md` explains the layout. The `fleet-gitops` skill covers the rules `fleetctl gitops` enforces, and ships an offline validator. Both tell the agent never to apply anything itself. Applying is CI's job, after a person merges.

This example uses Claude Code. See [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet) to sign in and open a repository.

1. Open your `it-and-security` repository in Claude Code.
2. Describe the change: "Create a branch, add a policy that checks FileVault is on for macOS hosts in Workstations, and open a pull request."
3. The agent reads `default.yml` and `fleets/workstations.yml`, writes the policy in `platforms/macos/policies/`, and checks the query against Fleet's osquery schema. It runs the skill's validator before it commits.
4. Open the pull request. CI posts a dry run that shows exactly what the change does to Fleet. For this change, that's one new policy.
5. Review the change the way you'd review a teammate's. Read the policy, check the dry run, and ask the agent to fix anything that's off.
6. Merge. CI applies the policy to Fleet within a few minutes.

> **Note:** The agent doesn't need `FLEET_API_TOKEN`, so don't give it one. It writes and validates YAML on your computer, and CI already has the token it needs to apply what you merge.

Every change after this one follows the same path, whether a person or an agent writes it. To make that path mandatory, protect `main`. Require a pull request with at least one approval, and require the dry run to pass before merging. See GitHub's [branch protection rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/managing-a-branch-protection-rule) or GitLab's [protected branches](https://docs.gitlab.com/user/project/repository/branches/protected/).

## Verify

1. The CI run for your merge passed: a green check on GitHub, or a passed pipeline on GitLab.
2. In Fleet, go to **Policies** and select the **💻 Workstations** fleet. The FileVault policy is listed.
3. Open any policy in the Fleet UI. With GitOps mode on, editing is turned off, and the UI points to your repository.

## Troubleshoot

**`fleetctl` isn't recognized on Windows**

Close and reopen your terminal and code editor, so they pick up the new `PATH`. If it's still not found, run `& "C:\Program Files\FleetDM\fleetctl\fleetctl.exe" --version` in PowerShell to confirm the install, or reinstall from [fleetdm.com/download](https://fleetdm.com/download).

**`Windows clients must configure rootca (secure) or tls-skip-verify (insecure)`**

This only affects `fleetctl` commands that connect to Fleet from a Windows computer, like `fleetctl login` and `generate-gitops`. CI isn't affected. Point `fleetctl` at a PEM file with the root certificate authority for your Fleet server. If your network inspects TLS, include your proxy's certificate authority in the same file:

```sh
fleetctl config set --rootca C:\path\to\ca-bundle.pem
```

Avoid `--tls-skip-verify`, which turns off certificate checks.

**CI fails before the dry run**

Check that `FLEET_URL` and `FLEET_API_TOKEN` are set, and that `FLEET_URL` has no trailing slash. A `401` or `403` error means the token is wrong, or the GitOps user's role can't make the change.

**The dry run fails on a key or value**

The error names the file and key. Check it against the [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files), or paste the error into your agent and ask it to fix the file.

**The first sync stops after `generate-gitops`**

The export left a placeholder for a secret or a setting it can't export yet. Search the files for the comments `generate-gitops` listed, replace each one, and push again.

## Further reading

- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet)
- [Build and validate configuration profiles with AI](https://fleetdm.com/guides/build-configuration-profiles-with-ai)
- [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl)
- [Set up GitOps for staging and production](https://fleetdm.com/guides/set-up-gitops-for-staging-and-production)
- [Revert a bad change in GitOps](https://fleetdm.com/guides/revert-a-bad-change-in-gitops)

[*Get started with Fleet*](https://fleetdm.com/docs/get-started/why-fleet)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="tux234">
<meta name="authorFullName" value="Mitch Francese">
<meta name="publishedOn" value="2026-10-21">
<meta name="articleTitle" value="Zero to GitOps: set up Fleet GitOps and make changes with AI">
<meta name="description" value="Go from nothing to a Git repository that keeps Fleet in sync, then make your first change with an AI agent through a pull request.">
