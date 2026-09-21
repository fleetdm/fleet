---
name: ai-profile-test
description: Propose a new prompt test case for the AI configuration profile generator's test suite and open a PR. Use when asked to "add a profile generator test", "propose a prompt test case", "add a test prompt for AI profiles", or when a generated profile came out wrong and the failure should become a regression test.
allowed-tools: Bash(git *), Bash(gh pr *), Bash(node *), Bash(cd website && sails run *), Read, Grep, Glob, Edit, WebFetch, WebSearch
effort: medium
---

Add a test case to `website/scripts/test-llm-generated-configuration-profile.js` and open a PR. Arguments: $ARGUMENTS

Usage: `/ai-profile-test "<natural-language instruction>" [csp|mobileconfig|ddm]`

- The instruction is what an admin would type into the profile generator (e.g. `"Disable the camera"`). If not provided, ask for it.
- The profile type is inferred from the instruction when obvious (Windows policy → `csp`; Apple → `mobileconfig` or `ddm`). Ask only if genuinely ambiguous.

## Step 1: Branch off main

The test script only exists on `main` — your current branch (e.g. an RC branch) may not have it.

```
git fetch origin
git checkout -b add-profile-test-<short-slug> origin/main
```

## Step 2: Check for existing coverage

Read every entry in `TEST_CASES` at the top of the script. If an existing case already exercises the same concrete setting — the same CSP policy node, mobileconfig payload key, or DDM setting key — with the same intended behavior, or a near-identical instruction, STOP and report the overlap to the user instead of opening a PR. Sweep by the exact key name (e.g. `RemovableDiskDenyWriteAccess`), not just by instruction wording.

Sharing a payload type or DDM declaration type alone is NOT duplicate coverage — the suite deliberately has multiple cases per declaration (e.g. `ddm-beta-enroll` and `ddm-beta-block` both use `softwareupdate.settings`; the two intelligence cases share `intelligence.settings`). A new setting inside an already-covered declaration or payload still needs its own case.

## Step 3: Verify the mechanism against vendor docs

Do NOT write assertions from memory — a plausible-but-wrong key produces a test that enforces the wrong output forever. Confirm the exact key names, casing, value types, and allowed values:

- **csp**: Microsoft CSP reference (`learn.microsoft.com/en-us/windows/client-management/mdm/`) — exact OMA-URI node path, `Format`, allowed values.
- **mobileconfig**: Apple's per-payload YAML schemas (`github.com/apple/device-management`, `mdm/profiles/` — e.g. `com.apple.applicationaccess.yaml`) — `PayloadType`, key names with exact casing, value types, allowed values. The rendered docs (`developer.apple.com/documentation/devicemanagement/profile-specific-payload-keys`) cover the same payloads but are JS-rendered; the YAML is easier to fetch and is the source the docs are built from.
- **ddm**: Apple's declarative device management schemas (same repo, `declarative/declarations/`) — declaration `Type` and payload keys.

Keep the URL(s) you verified against — they go in the PR body so the reviewer can check the assertions without redoing the research.

## Step 4: Write the case

Read the `CASES` comment at the top of the script first — it documents the assertion design. Then follow the file's conventions:

- `id`: `<profileType>-<short-slug>`, unique across the file.
- `instructions`: phrased the way an admin would type it, matching the tone of neighboring cases. Don't name the platform — the profile type implies it.
- `expect`: substring assertions only (compared with all whitespace stripped):
  - `mustContain` / `mustNotContain`: raw substrings.
  - `mustContainElement` / `mustNotContainElement`: `[tag, value]` pairs, e.g. `['Format', 'int']` or `['key', 'autohide']` — XML only (csp and mobileconfig). They compile to `<tag>value</tag>` regexes, so in a DDM case they never match the JSON output and a `mustNotContainElement` silently passes. DDM cases express keys and values as raw `mustContain` / `mustNotContain` substrings like `'"MinorPeriodInDays":30'`, following the existing DDM cases.
  - Bind a value to its key as one adjacent-pair substring — whitespace stripping makes `'<key>allowBookstore</key><true/>'` work. Asserting the key and the value separately lets a value elsewhere in the profile satisfy the check.
  - Assert against the tempting wrong answers too: the lookalike key that doesn't do what the instruction asks, wrong casing, `bool` where the CSP wants `int`, an inverted value.
- `readByEye`: only for properties assertions can't express (one dict per payload domain, distinct PayloadUUIDs, single-line CDATA). Omit otherwise.
- Do NOT set `canary: true` — canaries are a curated set of exact-substring sentinel cases, not a flag for new proposals.
- Append the case to the end of its profile-type section (`CSP` / `MOBILECONFIG` / `DDM`).

## Step 5: Validate

Always: `node --check website/scripts/test-llm-generated-configuration-profile.js`

If a website dev environment with an Anthropic API key is available, run the case for real (it spends API money — a few cents per run):

```
cd website && sails run test-llm-generated-configuration-profile --caseId=<id> --parallelTests=3
```

Skipping this when the environment isn't available is fine — but the PR body must say whether the case was run against the live model or not.

## Step 6: Open the PR

1. Commit with a subject like `Website: Add profile generator test case for <thing>`.
2. Push and open a PR against `main` with the body built from `.github/pull_request_template.md` (the `check-pr-template` CI check fails freeform descriptions).
3. The body must include: what the case asserts and why, the vendor doc URL(s) from Step 3, whether the case was run against the live model, and the `sails run … --caseId=<id> --parallelTests=5` command for the reviewer.
