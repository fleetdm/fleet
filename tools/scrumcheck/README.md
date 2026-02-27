# scrumcheck

Scans a GitHub Project v2 for items in ✔️Awaiting QA
that are missing or have an unchecked QA confirmation checklist.

## Build

export GITHUB_TOKEN=...
cd tools/scrumcheck
go build -o scrumcheck .

## Run

./scrumcheck -org fleetdm -project 71
./scrumcheck -org fleetdm -project 97

The app opens a local bridge URL (for example `http://127.0.0.1:61891/`).
This is now the primary non-hybrid UI path (no generated report file required).

## Frontend Runtime Modes

- Prod (embedded UI): default behavior.
- Dev (local UI files): pass `-ui-dev-dir` pointing at a directory containing:
  - `index.html`
  - `assets/app.css`
  - `assets/app.js`

Example:

`./scrumcheck -p 71 97 -l '#g-orchestration' -ui-dev-dir ./ui`
