# Tailworm: what to do about Tailscale SSH before the fix ships

*A researcher says one compromised device can turn Tailscale SSH into root on the servers it reaches. There's no patch yet. Here's how to find out how many of your hosts that describes, and how to close the path today.*

## Key takeaways

- **The bug is in the forwarding, not the login.** Tailscale SSH authenticates the user correctly, then performs Unix socket forwarding as root. Any device an SSH rule admits can reach sockets its user never could.
- **Your usual SSH hardening doesn't cover it.** Non-root users, check mode, and firewall rules all leave the forwarding path open, because none of them change who opens the socket.
- **Exposure is a host setting you can read.** Whether a server runs Tailscale SSH lives in Tailscale's state file on the host. Fleet's agent decodes it and reports which Linux servers and Macs are exposed.
- **One environment variable closes the path today.** `TS_SSH_DISABLE_FORWARDING=1` makes `tailscaled` refuse every kind of forwarding, and ordinary sessions keep working. Fleet verified both in a lab.
- **A policy keeps it closed.** A Fleet policy fails on any host where Tailscale SSH can still forward, and a script fixes it on the next check-in, including hosts that join later.
- **The control plane still decides who can try.** Narrow SSH rules, key expiry, and Tailnet Lock limit how far a stolen device gets, and they're worth doing whatever the patch says.

<a purpose="cta-button" href="https://fleetdm.com/linux-management">See Linux management in Fleet</a>

