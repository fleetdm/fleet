# fleet-gitops-export

An AI agent skill that turns a live Fleet instance into a GitOps repository laid out the way `fleetctl new` scaffolds it, then proves with a read-only `fleetctl gitops --dry-run` that the repository matches Fleet. It's meant to replace `fleetctl generate-gitops`; [references/parity-checklist.md](references/parity-checklist.md) tracks where the two stand.

Ask your agent something like "migrate our Fleet config to GitOps" or "export Fleet to a GitOps repo".

## Requirements

- `fleetctl` 4.84.0 or later, logged in to the Fleet instance as a global admin (`fleetctl login`, or `fleetctl config set --address … --token …`). Ideally the same version as the server.
- Python 3.9 or later. PyYAML or Ruby enables the YAML parsing and repo-vs-Fleet checks.

## Use it with your agent

The skill follows the [Agent Skills](https://agentskills.io) format: `SKILL.md` plus `scripts/` and `references/`. The scripts are plain Python and call nothing but `fleetctl`, so any agent that can run shell commands can use them.

- **In this repository:** Claude Code finds it in `.claude/skills/`. Codex, Cursor, GitHub Copilot, Gemini CLI, and Kilo Code find the pointer in `.agents/skills/fleet-gitops-export/`, which hands off to this folder.
- **Anywhere else:** copy this whole folder into your agent's skills folder, for example `~/.claude/skills/fleet-gitops-export/` for Claude Code, or `.agents/skills/fleet-gitops-export/` in the directory where you start Codex, Copilot, Gemini CLI, or Cursor. Some agents ask you to trust a skill before loading it.

## Run the scripts yourself

```bash
python3 scripts/gitops_export.py all --dir ~/fleet-gitops --org-name "Acme, Inc."
```

`all` runs `preflight`, `scaffold`, `export`, `place`, `validate`, and `report` in a row, and exits non-zero when validation fails. Run them one at a time to review the placement plan before anything is written. `clean` deletes the work folder afterward. See [SKILL.md](SKILL.md) for each step.

## Tests

```bash
python3 -m unittest discover -s scripts/tests
```

The tests run offline. Inside the Fleet repository they also check the templates `fleetctl new` ships and the list of deprecated GitOps keys.
