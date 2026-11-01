# Export Fleet to GitOps with an AI agent

If you've been configuring Fleet in the UI, moving to GitOps means writing all of that down as YAML first. The `fleet-gitops-export` skill lets an AI coding agent do it for you. It reads your Fleet and writes a repository laid out the way `fleetctl new` creates one, with your settings, fleets, profiles, scripts, policies, reports, labels, and software already in place. Secrets become environment variables, and a dry run proves that applying the repository would change nothing.

The export only reads from Fleet. Nothing changes until you merge the repository and CI applies it.

## Prerequisites

- A Fleet user with the global admin role.
- `fleetctl` 4.84.0 or later, ideally the same version as your Fleet server.
- Python 3.9 or later.
- An AI coding agent that can run commands, such as Claude Code, Codex, Cursor, GitHub Copilot, Gemini CLI, or Kilo Code.
- A GitHub or GitLab account for the new repository.

> **Note:** If you're trying the skill for the first time, point it at a test or staging Fleet instance.

## Install fleetctl and log in

1. Install `fleetctl`:

   ```bash
   npm install -g fleetctl
   ```

   To match your Fleet server's version, add it to the package name, for example `fleetctl@4.94.0`.

2. Point `fleetctl` at your Fleet server and log in:

   ```bash
   fleetctl config set --address https://fleet.example.com
   fleetctl login
   ```

> **Note:** If you sign in to Fleet with SSO or two-factor authentication, `fleetctl login` won't work. Get an API token from your account page in the Fleet UI and run `fleetctl config set --token <token>` instead. [Learn more](https://fleetdm.com/guides/fleetctl#users-with-single-sign-on-sso-or-email-two-factor-authentication-2-fa).

## Add the skill to your agent

1. Download the skill folder from the Fleet repository:

   ```bash
   git clone --depth 1 --filter=blob:none --sparse https://github.com/fleetdm/fleet.git fleet-skill
   git -C fleet-skill sparse-checkout set .claude/skills/fleet-gitops-export
   ```

2. Copy `fleet-skill/.claude/skills/fleet-gitops-export` into your agent's skills folder:
   - Claude Code: `~/.claude/skills/`
   - Codex, Cursor, GitHub Copilot, or Gemini CLI: `.agents/skills/` in the folder where you start the agent

Some agents ask you to trust a skill the first time they load it.

## Run the export

1. Open a terminal in the folder where the new repository should live, and start your agent.
2. Ask it to export your Fleet, for example: "Export our Fleet to a GitOps repo in ./fleet-gitops". In Claude Code, you can also type `/fleet-gitops-export ./fleet-gitops`.
3. Confirm the Fleet server URL the agent shows you. Say yes only if it's the Fleet you mean to export.
4. Confirm the organization name and the folder. The folder has to be empty.

The agent creates the layout, reads everything from Fleet, and writes a plan. On most instances, this takes a minute or two.

## Answer the review questions

Fleet doesn't record a few details, so the agent checks them with you before it writes the files:

- Script platform: Fleet runs `.sh` and `.py` scripts on macOS and Linux. The agent reads each script and asks only when it can't tell which one it's for.
- Apple profiles in mobile fleets: Fleet stores one Apple platform for macOS, iOS, and iPadOS, so the agent confirms where those profiles go.
- Fleet-maintained app scripts: Fleet keeps these apps' install and uninstall scripts current unless someone edits them. When a stored script looks like an older copy of Fleet's, the agent asks whether anyone changed it, and keeps it only if someone did.

The agent also lists gaps, such as bootstrap packages, users, and API tokens. GitOps can't represent them, so they stay as they are in Fleet.

## Check the result

1. Optional: to check files that use secret values with their real values, copy `.env.example` to `.env` in the new repository and fill in the values. Git ignores `.env`, and the agent doesn't open it.
2. Let the agent validate the repository. PASS means every file reference resolves, the repository matches Fleet item by item, and a dry run would change nothing.
3. Read through the files and the agent's report, including the list of environment variables you'll need in CI.

## Turn on GitOps in CI

> **Warning:** The repository's GitHub workflow applies it to Fleet on every push to `main` and on a nightly schedule. Add your secrets and check a pull request's dry run before anything reaches `main`.

1. Create an API-only user for CI with the GitOps role. On Fleet Free, use the global admin role. The command prints the user's API token.

   ```bash
   fleetctl user create --name "GitOps" --api-only --global-role gitops
   ```

2. Create an empty, private repository on GitHub or GitLab.
3. Add these as repository secrets (GitHub) or CI/CD variables (GitLab): `FLEET_URL`, `FLEET_API_TOKEN`, and every variable in `.env.example`.
4. Push the repository to a branch other than `main`, and open a pull request.
5. Check that the workflow's dry run passes and shows nothing you didn't expect, then merge.
6. In Fleet, go to **Settings > Integrations > Change management** and turn on **GitOps mode**, so changes in the UI don't get overwritten by the next run.

## Verify

1. In GitHub, open **Actions** (in GitLab, **Build > Pipelines**) and check that the run for your merge to `main` succeeded.
2. In Fleet, open a fleet and confirm its profiles, policies, and software look the way they did before.

## Troubleshoot

**The agent says you aren't a global admin**

Log in to `fleetctl` as a user with the global admin role.

**The agent warns that versions differ**

Install the `fleetctl` version that matches your Fleet server, so the dry run behaves the way it will in CI.

**The agent can't reach Fleet, but fleetctl works in your terminal**

Your agent runs commands in a sandbox without network access. Allow network access for these commands.

**The dry run fails on `org_settings.yara_rules`**

This is a known `fleetctl` bug. GitOps can't carry YARA rules until it's fixed.

## Further reading

- [The skill on GitHub](https://github.com/fleetdm/fleet/tree/main/.claude/skills/fleet-gitops-export)
- [GitOps mode](https://fleetdm.com/guides/gitops-mode)
- [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet)
- [YAML files reference](https://fleetdm.com/docs/configuration/yaml-files)

<meta name="articleTitle" value="Export Fleet to GitOps with an AI agent">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-31">
<meta name="category" value="guides">
<meta name="description" value="Use an AI agent and the fleet-gitops-export skill to turn your Fleet's current settings into a GitOps repository you can review and apply.">
