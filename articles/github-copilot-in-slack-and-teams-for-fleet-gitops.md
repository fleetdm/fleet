# Set up GitHub Copilot in Slack and Microsoft Teams for Fleet GitOps

Once your Fleet configuration lives in a GitOps repository, most changes are a pull request away. This guide connects GitHub Copilot to Slack and Microsoft Teams, so anyone on your IT team can mention `@GitHub` in a channel, describe a change, and get a pull request back. Copilot cloud agent writes the change and checks it with the `fleet-gitops` skill. Your CI runs a dry run, a person reviews it, and merging it deploys it. You'll also connect Copilot to Fleet's MCP server, so it can look up hosts, policies, and software in Fleet while it works.

This guide picks up where Fleet's GitOps level 1 certification leaves off: a repository created with `fleetctl new`, on GitHub, with CI applying it to Fleet. Copilot in Slack and Teams is in public preview, so labels and steps may change.

## Prerequisites

- A Fleet GitOps repository on GitHub, created with `fleetctl new`, with the `FLEET_URL` and `FLEET_API_TOKEN` secrets set and the **Apply latest configuration to Fleet** workflow passing.
- GitHub Copilot Business or Enterprise for the GitHub organization that owns the repository, and organization owner access to change its Copilot settings.
- For Slack: a Slack workspace admin to install the GitHub app.
- For Microsoft Teams: a Teams admin, if your organization restricts third-party apps.
- To connect Fleet's MCP server: Fleet 4.95 or later, and `fleetctl` logged in as a Fleet admin.

## Add the fleet-gitops skill to your repository

Copilot reads `AGENTS.md` and discovers skills in `.agents/skills/` on its own. Recent versions of `fleetctl new` create both, along with the `fleet-gitops` skill. If your repository already has `AGENTS.md` and `.agents/skills/fleet-gitops/SKILL.md`, skip to the next step.

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

3. Push the commit to your default branch. Copilot only reads what's on GitHub.

## Protect your default branch

A merge to your default branch is a deploy: the **Apply latest configuration to Fleet** workflow runs `fleetctl gitops` and pushes the change to every host it targets. Make sure a person approves every pull request before it merges.

1. In your repository, go to **Settings > Rules > Rulesets** and select **New ruleset > New branch ruleset**.
2. Name the ruleset, set **Enforcement status** to **Active**, and add your default branch under **Targets**.
3. Turn on **Require a pull request before merging** and set **Required approvals** to at least 1.
4. Turn on **Require status checks to pass** and add the `fleet-gitops` check, so a failed dry run blocks the merge.
5. Select **Create**.

> **Warning:** Don't give Copilot your `FLEET_API_TOKEN`. That token belongs to an API-only user with the GitOps role, which can apply changes, not only dry-run them. Copilot checks its work offline with the `fleet-gitops` skill's `validate.py`, and your CI runs the dry run with the token it already has.

## Turn on Copilot cloud agent

Copilot cloud agent does the work you start from Slack and Teams. Sessions started from chat run in a cloud sandbox, which is off by default.

1. In your GitHub organization, go to **Settings > Copilot > Policies** and set **Copilot cloud agent** to **Enabled**. On GitHub Enterprise, an enterprise owner may need to allow it first under **AI controls > Agents**.
2. Go to **Settings > Sandboxes**, set **Cloud sandbox access** to **Enabled**, and select **Save**.
3. Go to **Settings > Copilot > Cloud agent** and make sure **Repository access** includes your GitOps repository.

> **Note:** GitHub bills cloud sandbox usage separately from Copilot seats, by compute time and storage.

In your repository's **Settings > Copilot > Cloud agent**, leave **Require approval for workflow runs** on. GitHub Actions won't run on Copilot's pull requests until a person selects **Approve and run workflows**, so the dry run starts when someone is looking.

## Add Copilot to Slack

