# CIS benchmarks: authoring, testing, and automation

This document describes how CIS benchmark policies work in Fleet, how
to write and test them, and how to use an AI agent to generate them
from CIS PDF documents end-to-end.

## Directory layout

```
ee/cis/
  macos-13/
  macos-14/
  macos-15/
  macos-26/
  win-10/
  win-11/
  win-11-intune/
```

Each OS directory follows the same structure:

```
cis-policy-queries.yml          # All policies for this OS version
README.md                       # Limitations, org-decision policies, notes
test/
  scripts/                      # Shell scripts that remediate or break settings
    CIS_1.1_pass.sh
    CIS_1.1_fail.sh
    CIS_3.1.sh                  # Pass-only (no fail counterpart)
  profiles/                     # MDM configuration profiles (.mobileconfig)
    1.6.mobileconfig
    2.5.1-enable.mobileconfig
    2.5.1-disable.mobileconfig
    README.md                   # How to create new profiles
```

## Policy format

Every policy is a YAML document inside `cis-policy-queries.yml`:

```yaml
---
apiVersion: v1
kind: policy
spec:
  name: "CIS - <title from the benchmark> (<qualifier>)"
  cis_id: "<dotted CIS number, e.g. 2.3.3.4>"
  platforms: macOS
  platform: darwin
  description: |
    <description from the benchmark>
  resolution: |
    <remediation steps from the benchmark>
  query: |
    <osquery SQL that returns 1+ rows when compliant, 0 rows when not>
  purpose: Informational
  tags: compliance, CIS, CIS_Level<1 or 2>
  contributors: <github-username>
```

### Field reference

| Field | Required | Notes |
|-------|----------|-------|
| `name` | yes | Format: `CIS - <benchmark title>`. Append `(MDM Required)`, `(Fleetd Required)`, or `(FDA Required)` when the check depends on managed profiles, fleetd tables, or full disk access. |
| `cis_id` | yes | The dotted section number from the benchmark document (e.g. `"2.3.3.4"`). For combined checks, comma-separate: `"5.2.3, 5.2.4"`. |
| `platforms` | yes | Human-readable: `macOS`, `Windows`, etc. |
| `platform` | yes | Osquery platform string: `darwin`, `windows`, `linux`. |
| `description` | yes | Take directly from the benchmark's Description section. |
| `resolution` | yes | Take from the benchmark's Remediation section. Include both graphical and terminal methods when available. |
| `query` | yes | Osquery SQL. Must return 1+ rows when compliant, 0 rows when not. |
| `purpose` | yes | Always `Informational`. |
| `tags` | yes | `compliance, CIS, CIS_Level1` or `CIS_Level2`. |
| `contributors` | no | GitHub username of the author. |

### Query patterns

Every query must return 1+ rows when compliant and 0 rows when not.
The patterns below compose: most real policies use more than one.

**1. Direct table check** — query an osquery table directly for live state:
```sql
SELECT 1 FROM alf WHERE global_state >= 1;
```

**2. Managed policy check** — verify an MDM profile is installed.

Always pair an `EXISTS (good value)` with a `NOT EXISTS (conflicting
value)`. A single `EXISTS` passes if *any* managed_policies row has
the good value, even when a user-scope row on the same host is
overriding it with a bad value. The `NOT EXISTS` guard catches that.

```sql
SELECT 1 WHERE
  EXISTS (
    SELECT 1 FROM managed_policies WHERE
      domain='com.apple.SoftwareUpdate' AND
      name='CriticalUpdateInstall' AND
      (value = 1 OR value = 'true') AND
      username = ''
  )
  AND NOT EXISTS (
    SELECT 1 FROM managed_policies WHERE
      domain='com.apple.SoftwareUpdate' AND
      name='CriticalUpdateInstall' AND
      (value != 1 AND value != 'true')
  );
```

