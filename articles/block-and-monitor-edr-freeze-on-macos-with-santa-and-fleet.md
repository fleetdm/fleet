# Block and monitor EDR Freeze on macOS with Santa and Fleet

*EDR Freeze suspends a security tool instead of killing it, so the process looks healthy while it quietly stops working. Santa 2026.3 can block it on macOS, and Fleet lets you ship the fix and watch for the attack across every host.*

## Key takeaways

- **EDR Freeze is for staying hidden, not breaking in.** An attacker who already has root uses `pid_suspend` to freeze a security agent at the Mach task level. `ps` and Activity Monitor still show a healthy process, but it can't authorize, alert, or record anything.
- **Santa defends itself by default, and 2026.3 defends the rest of your stack.** List your other agents' code-signing identities under `AntiSuspendSigningIDs` and Santa denies the suspension before it ever takes effect.
- **The protection ships as a reviewed config change, not new infrastructure.** The key goes into a `.mobileconfig` profile in Git, deploys through Fleet GitOps and MDM, and rolls back with a `git revert`. No Santa sync server required.
- **Fleet watches every host for the conditions the attack relies on.** The `santa_status`, `santa_denied`, and `santa_allowed` tables catch agents that aren't answering, hosts in the wrong mode, missing rules, and denial gaps.
- **Santa's telemetry catches the suspend attempt itself.** Turn on `proc_suspend_resume` telemetry and forward it to your SIEM to alert the moment anything tries to freeze a protected process.
- **Independent signals cover what a frozen agent can't report.** Sequence-number gaps, server-side telemetry silence, deadline-kill log messages, and canary events all fire even when the agent is asleep.

<a purpose="cta-button" href="https://fleetdm.com/articles/deploy-santa-with-fleet-gitops-and-skip-the-sync-server">Deploy Santa with Fleet</a>

EDR Freeze started life on Windows — pause a security product so it stops alerting or responding, without crashing or uninstalling it. The same idea works on macOS: anything built on Apple's Endpoint Security framework can be frozen with the `pid_suspend` system call, and a suspended agent still looks healthy in every monitoring tool while an attacker slips actions past its checks or floods its event queue.

The good news is that the fix is well understood, and you can ship and monitor it through Fleet today. Santa, the open-source binary authorization agent for macOS, added the `AntiSuspendSigningIDs` configuration key in version 2026.3, and Fleet gives you the delivery pipeline and the visibility to go with it. Here's how the attack works and how to close it.

## How EDR Freeze works on macOS

`pid_suspend` is a long-standing macOS system call. It has been around since Snow Leopard 10.6 and predates the Endpoint Security framework, so it isn't going anywhere. It works at the Mach task level, beneath the BSD process layer that most software deals with. Called against a process, the kernel freezes every thread in the underlying task and pauses scheduling until something calls `pid_resume`.

A suspended security agent can't pull events off its queue or respond to them, because its threads aren't running. Yet `ps` and Activity Monitor still list the process as present, so from the outside everything looks normal.

One prerequisite is worth stating plainly — an attacker needs root to suspend a system extension that runs as root. This is a post-exploitation technique, not a way in. But if someone already has root and your Endpoint Security agent doesn't defend itself against suspension, the consequences are real.

Once a tool is frozen, there are two ways to take advantage of it.

### Authorization bypass

Endpoint Security clients can subscribe to authorization (AUTH) events, where the kernel holds an action until the client returns allow or deny. A suspended client can't respond. macOS gives each client a deadline, and when the suspended client misses it, the OS kills the client to avoid a deadlock — and the action goes through. So the attacker suspends the agent, performs the blocked action, waits out the deadline, and lets the OS finish the job.

Santa closed its exposure to this in version 2025.12.

### Detection bypass

Notification (NOTIFY) events reach clients through a per-client queue with a historical default size of 3,072 events. Once it's full, new events are dropped silently and the client never sees them. An attacker who can suspend a NOTIFY-only client floods the queue with harmless activity, carries out whatever they want to hide while the queue is saturated, then resumes the client. It wakes up, drains a queue full of noise, and carries on. No alerts, no obvious gap.

## Why Fleet makes Santa simple to run

Traditionally, Santa needs a dedicated sync server to distribute rules, collect events, and manage configuration. That's one more piece of infrastructure to stand up, secure, and maintain.

Fleet replaces it with tools you already use. You define Santa's configuration as code in `.mobileconfig` profiles in Git, Fleet and Apple MDM push them to your macOS hosts, and Fleet's agent collects Santa's events. Rule and configuration changes go through pull request review, deploy on their own, and roll back with a `git revert`. No sync server required.

