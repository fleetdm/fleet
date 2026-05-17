# screencap

A CLI tool that automatically captures screenshots of Fleet UI pages using a headless Chrome browser. It supports a built-in workflow that covers all major pages and modals, as well as custom recorded workflows that can be saved and replayed.

## Prerequisites

- **Go** (1.25+)
- **Google Chrome** installed (the tool uses Chrome via the Chrome DevTools Protocol)
- A running Fleet instance to capture

## Building

```bash
cd tools/screencap
make build
```

This produces a `screencap` binary in the current directory.

To remove the binary:

```bash
make clean
```

## Authentication

The tool persists a Chrome profile at `~/.fleet/screencap-profile/`, so sessions survive across runs. On the first run (or when your session expires), authenticate using one of these methods:

| Flag | Description |
|---|---|
| `-sso` | Opens a visible browser for SSO login. Complete the flow, then press ENTER. |
| `-email` / `-password` | Logs in with email and password. |
| `-cookie` / `-cookie-name` | Sets a session cookie directly (cookie name defaults to `Fleet-Session`). |
| `-login` | Re-authenticate interactively (opens a visible browser). |
| *(no auth flag)* | Reuses the saved session from a previous run. |

## Usage

### Run the built-in full workflow

Captures all major Fleet pages (dashboard, hosts, queries, policies, software, controls, settings, account) including modals and tabs:

```bash
./screencap https://fleet.example.com
```

Or with authentication:

```bash
./screencap -sso https://fleet.example.com
./screencap -email admin@example.com -password secret https://fleet.example.com
```

### Run a saved workflow

```bash
./screencap -workflow my-flow https://fleet.example.com
```

### Record a custom workflow

Opens a visible browser where you click through the pages you want to capture. Clicks, radio buttons, checkboxes, tabs, and tooltip hovers are recorded automatically.

```bash
./screencap -record my-flow https://fleet.example.com
```

During recording:
- Browse around in the browser window
- Press **ENTER** to save a screenshot step (optionally type a name first)
- Type **done** to finish and save the workflow

Workflow files are saved as JSON in the `workflows/` directory and can be committed to the repo.

### List saved workflows

```bash
./screencap -list
```

### Additional flags

| Flag | Default | Description |
|---|---|---|
| `-wait-time-seconds` | `6` | Seconds to wait for each page to load before capturing. |

## Output

Screenshots are saved to `screenshots/<timestamp>-<workflow>-<host>/` as numbered PNG files. Each page produces one or more images (one per viewport height if the page scrolls), and modals/tabs get their own files.

Example output structure:

```
screenshots/2026-03-11_113840-full-fleet.example.com/
  dashboard-1.png
  dashboard-modal-add-hosts-1.png
  dashboard-modal-add-hosts-macos-1.png
  hosts-manage-1.png
  hosts-manage-2.png
  ...
```
