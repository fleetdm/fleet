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

Creates 15 users across global, fleet-scoped, and API-only roles. Regular users use `$SEED_PASSWORD`.

| User | Email | Role | Persona |
|------|-------|------|---------|
| Anna G. Admin | anna@organization.com | Global admin | IT Director. Owns the Fleet instance org-wide. Manages users, sets global policies, approves fleet-level admin access. Her priority is consistent security posture across every device. |
| Mary G. Maintainer | mary@organization.com | Global maintainer | Senior security engineer. Writes and maintains policies and reports across all fleets. Doesn't need user management — Anna handles that. Cares about detection coverage and keeping reports accurate. |
| Oliver G. Observer | oliver@organization.com | Global observer | Compliance manager. Read-only visibility into everything for audits and executive reporting. Never changes configuration — just needs to verify the org is meeting its compliance commitments. |
| Opal G. Observer+ | opal@organization.com | Global observer+ | Security analyst on the incident response team. Needs everything Oliver has, plus the ability to run live queries during investigations. Can't change policies or configuration — just investigates. |
| Tessa G. Technician | tessa@organization.com | Global technician | IT support lead. Helps end users across every fleet — locks lost devices, pushes software, troubleshoots MDM enrollment. Works across all fleets because the helpdesk isn't organized by device type. |
| Gina G. GitOps | gina@organization.com | Global gitops | Platform engineer who manages Fleet's global configuration as code. Pushes policy and report changes via CI/CD, rarely touches the UI. Owns the GitOps repo and reviews PRs to it. |
| Marco Mixed Roles | marco@organization.com | Observer (Workstations), Maintainer (Mobile devices) | IT contractor who was brought in to stand up the mobile device program. Full maintainer access on Mobile devices (his project), but only observer on Workstations so he can reference existing policies without changing them. |
| Anita T. Admin | anita@organization.com | Admin (Workstations) | Workstations fleet owner. A senior IT manager who runs the corporate Mac program. Manages her fleet's users and settings, but has no access to Mobile devices — that's Marco's domain. |
| Manny T. Maintainer | manny@organization.com | Maintainer (Workstations) | Mac sysadmin on Anita's team. Writes fleet-specific policies, manages software deployments, maintains Workstations reports. Day-to-day hands on the corporate Mac fleet. |
| Toni T. Observer | toni@organization.com | Observer (Workstations) | Finance team lead. Has read-only access to the Workstations fleet so she can pull asset inventory numbers for quarterly hardware budgeting. Doesn't need to touch anything else. |
| Topanga T. Observer+ | topanga@organization.com | Observer+ (Workstations) | Senior engineer who occasionally troubleshoots deep macOS issues. Needs to run ad-hoc queries against the Workstations fleet to diagnose problems, but shouldn't change policies or configuration — that's Manny's job. |
| Terry T. Technician | terry@organization.com | Technician (Workstations) | Helpdesk technician dedicated to the Workstations fleet. Handles end-user requests — reinstalls software, wipes machines for offboarding, resets MDM profiles. No access to Mobile devices. |
| Gordon T. GitOps | gordon@organization.com | GitOps (Workstations) | The CI/CD service account identity for the Workstations fleet repo. Gordon is the human who maintains that repo, but this account is what the pipeline authenticates as. Scoped only to Workstations. |
| Apollo G. API-only (full access) | apollo@organization.com | Global maintainer, API-only | Service account for the SIEM integration. Pulls host data, vulnerability info, and policy results into Splunk on a scheduled sync. Full read/write API access but no human ever logs into the UI with it. |
| Reggie G. API-only (restricted) | auto-generated | Global admin, API-only (restricted to GET hosts) | Service account for an internal asset dashboard. Only needs to list hosts and get host details — nothing else. Locked down to two endpoints so a compromised token has minimal blast radius. |

API-only users created via `/users/api_only` have their API token printed once at creation. Save it — it cannot be retrieved later.

### Global policies (`standard-policies.yml`)

41 policies extracted from the standard query library (`docs/01-Using-Fleet/standard-query-library/standard-query-library.yml`). Covers macOS, Windows, and Linux.

### Global reports (`standard-reports.yml`)

Reports extracted from `docs/queries.yml`. Covers inventory, detection, and informational queries across platforms.

### Fleet-scoped policies

Applied automatically by `seed-users-and-fleets.sh` using `--policies-fleet`:

- **`fleet-workstations-policies.yml`** — 5 managed corporate macOS policies (SSH disabled, AirDrop disabled, Bluetooth sharing disabled, login window config, Time Machine encrypted)
- **`fleet-mobile-policies.yml`** — 5 personal laptop/BYOD policies (no plaintext passwords in dotfiles, screen saver password, Find My Mac, automatic OS updates, no world-writable files)

### Fleet-scoped reports

Applied automatically by `seed-users-and-fleets.sh`:

- **`fleet-workstations-reports.yml`** — 4 corporate macOS reports (FileVault status, MDM enrollment details, Gatekeeper config, managed profiles)
- **`fleet-mobile-reports.yml`** — 4 personal laptop/BYOD reports (Homebrew packages, disk space usage, SSH agent keys, browser extensions by user)
