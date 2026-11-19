# A 28-byte .pth file turned Microsoft's agent-framework into a credential stealer

*Two poisoned releases of a Microsoft Python package ran on every interpreter start, whether or not anyone imported it. The first job is finding out which of your hosts installed them.*

## Key takeaways

- **The bad versions are 1.26.1 and 1.26.2.** Both appeared on PyPI on November 17, 2026, and both were quarantined within the hour. Version 1.26.0 is the last clean release.
- **Installing the package was enough.** A startup hook ran the stealer whenever Python launched, including during `pip` and `pytest` runs, so "we never imported it" doesn't clear a host.
- **The stealer went after the credentials developers and build machines hold.** It read environment variables, cloud CLI and registry config files, other processes' environments, and asked Azure for fresh managed-identity tokens.
- **Version inventory turns a guess into a list.** Fleet already collects installed Python packages and their versions on macOS, Windows, and Linux, so you can search for the two bad versions across every enrolled host.
- **Inventory has blind spots, and you should name them.** Project-level virtual environments, ephemeral CI runners, and container images need their own checks.
- **Rotating keys is only half the cleanup.** Tokens already issued from a managed identity or OIDC federation stay valid until they expire, so review sign-in logs and role assignments too.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See how Fleet tracks software versions</a>

On November 17, someone with publishing access to Microsoft's official `agent-framework` project on PyPI uploaded two releases that carried a credential stealer. The package averages roughly 190,000 monthly downloads, according to figures OX Security shared with [Help Net Security](https://www.helpnetsecurity.com/2026/11/19/microsoft-agent-framework-package-pypi-compromised/). Automated dependency bots opened upgrade pull requests within six minutes of the first release, [JFrog reports](https://research.jfrog.com/post/agent-framework-hijack/).

The releases were live for well under an hour. That's short for a human to notice and long enough for CI pipelines, Renovate and Dependabot, and developers running `pip install -U` to pull them in. If your organization builds anything with Microsoft's agent tooling, the useful question isn't whether the package was popular. It's which of your machines took one of those two builds.

## What the two releases did