**`username = ''` vs. user-scoped keys.** System-scoped MDM keys
(most `com.apple.*` domains) deliver with `username = ''` on the
`managed_policies` row. A handful of domains — notably
`com.apple.Safari`, `com.apple.Terminal`, and a few
`com.apple.applicationaccess` user-preference keys — deliver at
user scope, so the row's `username` is the logged-in user. For
those, *omit* the `username = ''` clause on the `EXISTS`; the
`NOT EXISTS` guard is enough on its own. When in doubt, push a
test profile and run `SELECT * FROM managed_policies WHERE
domain = 'com.apple.<x>'` to see which scope delivered.

**Multi-key managed_policies check.** When one profile sets
multiple keys that must all be correct (firewall + stealth,
askForPassword + askForPasswordDelay), AND together an
`EXISTS`/`NOT EXISTS` pair per key:

```sql
SELECT 1 WHERE
  EXISTS ( /* key1 = good */ ) AND NOT EXISTS ( /* key1 = bad */ )
  AND EXISTS ( /* key2 = good */ ) AND NOT EXISTS ( /* key2 = bad */ );
```

**3. Negation check** — verify something is absent or disabled:
```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT * FROM plist WHERE
    path = '/var/db/com.apple.xpc.launchd/disabled.plist' AND
    key = 'com.openssh.sshd' AND
    value = '0'
);
```

**4. Absence-passes / numeric threshold** — for checks where an
unmanaged host is compliant and only a non-compliant *managed* value
should fail (e.g. "deferment ≤ 30 days" passes when deferment is not
managed at all):
```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM managed_policies WHERE
    domain='com.apple.applicationaccess' AND
    name='enforcedSoftwareUpdateDelay' AND
    CAST(value AS INTEGER) > 30
);
```

Unlike pattern 2, this pattern does not require the setting to be
present — it only requires that any present value is within the
acceptable range. Use when the benchmark explicitly states that
absence of the setting also satisfies the audit.

**5. Either-local-or-managed compound** — CIS sometimes accepts
either a local plist value or a managed profile as compliant (e.g.
Guest Account disabled, Automatic Login disabled). Query both paths
with `OR`:

```sql
SELECT 1 WHERE
  EXISTS (
    SELECT 1 FROM plist WHERE
      path = '/Library/Preferences/com.apple.loginwindow.plist' AND
      key = 'GuestEnabled' AND value = 0
  )
  OR EXISTS (
    SELECT 1 FROM managed_policies WHERE
      domain = 'com.apple.MCX' AND name = 'DisableGuestAccount' AND
      (value = 1 OR value = 'true') AND username = ''
  );
```

When the either-branch uses pattern 2, still include its
`NOT EXISTS` guard inside that branch.

### Naming qualifiers

Append these to the policy name when applicable:

- **(MDM Required)** — query checks `managed_policies`; needs an MDM profile installed
- **(Fleetd Required)** — query uses a fleetd-specific table (e.g. `software_update`)
- **(FDA Required)** — query reads paths that require full disk access

### Fleetd tables used by CIS queries

Fleetd ships custom osquery tables for settings that can't be read
from stock osquery. When writing a query against one, check the
source at `orbit/pkg/table/<name>/` to confirm the column names
and any required `WHERE` constraints. The table below is the set
currently in use by macOS CIS policies.

