# scrumcheck

`scrumcheck` is a Scrum-process health scanner for Fleet GitHub projects.

It helps teams quickly spot issues that block smooth delivery: stale QA items, missing sprint/milestone/assignee metadata, release-label policy gaps, checklist misses, and release-story TODO leftovers.

Yes, we are making scrumchecks great again.

## What It Checks

- Release stories TODO
- Generic queries
- Missing sprint
- Missing milestones
- Release label guard
- Awaiting QA stale watchdog
- Awaiting QA gate
- Drafting estimation gate
- Missing assignee
- Assigned to me
- Unassigned unreleased bugs
- Updates timestamp expiry

## Requirements

- Go installed
- `GITHUB_TOKEN` set with access to the relevant repos/projects

## Build

```bash
cd tools/scrumcheck
go build -o scrumcheck .
```

## Run

Basic:

```bash
./scrumcheck -p 71
```

Multiple projects:

```bash
./scrumcheck -p 71 97
```

Project + group labels:

```bash
./scrumcheck -p 71 97 -l '#g-orchestration' -l '#g-security-compliance'
```

No auto-open (just print URL):

```bash
./scrumcheck -p 71 97 -open-report=false
```

After launch, open the printed local URL (for example `http://127.0.0.1:61891/`).

## Flags

- `-org` GitHub org (default: `fleetdm`)
- `-p`, `-project` one or more project numbers
- `-l`, `-label` labels that represent groups e.g. #g-orchestration (one or more)
- `-limit` max project items to scan (default: `100`)
- `-stale-days` stale threshold for Awaiting QA watchdog (default: `21`)
- `-bridge-idle-minutes` auto-shutdown for local bridge inactivity (default: `10`)
- `-open-report` auto-open browser on completion (default: `true`)
- `-ui-dev-dir` serve UI from local files instead of embedded assets

## UI Runtime Modes

- Production mode (default):
  - Uses embedded frontend assets.
- Dev mode (`-ui-dev-dir`):
  - Serves local files from a directory containing:
    - `index.html`
    - `assets/app.css`
    - `assets/app.js`

Example:

```bash
./scrumcheck -p 71 97 -l '#g-orchestration' -ui-dev-dir ./ui
```

## Security Model (Bridge)

- Binds to loopback (`127.0.0.1`) only
- Session-protected API endpoints
- Strict cookie settings:
  - `HttpOnly`
  - `Secure`
  - `SameSite=Strict`
- Host/origin validation for mutation endpoints
- Operation allowlists to prevent out-of-scope writes

## Troubleshooting

- **`GITHUB_TOKEN env var is required`**
  - Export token before running.
- **Browser doesn’t open**
  - Use the printed local URL manually.
- **No items found but expected some**
  - Confirm project numbers and label filters match the board/query.
- **Stale code scanning alert on PR**
  - Re-run CodeQL checks after pushing the fix commit.
