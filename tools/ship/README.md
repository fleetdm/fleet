# Fleet ship

`ship` is an interactive terminal UI (TUI) that sets up and runs a local Fleet
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
2. Open your Terminal app (go to Applications, and double-click Terminal), then paste this command and press enter: `brew install go yarn ngrok`
3. In Terminal, paste the command below, to install Node Version Manager ([nvm](https://github.com/nvm-sh/nvm)):
   ```sh
   curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
   ```
4. Quit your Terminal app. If it's open press CMD + Q, then open it again.
5. In Terminal, paste this command, and press enter: `nvm install v24.10.0`.
6. Go to [ngrok.com](https://ngrok.com), and sign up. You can use your Google Workspace or GitHub account.
7. Go to <https://dashboard.ngrok.com/domains> and select
   **New Domain** to reserve a static domain. Free accounts get one (e.g. `bla-bla-superior.ngrok-free.app`).
8. Grab your auth token at
   <https://dashboard.ngrok.com/get-started/your-authtoken>.
9. In Terminal, paste this command `ngrok config add-authtoken <replace_with_token>`. Make sure to add token from step above instead of the `<replace_with_token>`.

### How to run `ship`

To run this tool you need to open the Terminal app, and change directory to your Fleet repo. Usually when you run terminal you're at home directory, and if your Fleet repo is downloaded to `~/projects/fleet` then you would run this command in your Terminal: `cd projects/fleet`. Once you ran the command, you would see something like `somename@examplemac fleet %`.

Once you navigated to your Fleet repo, just type `make ship` and press enter. This will start interactive development environment.

Once you setup everything once, you just navigate to your repo and run `make ship` to start local Fleet server again.

## First-time setup

The first `make ship` after a fresh install does some extra work that
subsequent runs skip:

1. **Wizard** runs and asks you to provide ngrok domain, server private key, and if you want premium or free instance of Fleet.
2. You can find ngrok domain at <https://dashboard.ngrok.com/domains> (but paste it without `https://`).
3. For server private key, run this command in Terminal app: `openssl rand -base64 32`. It will generate a random string that you can use as server private key.
4. Copy this key and create 1Password item to store this private key, because you can always reuse this one for your local instances.
5. In the wizard paste that key you saved, and proceed to next step.
6. Usually you'll need to use Fleet Premium, so you can just leave it because it's default.
7. After the setup, it will start all necessary services to spin up local Fleet instance.
3. Once Fleet is running, **open the public URL** from `ship` tool (e.g.
   `https://fleet-pm-jane.ngrok-free.app`) in your browser.

## Prerequisites

`make ship` runs a doctor screen on every launch that checks if all required dependencies are installed. If something is missing on your computer, you can find install instructions below.

| Dependency | macOS install |
|---|---|
| Docker Desktop | Download from [docker.com](https://www.docker.com/products/docker-desktop) and install the app. |
| Xcode Command Line Tools | Run `xcode-select --install` command in your Terminal app. |
| Homebrew | Follow instructions at [brew.sh](https://brew.sh) |
| Go | Run command in your Terminal app: `brew install go` |
| nvm | Run command in your Terminal app: `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh \| bash` |
| Node.js v24.10.0 | Run command in your Terminal app: `nvm install v24.10.0`, but make sure to navigate to your fleet repo, using `cd` command. See (#how-to-run-ship) on how to navigate to repo.|
| Yarn | Run command in your Terminal app: `brew install yarn` |
| ngrok | Run command in your Terminal app: `brew install ngrok` |
| Rosetta 2 (Apple Silicon only) | Run command in your Terminal app: `softwareupdate --install-rosetta --agree-to-license` |

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

### Browser certificate warning

The first time you visit the public URL, your browser will show a "Not
Secure" / "self-signed certificate" warning. That's expected — Fleet's
`--dev` mode generates a temporary self-signed cert for the local server, and
ngrok forwards to it. Click through (in Chrome: **Advanced → Proceed to…**).
The site is your own machine.

### Creating the admin user

When the database is empty (you spun up your first Fleet instance) Fleet's UI walks you through creating an admin account. Save those in 1Password too along with server private key. You'll need them every login, and again if you ever reset the database.

## Troubleshooting with coding agents

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
