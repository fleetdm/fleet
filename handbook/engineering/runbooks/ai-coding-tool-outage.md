# AI coding tool outage

Use this runbook when Claude Code is unavailable (Anthropic outage, account issue, or network problem) and you need to keep shipping. Fleet's fallback is GitHub Copilot, which is included with every engineer's GitHub seat.


## Confirm it's actually an outage

Before switching tools, rule out a local issue:

1. Check [status.anthropic.com](https://status.anthropic.com).
2. Check the [#help-engineering](https://fleetdm.slack.com/archives/C019WG4GH0A) Slack channel — someone may have already reported it.
3. If it looks auth-related, try `claude` with a fresh login.


## Set up your fallback before you need it

Verify your Copilot fallback works on a normal day, so you're not first-time-installing during the actual outage.

- **CLI**: install with `npm install -g @github/copilot`, then run `copilot` in any repo and complete the browser-based GitHub auth.
- **IDE**: confirm the GitHub Copilot extension is installed in VS Code or your JetBrains IDE and that you're signed into your GitHub account. Open Copilot Chat and switch to **Agent** mode.


## During an outage

Pick the path that matches how you normally use Claude Code.

### If you use Claude Code in the terminal

Use the GitHub Copilot CLI — it's the closest match for an agentic terminal workflow.

```
copilot
```

If you haven't installed it yet, see "Set up your fallback before you need it" above.

### If you use Claude Code in your IDE

Use Copilot Chat in **Agent** mode:

- **VS Code**: `Cmd+Ctrl+I` opens Copilot Chat. Switch the dropdown from Ask to Agent.
- **JetBrains**: open the Copilot Chat tool window and switch to Agent.

Plain inline completion or Ask mode won't cover Claude Code-style multi-file work — make sure you're in Agent mode.


## After the outage

Switch back to Claude Code once [status.anthropic.com](https://status.anthropic.com) reports the incident is resolved.


<meta name="maintainedBy" value="lukeheath">
<meta name="title" value="AI coding tool outage">
