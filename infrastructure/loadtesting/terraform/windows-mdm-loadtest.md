## Windows MDM load test plan

This document describes how to load test Fleet's Windows MDM at scale, with a target of 100k enrolled Windows devices. It complements the general load test runbook in [readme.md](./readme.md), which covers spinning up the terraform environment, running migrations, and adding `osquery-perf` containers. Read that first. This document focuses on the Windows-MDM-specific scenarios, what to measure, and a known tooling gap.

### What we are validating

The cost of Windows MDM at scale is driven by a few axes, not by host count alone. The scenarios below each stress one of them.

- Polling model. fleetd-capable hosts are relaxed to an 8h poll (`windowsMDMRelaxedPollIntervalMinutes = "480"`) plus on-demand wake, while pure Windows MDM hosts stay on the 1-minute fast poll (`windowsMDMFastPollIntervalMinutes = "1"`, `server/service/microsoft_mdm.go`). At 100k the difference is roughly 3-4 management sessions/sec versus ~1,670/sec. This dominates everything else, so the mix of fleetd vs pure-MDM hosts is the first thing to pin down.
- Host x profile fan-out. Each assigned profile creates one row per host in `windows_mdm_command_queue` and one in `host_mdm_windows_profiles`. 100 profiles x 100k hosts is ~10M rows in each table on a full re-push.
- Per-session SyncML payload size. Every management session reports status for all assigned settings and verification re-checks them, so the `windows_mdm_responses` payload grows with settings-per-host. This is what the response compression work targets.
- Reconcile cron throughput. `ReconcileWindowsProfiles` runs every 30s, processes ~2000 hosts/tick (delivery cap) with a 24s scan budget and a single Redis cursor, giving a full-sweep time of roughly 25 minutes at 100k.
- Disk encryption escrow and verification. Enabling encryption produces a burst of recovery-key writes into `host_disk_encryption_keys`, and the `verify_disk_encryption_keys` cron does CMS decryption per key (CPU bound).

### Prerequisites

- A terraform load test environment per [readme.md](./readme.md), sized for 100k hosts (see the reference architecture: roughly `db.r6g.2xlarge`, `cache.r6g.large`, and the corresponding fleet container count).
- A branch deployed to the environment (`$BRANCH_NAME`).
- Windows MDM enabled and configured on the server. Unlike Apple MDM, this needs the WSTEP identity configured so the discovery, enrollment, and management endpoints are live. Confirm `appConfig.MDM.WindowsEnabledAndConfigured` is true and `EnableTurnOnWindowsMDMManually` is false (the orbit config endpoint only returns `NeedsProgrammaticWindowsMDMEnrollment` when both hold, see `server/service/orbit.go`).
- `osquery-perf` containers launched with Windows templates and MDM enabled: `-os_templates windows_11` (or `windows_11_22H2_*`), `-mdm_prob 1.0`, `-orbit_prob 1.0`. `-mdm_prob` is what gives each simulated host a Windows MDM client; the default is 0.0 and without it no host will ever enroll.

### Tooling support and known gaps

What `osquery-perf` simulates today (`cmd/osquery-perf/agent.go`):

- Windows MDM enrollment via the orbit notification path (`runOrbitLoop` reacts to `NeedsProgrammaticWindowsMDMEnrollment` and calls `winMDMClient.Enroll()`).
- SyncML management sessions (`runWindowsMDMLoop` / `doWindowsMDMCheckIn`), including acking commands, honoring a server `Replace` on the DMClient poll node to adjust the poll interval, and on-demand wake.
- BitLocker on/off status reporting through the osquery `disk_encryption_windows` detail query (`diskEncryptionWindows()`), which sets the host's encryption status.
- Profile command failure injection via `mdmProfileFailureProb`.

Known gap: disk encryption key escrow is NOT simulated. `osquery-perf` does not handle the `RotateDiskEncryptionKey` orbit notification and never POSTs to `/api/fleet/orbit/disk_encryption_key`. The real agent does this in `orbit/pkg/update/notifications.go` (`SetOrUpdateDiskEncryptionKey`). As a result `host_disk_encryption_keys` stays empty under load, so neither the escrow write burst nor the `verify_disk_encryption_keys` cron (`cmd/fleet/cron.go`) is exercised. To load test disk encryption, `osquery-perf` must be extended to read `cfg.Notifications.RotateDiskEncryptionKey` in the orbit loop and POST a recovery key. For the verify cron to do representative crypto work the key should be CMS-encrypted against the server's WSTEP/SCEP certificate; an arbitrary blob still exercises the decrypt-attempt CPU cost but every key will be marked `decryptable = false`.