| Table | Key columns | Required constraints | Notes |
|-------|-------------|----------------------|-------|
| `authdb` | `right_name`, `json_result` | `right_name` must be equality-constrained | `json_result` is a JSON blob — use `json_extract(json_result, '$.rule')` to inspect rules. |
| `csrutil_info` | `ssv_enabled` | — | Integer 0/1. |
| `dscl` | `command`, `path`, `key`, `value` | `command`, `path`, `key` required; `value` is output only; currently only `command = 'read'` supported | Reads Directory Service records. |
| `find_cmd` | `directory`, `type`, `perm`, `path` | `directory` required (must be absolute); `type` and `perm` optional; `path` is output only | `find_cmd` shells out to `/usr/bin/find`; prefer it over walking the `file` table for large scopes like `/System/Volumes/Data/System` (core `file` exceeds osquery CPU/memory limits on 10k+ rows). |
| `nvram_info` | `amfi_enabled` | — | Integer 0/1. |
| `pmset` | `getting`, `json_result` | `getting` (e.g. `'custom'`) | `json_result` contains per-power-source nested dicts; use `JSON_EXTRACT(json_result, '$.AC Power:')` etc. |
| `pwd_policy` | `max_failed_attempts`, `expires_every_n_days`, `days_to_expiration`, `history_depth`, `min_mixed_case_characters` | — | See console-user caveat below. |
| `password_policy` | `policy_identifier`, `policy_content` | — | macOS-native osquery table, *not* fleetd — listed here for completeness. |
| `software_update` | `software_update_required` | — | — |
| `sudo_info` | `json_result` | — | Parsed output of `sudo -V`; use `JSON_EXTRACT(json_result, '$.Authentication timestamp timeout')`, `'$.Type of authentication timestamp record'`, `'$.Log when a command is allowed by sudoers'`. |
| `user_login_settings` | `password_hint_enabled` | — | See console-user caveat below. |

**Console-user-scope caveat.** `pwd_policy` and
`user_login_settings` (and the native `location_services` table)
execute their underlying commands as the current console user.
If no console user is logged in at query time — or on a headless
test VM where the console user is `root` — the tables return
empty results and the query silently fails (0 rows). Ensure a
non-root console user is logged in before evaluating any policy
that depends on these tables. Current affected policies:
`5.2.1`, `5.2.2`, `5.2.7`, `5.2.8`, `2.12.1`, `2.6.1.1`.

## Test artifacts

Every policy should be testable by at least one of the following:
a shell script, an MDM profile, or manual steps. The test runner
classifies each policy into a test type based on what artifacts exist.

### Test type classification (in priority order)

| Priority | Test type | Artifacts present | How the runner tests it |
|----------|-----------|-------------------|-------------------------|
| 1 | PASS_FAIL | `_pass.sh` + `_fail.sh` | Run fail script → verify query fails → run pass script → verify query passes |
| 2 | PASS_ONLY | `CIS_{id}.sh` | Run script → verify query passes |
| 3 | PROFILE | `.mobileconfig` in profiles dir (no scripts) | Verify query fails without profile → push profile to team → verify query passes |
| 4 | MANUAL | None of the above | Prompt user with resolution steps (or skip with `--skip-manual`) |

Scripts take priority over profiles. If a policy has both a script and
a profile, the script-based test type is used and the profile is
pushed alongside all other profiles during setup.

### Profile-only policies

Many MDM-dependent policies have no shell script — the only way to
make them pass is to install a `.mobileconfig` profile via Fleet. For
these, the test runner automatically:

1. Runs the query **before** pushing any profiles to verify it returns
   0 rows (confirms the query detects non-compliance)
2. Pushes all needed profiles to the Fleet team
3. Waits for profile delivery
4. Runs the query again to verify it now returns rows

If a query already passes before its profile is delivered, the runner
records a `note:` in the result details warning that the query may not
detect non-compliance — the test can still PASS if the post-delivery
query succeeds. Some queries (firewall, Gatekeeper) check OS state
that may be compliant regardless of the profile, which is why the
pre-delivery pass isn't automatically treated as a failure. An
unexpected pre-delivery pass is worth investigating but not
disqualifying.

### Test scripts

Test scripts live in `test/scripts/` and follow strict naming conventions.

**Pass/fail pairs** — for policies where both directions can be scripted:

- `CIS_{cis_id}_pass.sh` — applies the remediation so the query returns rows
- `CIS_{cis_id}_fail.sh` — undoes the remediation so the query returns 0 rows

