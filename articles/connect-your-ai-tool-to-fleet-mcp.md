# Connect your AI tool to Fleet's MCP server

Fleet lets your AI tool look up hosts, policies, and software in Fleet, so you can ask questions about your fleet in plain English. Fleet has a setup guide for Claude, Codex, GitHub Copilot, and Gemini. Each one also connects the tool to Slack or Microsoft Teams and to your GitOps repository, so your team can ask for a change in chat and get a pull request back. Pick the guide for the AI tool your organization already uses.

<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin: 32px 0px;">
<a href="https://fleetdm.com/guides/claude-in-slack-and-teams-for-fleet-gitops" style="display: flex; align-items: center; gap: 12px; padding: 12px 16px; border: 1px solid #E2E4EA; border-radius: 8px; color: #192147; font-weight: 700; text-decoration: none;"><img src="../website/assets/images/icon-claude-ai-28x28@2x.png" alt="" style="width: 28px; height: 28px; padding: 0; margin: 0;">Claude</a>
<a href="https://fleetdm.com/guides/codex-in-slack-and-teams-for-fleet-gitops" style="display: flex; align-items: center; gap: 12px; padding: 12px 16px; border: 1px solid #E2E4EA; border-radius: 8px; color: #192147; font-weight: 700; text-decoration: none;"><img src="../website/assets/images/icon-open-ai-28x28@2x.png" alt="" style="width: 28px; height: 28px; padding: 0; margin: 0;">Codex</a>
<a href="https://fleetdm.com/guides/github-copilot-in-slack-and-teams-for-fleet-gitops" style="display: flex; align-items: center; gap: 12px; padding: 12px 16px; border: 1px solid #E2E4EA; border-radius: 8px; color: #192147; font-weight: 700; text-decoration: none;"><img src="../website/assets/images/icon-github-28x28@2x.png" alt="" style="width: 28px; height: 28px; padding: 0; margin: 0;">GitHub Copilot</a>
<a href="https://fleetdm.com/guides/gemini-with-slack-and-teams-for-fleet-gitops" style="display: flex; align-items: center; gap: 12px; padding: 12px 16px; border: 1px solid #E2E4EA; border-radius: 8px; color: #192147; font-weight: 700; text-decoration: none;"><img src="../website/assets/images/app-icon-google-gemini-60x60@2x.png" alt="" style="width: 28px; height: 28px; padding: 0; margin: 0;">Gemini</a>
</div>

## Prerequisites

- Fleet 4.95 or later. The MCP server is at `https://<your-fleet-url>/api/mcp`.
- `fleetctl` logged in as a Fleet admin, to create a read-only API user for your AI tool.
- To request changes from chat: a Fleet GitOps repository on GitHub, created with `fleetctl new`.

## Pick your guide

