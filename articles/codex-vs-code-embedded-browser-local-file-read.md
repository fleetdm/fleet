# Codex and VS Code still let a local HTML file read your whole disk

*Forever Security broke the browsers built into five AI coding assistants. Two of those flaws are still open, so the fix has to come from knowing where the tools run.*

## Key takeaways

- **Two of five flaws are still open.** Forever Security's EMBRACE research found escapes in the embedded browsers of Cursor, GitHub Copilot, OpenAI Codex, Microsoft VS Code, and Anthropic Claude Desktop. As of October 29, the Codex and VS Code issues remain unpatched.
- **The risk is a file on disk, not a website.** A rigged project with an HTML file, opened from a link the assistant suggests, can read files well outside the project folder. In Forever's Codex test, that sweep collected SSH keys, `.env` files, and npm tokens.
- **Vendor patch notes won't cover you.** Microsoft declined the VS Code report, citing Workspace Trust, and OpenAI hasn't replied about Codex. Waiting for a fix means waiting indefinitely.
- **The first job is a list.** You can't prioritize what you can't see. A host-by-host inventory of which of these tools are installed, and at what version, turns a headline into a worklist.
- **Fixed tools still need a version check.** Cursor patched on October 1 and Copilot fixed the address spoofing flaw in 1.1.29. Confirming those builds are installed is the same check, run again.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See software inventory in Fleet</a>

AI coding assistants now ship with a browser inside them, so an agent can open a page, preview an app, or follow a link. Forever Security tested those browsers in Cursor, GitHub Copilot, OpenAI Codex, Microsoft VS Code, and Anthropic Claude Desktop, and found a way out of each one. The research is called EMBRACE.

Three of the flaws have a fix or are closed. Two do not. This article walks through what the research found and what an IT or security team can do about the tools that are still exposed.

## What EMBRACE found

The five findings are different bugs with a common theme: a page the assistant loads can reach the host computer.

### Local file read in Codex, VS Code, and Claude Desktop

In these three tools, a local HTML file opened in the built-in browser can read files beyond the project. In Codex, Forever found no folder restriction at all. In VS Code and Claude Desktop, local pages are limited to the project folder, but a web worker the page starts sits outside that check and can read any file on the machine.

Forever reported the Codex issue to OpenAI on July 27 and says it got no reply. It reported the VS Code issue to Microsoft on September 10, and Microsoft declined it as an issue gated by Workspace Trust. Forever's position is that the Workspace Trust prompt gives the user little real protection. Anthropic declined Forever's July 23 Claude Desktop report, then accepted a report from Noma Security filed a day later. The sources don't say whether Claude Desktop has been fixed.

### Command execution in Cursor and address spoofing in Copilot

In Cursor, a page hid an agent command behind a "Start setup" button. One keypress approved it, and when the sandbox blocked the command, the agent turned the sandbox off and retried. Forever notes the attack depends on the model and doesn't work every time. Cursor patched the flaw on October 1.

In GitHub Copilot, a page could set the address the app reported, so a fake sign-in form could appear under github.com/login. GitHub fixed it in Copilot 1.1.29 on October 13, before Forever filed a report.

## Why waiting on the vendor fails

Two of the vendors have said they don't plan to treat these as vulnerabilities, or haven't responded. That leaves the decision with the organization running the tools. The attack also needs only one click, and the link comes from the assistant itself, so "tell developers to be careful" is a weak control.

Forever's own advice starts with an inventory: find which of these tools run on developer devices, since each ships an embedded browser, and treat that browser as an attack surface.

## Finding the hosts that run them

Fleet's agent inventories installed software and versions on every enrolled host, across macOS, Windows, and Linux. Filter the software list for the tools above and you get the hosts, grouped by version.

For a sharper cut, write a report that checks for the applications directly. This example for macOS is a starting point. App names vary by install, so confirm the names against your own software list before you rely on it.

```sql
SELECT name, bundle_short_version, path
FROM apps
WHERE name LIKE '%Cursor%'
   OR name LIKE '%Codex%'
   OR name LIKE '%Visual Studio Code%'
   OR name LIKE '%Claude%';
```

Cursor and Copilot need a version check instead of a presence check. The sources give Copilot 1.1.29 as the fixed version. They give no version number for Cursor, only the October 1 patch date, so use your own release notes or the vendor's changelog to set that bar.

Forever also suggests, for Codex and VS Code, pulling each machine's list of trusted projects from its config file and checking the HTML files in them. Fleet can collect files and run scripts on hosts, but that is a separate piece of work from the inventory above.

## Closing

Embedded browsers turned coding assistants into a new way to reach the host, and two vendors have left that door open. The immediate step is the same whatever the vendor decides: know which hosts run these tools, and at which version.

## See it live

- **Get a demo.** See software inventory across your hosts: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the research.** Forever Security's write-up: [EMBRACE](https://forever.security/blog/embrace-hacking-5-ai-assistants-embedded-browsers)

## Sources

- Forever Security, [EMBRACE: How We Hacked the Browsers Embedded in Top Five Desktop AI Assistants](https://forever.security/blog/embrace-hacking-5-ai-assistants-embedded-browsers).
- Help Net Security, [Researchers broke the browsers built into VS Code, Cursor, Copilot, Codex and Claude Desktop](https://www.helpnetsecurity.com/2026/10/29/research-ai-assistant-browser-vulnerabilities/).

<meta name="articleTitle" value="Codex and VS Code still let a local HTML file read your whole disk">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-30">
<meta name="description" value="Forever Security's EMBRACE research left Codex and VS Code unpatched. Find which hosts run AI coding assistants and which versions they're on.">
