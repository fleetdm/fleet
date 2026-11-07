# Set up Claude in Slack and Microsoft Teams for Fleet GitOps

Once your Fleet configuration lives in a GitOps repository, most changes are a pull request away. This guide connects Claude to Slack, so anyone on your IT team can mention `@Claude` in a channel, describe a change, and get a pull request back. Claude writes the change and checks it with the `fleet-gitops` skill. Your CI runs a dry run, a person reviews it, and merging it deploys it. You'll also connect Claude to Fleet's MCP server, so it can look up hosts, policies, and software in Fleet while it works.

This guide picks up where Fleet's GitOps level 1 certification leaves off: a repository created with `fleetctl new`, on GitHub, with CI applying it to Fleet. Claude in Slack, which Anthropic calls Claude Tag, is in public beta, so labels and steps may change. Claude doesn't have a Microsoft Teams app. If your team works in Teams, the Microsoft Teams section below covers what works today.

## Prerequisites

- A Fleet GitOps repository on GitHub, created with `fleetctl new`, with the `FLEET_URL` and `FLEET_API_TOKEN` secrets set and the **Apply latest configuration to Fleet** workflow passing.
- A Claude Team or Enterprise plan, and the Owner role in your Claude organization. Claude Tag isn't available to organizations with Zero Data Retention, customer-managed encryption keys, or the HIPAA configuration.
- An owner of the GitHub organization that owns the repository. Linking GitHub is easiest when one person is both a GitHub organization owner and a Claude Owner.
- A Slack workspace admin to install the Claude app and pair the workspace.
- For Microsoft Teams: a Microsoft Entra Global Administrator to approve the Microsoft 365 connector.
- To connect Fleet's MCP server: Fleet 4.95 or later, and `fleetctl` logged in as a Fleet admin.

## Add the fleet-gitops skill to your repository

When Claude clones your repository, it loads `CLAUDE.md` and the skills in `.claude/skills/`. Recent versions of `fleetctl new` create both, with a `CLAUDE.md` that imports `AGENTS.md` and a `fleet-gitops` skill that points Claude at the full skill in `.agents/skills/`. If your repository already has `CLAUDE.md`, `AGENTS.md`, and `.agents/skills/fleet-gitops/SKILL.md`, skip to the next step.

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

3. Push the commit to your default branch. Claude only reads what's on GitHub.

## Protect your default branch

A merge to your default branch is a deploy: the **Apply latest configuration to Fleet** workflow runs `fleetctl gitops` and pushes the change to every host it targets. Make sure a person approves every pull request before it merges.

1. In your repository, go to **Settings > Rules > Rulesets** and select **New ruleset > New branch ruleset**.
2. Name the ruleset, set **Enforcement status** to **Active**, and add your default branch under **Targets**.
3. Turn on **Require a pull request before merging** and set **Required approvals**. Claude is the author of the pull requests it opens, so with 1 required approval, the person who asked Claude for the change can approve and merge it alone. Set it to 2 if you want a second person to review every change.
4. Turn on **Require status checks to pass** and add the `fleet-gitops` check, so a failed dry run blocks the merge.
5. Select **Create**.

> **Warning:** Don't give Claude your `FLEET_API_TOKEN`. That token belongs to an API-only user with the GitOps role, which can apply changes, not only dry-run them. Claude checks its work offline with the `fleet-gitops` skill's `validate.py`, and your CI runs the dry run with the token it already has.

## Connect Claude to GitHub

Claude opens pull requests as the Claude GitHub App, so its work shows up under its own name, not yours.

