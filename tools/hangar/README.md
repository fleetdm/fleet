# Fleet Hangar

A desktop control panel for [Fleet](https://github.com/fleetdm/fleet) contributors. Bundles the daily tasks of working on a Fleet clone — checking out branches, building, running `fleet serve`, tailing logs, managing the dev MySQL, driving `fleetctl`, applying GitOps repos, and spinning up `osquery-perf` load — into one app so you spend less time in eight terminal tabs.

Built with [Tauri 2](https://tauri.app) (Rust backend + React/TypeScript frontend). macOS-first.

## What it does

Each tab maps to one chunk of the dev loop:

- **Git** — list and check out branches in your Fleet clone. Filters for RC branches (last N minors), `main`, or everything. Checkout supports auto-stash or discard when the working tree is dirty.
- **Server** — orchestrates the full bring-up chain: `make deps` → `make generate` → `make build` → `docker compose up -d` → `fleet prepare db --dev` → `fleet serve --dev`. Toggles `--dev_license`, `--debug`, `--logging_debug`, and a custom `--config` from the UI. ngrok and a Python static server can be started alongside.
- **Logs** — tails `fleet serve` output in a virtualized list with level filters (debug/info/warn/error), search, and a time-window selector. Snapshot the current view to a file with one click.
- **Database** — back up / restore / delete dumps of the dev MySQL; one-click DB reset (`drop` + `prepare`). Backups land in a self-managed folder with its own `.gitignore` so they stay out of git.
- **fleetctl** — sub-tabs for `login`, `get`, `trigger` (with the cron registry grouped into *featured / mdm / maintenance / fast / migration*), and free-form custom commands. Reads and edits the fleetctl config to switch contexts.
- **GitOps** — scans a configured directory for gitops repos (anything with a `default.yml`) and runs `fleetctl gitops` apply or generate, with dry-run support and live streaming output.
- **osquery-perf** — launches the load generator with per-OS template host counts, MDM probability, SCEP challenge, and interval flags. Totals are derived from the per-template sum so the agent can't fatal on a mismatch.
- **Settings** — paths (Fleet repo, fleetctl binary, gitops dir), fleet serve flags, fleetctl contexts, ngrok / Python config, and a troubleshoot section that scans for processes by port or pattern and lets you kill them.

Plus, around the tabs:

- **First-run gate** — discovers Fleet clones automatically and runs a dependency checklist (git, go, docker, node, etc.) against the versions declared in the repo.
- **Status rail** — bottom bar showing current branch, running processes, and Docker health.
- **System tray** — status-at-a-glance icon with a Start-All / Stop-All menu and the same service indicators as the in-app rail.
- **Hide-to-tray** — closing the window (X / Cmd+W) hides it like Slack/Discord. Cmd+Q and tray ▸ Quit route through a confirm modal that tears down running services and runs `docker compose down` before exiting.

## Project layout

```
tools/hangar/
├── src/                    React/TypeScript frontend
│   ├── components/tabs/    One file per tab
│   └── lib/                Tauri IPC bindings, orchestration, hooks
└── src-tauri/              Rust backend
    └── src/
        ├── processes.rs    Spawn / log / lifecycle manager for every child process
        ├── git.rs          Branch listing, checkout, stash/discard
        ├── fleetctl.rs     fleetctl invocation + config reading
        ├── gitops.rs       GitOps repo discovery
        ├── db.rs           MySQL backup directory + metadata
        ├── deps.rs         First-run dependency checks
        ├── perf.rs         osquery-perf templates
        ├── settings.rs     Persisted settings
        ├── tray.rs         macOS tray menu
        ├── shellpath.rs    Login-shell PATH warming for the packaged build
        ├── troubleshoot.rs Port / pattern process scans
        └── lib.rs          Tauri bootstrap, quit/close flow
```

## Development

Requirements:

- Node 20+ (see `.nvmrc`)
- Rust stable
- Tauri prerequisites for your platform — see the [Tauri prerequisites guide](https://tauri.app/start/prerequisites/)

```sh
cd tools/hangar
npm install
npm run tauri dev
```

Other useful commands:

- `npm run dev` — frontend only (Vite at `localhost:1420`)
- `npm run build` — type-check + frontend production build
- `npm run tauri build` — package a release binary / `.app` bundle

The first-run gate handles the rest — it will discover your Fleet clones, check dependencies, and let you pick which clone to point at.

## Notes

- Bundle identifier is `com.fleetdm.fleet-hangar`. Settings, logs, and DB backups are scoped to that identifier under the standard macOS app-support paths.
