# Microsoft's agent-framework on PyPI stole credentials on every Python start: find hosts with 1.26.1 or 1.26.2

*Two poisoned releases of a Microsoft Python package ran on every interpreter start, whether or not anyone imported it. The first job is finding out which of your hosts installed them.*

## Key takeaways

- **The bad versions are 1.26.1 and 1.26.2.** Both reached PyPI on November 17, 2026, PyPI pulled both within the hour, and 1.26.0 remains the newest safe version.
- **Installing the package was enough.** A startup hook ran the stealer whenever Python launched, including during `pip` and `pytest` runs, so "we never imported it" doesn't clear a host.
- **The stealer went after the credentials developers and build machines hold.** It read environment variables, cloud CLI and registry config files, other processes' environments, and asked Azure for fresh managed-identity tokens.
- **Version inventory turns a guess into a list.** Fleet already collects installed Python packages and their versions on macOS, Windows, and Linux, so you can search for the two bad versions across every enrolled host.
- **Inventory has blind spots, and you should name them.** Project-level virtual environments, ephemeral CI runners, and container images need their own checks.
- **Rotating keys is only half the cleanup.** Tokens already issued from a managed identity or OIDC federation stay valid until they expire, so review sign-in logs and role assignments too.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See how Fleet tracks software versions</a>

On November 17, someone with publishing access to Microsoft's official `agent-framework` project on PyPI uploaded two releases that carried a credential stealer. The package averages roughly 190,000 monthly downloads, according to figures OX Security shared with [Help Net Security](https://www.helpnetsecurity.com/2026/11/19/microsoft-agent-framework-package-pypi-compromised/). Automated dependency bots opened upgrade pull requests within six minutes of the first release, [JFrog reports](https://research.jfrog.com/post/agent-framework-hijack/).

The releases were live for well under an hour. That's short for a human to notice and long enough for CI pipelines, Renovate and Dependabot, and developers running `pip install -U` to pull them in. If your organization builds anything with Microsoft's agent tooling, the useful question isn't whether the package was popular. It's which of your machines took one of those two builds.

## What the two releases did

