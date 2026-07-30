# Fleet Hangar

A desktop control panel for [Fleet](https://github.com/fleetdm/fleet) contributors, built
with Go and [Wails 3](https://v3alpha.wails.io). Bundles the daily tasks of working on a
Fleet clone (checking out branches, building, running `fleet serve`, tailing logs, managing
the dev MySQL, driving `fleetctl`, applying GitOps repos, spinning up `osquery-perf`) into
one app. macOS-first.

**Why Go?** To match the rest of the repo so Fleet engineers can contribute to it. The
backend is plain Go (`os/exec`, `syscall`, goroutines); only the desktop shell is Wails.
Hangar began as a Rust/Tauri app; it was ported to Go and that port is now the canonical
implementation.

## Architecture

- **`internal/`** — all the logic, pure and unit-tested (each package takes explicit
  paths/timestamps so tests are hermetic):
  - `processes` — spawn/log/lifecycle engine: child-process management, streamed log readers
    (level detection, secret scrubbing, on-disk rotation, in-memory ring), `running.json`
    crash-recovery, SIGTERM→SIGKILL on process groups, docker-compose orchestration, TLS probe
  - `settings`, `gitrepo`, `db`, `gitops`, `fleetctl`, `deps`, `troubleshoot`, `perf`,
    `perfconfig` — one per former `src-tauri/src/*.rs` module
  - `paths` (macOS dirs + path safety), `shellpath` (login-shell PATH warming), `traymenu`
    (tray menu model)
- **`services/`** — thin Wails-bound service structs; each exported method is callable from
  the frontend. They resolve real paths and delegate to `internal/`.
- **`main.go` / `tray.go` / `emitter.go`** — the native shell: app bootstrap, system tray,
  and window lifecycle (hide-to-tray, dock reopen, Cmd+Q→confirm).
- **`frontend/`** — the React + TypeScript UI (shared with the Rust app). The only
  Wails-specific glue is `src/lib/tauri.ts` (the `api.*` IPC layer over the generated
  bindings) and `src/lib/events.ts` (the `listen()` adapter over Wails events).

## Development

Requirements: Go (see `go.mod`), Node 24+, and the
[Wails 3 prerequisites](https://v3alpha.wails.io/getting-started/installation/). Install the
CLIs once:

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.98
go install github.com/go-task/task/v3/cmd/task@latest
```

Then, from this directory:

```sh
task dev       # live-reload dev mode (Vite + Go)
task build     # type-check + production build -> bin/fleet-hangar
task package   # build + bundle + ad-hoc sign -> "bin/Fleet Hangar.app"
task dist      # zip the existing .app into a shareable "bin/Fleet Hangar.zip"
go test ./...  # backend unit tests
```

After changing any Go service signature, regenerate the TypeScript bindings (also run by
`task build`):

```sh
wails3 generate bindings -clean=true -ts
```

## Notes

- **Names.** The bundle is `Fleet Hangar.app` (the `PRODUCT_NAME` Taskfile var) so Finder,
  Launchpad, and Spotlight show "Fleet Hangar". The executable inside stays `fleet-hangar`
  (the `APP_NAME` var) — it's also reused for the `-server` binary and Docker image tags,
  where a space would break things. The **bundle identifier** is `com.fleetdm.fleet-hangar`
  — the same ID the original Rust app used, so settings written by it carry over untouched.
  Settings live under `~/Library/Application Support/<id>/` and logs under
  `~/Library/Logs/<id>/`; DB backups live in `<repo>/db-backups/`.
- **Distribution.** `task package` produces only an ad-hoc-signed `.app`; `task dist` zips
  whatever bundle is in `bin/` into `bin/Fleet Hangar.zip` (via `ditto`, so the signature
  survives) for handoff. Two paths:
  - *Quick / trusted teammate:* `task package` → `task dist`, then the recipient clears
    quarantine after unzipping: `xattr -dr com.apple.quarantine "/path/to/Fleet Hangar.app"`
    (an ad-hoc-signed app from another machine is otherwise blocked by Gatekeeper).
  - *Clean install anywhere:* configure `SIGN_IDENTITY` + `KEYCHAIN_PROFILE` in
    `build/darwin/Taskfile.yml`, run `task darwin:sign:notarize` (Developer ID sign +
    Apple notarization), then `task dist`. No quarantine step needed.

  `dist` never rebuilds, so running it after `sign:notarize` preserves the notarized signature.

## Known issues

- **Rare crash on display sleep/wake or monitor changes** (upstream Wails v3 alpha bug, not
  ours). Wails' `screen_darwin.go` stores the autoreleased `[NSString UTF8String]` buffers for
  each screen's `id`/`name` in a C struct and reads them from Go later, after the autorelease
  pool has drained — a use-after-free. On an `ApplicationDidChangeScreenParameters` event the
  dangling pointer trips `fatal error: invalid pointer found on stack`. It's a `fatal error`,
  not a panic, so it can't be recovered, and the handler is registered inside Wails so we can't
  intercept it. It's infrequent (one occurrence observed over an ~11h session). Until a fixed
  Wails alpha ships, relaunch the app if it happens. Reported upstream:
  [wailsapp/wails#5556](https://github.com/wailsapp/wails/issues/5556).
