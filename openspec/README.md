# OpenSpec (optional tooling)

OpenSpec is a spec-driven workflow for proposing, designing, and tracking
larger code changes. In this repo it is an **opt-in tool**, not a required
part of the development process. The team has not adopted it as policy, and
no PR is required to use it.

## When it's useful

Reach for OpenSpec when a change is large enough that you want a written
record before any code is written:

- A cross-cutting feature that spans datastore, service, endpoint, and UI.
- A refactor that touches more than a handful of files and benefits from an
  agreed shape before implementation.
- An RFC-style design you want to review with collaborators (human or AI)
  without committing to code yet.

## When to skip it

Most day-to-day work does not need it:

- Bug fixes, small features, dependency bumps, doc tweaks.
- Anything where opening a PR with a good description is faster and just as
  clear.

If the change fits in a single PR description, write the PR description.

## Install

The slash commands shell out to the `openspec` CLI, so it must be on `$PATH`:

```
brew install openspec
```

Reading the Markdown artifacts under `openspec/` does not require the CLI.

## Flow

```
explore → propose → apply → archive
```

1. **`/opsx:explore`** — think through the idea. No code, no artifacts unless asked.
2. **`/opsx:propose`** — generate `proposal.md` (what & why), `design.md` (how),
   and `tasks.md` under `openspec/changes/<change-name>/`.
3. **`/opsx:apply`** — implement the tasks. Pass the change name (e.g.
   `/opsx:apply add-foo`) or let it infer from context.
4. **`/opsx:archive`** — once merged, move the change to
   `openspec/changes/archive/` and update specs under `openspec/specs/`.

Skip steps freely. Most changes only need `propose` + `apply`; small ones
might just be `explore`.

## What lives where

- `openspec/changes/<name>/` — in-flight proposals and tasks.
- `openspec/changes/archive/` — completed changes.
- `openspec/specs/` — accepted specifications.
- `openspec/config.yaml` — project context and rules the AI must respect.
  Points at `.claude/CLAUDE.md` for full project guidance.

## Vendored files — do not hand-edit

The OpenSpec CLI owns these directories and `openspec update` will overwrite
any local changes:

- `.claude/skills/openspec-*/`
- `.claude/commands/opsx/`

Customize behavior via `openspec/config.yaml` instead. If you truly need a
divergent skill, copy it under a new name so the updater won't touch it.

## Conventions

- Artifacts are Markdown; commit them alongside the code they describe.
- Use the new terminology in new artifacts: **Fleets** (not Teams), **Reports**
  (not Queries). Existing code keeps its current names.
- Treat `openspec/` artifacts as documentation, not as a contract: code review
  is still the source of truth.
