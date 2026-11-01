# Zero to GitOps: set up Fleet GitOps and make changes with AI

Managing Fleet in the UI works well until a second or third admin joins and nobody can answer "who changed this, and why?" GitOps moves your Fleet configuration into a Git repository. Every change becomes a pull request that a teammate reviews, CI checks, and a merge deploys. That same repository is what lets an AI agent do real work for you: it can read your configuration, write a change, and hand it back as a pull request.

This guide starts from an empty folder. It ends with a repository that keeps Fleet in sync, a workstation set up for AI-driven changes, guardrails that keep an agent from shipping anything a person hasn't approved, and a first change written by an agent. The examples use GitHub, with notes where GitLab differs, and work from macOS, Linux, or Windows.

## How the pieces fit together

It helps to see the loop you're building before you set anything up:

1. Your Fleet configuration lives as YAML files in a Git repository: organization-wide settings in `default.yml`, and one file per fleet in `fleets/`.
2. A person, or an AI agent, changes those files on a branch and opens a pull request.
3. CI runs `fleetctl gitops --dry-run` against your Fleet instance and reports what the change would do, or the error that would stop it.
4. A person reviews the change and the dry run, then merges.
5. CI applies the merged configuration to Fleet. It also re-applies it every night, so changes made outside Git don't stick.

Fleet's GitOps configuration is declarative. The repository describes the whole configuration, so Fleet creates what the files define and removes or resets what they don't. That's what makes the repository trustworthy as the source of truth. It's also why an existing Fleet instance starts differently from a new one, as you'll see below.

AI fits in at step 2, and only there. The agent writes changes and opens pull requests. It doesn't merge them, and it never talks to Fleet directly. Everything in this guide is set up to keep it that way. For the longer argument, read [Why AI-powered device management requires GitOps](https://fleetdm.com/articles/why-ai-powered-device-management-requires-gitops).

> **Note:** This workflow isn't tied to one AI tool. It works with any agent that can edit files, run Git, and open pull requests, using whichever model your team prefers. The repository's `AGENTS.md` is read by most agents, including Codex, Cursor, GitHub Copilot, Gemini CLI, and Kilo Code. This guide uses Claude for its examples.

## Prerequisites

- A Fleet instance, and a user with the global admin role.
- Fleet Premium for the GitOps role and GitOps mode. On Fleet Free, the GitOps user needs the global admin role, and GitOps mode isn't available.
- A GitHub account that can create a private repository and add Actions secrets. To protect `main` in a private repository, you need GitHub Pro, Team, or Enterprise. GitHub Free doesn't offer rulesets or branch protection for private repositories. On GitLab, you need a project where you have the Maintainer role. Required approvals need GitLab Premium or Ultimate.
- An AI coding agent. To follow this guide's examples, you need a Claude Pro, Max, Team, or Enterprise plan. The free Claude plan doesn't include Claude Code.

## Set up your workstation

You'll use four tools: Git, `fleetctl`, the GitHub CLI (`gh`), and the Claude app. `gh` matters more than it looks. Claude and most other agents open pull requests with `gh`, and Git pushes with your `gh` login, so the way you sign in to `gh` is the way your agent reaches GitHub.

### Install the tools

