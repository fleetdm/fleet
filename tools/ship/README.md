# Fleet ship

`Fleet ship` is an interactive terminal UI that sets up and runs a local Fleet
dev environment with one command. It's aimed at non-engineers (PMs, designers,
anyone using a coding agent to build Fleet features) who want to test Fleet
locally without learning Docker, MySQL, ngrok, and the multi-step build
pipeline.

> **macOS only for v1.** On Linux/Windows the tool exits with a pointer to
> [`docs/Contributing/getting-started/building-fleet.md`](../../docs/Contributing/getting-started/building-fleet.md).

## Quick start

From a fresh macOS install:

1. Install [Docker Desktop](https://www.docker.com/products/docker-desktop) and
   start it (whale icon in the menu bar).
2. `brew install go yarn ngrok`
3. Install [nvm](https://github.com/nvm-sh/nvm) and the Node version Fleet
   pins to:
   ```sh
   curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
   # restart your shell or `source ~/.zshrc`, then:
   nvm install v24.10.0
   ```
4. `ngrok config add-authtoken <your token>` — see [ngrok account
   setup](#ngrok-account-setup) for where to get the token.
5. From the Fleet repo root: `make ship`

About 5 minutes total on a fresh laptop, mostly downloads. The first
`make ship` runs a setup wizard; subsequent launches skip it.

> **Why nvm and not Homebrew for Node?** Fleet pins to a specific Node
> version (`"node": "^24.10.0"` in `package.json`'s `engines`). `brew install
> node@24` follows the latest patch release of v24, which can drift away from
> what Fleet expects. nvm lets you install and stay on the exact version,
> and switch between versions if you ever need to.

## Prerequisites

`make ship` runs a doctor screen on every launch that checks each of these and
shows `✓` / `✗` rows. If anything is missing, install or fix it and re-run.

| Dependency | macOS install |
|---|---|
| Docker Desktop | Download from [docker.com](https://www.docker.com/products/docker-desktop), launch the app |
| Xcode Command Line Tools | `xcode-select --install` |
| Homebrew | Follow installer at [brew.sh](https://brew.sh) |
| Go | `brew install go` |
| nvm | `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh \| bash` |
| Node.js v24.10.0 | `nvm install v24.10.0` (from inside the Fleet repo) |
| Yarn | `brew install yarn` |
| ngrok | `brew install ngrok` |
| ngrok auth token | `ngrok config add-authtoken <token>` |
| Rosetta 2 (Apple Silicon only) | `softwareupdate --install-rosetta --agree-to-license` |

Why Rosetta on Apple Silicon: Fleet's frontend asset pipeline (`make
generate-dev`) calls a tool that's x86_64-only. Without Rosetta you'd see
`Unknown system error -86` mid-build.

## ngrok account setup

Fleet ship uses ngrok to give your local server a stable public URL — needed
for any MDM testing where Apple servers or enrolled devices have to reach back
to your machine.

1. Create a free account at <https://dashboard.ngrok.com/signup>.
2. Reserve a static domain at <https://dashboard.ngrok.com/domains> →
   **New Domain**. Free accounts get one (e.g. `fleet-pm-jane.ngrok-free.app`).
3. Grab your auth token at
   <https://dashboard.ngrok.com/get-started/your-authtoken>.
4. Run `ngrok config add-authtoken <token>` in your terminal.

The wizard on first `make ship` will ask for the static domain string.

## MDM server private key

Fleet's MDM features encrypt sensitive data (certificates, tokens) using a
**server private key** — a 32+ character string that must be the same every
time the server runs. If it changes, previously-encrypted data can't be
decrypted.

**Strongly recommended:** keep this in 1Password alongside your dev login
credentials. The wizard will ask you to paste it on first launch. Once stored
at `~/.config/fleet-ship/server_private_key` (mode 0600), it's reused on
every subsequent launch silently — you'll never be re-prompted unless that
file goes missing.

If you ever wipe `~/.config/fleet-ship/`, switch machines, or otherwise lose
the file, the wizard re-prompts for it. Pull it back from 1Password — Fleet
ship intentionally never auto-generates a key, so you can't accidentally end
up with multiple keys for the same dev environment.

## First-time setup

The first `make ship` after a fresh install does some extra work that
subsequent runs skip:

1. **Wizard** runs (ngrok domain, MDM key, premium on/off).
2. **Bring-up sequence** runs every step from scratch — `make deps`, `make
   generate-dev`, `make` (build the binaries), `prepare db`. This can take
   several minutes; later runs reuse the build cache and are much faster.
3. Once Fleet is running, **open the public URL** (e.g.
   `https://fleet-pm-jane.ngrok-free.app`) in your browser.

### Browser certificate warning

The first time you visit the public URL, your browser will show a "Not
Secure" / "self-signed certificate" warning. That's expected — Fleet's
`--dev` mode generates a temporary self-signed cert for the local server, and
ngrok forwards to it. Click through (in Chrome: **Advanced → Proceed to…**).
The site is your own machine.

### Creating the admin user

When the database is empty, Fleet's UI walks you through creating an admin
user — email, password, name, organization. Save those in 1Password too;
you'll need them every login, and again if you ever reset the database.

## Daily use

```sh
cd ~/projects/fleet
make ship
```

The dashboard shows current status. Press `q` to shut down cleanly when done.
Docker volumes persist across runs — your admin user, hosts, and settings are
all there the next time you launch.

### Keybindings

| Key | Action |
|---|---|
| `l` | Show fleet server logs (`esc` to dismiss) |
| `w` | Show webpack logs |
| `n` | Open ngrok's traffic inspector in your browser (`localhost:4040`) |
| `q` | Quit Fleet ship and shut down everything |

Future PRs will add `r` (manual rebuild), `p` (pause auto-rebuild), `s`
(switch worktree), `t` (simulated hosts), `b`/`R` (snapshot/restore), and `?`
(help overlay).

### Reconfiguring

To re-run the first-time wizard (e.g. you changed your ngrok domain):

```sh
make ship ARGS=--reconfigure
```

## Where things live

| Path | What's there |
|---|---|
| `~/.config/fleet-ship/config.yaml` | ngrok domain, premium toggle |
| `~/.config/fleet-ship/server_private_key` | MDM private key (mode 0600) |
| `tools/ship/.state/active.json` | Live run state — fleet PID, log paths, branch, ngrok URL. Coding agents can `cat` it for debugging. Removed on clean shutdown. |
| `tools/ship/.state/logs/fleet.log` | Fleet server stdout/stderr |
| `tools/ship/.state/logs/webpack.log` | Webpack output |
| `tools/ship/.state/logs/ngrok.log` | ngrok output |

`tools/ship/.state/` is gitignored.

## Troubleshooting

When something goes wrong, the per-process log files are the first place to
look:

```sh
tail -200 tools/ship/.state/logs/fleet.log
tail -200 tools/ship/.state/logs/webpack.log
tail -200 tools/ship/.state/logs/ngrok.log
```

For coding agents helping debug: point them at
`tools/ship/.state/active.json`. It contains the running PID, log file paths,
branch, commit, and ngrok URL — enough for an agent to investigate with `cat`,
`tail`, and `grep` directly. A useful prompt:

> Read `tools/ship/.state/active.json` — it has the server PID, log paths,
> and ngrok URL for my running Fleet. Help me debug \<thing\>.

If the bring-up sequence fails partway through, the dashboard shows which step
failed and the relevant error. Common causes:

- **Docker isn't running** → open Docker Desktop, wait for the whale icon to
  go solid, retry.
- **ngrok auth token missing or wrong** → `ngrok config add-authtoken <token>`.
- **ngrok static domain not registered** → check
  <https://dashboard.ngrok.com/domains>.
- **Build fails after a recent `git pull`** → re-run `make ship`; the
  `generate-dev` step regenerates the `server/bindata/generated.go` file the
  build depends on.

## What's not in this version

This is the first cut of Fleet ship. Tracked follow-ups:

- **Auto-rebuild on file changes** — for now, edit code, quit ship, re-run.
- **Switching between branches and released versions** — manage worktrees
  with `git worktree` manually for now.
- **DB snapshots and rollback** before running migrations.
- **Simulated osquery hosts** to populate the UI with fake hosts for testing.
- **In-tool help overlay** and richer in-product docs.

## Linux / Windows

Not supported in this version. The tool exits cleanly with a pointer to the
manual setup docs:
[`docs/Contributing/getting-started/building-fleet.md`](../../docs/Contributing/getting-started/building-fleet.md).
