# Seed data for local development

Scripts and YAML files to populate a local Fleet instance with realistic users, policies, and reports for testing. Especially useful after a `make db-reset` to quickly get back to a working state.

## Prerequisites

- A running Fleet dev server (`make serve` or equivalent)
- A premium license (required for fleets)
- `fleetctl` built (`make build` or set `FLEETCTL` to your binary path)
- `python3` (used for JSON parsing in the scripts)

## Quick start

### 1. Set up your env file

If you just reset your database, create your admin account first by completing Fleet setup at https://localhost:8080.

Copy the example env file and fill in your values:

```bash
cp tools/seed/DO_NOT_COMMIT_ENV_FILE.example tools/seed/DO_NOT_COMMIT_ENV_FILE
```

Edit `tools/seed/DO_NOT_COMMIT_ENV_FILE` with your local server details:

```bash
export SERVER_URL=https://localhost:8080
export CURL_FLAGS='-k -s'
export TOKEN=<your-api-token>
export SEED_PASSWORD=<password-for-all-seed-users>
```

Get your API token from **My account** in the Fleet UI.

Then export the path:

```bash
export FLEET_ENV_PATH=tools/seed/DO_NOT_COMMIT_ENV_FILE
```

### 2. Clean up existing data (optional)

If you already have manually created users, policies, or reports and want a clean slate, run this first. It deletes all policies, reports, and users except your admin account (ID 1). Requires the env file from step 1.

```bash
bash tools/seed/clean-seed-data.sh
```

### 3. Run the seeds

```bash
bash tools/seed/seed-users-and-fleets.sh
```

This seeds everything in one shot: users, fleets, global policies, global reports, fleet-scoped policies, and fleet-scoped reports. The script configures `fleetctl` with your API token automatically — no separate `fleetctl login` needed.

The script expects `fleetctl` at `./build/fleetctl`. If yours is elsewhere:

```bash
FLEETCTL=/usr/local/bin/fleetctl bash tools/seed/seed-users-and-fleets.sh
```

Re-running is safe — existing users are skipped and policies/reports are upserted.

## What gets created

### Users (`seed-users-and-fleets.sh`)

Creates 12 users across global, fleet-scoped, and API-only roles. Regular users use `$SEED_PASSWORD`.

| User | Email | Role |
|------|-------|------|
| Anna G. Admin | anna@organization.com | Global admin |
| Mary G. Maintainer | mary@organization.com | Global maintainer |
| Oliver G. Observer | oliver@organization.com | Global observer |
| Opal G. Observer+ | opal@organization.com | Global observer+ |
| Tessa G. Technician | tessa@organization.com | Global technician |
| Marco Mixed Roles | marco@organization.com | Observer (Workstations), Maintainer (Mobile devices) |
| Anita T. Admin | anita@organization.com | Admin (Workstations) |
| Manny T. Maintainer | manny@organization.com | Maintainer (Workstations) |
| Toni T. Observer | toni@organization.com | Observer (Workstations) |
| Topanga T. Observer+ | topanga@organization.com | Observer+ (Workstations) |
| Apollo G. API-only | apollo@organization.com | Global maintainer, API-only (full access) |
| Rosie Restricted (API only) | auto-generated | Global admin, API-only (restricted to GET hosts) |

API-only users created via `/users/api_only` have their API token printed once at creation. Save it — it cannot be retrieved later.

### Global policies (`standard-policies.yml`)

41 policies extracted from the standard query library (`docs/01-Using-Fleet/standard-query-library/standard-query-library.yml`). Covers macOS, Windows, and Linux.

### Global reports (`standard-reports.yml`)

Reports extracted from `docs/queries.yml`. Covers inventory, detection, and informational queries across platforms.

### Fleet-scoped policies

Applied automatically by `seed-users-and-fleets.sh` using `--policies-fleet`:

- **`fleet-workstations-policies.yml`** — 5 macOS policies (Gatekeeper, FileVault, SIP, Firewall, Screen lock)
- **`fleet-mobile-policies.yml`** — 5 personal laptop policies (SSH keys, Linux FDE, Linux antivirus, 1Password emergency kit, Docker up to date)

### Fleet-scoped reports

Applied automatically by `seed-users-and-fleets.sh`:

- **`fleet-workstations-reports.yml`** — 4 macOS reports (Apple Intelligence, crashes, battery health, installed software)
- **`fleet-mobile-reports.yml`** — 4 personal laptop reports (Chrome extensions, memory hogs, unencrypted SSH keys, VS Code extensions)
