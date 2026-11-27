# Fleet Atlas

Interactive learning app for Fleet employees — mostly aimed at new engineers,
but useful to anyone who wants a faster way to build a mental model of how
Fleet is put together.

Content: architecture maps, request tracers, scenarios, a glossary,
flow-order quizzes, and self-assessments across the whole thing. Every claim
is hand-authored against the current `fleetdm/fleet` codebase (see the audit
script below for drift detection).

Not a Fleet product — a local companion to
[fleetdm/fleet](https://github.com/fleetdm/fleet). No server-side data leaves
your machine; quiz progress lives in `localStorage`.

## Run locally

```bash
yarn install
yarn dev
```

Open [http://localhost:3737](http://localhost:3737).

## Optional: link citations to a local Fleet checkout

Many pages cite specific files in `fleetdm/fleet`. By default those render as
`github.com/fleetdm/fleet/blob/main/...` links. If you keep a local checkout,
set `FLEET_REPO_PATH` and the same citations become clickable editor links
(VS Code by default; override with `EDITOR_URL_SCHEME`):

```bash
FLEET_REPO_PATH=~/src/fleet yarn dev
```

Any absolute path works; `~` is expanded.

## Keep the content honest: the audit script

Atlas makes specific technical claims (cron intervals, MySQL tables, API
paths, which features are premium-gated). Those claims drift when Fleet
changes. `scripts/audit-fleet-claims.py` cross-checks them against a real
`fleetdm/fleet` checkout and fails loudly on drift.

What it verifies:

- **Cron intervals** — compares `src/components/background/jobs.ts` against
  `cmd/fleet/cron.go` + `server/fleet/cron_schedules.go` in Fleet.
- **MySQL tables** — confirms tables cited in
  `src/components/data/key-tables.ts` exist in Fleet's schema.
- **Endpoint paths** — checks that endpoint paths mentioned in subsystem
  descriptions match handlers in Fleet.
- **Premium gates** — verifies that premium-only claims in
  `src/components/tiers/tiers-data.ts` are still backed by
  `ErrMissingLicense` call sites in Fleet source.

Run from this directory:

```bash
./scripts/audit-fleet-claims.py
```

The script auto-detects the Fleet root as `../../..` of itself when living
at `tools/fleet-atlas/scripts/`. To audit against a different Fleet checkout
(e.g. a release branch), set `FLEET_REPO_PATH`:

```bash
FLEET_REPO_PATH=~/src/fleet-release-branch ./scripts/audit-fleet-claims.py
```

Exits `0` if everything matches, `1` on drift, `2` on config errors. Python 3
stdlib only — no `pip install` needed.

This audit also runs automatically on every Fleet release. The workflow
at `.github/workflows/atlas-drift-check.yml` (at the Fleet repo root)
triggers on `fleet-*` tag pushes, runs the script against the tagged tree,
and opens a `fleet-atlas` / `drift` issue if anything slipped. Issue bodies
include the full report and a link back to the workflow run. Re-runs comment
on an existing open issue instead of filing duplicates.

## Stack

Next.js 16 (App Router, Turbopack), React 19, TypeScript, Tailwind v4,
React Flow (graphs), dnd-kit (sortable quizzes), better-sqlite3 (local
`.fleet-atlas.db` for user-authored custom scenarios; quiz progress itself
is `localStorage`-only).

## Content model

Graphs, scenarios, glossary entries, quiz pools, and tracer steps are all
plain TypeScript data in `src/components/`. No CMS, no runtime fetch — the
build output is static except for a handful of client components. Editing
content is a code change.

## Status

Active. Hand-authored content; PRs welcome from Fleet employees. If you
notice a claim that looks wrong, run the audit script first — if drift
confirms it, send a PR.
