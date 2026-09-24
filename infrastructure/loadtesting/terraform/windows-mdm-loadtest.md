## Windows MDM load test plan

This document describes how to load test Fleet's Windows MDM at scale, with a target of 100k enrolled Windows devices. It complements the general load test runbook in [readme.md](./readme.md), which covers spinning up the terraform environment, running migrations, and adding `osquery-perf` containers. Read that first. This document focuses on the Windows-MDM-specific scenarios, what to measure, and a known tooling gap.

### What we are validating

The cost of Windows MDM at scale is driven by a few axes, not by host count alone. The scenarios below each stress one of them.

- Profiles multiplied across hosts. Each assigned profile creates one row per host in `windows_mdm_command_queue` and one in `host_mdm_windows_profiles`. 100 profiles x 100k hosts is ~10M rows in each table on a full re-push. Note the row count is per profile, not per LocURI: a single profile becomes one command row per host regardless of how many settings it carries.
- Per-session SyncML payload size. Every management session reports status for all assigned settings and verification re-checks them. This scales with the number of LocURIs per profile, not just the profile count, so profile realism matters (see "Profile realism" below).
- Team transfers. Moving hosts between two teams that each carry a full profile set forces a per-host remove of the source team's profiles and an add of the destination team's, which is the heaviest profile-churn operation per host.
- Reconcile cron throughput. `ReconcileWindowsProfiles` runs every 30s, processes ~2000 hosts/tick (delivery cap) with a 24s scan budget and a single Redis cursor, giving a full-sweep time of roughly 25 minutes at 100k.
- Disk encryption escrow. Enabling encryption sends the `EnforceBitLockerEncryption` orbit notification to all hosts and produces a one-time burst of recovery-key writes into `host_disk_encryption_keys` (plus archive and activity rows). This axis is currently NOT testable.

### Prerequisites

- A terraform load test environment per [readme.md](./readme.md), sized for 100k hosts (see the reference architecture).
  - Use 150k sizing due to known issues, such as the [gorilla/mux route-matching CPU bottleneck](https://github.com/fleetdm/fleet/issues/48326).
- `osquery-perf` containers launched with Windows templates and MDM enabled:
```
    "--orbit_prob", "1.0",
    "--mdm_prob", "1.0",
    "--os_templates", "windows_11,windows_11_22H2_2861,windows_11_22H2_3007",
    "--logger_tls_period", "120s",
    "--http_message_signature_prob", "0",
    "--start_period", "60m"
```

### Profile realism

The load profiles must mirror real CSP profiles, not 100 trivial single-setting profiles, or the test understates per-session payload and verification cost. A Windows MDM profile is a SyncML payload that can carry many settings, each an `<Add>`/`<Replace>` with its own `<LocURI>`. The stored `raw_command` XML and the per-session status response both scale with the LocURI count, and non-atomic profiles are verified per top-level command (verification work is proportional to LocURI count), while atomic profiles report a single aggregate status. The command-queue row count, by contrast, is per profile per host regardless of LocURI count.

Use the real profiles under `it-and-security/lib/windows/configuration-profiles/` as the distribution to match. They range from 1 LocURI (for example "Advanced PowerShell logging", "Disable Guest account") up to 5-6 (for example "Windows Defender compliance settings" at 5, "Enable firewall" at 6), with several in the 2-3 range ("Password settings"). When building the 100-profile batch for scenario 3, weight the mix so a meaningful share have multiple LocURIs rather than making them all single-setting, so the per-session payload and verification path are exercised the way production hits them.

### Pages to check

The scenarios below drive changes into Fleet, but the slowness a customer would notice shows up on two pages that display profile status. Load each page and record its response time: once after profiles have finished applying to all 100k hosts, and again while profiles are being applied and deleted (scenarios 3 and 5), when the database is busiest.

- OS settings summary (Controls > OS settings): `GET /api/latest/fleet/configuration_profiles/summary`. Its response time should stay about the same as the number of hosts grows.
- Hosts page filtered by an OS settings status such as Pending: `GET /api/latest/fleet/hosts` and `/hosts/count` with `os_settings=pending`. This is the page most likely to slow down as the fleet grows, so watch its response time closely.

### Scenarios

Run these in order.

1. (Optional) Enrollment storm. Run this on 20k hosts, not the full fleet, and run it first during bring-up. Deploy the first 20k Windows hosts while Windows MDM is turned off in the UI, so they sit unenrolled. Then enable Windows MDM from the UI (this flips `WindowsEnabledAndConfigured`). Note that this is unlikely to happen in production. If a customer is enrolling all their Windows hosts at once, they would need to scale their deployment to absorb the spike in load.

2. Ramp to 100k and steady-state baseline.

3. Profile batch apply (add) and delivery throughput. Apply a single batch of 100 CSP profiles to all 100k hosts at once.

4. Profile batch apply (replace). Replace the 100 profiles with 100 differently named profiles in one batch. This exercises the modify/Replace path at scale. Then modify the 100 profiles in one batch.

5. Profile deletion. Delete the 100 profiles.

6. Team transfers. Move enrolled hosts from one team to another where both teams carry a full ~100-profile set. IMPORTANT: cap each transfer at 30K hosts. Larger single transfers are blocked by [#46894](https://github.com/fleetdm/fleet/issues/46894).

7. Disk encryption storm. NOT TESTABLE at this time, tracked by [#48322](https://github.com/fleetdm/fleet/issues/48322).
