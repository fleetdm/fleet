# Tensorlake 0.5.144 shipped with valid build provenance and still stole credentials

*A signed, provenance-stamped npm release is not proof that the code inside is clean. When the Tensorlake SDK was compromised, the only reliable answer to "who pulled it?" was an inventory of what developers' machines actually installed.*

## Key takeaways

- **Provenance told you where the build ran, not whether it was safe.** Endor Labs reports the malicious tensorlake release carried valid build provenance, because the attacker pushed straight to the repository's main branch and released from there. Trust signals on the package are not a substitute for knowing what landed on your hosts.
- **The malware ran at install time, so "we never imported it" is not an alibi.** A preinstall hook executes when the package is installed, which means a developer who only pulled it in as a dependency of something else is still exposed.
- **Finding the hosts is a version-matching problem you can solve fleet-wide.** Fleet's agent reports installed npm packages and versions across macOS, Windows, and Linux, so you can ask which hosts have tensorlake 0.5.144 instead of waiting for a compromised build to tell you.
- **Global-only visibility leaves a gap you need to close.** The `npm_packages` table covers globally installed packages, and most dependencies live in project-local `node_modules/` directories. A deep scan through Fleet's run-script covers the rest.
- **A hit means rotate credentials, but in the right order.** Reported targets include npm and GitHub tokens, cloud keys, SSH keys, and AI tool configuration files. One vendor warns that revoking a stolen GitHub token can trigger destructive cleanup on the host, so isolate first.
- **A standing policy beats a one-time sweep.** Once the check exists as a policy, every new or changed host is evaluated against the current known-bad list.

<a purpose="cta-button" href="https://fleetdm.com/contact">Talk to Fleet security</a>

On October 8, attackers published tensorlake@0.5.144 to npm. Tensorlake's SDK is used to build AI sandboxes and cloud applications, and according to [Socket](https://www.socket.dev/blog/tensorlake-compromise), the release contained obfuscated malware that harvests credentials, exfiltrates secrets, and sets up persistence. The bad version has since been removed from npm, and 0.5.143 is reported clean. Removal from the registry does nothing for the machines that already installed it.

That is the problem a security team inherits the next morning. The package is gone, the advisory is out, and the open question is which of your developer laptops and build hosts pulled 0.5.144 in the window it was live. Here is how to answer that, and what to do when the answer is yes.

## What the compromise did

According to [Endor Labs](https://www.endorlabs.com/learn/tensorlake-npm-package-compromised-by-shai-hulud-in-latest-software-supply-chain-attack), the malicious files were pushed directly to the main branch of the tensorlakeai/tensorlake repository under a maintainer's name, and the package was then released from that same repository. Because the release came out of the project's own pipeline, it carried valid build provenance. Provenance attests to where and how an artifact was built. It cannot tell you that the source going into the build was legitimate.

Socket and others tie the incident to the ChainDrop and Shai-Hulud family of npm worms. The reported targets include npm and GitHub tokens, AWS credentials, Kubernetes credentials, SSH keys, and configuration and MCP files for AI coding tools such as Claude, Cursor, Kiro, Windsurf, and Zed. Some coverage also describes dropped files that re-trigger the malware when a project is reopened in Claude Code or VS Code. We could not confirm that last detail in vendor write-ups, so treat it as a reported behavior to check for rather than an established indicator.

The execution path is what makes the version check urgent. A preinstall hook runs during `npm install`. Nobody has to import the SDK or start an agent for it to fire, and a developer who received tensorlake as a transitive dependency of something else ran the malware just the same.

## Finding the hosts that installed it

Start with the fast, fleet-wide check. Fleet's `npm_packages` table reports globally installed packages and versions, and a live report returns matches across every host in seconds:

```sql
SELECT name, version, path FROM npm_packages
WHERE name = 'tensorlake' AND version = '0.5.144';
```

Match on name and version together. Only 0.5.144 is reported as malicious, so a host running 0.5.143 should not light up. Keep the pattern but expect the set of bad versions to change if the campaign widens.

### Mind the global-only gap

The `npm_packages` table only covers global installs, looking at default paths like `~/.npm-global`, `/usr/local/lib/node_modules`, and `/opt/homebrew/lib/node_modules`. Almost nobody installs project dependencies globally. A developer whose project carries tensorlake in its own `node_modules/` directory will return zero rows from the query above. We hit the same limit documenting [an earlier npm worm](https://fleetdm.com/articles/detecting-the-chaindrop-npm-worm-before-it-reaches-your-devices), so run the query as a quick global check and not as a verdict.

### Close the gap with a deep scan

Pair the query with a filesystem scan through Fleet's run-script feature. It walks every `node_modules/` directory under user home directories and reads each package manifest:

```bash
#!/bin/sh
# Look for tensorlake 0.5.144 in project-local node_modules.
find "$HOME" -maxdepth 10 -type d -name node_modules 2>/dev/null | while read -r nm; do
  pkg="$nm/tensorlake/package.json"
  [ -f "$pkg" ] || continue
  version=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$pkg" | head -1)
  if [ "$version" = "0.5.144" ]; then
    echo "COMPROMISED: tensorlake@${version} at ${pkg}"
  fi
done
```

A lockfile can also record the version even after `node_modules/` is cleaned up. If your developers commit `package-lock.json`, searching repositories for `tensorlake` pinned at 0.5.144 tells you who resolved it, though not necessarily who installed it.

## Responding when you find a hit

Treat any host with 0.5.144 as having exposed every credential it could read. That includes the token types above and anything in the home directories of its users.

Order matters. StepSecurity [warns](https://stepsecurity.io/blog/tensorlake-npm-compromised-hostage-token-worm) that if you revoke the GitHub token the malware stole, a background job on the host deletes the home directory. This is one vendor's report, but the cost of being wrong is high, so the safer sequence is to isolate the machine from the network first, preserve what you need for investigation, and then rotate and revoke. Rotate from a clean machine.

## Make it a standing check

A one-time sweep answers today's question. Supply chain campaigns keep producing new bad versions, and a laptop that was offline last week will check in tomorrow. Save the version check as a Fleet policy, and every host that enrolls or changes is evaluated against the current list without anyone remembering to rerun it. Fleet's agent runs on macOS, Windows, and Linux, so the same policy covers the developer on a MacBook and the build runner on Linux.

## What to take from it

The registry pulled the package within days, and the provenance badge never flagged it at all. Neither of those tells you which of your machines ran the code. Only an inventory of what is installed, checked against a known-bad list on a schedule, closes that gap.

## See it live

- **Get a demo.** [Talk to Fleet](https://fleetdm.com/contact) about checking software versions across your fleet.
- **Join a GitOps training session.** Learn how to keep checks like this one in version control at the [GitOps workshop](https://fleetdm.com/gitops-workshop).

<meta name="articleTitle" value="Tensorlake 0.5.144 shipped with valid build provenance and still stole credentials">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-10-12">
<meta name="category" value="industry news">
<meta name="description" value="Tensorlake npm 0.5.144 had valid build provenance yet harvested credentials. How to find every host that installed it, and what to rotate first.">
