---
name: fleet-gitops-export
description: Export a live Fleet instance into a ready-to-use GitOps repository laid out the way `fleetctl new` scaffolds it (default.yml, fleets/*.yml, labels/, platforms/<os>/<kind>/), with secrets replaced by environment variables and a read-only `fleetctl gitops --dry-run` to prove the repo matches Fleet. Use when someone wants to migrate Fleet settings configured in the UI or API to GitOps, export Fleet to a GitOps repo, turn UI settings into YAML, adopt GitOps on an existing Fleet, or replace `fleetctl generate-gitops`, even if they never say "export". It builds a new repo; to edit or validate an existing GitOps repo, use the fleet-gitops skill instead.
compatibility: Needs a shell, Python 3.9+, fleetctl 4.84.0+ logged in to Fleet as a global admin, and network access to the Fleet server.
---

# Export Fleet to GitOps

This is the cross-agent entry point (Codex, Cursor, GitHub Copilot, Gemini CLI, Kilo Code, and other agents that read `.agents/skills/`) for the `fleet-gitops-export` skill. The skill lives at `.claude/skills/fleet-gitops-export/` in the root of this repository, with its `scripts/` and `references/` folders, so every agent follows the same instructions.

Read `.claude/skills/fleet-gitops-export/SKILL.md` now and follow it exactly as if it were this file. Its relative references (`scripts/...`, `references/...`) resolve from that folder.