1. In Claude, go to **Organization settings > Git providers** ([claude.ai/admin-settings/source-control](https://claude.ai/admin-settings/source-control)).
2. Beside **GitHub**, select **Connect** and authorize the app.
3. If your GitHub organization's row reads **Not linked**, select **Link**. If it isn't listed, select **Add organization** and install the app on github.com. When GitHub asks which repositories to allow, choose **Only select repositories** and pick your GitOps repository.
4. Confirm the organization's **Status** reads **Connected**.

## Add Claude to Slack

1. Go to **Admin settings > Capabilities** ([claude.ai/admin-settings/capabilities](https://claude.ai/admin-settings/capabilities)) and turn on **Remote sessions > Routines**. Without it, Claude replies to every mention that it's unavailable.
2. Go to **Organization settings > Claude Tag** ([claude.ai/admin-settings/claude-tag](https://claude.ai/admin-settings/claude-tag)) and select **Start setup**.
3. Select **Add the Claude app**, then **Add to Slack**, and approve the permissions.
4. In Slack, open the channel where your team will request changes and run `/invite @Claude`.
5. As a Slack workspace admin, send `@Claude connect` as a message with no other text. Claude replies with a pairing code that only you can see. The code expires in 15 minutes.
6. Back on the setup page, paste the code into **Paste the pairing code** and select **Pair workspace**.
7. If the setup page shows **Connect GitHub**, select **Skip**. You connected it in the previous section.
8. On a Team plan, buy usage credits when asked. Work in channels draws from your organization's usage balance, not from seats.
9. Set a monthly spend limit, select the channel from step 4, and select **Launch Claude Tag**.

## Give Claude access to your GitOps repository

1. In **Organization settings > Claude Tag**, under **Claude's access**, select the **Connectors** tab and open **GitHub**.
2. Under **Assign access**, find the row for your channel. If it isn't listed, select **Add place**, choose the channel, and select **Add**.
3. In the channel's **Access** column, select **Select access**, pick your GitOps repository, and select **Save changes**.

Claude only clones a repository when a request names it. Ask your team to name the repository in the first message of each request, for example `acme/it-and-security`.

## Use Claude with Microsoft Teams

Claude doesn't have an app for Microsoft Teams, so people can't mention Claude in a Teams channel. What you can do today is connect Microsoft 365, so Claude can read a request from a Teams thread, make the change in Claude Code on the web, and post the pull request back to the thread.

1. In Claude, go to **Organization settings > Connectors**, select **Add > All available**, find **Microsoft 365**, and select **Add to your team**.
2. Connect your account and have a Microsoft Entra Global Administrator grant consent for your tenant.
3. To let Claude post back to Teams, turn on write actions for the connector. Then open it under **Organization settings > Connectors**, find **Tool permission restrictions**, and allow the Teams messaging tools.
4. Each person who'll use it goes to **Customize > Connectors**, finds **Microsoft 365**, and selects **Connect**.

To make a change from a Teams request:

1. Go to [claude.ai/code](https://claude.ai/code), choose your GitOps repository, and make sure the Microsoft 365 connector is on for the session.
2. Ask Claude to pick up the request, for example:

```
Read the latest message in the IT requests channel in Teams and make that change in this repository. When you're done, reply in that Teams thread with what you changed.
```

3. Review the diff, then select **Create PR** and choose a draft.

> **Note:** Cloud sessions on [claude.ai/code](https://claude.ai/code) are included in Team plan seats, and in Enterprise seats that include Claude Code.

## Connect Claude to Fleet's MCP server

Fleet 4.95 and later include a read-only MCP server at `https://<your-fleet-url>/api/mcp`. Connect it so Claude can check what's already in Fleet before it writes a change, for example which fleets already have a FileVault policy.

### Create a read-only API user

Create a dedicated API-only user with the Observer role. The connection can read but never change anything, and its activity is easy to spot in Fleet's activity feed:

```bash
fleetctl user create --name "AI assistant (read-only)" --api-only --global-role observer
```

`fleetctl` prints the user's API token once. Copy it.

> **Note:** To limit the user to specific fleets, replace `--global-role observer` with `--fleet <fleet_id>:observer`, once per fleet.

### Connect it in Slack

Claude Tag doesn't load MCP servers from a repository's `.mcp.json`. Instead, you upload a small plugin that tells Claude where the server is, and add a credential that Claude's proxy attaches to requests for your Fleet server. The token never enters Claude's sandbox.

1. Create a folder named `fleet-mcp` with two files. Replace `fleet.example.com` with your Fleet server's host.

`fleet-mcp/.claude-plugin/plugin.json`:

```json
{
  "name": "fleet-mcp",
  "description": "Read-only access to Fleet through Fleet's MCP server",
  "version": "1.0.0"
}
```

`fleet-mcp/.mcp.json`:

```json
{
  "mcpServers": {
    "fleet": {
      "type": "http",
      "url": "https://fleet.example.com/api/mcp"
    }
  }
}
```

2. Zip the folder's contents, so `.claude-plugin/plugin.json` sits at the root of the archive:

```bash
cd fleet-mcp && zip -r ../fleet-mcp.zip .claude-plugin .mcp.json
```

3. Go to **Organization settings > Plugins & skills**, select **Add > Upload a plugin**, and upload `fleet-mcp.zip`.
4. Go to **Organization settings > Claude Tag**. Under **Claude's access**, select the **Skills and plugins** tab and add `fleet-mcp`.
5. Select the **Connectors** tab, then **Add > Custom connector**, and fill in the form:
   - **Name**: Fleet MCP
   - **Credential type**: **Bearer**, with the token from `fleetctl`
   - **Allowed websites**: your Fleet server's host, for example `fleet.example.com`
   - **Path prefixes**: `/api/mcp`
6. Save the connector.

### Connect it in Claude Code on the web

Cloud sessions on [claude.ai/code](https://claude.ai/code), including the Teams workflow above, load MCP servers from your repository's `.mcp.json`.

1. Add `.mcp.json` to the root of your GitOps repository with the same `mcpServers` block as the plugin above, and commit it to your default branch.
2. As a Claude Owner, go to **Admin settings > Cloud environments**, open the environment your team uses for editing, and find **Network secrets**.
3. Select **Add secret**, keep **Secret type** set to **Bearer**, paste the token, add your Fleet server's host as the allowed host, and save.

## Make your first change

1. In a Slack channel where Claude is on, name the repository, describe the change, and ask for a draft pull request:

```
@Claude In acme/it-and-security, add a macOS policy to the Workstations fleet that checks FileVault is on. Check Fleet first to see if one already exists. Open a draft pull request.
```

2. Claude replies in a thread and shows a checklist as it works. Reply in the thread to steer it. You don't need to mention `@Claude` again.
3. When Claude opens the pull request, review the new file under `platforms/macos/policies/` and the change to `fleets/workstations.yml`.
4. Wait for the `fleet-gitops` check. Claude follows the pull request's CI and reports the result in the thread. If the dry run fails, ask Claude in the thread to fix it.
5. When the check passes and the change looks right, mark the pull request ready for review, approve it, and merge it. The workflow applies it to Fleet.

## Verify

1. In Fleet, go to **Policies**, select the **Workstations** fleet, and confirm the new policy is there.
2. In a new Slack thread, ask `@Claude what can you access from this channel?` The reply lists GitHub with your GitOps repository, and Fleet MCP.
3. In a claude.ai/code session, run `/mcp` and confirm the `fleet` server is connected.

## Troubleshoot

**Claude replies "Claude isn't on in this channel yet"**

Finish setup on the Claude Tag page and select **Launch Claude Tag**.

**Claude replies to every mention that it's unavailable**

Turn on **Remote sessions > Routines** in **Admin settings > Capabilities**.

**Claude replies "That environment or repo isn't configured for Claude Code"**

Check that your GitHub organization shows **Connected** under **Organization settings > Git providers**, and that the repository is in the channel's **Claude's access** table. Then start a new thread and name the repository in the first message.

**Claude can't reach Fleet MCP**

Start a new thread. Threads that started before you added the connector don't pick it up. If it still fails, open **Fleet MCP** on the **Connectors** tab. **Not enabled** in its **Access from** table means no rule sends the credential yet, so select **Enable**. A `401` means the token is wrong. A connection error means your Fleet server is older than 4.95.

**The dry run fails on a key Claude added**

Reply in the thread: `The dry run failed. Run python3 .agents/skills/fleet-gitops/scripts/validate.py, fix what it reports, and check each key against Fleet's GitOps YAML reference.`

## Further reading

- [Connect your AI tool to Fleet's MCP server](https://fleetdm.com/guides/connect-your-ai-tool-to-fleet-mcp)
- [Set up Claude Tag](https://claude.com/docs/claude-tag/admins/setup-overview)
- [Configure GitHub access for Claude Tag](https://claude.com/docs/claude-tag/admins/configure-github)
- [Connect a custom MCP server to Claude Tag](https://claude.com/docs/claude-tag/admins/connections/custom)
- [Microsoft 365 connector](https://claude.com/docs/connectors/microsoft/365)
- [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet)
- [Set up GitHub Copilot in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/github-copilot-in-slack-and-teams-for-fleet-gitops)
- [Set up Codex in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/codex-in-slack-and-teams-for-fleet-gitops)
- [Set up Gemini with Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/gemini-with-slack-and-teams-for-fleet-gitops)

<meta name="articleTitle" value="Set up Claude in Slack and Microsoft Teams for Fleet GitOps">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-16">
<meta name="description" value="Ask Claude for Fleet changes in Slack and get a validated GitOps pull request. Covers Microsoft Teams and Fleet's MCP server.">
