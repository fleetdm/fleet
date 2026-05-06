# Manual test plan — extend-scep-cert-matcher (issue #44111)

QA: run the three scenarios below in order, on the same host. Each takes ~10 min.

The bug only reproduces under specific timing (replica lag, ingest race), so each scenario uses a single SQL statement to put the system into the post-bug state a real customer could end up in. Verification is then driven through the Fleet UI — no further SQL needed unless something looks off.

## Prerequisites

1. Dev Fleet on branch `44111-scep-autorenew-fix`, Custom SCEP CA configured.
2. **One Windows or macOS host** enrolled and reporting (fleetd healthy). Either platform exercises the same matcher code path — pick whichever is easier to set up.
3. Assign to the host a configuration profile that uses `$FLEET_VAR_SCEP_RENEWAL_ID`.
4. In the Fleet UI on **Host details → Configuration profiles**, wait until the SCEP profile shows status **Verified**. On **Host details → Certificates**, confirm a cert with subject starting `fleet-…` is listed.
5. Note the host's UUID (URL of the host details page) and the profile UUID (URL when editing the profile). Save as `$HUUID` and `$PUUID`.

---

## Scenario A — Missed-link recovery (the bug fix)

**Customer scenario being mimicked:** the host's SCEP cert renewed normally and is sitting in the device keychain. But Fleet's matcher missed linking it back to the profile metadata (replica lag, ingest race, or the cert landed in the "already exists" branch instead of "to insert"). Result: hmmc row stays NULL, the renewal cron's `HAVING validity_period IS NOT NULL` clause excludes it forever, the cert silently never re-renews. This is what customer-shackleton hit.

### Setup (one statement, mimics the stuck-NULL state past the in-flight grace)

```sql
UPDATE host_mdm_managed_certificates
   SET not_valid_before = NULL, not_valid_after = NULL, serial = NULL,
       updated_at = NOW() - INTERVAL 5 HOUR
 WHERE host_uuid = '$HUUID' AND profile_uuid = '$PUUID';
```

### Run

1. On the host, restart fleetd / Fleet Desktop. (This forces a fresh cert refetch cycle.)
2. Wait 2 minutes.
3. In the Fleet UI: refresh **Host details → Certificates**. The SCEP cert is still listed.
4. **Host details → Configuration profiles**: SCEP profile still shows **Verified**.
5. Run `fleetctl trigger --name cleanups_then_aggregation` to fire the renewal cron.
6. Refresh **Host details → Activity**.

### Pass if

- No new MDM push activity for the SCEP profile appears (the row got recovered, the cron sees it's not in the renewal window, no action needed).
- Profile status stays **Verified** the whole time.

### Fail if

- The profile status flips to **Pending** / **Verifying** and a new push appears in the activity feed → recovery did not happen, the cron found the row stuck and *something* re-pushed it. Investigate before merging.

---

## Scenario B — Recovery still fires when the cert inventory is stable

**Customer scenario being mimicked:** same stuck-NULL state as A, but the host's cert inventory hasn't changed since the bug occurred — no new certs landing, no certs being removed. Pre-PR, the matcher was gated on `len(toInsert) > 0` so this host could never recover. The customer would have to manually resend the profile.

### Setup (same one statement as A — same customer state)

If you just ran A successfully, the row is repopulated. Re-apply the setup SQL from A to put it back into the stuck state.

### Run

1. **Do not** restart fleetd this time. Wait for the natural osquery cert refetch cycle (default ~30 min). If your dev config has a shorter `fleet_desktop_refetch_interval`, use that.
   - To skip the wait: in the Fleet UI go to **Host details** and click **Refetch**. This kicks an osquery cycle that includes cert refetch.
2. After the refetch fires, wait 1 minute.
3. Run `fleetctl trigger --name cleanups_then_aggregation`.
4. Check **Host details → Activity**.

### Pass if

- Same as Scenario A — no spurious push, profile stays Verified. The recovery happened even though no new certs were inserted this cycle.

### Fail if

- Same as Scenario A.

---

## Scenario D — Failed profile does NOT get re-pushed every hour

**Customer scenario being mimicked:** the customer's SCEP server is broken (misconfiguration, network outage, expired CA cert). Fleet pushed the profile, the device tried SCEP, SCEP failed, the per-platform profile is parked at `status='failed'`. Pre-PR the cron's HAVING clause naturally excluded these rows so Fleet stopped trying — admin must fix SCEP and click Resend. **This scenario verifies the PR doesn't regress that behavior into an hourly push loop.**

### Setup (two statements, mimic broken-SCEP outcome — pick the table for your host's platform)

```sql
-- Mimics: SCEP delivery failed, per-platform profile parked at 'failed'.
-- Use host_mdm_windows_profiles for Windows, host_mdm_apple_profiles for macOS.
UPDATE host_mdm_windows_profiles  -- or host_mdm_apple_profiles
   SET status = 'failed'
 WHERE host_uuid = '$HUUID' AND profile_uuid = '$PUUID' AND operation_type = 'install';

-- Mimics: hmmc was blanked by the renewal cron before SCEP failed (so the
-- row looks "stuck" by time, the way it would 5+ hours after a failed renewal).
UPDATE host_mdm_managed_certificates
   SET not_valid_before = NULL, not_valid_after = NULL, serial = NULL,
       updated_at = NOW() - INTERVAL 5 HOUR
 WHERE host_uuid = '$HUUID' AND profile_uuid = '$PUUID';
```

### Run

1. In the Fleet UI: confirm **Host details → Configuration profiles** shows the SCEP profile as **Failed**.
2. On the host, restart fleetd to drive a cert refetch.
3. Wait 2 minutes.
4. Run `fleetctl trigger --name cleanups_then_aggregation`.
5. Wait 1 minute.
6. Refresh **Host details → Activity**.

### Pass if

- No new MDM push for the SCEP profile in the activity feed (the cron correctly leaves the row alone — no loop).
- Profile status stays **Failed**.

### Fail if

- Any new "Resent" / "Sent profile" / push activity appears for the SCEP profile → the matcher wrongly recovered the row, the cron re-armed, and we'd be in the hourly-push loop.

### Cleanup

```sql
-- Use the same table you set in setup.
UPDATE host_mdm_windows_profiles  -- or host_mdm_apple_profiles
   SET status = 'verified'
 WHERE host_uuid = '$HUUID' AND profile_uuid = '$PUUID';
```

Then click **Refetch** in the Fleet UI to let hmmc repopulate naturally.
