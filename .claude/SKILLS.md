# Fleet QA — Claude Code Skills Reference

Type `/skill-name` in Claude Code to invoke any skill below.

## Project skills (`.claude/skills/`)

| Skill | Description |
|---|---|
| `/aikido-tickets` | Create GitHub issues in fleetdm/confidential for Aikido pen test findings |
| `/bump-migration` | Bump a DB migration's timestamp when it's older than one already merged to main |
| `/cherry-pick` | Cherry-pick a merged PR into the current RC branch |
| `/command-palette` | Authoring guide for Fleet command palette entries |
| `/content-style` | Write/edit/review public-facing Fleet content following brand and style guidelines |
| `/find-related-tests` | Find test files and functions related to recent git changes; outputs exact `go test` commands |
| `/fix-ci` | Diagnose and fix failing GitHub Actions CI runs |
| `/fleet-gitops` | Help with Fleet GitOps YAML config (queries, profiles, software, DDM) |
| `/fleet-guide-formatting` | Ensure guides under `articles/` follow Fleet's step-by-step guide structure |
| `/lint` | Run linters on recently changed files (Go + JS/TS) |
| `/new-fma` | Add a Fleet-maintained app (FMA) for macOS/Windows |
| `/new-migration` | Scaffold a new Fleet database migration with timestamp, Up function, and test file |
| `/openspec-apply-change` | Implement tasks from an OpenSpec change |
| `/openspec-archive-change` | Archive a completed OpenSpec change |
| `/openspec-explore` | Think through ideas and clarify requirements before proposing a change |
| `/openspec-propose` | Propose a new change with design, specs, and tasks in one step |
| `/project` | Load or initialize a Fleet workstream project context |
| `/push-reference-docs` | Move reference doc updates between release branches when a feature is pushed to a later release |
| `/release-retro` | Format release retro notes into Slack recap post + GitHub issues |
| `/review-pr` | Review a Fleet PR for correctness, Go idioms, SQL safety, and test coverage |
| `/spec-story` | Break down a Fleet GitHub story issue into tasks |
| `/test` | Run tests related to recent changes with correct env vars |
| `/tier-modes` | Guide for Fleet Free / Primo mode UI gating in the frontend |
| `/vuln-triage` | Triage Fleet vulnerability false positives/negatives across NVD, OSV, OVAL, MSRC |
| `/who-blocks-this-pr` | Identify which files still need approval and from whom (CODEOWNERS) |

## Global user skills (`~/.claude/skills/`)

These must be installed individually per user — they are not in this repo.

| Skill | Description |
|---|---|
| `/find-skills` | Discover and install new Claude Code skills |
| `/playwright-best-practices` | Best practices for Playwright tests: POM, CI, mocking, a11y, visual, security, and more |
| `/playwright-cli` | Automate browser interactions and work with Playwright tests |
| `/playwright-generate-test` | Generate a Playwright test from a scenario using Playwright MCP |

## Installing global skills

```bash
# Install the Playwright skills globally so they're available in any project
claude /find-skills playwright
```

Or copy the skill directories from `~/.claude/skills/` on a teammate's machine to your own.

## Tips

- Skills with `.claude:` prefix and no-prefix in the autocomplete list are the same skill — both forms work.
- Run `/find-skills` to search for additional installable skills.
- Project skills live in `.claude/skills/` — commit changes there to share with the team.
