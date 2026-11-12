# Set up Codex in Slack and Microsoft Teams for Fleet GitOps

Once your Fleet configuration lives in a GitOps repository, most changes are a pull request away. This guide connects OpenAI's Codex to Slack and Microsoft Teams, so anyone on your IT team can mention `@ChatGPT` in a channel, describe a change, and get a pull request back. Codex writes the change in Codex Cloud and checks it with the `fleet-gitops` skill. Your CI runs a dry run, a person reviews it, and merging it deploys it. You'll also connect Codex to Fleet's MCP server, so it can look up hosts, policies, and software in Fleet.

This guide picks up where Fleet's GitOps level 1 certification leaves off: a repository created with `fleetctl new`, on GitHub, with CI applying it to Fleet. In September 2026, OpenAI introduced `@ChatGPT` for Slack and Teams, which can hand repository work to Codex Cloud. If your team used `@Codex` in Slack, start new requests with `@ChatGPT`.

## Prerequisites

- A Fleet GitOps repository on GitHub, created with [`fleetctl new`](https://github.com/fleetdm/fleet/blob/main/cmd/fleetctl/fleetctl/templates/new/README.md), with the `FLEET_URL` and `FLEET_API_TOKEN` secrets set and the **Apply latest configuration to Fleet** workflow passing.
- A ChatGPT Enterprise workspace with credits, and the owner or admin role in it. OpenAI documents handing work from chat to Codex Cloud for Enterprise workspaces.
- The **Manage workspace environments** permission, to create an environment shared with your workspace. It's off by default.
- An owner of the GitHub organization that owns the repository, to approve the ChatGPT app on GitHub.
- For Slack: a Slack workspace owner or app manager to approve the ChatGPT app.
- For Microsoft Teams: a Microsoft account with the Global Administrator or Privileged Role Administrator role, and a Teams admin to install the app.
- To connect Fleet's MCP server: Fleet Premium 4.95 or later, a Fleet admin account, and the ChatGPT desktop app, Codex CLI, or IDE extension. Fleet 4.95 is [coming soon](https://github.com/fleetdm/fleet/issues/44448).

## Add the fleet-gitops skill to your repository

Codex reads `AGENTS.md` before it works and discovers skills in `.agents/skills/`, including in Codex Cloud. `fleetctl new` 4.95 and later create both, along with the `fleet-gitops` skill. If your repository already has `AGENTS.md` and `.agents/skills/fleet-gitops/SKILL.md`, skip to the next step.

1. Install `fleetctl` 4.95 or later from [fleetdm.com/download](https://fleetdm.com/download). Earlier versions don't create `AGENTS.md`, `GEMINI.md`, or `.agents/`. Then create a fresh template in a folder that doesn't exist yet:

```bash
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

> **Warning:** Don't give Codex your `FLEET_API_TOKEN`, as an environment variable or a secret. That token can apply changes to Fleet, not only dry-run them. Codex checks its work offline with the `fleet-gitops` skill's `validate.py`, and your CI runs the dry run with the token it already has.

## Create a Codex Cloud environment for your repository

`@ChatGPT` runs repository work in a published Codex Cloud environment that's shared with your workspace. It won't use personal environments.

1. In Codex, go to **Settings > Codex Cloud > Environments** and select **Create environment**.
2. Select your GitOps repository. If prompted, connect GitHub and install the ChatGPT app on your GitHub organization. Give it access to only your GitOps repository.
3. Leave **Enable all public internet access** off. Allow **Package managers**, and add `fleetdm.com` under **Additional allowed domains** so Codex can read Fleet's YAML reference and osquery schema.
4. Codex walks you through setup. This repository needs nothing installed: the `fleet-gitops` validator uses only Python's standard library.
5. Under **Privacy > Who can use**, choose your workspace.
6. Select **Publish** and wait for **Environment published**.

## Add Codex to Slack

1. In the OpenAI admin console ([admin.openai.com](https://admin.openai.com)), select your workspace, open **Agents**, and connect Slack. A Slack owner or app manager approves the ChatGPT app's permissions.
2. Create a surface for your Slack workspace. For a pilot, choose **Selected channels only** and add the channel where your team will request changes. Channel restrictions don't apply to direct messages.
3. Turn on **Codex Cloud** for your ChatGPT workspace. It's off by default in Enterprise workspaces.
4. If your workspace uses Codex Cloud role-based access control, grant Codex Cloud access to the surface's service account and to each person who'll request changes. Tasks run as the person who asked.
5. In Slack, invite `@ChatGPT` to the channel. If your admin requires people to link their ChatGPT account, ChatGPT asks each person to link it the first time they mention it. After linking, they mention `@ChatGPT` again in the same thread.

## Add Codex to Microsoft Teams

1. In the OpenAI admin console, select your workspace, open **Agents**, and connect Microsoft Teams. Sign in with a Microsoft account that has the Global Administrator or Privileged Role Administrator role, and grant consent for your organization.
2. In the Teams admin center, open the ChatGPT app, go to **Users and groups**, and install it for the people who'll use it. Confirm your Teams app policies allow it.
3. Back in the OpenAI admin console, wait for the connection to show **Ready**.
4. Create a surface for your Teams tenant. Teams has no channel restrictions, so control who can use it with **Users and groups** in the Teams admin center.

> **Note:** OpenAI's docs disagree on whether Teams can hand work to Codex Cloud yet. Its setup guide describes Codex Cloud delegation for Slack only. Check [OpenAI's Slack and Teams setup guide](https://help.openai.com/en/articles/20001538-setting-up-and-managing-chatgpt-in-slack-and-microsoft-teams) for current status, and test a request from Teams before you roll it out.

## Connect Codex to Fleet's MCP server

Fleet Premium 4.95 and later include a built-in MCP server at `/api/v1/fleet/mcp`.

OpenAI documents MCP support for the ChatGPT desktop app, Codex CLI, and IDE extension, not for Codex Cloud tasks like the ones `@ChatGPT` starts from Slack and Teams. ChatGPT's custom MCP plugins support OAuth or no authentication, and can't send an API token. So connect Fleet's MCP server where your admins review and refine changes: the ChatGPT desktop app, Codex CLI, or IDE extension.

### Create an API-only user

Create a dedicated [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the Observer+ role, the least privileged role that can run live reports. It can't change anything in Fleet, and its activity is easy to spot in Fleet's activity feed. Copy the user's API token.

> **Note:** To limit the user to specific fleets, give it the Observer+ role on those fleets instead of a global role. To limit it to only the API endpoints Fleet's MCP server needs, see [Create an API-only user](https://fleetdm.com/guides/fleet-mcp#create-an-api-only-user).

### Add the server on each admin's computer

The ChatGPT desktop app, Codex CLI, and IDE extension share `~/.codex/config.toml`. Codex reads the token from the macOS keychain through a small helper script, so the token never sits in a file or an environment variable. The desktop app doesn't always load variables from `~/.zshrc` when you open it from the Dock, and a helper avoids that.

1. Store the token in the macOS keychain. `security` prompts for it:

```bash
security add-generic-password -a "$USER" -s fleet-mcp -w
```

2. Create the helper script. It prints the `Authorization` header as JSON:

```bash
cat > ~/.codex/fleet-mcp-headers.sh <<'EOF'
#!/bin/sh
printf '{"Authorization": "Bearer %s"}' "$(security find-generic-password -s fleet-mcp -w)"
EOF
chmod +x ~/.codex/fleet-mcp-headers.sh
```

3. Add Fleet to `~/.codex/config.toml`. Replace `fleet.example.com` with your Fleet server's host and `you` with your macOS username:

```toml
[mcp_servers.fleet]
url = "https://fleet.example.com/api/v1/fleet/mcp"
http_headers_helper = "/Users/you/.codex/fleet-mcp-headers.sh"
```

4. Restart the ChatGPT desktop app or the IDE extension so it reloads the configuration.

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
2. In the Codex CLI, run `codex mcp list` and confirm the `fleet` server is listed. In the ChatGPT desktop app, check **Settings > MCP servers**.

## Troubleshoot

**`@ChatGPT` answers but doesn't start a Codex task**

Check that **Codex Cloud** is on for your ChatGPT workspace. If your workspace uses Codex Cloud role-based access control, check that both the surface's service account and the person asking have Codex Cloud access.

**`@ChatGPT` can't find the repository**

Check that the environment is published and shared under **Privacy > Who can use**. `@ChatGPT` only uses published environments shared with the workspace. Name the repository in your request, and start a new thread to switch environments.

**The dry run fails on a key Codex added**

Reply in the thread: `@ChatGPT The dry run failed. Run python3 .agents/skills/fleet-gitops/scripts/validate.py, fix what it reports, and check each key against Fleet's GitOps YAML reference.`

**The `fleet` MCP server doesn't connect**

Run `~/.codex/fleet-mcp-headers.sh` in a terminal. It should print the `Authorization` header JSON. If it prints an error, check the keychain item's name, and that `http_headers_helper` in `~/.codex/config.toml` has the script's full path.

If Fleet answers with an error:

- `401`: the token is wrong.
- `402`: your Fleet server isn't on Fleet Premium.
- `404`: your Fleet server is older than 4.95.
- `400` "Fleet's MCP server is disabled": an admin turned it off. Turn it back on in **Settings > Organization settings > Advanced options**.

## Further reading

- [Connect your AI tool to Fleet](https://fleetdm.com/guides/connect-your-ai-tool-to-fleet)
- [ChatGPT in Slack and Microsoft Teams](https://learn.chatgpt.com/docs/enterprise/chatgpt-slack-and-teams)
- [Codex Cloud environments](https://learn.chatgpt.com/docs/environments/cloud-environments)
- [Codex MCP servers](https://learn.chatgpt.com/docs/extend/mcp)
- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Set up Claude in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/claude-in-slack-and-teams-for-fleet-gitops)
- [Set up GitHub Copilot in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/github-copilot-in-slack-and-teams-for-fleet-gitops)
- [Set up Gemini with Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/gemini-with-slack-and-teams-for-fleet-gitops)

<meta name="articleTitle" value="Set up Codex in Slack and Microsoft Teams for Fleet GitOps">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-16">
<meta name="description" value="Ask Codex for Fleet changes in Slack or Teams and get a validated GitOps pull request. Connect it to Fleet's MCP server.">
