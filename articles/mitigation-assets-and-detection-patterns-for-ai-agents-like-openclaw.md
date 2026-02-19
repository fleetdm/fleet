# Mitigation assets and detection patterns for AI agents like OpenClaw

### Links to article series:

- Part 1: [OpenClaw: Open for work?](https://fleetdm.com/articles/openclaw-open-for-work)
- Part 2: [Detecting AI agents like OpenClaw with automated tooling](https://fleetdm.com/articles/detecting-ai-agents-like-openclaw-with-automated-tooling)
- Part 3: Mitigation assets and detection patterns for AI agents like OpenClaw

Right now, at least one person on your team is using a personal AI assistant. Tools like OpenClaw connect to local files, messaging apps, calendars, and anything else on a device. They're showing up fast.

You don't need to ban these tools to want visibility. They seek broad system access by design. They interact with files, processes, and network resources. They install persistence mechanisms and drop binaries across multiple paths. Listening ports extend the attack surface of any device they run on.

Fleet and osquery make it straightforward to get that visibility across macOS, Windows, and Linux. This article covers detection policies, investigation queries, and threat hunting queries you can deploy today.

## Why OpenClaw matters

OpenClaw has been through several name changes: it started as **Clawd**, became **Clawdbot**, was briefly renamed **Moltbot**, and is now **OpenClaw**. Each name left behind config directories, binaries, and service registrations that may still be present on devices. The CLI commands `clawdbot` and `moltbot` remain as active aliases, and the application reads from `~/.clawdbot/` and `~/.moltbot/` for backwards compatibility.

OpenClaw listens on TCP 18789 (WebSocket Gateway) and 18793 (canvas/A2UI host). It can run as a native process, an npm global package, or inside a Docker container. On macOS it registers a `launchd` service under the label `ai.openclaw.gateway` (with legacy labels `com.clawdbot.gateway` and `bot.molt.gateway`). On Linux it uses a `systemd` unit called `openclaw-gateway.service`. On Windows it runs as a Task Scheduler entry or inside WSL2. The macOS app uses the bundle identifier domain `bot.molt.*` (legacy `com.clawdbot.*`). It installs via npm, Homebrew, the official install script at `clawd.bot/install-cli.sh`, or direct binary download.

The Gateway also broadcasts its presence via mDNS using the service type `_openclaw-gw._tcp` on UDP port 5353. TXT records can expose filesystem paths, hostnames, and SSH availability to anyone on the local network.

### CVE-2026-25253

In late January 2026, security researcher Mav Levin disclosed [CVE-2026-25253](https://github.com/advisories/GHSA-g8p2-7wf7-98mq) (CVSS 8.8), a 1-click remote code execution vulnerability in OpenClaw's WebSocket gateway. The Control UI accepted a `gatewayUrl` query parameter without validation and auto-connected on page load, sending the stored auth token to whichever server was specified. An attacker could steal the token, connect back to the victim's local gateway (the victim's own browser bridges the connection even when bound to localhost), disable sandboxing via the API, and execute arbitrary commands.

The flaw was patched in version 2026.1.29, which also removed gateway auth mode "none" as a breaking change. SecurityScorecard's STRIKE team found over 42,000 exposed instances, 15,200 of them vulnerable. Hunt.io identified 17,500 more across Clawdbot and Moltbot forks.

### Supply chain campaigns

Bitdefender Labs identified four attack campaigns distributed through ClawHub, OpenClaw's public skill registry. Nearly 20% of analysed packages were malicious:

- **ClawHavoc** (300+ skills): outputs a fake error, instructs the user to run a base64-encoded command that downloads the AMOS macOS infostealer, clears Gatekeeper with `xattr -c`
- **AuthTool**: dormant reverse shell inside a Polymarket skill, activates on a natural language prompt
- **Hidden Backdoor**: executes during skill installation, shows a fake Apple Software Update, establishes an encrypted tunnel to C2
- **Credential Exfiltration**: JavaScript payload finds `.clawdbot/.env` files containing plaintext API keys, sends them to a webhook

OpenClaw runs with full user permissions and has no sandboxing by default. A compromised skill has the same access as the user who installed it.

## How Fleet policies work

[Fleet policies](https://fleetdm.com/securing/what-are-fleet-policies) are yes/no questions about your devices. You write a SQL query, and Fleet runs it on a regular interval (hourly by default). If the query returns rows, the device passes. If it returns nothing, the device fails.

For detection policies the logic is inverted: the query returns a row when the tool is absent (passing) and nothing when it's present (failing). See [Understanding the intricacies of Fleet policies](https://fleetdm.com/guides/understanding-the-intricacies-of-fleet-policies) for more on how this works.

## How the detection works

Each policy uses a scored approach with Common Table Expressions (CTEs). Individual CTEs check one detection vector (processes, ports, files, packages, services, Docker) and count the matches. A final CTE sums every count. The policy only passes if the total is 2 or below. This means a device needs three or more indicators before it fails, which avoids false positives from a single vector matching coincidentally.

Here's the pattern:

```sql
WITH process_hits AS (
    SELECT COUNT(*) AS total
    FROM processes
    WHERE name LIKE '%openclaw%'
        OR name LIKE '%clawdbot%'
        OR name LIKE '%moltbot%'
        OR name LIKE '%clawd%'
        OR cmdline LIKE '%openclaw%'
        OR cmdline LIKE '%clawdbot%'
        OR cmdline LIKE '%moltbot%'
),
-- ... more CTEs for ports, files, packages, services, Docker ...
score AS (
    SELECT process_hits.total + port_hits.total + file_hits.total + ...
    AS total
    FROM process_hits, port_hits, file_hits, ...
)
SELECT 1 AS passing
FROM score
WHERE total <= 2;
```

`COUNT(*)` always returns exactly one row even when there are no matches, so the cross-join between CTEs always produces a single row with the combined score. The `WHERE total <= 2` filter either keeps that row (passing) or discards it (failing).

## What each policy checks

There are three policies, one per platform:

| Detection vector | macOS | Linux | Windows |
|---|---|---|---|
| Running processes | ✓ | ✓ | ✓ |
| Listening ports (18789, 18793) | ✓ | ✓ | ✓ |
| Config directories and binaries | ✓ | ✓ | ✓ |
| npm packages | ✓ | ✓ | ✓ |
| Docker images and containers | ✓ | ✓ | ✓ |
| Homebrew packages | ✓ | | |
| launchd services | ✓ | | |
| Installed apps (bundle ID) | ✓ | | |
| systemd units | | ✓ | |
| deb / rpm packages | | ✓ | |
| Windows services | | | ✓ |
| Scheduled tasks | | | ✓ |
| Installed programs | | | ✓ |

## Investigation queries

When a device fails one of these policies, run the investigation queries in Fleet's live query mode to see what's installed.

Find running processes:

```sql
SELECT pid, name, path, cmdline, uid
FROM processes
WHERE name LIKE '%openclaw%'
    OR name LIKE '%clawdbot%'
    OR name LIKE '%moltbot%'
    OR name LIKE '%clawd%'
    OR cmdline LIKE '%openclaw%'
    OR cmdline LIKE '%clawdbot%'
    OR cmdline LIKE '%moltbot%';
```

See what's behind the known ports:

```sql
SELECT lp.port, lp.address, lp.protocol, p.name, p.path AS binary_path, p.cmdline
FROM listening_ports lp
LEFT JOIN processes p USING (pid)
WHERE lp.port IN (18789, 18793)
    OR p.name LIKE '%openclaw%'
    OR p.name LIKE '%clawdbot%'
    OR p.name LIKE '%moltbot%';
```

Check macOS `launchd` persistence with the exact labels from OpenClaw's documentation:

```sql
SELECT name, label, program, path, run_at_load, keep_alive
FROM launchd
WHERE label = 'ai.openclaw.gateway'
    OR label = 'com.clawdbot.gateway'
    OR label = 'bot.molt.gateway'
    OR label LIKE 'ai.openclaw.%'
    OR label LIKE 'com.clawdbot.%'
    OR label LIKE 'bot.molt.%'
    OR name LIKE '%openclaw%'
    OR name LIKE '%clawdbot%'
    OR name LIKE '%moltbot%'
    OR program LIKE '%openclaw%'
    OR program LIKE '%clawdbot%'
    OR program LIKE '%moltbot%';
```

Check Linux `systemd` persistence with the exact unit names:

```sql
SELECT id, description, load_state, active_state, sub_state, fragment_path
FROM systemd_units
WHERE id = 'openclaw-gateway.service'
    OR id LIKE 'openclaw-gateway-%.service'
    OR id LIKE '%openclaw%'
    OR id LIKE '%clawdbot%'
    OR id LIKE '%moltbot%';
```

The full set of eleven investigation queries covers processes, ports, Docker, launchd (macOS), installed apps and bundle identifiers (macOS), Homebrew packages (macOS), npm global packages, systemd (Linux), and Windows services and scheduled tasks. All use `logging: differential` so Fleet tracks when OpenClaw appears or disappears over time.

## Threat hunting

The investigation queries tell you what's installed. The threat hunting queries go further: they look for signs that an installation has been compromised or misconfigured. The YAML includes twelve threat hunting queries. Here's each one.

### Shell spawning

If OpenClaw (typically a Node.js process) spawns a shell like `bash`, `zsh`, or `sudo`, something is executing system commands. This could be a malicious skill or a prompt injection attack. This query walks the process tree:

```sql
SELECT p.pid, p.name, p.cmdline,
    pp.name AS parent_name, pp.cmdline AS parent_cmd
FROM processes p
JOIN processes pp ON p.parent = pp.pid
WHERE (pp.name = 'node'
    OR pp.cmdline LIKE '%openclaw%'
    OR pp.cmdline LIKE '%clawdbot%'
    OR pp.cmdline LIKE '%moltbot%')
AND p.name IN ('sh', 'zsh', 'bash', 'sudo', 'python3');
```

### Exposed network bindings

Many tutorials tell people to bind to `0.0.0.0` instead of `127.0.0.1`. If OpenClaw is listening on all interfaces, anyone on the network can reach it:

```sql
SELECT p.name, p.pid, p.cmdline, lp.address, lp.port, lp.protocol,
    CASE
        WHEN lp.address = '0.0.0.0' THEN 'EXPOSED'
        WHEN lp.address = '::' THEN 'EXPOSED (IPv6)'
        ELSE 'LOCAL'
    END AS exposure
FROM listening_ports lp
JOIN processes p ON lp.pid = p.pid
WHERE (p.name LIKE '%openclaw%'
    OR p.name LIKE '%clawdbot%'
    OR p.cmdline LIKE '%openclaw-gateway%')
AND lp.port NOT IN (22, 443, 80);
```

### Memory poisoning

OpenClaw stores persistent behavioural instructions in `SOUL.md` and conversation context in `MEMORY.md`. If an attacker modifies these files, the backdoor survives restarts. This query catches recent modifications:

```sql
SELECT path, filename, size,
    datetime(mtime, 'unixepoch') AS modified
FROM file
WHERE (path LIKE '/Users/%/.clawdbot/%/SOUL.md'
    OR path LIKE '/Users/%/.clawdbot/%/MEMORY.md'
    OR path LIKE '/Users/%/.openclaw/%/SOUL.md'
    OR path LIKE '/Users/%/.openclaw/%/MEMORY.md'
    OR path LIKE '/home/%/.clawdbot/%/SOUL.md'
    OR path LIKE '/home/%/.clawdbot/%/MEMORY.md'
    OR path LIKE '/home/%/.openclaw/%/SOUL.md'
    OR path LIKE '/home/%/.openclaw/%/MEMORY.md'
    OR path LIKE '/root/.clawdbot/%/SOUL.md'
    OR path LIKE '/root/.clawdbot/%/MEMORY.md'
    OR path LIKE '/root/.openclaw/%/SOUL.md'
    OR path LIKE '/root/.openclaw/%/MEMORY.md')
AND mtime > (strftime('%s', 'now') - 7200);
```

### Session transcript exposure

OpenClaw stores full conversation transcripts at `~/.openclaw/agents/<agentId>/sessions/*.jsonl`. These files persist on disk and are readable by any process with filesystem access:

```sql
SELECT path, filename, size,
    datetime(mtime, 'unixepoch') AS modified,
    datetime(ctime, 'unixepoch') AS created
FROM file
WHERE (path LIKE '/Users/%/.openclaw/agents/%/sessions/%.jsonl'
    OR path LIKE '/Users/%/.clawdbot/agents/%/sessions/%.jsonl'
    OR path LIKE '/home/%/.openclaw/agents/%/sessions/%.jsonl'
    OR path LIKE '/home/%/.clawdbot/agents/%/sessions/%.jsonl'
    OR path LIKE '/root/.openclaw/agents/%/sessions/%.jsonl'
    OR path LIKE '/root/.clawdbot/agents/%/sessions/%.jsonl');
```

### Credential and API key files

Plaintext API keys in `.env` and `credentials/*.json` are the primary target of the Credential Exfiltration campaign. This query finds them:

```sql
SELECT path, filename, size, mode,
    datetime(mtime, 'unixepoch') AS modified
FROM file
WHERE (path LIKE '/Users/%/.clawdbot/.env'
    OR path LIKE '/Users/%/.openclaw/.env'
    OR path LIKE '/Users/%/.openclaw/credentials/%.json'
    OR path LIKE '/Users/%/.clawdbot/credentials/%.json'
    OR path LIKE '/home/%/.clawdbot/.env'
    OR path LIKE '/home/%/.openclaw/.env'
    OR path LIKE '/home/%/.openclaw/credentials/%.json'
    OR path LIKE '/home/%/.clawdbot/credentials/%.json'
    OR path LIKE '/root/.clawdbot/.env'
    OR path LIKE '/root/.openclaw/.env'
    OR path LIKE '/root/.openclaw/credentials/%.json'
    OR path LIKE '/root/.clawdbot/credentials/%.json');
```

### Config file permissions

OpenClaw's documentation recommends `600` permissions for `openclaw.json` and all credential files. This query finds world-readable ones:

```sql
SELECT path, mode, uid, gid, size,
    datetime(mtime, 'unixepoch') AS modified
FROM file
WHERE (path LIKE '/Users/%/.clawdbot/%'
    OR path LIKE '/Users/%/.openclaw/%'
    OR path LIKE '/home/%/.clawdbot/%'
    OR path LIKE '/home/%/.openclaw/%'
    OR path LIKE '/root/.clawdbot/%'
    OR path LIKE '/root/.openclaw/%')
AND (mode LIKE '%7' OR mode LIKE '%5' OR mode LIKE '%4')
AND (filename LIKE '%.env%'
    OR filename LIKE '%credentials%'
    OR filename LIKE '%gateway.yaml%'
    OR filename LIKE '%openclaw.json%');
```

### Gatekeeper quarantine bypass (macOS)

ClawHavoc uses `xattr -c` to strip macOS quarantine attributes from the AMOS payload before execution:

```sql
SELECT p.pid, p.name, p.cmdline, u.username
FROM processes p
LEFT JOIN users u ON p.uid = u.uid
WHERE p.cmdline LIKE '%xattr -c%'
    OR p.cmdline LIKE '%xattr -d com.apple.quarantine%';
```

### Reverse shell indicators

AuthTool deploys a reverse shell triggered by natural language interaction with a malicious skill:

```sql
SELECT p.pid, p.name, p.cmdline, u.username,
    pp.name AS parent_name, pp.cmdline AS parent_cmd
FROM processes p
LEFT JOIN users u ON p.uid = u.uid
LEFT JOIN processes pp ON p.parent = pp.pid
WHERE p.cmdline LIKE '%/dev/tcp/%'
    OR p.cmdline LIKE '%nohup%bash%-i%'
    OR p.cmdline LIKE '%bash -i >%';
```

### Supply chain delivery

Two queries cover the main delivery mechanisms. The first catches `npx -y`, which runs packages without prompting:

```sql
SELECT p.pid, p.name, p.cmdline, u.username
FROM processes p
LEFT JOIN users u ON p.uid = u.uid
WHERE p.name = 'npx'
    OR p.cmdline LIKE '%npx -y%';
```

The second catches `curl` or `wget` piped to a shell. It also matches the official install script at `clawd.bot/install-cli.sh`:

```sql
SELECT p.pid, p.name, p.cmdline, u.username
FROM processes p
LEFT JOIN users u ON p.uid = u.uid
WHERE p.cmdline LIKE '%curl%|%bash%'
    OR p.cmdline LIKE '%curl%|%sh%'
    OR p.cmdline LIKE '%wget%|%bash%'
    OR p.cmdline LIKE '%wget%|%sh%'
    OR p.cmdline LIKE '%clawd.bot/install%';
```

### mDNS auto-discovery

OpenClaw gateways broadcast `_openclaw-gw._tcp` via mDNS on UDP 5353. TXT records can expose filesystem paths, hostnames, and SSH availability. Attackers on the same LAN can use this for reconnaissance:

```sql
SELECT p.name, p.pid, u.username,
    po.local_address, po.local_port,
    po.remote_address, po.remote_port
FROM process_open_sockets po
JOIN processes p ON po.pid = p.pid
LEFT JOIN users u ON p.uid = u.uid
WHERE po.remote_port = 5353
    AND po.protocol = 17;
```

### C2 indicators

This query catches connections to known infrastructure from the ClawHavoc campaign (Glot.io / 91.92.242.30), AuthTool reverse shell (54.91.154.110), and AMOS exfiltration (socifiapp.com):

```sql
SELECT p.pid, p.name, p.cmdline, u.username,
    po.remote_address, po.remote_port
FROM processes p
LEFT JOIN users u ON p.uid = u.uid
LEFT JOIN process_open_sockets po ON p.pid = po.pid
WHERE p.cmdline LIKE '%glot.io%'
    OR p.cmdline LIKE '%socifiapp%'
    OR po.remote_address = '91.92.242.30'
    OR po.remote_address = '54.91.154.110';
```

## Automate the response

Fleet can send a [webhook](https://fleetdm.com/docs/using-fleet/automations#policy-automations) every time a device fails one of these policies. Point it at your ticketing system or a Slack channel so the security team knows the moment OpenClaw shows up on a new device.

If your organisation's policy is to harden rather than remove OpenClaw, the project includes its own audit command. Running `openclaw security audit --deep` produces a detailed report covering exposed credentials, insecure file permissions, missing gateway authentication, and network binding issues. The `--fix` flag tightens defaults and fixes permissions automatically. You can script this as a Fleet remediation step alongside the webhook.

## Deploy with GitOps

Download the two YAML files and place them in your Fleet GitOps repository's `lib/` folder:

- [`openclaw-detection.policies.yml`](https://github.com/AdamBaali/fleet-gitops/blob/main/lib/all/policies/openclaw-detection.policies.yml) contains three platform-specific detection policies (macOS, Linux, Windows)
- [`openclaw-detection.queries.yml`](https://github.com/AdamBaali/fleet-gitops/blob/main/lib/all/queries/openclaw-detection.queries.yml) contains eleven investigation queries and twelve threat hunting queries with differential logging

Reference them from your team YAML:

```yaml
policies:
  - path: ../lib/openclaw-detection.policies.yml
queries:
  - path: ../lib/openclaw-detection.queries.yml
```

Or apply directly:

```bash
fleetctl apply -f openclaw-detection.policies.yml
fleetctl apply -f openclaw-detection.queries.yml
```

osquery tables like `docker_images`, `docker_containers`, and `npm_packages` return zero rows when Docker or Node.js isn't installed rather than erroring. The CTEs that check those tables simply contribute 0 to the score. On Linux, `deb_packages` returns nothing on RPM-based distros and vice versa; both are included so the policy works across all distributions. Test against a canary team first to confirm behaviour in your environment.

Personal AI assistants are here to stay. The goal in a managed environment is to understand what's running to make informed decisions on governance and use. Get in touch with [Fleet](https://fleetdm.com/) to learn more about how our modern, powerful device management and data observation capabilities can help any organization navigate the challenges IT faces today.

<meta name="articleTitle" value="Mitigation assets and detection patterns for AI agents like OpenClaw">
<meta name="authorFullName" value="Adam Baali">
<meta name="authorGitHubUsername" value="adambaali">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-02-19">
<meta name="description" value="Part 3 of 3 - OpenClaw: Mitigations and patterns to help investigate autonomous AI agent use on managed endpoints.">
