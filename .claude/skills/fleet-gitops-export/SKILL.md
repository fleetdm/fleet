---
name: fleet-gitops-export
description: Export a live Fleet instance into a ready-to-use GitOps repository laid out the way `fleetctl new` scaffolds it (default.yml, fleets/*.yml, labels/, platforms/<os>/<kind>/), with secrets replaced by environment variables and a read-only `fleetctl gitops --dry-run` to prove the repo matches Fleet. Use when someone wants to migrate Fleet settings configured in the UI or API to GitOps, export Fleet to a GitOps repo, turn UI settings into YAML, adopt GitOps on an existing Fleet, or replace `fleetctl generate-gitops`, even if they never say "export".
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(python3:*), Bash(fleetctl --version), Bash(fleetctl new:*), Bash(fleetctl config get:*), Bash(ls:*), Bash(find:*), Bash(cat:*), Bash(git init:*), Bash(git status:*)
effort: high
---

# Export Fleet to GitOps

Task: $ARGUMENTS

This skill turns the Fleet instance that `fleetctl` is logged in to into a GitOps repo: it runs `fleetctl new` for the layout, reads every setting, fleet, profile, script, policy, report, label, and software item through Fleet's API, writes each one where the scaffold's files expect it, and validates that applying the repo would change nothing.

The scripts in `scripts/` (next to this file) do the deterministic work. Your job is the workflow, the conversation with the user, and the judgment calls listed below. Run the scripts with `python3 <skill folder>/scripts/gitops_export.py <command> --dir <target>`, where the skill folder is the directory containing this file.

## Ground rules

- **Read-only against Fleet.** Every API call is a GET through `fleetctl api`; the only `fleetctl gitops` call is `--dry-run`, hardcoded in the scripts. Never run `fleetctl gitops` without `--dry-run`, never call a write endpoint, and never suggest the user apply until they've reviewed the repo. Applying is CI's job after a human merges.
- **No secret values on disk.** Secrets become `$VARIABLE` references. The dry run fetches the values it needs into the `fleetctl` process's environment only. Don't put a secret in `plan.json`, a YAML file, a commit, or your reply, even if the user pastes one.
- **Confirm the server.** Preflight prints the Fleet URL `fleetctl` points at. Show it to the user and get a yes before exporting. If they're testing the skill, it must be a dev or staging instance, not production.
- **Don't write into a non-empty directory** unless the user explicitly agrees; only then pass `--force`.

## Workflow

### 1. Preflight

Ask for the target directory and the organization name if the user didn't give them (offer the org name preflight reports from Fleet as the default). Then:

```bash
python3 <skill folder>/scripts/gitops_export.py preflight --dir <target> [--context <ctx> | --config <file>]
```

It checks that fleetctl has `fleetctl new` (4.84.0 or later), that it's logged in, that the user is a global admin, which server and version it reaches, the license tier, MDM status, and Fleet's GitOps-mode exceptions, and that the target is empty. Fix every FAIL before going on. A version WARN means fleetctl and the server differ; suggest installing the fleetctl that matches the server (CI does), since a mismatch can hide or invent dry-run differences. Connection flags are remembered for the directory.

### 2. Scaffold

```bash
python3 <skill folder>/scripts/gitops_export.py scaffold --dir <target> --org-name "<org>"
```

Runs `fleetctl new` and records what it generated: fleet templates, their `paths:` globs, and the `platforms/<os>/<kind>/` folders. Every destination later comes from this scan, so nothing about the layout is hardcoded.

### 3. Export

```bash
python3 <skill folder>/scripts/gitops_export.py export --dir <target>
```

Reads Fleet and writes `plan.json` to a work folder outside the repo (the command prints where). The plan lists every file with its destination, what goes into each YAML file, and every environment variable. It prints `REVIEW` lines for placements you should check and `GAP` lines for things GitOps can't represent.

### 4. Review the plan (judgment)

Handle each `REVIEW` line before placing. To move a file, edit its `"path"` in `plan.json` (files are keyed by id; every reference to the file follows the id, so nothing else needs to change). Keep the folder pattern `platforms/<os>/<kind>/`.

- **`.sh` and `.py` scripts:** Fleet runs them on macOS and Linux alike and doesn't record which one a script is for. The script guesses from its contents and flags the ones it can't tell. Read them: `launchctl`, `defaults`, `/Library` mean macOS; `apt`, `systemctl`, `/etc/os-release` mean Linux. If the fleet only has hosts of one platform, use that. Two scripts with the same name in one folder can't coexist, so keep fleet subfolders (`scripts/<fleet>/`) the planner created.
- **Apple profiles:** Fleet stores one Apple platform for macOS, iOS, and iPadOS. The planner uses `macos/` unless the fleet looks mobile-only, then `ios/`. Move a profile when its payloads or the fleet's purpose clearly point elsewhere.
- **Fleet files:** each fleet maps to a template file by name (`"💻 Workstations"` fills `fleets/workstations.yml`) or is cloned from the template that fits it (computers or mobile). Check `fleets[].template` for fleets that look wrong, and tell the user which template file each fleet came from.
- **Naming:** files are named from item names (lowercase, dashes, emoji dropped). Scripts and Windows/DDM/Android profiles keep their exact names, because Fleet derives the name from the file name. Don't rename those.

If you change anything, re-run `place` (it starts from the pristine scaffold each time).

### 5. Place

```bash
python3 <skill folder>/scripts/gitops_export.py place --dir <target>
```

Writes every file, fills `default.yml` and the fleet files in place (the templates' guidance comments stay; a commented-out example becomes the real setting when Fleet has one), deletes template examples that don't match anything in Fleet (example labels, the example policy and ADE profile, template fleets with no matching fleet), writes `.env.example`, adds an "Environment variables" section to the README, and adds each variable to the GitHub workflow's `env:` block. `NEW FOLDER` lines name folders the layout didn't have; mention them to the user.

How placement works is in [references/mapping.md](references/mapping.md). The parts worth knowing before you explain the result:

- Platform folders are shared by every fleet. A fleet keeps a template glob (`paths: ../platforms/macos/policies/*.yml`) only when the folder holds exactly that fleet's files; otherwise the glob becomes explicit `path:` entries, so another fleet's files can't leak in. An identical item used by several fleets is written once.
- Controls for hosts in "Unassigned" go in `default.yml`'s `controls:`, because fleetctl accepts controls in `default.yml` or `fleets/unassigned.yml`, not both, and the template keeps global controls there. `fleets/unassigned.yml` holds Unassigned's policies, software, and webhooks.

### 6. Validate

```bash
python3 <skill folder>/scripts/gitops_export.py validate --dir <target>
```

Before running it, ask the user to export any `FLEET_SECRET_*` variables the plan lists (custom variables stored in Fleet, whose values the API never returns) and any masked integration secrets they want checked for real; the rest it fetches from Fleet itself. Then it:

1. runs the scaffold's own `fleet-gitops` validator (static checks; its FAILs are reported but don't fail the export, because some come from content as it exists in Fleet),
2. parses every YAML file and checks that every reference exists and no file is orphaned,
3. reconciles the repo with Fleet item by item (policies, reports, labels, scripts, profiles per file),
4. runs `fleetctl gitops -f default.yml -f fleets/*.yml --delete-other-fleets --dry-run` with secrets in the process environment.

The export passes when references and reconcile are clean, the dry run exits 0, and it prints no deletion, creation, or move. Treat any of those as an export bug or a gap: find which, fix the plan or explain the gap, and re-run. Don't edit generated YAML by hand to make a check pass without understanding why it failed. Known cases:

- **`unknown key "org_settings.yara_rules.[0].path"`**: a current fleetctl bug; GitOps can't represent YARA rules until it's fixed. Report it; don't hand-edit around it.
- **Validator FAILs about `install_software` or labels with GitOps exceptions on**: the validator doesn't know software or labels are managed in the UI; they're expected.
- **Dry-run notes about placeholders or masks**: the value wasn't available, so that file or integration wasn't checked with its real value. Say so.

### 7. Report

```bash
python3 <skill folder>/scripts/gitops_export.py report --dir <target>
```

Relay the report in your own words, without pasting it whole: what was exported for each fleet; anything that couldn't be represented and why; every environment variable to set (from the table); the placement choices you made in step 4; the validation results, including that the dry run alone can't see profile or script changes; and the exact command to rerun the dry run. Suggest `git init` and a first commit, and point to the README's CI setup (`FLEET_URL`, `FLEET_API_TOKEN`, then the variables). Before they turn on the workflow's apply step, recommend one more dry run from CI.

## Things GitOps can't carry

Report these as gaps, never as failures you worked around (details in [references/mapping.md](references/mapping.md)):

- Bootstrap packages (Fleet doesn't keep the download URL; leaving the key out keeps the package).
- Values Fleet only returns masked (Jira and Zendesk tokens, Google service account keys, Apple account provisioning client secret, EJBCA client certificates) and `FLEET_SECRET_*` custom variable values. They become environment variables the user fills in.
- Fleet-level Jira and Zendesk settings, users and roles, ABM/VPP/APNs tokens and certificates, SMTP settings, GitOps-mode settings.
- Packages uploaded without a URL: pinned by `hash_sha256` only, so they can't be recreated on another Fleet instance.
- Hardcoded credentials inside profiles or scripts as they exist in Fleet (a Wi-Fi password in a `.mobileconfig`, say). They're exported as-is, because changing them would change what Fleet deploys. If you see one, tell the user and suggest moving it into a `$FLEET_SECRET_` variable.

Secrets and variable names are in [references/secrets.md](references/secrets.md); how this export compares with `fleetctl generate-gitops` is in [references/parity-checklist.md](references/parity-checklist.md).

## Troubleshooting

- **`refusing to call …: not an allowlisted read-only endpoint`**: a script tried a path outside its allowlist. That's a bug in the skill; don't widen the allowlist to a write endpoint.
- **400 errors reading profiles or setup experience**: MDM for that platform isn't turned on; those sections are skipped.
- **Re-running from scratch**: delete the target directory only with the user's OK, then start again at step 2. `place` alone can be re-run any number of times.
- **No YAML parser**: install PyYAML (`pip install pyyaml`) or use a machine with Ruby; without either, step 6 skips parsing, reference, and reconcile checks and relies on the dry run.
