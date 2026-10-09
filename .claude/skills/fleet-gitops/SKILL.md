---
name: fleet-gitops
description: Make changes to a Fleet GitOps repository (default.yml, fleets/*.yml or teams/*.yml, and the policies, reports, labels, scripts, software, configuration profiles, and DDM declarations they reference). Use this whenever a task touches Fleet-managed devices through YAML, even if the user never says "GitOps" - adding, scoping, or updating software (Fleet-maintained apps, custom packages, App Store or Google Play apps), writing a policy or report, remediating a CVE, creating or editing a .mobileconfig, DDM .json, Windows CSP .xml, or Android policy .json, changing labels, controls, OS update deadlines, or org settings, or reviewing a GitOps pull request. Covers the layout rules fleetctl enforces, validation against upstream schemas (osquery tables, Apple, Microsoft, Google), and fleetctl dry runs.
allowed-tools: Read, Grep, Glob, Edit, Write, WebFetch, WebSearch, Bash(python3:*), Bash(plutil:*), Bash(xmllint:*), Bash(jq:*), Bash(contour:*), Bash(git diff:*), Bash(git status:*), Bash(git log:*), Bash(ls:*), Bash(find:*)
effort: high
---

# Fleet GitOps

Task: $ARGUMENTS

A Fleet GitOps repo is the source of truth for a live Fleet instance. CI runs `fleetctl gitops` against the YAML on every merge to the default branch, so a merged change reaches real laptops, phones, and servers within minutes; pull requests get a dry run only. Work as if every file you touch ships to devices: match what the repo already does, validate against the real schema rather than memory, and never run `fleetctl gitops` without `--dry-run`. Applying is CI's job, after a human merges.

The reference for every key is https://fleetdm.com/docs/configuration/yaml-files. Fetch the relevant section whenever you're unsure what a section accepts; `fleetctl gitops` rejects unknown keys, and a key it accepts in the wrong place is often silently ignored.

## Orient before writing

1. **Find the GitOps root**: the directory containing `default.yml` (org-wide settings plus global labels, policies, and reports). Fleet files are `fleets/*.yml` (older repos: `teams/*.yml`), one per fleet of hosts, plus `fleets/unassigned.yml` for hosts in no fleet. Everything else is an artifacts tree, usually `lib/<platform>/<kind>/` or `platforms/<platform>/<kind>/`, sometimes with `labels/` beside it.
2. **Read `default.yml` and the fleet file(s) you'll change, end to end.** They show the conventions you must match: inline items vs `path:` references, `paths:` globs, label names, how entries are commented, how files are named. Copy the nearest existing file of the same kind as your template and keep its field order. Look for an existing file that already does part of the job before creating a new one; two profiles that set the same key, or two policies that check the same thing, are a reviewer's problem.
3. **Check whether the folder you're writing into is already covered by a `paths:` glob** in the fleet file. If it is, dropping the file in is enough. If not, add a `path:` entry. A file nobody references does nothing.
4. **Confirm the scope.** The user names it ("workstations", "all Macs", "engineering"); you map it to fleet files plus labels. When a request could mean more than one fleet, say which you picked and why rather than guessing silently.

## Where things go

| You need to… | Mechanism | Lives in | Read first |
|---|---|---|---|
| Check a condition and track compliance, optionally remediate | `policies:` item (osquery SQL plus `resolution`), with `run_script`, `install_software`, or `resend_configuration_profile` | fleet file, or `default.yml` for a global policy without automations | [references/queries.md](references/queries.md) |
| Collect data on a schedule | `reports:` item | `default.yml` or fleet file | queries.md |
| Target a subset of hosts | `labels:` (dynamic SQL, manual host list, or host-vitals criteria), then `labels_include_any` / `labels_include_all` / `labels_exclude_any` on the item | `default.yml` (global) or fleet file | queries.md |
| Install, update, or make software self-service | `software:` with `fleet_maintained_apps` (check the catalog first), `packages`, or `app_store_apps`; a `type: patch` policy keeps it current | fleet file only | [references/software.md](references/software.md) |
| Fix a CVE | identify the affected title, then the software row above | fleet file | software.md |
| Configure macOS, iOS, or iPadOS | `.mobileconfig` or DDM `.json` listed under `controls.apple_settings.configuration_profiles` | fleet file (`default.yml` covers unassigned hosts) | [references/profiles-apple.md](references/profiles-apple.md) |
| Configure Windows | SyncML `.xml` listed under `controls.windows_settings.configuration_profiles` | same | [references/profiles-windows.md](references/profiles-windows.md) |
| Configure Android | Android Management API Policy `.json` listed under `controls.android_settings.configuration_profiles` | same | [references/profiles-android.md](references/profiles-android.md) |
| Run a script from a policy or on demand | script file listed in `controls.scripts` | fleet file | queries.md |
| OS update deadlines, disk encryption, setup experience, enrollment | `controls` keys (`macos_updates`, `windows_updates`, `enable_disk_encryption`, `setup_experience`, …) | fleet file; a few keys are `default.yml` only | yaml-files doc, `controls` |
| Org name, SSO, Apple Business Manager, VPP, webhooks, integrations | `org_settings` | `default.yml` only | yaml-files doc, `org_settings` |

Software management and label-scoped profiles and software need Fleet Premium; on Fleet Free the `software:` section is skipped entirely.

Before authoring a profile, policy, script, or label from scratch, look for a vetted example in Fleet's solutions library at https://github.com/fleetdm/fleet/tree/main/docs/solutions: CIS benchmark profiles, policies, and scripts for macOS 13 to 15 and Windows 10 and 11, Android and iOS profiles, DDM declarations, rollout-ring labels, Windows Update CSPs, and platform scripts. Fetch the raw file, copy it into this repo's layout and naming, adapt it, and say in your report where it came from.

## Rules the repo can't show you

These are enforced by `fleetctl gitops` or by Fleet, and are the usual reasons a reasonable-looking change fails the dry run or deploys and does nothing:

- **Omitting a key clears it.** A top-level key missing from a file (`policies`, `reports`, `controls`, `agent_options`, …) means "delete everything in this section," not "leave it alone"; `custom_host_vitals` omitted deletes every vital. Edit sections in place and never drop a key to skip it.
- **Paths are relative to the file that contains them.** From `default.yml` that's `./lib/...`; from `fleets/x.yml` it's `../lib/...`; a policy's `run_script.path` is relative to the policy file. `path:` names one file and can't contain `*`, `?`, `[`, or `{`; `paths:` is a glob, and every option on that entry (labels, `self_service`, …) applies to all matched files.
- **Labels must exist before they're referenced.** Every name in a `labels_*` list must be declared in a `labels:` section (global in `default.yml`, or in that fleet's file), unless this instance manages labels in the UI, in which case the label must already exist there. Built-in labels (`macOS`, `MS Windows`, `All Linux`, `iOS`, …) can't be referenced; use `platform` instead. An item takes one of `labels_include_all` or `labels_include_any`, may add `labels_exclude_any`, and can't list a label in both. Label keys go on each item, not on the `policies:` list. Renaming a label deletes and recreates it, so membership is empty until hosts check in.
- **Automations are fleet-scoped.** `run_script`, `install_software`, `resend_configuration_profile`, and `calendar_events_enabled` only work on policies in a fleet file (or `unassigned.yml`), and the script, package, Fleet-maintained app, or profile they point at must be defined in that same fleet's `controls.scripts`, `software`, or `controls` section.
- **File roles are fixed.** `org_settings` only in `default.yml`; `software:` only in fleet files; `agent_options` and `reports` not in `unassigned.yml`; each fleet file's `name` must be unique.
- **Quote anything YAML would mangle.** `version: "1.2"`, `app_store_id: "361285480"`, `auto_update_window_start: "21:00"` (unquoted `21:00` parses as the integer 75600), and fleet or label names containing emoji or colons.
- **Secrets never land in git.** The repo already uses `$VARIABLE` placeholders that CI substitutes (enroll secrets, webhook URLs, API keys). Follow that pattern and tell the user which new CI variable to add. Inside scripts and profiles, `$FLEET_SECRET_NAME` is a secret stored in Fleet and `$FLEET_VAR_HOST_…` is a per-host value Fleet fills in at delivery. A dry run skips any profile containing `$FLEET_SECRET_`, so say so when it applies.
- **Naming.** The product says "fleet" where the API and some keys still say `team`, and "report" where older YAML says `queries`. Use the new words in prose and new files; keep the existing keys in files you edit.

## Validate before you report

Validation is what makes an agent-written change safe to merge. Run these in order and report exactly what ran:

1. **Static checks.** From the GitOps root (where `default.yml` is), run the validator that ships next to this file, giving it the path of this skill's `scripts/validate.py` (for example `.claude/skills/fleet-gitops/scripts/validate.py`):
   ```bash
   python3 .claude/skills/fleet-gitops/scripts/validate.py
   ```
   It parses every YAML file, resolves every `path:` and `paths:`, checks label references, policy, report, and label fields, osquery table names and platforms against Fleet's schema, Fleet-maintained app slugs against the catalog, software quoting rules, and the structure of every `.mobileconfig`, DDM `.json`, Windows `.xml`, and Android `.json`. Fix every FAIL; carry each WARN into your report. `--help` lists options such as `--table <name>` for an osquery schema lookup. If the script isn't present, do the same checks by hand with the commands in the reference files.
2. **Schema checks for profiles**, per the platform reference: `contour profile validate --strict` and `contour profile ddm validate` for Apple when contour is installed, the DDF node lookup for Windows, the Policy discovery document for Android. These catch invented keys and wrong value types that static checks can't.
3. **Server dry run**, if `fleetctl` is installed and configured against the right instance, exactly the way CI runs it, from the GitOps root:
   ```bash
   fleetctl gitops -f default.yml $(for f in fleets/*.yml; do echo -f "$f"; done) --dry-run
   ```
   (Use `teams/*.yml` in older repos.) This is the same gate the pull request will hit. If fleetctl isn't configured, say that the PR's CI dry run is the remaining gate; don't ask the user for an API token, and never run the command without `--dry-run`.

Never describe a validation you didn't run, and quote the relevant output when something fails. A clean static pass plus a dry run is strong evidence; a profile that only looks right is not.

## Reviewing someone else's change

Run the validator on the branch, read the diff, and check what tools can't: does the change address the request rather than its wording, is the scope right (fleets and labels), does it overlap or conflict with an existing profile or policy, are identifiers and names unique, did a secret or an environment-specific URL get hardcoded, and for Windows, do the target hosts meet the node's edition and build applicability.

## Report

End with a short summary a reviewer can act on: which files changed and why; which fleets and labels the change targets; each validation you ran with its result; what wasn't validated and why (no fleetctl config, profile skipped by dry run, enum value only checkable on device); and any CI variable the user must add. When opening a pull request, put the same information in the description.