Example (`CIS_2.3.3.4_pass.sh`):
```bash
#!/bin/bash
# CIS 2.3.3.4 - Ensure Remote Login Is Disabled
# Disables SSH so the policy query passes.
/usr/bin/sudo /usr/sbin/systemsetup -setremotelogin off <<< "yes"
```

Example (`CIS_2.3.3.4_fail.sh`):
```bash
#!/bin/bash
# CIS 2.3.3.4 - Ensure Remote Login Is Disabled
# Enables SSH so the policy query fails.
/usr/bin/sudo /usr/sbin/systemsetup -setremotelogin on
```

**Pass-only scripts** — for policies where the fail state is the default
or can't be easily scripted:

- `CIS_{cis_id}.sh` — applies the remediation

### Script conventions

- Always use `#!/bin/bash`.
- Use full paths for system commands (`/usr/bin/sudo`, `/usr/sbin/systemsetup`).
- Include a comment with the CIS ID and policy name.
- Use `sudo` for privileged operations (the test runner provides the password).
- Prefix unreliable scripts with `not_always_working_` — the test runner skips these.
- Fail scripts that *create* artifacts (stub apps, stub directories,
  extra plist keys) should pair with a pass script that removes
  them, or note in the README that runner teardown must clean up.
  Leaving a stub world-writable `.app` in `/Applications` after the
  test will break subsequent runs.

### Common script patterns

These idioms come up repeatedly and are worth reusing verbatim.

**Console user** — needed when setting or reading a per-user
preference from within a system-level script:
```bash
user=$(/usr/bin/stat -f "%Su" /dev/console)
if [ -n "$user" ] && [ "$user" != "root" ]; then
  /usr/bin/sudo -u "$user" /usr/bin/defaults write <domain> <key> ...
fi
```
On a headless test VM without a logged-in console user, the value
is `root` — the script should no-op rather than fail.

**Iterate `/Users/*`** — for per-user settings that must be applied
to every local account. Skip `Shared`, `Guest`, and dot-prefixed
directories (`/Users/.localized` etc.):
```bash
for userhome in /Users/*; do
  user=$(basename "$userhome")
  case "$user" in Shared|Guest|.*) continue ;; esac
  [ -d "$userhome/Library/Preferences" ] || continue
  /usr/bin/sudo -u "$user" /usr/bin/defaults write com.apple.dock ...
done
```

**Idempotent delete** — use `|| true` so a missing key doesn't
trip the `-e` exit behavior:
```bash
/usr/bin/sudo /usr/bin/defaults delete <domain> <key> 2>/dev/null || true
```

**sudoers.d filename rule** — macOS ignores files in
`/etc/sudoers.d/` whose names contain a dot. Use underscores and
no extension:
```bash
echo 'Defaults timestamp_timeout=0' | \
  /usr/bin/sudo /usr/bin/tee /etc/sudoers.d/CIS_5_4_sudoconfiguration > /dev/null
/usr/bin/sudo /bin/chmod 0440 /etc/sudoers.d/CIS_5_4_sudoconfiguration
```

**Atomic config-file edits** — when rewriting a system config
file (audit_control, asl/com.apple.install), write to a temp file
via `awk`/`sed` and rename. Never edit in place:
```bash
TMP="$(/usr/bin/mktemp /tmp/audit_control.XXXXXX)"
/usr/bin/sudo /usr/bin/awk '...' /etc/security/audit_control > "$TMP"
/usr/bin/sudo /bin/mv "$TMP" /etc/security/audit_control
/usr/bin/sudo /usr/sbin/chown root:wheel /etc/security/audit_control
/usr/bin/sudo /bin/chmod 0440 /etc/security/audit_control
```

### Choosing between scripts and profiles

The decision is driven by **what the query reads**, not by what
remediation methods the PDF provides. Decide based on how the setting
is configured:

- **Query reads local state** (osquery table that reflects
  system/service state, local `plist` file, `launchd`, `file`, etc.):
  create shell scripts. These are more reliable and don't require
  MDM.
