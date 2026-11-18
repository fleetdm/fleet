# AGENTS.md

Instructions for AI coding agents working in this repository. Claude Code users: see `.claude/CLAUDE.md` for the full project guide; the rules below apply to every agent.

## Orbit (fleetd) startup order

- In `orbit/cmd/orbit/orbit.go`, add new code or sub-systems **after** the early update check (`updateRunner.UpdateAction()`). Code that runs before it and crashes or hangs stops devices from auto-updating to a fix, so they need a manual reinstall.

## Opening a pull request

- The PR description MUST start from `.github/pull_request_template.md`. Use that file as the body and fill it in. Do not open a PR with an empty or freeform description.
- Add a `## AI` section to the PR description, between the Testing and Frontend sections, containing an `**AI:** <tool> (<model ID>)` line with the tool you are running in and the exact model ID your environment reports (write `unknown` if it does not).