On October 25, 2026, Erica Windisch of Cyberdione Labs published [Tailworm](https://tailworm.com/), 11 findings from three days of lab testing against Tailscale 1.104.2 and headscale 0.29.5. The headline finding: with Tailscale SSH turned on, one compromised device that an SSH rule lets in can get root on the server it connects to. Windisch calls it wormable, and advises turning Tailscale SSH off.

Tailscale [told](https://thehackernews.com/2026/10/unpatched-tailscale-ssh-flaws-could-let.html) The Hacker News it had received the report, had already been looking at the same areas with outside specialists, and is fixing what it found. It also said it isn't recommending that everyone turn Tailscale SSH off. As of October 26, there's no Tailscale security bulletin, no CVE, and no fixed release. That leaves the next move with you, and the first question is the same as with any vulnerability: which of my hosts does this describe?

## What the report found

Tailscale SSH replaces OpenSSH on a host with a server built into `tailscaled`, the Tailscale daemon. It authenticates people with their tailnet identity instead of keys, and an `ssh` rule in the tailnet's policy file decides who can log in to which host as which user.

The problem starts after login. In release 1.98.0, Tailscale SSH gained Unix socket forwarding, the feature behind `ssh -L` and `ssh -R` with socket paths. Tailscale checks that the requested path is in the user's home directory, `/tmp`, or `/run/user/<uid>`. But `tailscaled` runs as root, and in 1.98.0 through 1.104.2 it opens, creates, or replaces the socket itself. The kernel never checks the logged-in user's permissions. A root-only socket inside those directories is reachable as is, and the report shows the path check can be raced to reach sockets elsewhere, such as Docker's or systemd's. Either of those is root on the host.

The rest of the report covers app connectors that can reach destinations nobody approved, a `tailscale serve` setup that exposes `tailscaled`'s local API to peers, and memory exhaustion in headscale and the DERP relay client. Windisch says the app connector issue may matter more to some organizations than the SSH one.

Some of this is disputed. The proof-of-concept code went only to Tailscale, nobody has reproduced the findings publicly, and [commenters](https://news.ycombinator.com/item?id=50227138) on Hacker News pushed back on "wormable" and on how the report was disclosed. One thing isn't in doubt. Tailscale's public repository has commits from October 24 that move forwarding into a separate process running as the logged-in user. That change targets the root cause the report describes.

Fleet set up the core finding in a lab, with Ubuntu 24.04, Tailscale 1.104.2, and headscale 0.29.5. A user who got `Permission denied` opening a root-only socket on the host reached the same socket through `ssh -L` over Tailscale SSH. A remote forward created its socket as `root:root`, in a directory the ordinary user controls.

## Why your usual SSH defenses don't apply

Most SSH hardening is about who gets in and as whom. This bug sits after both questions have been answered correctly.

Restricting SSH rules to `autogroup:nonroot` doesn't help, because the forwarding runs as root no matter which user logged in. Check mode doesn't help either. It asks the user to sign in again before a session starts, then the session forwards like any other. Host firewalls don't see Unix socket operations at all. And on headscale 0.29.5, every SSH rule allows forwarding. In Fleet's lab, the rule headscale sent to the host had local, remote, and agent forwarding all switched on, with no setting in the policy file to change that.

What does change the outcome is on the host: whether it runs a Tailscale SSH server, and whether that server will forward.

## Exposure is a host setting you can read

Only Linux hosts, and Macs running the open-source `tailscaled` from Homebrew, run a Tailscale SSH server. Windows doesn't, and neither does the Tailscale app for macOS from the App Store or tailscale.com. That narrows the search, but you still have to look. `tailscale set --ssh` is one command, and nothing about it shows up in a package inventory.

The setting does live on disk. `tailscaled` keeps its preferences in a state file, as JSON with each value base64-encoded, and the `RunSSH` field says whether Tailscale SSH is on. Fleet's agent reads that file, decodes the value, and checks the running `tailscaled` for the forwarding switch. The [Tailscale SSH forwarding reports](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/reports/tailscale-ssh-forwarding.reports.yml) in Fleet's solutions library return one row per finding: the Tailscale version, whether Tailscale SSH is on, whether forwarding is off, and an overall `EXPOSED`, `FORWARDING_DISABLED`, or `OFF`.

The same reports surface the live signals that matter while the path is open: active Tailscale SSH sessions, Unix sockets that `tailscaled` is listening on outside its own directory (a remote forward in progress), and `tailscale serve` configurations that point at a Unix socket. A Mac with only the Tailscale app shows up too, labeled as not a Tailscale SSH server, so you can rule it out instead of wondering.

## A one-line switch, verified

`tailscaled` has an environment variable, `TS_SSH_DISABLE_FORWARDING`, that makes it refuse TCP port forwarding, Unix socket forwarding, and agent forwarding. It's in every release that has Unix socket forwarding. Tailscale doesn't document it as a setting, so treat it as a stopgap, but it's a small change with a clear effect.

In Fleet's lab, with the variable set, the same `ssh -L` request failed with `administratively prohibited: unix forwarding is disabled`, a remote forward was refused, and an ordinary Tailscale SSH session still worked. For anyone who uses Tailscale SSH to get a shell, nothing changes.

On Linux, Fleet's [script](https://github.com/fleetdm/fleet/blob/main/docs/solutions/linux/scripts/disable-tailscale-ssh-forwarding-linux.sh) adds a systemd drop-in with the variable, restarts `tailscaled`, and confirms the running daemon picked it up. On a Mac with the open-source `tailscaled`, the [macOS script](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/scripts/disable-tailscale-ssh-forwarding-macos.sh) writes it to `/etc/tailscale/tailscaled-env.txt` and restarts the launch daemon. The restart drops Tailscale connections on the host for a few seconds. Where nobody needs Tailscale SSH at all, `tailscale set --ssh=false` is the simpler answer.

## A policy keeps it closed

A report tells you where you stand today. A server that someone enables Tailscale SSH on next week won't show up in it. The [Tailscale SSH forwarding policies](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/policies/tailscale-ssh-forwarding.policies.yml) check every Linux host and Mac on every check-in. They pass when `tailscaled` isn't running, when Tailscale SSH is off, or when forwarding is disabled. If the state file can't be read, for example because it's sealed with a TPM, the policy fails, since it can't rule Tailscale SSH out.

Attach the matching script with Fleet's [run-script automation](https://fleetdm.com/guides/policy-automation-run-script), and a failing host fixes itself and turns green on its next check-in. When Tailscale ships the fix, the reports already show each host's Tailscale version, and a version check is a few lines added to the same policy file.

## The control plane still decides who can try

Exploiting this takes a device the attacker already controls and an SSH rule that admits it. Host settings close the forwarding path. The policy file decides how many hosts a single stolen laptop or poisoned CI runner can try it on.

The report's advice is worth taking whatever the patch looks like. Write SSH rules as one source and one destination, not whole groups to whole tags. Turn on key expiry, so a device key copied off a compromised host stops working. Turn on Tailnet Lock where you can: in the report, the final step of its chain, enrolling a rogue node, didn't complete on tailnets with Tailnet Lock on. Remove `tailscale serve` targets that point at Unix sockets you didn't mean to share. And if a host is compromised, delete the device in your admin console. Reimaging the host doesn't revoke keys that were already copied off it.

## The cost of waiting on a vendor

Some disclosures arrive with a patch, and the work is getting it everywhere. This one arrived without one, and the work is knowing your exposure well enough to act on a stopgap. Tailscale may be right that most tailnets aren't at risk. The only way to know whether yours is one of them is to ask your hosts.

That's the same job whatever the bug: find out which hosts run the risky thing, change the setting that matters, and keep checking so it stays changed. With Fleet, that's a report, a policy, and a script, across Linux and macOS, before the fix ships and after it lands.

## See it live

- [Find and disable Tailscale SSH forwarding with Fleet](https://fleetdm.com/guides/find-and-disable-tailscale-ssh-forwarding-with-fleet) walks through the reports, policies, and scripts step by step.
- **Get a demo** of Fleet's reports, policies, and scripts at [fleetdm.com/contact](https://fleetdm.com/contact).

<meta name="articleTitle" value="Tailworm: what to do about Tailscale SSH before the fix ships">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-26">
<meta name="category" value="articles">
<meta name="description" value="Tailworm says Tailscale SSH forwarding runs as root, and there's no patch yet. Find exposed hosts and close the path today with Fleet.">