According to JFrog's timeline, the first malicious build, 1.26.1, sat on PyPI from 20:35 UTC until PyPI quarantined it at 21:06. Two minutes later, the attacker published 1.26.2, which proved the publishing access had survived the first takedown, and both versions were off PyPI within the hour. JFrog found the wheels were repacked by hand instead of built by Microsoft's release pipeline. [Wiz](https://www.wiz.io/blog/malicious-version-of-microsofts-agent-framework-published-to-pypi) then traced the uploads to what appears to be ESRP signing material in the attacker's hands (a stolen certificate along with its key), ESRP being the internal Microsoft service that signs and publishes the package. A Microsoft maintainer confirmed the releases were pulled and an investigation was under way, [SecurityWeek reports](https://www.securityweek.com/microsoft-agent-framework-releases-poisoned-with-credential-stealer/).

Neither release touched the pin on the real `agent-framework-core` 1.26.0, so applications built on the framework ran exactly as before, and no failed build or crash gave the attack away.

## Why "we never imported it" doesn't help

Each time the interpreter starts, Python scans its site-packages folders for `.pth` files and executes every line in them that starts with `import`. Version 1.26.1 shipped a 28-byte file named `agent_framework_meta.pth` whose only content was an import of the stealer module. In 1.26.2, the attacker put Microsoft's original package code back and relocated the hook to `af_runtime_diag.pth` and `af_runtime_diag.py`, so a quick diff of the package came up clean.

As a result, starting Python for any reason on an infected machine set the stealer off, whether that was a test run, an editor's language server, or `pip` itself. On macOS and Linux it forked into the background so the original command finished normally. It wrote no marker file, so it ran again on every start.

## What it collected

JFrog, StepSecurity, Socket, and OX Security published overlapping analyses. Taken together, the stealer gathered:

- Environment variables matching 31 name patterns, including Azure, AWS, GitHub, OpenAI, and Hugging Face, plus broad matches on `KEY`, `TOKEN`, and `SECRET`.
- Up to 60 small files (80 in 1.26.2), including `~/.azure/*.json`, `pip.conf`, `.npmrc`, `.netrc`, `.git-credentials`, `.docker/config.json`, and `.env` files in common app paths.
- On Linux, the environment variables of as many as 300 other processes running on the same machine.
- Newly minted Azure managed-identity tokens for 12 different services, Key Vault and Microsoft Graph among them. The 1.26.2 build also asked GitHub Actions for OIDC tokens.

Everything went to a single `webhook.site` URL in one compressed, unencrypted POST. JFrog notes that most of the file paths were Linux-style, so the file sweep missed most macOS home-directory locations. Environment variables and token requests didn't depend on those paths, so a Mac that installed the package still belongs on your list.

## Finding affected hosts with Fleet

Fleet's agent, built on osquery, already reports installed Python packages with their versions and install paths on macOS, Windows, and Linux. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists where it looks: system and per-user site-packages, plus pipx, uv tools, conda, pipenv, pyenv, and mise. Searching **Software** for `agent-framework` in Fleet shows every version found across your hosts and which machines run each one, which turns 1.26.1 and 1.26.2 into a list of names instead of a guess.

To keep watching after this week, a [Fleet policy](https://fleetdm.com/securing/what-are-fleet-policies) can flag any host that still carries either bad version, and [automations](https://fleetdm.com/guides/automations) can open a ticket or send a webhook when a host fails. The same approach can look for the four leftover hook files by name, which matters when someone deleted the package by hand and left the `.pth` file behind. For help writing those checks for your own environment, reach out to [Fleet support](https://fleetdm.com/support) or your customer success team, or [talk to Fleet](https://fleetdm.com/contact) if you aren't a customer yet.

## What inventory won't see

Be honest with yourself about coverage. Fleet doesn't collect packages from project-level `.venv` directories, which is where many developers install dependencies. It sees only hosts with Fleet's agent, so ephemeral CI runners and container images built on November 17 need a separate audit of lockfiles, build logs, and image layers. JFrog points out that any image with either version in site-packages runs the payload on every container start.

For those gaps, Fleet can [run a script](https://fleetdm.com/guides/scripts) on enrolled hosts to search project directories for the hook files. Set `_AF_TEL=1` in the script's environment before it calls Python or `pip`. JFrog found that the stealer checks for that variable and skips its work when it's set, so cleanup doesn't trigger another upload.

## Cleanup goes past the package

Pinning `agent-framework==1.26.0` and removing the bad files stops new collection. It doesn't undo what was sent. The researchers' advice lines up:

- Assume any secret an infected host or build runner could read is now in the attacker's hands, and replace it.
- Revoke Entra ID sessions for users whose `~/.azure` files were on an affected host, since a password reset doesn't invalidate cached refresh tokens.
- Review managed-identity role assignments, federated credentials, and sign-in logs. JFrog notes that managed-identity tokens can't be revoked, and they keep working for as long as a day after they're issued.
- Search DNS and proxy logs for `webhook.site`. An HTTP 200 response on the telemetry path means the upload completed.

## Know where your packages live before the next one

According to The Hacker News, a Microsoft-linked package on PyPI has now been hijacked twice this year, since `durabletask` was hit in May. Language packages install outside the operating system's package manager, under user directories and project folders, which is exactly where traditional patch tools stop looking.

The list of hosts that installed a bad build should come from those hosts, within minutes, not from a reconstruction of who might have run `pip install` on a Tuesday afternoon.

## See it live

- **Get a demo** to search Python and npm packages across your own hosts: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read which package types Fleet collects, and where it looks:** [fleetdm.com/guides/software-inventory-reference](https://fleetdm.com/guides/software-inventory-reference)

## Sources

- JFrog Security Research, [Microsoft's agent-framework hijacked on PyPI via silent .pth startup hook](https://research.jfrog.com/post/agent-framework-hijack/).
- Wiz Research, [Microsoft agent-framework compromised: Credential stealer targets Azure and CI pipelines](https://www.wiz.io/blog/malicious-version-of-microsofts-agent-framework-published-to-pypi).
- BleepingComputer, [Microsoft's Agent Framework PyPI package compromised to steal credentials](https://www.bleepingcomputer.com/news/security/microsofts-agent-framework-pypi-package-compromised-to-steal-credentials/).
- Help Net Security, [Credential stealer slipped into Microsoft's official Python agent framework](https://www.helpnetsecurity.com/2026/11/19/microsoft-agent-framework-package-pypi-compromised/).
- The Hacker News, [Attackers compromise Microsoft's Agent Framework PyPI package to steal cloud credentials](https://thehackernews.com/2026/11/attackers-compromise-microsofts-agent.html).
- SecurityWeek, [Microsoft Agent Framework releases poisoned with credential stealer](https://www.securityweek.com/microsoft-agent-framework-releases-poisoned-with-credential-stealer/).

<meta name="articleTitle" value="Microsoft's agent-framework on PyPI stole credentials on every Python start: find hosts with 1.26.1 or 1.26.2">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-19">
<meta name="description" value="Poisoned agent-framework 1.26.1 and 1.26.2 stole credentials on every Python start. Here's how to find which of your hosts installed them.">
