# Set up GitHub Copilot in Slack and Microsoft Teams

This guide connects GitHub Copilot to Fleet, Slack, and Microsoft Teams, so anyone on your IT team can mention `@GitHub` in a channel to ask questions about your hosts or request a change to Fleet. For a change, Copilot cloud agent writes it in your GitOps repository, checks it with the `fleet-gitops` skill, and opens a pull request. Your CI runs a dry run, a person reviews it, and merging it deploys it.

This guide has two parts:

- [Fleet's MCP server](#connect-copilot-to-fleets-mcp-server): ask Copilot questions about your hosts, policies, and software, for example "Which fleets don't have a policy that checks FileVault is on?"
- [GitOps repository setup](#add-the-fleet-gitops-skill-to-your-repository): make changes to Fleet from chat. This part needs a Fleet GitOps repository on GitHub, created with `fleetctl new`, with CI applying it to Fleet.

## Prerequisites

- A Fleet GitOps repository on GitHub, created with [`fleetctl new`](https://github.com/fleetdm/fleet/blob/main/cmd/fleetctl/fleetctl/templates/new/README.md), with the `FLEET_URL` and `FLEET_API_TOKEN` secrets set and the **Apply latest configuration to Fleet** workflow passing.
- GitHub Copilot Business or Enterprise for the GitHub organization that owns the repository, and organization owner access to change its Copilot settings.
- For Slack: a Slack workspace admin to install the GitHub app.
- For Microsoft Teams: a Teams admin, if your organization restricts third-party apps.
- To connect Fleet's MCP server: Fleet Premium 4.95 or later, and a Fleet admin account. Fleet 4.95 is [coming soon](https://github.com/fleetdm/fleet/issues/44448).

## Add the fleet-gitops skill to your repository

Copilot reads `AGENTS.md` and discovers skills in `.agents/skills/` on its own. `fleetctl new` 4.95 and later create both, along with the `fleet-gitops` skill. If your repository already has `AGENTS.md` and `.agents/skills/fleet-gitops/SKILL.md`, skip to the next step.

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

3. Push the commit to your default branch. Copilot only reads what's on GitHub.

## Protect your default branch

A merge to your default branch is a deploy: the **Apply latest configuration to Fleet** workflow runs `fleetctl gitops` and pushes the change to every host it targets. Make sure a person approves every pull request before it merges.

1. In your repository, go to **Settings > Rules > Rulesets** and select **New ruleset > New branch ruleset**.
2. Name the ruleset, set **Enforcement status** to **Active**, and add your default branch under **Targets**.
3. Turn on **Require a pull request before merging** and set **Required approvals** to at least 1. Copilot opens pull requests from Slack and Teams channels under its own app identity, so by default they need one more approval than you set here.
4. Turn on **Require status checks to pass** and add the `fleet-gitops` check, so a failed dry run blocks the merge.
5. Select **Create**.

> **Warning:** Don't give Copilot your `FLEET_API_TOKEN`. That token can apply changes to Fleet, not only dry-run them. Copilot checks its work offline with the `fleet-gitops` skill's `validate.py`, and your CI runs the dry run with the token it already has.

## Check that Copilot in the cloud is on

Copilot cloud agent does the work you start from Slack and Teams, in a cloud sandbox. Cloud sandboxes don't have a separate setting. They come with the **Copilot in the cloud** policy.

1. In your GitHub organization, go to **Settings > Copilot > Policies** and make sure **Copilot in the cloud** is **Enabled**. It's on by default for Copilot Business and Enterprise. On GitHub Enterprise, an enterprise owner may need to allow it first under **AI controls > Copilot > Configure features & clients**.
2. Go to **Settings > Copilot > Cloud** and make sure **Repository access** includes your GitOps repository.

> **Note:** GitHub bills cloud sandbox usage separately from Copilot seats, by compute, memory, and storage.

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
2. In Teams, select **Apps**, search for **GitHub**, and select **Add**. If your team already uses **GitHub Notifications**, update that app instead. Your existing subscriptions stay.
3. Add the app to the team and channel where people will request changes.
4. Mention `@GitHub` in the channel and sign in with your GitHub account when prompted.
5. Set the channel's default repository:

```
@GitHub settings
```

## Connect Copilot to Fleet's MCP server

Fleet Premium 4.95 and later include a built-in MCP server at `/api/v1/fleet/mcp`. Connect it so you can ask Copilot questions about your hosts, policies, and software. Copilot also uses it to check what's already in Fleet before it writes a change.

### Create an API-only user

Create a dedicated [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the Observer+ role, the least privileged role that can run live reports. It can't change anything in Fleet, and its activity is easy to spot in Fleet's activity feed. Copy the user's API token.

> **Note:** To limit the user to specific fleets, give it the Observer+ role on those fleets instead of a global role. To limit it to only the API endpoints Fleet's MCP server needs, see [Create an API-only user](https://fleetdm.com/guides/fleet-mcp#create-an-api-only-user).

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
      "url": "https://fleet.example.com/api/v1/fleet/mcp",
      "headers": {
        "Authorization": "$COPILOT_MCP_FLEET_AUTHORIZATION"
      },
      "tools": ["*"]
    }
  }
}
```

4. Select **Save MCP configuration**.
5. Go to **Settings > Copilot > Internet access**. Under **Custom allowlist**, select **Add rule**, enter your Fleet server's host, and select **Save changes**.

Copilot runs MCP tools without asking first. That's why the token above belongs to an Observer+ user, which can't change anything in Fleet.

## Make your first change

1. In a channel with the GitHub app, describe the change and ask for a pull request. Copilot doesn't open one from chat unless you ask:

```
@GitHub Add a macOS policy to the Workstations fleet that checks FileVault is on. Check Fleet first to see if one already exists. Open a pull request.
```

In Teams, add `repo=acme/it-and-security` (your repository) if the channel has no default.

2. In Slack, Copilot opens a dedicated **Slack Code** channel for the task. Steer the session there. In Teams, reply in the same thread and mention `@GitHub`.
3. When Copilot opens the pull request, review the new file under `platforms/macos/policies/` and the change to `fleets/workstations.yml`.
4. Wait for the `fleet-gitops` check to pass. If the pull request shows **Approve and run workflows**, select it first.
5. To ask for changes, comment on the pull request and mention `@copilot`. Copilot pushes a new commit to the same branch.
6. When the check passes and the change looks right, mark the pull request ready for review, approve it, and merge it. The workflow applies it to Fleet.

## Verify

1. In Fleet, go to **Policies**, select the **Workstations** fleet, and confirm the new policy is there.
2. To confirm the MCP connection, open the Copilot session from the pull request, select **...**, then **Copilot**, and expand **Start MCP Servers**. The `fleet` server and its tools are listed at the bottom.

## Troubleshoot

**Copilot answers in the thread but doesn't open a pull request**

Ask for one in your request ("Open a pull request"). Sessions started from Slack and Teams don't open one on their own.

**`@GitHub` doesn't respond in a channel**

Run `/invite @GitHub` in Slack, or add the app to the channel in Teams. If it responds but can't start work, check that **Copilot in the cloud** is enabled for your organization, and that you have write access to the repository.

**The dry run never starts**

If the pull request shows **Approve and run workflows**, select it. Otherwise, check that the **Apply latest configuration to Fleet** workflow is on your default branch.

**The dry run fails on a key Copilot added**

Comment on the pull request: `@copilot The dry run failed. Run python3 .agents/skills/fleet-gitops/scripts/validate.py, fix what it reports, and check each key against Fleet's GitOps YAML reference.`

**The `fleet` MCP server doesn't start**

Check the **Start MCP Servers** step in the session logs. If Fleet answers with an error:

- `401`: the secret's value is missing the `Bearer ` prefix, or the token is wrong.
- `402`: your Fleet server isn't on Fleet Premium.
- `404`: your Fleet server is older than 4.95.
- `400` "Fleet's MCP server is disabled": an admin turned it off. Turn it back on in **Settings > Organization settings > Advanced options**.

## Further reading

- [Connect your AI tool to Fleet](https://fleetdm.com/guides/connect-your-ai-tool-to-fleet)
- [Integrate Copilot cloud agent with Slack](https://docs.github.com/en/copilot/how-tos/copilot-integrations/integrate-cloud-agent-with-slack)
- [Integrate Copilot cloud agent with Microsoft Teams](https://docs.github.com/en/copilot/how-tos/copilot-integrations/integrate-cloud-agent-with-teams)
- [Configure MCP servers for Copilot](https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/configure-mcp-servers)
- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Set up Claude in Slack and Microsoft Teams](https://fleetdm.com/guides/claude-in-slack-and-teams)
- [Set up Codex in Slack and Microsoft Teams](https://fleetdm.com/guides/codex-in-slack-and-teams)
- [Set up Gemini with Slack and Microsoft Teams](https://fleetdm.com/guides/gemini-with-slack-and-teams)

<meta name="articleTitle" value="Set up GitHub Copilot in Slack and Microsoft Teams">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-16">
<meta name="description" value="Ask GitHub Copilot about your hosts and request Fleet changes from Slack or Teams, with Fleet's MCP server and your GitOps repository.">
