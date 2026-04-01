# Fleet Claude Code configuration

This directory contains team-shared [Claude Code](https://claude.ai/code) configuration for the Fleet project. Everything here works out of the box with no MCP servers, plugins, or external dependencies required. The full setup adds ~2,700 tokens at startup (under 0.3% of your context window) — rules, skill bodies, and agent bodies only load on demand.

This setup is a starting point. You can customize it by creating `.claude/settings.local.json` (gitignored) to add your own permissions, MCP servers, and plugins. See [Customize your setup](#customize-your-setup) for details.

If you're new to Claude Code, start with the [primer](#claude-code-primer) below. If you already know Claude Code, skip to [what's here](#whats-here).

### Try it on your branch

To test this setup without switching branches, pull the `.claude/` folder into your current working branch:

```bash
# Add the configuration to your branch
git checkout origin/cc-setup-teamwide -- .claude/

# Start a Claude Code session and work normally
claude

# When you're done testing, remove it so it doesn't end up in your PR
git checkout -- .claude/
```

This drops the full setup (rules, skills, agents, hooks, and permissions) into your working tree. Start a new Claude Code session and everything loads automatically. When you're done, the second command reverts `.claude/` to whatever's on your branch.

---

## Claude Code primer

Claude Code is an AI coding assistant that runs in your terminal, VS Code, JetBrains, desktop app, or browser. It reads your codebase, writes code, runs commands, and understands project context through configuration files like the ones in this directory.

### Core concepts

**CLAUDE.md** — Project instructions loaded at session start, like a `.editorconfig` for AI. Claude reads these automatically to understand your project's conventions, architecture, and workflows. There can be multiple: root-level, `.claude/CLAUDE.md`, and user-level `~/.claude/CLAUDE.md`.

**Skills** — Reusable workflows invoked with `/` (e.g., `/test`, `/fix-ci`). Each skill is a `SKILL.md` file with YAML frontmatter that controls when it triggers, which tools it can use, and whether it runs in an isolated context. Skills replace the older `.claude/commands/` format, adding auto-invocation, tool restrictions, and isolated execution.

**Agents (subagents)** — Specialized AI assistants that run in isolated contexts with their own tools and model. Claude can delegate to them automatically (if their description includes "PROACTIVELY") or you can invoke them by name.

**Rules** — Coding conventions that auto-apply based on file paths. When you edit a `.go` file, Go rules load automatically. When you edit `.tsx`, frontend rules load.

**Hooks** — Shell scripts that run automatically on events like editing files (`PostToolUse`) or before running a tool (`PreToolUse`). Our hooks auto-format Go and TypeScript files on every edit.

**MCP servers** — External tool integrations via the Model Context Protocol. Connect Claude to GitHub, databases, documentation search, and other services. These aren't required for the team setup but can enhance your personal workflow.

**Plugins** — Bundled packages of skills, agents, hooks, and MCP configs from the Claude Code marketplace. Like MCP servers, these are optional personal enhancements.

**Memory** — Claude maintains auto-generated memory across sessions at `~/.claude/projects/<project>/memory/`. It remembers patterns, preferences, and lessons learned. View with `/memory`.

### Commands, shortcuts, and session management

**Sessions**

| Action | How |
|--------|-----|
| Start a session | `claude` (terminal) or open in IDE |
| Continue last session | `claude -c` or `/resume` |
| Resume a named session | `claude -r "name"` or `/resume` |
| Rename session | `/rename <name>` |
| Branch conversation | `/branch` (explore alternatives in parallel) |
| Rewind to checkpoint | `Esc` twice, or `/rewind` |
| Export session | `/export` |
| Side question | `/btw <question>` (doesn't affect conversation history) |

**Context** — The context window fills over time. Manage it actively:

| Action | How |
|--------|-----|
| Check context usage | `/context` |
| Compress conversation | `/compact` or `/compact <focus>` (e.g., `/compact keep the migration plan, drop debugging`) |
| Clear and start fresh | `/clear` |

Use `/clear` between unrelated tasks — context pollution degrades quality. Use `/compact` when context gets large. Delegate heavy investigation to subagents to keep the main context clean. Press `Esc` twice to rewind if Claude goes off track.

**Configuration and diagnostics**

| Action | How |
|--------|-----|
| Invoke a skill | Type `/` then select from menu |
| Switch model | `/model` (sonnet/opus/haiku) |
| Set effort level | `/effort` (low/medium/high) |
| Toggle extended thinking | `Option+T` (macOS) / `Alt+T` |
| Cycle permission mode | `Shift+Tab` |
| Enter plan mode | `/plan <description>` or `Shift+Tab` |
| Edit plan externally | `Ctrl+G` |
| Manage permissions | `/permissions` or `/allowed-tools` |
| Open settings | `/config` |
| View diff of changes | `/diff` |
| Check session cost | `/cost` |
| Check version and status | `/status` |
| Run installation health check | `/doctor` |
| List all commands | `/help` |

### Advanced features

**Plan mode** — Separates research from implementation. Claude explores the codebase and writes a plan for your review before making changes. Activate with `Shift+Tab`, `/plan`, or `--permission-mode plan`. Edit the plan externally with `Ctrl+G`.

**Extended thinking** — Gives Claude more reasoning time for complex problems. Toggle with `Option+T` (macOS) / `Alt+T`. Set effort level with `/effort`. Include "ultrathink" in prompts for maximum depth.

**Auto mode** — Uses a background safety classifier to auto-approve safe tool calls without prompting. Cycle to it with `Shift+Tab`. Configure trusted domains and environments in `settings.json` under `autoMode`.

**Permission modes** — A spectrum from restrictive to autonomous:
- `default` — Reads freely, prompts for writes and commands
- `acceptEdits` — Auto-approves file edits, prompts for commands
- `plan` — Read-only exploration
- `auto` — Classifier-based decisions
- `dontAsk` — Auto-denies tools unless pre-approved via `/permissions` or settings
- `bypassPermissions` — No checks (CI/CD use only)

**Headless and CI mode** — Run non-interactively with `claude -p "prompt" --output-format json`. Useful for CI pipelines, batch processing, and scripted workflows.

**Background tasks** — Long-running work continues while you chat. Skills with `context: fork` run in isolated subagents.

**Git worktrees** — Run `claude --worktree` to work in an isolated git worktree so experimental changes don't affect your working directory.

### Settings hierarchy

Settings are applied in this order (highest to lowest priority):

1. **Managed** — Organization-wide policies (IT/admin controlled)
2. **Local** — `.claude/settings.local.json` (personal, gitignored)
3. **Project** — `.claude/settings.json` (team-shared, checked in)
4. **User** — `~/.claude/settings.json` (personal, all projects)

Your local settings override project settings, so you can always customize without affecting the team.

---

## What's here

```
.claude/
├── CLAUDE.md                  # Project instructions (architecture, patterns, commands)
├── settings.json              # Team settings (env vars, permissions, hooks)
├── settings.local.json        # Personal overrides (gitignored)
├── README.md                  # This file
├── rules/                     # Path-scoped coding conventions (auto-applied)
│   ├── fleet-go-backend.md    #   Go: ctxerr, service patterns, logging, testing
│   ├── fleet-frontend.md      #   React/TS: components, React Query, BEM, interfaces
│   ├── fleet-database.md      #   MySQL: migrations, goqu, reader/writer
│   ├── fleet-api.md           #   API: endpoint registration, versioning, error responses
│   └── fleet-orbit.md         #   Orbit: agent packaging, TUF updates, platform-specific code
├── skills/                    # Workflow skills (invoke with /)
│   ├── review-pr/             #   /review-pr <PR#>
│   ├── fix-ci/                #   /fix-ci <run-url>
│   ├── test/                  #   /test [filter]
│   ├── find-related-tests/    #   /find-related-tests
│   ├── lint/                  #   /lint [go|frontend]
│   ├── fleet-gitops/          #   /fleet-gitops
│   ├── project/               #   /project <name>
│   ├── new-endpoint/          #   /new-endpoint
│   ├── new-migration/         #   /new-migration
│   ├── spec-story/            #   /spec-story <issue#>
│   └── update-data-dictionary/ #  /update-data-dictionary
├── agents/                    # Specialized AI agents
│   ├── go-reviewer.md         #   Go reviewer (proactive, sonnet)
│   ├── frontend-reviewer.md   #   Frontend reviewer (proactive, sonnet)
│   └── fleet-security-auditor.md  # Security auditor (on-demand, opus)
└── hooks/                     # Automated hooks
    ├── guard-dangerous-commands.sh  # PreToolUse: blocks rm -rf, force push, pipe-to-shell
    ├── goimports.sh           #   PostToolUse: formats Go files
    └── prettier-frontend.sh   #   PostToolUse: formats frontend files
```

## Skills reference

Several skills use the `gh` CLI for GitHub operations (PR review, CI diagnosis, issue speccing). Make sure you have [`gh`](https://cli.github.com/) installed and authenticated with `gh auth login`.

| Skill | Usage | What it does |
|-------|-------|-------------|
| `/review-pr` | `/review-pr 12345` | Reviews a PR for correctness, Go idioms, SQL safety, test coverage, and Fleet conventions. Runs in isolated context. Requires `gh`. |
| `/fix-ci` | `/fix-ci https://github.com/.../runs/123` | Diagnoses CI failures in 8 steps: identifies failing suites, fetches logs, classifies failures as stale assertions vs real bugs, fixes stale assertions, and reports real bugs. Requires `gh`. |
| `/test` | `/test` or `/test TestFoo` | Detects which packages changed via `git diff` and runs their tests with the correct env vars (`MYSQL_TEST`, `REDIS_TEST`). |
| `/find-related-tests` | `/find-related-tests` | Maps changed files to their `_test.go` files, integration tests, and test helpers. Outputs exact `go test` commands. |
| `/fleet-gitops` | `/fleet-gitops` | Validates GitOps YAML: osquery queries against Fleet schema, Apple/Windows/Android profiles against upstream references, and software against the Fleet-maintained app catalog. |
| `/project` | `/project android-mdm` | Loads workstream context from `~/.fleet/claude-projects/`. Creates new project files if none exist. |
| `/new-endpoint` | `/new-endpoint` | Scaffolds a Fleet API endpoint: request/response structs, endpoint function, service method, datastore interface, handler registration, and test stubs. |
| `/new-migration` | `/new-migration` | Creates a timestamped migration file and test file with proper naming, init registration, and Up/Down functions. |
| `/spec-story` | `/spec-story 12345` | Breaks down a GitHub story into implementable sub-issues: maps codebase impact, decomposes into atomic tasks per layer (migration/datastore/service/API/frontend), and writes specs with acceptance criteria and a dependency graph. Requires `gh`. |
| `/lint` | `/lint` or `/lint go` | Runs the appropriate linters (golangci-lint, eslint, prettier) on recently changed files. Accepts `go`, `frontend`, or a file path to narrow scope. |
| `/update-data-dictionary` | `/update-data-dictionary` | Compares recent migrations against `DATA-DICTIONARY.md` and updates it with missing tables, columns, renames, and schema changes. |

## Agents reference

### go-reviewer (sonnet, proactive)
Runs automatically after Go file changes. Checks:
- Error handling (ctxerr wrapping, no swallowed errors)
- Database patterns (parameterized queries, reader/writer, and index coverage)
- API conventions (auth checks, response types, and HTTP status codes)
- Test coverage (integration tests for DB code, edge cases)
- Logging (structured slog, no print statements)

### frontend-reviewer (sonnet, proactive)
Runs automatically after TypeScript and React file changes. Checks:
- TypeScript strictness (no `any`, proper type narrowing)
- React Query patterns (query keys, `enabled` option)
- Component structure (4-file pattern, BEM naming)
- Interface consistency (`I` prefix, `frontend/interfaces/` types)
- Accessibility (ARIA attributes, keyboard navigation)

### fleet-security-auditor (opus, on-demand)
Invoke when touching auth, MDM, enrollment, or user data. Uses Opus for deeper adversarial reasoning. Checks:
- API authorization gaps (missing `svc.authz.Authorize` calls)
- MDM profile payload injection
- osquery query injection
- Team permission boundary violations
- Certificate and SCEP handling
- PII in logs, license enforcement bypass

You can add your own agents by creating files in `.claude/agents/` on a branch, or in `~/.claude/agents/` for personal agents that apply across all projects.

## Hooks

Three hooks run automatically:

| Hook | Event | Files | What it does |
|------|-------|-------|-------------|
| `guard-dangerous-commands.sh` | PreToolUse (Bash) | All commands | Blocks `rm -rf /`, force push to main/master, `git reset --hard origin/`, and pipe-to-shell attacks |
| `goimports.sh` | PostToolUse (Edit/Write) | `server/**/*.go`, `cmd/**/*.go`, `orbit/**/*.go`, `ee/**/*.go` | Formats with `goimports` → `gofumpt` → `gofmt` (first available) |
| `prettier-frontend.sh` | PostToolUse (Edit/Write) | `frontend/**` | Formats with `npx prettier --write` |

Formatting hooks only run on files matching their path patterns (not on every edit). All hooks exit gracefully if the tool isn't installed. To add project-level hooks, edit `.claude/settings.json` on a branch. For personal hooks, add them to `~/.claude/settings.json`.

## Rules

Rules auto-apply when you edit files matching their path globs:

| Rule | Paths | Key conventions |
|------|-------|----------------|
| `fleet-go-backend.md` | `server/**/*.go`, `cmd/**/*.go`, `orbit/**/*.go`, `ee/**/*.go`, `pkg/**/*.go`, `tools/**/*.go`, `client/**/*.go`, `test/**/*.go` | ctxerr errors, error types, input validation, viewer context, auth pattern, service signatures, slog logging, ptr utilities, and pagination |
| `fleet-frontend.md` | `frontend/**/*.ts`, `frontend/**/*.tsx` | React Query, component structure, BEM/SCSS, permissions utilities, team context, notifications, XSS prevention, and string/URL utilities |
| `fleet-database.md` | `server/datastore/**/*.go` | Migration naming and testing, goqu queries, reader/writer, parameterized SQL, and batch operations |
| `fleet-api.md` | `server/service/**/*.go` | Endpoint registration, API versioning, and error-in-response pattern |
| `fleet-orbit.md` | `orbit/**/*.go` | Agent architecture, TUF updates, platform-specific code, packaging, keystore, and security considerations |

### Not covered by this configuration

The following areas have their own conventions and aren't covered by the current rules, hooks, or skills:

- **`website/`** — Fleet marketing website (Sails.js, separate `package.json` and conventions)
- **`ee/fleetd-chrome/`** — Chrome extension for ChromeOS (TypeScript, separate test setup)
- **`ee/vulnerability-dashboard/`** — Vulnerability dashboard (Sails.js/Grunt, legacy patterns)
- **`third_party/`** — Forked external code (not Fleet's conventions)
- **MDM-specific patterns** — `server/mdm/` has complex multi-platform patterns (Apple, Windows, Android) beyond what the Go backend rule covers

## Permissions

`settings.json` pre-approves safe operations so you don't get prompted:

**Allowed:** `go test`, `go vet`, `go build`, `golangci-lint`, `yarn test/lint`, `npx prettier/eslint/tsc/jest`, `make test/lint/build/generate/serve/db-*/migration/deps/e2e-*`, `git status/diff/log/show/branch`, and `gh pr/issue/run/api`

**Denied:** `git push --force`, `git push -f`, `rm -rf /`, and `rm -rf ~`

Commands not in either list (like `git commit` or `git push`) will prompt for permission on first use. To pre-approve them, add them to your `.claude/settings.local.json` — see [local settings](#local-settings) below.

## Customize your setup

Everything above works without extra configuration. The sections below describe optional enhancements you can add to your personal setup.

### Local settings

Create `.claude/settings.local.json` (gitignored) for personal overrides. Local settings take priority over team settings, so you can add permissions, change defaults, or configure MCP servers without affecting anyone else.

Common things to add to your local settings:
- Git write permissions (the shared setup only allows read operations)
- MCP server tool permissions
- Additional `make` or `bash` commands specific to your workflow

```json
{
  "permissions": {
    "allow": [
      "Bash(git add*)",
      "Bash(git commit*)",
      "Bash(git push)",
      "mcp__github__*",
      "mcp__my-mcp-server__*"
    ]
  }
}
```

### MCP servers

The shared setup doesn't require any MCP servers. Skills use the `gh` CLI for GitHub operations, which works without MCP. However, MCP servers can enhance your workflow:

```bash
# GitHub MCP — richer GitHub integration beyond what gh CLI provides
claude mcp add --transport http github https://api.github.com/mcp

# Semantic code search — understand code structure, not just text patterns
claude mcp add --transport stdio serena -- uvx --from git+https://github.com/oraios/serena serena start-mcp-server --context=claude-code --project-from-cwd

# Documentation search — look up third-party library docs
claude mcp add --transport stdio context7 -- npx -y @upstash/context7-mcp@latest
```

After adding an MCP server, grant its tools in your local settings:
```json
{
  "permissions": {
    "allow": ["mcp__github__*", "mcp__serena__*", "mcp__context7__*"]
  }
}
```

### Plugins

Plugins bundle skills, agents, hooks, and MCP configs. Browse and install from the marketplace:

```bash
claude plugins list              # Browse available plugins
claude plugins install <name>    # Install a plugin
claude plugins remove <name>     # Remove a plugin
```

Useful plugins for Fleet development: `gopls-lsp` (Go LSP), `typescript-lsp` (TS LSP), `feature-dev` (code explorer, architect, and reviewer agents), and `security-guidance` (security warnings on sensitive patterns).

## Contribute to this configuration

1. Create a branch.
2. Edit files in `.claude/`.
3. Start a new Claude Code session to test. Use `/context` to verify your changes load correctly.
4. Open a PR for review.

### Add a skill

Create `.claude/skills/your-skill/SKILL.md`:
```yaml
---
name: your-skill
description: When to trigger. Use when asked to "do X" or "Y".
allowed-tools: Read, Grep, Glob, Bash(specific command*)
disable-model-invocation: true  # Optional: user-only, no auto-trigger
context: fork                    # Optional: run in isolated subagent
---

Instructions for Claude when this skill is invoked.
Use $ARGUMENTS for user input.
```

### Add a rule

Create `.claude/rules/your-rule.md`:
```yaml
---
paths:
  - "path/**/*.ext"
---

# Rule title
- Convention 1
- Convention 2
```

### Add an agent

Create `.claude/agents/your-agent.md`:
```yaml
---
name: your-agent
description: What it does. Include "PROACTIVELY" for auto-invocation.
tools: Read, Grep, Glob, Bash
model: sonnet  # or opus for deep reasoning
---

System prompt describing the agent's role and review criteria.
```
