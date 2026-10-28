# LiteLLM 1.107.0 backdoor: Why running pip reruns the malware, and how to find affected hosts

*Attackers published a malicious LiteLLM release on Oct. 21 that reruns itself every time Python starts, including when you run pip to fix it. Finding every host that installed it comes first.*

## Key takeaways

- **The cleanup order matters.** [LiteLLM's incident notes](https://docs.litellm.ai/blog/security-incident-oct-21-2026) say the malware persists through `cook.pth` and `sitecustomize.py` in site-packages, so any Python launch in an affected environment reruns it. Running `pip install` first re-triggers the payload, so delete the files before you reinstall anything.
- **The exposure window was short, and the footprint is wider than PyPI.** PyPI served `litellm==1.107.0` for less than two hours. The `v1.107.0`, `latest`, `main-latest`, and `main-stable` Docker tags were affected for about 2.5 hours, so a pip install isn't the only way a host got it.
- **Treat any host that ran it as compromised.** The payload grabs credentials and opens a backdoor for running remote code. If the host had outbound internet access, assume exfiltration, and rotate every secret that any process on the host could reach.
- **A clean source tag proves nothing about the artifact.** The official `v1.107.0` tag on GitHub doesn't contain the malicious files. The poisoned wheels and images came out of the release pipeline, so you have to check what is installed, not what the repository says.
- **Pinning to `1.106.2` is the fix, and a host report is how you prove it.** LiteLLM's guidance is to pin to `litellm==1.106.2` or any version published before the incident. A list of hosts and versions turns that into evidence.
- **A cooldown would have stopped it.** A policy that holds back packages for a few days after they're published would have kept a release that lived less than two hours off every host.
- **Developer machines are part of the story.** The attackers got in with developer GitHub CLI credentials, so the machines where engineers run AI coding tools, and the logs those tools write, deserve the same scrutiny as production servers.

<a purpose="cta-button" href="https://fleetdm.com/software-management">Find every host that pulled LiteLLM 1.107.0</a>

LiteLLM is an open-source gateway that many teams run to route requests to LLM providers. On Oct. 21, attackers pushed seven malicious `litellm==1.107.0` wheels to PyPI at about 11:39 a.m. UTC, and malicious Docker images followed a few minutes later. [The Hacker News](https://thehackernews.com/2026/10/litellm-compromised-in-supply-chain.html) reports it is the second LiteLLM compromise this year, after one in March.

The patch is simple: stay off 1.107.0. The hard part is finding every laptop, build runner, and container that pulled it during a window of a couple of hours, and cleaning each one in the right order.

## What happened on October 21

According to LiteLLM's timeline, a privileged developer credential overrode branch protection on a private release workflow, and a staging run was pointed at a lookalike repository named `Berri-AI/litellm` instead of `BerriAI/litellm`. A second developer credential satisfied the production approval gate, and a third account wiped the workflow records afterward.

PyPI had pulled the release by 1:23 p.m. UTC. GitHub's container registry tags were unavailable by 1:51 p.m. UTC, and by 2:20 p.m. UTC every Docker Hub tag had been removed or pointed back to a clean image. LiteLLM says customers on its private enterprise registry were not affected.

### How the credentials leaked

LiteLLM's preliminary root cause, [according to its incident post](https://docs.litellm.ai/blog/security-incident-oct-21-2026), is that in some situations Claude Code can include a GitHub CLI OAuth token in its response, and an internal LiteLLM deployment that was open to the internet logged those responses. The maintainers suspect the attacker harvested the tokens from those logs. They haven't confirmed how the attacker got into that deployment, though they suspect a recently patched vulnerability, so treat this as preliminary.

## Why the order of cleanup matters

The malware installs a `cook.pth` file and a `sitecustomize.py` file in site-packages. Python processes `.pth` files and `sitecustomize` at startup, so every Python launch in that environment reruns the loader, and that includes `pip`. A detached Python daemon also holds `/tmp/remote_code_loader.lock`.

LiteLLM's steps, in order:

1. Don't run `pip install` first.
2. Delete `cook.pth`, `cook.py`, `sitecustomize.py`, and the matching `__pycache__` files from every affected site-packages directory.
3. Stop the background Python process that holds `/tmp/remote_code_loader.lock`.
4. Reinstall pinned to `litellm==1.106.2`, and rebuild images and virtualenvs from scratch where you can.
5. Rotate any credential the host had access to. That includes instance or task IAM roles, LLM provider keys saved in the proxy's database, the proxy's master and virtual keys, the password in `DATABASE_URL`, and secrets in any running process's environment variables.

LiteLLM also suggests looking in CloudTrail for `GetCallerIdentity` calls made by the instance role close to when the package was installed. Finding these files confirms exposure. Not finding them doesn't mean a host is clean.

## Finding the hosts with Fleet

**Start with software inventory.** Fleet's agent, built on osquery, collects installed Python packages alongside other software, so a host with `litellm` 1.107.0 can show up on the **Software** page. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists what Fleet collects on each platform. Project-level `.venv` directories aren't collected, so don't treat an empty result as an all-clear.

**Use a report for the files.** A [report](https://fleetdm.com/guides/reports) can look for the version and for the files LiteLLM names, `cook.pth` and `sitecustomize.py`, in the paths your teams use. If you'd like help scoping one, reach out to [Fleet support](https://fleetdm.com/support) or your customer success team.

For containers, check the image digest instead. LiteLLM lists `sha256:1bbcb1443bc962e45f5aad53fe3d2c30600e0379715fb6c664cdd3ac126c339e` for the affected `litellm` image, and Fleet's software inventory doesn't look inside running containers.

## Developer machines and build runners

A short exposure window is easy to miss on a fleet, because most hosts never pulled the release. The ones that did tend to be the ones where unpinned installs run automatically: CI runners, ephemeral dev containers, and laptops where someone ran `pip install -U litellm` that morning. Check those first.

The initial access also points at developer tooling. If your engineers use AI coding assistants, ask what those tools write into logs, and where those logs go. A token that appears in a log is a token somebody can collect.

## Add a cooldown before the next one

This release was live on PyPI for less than two hours. A cooldown policy, which holds back any package version published in the last few days, would have kept it off every host that respects the policy. [Chainguard](https://www.chainguard.dev/unchained/litellm-attacked-again-malicious-1-107-0-wheels-on-pypi-and-docker-hub-october-2026) and [Cycode](https://cycode.com/blog/litellm-1-107-0-compromise/) both recommend one after this incident. Pair it with hash-pinned lockfiles and container images pulled by digest instead of `latest`, then use Fleet's software inventory to confirm the pins held.

## Where this leaves you

Remove the files, stop the daemon, reinstall at 1.106.2, and rotate secrets, in that order. Then confirm from the hosts, not from the advisory, that nothing is still running 1.107.0. The advisory tells you what was bad. Your hosts tell you what's installed.

## See it live

- **Get a demo** at [fleetdm.com/contact](https://fleetdm.com/contact) to see software inventory across Linux, macOS, and Windows hosts.
- **Run a report on your own fleet.** Learn how in [Reports](https://fleetdm.com/guides/reports).

## Further reading

- [Detecting the Mini Shai-Hulud npm supply chain worm with Fleet](https://fleetdm.com/securing/detecting-the-mini-shai-hulud-npm-supply-chain-worm-with-fleet)

## Sources

- LiteLLM, [Contained: Oct 21, 2026 LiteLLM release v1.107.0 compromised](https://docs.litellm.ai/blog/security-incident-oct-21-2026).
- The Hacker News, [LiteLLM compromised in supply chain attack delivering credential stealer and backdoor](https://thehackernews.com/2026/10/litellm-compromised-in-supply-chain.html).
- Chainguard, [LiteLLM attacked again: Malicious 1.107.0 wheels on PyPI and Docker Hub (October 2026)](https://www.chainguard.dev/unchained/litellm-attacked-again-malicious-1-107-0-wheels-on-pypi-and-docker-hub-october-2026).
- Cycode, [LiteLLM 1.107.0: Stopping a poisoned package before it installs](https://cycode.com/blog/litellm-1-107-0-compromise/).
- The Hacker News, [LiteLLM suspects GitHub token leak in AI gateway logs behind malicious package release](https://thehackernews.com/2026/10/litellm-suspects-github-token-leak-in.html).

<meta name="articleTitle" value="LiteLLM 1.107.0 backdoor: Why running pip reruns the malware, and how to find affected hosts">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-22">
<meta name="description" value="LiteLLM 1.107.0 reruns its backdoor on every Python start, pip included. Here's the cleanup order and how to find every host that installed it.">
