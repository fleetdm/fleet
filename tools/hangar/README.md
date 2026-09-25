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
[Wails 3 prerequisites](https://v3.wails.io/getting-started/installation/). Install the
CLIs once:

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.8
go install github.com/go-task/task/v3/cmd/task@latest
```

Keep all three Wails versions in lockstep when upgrading — the `wails3` CLI above, the
`github.com/wailsapp/wails/v3` module in `go.mod`, and `@wailsio/runtime` in
`frontend/package.json` (pinned to an exact version on purpose: the Go and JS runtimes speak the
same protocol to each other, `package-lock.json` is gitignored repo-wide, so a `latest` range
would hand every fresh clone whatever runtime happened to be newest that day).

Then, from this directory:

```sh
task dev       # live-reload dev mode (Vite + Go)
task build     # type-check + production build -> bin/fleet-hangar
task package   # build + bundle + ad-hoc sign -> "bin/Fleet Hangar.app"
task dist      # zip the existing .app into a shareable "bin/Fleet Hangar.zip"
task pkg       # wrap the existing .app into a Fleet-installable "bin/Fleet Hangar.pkg"
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
- **Install onto hosts via Fleet.** `task pkg` wraps the existing `bin/Fleet Hangar.app` into
  `bin/Fleet Hangar.pkg` — a component installer that drops the app into `/Applications`. Upload
  it under *Fleet > Software > Add software > Custom package*; Fleet reads the identifier and
  version from the app's `Info.plist` to track install status. Like `dist`, it never rebuilds, so
  notarize first (`task darwin:sign:notarize`) for real hosts — an ad-hoc build installs cleanly
  only because a pkg install skips the quarantine flag.

## Versioning

`build/config.yml`'s `info.version` is the **single place** to set the release — bump it there
whenever you hand a build to someone. `task build` / `task package` stamp it, plus the git commit
and build time, into three places that would otherwise disagree:

| Where | What it shows | Fed by |
|---|---|---|
| `--version` | `Fleet Hangar 1.1.0 (a1b2c3d, built …Z)` | `-ldflags -X` → `internal/buildinfo` |
| Settings sidebar (bottom) | `v1.1.0 · a1b2c3d`, click to copy the full string | same |
| `hangar.log`, first line of every session | same | same |
| `Info.plist` → Finder, `task pkg`, Fleet inventory | `CFBundleShortVersionString`, `CFBundleVersion`, `FleetHangarGitCommit` | `PlistBuddy` at bundle time |

Check any bundle without launching it:

```sh
"/Applications/Fleet Hangar.app/Contents/MacOS/fleet-hangar" --version
```

The commit carries a `-dirty` suffix when the working tree had uncommitted changes — the normal
state of a build made to test a branch, and worth knowing when someone reports what it did. A
build made outside the task pipeline (a bare `go build`, `go run`, a test binary) reports
`dev (unknown)`, so an unstamped build can't be mistaken for a release.

Bumping the version is what makes an install register as an upgrade: Fleet's software inventory
keys off `CFBundleShortVersionString`, so shipping two different builds both claiming `1.0.0`
leaves hosts reporting the old version.

## Logs

Two different things live in `~/Library/Logs/com.fleetdm.fleet-hangar/`:

- **`<channel>.log`** — the output of the processes Hangar spawns (`fleet-serve-s1`, `scep-scep1`,
  `tuf-build`, ...). This is what the Logs tab shows, and its *Reveal in Finder* button opens this
  folder. Rotated at 16 MB, one generation kept as `.log.1`.
- **`hangar.log`** — Hangar's own diagnostics: startup, the macOS sleep/wake and screen-change
  events it saw, which quit path it took, a heartbeat every five minutes, and — because file
  descriptor 2 is redirected into it — whatever the Go runtime prints on its way out when the app
  dies unexpectedly. A packaged `.app` launched from Finder has fd 2 on `/dev/null`, so without
  that redirect those crash dumps are simply lost. Rotated at 8 MB, at startup only (fd 2 points
  at the open file for the whole session).

`hangar.log` is the file to ask for when someone reports that Hangar "just closed itself". Every
session opens with the build that produced it (see [Versioning](#versioning)) and then reports
what became of the previous one, from a liveness marker (`last-session.json`) refreshed every
30 seconds:

```
level=WARN msg="previous session never recorded an exit: it crashed or was killed" pid=… last_alive=…
```

Any crash output sits immediately above that line. A deliberate quit logs `session end` with a
reason instead — including `signal: terminated`, so a `killall` or a logout doesn't read as a
crash. `HANGAR_LOG_LEVEL=debug` raises the level for a run.

### Reporting a crash

Ask for `hangar.log`, and ask them to **relaunch Hangar first** — the verdict on a dead session is
written at the *next* startup, so a relaunch is what turns "it vanished" into a timestamped line:

```sh
zip -j ~/Desktop/hangar-logs.zip ~/Library/Logs/com.fleetdm.fleet-hangar/hangar.log*
```

The glob picks up `hangar.log.1` too, in case the log rotated and the crash landed in the previous
generation. Worth collecting alongside it: the build (bottom of the Settings sidebar), roughly when
it happened and what the Mac was doing (lid closed, monitor plugged or unplugged, idle), whether
their `fleet serve`/ngrok survived — that separates "Hangar died" from "everything died" — and
`~/Library/Logs/DiagnosticReports/Fleet Hangar*.ips` if it exists. That last one won't exist for a
Go `fatal error`, which prints its dump and calls `exit(2)` — a normal exit as far as macOS is
concerned, and the reason these crashes left no trace at all before `hangar.log`. A segfault down
in the Objective-C/webview layer does die by signal and does produce one.

Reading it, working backwards from the end:

| What you see | What happened |
|---|---|
| `session end` with a reason | Quit, or signalled. Not a crash. |
| No `session end`; next session logs `previous session never recorded an exit` | It died — `last_alive` dates it to within 30s |
| `fatal error:` / `panic:` + goroutine dump | The cause, with every goroutine (a stalled main thread shows as the pile-up behind it) |
| `screen parameters changed` or `displays woke` just before the end | The Wails screen crash is back — reopen [wailsapp/wails#5556](https://github.com/wailsapp/wails/issues/5556) |
| Nothing — the log just stops | Killed hard (jetsam, OOM, `kill -9`). The last heartbeat's `heap_mb`/`goroutines` and an `.ips` are the evidence |

## Known issues

- **Crash on display sleep/wake or monitor changes — fixed upstream, keep an eye out for it.**
  Wails' `screen_darwin.go` used to store the autoreleased `[NSString UTF8String]` buffers for
  each screen's `id`/`name` in a C struct and read them from Go later, after the autorelease pool
  had drained — a use-after-free. On an `ApplicationDidChangeScreenParameters` event the dangling
  pointer tripped `fatal error: invalid pointer found on stack`: a `fatal error` rather than a
  panic, so unrecoverable, and raised inside Wails where we couldn't intercept it. Symptom was
  Hangar simply being gone on return to the machine, with nothing written down anywhere.

  Reported as [wailsapp/wails#5556](https://github.com/wailsapp/wails/issues/5556) and fixed in
  `v3.0.0-alpha.101` (the strings are `strdup`'d now, and screen enumeration runs in an explicit
  autorelease pool). Hangar was pinned to `alpha.98`, one release short of the fix, until the
  bump to `beta.8`.

  If it ever comes back, `hangar.log` will say so (see [Logs](#logs)): a hit ends with
  `screen parameters changed` — logged from a synchronous event hook that runs just before Wails
  refreshes its screen cache — followed by a `fatal error` dump naming `cScreenToScreen` /
  `processAndCacheScreens`.
