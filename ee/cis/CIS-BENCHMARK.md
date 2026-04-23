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
  win-10/
  win-11/
  win-11-intune/

Each OS directory follows the same structure:

  cis-policy-queries.yml        # All policies for this OS version
  README.md                     # Limitations, org-decision policies, notes
  test/
    scripts/                    # Shell scripts that remediate or break settings
      CIS_1.1_pass.sh
      CIS_1.1_fail.sh
      CIS_3.1.sh               # Pass-only (no fail counterpart)
    profiles/                   # MDM configuration profiles (.mobileconfig)
      1.6.mobileconfig
      2.5.1-enable.mobileconfig
      2.5.1-disable.mobileconfig
      README.md                 # How to create new profiles
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

### Coverage

Generate policies for **all** Automated recommendations in the
benchmark, regardless of level. Both Level 1 and Level 2
recommendations must be covered — the level is recorded in `tags`
(`CIS_Level1` or `CIS_Level2`) and does not determine whether a
policy is created.

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
| `tags` | yes | `compliance, CIS, CIS_Level1` or `CIS_Level2` — set to match the recommendation's Profile Applicability in the benchmark. Both levels are in scope; do not filter by level. |
| `contributors` | no | GitHub username of the author. |

### Query patterns

There are three common query patterns:

**1. Direct table check** — query an osquery table directly:
```sql
SELECT 1 FROM alf WHERE global_state >= 1;
```

**2. Managed policy check** — verify an MDM profile is installed (requires MDM enrollment):
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

The EXISTS/NOT EXISTS pattern ensures the setting is actively managed
AND that no conflicting value exists at another scope level.

**3. Negation check** — verify something is absent or disabled:
```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT * FROM plist WHERE
    path = '/var/db/com.apple.xpc.launchd/disabled.plist' AND
    key = 'com.openssh.sshd' AND
    value = '0'
);
```

### Naming qualifiers

Append these to the policy name when applicable:

- **(MDM Required)** — query checks `managed_policies`; needs an MDM profile installed
- **(Fleetd Required)** — query uses a fleetd-specific table (e.g. `software_update`)
- **(FDA Required)** — query reads paths that require full disk access

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
| 4 | MANUAL | None of the above | Prompt user with resolution steps (or skip with `--skip-no-script`) |

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

- Always use `#!/bin/bash`
- Use full paths for system commands (`/usr/bin/sudo`, `/usr/sbin/systemsetup`)
- Include a comment with the CIS ID and policy name
- Use `sudo` for privileged operations (the test runner provides the password)
- Prefix unreliable scripts with `not_always_working_` — the test runner skips these

### Choosing between scripts and profiles

When creating tests for a new policy, decide based on how the setting
is configured:

- **System service or plist-based setting** (e.g. enable/disable SSH,
  launchd service, defaults write): create shell scripts. These are
  more reliable and don't require MDM.
- **MDM-only setting** (e.g. managed_policies check, no terminal
  remediation method in the benchmark): create a `.mobileconfig`
  profile only. The test runner handles these automatically.
- **Both available**: create shell scripts (they give better test
  coverage since they test both directions explicitly). Also create
  the profile — it gets pushed during setup and is available if
  needed.
- **Neither** (GUI-only, requires user interaction): document in
  README.md as a limitation. The test runner will prompt the user or
  skip.

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

1. Copy an existing profile as a template
2. Generate two UUIDs: `uuidgen` (one for payload, one for top-level)
3. Set `PayloadType` to the MDM domain (e.g. `com.apple.SoftwareUpdate`)
4. Set `PayloadIdentifier` to `com.fleetdm.cis-{cis_id}` (top-level) and `com.fleetdm.cis-{cis_id}.check` (payload)
5. Add the configuration keys and values from the benchmark

See `test/profiles/README.md` for the full XML template.

## README.md per OS version

Each OS directory has a `README.md` that must document:

1. Which benchmark version the policies target
2. **Limitations** — benchmarks that cannot be checked as a Fleet policy (manual audits, GUI-only settings)
3. **Org-decision policies** — where CIS leaves the choice to the organization; Fleet provides both enable/disable variants
4. **Optional policies** — benchmarks CIS includes but does not require (e.g. password complexity)

## Test runner

`tools/cis/cis-test-runner.py` automates the full test cycle:

```bash
# Test everything, skip policies without scripts
python3 tools/cis/cis-test-runner.py \
    --macos-version 14 --all --skip-no-script \
    --fleet-url $FLEET_URL --fleet-token $FLEET_API_TOKEN

# Test specific CIS IDs
python3 tools/cis/cis-test-runner.py \
    --macos-version 14 --cis-ids 2.3.3.4,1.1

# Clean up everything after
python3 tools/cis/cis-test-runner.py \
    --macos-version 14 --all --skip-no-script --cleanup
```

The runner creates a Fleet team, builds and installs a fleet agent in
a tart VM, enrolls it, runs each test, and prints a summary. See
`tools/cis/README.md` for full flag reference.

---

## Updating benchmarks when a new CIS version is released

### Manual process

1. Download the new PDF from the CIS website
2. Read the **Appendix: Change History** at the end of the document
3. For each change:
   - **Added**: write a new policy entry with all fields
   - **Modified**: update the changed fields (description, resolution, audit) from the new document
   - **Removed**: delete the policy entry
4. For each added or modified policy, create test scripts (`_pass.sh`/`_fail.sh`)
5. For MDM-dependent policies, create the `.mobileconfig` profile
6. Update `README.md` with any new limitations or org-decision policies
7. Run the test runner against the updated policies

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
