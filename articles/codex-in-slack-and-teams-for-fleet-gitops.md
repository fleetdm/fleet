# Set up Codex in Slack and Microsoft Teams for Fleet GitOps

Once your Fleet configuration lives in a GitOps repository, most changes are a pull request away. This guide connects OpenAI's Codex to Slack and Microsoft Teams, so anyone on your IT team can mention `@ChatGPT` in a channel, describe a change, and get a pull request back. Codex writes the change in Codex Cloud and checks it with the `fleet-gitops` skill. Your CI runs a dry run, a person reviews it, and merging it deploys it. You'll also connect Codex to Fleet's MCP server, so it can look up hosts, policies, and software in Fleet.

This guide picks up where Fleet's GitOps level 1 certification leaves off: a repository created with `fleetctl new`, on GitHub, with CI applying it to Fleet. In October 2026, OpenAI replaced the `@Codex` Slack app with `@ChatGPT`, one app for Slack and Teams that hands repository work to Codex Cloud. If you set up `@Codex` before, follow these steps again.

## Prerequisites

- A Fleet GitOps repository on GitHub, created with `fleetctl new`, with the `FLEET_URL` and `FLEET_API_TOKEN` secrets set and the **Apply latest configuration to Fleet** workflow passing.
- A ChatGPT Enterprise workspace, and the owner or admin role in it. Handing work from Slack and Teams to Codex Cloud requires Enterprise.
- An owner of the GitHub organization that owns the repository, to approve the ChatGPT app on GitHub.
- For Slack: a Slack workspace owner or app manager to approve the ChatGPT app.
- For Microsoft Teams: a Microsoft account with the Global Administrator or Privileged Role Administrator role, and a Teams admin to install the app.
- To connect Fleet's MCP server: Fleet 4.95 or later, `fleetctl` logged in as a Fleet admin, and the Codex app, CLI, or IDE extension.

## Add the fleet-gitops skill to your repository

Codex reads `AGENTS.md` before it works and discovers skills in `.agents/skills/`, including in Codex Cloud. Recent versions of `fleetctl new` create both, along with the `fleet-gitops` skill. If your repository already has `AGENTS.md` and `.agents/skills/fleet-gitops/SKILL.md`, skip to the next step.

1. Update `fleetctl` and create a fresh template in a temporary folder:

```bash
npm install -g fleetctl@latest
fleetctl new --dir /tmp/fleet-template --org-name "Acme"
```

2. From the root of your repository, copy the agent files in and commit them:

```bash
cp -R /tmp/fleet-template/AGENTS.md /tmp/fleet-template/CLAUDE.md /tmp/fleet-template/GEMINI.md /tmp/fleet-template/.agents /tmp/fleet-template/.claude .
git add AGENTS.md CLAUDE.md GEMINI.md .agents .claude
git commit -m "Add AI agent instructions and the fleet-gitops skill"
```

3. Push the commit to your default branch. Codex only reads what's on GitHub.

## Protect your default branch

A merge to your default branch is a deploy: the **Apply latest configuration to Fleet** workflow runs `fleetctl gitops` and pushes the change to every host it targets. Make sure a person approves every pull request before it merges.

1. In your repository, go to **Settings > Rules > Rulesets** and select **New ruleset > New branch ruleset**.
2. Name the ruleset, set **Enforcement status** to **Active**, and add your default branch under **Targets**.
3. Turn on **Require a pull request before merging** and set **Required approvals** to at least 1.
4. Turn on **Require status checks to pass** and add the `fleet-gitops` check, so a failed dry run blocks the merge.
5. Select **Create**.

> **Warning:** Don't give Codex your `FLEET_API_TOKEN`, as an environment variable or a secret. That token belongs to an API-only user with the GitOps role, which can apply changes, not only dry-run them. Codex checks its work offline with the `fleet-gitops` skill's `validate.py`, and your CI runs the dry run with the token it already has.

## Create a Codex Cloud environment for your repository

`@ChatGPT` runs repository work in a published Codex Cloud environment that's shared with your workspace. It won't use personal environments.

1. In Codex, go to **Settings > Codex Cloud > Environments** and select **Create environment**.
2. Select your GitOps repository. If prompted, connect GitHub and install the ChatGPT app on your GitHub organization. Give it access to only your GitOps repository.
3. Turn on **Allow Codex to access internet**. Under **Allow domains**, choose **Package managers**, and add `fleetdm.com` under **Additional allowed domains** so Codex can read Fleet's YAML reference and osquery schema.
4. Codex walks you through setup. This repository needs nothing installed: the `fleet-gitops` validator uses only Python's standard library.
5. Under **Privacy > Who can use**, choose your workspace.
6. Select **Publish** and wait for **Environment published**.

## Add Codex to Slack