Other timing facts that shape the tests, both currently hardcoded in `osquery-perf`: the orbit config poll runs on a 30s ticker, and per-host enrollment attempts are throttled to once per hour (`windowsMDMEnrollmentAttemptFrequency`). Agent startup is staggered over `-start_period` (default 10s).

### Scenarios

Run these in order. Capture the metrics listed under "Metrics to capture" for each.

1. Steady-state check-in. Bring up the full 100k host fleet and let it settle. With fleetd-capable hosts this is the relaxed-poll baseline. To measure the worst case, also run a variant where hosts stay on the fast poll. Pass criteria: MySQL writer CPU and IOPS stay within headroom, replication lag stays low, and management session p95 latency is acceptable.

2. Profile batch fan-out. Apply a batch of 40 CSP profiles, then separately a batch of 100, to all hosts. Test add, modify (Replace), and delete as distinct steps; delete has its own batching path. Measure the write burst into `windows_mdm_command_queue` and `host_mdm_windows_profiles`, and the time for the change to fan out to all hosts through the reconcile cron. Pass criteria: fan-out completes within the product's acceptable window and the writer is not saturated during the burst.

3. Reconcile cron throughput. While the profile batch from scenario 2 is fanning out, watch the reconcile cron tick duration against its 30s interval and 24s scan budget. Pass criteria: ticks do not consistently overrun, and the single-cursor sweep advances steadily without re-scanning.

4. Enrollment storm. Launch 100k Windows hosts with `-mdm_prob 1.0` while Windows MDM is disabled (so they sit unenrolled), then enable Windows MDM on the server. On the next orbit poll the hosts self-enroll in a wave bounded by the 30s poll window, roughly 3,300-10,000 enrollments/sec depending on `-start_period`. Each enrollment also triggers an immediate `ReconcileWindowsProfilesForEnrollingHost` that forces a primary read, so this hits both the enroll path and the primary DB. Pass criteria: the enroll endpoint and `storeWindowsMDMEnrolledDevice` writes survive the spike without errors or runaway latency. For a gradual onboarding comparison, instead enable MDM first and launch hosts with a long `-start_period`.

5. Failure injection and resend. Run a profile batch with `mdmProfileFailureProb` set so a fraction of Add commands fail. Failed Adds are rewritten to Replace with a new command UUID and re-queued (`handleResendingAlreadyExistsCommands`), which amplifies queue volume. Pass criteria: the resend path does not multiply queue rows unboundedly and converges.

6. Disk encryption (blocked on the tooling gap above). Once `osquery-perf` can escrow keys, enable disk encryption fleet-wide and measure the burst of writes into `host_disk_encryption_keys` and the `verify_disk_encryption_keys` cron throughput against 100k unverified keys. Pass criteria: the cron drains the backlog within an acceptable window; note there is no per-host retry, so unverified keys persist if it cannot keep up.

### Metrics to capture

- MySQL writer CPU, IOPS, and replication lag (RDS Performance Insights). This is the primary saturation signal, consistent with the prior 20k-host incident where the writer was the bottleneck.
- Management session request rate and p50/p95 latency on `POST /api/mdm/management`.
- Reconcile cron tick duration versus the 30s interval, and cursor progress.
- Row counts and growth of `windows_mdm_command_queue`, `windows_mdm_command_results`, `windows_mdm_responses`, and `host_mdm_windows_profiles`.
- `windows_mdm_responses` size and compression ratio, to quantify the payload reduction at high profile counts.
- Redis load (ElastiCache metrics).

See [readme.md](./readme.md) for how to reach the APM dashboard and the database and Redis monitoring consoles.

### Summary of steps

1. Stand up the terraform environment sized for 100k hosts and run migrations (see [readme.md](./readme.md)).
2. Enable and configure Windows MDM on the server (WSTEP identity, `WindowsEnabledAndConfigured`).
3. Decide and configure the fleetd vs pure-MDM host mix, since it sets the steady-state load.
4. Launch `osquery-perf` Windows containers with `-os_templates windows_11 -mdm_prob 1.0 -orbit_prob 1.0`.
5. Run scenario 1 (steady-state) and capture baseline metrics.
6. Run scenario 2 (40 then 100 profiles: add, modify, delete) and measure fan-out and the write burst.
7. Run scenario 3 (reconcile throughput) concurrently with the batch fan-out.
8. Run scenario 4 (enrollment storm), plus the gradual-onboarding comparison.
9. Run scenario 5 (failure injection and resend).
10. Extend `osquery-perf` to escrow keys, then run scenario 6 (disk encryption escrow and verify cron). Until then, mark disk encryption as not covered.
11. Record all metrics and compare against the pass criteria for each scenario.