- **Query reads `managed_policies`**: create a `.mobileconfig`
  profile only, regardless of whether the PDF also lists a Terminal
  Method. A local `defaults write` does not populate
  `managed_policies` — only an MDM-installed profile will change what
  the query sees. Scripts would pass on disk but leave the query's
  result unchanged.
- **Query reads both** (rare, e.g. a policy that checks either a
  local setting or its managed override): create scripts for the
  local path and a profile for the managed path. Scripts take
  priority in the runner; the profile is installed alongside.
- **Neither** (GUI-only, requires user interaction): document in
  `README.md` as a limitation. The test runner will prompt the user
  or skip.

## MDM configuration profiles

Profiles live in `test/profiles/` as `.mobileconfig` XML plist files.

### Naming conventions

| Pattern | When to use |
|---------|-------------|
| `{cis_id}.mobileconfig` | Single profile that makes the policy pass |
| `{cis_id}-enable.mobileconfig` | Enables a setting (org-decision policies) |
| `{cis_id}-disable.mobileconfig` | Disables a setting (org-decision policies) |
| `{cis_id}.enable.mobileconfig` | Alternate dot-separated variant (same purpose) |
| `{cis_id}-part1.mobileconfig` | Multi-part profiles that must be installed together |
| `{id1}-and-{id2}.mobileconfig` | Covers multiple CIS IDs in one profile |

### Creating a new profile

1. Generate two UUIDs with `uuidgen` — one for the top-level
   `PayloadUUID` and one for the inner payload `PayloadUUID`.
2. Set the inner `PayloadType` to the MDM domain (e.g.
   `com.apple.SoftwareUpdate`).
3. Set `PayloadIdentifier` to `com.fleetdm.cis-{cis_id}` (top-level)
   and `com.fleetdm.cis-{cis_id}.check` (inner).
4. Add the configuration keys and values from the benchmark to the
   inner payload dict. **Multiple keys for the same `PayloadType`
   belong in a single payload dict**, not separate profiles — some
   benchmarks explicitly require this (e.g. a deferment profile
   needing both `enforcedSoftwareUpdateDelay` and
   `forceDelayedSoftwareUpdates` to be effective). Use
   `{cis_id}-part1.mobileconfig` only when a benchmark genuinely
   requires *multiple profiles* to be installed together.
5. Validate the generated file with `/usr/bin/plutil -lint
   path/to/file.mobileconfig` before committing.

### XML template

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadDisplayName</key>
      <string>CIS {cis_id}</string>
      <key>PayloadType</key>
      <string>{PayloadType from benchmark, e.g. com.apple.SoftwareUpdate}</string>
      <key>PayloadIdentifier</key>
      <string>com.fleetdm.cis-{cis_id}.check</string>
      <key>PayloadUUID</key>
      <string>{inner UUID from uuidgen}</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <!-- One or more setting keys, all within this payload dict: -->
      <key>{SettingKey1}</key>
      <{true|false|string|integer|...}>{value}</...>
      <key>{SettingKey2}</key>
      <{...}>{value}</...>
    </dict>
  </array>
  <key>PayloadDescription</key>
  <string>CIS {cis_id} - {title}</string>
  <key>PayloadDisplayName</key>
  <string>{title}</string>
  <key>PayloadIdentifier</key>
  <string>com.fleetdm.cis-{cis_id}</string>
  <key>PayloadRemovalDisallowed</key>
  <false/>
  <key>PayloadScope</key>
  <string>System</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>{top-level UUID from uuidgen}</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

Note that `PayloadVersion` appears at **both** the top-level and
inner-payload level. Both are required for profiles to install
reliably.

### Multi-key profiles

Benchmarks sometimes require multiple settings keys to be present
together for a check to pass (firewall + stealth mode, askForPassword
+ askForPasswordDelay, BlockStoragePolicy + WebKit storage blocking,
etc.). Three ways this can show up; pick the right one:

