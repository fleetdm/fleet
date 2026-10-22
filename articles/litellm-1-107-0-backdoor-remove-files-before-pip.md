# Don't run pip on a LiteLLM 1.107.0 host until the backdoor files are gone

*Attackers published a malicious LiteLLM release on October 21 that reruns itself every time Python starts, including when you run pip to fix it. Finding every host that installed it comes first.*

## Key takeaways

- **The cleanup order matters.** [LiteLLM's incident notes](https://docs.litellm.ai/blog/security-incident-oct-21-2026) say the malware persists through `cook.pth` and `sitecustomize.py` in site-packages, so any Python launch in an affected environment reruns it. Running `pip install` first re-triggers the payload, so delete the files before you reinstall anything.
- **The exposure window was short, and the footprint is wider than PyPI.** PyPI served `litellm==1.107.0` for about 1 hour and 44 minutes. The `v1.107.0`, `latest`, `main-latest`, and `main-stable` Docker tags were affected for about 2.5 hours, so a pinned wheel is not the only way a host got it.
- **Treat any host that ran it as compromised.** The payload steals credentials and loads remote-code execution tools. If the host had outbound internet access, assume exfiltration, and rotate every secret that any process on the host could reach.
- **A clean source tag proves nothing about the artifact.** The official `v1.107.0` tag on GitHub doesn't contain the malicious files. The poisoned wheels and images came out of the release pipeline, so you have to check what is installed, not what the repository says.
- **Pinning to `1.106.2` is the fix, and a host report is how you prove it.** LiteLLM's guidance is to pin to `litellm==1.106.2` or any version published before the incident. A list of hosts and versions turns that into evidence.
- **Developer machines are part of the story.** The attackers got in with developer GitHub CLI credentials, so the machines where engineers run AI coding tools, and the logs those tools write, deserve the same scrutiny as production servers.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See software inventory in Fleet</a>

LiteLLM is an open source gateway that many teams run to route requests to LLM providers. On October 21, seven malicious `litellm==1.107.0` wheels were uploaded to PyPI at about 11:39 UTC, and malicious Docker images followed a few minutes later. [The Hacker News](https://thehackernews.com/2026/10/litellm-compromised-in-supply-chain.html) reports it is the second LiteLLM compromise this year, after one in March.

The patch is simple: stay off 1.107.0. The hard part is finding every laptop, build runner, and container that pulled it during a window of a couple of hours, and cleaning each one in the right order.

## What happened on October 21

According to LiteLLM's timeline, a privileged developer credential overrode branch protection on a private release workflow, and a staging run was pointed at a lookalike repository named `Berri-AI/litellm` instead of `BerriAI/litellm`. A second developer credential satisfied the production approval gate, and a third deleted workflow records afterward.

PyPI removed the release at 13:23 UTC. The Docker Hub tags were repointed to clean artifacts by 14:20 UTC, and the GHCR tags were found unavailable at 13:51 UTC. LiteLLM says customers on its private enterprise registry were not affected.

### How the credentials leaked

LiteLLM's preliminary root cause, [as reported by The Hacker News](https://thehackernews.com/2026/10/litellm-suspects-github-token-leak-in.html), is that GitHub CLI OAuth tokens were embedded in response bodies by Claude Code and then logged by an internal LiteLLM deployment that was exposed to the internet. The maintainers suspect the attacker harvested the tokens from those logs. They have not confirmed how the attacker got into that deployment, and the root cause analysis is still open, so treat this as preliminary.

## Why the order of cleanup matters

The malware installs a `cook.pth` file and a `sitecustomize.py` file in site-packages. Python processes `.pth` files and `sitecustomize` at startup, so every Python launch in that environment reruns the loader, and that includes `pip`. A detached Python daemon also holds `/tmp/remote_code_loader.lock`.

LiteLLM's steps, in order:

1. Don't run `pip install` first.
2. Delete `cook.pth`, `cook.py`, `sitecustomize.py`, and the matching `__pycache__` files from every affected site-packages directory.
3. Kill the daemon holding `/tmp/remote_code_loader.lock`.
4. Reinstall pinned to `litellm==1.106.2`, and rebuild images and virtualenvs from scratch where you can.
5. Rotate every credential the host could reach: instance or task IAM roles, provider keys stored in the LiteLLM proxy database, master and virtual keys, the `DATABASE_URL` password, and any secret in the environment of any process on the host.

LiteLLM also suggests checking CloudTrail for `GetCallerIdentity` calls from the instance role around install time. Finding these files confirms exposure. Not finding them does not prove a host is clean.

## Finding the hosts with Fleet

**Start with software inventory.** Fleet's agent collects installed Python packages alongside other software, so a host with `litellm` 1.107.0 can show up in the **Software** tab. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists what Fleet collects on each platform. Python packages inside a virtualenv or a project directory may not be collected, so don't treat an empty result as an all-clear.

**Use a report for the files.** A [report](https://fleetdm.com/guides/queries) can check for the version and for the persistence files directly. The first query below lists hosts with the bad version. The second looks for the files LiteLLM names, in the common system site-packages locations:

```sql
SELECT name, version, path FROM python_packages WHERE name = 'litellm' AND version = '1.107.0';
```

```sql
SELECT path FROM file WHERE path LIKE '/usr/lib/python3%/%-packages/cook.pth' OR path LIKE '/usr/local/lib/python3%/%-packages/cook.pth';
```

These are sketches, and they haven't been run against a host that has the malware. Virtualenvs and containers live in other paths, so add the paths your teams use. A report that returns zero rows only reassures you if you have confirmed it can find a known file.

For containers, check the image digest instead. LiteLLM lists `sha256:1bbcb1443bc962e45f5aad53fe3d2c30600e0379715fb6c664cdd3ac126c339e` for the affected `litellm` image, and Fleet's software inventory may not see inside a running container.

## Developer machines and build runners

A short exposure window is easy to miss on a fleet, because most hosts never pulled the release. The ones that did tend to be the ones where unpinned installs run automatically: CI runners, ephemeral dev containers, and laptops where someone ran `pip install -U litellm` that morning. Check those first.

The initial access also points at developer tooling. If your engineers use AI coding assistants, ask what those tools write into logs, and where those logs go. A token that appears in a log is a token somebody can collect.

## Where this leaves you

Remove the files, stop the daemon, reinstall at 1.106.2, and rotate secrets, in that order. Then confirm from the hosts, not from the advisory, that nothing is still running 1.107.0. A package version is something a host can report back to you.

## See it live

- **Get a demo** at [fleetdm.com/contact](https://fleetdm.com/contact) to see software inventory across Linux, macOS, and Windows hosts.
- **Run a report on your own fleet.** Learn how in [Reports](https://fleetdm.com/guides/queries).

## Further reading

- [GitLab's CVSS 10.0 regex bug needs no login, and the fix is a version number you have to verify](https://fleetdm.com/industry-news/gitlab-cvss-10-regex-flaw-needs-no-login-verify-the-version)

## Sources

- LiteLLM, [Contained: Oct 21, 2026 LiteLLM release v1.107.0 compromised](https://docs.litellm.ai/blog/security-incident-oct-21-2026).
- The Hacker News, [LiteLLM compromised in supply chain attack delivering credential stealer and backdoor](https://thehackernews.com/2026/10/litellm-compromised-in-supply-chain.html).
- The Hacker News, [LiteLLM suspects GitHub token leak in AI gateway logs behind malicious package release](https://thehackernews.com/2026/10/litellm-suspects-github-token-leak-in.html).

<meta name="articleTitle" value="Don't run pip on a LiteLLM 1.107.0 host until the backdoor files are gone">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-22">
<meta name="description" value="LiteLLM 1.107.0 reruns its backdoor on every Python start, pip included. Here's the cleanup order and how to find every host that installed it.">