If you haven't deployed Santa with Fleet yet, start with the two-part series and come back:

- [Part 1: Deploy Santa with Fleet GitOps and skip the sync server](https://fleetdm.com/articles/deploy-santa-with-fleet-gitops-and-skip-the-sync-server)
- [Part 2: How we deployed Santa at Fleet](https://fleetdm.com/articles/how-we-deployed-santa-at-fleet)

The series covers the full setup — deploying the Santa app, splitting app config and rules into separate profiles, and collecting denied-binary logs. EDR Freeze protection slots straight into that same model.

## Block it with Santa 2026.3

Santa already protects itself. It subscribes to the `ES_EVENT_TYPE_AUTH_PROC_SUSPEND_RESUME` authorization event, which macOS has supported since version 11, and denies any attempt to suspend its own process. Because this is an AUTH event, the denial lands before the suspension takes effect, so Santa is never frozen in the first place.

Version 2026.3 extends that same protection to other processes through the `AntiSuspendSigningIDs` configuration key. You list the code-signing identities you want to protect, and Santa denies any `pid_suspend` call that targets them. In effect, you put Santa's self-defense in front of the rest of your security stack — another EDR agent, a telemetry collector, or any process that would leave a blind spot if it were frozen.

Add the key to your Santa configuration profile (the app-config profile from the deployment series, not the rules profile):

```xml
<key>AntiSuspendSigningIDs</key>
<array>
  <!-- Protect your EDR agent -->
  <string>EXAMPLE1234:com.example.edr-agent</string>
  <!-- Protect your telemetry log collector -->
  <string>EXAMPLE5678:com.example.telemetry-collector</string>
</array>
```

Each entry is `TeamID:SigningID`. Get the real values for a binary by running:

```bash
codesign -dvvv /path/to/the/binary 2>&1 | grep -E 'TeamIdentifier|Identifier'
```

Use `TeamIdentifier` for the team ID and the `Identifier` field for the signing ID. The values above are placeholders, so swap in the IDs for the tools you actually run. You don't need to add Santa itself — its self-protection is built in.

While you're in the profile, make sure Santa is emitting the suspend/resume telemetry you'll use for alerting:

```xml
<key>Telemetry</key>
<array>
  <string>Execution</string>
  <string>proc_suspend_resume</string>
</array>
```

Ship the edit the way you ship any Santa change — open a pull request, get it reviewed, merge, and let Fleet GitOps apply the updated profile through MDM. Roll it out to a test group first, then the rest of your fleet. If anything misbehaves, revert the commit and Fleet redeploys the previous profile.

## Monitor the attack with Fleet's Santa tables

Blocking the attack is only half the job. You also want to know when something tries, and to confirm your agents stay healthy and enforcing. Fleet's agent ships three Santa tables, so you can monitor every macOS host from one place — `santa_status`, `santa_denied`, and `santa_allowed`. Turn them into scheduled reports and policies and you've got continuous coverage of the conditions EDR Freeze relies on.

### Confirm Santa is alive and healthy: santa_status

The authorization-bypass variant ends with macOS killing Santa when it misses its deadline. Santa restarts, but that window is exactly what you want to catch — along with any agent that's unhealthy or running in the wrong mode.

The `santa_status` table mirrors `santactl status` for every host:

```sql
SELECT mode, events_pending_upload, watchdog_cpu_events, watchdog_ram_events
FROM santa_status
```

Two things to watch for:
- A host that should be running Santa but returns no rows. If `santa_status` comes back empty where you expect a result, Santa isn't answering. Go look at that host.
- A mode that isn't what your profile sets — for example, a host in Monitor when your fleet should be in Lockdown. Make it a Fleet policy so you're alerted automatically.

The policy passes only when Santa is in the expected mode:

```sql
SELECT 1 FROM santa_status WHERE mode = 'Lockdown'
```

Any host that fails it is either not running Santa or not enforcing.

`santa_status` also reports how many rules are loaded, which is a fast way to confirm your controls are actually present. Since EDR Freeze is used to slip past a control, check that the rule counts match what your committed profiles should produce:

```sql
SELECT binary_rules, signingid_rules, teamid_rules, static_rule_count
FROM santa_status
```

A host with fewer rules than expected is a gap worth closing, whether or not it came from tampering.

### Watch execution decisions: santa_denied and santa_allowed

These tables log what Santa blocked and allowed. The deployment series already collects denied logs to a SIEM, and the same data is queryable here:

```sql
SELECT application, reason, sha256, timestamp
FROM santa_denied
ORDER BY timestamp DESC
```

The whole point of the bypass is to run something that should have been denied. So if a binary you expect Santa to block stops showing up in `santa_denied` on one host while it's still denied everywhere else, correlate that host's `santa_status` and suspend telemetry.

Fleet keeps the most recent 10,000 allowed and denied events per host — for scheduled reports, set `differential_ignore_removals` to stay within the agent's watchdog limits.

### Catch the suspend attempt itself: Santa telemetry

The tables above cover Santa's state and its execution decisions. They don't record `pid_suspend` calls. The suspend attempt itself is captured by Santa's `proc_suspend_resume` telemetry, which you turned on earlier in the profile.

Collect Santa's event log the same way the deployment series collects denied logs, forward it to your SIEM, and alert on any suspend that targets one of your protected processes or any Endpoint Security client.

For a quick check on a single host, `sudo eslogger proc_suspend_resume` shows each call live, including the `teamid` and `signingid` of both the instigator and the target.

## Additional detection signals

The Fleet tables and telemetry above are the core of your coverage. These independent signals don't depend on the agent staying awake, and they're worth wiring into your SIEM too:

- **Sequence-number gaps.** Endpoint Security messages carry monotonic sequence numbers. A gap larger than one means events were dropped, which is the signature of the queue-exhaustion bypass.
- **Server-side telemetry gaps.** A host that was reporting steadily and then goes quiet is a strong signal on its own. Fleet's host vitals and query schedules make those gaps visible.
- **Deadline kills.** When the OS terminates a client for missing an AUTH deadline, it logs it. Alert on the message `EndpointSecurity client terminated because it failed to respond to a message before its deadline`.
- **Canary events.** Periodically perform a known, observable action and confirm your tools report it. A missing canary suggests a tool is frozen or its queue is saturated.

## Verify a tool is actually protected

To check whether any Endpoint Security tool defends itself against suspension, run `sysdiagnose` on a host where it's installed and open `logs/EndpointSecurity/EndpointSecurity.log`.

This log records the events each connected client subscribes to. Search for event number 92, the value of `ES_EVENT_TYPE_AUTH_PROC_SUSPEND_RESUME`. If it's in the client's subscription list, the tool is at least subscribing to the authorization event it needs to deny suspension. If it isn't there, the tool isn't protecting itself, and the bypasses above may apply.

On a test host, you can also confirm Santa's protection end to end by writing a small tool that calls `pid_suspend` against a protected process and checking that the call is denied. Keep that testing to non-production machines.

## The bottom line

EDR Freeze proved that suspending a security process is a practical evasion technique, and the same principle carries over to macOS through `pid_suspend`. Santa's self-protection has been possible since macOS 11, and version 2026.3 lets you extend it to the rest of your stack with `AntiSuspendSigningIDs`.

With Fleet, blocking and monitoring the attack is a reviewed, version-controlled change to a profile you already manage, backed by tables you can query across every host. No sync server, no extra infrastructure, and a clear audit trail for every change.

## See it live

The fastest path is to work through the [Santa deployment series](https://fleetdm.com/articles/deploy-santa-with-fleet-gitops-and-skip-the-sync-server) and add the `AntiSuspendSigningIDs` key to your app-config profile. If you'd like a hand getting there, two good next steps:

- [**Get a demo**](https://fleetdm.com/contact)**.** We'll walk through blocking and monitoring EDR Freeze in your environment.
- [**Join a GitOps training session**](https://fleetdm.com/gitops-workshop)**.** Managing Santa's configuration as code is exactly what our hands-on workshop covers: profiles in Git, reviewed in pull requests, deployed through CI.

## Resources

- [Deploy Santa with Fleet GitOps and skip the sync server](https://fleetdm.com/articles/deploy-santa-with-fleet-gitops-and-skip-the-sync-server)
- [How we deployed Santa at Fleet](https://fleetdm.com/articles/how-we-deployed-santa-at-fleet)
- [santa_status table](https://fleetdm.com/tables/santa_status)
- [santa_denied table](https://fleetdm.com/tables/santa_denied)
- [santa_allowed table](https://fleetdm.com/tables/santa_allowed)
- [Santa on GitHub](https://github.com/northpolesec/santa)

<meta name="articleTitle" value="Block and monitor EDR Freeze on macOS with Santa and Fleet">
<meta name="authorFullName" value="Dhruv Majumdar">
<meta name="authorGitHubUsername" value="karmine05">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-06-29">
<meta name="description" value="Learn how EDR Freeze works on macOS, how Santa 2026.3 blocks it with AntiSuspendSigningIDs, and how to monitor for it with Fleet's Santa tables.">