| Situation | Pattern |
|-----------|---------|
| One benchmark recommendation, multiple keys on the same `PayloadType` | One profile, multiple keys in the inner `PayloadContent` dict. |
| Two separate benchmark IDs that CIS *explicitly* wants enforced via the same profile (e.g. 2.2.1 Firewall + 2.2.2 Stealth Mode — CIS says putting them in separate profiles makes them fail) | One profile named `{id1}-and-{id2}.mobileconfig`. The query for each ID checks its own key. |
| One benchmark that genuinely requires multiple *profiles* to be installed together (rare — different `PayloadType`s that can't coexist in one payload) | Two files named `{cis_id}-part1.mobileconfig` and `{cis_id}-part2.mobileconfig`. |

**Worked example.** CIS 2.2.1 (Firewall) + 2.2.2 (Stealth Mode)
share the `com.apple.security.firewall` payload. CIS explicitly
requires them in the same profile:

```xml
<key>PayloadContent</key>
<array>
  <dict>
    <key>PayloadType</key>
    <string>com.apple.security.firewall</string>
    <key>PayloadIdentifier</key>
    <string>com.fleetdm.cis-2.2.1-and-2.2.2.check</string>
    <key>PayloadUUID</key>
    <string>{inner UUID}</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>EnableFirewall</key>
    <true/>
    <key>EnableStealthMode</key>
    <true/>
  </dict>
</array>
```

File name: `2.2.1-and-2.2.2.mobileconfig`. Each CIS ID gets its
own policy YAML entry, each with its own query checking its own
key.

## README.md per OS version

Each OS directory has a `README.md` that must document:

1. Which benchmark version the policies target.
2. **Status** — which sections are complete and which are still WIP.
3. **Sections covered** — a table listing §1..§N with status per
   section (complete, in progress, not started, skipped).
4. **Limitations** — benchmarks that cannot be checked as a Fleet
   policy (manual audits, GUI-only settings). Include every
   Manual-assessment recommendation from the PDF with its CIS ID,
   level, title, and a one-line reason it can't be automated.
5. **Org-decision policies** — where CIS leaves the choice to the
   organization; Fleet provides both enable/disable variants.
6. **Optional policies** — benchmarks CIS ships at a level
   higher than what Fleet enforces by default (e.g. Level 2
   recommendations on a deployment that targets Level 1), or
   org-chosen alternatives for items CIS leaves open.
7. **Per-section notes** — for each section that shipped, a short
   block (`### Section N notes`) explaining any query patterns,
   table-schema quirks, test artifact tradeoffs, or caveats the
   next maintainer will want to see. Anything you flagged in the
   state file's "Open questions" is a candidate.

**Sections with no automated checks.** If a whole section (e.g.
`§2.4 Menu Bar`) contains only Manual recommendations in the
current benchmark version, say so explicitly rather than leaving
it blank — readers otherwise assume it's a gap:

> §2.4 (Menu Bar), §2.8 (Displays), §2.14–2.17 (Game Center,
> Notifications, Wallet, Internet Accounts) contain only
> Manual-assessment recommendations in this version and are
> therefore not represented in `cis-policy-queries.yml`. See
> Limitations for the individual items.

## Test runner

`tools/cis/cis-test-runner.py` automates the full test cycle:

```bash
# Test everything, skip manual policies / policies without test artifacts
python3 tools/cis/cis-test-runner.py \
    --macos-version 14 --all --skip-manual \
    --fleet-url $FLEET_URL --fleet-token $FLEET_API_TOKEN

# Test specific CIS IDs
python3 tools/cis/cis-test-runner.py \
    --macos-version 14 --cis-ids 2.3.3.4,1.1

# Clean up everything after
python3 tools/cis/cis-test-runner.py \
    --macos-version 14 --all --skip-manual --cleanup
```

The runner creates a Fleet team, builds and installs a fleet agent in
a tart VM, enrolls it, runs each test, and prints a summary. See
`tools/cis/README.md` for full flag reference.

---

## Adding a new macOS version

Creating the `ee/cis/macos-NN/` directory is only half the job —
the Python test runner also needs to know the new version exists.
Missing any of these registrations will make the runner either
reject `--macos-version NN` outright or fail unpredictably mid-run.

**1. Scaffold the OS directory.**

```
ee/cis/macos-NN/
  cis-policy-queries.yml   # starts empty; append as you go
  README.md                # use the structure from "README.md per OS version"
  test/
    scripts/               # empty
    profiles/              # empty
```

**2. Register the version in `tools/cis/cis-test-runner.py`.**
Four dicts near the top of the file key off the OS version
string (`"13"`, `"14"`, `"15"`, `"NN"`). Add an entry to each:

- `VERSION_MAP`: the Tart base image URL for the OS and the
  matching `ee/cis/` directory name.
  ```python
  "26": {
      "image": "ghcr.io/cirruslabs/macos-tahoe-base:latest",
      "dir": "macos-26",
  },
  ```

- `SSH_BREAKING_CIS_IDS`: CIS IDs whose pass scripts disable
  sshd (usually Remote Login and Remote Management). The runner
  flips these to MANUAL so it doesn't lose its SSH session:
  ```python
  "26": {"2.3.3.4", "2.3.3.5"},
  ```

- `PASSWORD_POLICY_CIS_IDS`: CIS IDs whose MDM profiles enforce
  a password policy strong enough to reject the VM's test user
  password. The runner installs these individually with a
  restore step:
  ```python
  "26": {"5.2.1", "5.2.2", "5.2.7", "5.2.8"},
  ```

- `NON_AUTOMATABLE_CIS_IDS`: CIS IDs that cannot be reliably
  tested on a VM at all, with a reason each. Seed with the
  usual suspects; add more as the runner surfaces unreliable
  tests:
  ```python
  "26": {
      "1.1": "Requires real hardware to install Apple updates",
      "2.6.1.1": "VM cannot satisfy user-privacy gate for Location Services",
  },
  ```

CIS section numbers are *not* stable across releases — verify
each mapping against the new version's PDF before copying it
from an older OS entry.

**3. Confirm the Tart base image exists.** Cirrus Labs usually
publishes `ghcr.io/cirruslabs/macos-<codename>-base:latest` soon
after a macOS release. If it isn't published yet, the runner
won't be able to spin up a test VM — mark the `VERSION_MAP`
entry TODO in the state file and plan to revisit.

**4. Only then** start writing policies. Trying to run the
runner before these registrations will fail at argument parsing
with `argument --macos-version: invalid choice: 'NN' (choose
from '13', '14', '15')`.

## Updating benchmarks when a new CIS version is released

### Manual process

1. Download the new PDF from the CIS website.
2. Read the **Appendix: Change History** at the end of the document.
3. For each change:
   - **Added**: write a new policy entry with all fields.
   - **Modified**: update the changed fields (description, resolution, audit) from the new document.
   - **Removed**: delete the policy entry, its test scripts, and any associated profiles.
4. **Audit every previous-version policy for an
   Automated → Manual downgrade.** CIS frequently moves
   recommendations from Automated to Manual between releases
   (e.g. Tahoe moved Hey Siri and the password-complexity
   items to Manual). The Change History usually flags these,
   but not always — diff the new PDF's per-section
   "Assessment Status: Automated | Manual" lines against the
   previous version's. **Any policy that went Manual must be
   deleted from the YAML, scripts removed, and the
   recommendation moved to the README's Limitations section.**
   Shipping a query for a Manual recommendation is worse than
   shipping nothing, because the query will produce
   non-authoritative results.
5. For each added or modified policy, create test scripts
   (`_pass.sh`/`_fail.sh`).
6. For MDM-dependent policies, create the `.mobileconfig` profile.
7. Update `README.md` with any new limitations or org-decision
   policies.
8. Run the test runner against the updated policies.

### What changes between versions

CIS benchmark updates typically involve:
- **Renumbered sections** — a recommendation moves to a different section
- **Title changes** — wording updates (e.g. "Ensure All Apple-provided Software Is Current" -> "Ensure Apple-provided Software Updates Are Installed")
- **Description/rationale updates** — expanded context or new references
- **Audit method changes** — new or updated terminal commands for verification
- **Remediation changes** — updated steps, new profile keys
- **Assessment status changes** — Automated to Manual or vice versa (Manual policies should be removed from the YAML since they can't be queried)
- **New recommendations** — entirely new security checks
- **Removed recommendations** — moved to supplemental or deleted entirely

---

## AI agent prompt for generating CIS benchmarks

An agent-ready prompt for generating or updating a full benchmark
(policies, test scripts, profiles, README) from a CIS PDF lives in
[`prompt.md`](./prompt.md) in this directory. It references the
conventions defined above — update both if conventions change.

## Durable state for long generation runs (optional)

When generating a full benchmark from a large PDF in multiple
sessions, it helps to maintain a state file at
`tmp/<os>-<version>-state.md` that records locked decisions,
per-section progress, open questions, and a **Next action**
pointer so a future session can resume from cold context.

Not a hard convention — just a useful checkpoint mechanism when a
single session can't finish the job. The template below is the
minimum that has proven useful; extend as needed.

````markdown
# <OS> <version> CIS benchmark — state file

**Purpose:** durable checkpoint for generating Fleet's CIS
benchmark for <OS> <version>, resumable across sessions.

## How to resume

In a new session, say: *"Resume <OS> <version> CIS work from
`tmp/<os>-<version>-state.md`."* Then re-read this file and
continue from the **Next action** line.

## Task (frozen)

Create `ee/cis/<os>-<version>/` — policies, test scripts, MDM
profiles, README — from `pdf/<filename>.pdf` alone, following
`ee/cis/CIS-BENCHMARKS.md` and `ee/cis/prompt.md`.

## Decisions (locked for this session)

- Approach (thin slice end-to-end vs. by-file)
- Levels covered (Level 1, Level 2, or both)
- Branch
- Org-decision default (ship both enable/disable, or pick one)
- Handling of uncertain profile keys (skip + TODO, guess, or ask)
- Whether §7 Supplemental is skipped

## Progress tracker

| Section | Status | Policies | Scripts | Profiles | README | Notes |
|---------|--------|----------|---------|----------|--------|-------|
| 1 …     | ⬜     |          |         |          |        |       |
| 2.2 …   | ⬜     |          |         |          |        |       |

Status legend: ⬜ not started · 🟨 in progress · ✅ done ·
⏭ skipped.

### Validation
- [ ] Runner registered — all four dicts in `cis-test-runner.py`
      (`VERSION_MAP`, `SSH_BREAKING_CIS_IDS`,
      `PASSWORD_POLICY_CIS_IDS`, `NON_AUTOMATABLE_CIS_IDS`)
- [ ] YAML parses, `cis_id`s unique
- [ ] All profiles `plutil -lint` OK
- [ ] Profile UUIDs unique (top-level + inner, no duplicates)
- [ ] Fleetd table schemas verified against `orbit/pkg/table/`
- [ ] Test runner dry run against generated policies
- [ ] Summary reviewed, failures fixed

## Open questions / TODOs

### §<section>
- <uncertainty, audit ambiguity, or runtime risk>

## Files touched

- `tmp/<os>-<version>-state.md` (this file)
- `ee/cis/<os>-<version>/cis-policy-queries.yml`
- `ee/cis/<os>-<version>/README.md`
- `ee/cis/<os>-<version>/test/scripts/` — N files …
- `ee/cis/<os>-<version>/test/profiles/` — N files …

## Next action

<One sentence describing exactly where to pick up.>
````
