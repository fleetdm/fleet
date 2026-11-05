# How a single asterisk left Alpine Linux open to any long password for ten years

*On Alpine Linux, any password over 256 bytes can open accounts that were never meant to take one. Here's what happened, who's exposed, and what to check this week.*

## Key takeaways

- **One oversized password gets an attacker in.** On affected systems, any password from 257 to 1,024 bytes opens accounts that have no password set.
- **Alpine's official cloud images can turn that login into root.** Their default `alpine` user runs `doas` without a password.
- **Debian, Ubuntu, Fedora, and RHEL aren't affected.** The bug needs the musl C library without PAM, and those distributions use glibc.
- **Updating the package doesn't finish the job.** The old sshd keeps running until it restarts, and containers need a rebuild on a patched base.

<a purpose="cta-button" href="https://fleetdm.com/linux-management">See how Fleet watches containers on your Linux servers</a>

On November 5, 2026, OpenSSH released version 10.6p2 to fix what the project called a critical vulnerability, and Alpine Linux released patched packages the same day. Maintainer Damien Miller's [announcement](https://marc.info/?l=openssh-unix-dev&m=179388064263131&w=2) told anyone running musl without PAM to turn off password logins or SSH "IMMEDIATELY."

The flaw is [CVE-2026-123158](https://nvd.nist.gov/vuln/detail/CVE-2026-123158), rated 8.1 (high). Özgün Kültekin of Wakefield found it and named it *sshole. CISA's notes in the CVE record list public proof-of-concept code.

## Why it matters to the business

Most SSH break-ins start with a stolen password, a leaked key, or noisy guessing. This attack needs only network access, the name of an account with no password, and one login attempt. A single successful try doesn't look like an attack in most monitoring.

Alpine is also easy to lose track of, because teams pick it for small containers and cloud images that run for years unnoticed.

## How two safe decisions added up to a bypass

Linux marks a no-password account with an asterisk where the password hash would go. Nothing hashes to a lone asterisk, so the account stays shut to passwords and still works with SSH keys.

The bypass comes from two choices that were each reasonable on their own. In musl, the `crypt()` function that hashes passwords returns `*` when it fails, and any password over 256 bytes makes it fail. In 2016, OpenSSH fixed a username-guessing bug, CVE-2016-6210, by hashing every attempt, even for accounts with no real hash, using another account's salt. BleepingComputer [reports](https://www.bleepingcomputer.com/news/security/openssh-flaw-lets-attackers-log-into-password-disabled-linux-accounts/) that this 2016 change is where the flaw began.

Put together, an oversized password makes the hash fail and return `*`. The SSH daemon compares that with the stored `*`, finds a match, and logs the attacker in.

Kültekin found the bug by having AI agents run about 17,600 login tests across OpenSSH builds, according to BleepingComputer. Miller wrote that the team released the fix a day after learning of it, because AI tools now rediscover bugs within hours.

## Who's exposed

[Alpine's advisory](https://alpinelinux.org/posts/2026-11-05-openssh-password-authentication-bypass/) says a host is vulnerable only when all four conditions hold:

- It runs `openssh-server`, which Alpine builds without PAM, or `openssh-server-pam` with `UsePAM no`.
- `PasswordAuthentication` is `yes`, which is OpenSSH's default.
- At least one account has `*` in its password field.
- Some other account on the machine has a real `$5$` or `$6$` hash.

Every Alpine release from 3.16 through 3.24 and edge is affected, and the official cloud images become exploitable once any account gets a real password hash. BleepingComputer adds that OpenWrt, Buildroot, and Yocto builds running OpenSSH the same way are in scope.

## How Fleet helps you find them

Alpine usually runs inside containers on the Linux servers Fleet already manages. Fleet's agent, built on [osquery](https://fleetdm.com/tables/docker_containers), sees every running Docker container on those servers, the operating system inside it, and the processes it runs. A Fleet [report](https://fleetdm.com/guides/reports) can ask each server one question: is an Alpine container here running sshd? The answer gives you a short list of containers to open first.

A Fleet [policy](https://fleetdm.com/securing/what-are-fleet-policies) keeps asking that question every hour, so a container launched next month gets flagged too. A [policy automation](https://fleetdm.com/docs/using-fleet/automations) can open a ticket for the owning team when it does. Fleet's [Linux management](https://fleetdm.com/linux-management) page shows what else the agent reports from your servers.

## How to respond

On each Alpine host, run `apk upgrade -U -a`, then `rc-service sshd restart`. Fixed builds include 10.3_p1-r2 on 3.24 and 10.2_p1-r1 on 3.23, and Alpine has patched every branch back to 3.16. For containers, rebuild from a fixed base image such as Alpine 3.24.4 and redeploy.

If you can't update yet, set `PasswordAuthentication no` and restart sshd. To check for past abuse, search `/var/log/messages` for `Accepted password` entries on accounts that have no password. Alpine says to treat any machine with one as compromised.

## Questions to ask your team

- Do the images we build turn password logins off by default?
- If the next fix lands the same day as the advisory, how fast can we find every host that needs it?

## The fix is easy once you know where to look

On a single server, patching takes one command and a restart. Finding every machine that needs it is the real work, and that list should exist before the next advisory.

## See it live

See how Fleet spots Alpine containers running sshd across your Linux servers, and how a policy keeps flagging new ones as they appear.

<a purpose="cta-button" href="https://fleetdm.com/contact">Get a demo of Fleet</a>

- **Join the [Fleet Virtual Summit](https://www.linkedin.com/events/7518683538410459136)** on Tuesday, November 10, at 11 AM ET. Four practitioner panels on GitOps, open source, and AI agents stream free on LinkedIn Live.
- **Take a [GitOps workshop](https://fleetdm.com/workshops)** to learn how to write policies like the Alpine sshd check, keep them in Git, and review every change before it ships.

## Sources

- Alpine Linux, [Security advisory: openssh-server password authentication bypass](https://alpinelinux.org/posts/2026-11-05-openssh-password-authentication-bypass/), November 5, 2026
- Damien Miller, [urgent security release: 10.6p2](https://marc.info/?l=openssh-unix-dev&m=179388064263131&w=2), openssh-unix-dev mailing list, November 5, 2026
- NIST NVD, [CVE-2026-123158](https://nvd.nist.gov/vuln/detail/CVE-2026-123158)
- Lawrence Abrams, [OpenSSH flaw lets attackers log into password-disabled Linux accounts](https://www.bleepingcomputer.com/news/security/openssh-flaw-lets-attackers-log-into-password-disabled-linux-accounts/), BleepingComputer, November 5, 2026

<meta name="articleTitle" value="How a single asterisk left Alpine Linux open to any long password for ten years">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-05">
<meta name="description" value="On Alpine, sshd accepts any 257+ byte password for accounts with no password. Find exposed hosts and containers, patch, and restart sshd.">