| Tool | macOS | Windows (PowerShell) | Linux |
|------|-------|----------------------|-------|
| Git | `xcode-select --install` | `winget install --id Git.Git -e --source winget` | Your distribution's package manager |
| fleetctl | `.pkg` installer from [fleetdm.com/download](https://fleetdm.com/download) | `.msi` installer from [fleetdm.com/download](https://fleetdm.com/download) | `curl -sSL https://fleetdm.com/resources/install-fleetctl.sh \| bash` |
| GitHub CLI | `brew install gh` | `winget install --id GitHub.cli --source winget` | [GitHub CLI's Linux instructions](https://github.com/cli/cli/blob/trunk/docs/install_linux.md) |
| Claude app | [claude.ai/download](https://claude.ai/download) | [claude.ai/download](https://claude.ai/download) | [Claude for Linux (beta)](https://code.claude.com/docs/en/desktop-linux) |

A few details are worth knowing:

- The `fleetctl` installers for macOS and Windows are signed and add `fleetctl` to your `PATH`. On macOS, it goes in `/usr/local/bin`. On Windows, it goes in `C:\Program Files\FleetDM\fleetctl`. On Linux, the script installs `fleetctl` in `~/.fleetctl` and prints the `export` line to add to your shell profile if that folder isn't on your `PATH` yet.
- Use Claude Code through the Claude app's **Code** tab. The app adds things the command-line version doesn't have: a diff view you can comment on line by line, pull request and CI status for each session, and parallel sessions side by side. If you prefer the terminal, the [Claude Code CLI](https://code.claude.com/docs/en/setup) works with this guide too.
- On GitLab, install the GitLab CLI (`glab`) instead of `gh`. See [glab's installation options](https://gitlab.com/gitlab-org/cli/#installation).

> **Note:** On Windows, programs only see an updated `PATH` if they start after the installer finishes. Open a new Windows Terminal window, not just a new tab, before you use the new commands.

Confirm the command-line tools are installed:

```sh
git --version
fleetctl --version
gh --version
```

### Sign in to GitHub with gh

There are several ways to give Git access to GitHub: SSH keys, personal access tokens, or the GitHub CLI's browser sign-in. Use the browser sign-in. There's no SSH key to generate, upload, and rotate, and no token to copy into a file. `gh` stores its token in your operating system's credential store, such as Keychain on macOS or Credential Manager on Windows. Git then uses `gh` for HTTPS credentials, so `git push` and `gh pr create` share one login. HTTPS also works on networks that block SSH.

```sh
gh auth login --web --git-protocol https --scopes workflow
```

`gh` shows a one-time code and opens your browser. Paste the code, approve the request, and return to the terminal. The `workflow` scope lets you push files under `.github/workflows/`, which your repository's CI lives in. For GitHub Enterprise Server, add `--hostname` with your server's host name.

If your GitHub organization uses SAML single sign-on, sign in through your identity provider in that browser before you approve the code. See GitHub's [docs on SAML single sign-on](https://docs.github.com/en/authentication/authenticating-with-saml-single-sign-on/about-authentication-with-saml-single-sign-on).

Make sure Git uses `gh` for credentials, then check your login:

```sh
gh auth setup-git
gh auth status
```

`gh auth status` lists your account, the protocol Git uses, and the token's scopes.

On GitLab, run `glab auth login` and choose the web sign-in. See [GitLab's authentication docs](https://docs.gitlab.com/cli/authentication/).

### Sign in to Claude

Open the Claude app, sign in with your Claude account, and select the **Code** tab. You'll point it at your repository once the repository exists. See [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet) for more on the **Code** tab.

Using a different agent? Install it and sign in now. The rest of the guide works the same way.

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

Read through the export before you push. Fleet treats it as your complete configuration, so anything missing from it is removed on the first sync. Secrets come out as comments for you to replace with variables, such as `$ENROLL_SECRET`, which you'll add as CI secrets when you set up the repository. Settings the export can't handle yet come out as placeholders that fail the dry run, so CI stops before it changes anything. [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl) covers both, along with the export's other options.

## Give CI access to Fleet

CI talks to Fleet with an API token. Give it a dedicated API-only user rather than your own account. API-only users can't sign in to the Fleet UI, and their tokens don't expire, so CI keeps working when a person's session times out. The [fleetctl guide](https://fleetdm.com/guides/fleetctl) has more on API-only users.

To create the user in the Fleet UI:

1. Go to **Settings > Users**.
2. Select **Add user > API-only user**.
3. Name it something like "GitOps", and give it the global **GitOps** role. On Fleet Free, choose **Admin**.
4. Save, and copy the API token Fleet shows you.

You can also create the user from the command line, after `fleetctl login`. On Fleet Free, use `--global-role admin`:

```sh
fleetctl user create --name GitOps --api-only --global-role gitops
```

If you logged in with `fleetctl` for any of the steps above, log out now:

```sh
fleetctl logout
```

From here on, nothing on your computer needs access to Fleet. CI has its own token, and the agent you'll set up later works only with files and pull requests. Logging out means an agent running on this computer can't reach Fleet with your admin session.

## Push the repository and watch the first sync

Create the repository on GitHub, give CI its two secrets, and then push. The order matters. CI starts as soon as `main` receives its first commit, and it can't reach Fleet without the secrets.

From the `it-and-security` folder, create a private repository and connect it to the folder. Replace `your-org` with your GitHub organization, or leave it off to use your personal account:

```sh
git init -b main
gh repo create your-org/it-and-security --private --source=. --remote=origin
```

Add the secrets. Each command asks you to paste the value, which keeps it out of your shell history:

```sh
gh secret set FLEET_URL
gh secret set FLEET_API_TOKEN
```

Set `FLEET_URL` to your Fleet URL, such as `https://fleet.example.com`, with no trailing slash. Set `FLEET_API_TOKEN` to the GitOps user's token. If `generate-gitops` asked you to replace any secrets, add those with `gh secret set` too.

Then commit, push, and follow the run from your terminal:

```sh
git add .
git commit -m "Add Fleet GitOps configuration"
git push -u origin main
gh run watch
```

You can also watch it in the repository's **Actions** tab, under **Apply latest configuration to Fleet**. Each run does three things:

1. Asks your Fleet server for its version, and installs the matching `fleetctl`.
2. Runs `fleetctl gitops --dry-run` with `default.yml` and every file in `fleets/`. If the dry run fails, the run stops and Fleet doesn't change.
3. Applies the configuration.

From now on, the workflow runs on every push to `main`, on every pull request (dry run only), and every night at 6:00 AM UTC. The nightly run puts Fleet back in line with `main` if someone changed something in the UI.

On GitLab, create the project in the GitLab UI, and add `FLEET_URL` and `FLEET_API_TOKEN` as masked CI/CD variables in **Settings > CI/CD > Variables** before you push. The pipeline does a dry run on merge requests and applies on the default branch. To sync nightly, add a [pipeline schedule](https://docs.gitlab.com/ci/pipelines/schedules/). The repository's `README.md` has suggested settings.

## Turn on GitOps mode

The nightly sync quietly undoes changes made in the UI, which confuses anyone who doesn't know Fleet is managed from Git. GitOps mode makes it explicit. It turns off editing in the Fleet UI for everything GitOps manages, and points people to your repository instead. It's available in Fleet Premium.

1. Go to **Settings > Integrations > Change management**.
2. Select **Enable GitOps mode**.
3. Enter your repository's URL in **Git repository URL**, then save.

To keep managing labels, software, or enroll secrets in the UI, see [GitOps mode](https://fleetdm.com/guides/gitops-mode) for exceptions.

## Protect main before an agent touches it

Your agent pushes branches and opens pull requests with your GitHub login. That's convenient, and it also means that anything you can do on GitHub, the agent can attempt. Instructions and local settings make an agent behave, but they live on your computer and can be worked around. The rules that hold no matter what run on GitHub, so set them up before you hand the repository to an agent.

### Add a ruleset for main

1. In your repository, go to **Settings > Rules > Rulesets**.
2. Select **New ruleset > New branch ruleset**, and name it something like "Protect main".
3. Set **Enforcement status** to **Active**.
4. Under **Target branches**, add the default branch.
5. Leave the **Bypass list** empty. Anyone on it can skip these rules.
6. Turn on these rules:
   - **Restrict deletions** and **Block force pushes**, so history can't be rewritten.
   - **Require a pull request before merging**. Under it, set **Required approvals** to 1 if anyone else can review. Also turn on **Dismiss stale pull request approvals when new commits are pushed**, and **Require approval of the most recent reviewable push**.
   - **Require status checks to pass**, and add the **fleet-gitops** check. That's the job that runs the dry run, so a change that fails it can't merge.
7. Save the ruleset.

The approval rule does more than it looks like. GitHub doesn't let pull request authors approve their own pull requests. The agent opens pull requests under your account, so with one required approval, nothing it writes can merge until someone other than you approves it. That includes merges the agent might attempt itself.

This depends on who the pull request's author is. Some agents open pull requests under their own GitHub App instead of your login, like Claude in Slack. With those, the person who asked for the change can approve it alone, so set **Required approvals** to 2 to keep a second reviewer.

If you're the only admin, set approvals to 0. The status check, the force-push block, and the pull request requirement still apply. Without a required approval, though, your login can merge, and so can an agent using it. The instructions and guardrails in the next section are what stop it, so treat a second reviewer as the goal.

> **Note:** GitHub only lists checks that have run in the repository in the last seven days. If **fleet-gitops** doesn't appear, run the workflow from the **Actions** tab, then add the check.

### Require review for the files that matter most

Some files deserve a second set of eyes every time. `default.yml` holds organization-wide settings, and `.github/` holds the CI that applies everything. Add a `CODEOWNERS` file to the `.github/` folder:

```text
/default.yml @your-org/it-admins
/.github/ @your-org/it-admins
```

Then, in the ruleset, turn on the option to require review from Code Owners. Code owners need write access to the repository.

### Keep CI from approving its own work

In **Settings > Actions > General**, under **Workflow permissions**, leave **Allow GitHub Actions to create and approve pull requests** turned off. Your GitOps workflow never needs it.

### On GitLab

In **Settings > Repository > Branch rules**, set **Allowed to push and merge** on the default branch to no one, so every change arrives as a merge request. In **Settings > Merge requests > Merge checks**, turn on **Pipelines must succeed**. With GitLab Premium or Ultimate, add an approval rule and turn on **Prevent approval by merge request creator (author)**.

## Tell your agent how your team works

The template's `AGENTS.md` already tells agents how the repository is laid out, and never to apply changes themselves. Add the rules for how your team ships changes. Every agent that reads `AGENTS.md` follows them, and Claude reads it through `CLAUDE.md`.

`AGENTS.md` is also where your configuration conventions go, like which Windows CSP you prefer or how new profiles get wired into a fleet. Step 2 of [Build and validate configuration profiles with AI](https://fleetdm.com/guides/build-configuration-profiles-with-ai) covers those. The rules below are about something else: how changes get from the agent to `main`.

Add this section to the end of `AGENTS.md`:

```markdown
## How we ship changes

- Never commit to `main` or push to it. Start every change on a new branch,
  in its own git worktree, based on the latest `origin/main`.
  - If your tool creates worktrees for you, use them. Claude creates them
    under `.claude/worktrees/`.
  - Otherwise, run `git fetch origin` then
    `git worktree add ../it-and-security-<change> -b <change> origin/main`.
- Use a short branch name that describes the change, such as
  `add-filevault-policy`.
- Make one change per pull request. Don't bundle unrelated policies,
  profiles, or software.
- Before you commit, run the fleet-gitops skill's validator and fix what it
  reports.
- Push the branch and open a draft pull request with `gh pr create --draft`.
  In the description, list what changed, which fleets and labels it targets,
  what you validated, and any CI variables a person needs to add.
- Never merge, approve, or enable auto-merge on a pull request. A person does
  that after reading the CI dry run.
- Never run `fleetctl gitops` or ask for `FLEET_API_TOKEN`. CI runs the dry
  run on the pull request. This replaces the local dry run described in
  "Validating changes" above.
- Never force-push. Don't change `.github/`, `.gitlab-ci.yml`, or this file
  unless a person asks you to.
```

A worktree is a separate folder with its own branch, sharing the repository's history. Each change gets its own copy of the files, so an agent can't touch your main checkout, and two changes in progress can't collide. Claude creates worktrees in `.claude/worktrees/` at the root of the repository. Add that folder to `.gitignore`, so worktrees don't show up as untracked files:

```sh
echo ".claude/worktrees/" >> .gitignore
```

### Add guardrails to your agent

Instructions shape what an agent tries. Permission rules decide what it's allowed to run. Most agents can block specific commands, so check your tool's documentation for its version of the rules below.

For Claude, create `.claude/settings.json` in the repository with rules that block the commands your instructions already forbid. The Claude app and the Claude Code CLI both read this file:

```json
{
  "permissions": {
    "deny": [
      "Bash(gh pr merge *)",
      "Bash(gh pr review *)",
      "Bash(git push --force *)",
      "Bash(git push -f *)",
      "Bash(git push origin main *)",
      "Bash(fleetctl gitops *)"
    ]
  }
}
```

A deny rule wins over any allow rule, so approving a command later with "Yes, and don't ask again" can't undo it. It's still a seatbelt, not a lock. Deny rules match how a command is written, so a differently worded command can get past them. Claude's [permissions docs](https://code.claude.com/docs/en/permissions) cover this, along with hooks for stricter checks. The lock is the ruleset on GitHub, which holds no matter which agent you use.

Commit both changes through a pull request, the same way every change will go from now on. This one is a good first test of your ruleset:

```sh
git switch -c agent-rules
git add AGENTS.md .gitignore .claude/settings.json
git commit -m "Add agent workflow rules and guardrails"
git push -u origin agent-rules
gh pr create --fill
```

Once the **fleet-gitops** check passes, and a teammate approves if your ruleset requires it, merge it.

## Make your first change with AI

This is where the setup pays off. Instead of looking up YAML keys and osquery tables, you describe what you want, and the agent writes it as a pull request.

1. In the Claude app, select the **Code** tab.
2. Under the prompt box, select the project folder control and choose your `it-and-security` folder. Turn on **Worktree**. Claude works in a new worktree under `.claude/worktrees/`, on its own branch, and leaves your main checkout alone.
3. Describe the change: "Add a policy that checks FileVault is on for macOS hosts in Workstations. Validate it, then commit, push, and open a draft pull request."
4. The agent reads `default.yml` and `fleets/workstations.yml`, writes the policy in `platforms/macos/policies/`, and checks the query against Fleet's osquery schema. It runs the skill's validator, commits, pushes the branch, and opens a draft pull request. If `gh` isn't installed, the Claude app offers to install it the first time it opens a pull request.
5. Review the change in the app's diff view. Comment on any line that needs work and send the comments to Claude, which fixes them in the same session. The pull request appears in the session's **Changes** card, along with the status of its CI checks.
6. Open the pull request on GitHub with `gh pr view --web`. The **fleet-gitops** check runs the dry run. Select the check's **Details** to read what the change does to Fleet. For this change, that's one new policy.
7. When you're satisfied, select **Ready for review**, and merge once the ruleset's requirements are met. CI applies the policy to Fleet within a few minutes.
8. Once the pull request is merged, remove the worktree. Run `git worktree list` to find it, and `git worktree remove <path>` to delete it.

To pick up a pull request later, for example to address review comments, start a new session with **Worktree** turned on and ask Claude to check out the pull request's branch.

Using a different agent? The steps are the same: start the change on a fresh branch or worktree, describe what you want, and have the agent open a draft pull request. To use Claude without a GitOps repository, see [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet).

## Review and merge with care

The ruleset makes sure a person merges every change. These habits make sure that person merges the right ones. For profile-specific questions, like overlapping settings or Windows build applicability, see step 6 of [Build and validate configuration profiles with AI](https://fleetdm.com/guides/build-configuration-profiles-with-ai).

- **Read the diff, not just the summary.** The agent's description says what it meant to do. The diff shows what it did.
- **Read the dry run.** Pay the most attention to anything the dry run deletes or resets. A deletion you didn't expect usually means a key went missing from a file.
- **Check the scope.** Confirm which fleets and labels a change targets. "All Macs" and "Workstations" aren't the same thing.
- **Look for secrets.** Credentials, enroll secrets, and webhook URLs belong in CI variables, referenced as `$VARIABLE`, never typed into a file.
- **Keep pull requests small.** One change per pull request is easier to review and easier to undo.
- **Merge when you can watch.** A merged change reaches devices within minutes. Run `gh run watch` after you merge, and don't merge and walk away.
- **Don't turn on auto-merge for agent pull requests.** That includes **Merge when ready** in the Claude app, which turns on GitHub's auto-merge. The point is that a person decides when a change ships.
- **Know how to undo it.** Revert the merged pull request with the **Revert** button on GitHub, or `gh pr revert <number>`. The revert is itself a pull request, so it gets the same dry run before it ships. See [Revert a bad change in GitOps](https://fleetdm.com/guides/revert-a-bad-change-in-gitops).

## Verify

1. The CI run for your merge passed. On GitHub, run `gh run list` or check the **Actions** tab.
2. In Fleet, go to **Policies** and select the **💻 Workstations** fleet. The FileVault policy is listed.
3. Open any policy in the Fleet UI. With GitOps mode on, editing is turned off, and the UI points to your repository.
4. In the repository's **Settings > Rules > Rulesets**, your ruleset shows as **Active**, and the merged pull requests on the **Pull requests** tab show the **fleet-gitops** check.

## Troubleshoot

**A new command isn't recognized on Windows**

Open a new Windows Terminal window, not a new tab, so it picks up the updated `PATH`. Restart VS Code if you use its terminal.

**`git push` is rejected because of the `workflow` scope**

Your `gh` login doesn't include the `workflow` scope, which GitHub requires to push files under `.github/workflows/`. Add it, then push again:

```sh
gh auth refresh -s workflow
```

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

**Worktree folders show up in `git status`**

Add `.claude/worktrees/` to `.gitignore`, as described in [Tell your agent how your team works](#tell-your-agent-how-your-team-works).

## Further reading

- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet)
- [Build and validate configuration profiles with AI](https://fleetdm.com/guides/build-configuration-profiles-with-ai)
- [Migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl)
- [Set up GitOps for staging and production](https://fleetdm.com/guides/set-up-gitops-for-staging-and-production)
- [Revert a bad change in GitOps](https://fleetdm.com/guides/revert-a-bad-change-in-gitops)
- [Claude Code in the Claude app](https://code.claude.com/docs/en/desktop)
- [Claude Code: run parallel sessions with worktrees](https://code.claude.com/docs/en/worktrees)
- [GitHub: about rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)

[*Get started with Fleet*](https://fleetdm.com/docs/get-started/why-fleet)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="tux234">
<meta name="authorFullName" value="Mitch Francese">
<meta name="publishedOn" value="2026-10-22">
<meta name="articleTitle" value="Zero to GitOps: set up Fleet GitOps and make changes with AI">
<meta name="description" value="Go from nothing to a Git repository that keeps Fleet in sync, then set up an AI agent to propose changes safely through pull requests.">
