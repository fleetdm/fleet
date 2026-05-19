# CIS benchmark generation prompt

This file is an AI agent prompt for generating or updating Fleet's CIS
benchmark policies, test scripts, MDM profiles, and documentation from
a CIS PDF. Feed it to the agent alongside the relevant PDFs.

Conventions (directory layout, YAML format, query patterns, script and
profile naming, test runner invocation) live in `CIS-BENCHMARKS.md` in
this directory. This file is a task harness; the reference doc is the
source of truth. If they disagree, update whichever is wrong.

---

You are a security compliance engineer updating Fleet's CIS benchmark
policies. You have access to the Fleet codebase and CIS benchmark PDF
documents.

All conventions (directory layout, YAML format, required fields, query
patterns, script/profile naming, README structure, test runner
invocation) are documented in `ee/cis/CIS-BENCHMARKS.md`. Read that
file before you begin. Do not invent conventions — if something is not
documented, ask the user.

## Your task

Given a CIS benchmark PDF for a specific OS and version, generate or
update the complete set of Fleet policies, test scripts, MDM profiles,
and documentation.

## Input files

- CIS benchmark PDF (new version): `pdf/<filename>.pdf`
- CIS benchmark PDF (previous version, if upgrading): `pdf/<filename>.pdf`
- Existing policies (if upgrading): `ee/cis/<os-dir>/cis-policy-queries.yml`
- Existing tests: `ee/cis/<os-dir>/test/scripts/` and `test/profiles/`
- Conventions: `ee/cis/CIS-BENCHMARKS.md`

## Step-by-step workflow

### Step 0: If this is a new OS version, scaffold and register it first

Skip this step when upgrading an existing OS version's benchmark.

For a brand-new OS version (e.g. `macos-27` when the latest
directory in `ee/cis/` is `macos-26`), follow CIS-BENCHMARKS.md
§Adding a new macOS version **before** writing any policies:

1. Create the `ee/cis/<os-dir>/` scaffold (policy YAML, README,
   `test/scripts/`, `test/profiles/`).
2. Register the version in `tools/cis/cis-test-runner.py` — four
   dict entries (`VERSION_MAP`, `SSH_BREAKING_CIS_IDS`,
   `PASSWORD_POLICY_CIS_IDS`, `NON_AUTOMATABLE_CIS_IDS`).
3. Confirm the Tart base image for the target macOS version is
   published; if not, flag it in the state file and continue.

### Step 1: Extract the changelog

Read the "Appendix: Change History" from the end of the new PDF.
Identify every entry for the target version. Classify each as ADDED,
MODIFIED, or REMOVED.

If this is a new OS version (no existing policies), treat every
recommendation as ADDED.

**Also diff Assessment Status against the previous version.**
The Change History does not always flag Automated → Manual
downgrades. Scan each previous-version Automated recommendation
in the new PDF's §<same-id> and check whether its Assessment
Status is still Automated. Any downgrade means the existing
policy must be deleted (CIS-BENCHMARKS.md §Updating benchmarks
step 4).

### Step 2: Read affected sections

For each changed recommendation, read its full section in the new PDF.
Extract:
- Section number (becomes `cis_id`)
- Title
- Profile Applicability (Level 1 or Level 2)
- Assessment Status (Automated or Manual)
- Description
- Audit method (terminal command)
- Remediation method (terminal and/or profile method)
- PayloadType, key name, and value (if profile-based)

Skip Manual-assessment recommendations — they cannot be automated as
Fleet policies. Note them for the README.md limitations section.

### Step 3: Generate policy YAML

For each Automated recommendation, write a policy document following
the format in CIS-BENCHMARKS.md (§Policy format). Follow the query
rules in §Query patterns — queries MUST return 1+ rows when compliant
and 0 rows when not.

Append name qualifiers per §Naming qualifiers when the query depends
on `managed_policies` (`(MDM Required)`), fleetd-only tables
(`(Fleetd Required)`), or files needing full disk access
(`(FDA Required)`).

Before writing a query against a fleetd table, verify its column
names against CIS-BENCHMARKS.md §Fleetd tables used by CIS
queries, and check the console-user-scope caveat. If you reach
for a table not listed there, read its source at
`orbit/pkg/table/<name>/` first.

### Step 4: Generate test artifacts

Decide which artifact to create based on the remediation method in
the PDF, following §Choosing between scripts and profiles.

- Scripts: shell commands exist in the PDF's remediation. Create
  `test/scripts/CIS_<cis_id>_pass.sh` and `_fail.sh`, or a single
  `CIS_<cis_id>.sh` if only the pass direction is scriptable. Use
  §Test scripts, §Script conventions, and §Common script patterns
  (console-user detection, `/Users/*` iteration, sudoers.d
  filename rule, atomic config-file edits).
- Profiles only: the setting is MDM-only (query uses
  `managed_policies`, PDF only provides a Profile Method). Create
  `test/profiles/<cis_id>.mobileconfig`. The test runner handles
  profile-only policies automatically (see §Profile-only policies).
- Both: prefer scripts (better coverage) and also create the profile.

### Step 5: Generate MDM profiles

For each policy that checks `managed_policies`, create a
`.mobileconfig` using the XML template and naming conventions in
§MDM configuration profiles. For org-decision policies, create both
`-enable` and `-disable` variants.

When a benchmark specifies multiple keys for the same PayloadType
(or explicitly requires two IDs to share one profile), follow
§Multi-key profiles — one payload dict with multiple keys, named
either `{cis_id}.mobileconfig` or `{id1}-and-{id2}.mobileconfig`
depending on scope.

### Step 6: Update README.md

Per §README.md per OS version, document: benchmark version targeted,
limitations (Manual-only recommendations), org-decision policies with
both variants, and optional policies.

### Step 7: Handle removals

For REMOVED recommendations: delete the policy entry from the YAML,
delete associated test scripts and profiles, and remove any mention
from README.md.

### Step 8: Validate

Run the test runner per §Test runner. Review the summary. Fix any
failures: if a query fails after its pass script runs, the query
logic is wrong; if a query passes after its fail script runs, the
fail script isn't effective.

## Important rules

- Never invent query logic — derive it from the PDF's audit section
  and the osquery schema (https://osquery.io/schema/).
- When updating an existing policy, preserve the query unless the
  audit method changed. Only update description, resolution, name,
  and tags from the new document.
- When a recommendation changes from Automated to Manual, remove the
  policy entirely.
- When a recommendation changes from Manual to Automated, add a new
  policy.
- For recommendations where CIS says "audit" (org decides), provide
  both enable and disable policy variants.
- Always include `cis_id` — it is the primary key for mapping
  policies to scripts, profiles, and the benchmark document.
- Do not create policies for supplemental sections (section 7+).
- Ask the user for clarification if the audit method is ambiguous or
  relies on information not available through osquery.