1. In the OpenAI admin console ([admin.openai.com](https://admin.openai.com)), select your workspace, open **Agents**, and connect Slack. A Slack owner or app manager approves the ChatGPT app's permissions.
2. Create a surface for your Slack workspace. For a pilot, choose **Selected channels only** and add the channel where your team will request changes.
3. Under **Enable Codex Cloud for Slack**, turn on Codex Cloud delegation.
4. In role-based access control, grant Codex Cloud access to the surface's service account and to each person who'll request changes. Both need it, because tasks run as the person who asked.
5. In Slack, invite `@ChatGPT` to the channel. The first time each person mentions it, ChatGPT asks them to link their account. After linking, they mention `@ChatGPT` again in the same thread.

## Add Codex to Microsoft Teams

1. In the OpenAI admin console, select your workspace, open **Agents**, and connect Microsoft Teams. Sign in with a Microsoft account that has the Global Administrator or Privileged Role Administrator role, and grant consent for your organization.
2. In the Teams admin center, open the ChatGPT app, go to **Users and groups**, and install it for the people who'll use it. Confirm your Teams app policies allow it.
3. Back in the OpenAI admin console, wait for the connection to show **Ready**.
4. Create a surface for your Teams tenant, add the channel where your team will request changes, and grant Codex Cloud access to the surface's service account and to the people who'll use it.

> **Note:** OpenAI's admin docs describe Codex Cloud delegation for Slack in the most detail. If a Teams surface doesn't show the same Codex Cloud setting, check [OpenAI's Slack and Teams setup guide](https://help.openai.com/en/articles/20001538-setting-up-and-managing-chatgpt-in-slack-and-microsoft-teams) for its current status.

## Connect Codex to Fleet's MCP server

Fleet 4.95 and later include a read-only MCP server at `https://<your-fleet-url>/api/mcp`.

Codex Cloud tasks, including the ones `@ChatGPT` starts from Slack and Teams, don't load MCP servers yet. ChatGPT's custom MCP plugins support OAuth or no authentication, and can't send an API token. So connect Fleet's MCP server where your admins review and refine changes: the Codex app, CLI, or IDE extension.

### Create a read-only API user

Create a dedicated API-only user with the Observer role. The connection can read but never change anything, and its activity is easy to spot in Fleet's activity feed:

```bash
fleetctl user create --name "AI assistant (read-only)" --api-only --global-role observer
```

`fleetctl` prints the user's API token once. Copy it.

> **Note:** To limit the user to specific fleets, replace `--global-role observer` with `--fleet <fleet_id>:observer`, once per fleet.

### Add the server to your repository

1. Add `.codex/config.toml` to your GitOps repository, replacing `fleet.example.com` with your Fleet server's host:

```toml
[mcp_servers.fleet]
url = "https://fleet.example.com/api/mcp"
bearer_token_env_var = "FLEET_MCP_TOKEN"
```

2. Commit it. The file holds no secret, so it's safe in the repository.
3. On each admin's computer, store the token in the macOS keychain and load it in their shell profile:

```bash
security add-generic-password -a "$USER" -s fleet-mcp -w
echo 'export FLEET_MCP_TOKEN="$(security find-generic-password -s fleet-mcp -w)"' >> ~/.zshrc
```

4. Open the repository in Codex and trust it when asked. Codex only loads a project's `.codex/config.toml` for trusted projects.

## Make your first change

1. In a channel where `@ChatGPT` is on, name the repository, describe the change, and ask for a draft pull request:

```
@ChatGPT In acme/it-and-security, add a macOS policy to the Workstations fleet that checks FileVault is on. Open a draft pull request.
```

2. ChatGPT posts that the task started, and sends you a private approval card. Select **Allow**. Select **Follow along** to watch the task.
3. When the task finishes, the result lands in the thread. If it doesn't include a pull request link, open the task and create a draft pull request from there.
4. Review the new file under `platforms/macos/policies/` and the change to `fleets/workstations.yml`, and wait for the `fleet-gitops` check to pass.
5. To ask for changes, reply in the thread and mention `@ChatGPT`.
6. When the check passes and the change looks right, mark the pull request ready for review, approve it, and merge it. The workflow applies it to Fleet.

## Verify

1. In Fleet, go to **Policies**, select the **Workstations** fleet, and confirm the new policy is there.
2. In the Codex CLI, run `codex mcp list` from your GitOps repository and confirm the `fleet` server is listed. In the Codex app, check **Settings > MCP servers**.

## Troubleshoot

**`@ChatGPT` answers but doesn't start a Codex task**

Check that Codex Cloud delegation is on for the channel's surface, and that both the surface's service account and the person asking have Codex Cloud access.

**`@ChatGPT` can't find the repository**

Check that the environment is published and shared under **Privacy > Who can use**. `@ChatGPT` only uses published environments shared with the workspace. Name the repository in your request, and start a new thread to switch environments.

**The dry run fails on a key Codex added**

Reply in the thread: `@ChatGPT The dry run failed. Run python3 .agents/skills/fleet-gitops/scripts/validate.py, fix what it reports, and check each key against Fleet's GitOps YAML reference.`

**The `fleet` MCP server doesn't connect**

Run `echo $FLEET_MCP_TOKEN` in a new terminal to confirm the token loads. A `401` means the token is wrong. A connection error means your Fleet server is older than 4.95. If Codex ignores `.codex/config.toml`, trust the project and restart Codex.

## Further reading

- [ChatGPT in Slack and Microsoft Teams](https://learn.chatgpt.com/docs/enterprise/chatgpt-slack-and-teams)
- [Codex Cloud environments](https://learn.chatgpt.com/docs/environments/cloud-environments)
- [Codex MCP servers](https://learn.chatgpt.com/docs/extend/mcp)
- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Set up Claude in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/claude-in-slack-and-teams-for-fleet-gitops)
- [Set up GitHub Copilot in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/github-copilot-in-slack-and-teams-for-fleet-gitops)

<meta name="articleTitle" value="Set up Codex in Slack and Microsoft Teams for Fleet GitOps">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-16">
<meta name="description" value="Ask Codex for Fleet changes in Slack or Teams and get a validated GitOps pull request. Connect it to Fleet's MCP server.">