| AI tool | Chat | Ask Fleet questions from | Plan you need |
|:--|:--|:--|:--|
| <span style="white-space: nowrap;"><img src="../website/assets/images/icon-claude-ai-28x28@2x.png" alt="" style="display: inline; width: 20px; height: 20px; padding: 0; margin: 0px 8px 0px 0px; vertical-align: -4px;">[Claude](https://fleetdm.com/guides/claude-in-slack-and-teams-for-fleet-gitops)</span> | `@Claude` in Slack (public beta). No Teams app yet. The guide covers a workaround. | Slack, and Claude Code on the web | Claude Team or Enterprise |
| <span style="white-space: nowrap;"><img src="../website/assets/images/icon-open-ai-28x28@2x.png" alt="" style="display: inline; width: 20px; height: 20px; padding: 0; margin: 0px 8px 0px 0px; vertical-align: -4px;">[Codex](https://fleetdm.com/guides/codex-in-slack-and-teams-for-fleet-gitops)</span> | `@ChatGPT` in Slack and Teams | The Codex app, CLI, and IDE extension. Not from chat yet. | ChatGPT Enterprise |
| <span style="white-space: nowrap;"><img src="../website/assets/images/icon-github-28x28@2x.png" alt="" style="display: inline; width: 20px; height: 20px; padding: 0; margin: 0px 8px 0px 0px; vertical-align: -4px;">[GitHub Copilot](https://fleetdm.com/guides/github-copilot-in-slack-and-teams-for-fleet-gitops)</span> | `@GitHub` in Slack and Teams (public preview) | Slack and Teams | Copilot Business or Enterprise, plus cloud sandbox usage |
| <span style="white-space: nowrap;"><img src="../website/assets/images/app-icon-google-gemini-60x60@2x.png" alt="" style="display: inline; width: 20px; height: 20px; padding: 0; margin: 0px 8px 0px 0px; vertical-align: -4px;">[Gemini](https://fleetdm.com/guides/gemini-with-slack-and-teams-for-fleet-gitops)</span> | No chat agent yet. Create a GitHub issue from Slack or Teams, then comment `@gemini-cli`. | GitHub issues, and Antigravity CLI | A Gemini API key with billing, or Vertex AI |

Not sure which one fits? Start here:

- If your organization already pays for one of these plans, use that tool.
- If your team works in Microsoft Teams, GitHub Copilot and Codex have a Teams app. Claude and Gemini need a workaround.
- If you want to ask Fleet questions without leaving chat, choose Claude in Slack, or GitHub Copilot in Slack or Teams.
- If your admins mostly work in a terminal or code editor, Codex or Gemini is a good fit. Their Fleet MCP connections live there.
- If your organization uses Google Cloud, choose Gemini. It runs in your own GitHub Actions and can bill through Vertex AI.

## What every guide sets up

Each guide follows the same five steps, so switching tools later means redoing only the tool-specific ones:

1. Add `AGENTS.md` and the `fleet-gitops` skill to your repository, so the AI tool checks its work against Fleet's GitOps format.
2. Protect your default branch, so a person approves every merge and a failed dry run blocks it.
3. Connect the AI tool to GitHub, and to Slack or Microsoft Teams.
4. Connect it to Fleet's MCP server with a read-only API user.
5. Make your first change and verify it in Fleet.

> **Warning:** Never give an AI tool your GitOps `FLEET_API_TOKEN`. That token can apply changes to Fleet. Each guide creates a separate API-only user with the Observer role, which can read but not change anything.

## Ask your first questions

Once Fleet's MCP server is connected, try a question you'd otherwise answer by clicking through the Fleet UI or writing a report:

```
Which fleets don't have a policy that checks FileVault is on?
```

```
How many macOS hosts haven't checked in to Fleet in the last 7 days?
```

```
Which hosts in the Workstations fleet have software with known vulnerabilities?
```

```
Which policies fail on the most hosts, and which fleets are those hosts in?
```

The AI tool connects as an observer, so asking questions never changes anything in Fleet.

## Use another AI tool

Fleet's MCP server works with other AI tools too, as long as they support remote MCP servers over HTTP and let you set a request header.

1. Create a read-only API user and copy the token that `fleetctl` prints:

```bash
fleetctl user create --name "AI assistant (read-only)" --api-only --global-role observer
```

2. Add Fleet to your tool's MCP configuration. Replace `fleet.example.com` with your Fleet server's host. Many tools use this format:

```json
{
  "mcpServers": {
    "fleet": {
      "type": "http",
      "url": "https://fleet.example.com/api/mcp",
      "headers": {
        "Authorization": "Bearer <token>"
      }
    }
  }
}
```

> **Warning:** Don't commit the token to a repository. Most AI tools can read it from an environment variable instead. Check your tool's MCP docs for the syntax.

> **Note:** On Fleet versions older than 4.95, use the standalone [Fleet MCP server](https://fleetdm.com/guides/fleet-mcp) instead.

## Further reading

- [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet)
- [Why AI-powered device management requires GitOps](https://fleetdm.com/articles/why-ai-powered-device-management-requires-gitops)
- [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)

<meta name="articleTitle" value="Connect your AI tool to Fleet's MCP server">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-21">
<meta name="description" value="Pick the setup guide for Claude, Codex, GitHub Copilot, or Gemini, then ask questions about your fleet with Fleet's MCP server.">
