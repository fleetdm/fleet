# Find and disable Tailscale SSH forwarding with Fleet

On October 25, 2026, Erica Windisch of Cyberdione Labs published [Tailworm](https://tailworm.com/), a report with 11 findings against Tailscale 1.104.2 and headscale 0.29.5. The most serious one is in Tailscale SSH. When a client asks to forward a Unix socket, `tailscaled` opens, creates, or replaces that socket itself, as root, instead of as the user who logged in. Anyone an SSH rule lets in as an ordinary user can use that to reach sockets only root can reach, like Docker's or systemd's, and from there get root. Tailscale [says](https://thehackernews.com/2026/10/unpatched-tailscale-ssh-flaws-could-let.html) it is working on fixes. As of October 26, there is no Tailscale security bulletin, CVE, or fixed release. This guide shows you how to find the hosts that run a Tailscale SSH server, turn forwarding off on them, and check for signs of misuse, with Fleet reports, policies, and scripts.

## What's affected

- **Hosts that run a Tailscale SSH server.** That's Linux, and macOS with the open-source `tailscaled` from Homebrew or `tailscaled install-system-daemon`. Windows and the Tailscale app for macOS (App Store or tailscale.com) don't run a Tailscale SSH server. Any device on the tailnet can be the attacker's starting point.
- **Tailscale 1.98.0 through 1.104.2.** Tailscale SSH gained Unix socket forwarding in 1.98.0, and 1.104.2 is the latest release.
- **An SSH rule that admits the attacker's device, with forwarding allowed.** headscale 0.29.5 allows forwarding on every SSH rule, and the policy file has no setting to turn it off. The report says Tailscale's hosted control plane lets you deny forwarding per rule, but Tailscale's policy file reference doesn't document that setting yet.

Two common defenses don't help. Limiting SSH rules to `autogroup:nonroot` doesn't, because the forwarding runs as root no matter which user logs in. Check mode doesn't either, because it only adds a login step before the session starts.

Fleet reproduced the core finding in a lab with Ubuntu 24.04, Tailscale 1.104.2, and headscale 0.29.5. A user who got `Permission denied` on a root-only socket on the host reached it through `ssh -L` over Tailscale SSH. With forwarding disabled, the same request failed with `administratively prohibited: unix forwarding is disabled`, and ordinary sessions kept working.

The report also covers app connectors, `tailscale serve`, DERP relays, and headscale memory use. This guide covers SSH forwarding and `serve` targets, because those are the parts you can see and change on the host.

## Prerequisites

- A Fleet account with permission to run live reports, add policies, and run scripts.
- [Scripts enabled](https://fleetdm.com/guides/scripts) on Fleet's agent (fleetd) for the hosts you want to fix.
- fleetd on the hosts. The reports and policies read Tailscale's state file with the `file_lines` table, which ships with fleetd.
- Admin access to your Tailscale admin console or your headscale policy file, for step 4.

> **Warning:** The scripts in this guide restart `tailscaled`. That drops every Tailscale connection on the host for a few seconds, including open Tailscale SSH sessions. If fleetd reaches your Fleet server over Tailscale, expect a short gap in check-ins. Run the scripts on a test host first, then attach them to the policies.

## Step 1: Find hosts that run Tailscale SSH

Add the two reports in [`tailscale-ssh-forwarding.reports.yml`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/reports/tailscale-ssh-forwarding.reports.yml), one for Linux and one for macOS, or paste a report's SQL into a live report in the Fleet UI. Each returns one row per finding with the columns `finding`, `item`, `detail`, and `status`. No row for a finding means it wasn't found.

| Finding | What it means |
|---|---|
| `tailscale_package` (Linux), `homebrew_tailscale` (macOS) | The installed Tailscale version. `UPDATE` means the host is also missing earlier Tailscale SSH fixes. On openSUSE Leap 15.6, Fleet's agent returned no RPM packages in Fleet's lab, so this row is missing there. The other rows work. |
| `tailscale_app` (macOS) | The Tailscale app, which doesn't run a Tailscale SSH server. No action needed for this report. |
| `tailscaled` | The running daemon and its command line. |
| `tailscale_ssh` | `EXPOSED` when Tailscale SSH is on and forwarding is allowed. `FORWARDING_DISABLED` after step 2 or 3, `OFF` when Tailscale SSH is off, and `UNKNOWN` when the state file can't be read. Treat `UNKNOWN` as exposed. |
| `serve_unix_target` | A `tailscale serve` configuration that forwards to a Unix socket. See step 4. |
| `ssh_session` | A live Tailscale SSH session, shown as a child process of `tailscaled`. |
| `tailscaled_unix_listener` | A Unix socket that `tailscaled` listens on outside its own directory. That's a remote (`-R`) forward or a `serve` target. See step 5. |
| `host` | The OS version, for context. |

The reports read the `RunSSH` setting of the current Tailscale profile from the state file, at the path in the `--state` or `--statedir` flag, or the default (`/var/lib/tailscale/tailscaled.state` on Linux, `/Library/Tailscale/tailscaled.state` on macOS). A `tailscaled` that has never logged in has nothing to expose and shows `OFF`. State that's sealed with a TPM (`--encrypt-state`) or not kept in a file (`mem:`, `kube:`) can't be read, so those hosts show `UNKNOWN`.

> **Note:** `ssh_session` and `tailscaled_unix_listener` are a point-in-time view. A forwarding-only connection (`ssh -N`) has no session process, so it only shows up as a listener, and only while it's open.

## Step 2: Turn off forwarding on Linux

`tailscaled` has a built-in switch, `TS_SSH_DISABLE_FORWARDING`, that refuses TCP port forwarding, Unix socket forwarding, and agent forwarding in both directions. It's an environment variable that `tailscaled` reads at startup, and every release with Unix socket forwarding supports it. Tailscale doesn't document it as a setting, so treat it as a stopgap until the fix ships.

Add this policy. It passes when `tailscaled` isn't running, when Tailscale SSH is off, or when the running `tailscaled` has `TS_SSH_DISABLE_FORWARDING=1`:

```yaml
- name: Linux - Tailscale SSH forwarding disabled (Tailworm interim mitigation)
  query: |
    WITH td AS (
      -- Pad and normalize flags so " -state=" also matches --state, "-state /path", and never --encrypt-state
      SELECT pid, replace(replace(replace(' ' || cmdline || ' ', ' --', ' -'), ' -state ', ' -state='), ' -statedir ', ' -statedir=') AS args
      FROM processes
      WHERE name = 'tailscaled' AND cmdline NOT LIKE '% be-child %'
    ),
    state AS (
      SELECT pid, CASE
          WHEN instr(args, ' -state=') > 0 THEN
            replace(substr(args, instr(args, ' -state=') + 8, instr(substr(args, instr(args, ' -state=') + 8), ' ') - 1), 'tpmseal:', '')
          WHEN instr(args, ' -statedir=') > 0 THEN
            substr(args, instr(args, ' -statedir=') + 11, instr(substr(args, instr(args, ' -statedir=') + 11), ' ') - 1) || '/tailscaled.state'
          ELSE '/var/lib/tailscale/tailscaled.state'
        END AS path
      FROM td
    ),
    ssh AS (
      -- file_lines aborts the query on a missing path, so every read sits behind a CASE that checks the file first.
      -- Values are base64; prefs are JSON. Missing, TPM-sealed, or non-file state counts as on.
      SELECT s.pid, CASE
          WHEN NOT EXISTS (SELECT 1 FROM file f WHERE f.path = s.path) THEN 1
          WHEN EXISTS (SELECT 1 FROM file_lines l WHERE l.path = s.path AND l.line LIKE '%"nonce":%') THEN 1
          ELSE coalesce((
            SELECT max(json_extract(from_base64(rtrim(substr(l.line, instr(l.line, '": "') + 4), '",')), '$.RunSSH'))
            FROM file_lines l
            WHERE l.path = s.path AND l.line LIKE '%"profile-%'
              -- Only the current profile counts; without a current-profile key, any profile does
              AND (NOT EXISTS (SELECT 1 FROM file_lines c WHERE c.path = s.path AND c.line LIKE '%"\_current-profile":%' ESCAPE '\')
                   OR l.line LIKE '%"' || (SELECT from_base64(rtrim(substr(c.line, instr(c.line, '": "') + 4), '",'))
                                           FROM file_lines c WHERE c.path = s.path AND c.line LIKE '%"\_current-profile":%' ESCAPE '\') || '":%')
          ), 0)
        END AS ssh_on
      FROM state s
    )
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM ssh
      WHERE ssh_on = 1
        AND NOT EXISTS (SELECT 1 FROM process_envs e
                        WHERE e.pid = ssh.pid AND e.key = 'TS_SSH_DISABLE_FORWARDING'
                          AND lower(e.value) IN ('1', 't', 'true'))
    );
  critical: false
  description: |
    Passes when tailscaled isn't running, when Tailscale SSH is off for the current profile
    (including a tailscaled that never logged in), or when tailscaled runs with
    TS_SSH_DISABLE_FORWARDING=1. Fails when Tailscale SSH is on and forwarding is allowed,
    which lets any device an SSH rule admits use tailscaled's root privileges to reach Unix
    sockets on this host. Also fails when the state can't be read (TPM-sealed, mem:, kube:,
    or no file at the path), because Tailscale SSH can't be ruled out.
  resolution: |
    Run disable-tailscale-ssh-forwarding-linux.sh. It adds a systemd drop-in that sets
    TS_SSH_DISABLE_FORWARDING=1 and restarts tailscaled, which drops Tailscale connections on
    the host for a few seconds. Tailscale SSH sessions keep working. If nobody needs Tailscale
    SSH on this host, run `tailscale set --ssh=false` instead.
  platform: linux
```

For hosts that fail, run [`disable-tailscale-ssh-forwarding-linux.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/linux/scripts/disable-tailscale-ssh-forwarding-linux.sh), or set it as the policy's [run-script automation](https://fleetdm.com/guides/policy-automation-run-script). The script writes a systemd drop-in at `/etc/systemd/system/tailscaled.service.d/50-disable-ssh-forwarding.conf` that sets `TS_SSH_DISABLE_FORWARDING=1`, restarts `tailscaled`, and confirms the running daemon has the setting. It exits `0` when the setting is applied or there's nothing to do, `2` when something needs a look, and `1` on error. To roll back, delete the drop-in, then run `systemctl daemon-reload` and `systemctl restart tailscaled`.

> **Note:** The policy doesn't check the Tailscale version. A host older than 1.98.0 with Tailscale SSH on also fails it. That host has no Unix socket forwarding, but it still allows TCP forwarding and is missing the Tailscale SSH fixes in 1.98.9, so update it and turn forwarding off.

> **Note:** The script only handles `tailscaled` running as the `tailscaled.service` systemd unit, which is how Tailscale's packages install it. In a container, on a host with a custom service, or when someone started `tailscaled` by hand, it exits `2`. Set `TS_SSH_DISABLE_FORWARDING=1` in the environment that starts `tailscaled` there, then restart it. It also exits `2` when another setting overrides the drop-in, such as `TS_SSH_DISABLE_FORWARDING=0` in `/etc/default/tailscaled`. Fleet shows an exit `2` as a failed script run, retries the automation up to three times, and the host keeps failing the policy until you fix it by hand.

If nobody needs Tailscale SSH on a host, turn it off instead. Make sure you have another way in first:

```bash
tailscale set --ssh=false
```

## Step 3: Turn off forwarding on macOS

On a Mac, only the open-source `tailscaled` runs a Tailscale SSH server. It reads extra environment variables from `/etc/tailscale/tailscaled-env.txt` at startup. Add this policy. It passes when the open-source `tailscaled` isn't running, when Tailscale SSH is off, or when `tailscaled` started after that file set `TS_SSH_DISABLE_FORWARDING=1`. Because `tailscaled` applies the file's lines in order, the policy fails if any other line sets a different value:

```yaml
- name: macOS - Tailscale SSH forwarding disabled (Tailworm interim mitigation)
  query: |
    WITH td AS (
      -- Open-source tailscaled only: the Tailscale app (App Store or standalone) runs as a network
      -- extension and doesn't run a Tailscale SSH server.
      -- Pad and normalize flags so " -state=" also matches --state, "-state /path", and never --encrypt-state
      SELECT pid, start_time, replace(replace(replace(' ' || cmdline || ' ', ' --', ' -'), ' -state ', ' -state='), ' -statedir ', ' -statedir=') AS args
      FROM processes
      WHERE name = 'tailscaled' AND cmdline NOT LIKE '% be-child %'
    ),
    state AS (
      SELECT pid, CASE
          WHEN instr(args, ' -state=') > 0 THEN
            substr(args, instr(args, ' -state=') + 8, instr(substr(args, instr(args, ' -state=') + 8), ' ') - 1)
          WHEN instr(args, ' -statedir=') > 0 THEN
            substr(args, instr(args, ' -statedir=') + 11, instr(substr(args, instr(args, ' -statedir=') + 11), ' ') - 1) || '/tailscaled.state'
          ELSE '/Library/Tailscale/tailscaled.state'
        END AS path
      FROM td
    ),
    ssh AS (
      -- file_lines aborts the query on a missing path, so every read sits behind a CASE that checks the file first.
      -- Values are base64; prefs are JSON. A missing state file counts as on.
      SELECT s.pid, CASE
          WHEN NOT EXISTS (SELECT 1 FROM file f WHERE f.path = s.path) THEN 1
          ELSE coalesce((
            SELECT max(json_extract(from_base64(rtrim(substr(l.line, instr(l.line, '": "') + 4), '",')), '$.RunSSH'))
            FROM file_lines l
            WHERE l.path = s.path AND l.line LIKE '%"profile-%'
              -- Only the current profile counts; without a current-profile key, any profile does
              AND (NOT EXISTS (SELECT 1 FROM file_lines c WHERE c.path = s.path AND c.line LIKE '%"\_current-profile":%' ESCAPE '\')
                   OR l.line LIKE '%"' || (SELECT from_base64(rtrim(substr(c.line, instr(c.line, '": "') + 4), '",'))
                                           FROM file_lines c WHERE c.path = s.path AND c.line LIKE '%"\_current-profile":%' ESCAPE '\') || '":%')
          ), 0)
        END AS ssh_on
      FROM state s
    ),
    env_file AS (
      -- tailscaled reads this file once at startup, so it only counts if tailscaled started after it changed
      SELECT f.mtime,
        CASE WHEN EXISTS (
          SELECT 1 FROM file_lines l WHERE l.path = '/etc/tailscale/tailscaled-env.txt'
            AND trim(substr(l.line, 1, instr(l.line, '=') - 1)) = 'TS_SSH_DISABLE_FORWARDING'
            AND lower(trim(trim(substr(l.line, instr(l.line, '=') + 1)), '"')) IN ('1', 't', 'true'))
          -- tailscaled applies the lines in order, so any other value for the key may win
          AND NOT EXISTS (
            SELECT 1 FROM file_lines l WHERE l.path = '/etc/tailscale/tailscaled-env.txt'
              AND trim(substr(l.line, 1, instr(l.line, '=') - 1)) = 'TS_SSH_DISABLE_FORWARDING'
              AND lower(trim(trim(substr(l.line, instr(l.line, '=') + 1)), '"')) NOT IN ('1', 't', 'true'))
        THEN 1 ELSE 0 END AS knob
      FROM file f WHERE f.path = '/etc/tailscale/tailscaled-env.txt'
    )
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM ssh JOIN td USING (pid)
      WHERE ssh.ssh_on = 1
        AND NOT EXISTS (SELECT 1 FROM env_file WHERE knob = 1 AND mtime <= td.start_time)
        AND NOT EXISTS (SELECT 1 FROM process_envs e
                        WHERE e.pid = td.pid AND e.key = 'TS_SSH_DISABLE_FORWARDING'
                          AND lower(e.value) IN ('1', 't', 'true'))
    );
  critical: false
  description: |
    Passes when the open-source tailscaled isn't running (the Tailscale app from the App Store
    or tailscale.com doesn't run a Tailscale SSH server), when Tailscale SSH is off for the
    current profile, or when tailscaled started after /etc/tailscale/tailscaled-env.txt set
    TS_SSH_DISABLE_FORWARDING=1 with no other line setting a different value (tailscaled
    applies the lines in order), or when its launchd environment sets it. Fails when Tailscale
    SSH is on and forwarding is allowed.
  resolution: |
    Run disable-tailscale-ssh-forwarding-macos.sh. It writes the setting to
    /etc/tailscale/tailscaled-env.txt and restarts tailscaled. If nobody needs Tailscale SSH
    on this Mac, run `tailscale set --ssh=false` instead.
  platform: darwin
```

For hosts that fail, run [`disable-tailscale-ssh-forwarding-macos.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/scripts/disable-tailscale-ssh-forwarding-macos.sh). It sets `TS_SSH_DISABLE_FORWARDING=1` in `/etc/tailscale/tailscaled-env.txt`, replaces any existing line for that setting, keeps the file's other lines, and restarts the `com.tailscale.tailscaled` or `homebrew.mxcl.tailscale` launch daemon. On a Mac with only the Tailscale app, it exits `0` without changing anything.

> **Note:** Fleet ran the macOS policy, report, and script on macOS against a stand-in daemon managed by launchd, but not against the open-source `tailscaled` running as root. Try the script on one Mac before you attach it to the policy.

## Step 4: Tighten SSH rules and serve targets

Host settings close the forwarding path. The control plane decides who can reach it.

1. In your policy file, replace broad SSH rules with narrow ones. A rule whose `src` is a whole group or tag lets every device in it reach every `dst`. The report recommends one source and one destination per rule.
2. On headscale, treat every SSH rule as granting root on its `dst` until Tailscale ships a fix, because headscale can't turn forwarding off. The host switch from steps 2 and 3 is your only control.
3. On Tailscale's hosted control plane, ask Tailscale support how to deny forwarding per rule. Fleet couldn't find the setting in Tailscale's published policy file reference as of October 26.
4. Remove `tailscale serve` configurations that forward to a Unix socket you didn't intend to share. The report shows a `serve` forward to `tailscaled`'s own socket giving remote peers its local API. Turn one off with `tailscale serve --tcp=<port> off`, or clear all of them with `tailscale serve reset`.
5. Turn on [key expiry](https://tailscale.com/docs/features/access-control/key-expiry) and, if you can, [Tailnet Lock](https://tailscale.com/docs/features/tailnet-lock). The report's chain from root on one host to enrolling a rogue node didn't finish where Tailnet Lock was on.
6. Turn on Tailscale's [automatic updates](https://tailscale.com/docs/features/client/update) with `tailscale set --auto-update`, so the fix lands as soon as it ships.

## Step 5: Look for signs of misuse

Exploiting this needs a device the attacker already controls and an SSH rule that admits it, so start by reviewing who your SSH rules admit. On the host, the step 1 reports show three things worth a look:

- `tailscaled_unix_listener`: `tailscaled` listening on a Unix socket in a user's home directory, `/tmp`, or `/run/user/<uid>`. In Fleet's lab, a remote forward created its socket as `root:root` with mode `0600`, even though the session user was an ordinary user. A root-owned socket in a place where an ordinary user forwarded one is a sign of this path being used.
- `serve_unix_target`: a `serve` configuration nobody remembers adding, especially one that points at `/run/tailscale/tailscaled.sock`, `/run/docker.sock`, or `/run/systemd/private`.
- `ssh_session`: who's connected right now, and as which local user.

In `tailscaled`'s log (`journalctl -u tailscaled` on Linux), every Tailscale SSH connection logs a line like `ssh-conn-<id>: handling conn: <source IP>:<port>-><user>@<host IP>:22`. Compare the source devices against the ones you expect.

The report lists more signals: auth keys issued or used outside change windows, a device joining with a key you haven't seen, and Tailscale login prompts nobody asked for. If you find a compromised host, delete the device in your Tailscale admin console or headscale. Reimaging doesn't revoke keys that were already copied off it. Then rotate any auth keys the host could reach, and your Tailnet Lock keys if it was a signing node.

## Verify

1. Go to **Policies** and confirm both Tailscale SSH forwarding policies show no failing hosts.
2. Re-run the step 1 reports. Every `tailscale_ssh` row should read `OFF` or `FORWARDING_DISABLED`.
3. On a Linux host, confirm the running daemon has the setting:

   ```bash
   tr '\0' '\n' < "/proc/$(systemctl show -p MainPID --value tailscaled)/environ" | grep TS_SSH_DISABLE_FORWARDING
   ```

4. From a device your SSH rules allow, try a forward. It should fail with `administratively prohibited: unix forwarding is disabled`:

   ```bash
   ssh -N -L 127.0.0.1:9000:/tmp/test.sock <user>@<host>
   ```

## When Tailscale ships a fix

Commits in Tailscale's [public repository](https://github.com/tailscale/tailscale/commits/main/ssh/tailssh) from October 24 move SSH forwarding into a separate process that runs as the session user. No release includes them yet. Watch Tailscale's [security bulletins](https://tailscale.com/security-bulletins) for the fixed version.

When it ships, update Tailscale on every host that runs a Tailscale SSH server, with your package manager or `tailscale update`. The step 1 reports show each host's version, and Fleet's software inventory lists Tailscale on every host. Add a policy that checks for the fixed version, so new hosts can't slip in on an old one. Keep forwarding off on hosts that never needed it.

Fleet's vulnerability detection matches software against published CVEs. Tailscale hasn't published a CVE for these findings yet, so nothing shows up under **Software > Vulnerabilities** for them. Use the reports and policies in this guide until it does.

## Further reading

- [Tailworm: security assessment report for Tailscale 1.104.2 and headscale 0.29.5](https://tailworm.com/)
- [The Hacker News: Unpatched Tailscale SSH flaws could let a hacked device reach root on other servers](https://thehackernews.com/2026/10/unpatched-tailscale-ssh-flaws-could-let.html)
- [Tailscale security bulletins](https://tailscale.com/security-bulletins)
- [Tailscale SSH](https://tailscale.com/docs/features/tailscale-ssh)
- [Reports in Fleet](https://fleetdm.com/guides/reports)
- [Scripts in Fleet](https://fleetdm.com/guides/scripts)
- [Automatically run scripts on failing policies](https://fleetdm.com/guides/policy-automation-run-script)

<meta name="articleTitle" value="Find and disable Tailscale SSH forwarding with Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-26">
<meta name="category" value="guides">
<meta name="description" value="Find hosts that run a Tailscale SSH server, turn off the forwarding path in the Tailworm report, and watch for misuse with Fleet.">