According to JFrog's timeline, version 1.26.1 landed at 20:35 UTC and PyPI quarantined it at 21:06. Version 1.26.2 followed at 21:08, which shows the attacker still had publishing access after the first takedown. Both were gone by 21:16. JFrog found that the wheels were repacked by hand rather than built by Microsoft's pipeline, and [The Hacker News](https://thehackernews.com/2026/11/attackers-compromise-microsofts-agent.html) reports no matching tag or malicious code in Microsoft's GitHub repository. As of BleepingComputer's [report](https://www.bleepingcomputer.com/news/security/microsofts-agent-framework-pypi-package-compromised-to-steal-credentials/) on November 18, Microsoft hadn't explained how the attacker got publishing rights.

Both releases kept their dependency on the genuine `agent-framework-core` 1.26.0, so the framework kept working. Nothing looked broken.

## Why "we never imported it" doesn't help

Python reads every `.pth` file in a site-packages directory when the interpreter starts, and it runs any line that begins with `import`. Version 1.26.1 shipped a 28-byte file named `agent_framework_meta.pth` whose only content was an import of the stealer module. Version 1.26.2 restored Microsoft's original package code and moved the hook into `af_runtime_diag.pth` and `af_runtime_diag.py`, which made a quick diff of the package look clean.

That means the stealer ran on any Python launch in an affected environment: a test run, a language server in an editor, or `pip` itself. On macOS and Linux it forked into the background so the original command finished normally. It wrote no marker file, so it ran again on every start.

## What it collected

JFrog, StepSecurity, Socket, and OX Security published overlapping analyses. Taken together, the stealer gathered:

- Environment variables matching 31 name patterns, including Azure, AWS, GitHub, OpenAI, and Hugging Face, plus broad matches on `KEY`, `TOKEN`, and `SECRET`.
- Up to 60 small files (80 in 1.26.2), including `~/.azure/*.json`, `pip.conf`, `.npmrc`, `.netrc`, `.git-credentials`, `.docker/config.json`, and `.env` files in common app paths.
- On Linux, the environments of up to 300 other running processes.
- Fresh OAuth tokens from Azure managed identity for 12 audiences, including Azure Resource Manager, Key Vault, and Microsoft Graph. Version 1.26.2 added GitHub Actions OIDC token requests.

Everything went to a single `webhook.site` URL in one compressed, unencrypted POST. JFrog notes that most of the file paths were Linux-style, so the file sweep missed most macOS home-directory locations. Environment variables and token requests didn't depend on those paths, so a Mac that installed the package still belongs on your list.

## Finding affected hosts with Fleet

Fleet's agent already reports installed Python packages, with versions and install paths, on macOS, Windows, and Linux. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists the locations it reads: system and per-user site-packages, plus pipx, uv tools, conda, pipenv, pyenv, and mise. In the Fleet UI, search **Software** for `agent-framework` to see every version found and the hosts running each one.

To answer the question directly, run a live report across your hosts:

```sql
SELECT name, version, path
FROM users CROSS JOIN python_packages USING (uid)
WHERE name IN ('agent-framework', 'agent_framework')
  AND version IN ('1.26.1', '1.26.2');
```

The `users` join tells Fleet's agent to search each user's directories as well as system locations. It needs a version of Fleet's agent built on osquery 5.16.0 or later, and [this guide](https://fleetdm.com/guides/osquery-consider-joining-against-the-users-table) explains why. Test the query on a host where you've installed 1.26.0 to confirm how the package name appears before you rely on it.

A removed package can leave its hook behind if someone deleted files by hand. To catch leftover hook files, check for them by name. Adjust the paths to match where Python lives on your hosts:

```sql
SELECT path FROM file
WHERE path LIKE '/usr/lib/python3.%/site-packages/af_runtime_diag.pth'
   OR path LIKE '/usr/local/lib/python3.%/site-packages/af_runtime_diag.pth'
   OR path LIKE '/home/%/.local/lib/python3.%/site-packages/af_runtime_diag.pth'
   OR path LIKE '/Users/%/Library/Python/3.%/lib/python/site-packages/af_runtime_diag.pth';
```

Repeat it for `agent_framework_meta.pth`. To keep watching after this week, turn the version check into a [policy](https://fleetdm.com/securing/what-are-fleet-policies) that passes only when neither bad version is present:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM users CROSS JOIN python_packages USING (uid)
  WHERE name IN ('agent-framework', 'agent_framework')
    AND version IN ('1.26.1', '1.26.2')
);
```

Policy automations can then open a ticket or send a webhook when a host fails.

## What inventory won't see

Be honest with yourself about coverage. Fleet doesn't collect packages from project-level `.venv` directories, which is where many developers install dependencies. It sees only hosts with Fleet's agent, so ephemeral CI runners and container images built on November 17 need a separate audit of lockfiles, build logs, and image layers. JFrog points out that any image with either version in site-packages runs the payload on every container start.

For those gaps, Fleet can [run a script](https://fleetdm.com/guides/scripts) on enrolled hosts to search project directories for the hook files. Set `_AF_TEL=1` in the script's environment before it calls Python or `pip`. JFrog found that the stealer checks for that variable and skips its work when it's set, so cleanup doesn't trigger another upload.

## Cleanup goes past the package

Pinning `agent-framework==1.26.0` and removing the bad files stops new collection. It doesn't undo what was sent. The researchers' advice lines up:

- Treat every secret reachable from an affected host or runner as compromised and rotate it.
- Revoke Entra ID sessions for users whose `~/.azure` files were on an affected host, since a password reset doesn't invalidate cached refresh tokens.
- Review managed-identity role assignments, federated credentials, and sign-in logs. JFrog notes that managed-identity tokens can't be revoked and stay valid for up to 24 hours.
- Search DNS and proxy logs for `webhook.site`. An HTTP 200 response on the telemetry path means the upload completed.

## Know where your packages live before the next one

This was the second Microsoft-related PyPI package compromised this year, after `durabletask` in May, according to The Hacker News. Language packages install outside the operating system's package manager, under user directories and project folders, which is exactly where traditional patch tools stop looking.

The list of hosts that installed a bad build should come from those hosts, within minutes, not from a reconstruction of who might have run `pip install` on a Tuesday afternoon.

## See it live

- **Get a demo** to search Python and npm packages across your own hosts: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read which package types Fleet collects, and where it looks:** [fleetdm.com/guides/software-inventory-reference](https://fleetdm.com/guides/software-inventory-reference)

## Sources

- JFrog Security Research, [Microsoft's agent-framework hijacked on PyPI via silent .pth startup hook](https://research.jfrog.com/post/agent-framework-hijack/).
- BleepingComputer, [Microsoft's Agent Framework PyPI package compromised to steal credentials](https://www.bleepingcomputer.com/news/security/microsofts-agent-framework-pypi-package-compromised-to-steal-credentials/).
- Help Net Security, [Credential stealer slipped into Microsoft's official Python agent framework](https://www.helpnetsecurity.com/2026/11/19/microsoft-agent-framework-package-pypi-compromised/).
- The Hacker News, [Attackers compromise Microsoft's Agent Framework PyPI package to steal cloud credentials](https://thehackernews.com/2026/11/attackers-compromise-microsofts-agent.html).
- SecurityWeek, [Microsoft Agent Framework releases poisoned with credential stealer](https://www.securityweek.com/microsoft-agent-framework-releases-poisoned-with-credential-stealer/).

<meta name="articleTitle" value="A 28-byte .pth file turned Microsoft's agent-framework into a credential stealer">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-19">
<meta name="description" value="Poisoned agent-framework 1.26.1 and 1.26.2 stole credentials on every Python start. Here's how to find which of your hosts installed them.">
