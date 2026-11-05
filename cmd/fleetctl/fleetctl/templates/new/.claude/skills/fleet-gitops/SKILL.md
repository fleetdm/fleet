---
name: fleet-gitops
description: Make changes to a Fleet GitOps repository (default.yml, fleets/*.yml or teams/*.yml, and the policies, reports, labels, scripts, software, configuration profiles, and DDM declarations they reference). Use this whenever a task touches Fleet-managed devices through YAML, even if the user never says "GitOps" - adding, scoping, or updating software (Fleet-maintained apps, custom packages, App Store or Google Play apps), writing a policy or report, remediating a CVE, creating or editing a .mobileconfig, DDM .json, Windows CSP .xml, or Android policy .json, changing labels, controls, OS update deadlines, or org settings, or reviewing a GitOps pull request. Covers the layout rules fleetctl enforces, validation against upstream schemas (osquery tables, Apple, Microsoft, Google), and fleetctl dry runs.
allowed-tools: Read, Grep, Glob, Edit, Write, WebFetch, WebSearch, Bash(python3:*), Bash(plutil:*), Bash(xmllint:*), Bash(jq:*), Bash(contour:*), Bash(git diff:*), Bash(git status:*), Bash(git log:*), Bash(ls:*), Bash(find:*)
effort: high
---

# Fleet GitOps

Task: $ARGUMENTS

This is the Claude Code entry point for the `fleet-gitops` skill. The skill itself lives at `.agents/skills/fleet-gitops/` in the root of this repository, the cross-agent location that Codex, Cursor, GitHub Copilot, Gemini CLI, and Kilo Code also read, with its `references/` files and the `scripts/validate.py` checker beside it. Keeping one copy there means every agent follows the same instructions.

Read `.agents/skills/fleet-gitops/SKILL.md` now and follow it exactly as if it were this file. Its relative references (`references/...`, `scripts/validate.py`) resolve from that folder.
