---
name: fleet-gitops
description: Make changes to a Fleet GitOps repository (default.yml, fleets/*.yml or teams/*.yml, and the policies, reports, labels, scripts, software, configuration profiles, and DDM declarations they reference). Use this whenever a task touches Fleet-managed devices through YAML, even if the user never says "GitOps" - adding, scoping, or updating software (Fleet-maintained apps, custom packages, App Store or Google Play apps), writing a policy or report, remediating a CVE, creating or editing a .mobileconfig, DDM .json, Windows CSP .xml, or Android policy .json, changing labels, controls, OS update deadlines, or org settings, or reviewing a GitOps pull request. Covers the layout rules fleetctl enforces, validation against upstream schemas (osquery tables, Apple, Microsoft, Google), and fleetctl dry runs.
allowed-tools: Read, Grep, Glob, Edit, Write, WebFetch, WebSearch, Bash(python3:*), Bash(plutil:*), Bash(xmllint:*), Bash(jq:*), Bash(contour:*), Bash(git diff:*), Bash(git status:*), Bash(git log:*), Bash(ls:*), Bash(find:*)
effort: high
---

# Fleet GitOps

Task: $ARGUMENTS

The `fleet-gitops` skill lives at `cmd/fleetctl/fleetctl/templates/new/.agents/skills/fleet-gitops/`, the copy `fleetctl new` scaffolds into customer repositories. Keeping one copy there means this repository and customers' repositories follow the same instructions.

Read `cmd/fleetctl/fleetctl/templates/new/.agents/skills/fleet-gitops/SKILL.md` now and follow it exactly as if it were this file. Its relative references (`references/...`, `scripts/validate.py`) resolve from that folder.