1. As a Slack workspace admin, go to [slack.github.com](https://slack.github.com/) and select **Add to Slack**. On Slack Enterprise Grid, add the app to each workspace that needs it.
2. In the channel where your team will request changes, run `/invite @GitHub`.
3. Mention `@GitHub` in the channel. Slack asks each person to connect their GitHub account the first time.
4. Set the channel's default repository so people don't have to name it every time:

```
@GitHub settings
```

Choose your GitOps repository.

> **Note:** People need write access to the repository to start a session. Guests and outside collaborators can't start or steer one.

## Add Copilot to Microsoft Teams

1. If your organization restricts third-party apps, a Teams admin allows the app first: in the Teams admin center, go to **Teams apps > Manage apps**, find **GitHub**, and select **Allow**.
2. In Teams, select **Apps**, search for **GitHub**, and select **Add**. Install the app published by GitHub, not **GitHub Notifications**.
3. Add the app to the team and channel where people will request changes.
4. Mention `@GitHub` in the channel and sign in with your GitHub account when prompted.
5. Set the channel's default repository:

```
@GitHub settings
```

## Connect Copilot to Fleet's MCP server

Fleet 4.95 and later include a read-only MCP server at `https://<your-fleet-url>/api/mcp`. Connect it so Copilot can check what's already in Fleet before it writes a change, for example which fleets already have a FileVault policy.

### Create a read-only API user

Create a dedicated API-only user with the Observer role. The connection can read but never change anything, and its activity is easy to spot in Fleet's activity feed:

```bash
fleetctl user create --name "AI assistant (read-only)" --api-only --global-role observer
```

`fleetctl` prints the user's API token once. Copy it.

> **Note:** To limit the user to specific fleets, replace `--global-role observer` with `--fleet <fleet_id>:observer`, once per fleet.

### Store the token as an Agents secret

1. In your GitOps repository, go to **Settings > Secrets and variables > Agents**.
2. Select **New repository secret**.
3. Name it `COPILOT_MCP_FLEET_AUTHORIZATION`. Copilot only passes secrets that start with `COPILOT_MCP_` to MCP servers.
4. For the value, enter `Bearer ` followed by the token, for example `Bearer abc123...`.
5. Select **Add secret**.

### Add the MCP configuration

1. In your organization's **Settings > Copilot > Policies**, make sure **MCP servers in Copilot** is **Enabled**.
2. In your GitOps repository, go to **Settings > Copilot > MCP servers**.
3. Paste this into **MCP configuration**, replacing `fleet.example.com` with your Fleet server's host:

```json
{
  "mcpServers": {
    "fleet": {
      "type": "http",
      "url": "https://fleet.example.com/api/mcp",
      "headers": {
        "Authorization": "$COPILOT_MCP_FLEET_AUTHORIZATION"
      },
      "tools": ["*"]
    }
  }
}
```

4. Select **Save MCP configuration**.
5. Go to **Settings > Copilot > Internet access**, add your Fleet server's host under **Custom allowlist**, and select **Save changes**.

Copilot runs MCP tools without asking first. That's why the token above belongs to a read-only user.

## Make your first change

1. In a channel with the GitHub app, describe the change and ask for a pull request. Copilot doesn't open one from chat unless you ask:

```
@GitHub Add a macOS policy to the Workstations fleet that checks FileVault is on. Check Fleet first to see if one already exists. Open a pull request.
```

In Teams, add `repo=acme/it-and-security` (your repository) if the channel has no default.

2. Copilot replies in the thread with a link to the session. Reply in the thread to steer it.
3. When Copilot opens the pull request, review the new file under `platforms/macos/policies/` and the change to `fleets/workstations.yml`.
4. Select **Approve and run workflows** to start the dry run, and wait for the `fleet-gitops` check to pass.
5. To ask for changes, comment on the pull request and mention `@copilot`. Copilot pushes a new commit to the same branch.
6. When the check passes and the change looks right, mark the pull request ready for review, approve it, and merge it. The workflow applies it to Fleet.

## Verify

1. In Fleet, go to **Policies**, select the **Workstations** fleet, and confirm the new policy is there.
2. To confirm the MCP connection, open the Copilot session from the pull request, select **...**, then **Copilot**, and expand **Start MCP Servers**. The `fleet` server and its tools are listed at the bottom.

## Troubleshoot

**Copilot answers in the thread but doesn't open a pull request**

Ask for one in your request ("Open a pull request"). Sessions started from Slack and Teams don't open one on their own.

**`@GitHub` doesn't respond in a channel**

Run `/invite @GitHub` in Slack, or add the app to the channel in Teams. If it responds but can't start work, check that **Copilot cloud agent** and **Cloud sandbox access** are both enabled for your organization.

**The dry run never starts**

Open the pull request and select **Approve and run workflows**. Copilot's pull requests wait for a person to approve workflow runs.

**The dry run fails on a key Copilot added**

Comment on the pull request: `@copilot The dry run failed. Run python3 .agents/skills/fleet-gitops/scripts/validate.py, fix what it reports, and check each key against Fleet's GitOps YAML reference.`

**The `fleet` MCP server doesn't start**

Check the **Start MCP Servers** step in the session logs. A `401` means the secret's value is missing the `Bearer ` prefix or the token is wrong. A connection error means the Fleet host isn't on the custom allowlist, or your Fleet server is older than 4.95.

## Further reading

- [Connect your AI tool to Fleet's MCP server](https://fleetdm.com/guides/connect-your-ai-tool-to-fleet-mcp)
- [Integrate Copilot cloud agent with Slack](https://docs.github.com/en/copilot/how-tos/copilot-integrations/integrate-cloud-agent-with-slack)
- [Integrate Copilot cloud agent with Microsoft Teams](https://docs.github.com/en/copilot/how-tos/copilot-integrations/integrate-cloud-agent-with-teams)
- [Configure MCP servers for Copilot](https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/configure-mcp-servers)
- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Set up Claude in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/claude-in-slack-and-teams-for-fleet-gitops)
- [Set up Codex in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/codex-in-slack-and-teams-for-fleet-gitops)

<meta name="articleTitle" value="Set up GitHub Copilot in Slack and Microsoft Teams for Fleet GitOps">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-16">
<meta name="description" value="Ask GitHub Copilot for Fleet changes in Slack or Teams and get a validated GitOps pull request. Connect it to Fleet's MCP server.">
