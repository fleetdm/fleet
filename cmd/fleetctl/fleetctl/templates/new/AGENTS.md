# AGENTS.md

Guidance for AI coding agents (Codex, Cursor, GitHub Copilot, Gemini CLI, Kilo Code, Claude Code, and others) working in this repository. Claude Code reads it through `CLAUDE.md`, Gemini CLI through `GEMINI.md`.

## About this repository

This is a Fleet GitOps repository: the git-backed source of truth for configuring computers (macOS, Windows, Linux, iOS, iPadOS, Android, ChromeOS) managed by [Fleet](https://fleetdm.com). There's no app to build here. A commit to the default branch is a deploy: CI runs `fleetctl gitops` to apply the YAML in this repo to the live Fleet instance, and pull requests get a dry run only. See `.github/workflows/workflow.yml` (GitHub) or `.gitlab-ci.yml` (GitLab). Never run `fleetctl gitops` without `--dry-run` yourself; applying is CI's job after a human merges.

## The fleet-gitops skill

Before changing anything that reaches devices (policies, reports, labels, scripts, software, configuration or declaration profiles, OS update or enrollment controls, org settings), read `.agents/skills/fleet-gitops/SKILL.md` and follow it. It explains the layout rules `fleetctl gitops` enforces, points at per-platform references under `.agents/skills/fleet-gitops/references/`, and ships `scripts/validate.py`, an offline checker to run from the repository root before a dry run:

```bash
python3 .agents/skills/fleet-gitops/scripts/validate.py
```

Agents that implement the Agent Skills standard discover the skill automatically from `.agents/skills/`; Claude Code finds it through `.claude/skills/fleet-gitops/`. Some agents ask you to trust the repository's skills before loading them.

## Layout

- `default.yml`: org-wide settings (org info, server URL, SSO, Apple Business Manager and VPP, migration and Windows MDM controls) and the `labels:` registration.
- `fleets/*.yml`: one file per fleet (a named group of hosts, for example "💻 Workstations"). Each fleet's controls, software, policies, and reports live in its own file.
- `labels/*.yml`: reusable host groupings (dynamic queries, manual host lists, or host-vitals criteria) used to scope profiles, software, and policies to a subset of hosts.
- `platforms/<macos|windows|linux|ios|ipados|android|all>/`: the actual artifacts, one folder per kind:
  - `policies/`: compliance policies (osquery-based checks with a resolution)
  - `reports/`: data collection reports
  - `software/`: custom package definitions
  - `scripts/`: `.sh` and `.ps1` scripts available for automations and manual runs
  - `configuration-profiles/`: `.mobileconfig` (Apple), `.xml` (Windows), or `.json` (Android) profiles
  - `declaration-profiles/`: Apple DDM declarations (`.json`)
  - `enrollment-profiles/`: Apple automated enrollment profiles
  - `commands/`: MDM commands
  - `managed-app-configurations/`: Android per-app managed configurations
  - `icons/`: custom software icons (`platforms/all` only)

A fleet's YAML file pulls files from these platform folders with glob patterns, for example `paths: ../platforms/macos/policies/*.yml`. Dropping a new file into the right folder is usually enough; check the fleet's YAML to confirm which globs apply before assuming a file needs to be wired up by hand.

## Making changes

- Add a policy, report, profile, or script by creating a file in the matching `platforms/<platform>/<kind>/` folder. Follow the format of an existing file in that folder.
- For software, prefer a [Fleet-maintained app](https://github.com/fleetdm/fleet/tree/main/ee/maintained-apps) (`fleet_maintained_apps` in a fleet's `software:` section) over a custom package when one exists.
- To scope something to part of the fleet, reference a label from `labels/*.yml` with `labels_include_any` / `labels_include_all` rather than inventing a new query inline.
- Validate keys and values against [Fleet's GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files) before inventing one; an unrecognized key fails the dry run.
- osquery queries (in policies and reports) must use tables and columns that exist in the [Fleet osquery schema](https://fleetdm.com/tables) for the target platform.

## Validating changes

Run the skill's validator (above), then dry-run before committing, the same way CI does for pull requests:

```bash
fleetctl gitops -f default.yml $(for f in fleets/*.yml; do echo -f "$f"; done) --dry-run
```

## Terminology

- A "fleet" (this repo's `fleets/*.yml`) was formerly called a "team." The product and this repo use "fleet" now, but Fleet's REST API and some `fleetctl` flags still use `team` / `team_id`; expect that name when working with the API directly.
- A "report" is what was formerly called a "query" in Fleet's product. "Query" now refers only to the SQL itself, one part of a report or policy, not a top-level concept with its own folder.

## Apple profile authoring (optional)

[contour](https://github.com/macadmins/contour) is a community CLI that generates and validates `.mobileconfig` and DDM JSON files against Apple's schema. If `.contour/config.toml` exists, it's set up for this repo; run `contour help-ai` for its command index. It isn't part of Fleet and isn't installed by default.

## References

- GitOps YAML reference: https://fleetdm.com/docs/configuration/yaml-files
- Fleet REST API reference: https://fleetdm.com/docs/rest-api/rest-api
- Fleet-maintained app catalog: https://github.com/fleetdm/fleet/tree/main/ee/maintained-apps
- osquery table reference: https://fleetdm.com/tables
- Vetted example profiles, policies, scripts, and labels: https://github.com/fleetdm/fleet/tree/main/docs/solutions
