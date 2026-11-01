---
name: fleet-gitops-export
description: Export a live Fleet instance into a ready-to-use GitOps repository laid out the way `fleetctl new` scaffolds it (default.yml, fleets/*.yml, labels/, platforms/[os]/[kind]/), with secrets replaced by environment variables and a read-only `fleetctl gitops --dry-run` to prove the repo matches Fleet. Use when someone wants to migrate Fleet settings configured in the UI or API to GitOps, export Fleet to a GitOps repo, turn UI settings into YAML, adopt GitOps on an existing Fleet, or replace `fleetctl generate-gitops`, even if they never say "export". It builds a new repo; to edit or validate an existing GitOps repo, use the fleet-gitops skill instead.
compatibility: Needs a shell, Python 3.9+, fleetctl 4.84.0+ logged in to Fleet as a global admin, and network access to the Fleet server.
allowed-tools: Read, Grep, Glob, Edit, Write, Bash(python3 ${CLAUDE_SKILL_DIR}/scripts/gitops_export.py *), Bash(fleetctl --version), Bash(fleetctl new:*), Bash(fleetctl config get:*), Bash(ls:*), Bash(find:*), Bash(cat:*), Bash(git init:*), Bash(git status:*)
effort: high
---

# Export Fleet to GitOps

This skill turns the Fleet instance that `fleetctl` is logged in to into a GitOps repo: it runs `fleetctl new` for the layout, reads every setting, fleet, profile, script, policy, report, label, and software item through Fleet's API, writes each one where the scaffold's files expect it, and validates that applying the repo would change nothing.

The scripts in `scripts/` (next to this file) do the deterministic work. Your job is the workflow, the conversation with the user, and the judgment calls listed below. Run the scripts with `python3 <skill folder>/scripts/gitops_export.py <command> --dir <target>`, where the skill folder is the absolute path of the directory containing this file. On Windows, use `py -3` (or `python`) where this file says `python3`. [README.md](README.md) is the same workflow written for people; point a user who's new to `fleetctl` or GitOps there, for example for logging in or setting up CI.

## Ground rules

- **Read-only against Fleet.** Every API call is a GET through `fleetctl api`; the only `fleetctl gitops` call is `--dry-run`, hardcoded in the scripts. Never run `fleetctl gitops` without `--dry-run`, never call a write endpoint, and never suggest the user apply until they've reviewed the repo. Applying is CI's job after a human merges.
- **No secret values on disk.** Secrets become `$VARIABLE` references. The dry run fetches the values it can from Fleet into the `fleetctl` process's environment only; the user puts the rest in the repo's `.env`, which git ignores. Don't put a secret in `plan.json`, a YAML file, a commit, or your reply, even if the user pastes one, and don't open, print, or edit `.env`.
- **Confirm the server.** Preflight prints the Fleet URL `fleetctl` points at. Show it to the user and get a yes before exporting. If they're testing the skill, it must be a dev or staging instance, not production. If nobody can answer (a non-interactive run), a Fleet URL and target directory named in the task count as the yes when they match what preflight prints; otherwise stop after preflight and report what it found.
- **Don't write into a non-empty directory** unless the user explicitly agrees; only then pass `--force`.

## Running the commands

- `export` and `validate` can run for several minutes on a large instance: `export` makes a request for every profile, script, and software title, and `validate` runs the validator and the dry run. Give them the longest command timeout your agent allows, or run them in the background and wait for them to finish. Every step is safe to re-run.
- Work files go in a temp folder outside the repo, which each command prints. If a later step says that folder doesn't exist (a new session, a reboot), pass `--work <folder>` to every command with the folder an earlier command printed, if it's still there. Otherwise start again at step 2, which needs the user's OK to reuse the directory.
- If your commands run in a sandbox, `fleetctl` needs network access to the Fleet server, and the target directory must be writable.

## Workflow

### 1. Preflight

Ask for the target directory and the organization name if the user didn't give them (offer the org name preflight reports from Fleet as the default). Then:

```bash
python3 <skill folder>/scripts/gitops_export.py preflight --dir <target> [--context <ctx> | --config <file>]
```

It checks that fleetctl has `fleetctl new` (4.84.0 or later), that it's logged in, that the user is a global admin, which server and version it reaches, the license tier, MDM status, and Fleet's GitOps-mode exceptions, and that the target is empty. Fix every FAIL before going on. A version WARN means fleetctl and the server differ; suggest installing the fleetctl that matches the server (CI does), since a mismatch can hide or invent dry-run differences. Connection flags are remembered for the directory.

#### Set up fleetctl

Many users haven't set up `fleetctl` yet. Walk them through it one FAIL at a time instead of stopping, and run preflight again after each fix:

- **`FAIL install fleetctl`**: ask before installing anything. With the user's OK, run the `npm install -g fleetctl` command the FAIL names (it includes the server's version once preflight knows it). If `npm` isn't available, or the install needs `sudo`, give the user the command or [the install guide](https://fleetdm.com/guides/fleetctl#installing-fleetctl) to run themselves.
- **`FAIL set the Fleet address`**: ask for the URL they open Fleet at, then run `fleetctl config set --address <url>` (with the same `--context` or `--config` flags, if any).
- **`FAIL log in to Fleet`**: the user does this step in their own terminal, because it takes their password or an API token. Give them the command from the FAIL: `fleetctl login`, which prompts for email and password, or, with SSO or two-factor authentication, `fleetctl config set --token <token>` with a token from their account page in Fleet. Never ask for the password or token in chat, and don't run `fleetctl login` yourself. If they paste a secret anyway, don't repeat it or put it in a command; ask them to run the command themselves.
- **`FAIL log in as a global admin`**: the export needs a global admin's view of Fleet. They log in again (as above) as a user with the global admin role.
- **`FAIL reach Fleet`**: check the address with them. If `fleetctl` works in their own terminal, your commands are sandboxed without network access; allow it for these commands.

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
- **Fleet-maintained app scripts:** Fleet supplies these apps' install and uninstall scripts and replaces them on each app update unless an admin edited them, but the API doesn't say which were edited. The planner leaves out a stored script that's close to the app's current Fleet script (an older copy Fleet hasn't replaced yet) and lists those in one `REVIEW` line. Ask the user whether anyone edited any of them in Fleet; if so, re-run `export` with `--keep-fma-scripts <slug>[,<slug>...]` (or `all`) so those are kept. A script that differs a lot is kept without asking.
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

Before running it, ask the user to copy `.env.example` to `.env` in the repo and fill in the `FLEET_SECRET_*` variables the plan lists (custom variables stored in Fleet, whose values the API never returns) and any masked integration secrets they want checked for real. They edit the file themselves, so the values never pass through you. `validate` reads `.env` (or `--env-file <file>`), then the environment, and fetches everything else from Fleet itself. Entries left empty get a placeholder or Fleet's mask, and the output says which. Then it:

1. runs the scaffold's own `fleet-gitops` validator (static checks; its FAILs are reported but don't fail the export, because some come from content as it exists in Fleet),
2. parses every YAML file and checks that every reference exists and no file is orphaned,
3. reconciles the repo with Fleet item by item and field by field: policy, report, and label definitions against the values Fleet returned, scripts, profiles, activations, and DDM assets by content hash, and software by package URL (hash when there's no URL), Fleet-maintained app slug, or App Store ID with its self-service, category, and label settings,
4. runs `fleetctl gitops -f default.yml -f fleets/*.yml --delete-other-fleets --dry-run` with secrets in the process environment.

The export passes when references and reconcile are clean, the dry run exits 0, and it prints no deletion, creation, or move. Treat any of those as an export bug or a gap: find which, fix the plan or explain the gap, and re-run. Don't edit generated YAML by hand to make a check pass without understanding why it failed. Known cases:

- **`unknown key "org_settings.yara_rules.[0].path"`**: a current fleetctl bug; GitOps can't represent YARA rules until it's fixed. Report it; don't hand-edit around it.
- **Validator FAILs about `install_software` or labels with GitOps exceptions on**: the validator doesn't know software or labels are managed in the UI; they're expected.
- **Dry-run notes about placeholders or masks**: the value wasn't available, so that file or integration wasn't checked with its real value. Say so.

### 7. Report

```bash
python3 <skill folder>/scripts/gitops_export.py report --dir <target>
```

When the user is done, run `clean --dir <target>` to delete the work folder: it holds copies of profiles and scripts as Fleet stores them.

Relay the report in your own words, without pasting it whole: what was exported for each fleet; anything that couldn't be represented and why; every environment variable to set (from the table); the placement choices you made in step 4; the validation results, including that the dry run alone can't see profile or script changes; and the exact command to rerun the dry run. Suggest `git init` and a first commit, and point to the README's CI setup (`FLEET_URL`, `FLEET_API_TOKEN`, then the variables). Before they turn on the workflow's apply step, recommend one more dry run from CI.

## Things GitOps can't carry

Report these as gaps, never as failures you worked around (details in [references/mapping.md](references/mapping.md)):

- Bootstrap packages (Fleet doesn't keep the download URL; leaving the key out keeps the package).
- Values Fleet only returns masked (Jira and Zendesk tokens, Google service account keys, Apple account provisioning client secret, EJBCA client certificates) and `FLEET_SECRET_*` custom variable values. They become environment variables the user fills in.
- Fleet-level Jira and Zendesk settings, users and roles, ABM/VPP/APNs tokens and certificates, SMTP settings, GitOps-mode settings.
- Packages uploaded without a URL: pinned by `hash_sha256` only, so they can't be recreated on another Fleet instance.
- Hardcoded credentials inside profiles or scripts as they exist in Fleet (a Wi-Fi password in a `.mobileconfig`, say). They're exported as-is, because changing them would change what Fleet deploys. If you see one, tell the user and suggest moving it into a `$FLEET_SECRET_` variable. Enroll secrets are the exception the export can recognize: in a profile or DDM asset it writes the enroll secret's variable (Fleet stores those files after `fleetctl` expanded it), and in a script, which `fleetctl` doesn't expand, it prints a `REVIEW` line.

Secrets and variable names are in [references/secrets.md](references/secrets.md); how this export compares with `fleetctl generate-gitops` is in [references/parity-checklist.md](references/parity-checklist.md).

## Troubleshooting

- **`refusing to call …: not an allowlisted read-only endpoint`**: a script tried a path outside its allowlist. That's a bug in the skill; don't widen the allowlist to a write endpoint.
- **`can't reach Fleet` although `fleetctl` works in the user's terminal**: your sandbox is probably blocking network access. Allow it for these commands instead of asking the user to log in again.
- **A step says a work file doesn't exist**: the work folder from an earlier step is gone; see "Running the commands".
- **400 errors reading profiles or setup experience**: MDM for that platform isn't turned on; those sections are skipped.
- **Re-running from scratch**: delete the target directory only with the user's OK, then start again at step 2. `place` alone can be re-run any number of times.
- **No YAML parser**: install PyYAML (`pip install pyyaml`) or use a machine with Ruby; without either, step 6 skips parsing, reference, and reconcile checks and relies on the dry run.
