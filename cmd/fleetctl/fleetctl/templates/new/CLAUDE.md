@AGENTS.md

## Claude Code notes

- The repository's agent instructions live in `AGENTS.md`, imported above, so Claude Code and the other agents on the team follow the same rules.
- The `fleet-gitops` skill is registered under `.claude/skills/fleet-gitops/` and hands off to the full skill in `.agents/skills/fleet-gitops/`. Invoke it with `/fleet-gitops`, or let it trigger on its own for GitOps work.
