# Codex and VS Code still let a local HTML file read your whole disk

*Forever Security broke the browsers built into five AI coding assistants. Two of those flaws are still open, so the fix has to come from knowing where the tools run.*

## Key takeaways

- **Two of five flaws are still open.** Forever Security's EMBRACE research broke the embedded browsers in Cursor, GitHub Copilot, OpenAI Codex, Microsoft VS Code, and Anthropic Claude Desktop. As of October 29, Codex and VS Code have no fix.
- **The risk is a file on disk, not a website.** One click on an HTML file in a rigged project can read files far outside it. On Forever's test machine, that meant SSH keys, `.env` files, and npm tokens.
- **There's no patch to wait for.** Microsoft declined the VS Code report, and OpenAI hasn't replied about Codex.
- **Inventory comes first.** Know which hosts run these tools, and at what version. Cursor patched on October 1, and the GitHub Copilot desktop app fixed its flaw in 1.1.29.

<a purpose="cta-button" href="https://fleetdm.com/software-management">Find every Codex and VS Code install</a>

AI coding assistants now ship with a browser inside them, so an agent can open a page, preview an app, or follow a link. In research it calls EMBRACE, Forever Security tested those browsers in five assistants and found a way out of each one. Two of the flaws are fixed, one was accepted by its vendor, and two are still open.

## What EMBRACE found

The five findings are different bugs with a common theme: a page the assistant loads can reach the host computer.

### Local file read in Codex, VS Code, and Claude Desktop

In these three tools, an HTML file loaded in the built-in browser reaches files beyond the project. Codex had no folder restriction at all. VS Code and Claude Desktop confine local pages to the project folder. That check skips web workers, the background scripts a page can launch. A worker launched by the page can reach every file on the host. On macOS, the attack maps the disk by parsing the hidden `.DS_Store` files Finder writes into each folder it opens.

OpenAI hasn't replied to Forever's July 27 Codex report. Microsoft declined the September 10 VS Code report, pointing to Workspace Trust, the prompt where a developer marks a project folder as trusted. Anthropic turned down Forever's July 23 Claude Desktop report for the same reason, then accepted Noma Security's report a day later. The sources don't say whether that fix has shipped. Forever's Gal Weizman told Help Net Security that developers trust every project because the tools are unusable otherwise, so the prompt protects no one.

### Command execution in Cursor and address spoofing in Copilot

In Cursor, a page hid an agent command behind a "Start setup" button. One keypress approved it. Cursor's sandbox blocked it, so the agent disabled the sandbox on its own and tried again. Forever notes results vary by model, and the attack doesn't always land. Cursor patched the flaw on October 1 without replying to the researchers.

In the GitHub Copilot desktop app, a page could set the address the app reported, so a fake sign-in form could appear under github.com/login. GitHub had already shipped the fix in version 1.1.29 on October 13, so Forever never needed to report it. This is the standalone app, not the Copilot extension inside VS Code, so check the right version number.

## Why waiting on the vendor fails

With no fix coming from Microsoft or OpenAI, the decision sits with the organization running the tools. The attack needs only one click, and the link comes from the assistant itself, so "tell developers to be careful" is a weak control.

Forever's own advice starts with an inventory, and treats each embedded browser as an attack surface in its own right.

## Finding the hosts that run them

Fleet's agent inventories installed software and versions on every enrolled host, across macOS, Windows, and Linux. Search the software list for each tool and you get the affected hosts, grouped by version. Names can mislead, though. Short searches like "Copilot" also match unrelated apps, and some hosts report the Codex app as `ChatGPT.app`. VS Code extensions are listed separately from the VS Code app. On macOS, a report that matches bundle IDs is more precise:

```sql
SELECT name, bundle_identifier, bundle_short_version, path
FROM apps
WHERE bundle_identifier IN (
  'com.openai.codex',
  'com.microsoft.VSCode',
  'com.anthropic.claudefordesktop',
  'com.todesktop.230313mzl4w4u92',
  'com.github.githubapp'
);
```

In order: Codex, VS Code, Claude Desktop, Cursor, and the GitHub Copilot app. Check every row, since a host can have more than one copy of an app.

Codex and VS Code need a presence check, because no fixed build exists yet. Decide host by host whether each install is approved, and remove the ones that aren't.

Cursor and the GitHub Copilot app need a version check: 1.1.29 or later for Copilot. The sources give no version for Cursor, only the October 1 patch date, so set that bar from Cursor's changelog. All five tools are [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps) on macOS, so you can push current builds from Fleet.

For Codex and VS Code, Weizman also suggested reading each machine's trusted-project list from its config file and scanning the HTML files in those projects. [Fleet scripts](https://fleetdm.com/guides/scripts) can run that kind of check. For help building it, contact [Fleet support](https://fleetdm.com/support) or your customer success team.

## Keep the check running

A one-time sweep goes stale the next time someone installs an assistant. Set up a Fleet policy that fails when the GitHub Copilot app is older than 1.1.29. Policies run hourly on every online host. When the policy fails, a policy automation can [install the current build](https://fleetdm.com/guides/automatic-software-install-in-fleet). When Codex or VS Code ships a fix, the same pattern covers them.

## Where this leaves you

Embedded browsers turned coding assistants into a new way to reach the host, and two vendors have left that door open. The immediate step is the same whatever the vendor decides: know which hosts run these tools, and at which version.

## See it live

- **Get a demo:** [Talk to Fleet](https://fleetdm.com/contact) about finding every AI coding assistant on your hosts and the version it runs.
- **Join a GitOps workshop:** [Learn how to write policies like the Copilot version check](https://fleetdm.com/workshops), keep them in Git, and review every change before it ships.

## Sources

- Forever Security, [EMBRACE: How We Hacked the Browsers Embedded in Top Five Desktop AI Assistants](https://forever.security/blog/embrace-hacking-5-ai-assistants-embedded-browsers).
- Forever Security, [EMBRACE technical companion](https://forever.security/blog/embrace-hacking-5-ai-assistants-embedded-browsers-technical).
- Help Net Security, [Researchers broke the browsers built into VS Code, Cursor, Copilot, Codex and Claude Desktop](https://www.helpnetsecurity.com/2026/10/29/research-ai-assistant-browser-vulnerabilities/).

<meta name="articleTitle" value="Codex and VS Code still let a local HTML file read your whole disk">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-30">
<meta name="description" value="Forever Security's EMBRACE research found Codex and VS Code still unpatched. Find which hosts run AI coding assistants and which versions they're on.">
