# Set up Gemini with Slack and Microsoft Teams for Fleet GitOps

Once your Fleet configuration lives in a GitOps repository, most changes are a pull request away. This guide connects Google's Gemini to your repository, so anyone on your IT team can turn a Slack or Teams message into a GitHub issue, mention `@gemini-cli`, and get a pull request back. Gemini plans the change, waits for a person to approve the plan, and then writes it. Your CI runs a dry run, a person reviews it, and merging it deploys it. You'll also connect Gemini to Fleet's MCP server, so it can look up hosts, policies, and software in Fleet while it works.

This guide picks up where Fleet's GitOps level 1 certification leaves off: a repository created with `fleetctl new`, on GitHub, with CI applying it to Fleet. Google doesn't have a Gemini agent you can mention in Slack or Teams that changes code yet. The Gemini Enterprise app for Slack answers search questions only. So this guide uses Google's [`run-gemini-cli`](https://github.com/google-github-actions/run-gemini-cli) GitHub Action, which is in beta, and the GitHub app for Slack and Teams to get requests from chat into GitHub.

## Prerequisites

- A Fleet GitOps repository on GitHub, created with `fleetctl new`, with the `FLEET_URL` and `FLEET_API_TOKEN` secrets set and the **Apply latest configuration to Fleet** workflow passing.
- A Gemini API key with billing turned on, or a Google Cloud project with Vertex AI and Workload Identity Federation set up for GitHub Actions.
- An owner of the GitHub organization that owns the repository, to create and install a GitHub App.
- For Slack: a Slack workspace admin to install the GitHub app.
- For Microsoft Teams: a Teams admin, if your organization restricts third-party apps.
- To connect Fleet's MCP server: Fleet 4.95 or later, and `fleetctl` logged in as a Fleet admin.
- To use Gemini on your computer: Antigravity CLI, with a Gemini Enterprise Standard, Plus, or Pay-as-you-go plan, or a Gemini API key.

## Add the fleet-gitops skill to your repository

Gemini CLI, which runs inside the Action, reads `GEMINI.md` and discovers skills in `.agents/skills/`. Recent versions of `fleetctl new` create a `GEMINI.md` that imports `AGENTS.md`, along with the `fleet-gitops` skill. Antigravity CLI reads `AGENTS.md` and `.agents/skills/` directly. If your repository already has `GEMINI.md`, `AGENTS.md`, and `.agents/skills/fleet-gitops/SKILL.md`, skip to the next step.

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

3. Push the commit to your default branch. The Action only reads what's on GitHub.

## Protect your default branch

A merge to your default branch is a deploy: the **Apply latest configuration to Fleet** workflow runs `fleetctl gitops` and pushes the change to every host it targets. Make sure a person approves every pull request before it merges.

1. In your repository, go to **Settings > Rules > Rulesets** and select **New ruleset > New branch ruleset**.
2. Name the ruleset, set **Enforcement status** to **Active**, and add your default branch under **Targets**.
3. Turn on **Require a pull request before merging** and set **Required approvals** to at least 1.
4. Turn on **Require status checks to pass** and add the `fleet-gitops` check, so a failed dry run blocks the merge.
5. Select **Create**.

> **Warning:** Don't give Gemini your `FLEET_API_TOKEN`, and don't add it to Gemini's workflows. That token belongs to an API-only user with the GitOps role, which can apply changes, not only dry-run them. In the Action, Gemini edits files through GitHub's API and can't run the `fleet-gitops` skill's `validate.py`, so your CI's dry run is the check that catches mistakes.

## Create a GitHub App for Gemini

Gemini's workflows sign in to GitHub as a GitHub App. Its pull requests show up under the app's name, and your `fleet-gitops` check runs on them right away. Without an app, Gemini falls back to the workflow's built-in token, and the dry run waits until someone selects **Approve workflows to run**.

1. In your GitHub organization, go to **Settings > Developer settings > GitHub Apps** and select **New GitHub App**.
2. Name the app, for example `Acme Gemini`, and enter your repository's URL as the **Homepage URL**.
3. Under **Webhook**, clear **Active**.
4. Under **Repository permissions**, set **Contents**, **Issues**, and **Pull requests** to **Read and write**.
5. Select **Create GitHub App** and copy the **App ID**.
6. Under **Private keys**, select **Generate a private key**. Your browser downloads a `.pem` file.
7. Select **Install App**, choose your organization, select **Only select repositories**, and pick your GitOps repository.

## Add the Gemini workflows to your repository

1. From the root of your repository, download Google's dispatch and assistant workflows:

```bash
base=https://raw.githubusercontent.com/google-github-actions/run-gemini-cli/main/examples/workflows
mkdir -p .github/workflows
curl -o .github/workflows/gemini-dispatch.yml $base/gemini-dispatch/gemini-dispatch.yml
curl -o .github/workflows/gemini-invoke.yml $base/gemini-assistant/gemini-invoke.yml
curl -o .github/workflows/gemini-plan-execute.yml $base/gemini-assistant/gemini-plan-execute.yml
```

2. Open `.github/workflows/gemini-dispatch.yml` and remove what calls the workflows you didn't download:
   - Under `on:`, delete the `pull_request` and `issues` triggers. Otherwise Gemini comments on every new issue and pull request.
   - Delete the `review` and `triage` jobs.
   - In the `fallthrough` job, delete `'review'` and `'triage'` from the `needs` list.
3. In your repository, go to **Settings > Secrets and variables > Actions** and add:
   - A variable named `APP_ID`, with the app ID from the previous section.
   - A secret named `APP_PRIVATE_KEY`, with the contents of the `.pem` file.
   - A secret named `GEMINI_API_KEY`, with your Gemini API key. To use Vertex AI instead, follow [the Action's authentication guide](https://github.com/google-github-actions/run-gemini-cli/blob/main/docs/authentication.md). Set up only one method.
4. Commit the workflows and push them to your default branch. They only run from there.

> **Note:** Only people with the Owner, Member, or Collaborator role on the repository can start Gemini. Comments from anyone else are ignored.

## Add the GitHub app to Slack

1. As a Slack workspace admin, go to [slack.github.com](https://slack.github.com/) and select **Add to Slack**.
2. In the channel where your team will request changes, run `/invite @GitHub`.
3. Each person who'll request changes connects their GitHub account the first time they use the app.

> **Warning:** Don't mention `@GitHub` with a request. The GitHub app sends those to GitHub Copilot, not Gemini. Create an issue and mention `@gemini-cli` in a comment instead.

## Add the GitHub app to Microsoft Teams

1. If your organization restricts third-party apps, a Teams admin allows the app first: in the Teams admin center, go to **Teams apps > Manage apps**, find **GitHub**, and select **Allow**.
2. In Teams, select **Apps**, search for **GitHub**, and select **Add**. Install the app published by GitHub, not **GitHub Notifications**.
3. Add the app to the team and channel where people will request changes, and sign in with your GitHub account when prompted.

## Connect Gemini to Fleet's MCP server

Fleet 4.95 and later include a read-only MCP server at `https://<your-fleet-url>/api/mcp`. Connect it so Gemini can check what's already in Fleet before it plans a change, for example which fleets already have a FileVault policy.

### Create a read-only API user

Create a dedicated API-only user with the Observer role. The connection can read but never change anything, and its activity is easy to spot in Fleet's activity feed:

```bash
fleetctl user create --name "AI assistant (read-only)" --api-only --global-role observer
```

`fleetctl` prints the user's API token once. Copy it.

> **Note:** To limit the user to specific fleets, replace `--global-role observer` with `--fleet <fleet_id>:observer`, once per fleet.

### Connect it in GitHub Actions

1. In your repository, go to **Settings > Secrets and variables > Actions** and add a secret named `FLEET_MCP_TOKEN`, with the token from `fleetctl`.
2. In `.github/workflows/gemini-invoke.yml`, find the **Run Gemini CLI** step. Add the secret under its `env:` block:

```yaml
FLEET_MCP_TOKEN: '${{ secrets.FLEET_MCP_TOKEN }}'
```

3. In the same step's `settings`, add a `fleet` server to `mcpServers`. Put it after the `github` server, and add a comma after the `github` server's closing brace. Replace `fleet.example.com` with your Fleet server's host:

```json
"fleet": {
  "httpUrl": "https://fleet.example.com/api/mcp",
  "headers": {
    "Authorization": "Bearer ${FLEET_MCP_TOKEN}"
  }
}
```

4. Make the same two changes in `.github/workflows/gemini-plan-execute.yml`, then commit and push.

The Action runs Gemini CLI without asking before each tool call, including MCP tools. That's why the token belongs to a read-only user.

### Connect it in Antigravity CLI

Admins who use Gemini on their own computers can connect Fleet's MCP server to Antigravity CLI, Google's replacement for Gemini CLI.

1. Install Antigravity CLI:

```bash
curl -fsSL https://antigravity.google/cli/install.sh | bash
```

2. Add Fleet to `~/.gemini/config/mcp_config.json`, replacing `fleet.example.com` and `<token>`. Use this file in your home folder, not `.agents/mcp_config.json` in the repository, so the token never gets committed.

```json
{
  "mcpServers": {
    "fleet": {
      "serverUrl": "https://fleet.example.com/api/mcp",
      "headers": {
        "Authorization": "Bearer <token>"
      }
    }
  }
}
```

3. Run `agy` from your GitOps repository and trust the folder when asked.

## Make your first change

1. In a Slack channel with the GitHub app, open an issue in your GitOps repository:

```
/github open acme/it-and-security
```

Give it a title, and describe the change in the description, for example: "Add a macOS policy to the Workstations fleet that checks FileVault is on. Check Fleet first to see if one already exists."

In Teams, hover over the message with the request, select **...**, then **More actions > Create an Issue**.

2. On the issue preview in Slack or Teams, select **Comment** and start your comment with `@gemini-cli`:

```
@gemini-cli Make the change described in this issue and open a pull request.
```

Mentioning `@gemini-cli` in the issue's description doesn't start Gemini. It has to be a comment that starts with `@gemini-cli`.

3. Gemini replies on the issue with a plan. Read it.
4. If the plan looks right, comment `@gemini-cli /approve`. Gemini creates a branch, writes the change, and opens a pull request.
5. Review the new file under `platforms/macos/policies/` and the change to `fleets/workstations.yml`, and wait for the `fleet-gitops` check to pass.
6. To ask for changes, comment on the issue, starting with `@gemini-cli`, and approve the new plan. Close the earlier pull request if Gemini opens a new one.
7. When the check passes and the change looks right, approve the pull request and merge it. The workflow applies it to Fleet.

## Verify

1. In Fleet, go to **Policies**, select the **Workstations** fleet, and confirm the new policy is there.
2. On any issue in your GitOps repository, comment `@gemini-cli Which fleets have a policy that checks FileVault is on? Check Fleet.` Gemini answers with what it found in Fleet.
3. In Antigravity CLI, run `/mcp` and confirm the `fleet` server is connected.

## Troubleshoot

**Gemini doesn't respond to a comment**

Check that the comment starts with `@gemini-cli`, that you're an Owner, Member, or Collaborator on the repository, and that the workflows are on your default branch. A mention in the issue's description doesn't count.

**Gemini posts a plan but doesn't open a pull request**

Comment `@gemini-cli /approve` on the issue. Gemini only writes changes after a person approves the plan.

**The dry run never starts on Gemini's pull request**

Check that the `APP_ID` variable and `APP_PRIVATE_KEY` secret are set and that the app is installed on the repository. Without them, open the pull request and select **Approve workflows to run**.

**A Gemini run fails with an authentication error**

Check that billing is turned on for your `GEMINI_API_KEY`, and that you set up only one authentication method. To see more detail, add a repository variable named `GEMINI_DEBUG` set to `true` and run it again.

**The dry run fails on a key Gemini added**

Comment on the issue: `@gemini-cli The dry run failed on the pull request. Read its fleet-gitops check output, fix what it reports, and check each key against Fleet's GitOps YAML reference.` Then approve the new plan.

**Gemini can't reach Fleet MCP**

Check that `FLEET_MCP_TOKEN` is under the step's `env:` block in both workflows, and that the server's name doesn't contain an underscore. A `401` means the token is wrong. A connection error means your Fleet server is older than 4.95.

## Further reading

- [Connect your AI tool to Fleet's MCP server](https://fleetdm.com/guides/connect-your-ai-tool-to-fleet-mcp)
- [run-gemini-cli GitHub Action](https://github.com/google-github-actions/run-gemini-cli)
- [Antigravity CLI MCP servers](https://antigravity.google/docs/mcp)
- [Create GitHub issues from Slack](https://docs.github.com/en/integrations/tutorials/slack/create-issues)
- [Create GitHub issues from Microsoft Teams](https://docs.github.com/en/integrations/tutorials/teams/create-issues)
- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)
- [Set up Claude in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/claude-in-slack-and-teams-for-fleet-gitops)
- [Set up Codex in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/codex-in-slack-and-teams-for-fleet-gitops)
- [Set up GitHub Copilot in Slack and Microsoft Teams for Fleet GitOps](https://fleetdm.com/guides/github-copilot-in-slack-and-teams-for-fleet-gitops)

<meta name="articleTitle" value="Set up Gemini with Slack and Microsoft Teams for Fleet GitOps">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-21">
<meta name="description" value="Turn a Slack or Teams request into a Fleet GitOps pull request with Gemini, and connect Gemini to Fleet's MCP server.">
