# Fleet Tools Directory

This directory contains development, testing, and operational tools for Fleet. The tools span a wide range of purposes including API testing, load testing, MDM functionality, database management, release automation, and more. Each tool is designed to help with specific aspects of Fleet development, testing, or operations.

**If you add a new tool to this directory, please update this README with the tool's purpose and usage.**

## Overview

Tools are organized into functional categories:
- **api/**: API interaction and testing
- **loadtest/**: Performance and load testing
- **tuf/**: Update system (The Update Framework)
- **mdm/**: Mobile Device Management (Apple, Windows, Android, migrations)
- **software/**: Software and vulnerability management
- **osquery/**: osquery testing and development
- Root-level tools are generally single-purpose utilities

## Common Requirements

- Many tools require Fleet server running locally (default: `https://localhost:8080`)
- Database tools assume MySQL running (default: `localhost:3306`, user: `fleet`, password: `insecure`)
- Docker is required for many packaging and testing tools
- See individual tool READMEs for detailed documentation and requirements
- Tools in subdirectories (like `mdm/`, `tuf/`, `loadtest/`) often have their own READMEs
- MDM tools often require `-server-private-key` flag to decrypt MDM assets from the database
- Integration tests (Jira, Zendesk) require environment variables for credentials

## Getting Help

For detailed usage of specific tools:
1. Check for a README in the tool's directory
2. Run the tool with `-h` or `--help` flag
3. Examine the source code for usage comments at the top of main.go files
4. Many tools have extensive comments explaining their purpose and usage

## Common Workflows

### Local Development
```bash
# Start test osqueryd instances
cd tools/osquery && docker-compose up

# Serve files locally
go run ./tools/file-server 8081 ./my-files

# Take database snapshot before testing
go run ./tools/snapshot s

# Restore database after testing
go run ./tools/snapshot r

# Backup database
./tools/backup_db/backup.sh
```

### Testing
```bash
# Run osquery load tests
./tools/loadtest/osquery/gnuplot_osqueryd_cpu_memory.sh

# Test webhooks
go run ./tools/webhook 8082

# Check community issues
./tools/oncall/oncall.sh issues

# Test rate limiting
go run ./tools/desktop-rate-limit -fleet_url https://localhost:8080
```

### MDM Testing
```bash
# Export MDM assets
go run ./tools/mdm/assets export -key=mykey -dir=./assets

# Import MDM assets
go run ./tools/mdm/assets import -key=mykey -dir=./assets -name=scep_challenge -value="challenge"

# Send APNS push notification
go run ./tools/mdm/apple/apnspush -mysql localhost:3306 -server-private-key <key> <UUID>

# Query Apple Business Manager
go run ./tools/mdm/apple/applebmapi -mysql localhost:3306 -server-private-key <key> -org-name "My Org"

# Generate app manifest from pkg
go run ./tools/mdm/apple/appmanifest -pkg-file app.pkg -pkg-url https://example.com/app.pkg

# Decrypt disk encryption key (FileVault/BitLocker)
go run ./tools/mdm/decrypt-disk-encryption-key -cert file.crt -key file.key -value-to-decrypt <base64>

# Test Jamf migration
go run ./tools/mdm/migration/jamf -username admin -password secret -url https://jamf.example.com -port 4648
```

### Android Management
```bash
# Android management API operations (requires FLEET_DEV_ANDROID_GOOGLE_SERVICE_CREDENTIALS env var)
go run ./tools/android -command list-devices -enterprise_id <id>
go run ./tools/android -command get-device -enterprise_id <id> -device_id <id>
```

### Release Management
```bash
# Create release candidate (minor)
./tools/release/publish_release.sh -m

# Create release candidate (patch)
./tools/release/publish_release.sh

# Check TUF channel versions
go run tools/tuf/status/tuf-status.go channel-version -channel stable

# Release to TUF edge
./tools/tuf/releaser.sh  # See tuf/README.md for required env vars
```

### API Testing
```bash
# Set up environment
export FLEET_ENV_PATH=./tools/api/env

# Test API endpoints
./tools/api/fleet/me
./tools/api/fleet/queries/list
./tools/api/fleet/queries/create 'my_query' 'SELECT * FROM processes;'
```

### Integration Testing
```bash
# Test Jira integration (requires JIRA_PASSWORD env var)
go run ./tools/jira-integration \
  -jira-url https://example.atlassian.net \
  -jira-username admin@example.com \
  -jira-project-key FLEET \
  -cve CVE-2024-1234 \
  -hosts-count 5

# Test Zendesk integration (requires ZENDESK_TOKEN env var)
go run ./tools/zendesk-integration \
  -zendesk-url https://example.zendesk.com \
  -zendesk-email admin@example.com \
  -zendesk-group-id 12345 \
  -cve CVE-2024-1234
```

### Database Management
```bash
# Generate database schema
go run ./tools/dbutils/schema_generator.go ./schema.sql

# Bump migration timestamp (when PR migration is older than main)
go run ./tools/bump-migration -source-migration 20240101120000_MyMigration.go -regen-schema
```

### GitHub Management
```bash
# Search issues
./tools/github-manage/gm issues --search "is:open label:bug"

# View project items
./tools/github-manage/gm project 58 --limit 50

# View estimated tickets
./tools/github-manage/gm estimated mdm --limit 25
```

### GitOps Migration
```bash
# Migrate software YAML files to Fleet 4.74.0+ format
./tools/gitops-migrate/migrate.sh it-and-security/teams/
```

### Software & Vulnerability Testing
```bash
# Seed vulnerable software
go run ./tools/software/vulnerabilities/seed_vuln_data.go \
  --ubuntu 1 --macos 1 --windows 1 --linux-kernels 1

# Parse VEX document
go run ./tools/vex-parser <path-to-vex-file.json>
```

### Code Quality Tools
```bash
# Check cloner implementations
go run ./tools/cloner-check --check

# Update cloner implementations
go run ./tools/cloner-check --update

# Generate osquery agent options (macOS only)
go run ./tools/osquery-agent-options ./output.go
```

### Script Execution Testing
```bash
# Test Orbit script execution locally
go run ./tools/run-scripts -exec-id my-test-id -content 'echo "Hello, world!"'

# Test with multiple scripts
go run ./tools/run-scripts -scripts-count 10

# Test with scripts disabled
go run ./tools/run-scripts -scripts-disabled -content 'echo "Test"'
```

## Quick Reference Table

| Tool | Purpose | Usage |
|------|---------|-------|
| **API & Integration** | | |
| `api/` | Fleet API testing scripts using curl + jq | `export FLEET_ENV_PATH=./env && ./tools/api/fleet/me` |
| `jira-integration/` | Test Jira ticket creation | `JIRA_PASSWORD=<pwd> go run ./tools/jira-integration -jira-url <url> -jira-username <user> -jira-project-key <key> -cve CVE-2024-1234` |
| `webhook/` | Test webhook integrations | `go run ./tools/webhook 8082` |
| `zendesk-integration/` | Test Zendesk ticket creation | `ZENDESK_TOKEN=<token> go run ./tools/zendesk-integration -zendesk-url <url> -zendesk-email <email> -zendesk-group-id <id> -cve CVE-2024-1234` |
| **Database & Data** | | |
| `backup_db/` | Database backup scripts | `./tools/backup_db/backup.sh` and `./tools/backup_db/restore.sh` |
| `branch_snapshot.sh` | Auto backup/restore DB on git branch checkout | Link to `.git/hooks/post-checkout` |
| `dbutils/` | Database schema generator | `go run ./tools/dbutils/schema_generator.go <dumpfile>` |
| `mysql-replica-testing/` | MySQL replica testing | See [mysql-replica-testing/README.md](mysql-replica-testing/README.md) |
| `mysql-tests/` | MySQL testing configs | Docker configs for MySQL testing |
| `redis-stress/` | Redis stress testing | `go run` tools in directory |
| `redis-tests/` | Redis testing configs | ElastiCache and general Redis test configs |
| `snapshot/` | Database snapshot/restore tool | `go run ./tools/snapshot s` or `go run ./tools/snapshot r` |
| **Development Tools** | | |
| `app/` | Prometheus config for local dev | See `prometheus.yml` |
| `ci/` | CI helper tools (golangci-lint rules) | `rules.go` - ruleguard custom linting rules |
| `desktop/` | Fleet Desktop development tool | `go run ./tools/desktop` - builds Desktop app |
| `dialog/` | Test zenity/kdialog dialogs on Linux | `go run ./tools/dialog -dialog zenity` |
| `file-server/` | Serve local directory via HTTP | `go run ./tools/file-server 8081 /path/to/dir` |
| `oncall/` | Find community issues/PRs | `./tools/oncall/oncall.sh issues` or `./tools/oncall/oncall.sh prs` |
| **Infrastructure** | | |
| `apm-elastic/` | Elastic APM config | See [apm-elastic/README.md](apm-elastic/README.md) |
| `calendar/` | Calendar integration tools | See [calendar/README.md](calendar/README.md) |
| `fdm/` | FleetDM developer tools | `fdm <command>` - Wrapper for Fleet make targets |
| `fleet-docker/` | Fleet Docker configs | Docker configuration for Fleet |
| `github-manage/` | GitHub management automation | `./gm issues --search "is:open"` or `./gm project 58` - See [README](github-manage/README.md) |
| `github-releases/` | GitHub release tools | `go run ./tools/github-releases --last-minor-releases <n>` or `--all-cpes` |
| `gitops-migrate/` | GitOps YAML migration | `./tools/gitops-migrate/migrate.sh <teams_dir>` - See [README](gitops-migrate/README.md) |
| `mailpit/` | Local email testing | Mailpit SMTP server for local dev (uses `auth.txt` config) |
| `open/` | Test "open" package | `go run ./tools/open -url <url>` - Opens URL in default browser |
| `percona/` | Percona testing | Percona MySQL testing configs - See [percona/test/README.md](percona/test/README.md) |
| `sentry-self-hosted/` | Self-hosted Sentry | See [sentry-self-hosted/README.md](sentry-self-hosted/README.md) |
| `smtp4dev/` | Local SMTP testing | SMTP4Dev server with TLS certs for email testing |
| `telemetry/` | Jaeger + Prometheus for tracing | `docker compose up` - See [telemetry/README.md](telemetry/README.md) |
| `terraform/` | Terraform provider for Fleet teams | `make install && make apply` - See [terraform/README.md](terraform/README.md) |
| **MDM Tools** | | |
| `android/` | Android management API tool | `go run ./tools/android -command <cmd> -enterprise_id <id> -device_id <id>` |
| `mdm/apple/applebmapi/` | Query Apple Business Manager API | `go run ./tools/mdm/apple/applebmapi -mysql localhost:3306 -server-private-key <key> -org-name <org>` |
| `mdm/apple/appmanifest/` | Generate app manifest XML from .pkg | `go run ./tools/mdm/apple/appmanifest -pkg-file app.pkg -pkg-url https://example.com/app.pkg` |
| `mdm/apple/apnspush/` | Send APNS push to enrolled devices | `go run ./tools/mdm/apple/apnspush -mysql localhost:3306 -server-private-key <key> <HOST_UUID>` |
| `mdm/apple/loadtest/` | MDM load testing | `go run ./tools/mdm/apple/loadtest` |
| `mdm/apple/macos-vm-auto-enroll/` | Auto-enroll macOS VMs in MDM | `./tools/mdm/apple/macos-vm-auto-enroll/macos-vm-auto-enroll.sh` |
| `mdm/apple/setupexperience/` | Test setup experience flows | `go run ./tools/mdm/apple/setupexperience` |
| `mdm/assets/` | Export/import MDM assets (SCEP, APNS, etc.) | `go run ./tools/mdm/assets export -key=<key> -dir=<dir>` or `import` |
| `mdm/decrypt-disk-encryption-key/` | Decrypt FileVault/BitLocker keys | `go run ./tools/mdm/decrypt-disk-encryption-key -cert file.crt -key file.key -value-to-decrypt <base64>` |
| `mdm/make_cfg_profiles.sh` | Generate configuration profiles | `./tools/mdm/make_cfg_profiles.sh` |
| `mdm/migration/echo/` | Echo MDM migration tools | `go run ./tools/mdm/migration/echo` |
| `mdm/migration/jamf/` | Jamf to Fleet migration webhook | `go run ./tools/mdm/migration/jamf -username <user> -password <pwd> -url <jamf_url>` |
| `mdm/migration/kandji/` | Kandji migration tools | `go run ./tools/mdm/migration/kandji` |
| `mdm/migration/mdmproxy/` | MDM proxy for migration testing | `./tools/mdm/migration/mdmproxy/entrypoint.sh` |
| `mdm/migration/micromdm/` | MicroMDM migration tools | See [mdm/migration/micromdm/README.md](mdm/migration/micromdm/README.md) |
| `mdm/migration/simplemdm/` | SimpleMDM migration tools | `go run ./tools/mdm/migration/simplemdm` |
| `mdm/windows/bitlocker/` | BitLocker key management | Go utilities for BitLocker |
| `mdm/windows/poc-mdm-server/` | PoC Windows MDM server | See [mdm/windows/poc-mdm-server/README.md](mdm/windows/poc-mdm-server/README.md) |
| `mdm/windows/programmatic-enrollment/` | Windows MDM enrollment | `go run ./tools/mdm/windows/programmatic-enrollment` |
| `windows-mdm-enroll/` | Windows MDM enrollment | Enrollment utilities for Windows |
| **Other Utilities** | | |
| `bump-migration/` | Bump migration timestamp | `go run ./tools/bump-migration -source-migration <file> [-regen-schema]` |
| `cis/` | CIS benchmark tools | `python tools/cis/CIS-Benchmark-diff.py` |
| `cloner-check/` | Verify fleet.Cloner implementations | `go run ./tools/cloner-check --check` or `--update` |
| `luks/luks/` | LUKS key escrow tool (Linux only) | `go run ./tools/luks/luks` - Adds escrow key to LUKS partition |
| `luks/lvm/` | Find root disk for LVM (Linux only) | `go run ./tools/luks/lvm` - Detects root partition path |
| `makefile-support/` | Makefile helper utilities | `./tools/makefile-support/makehelp.sh` - Generate help text |
| `osquery-agent-options/` | Generate osquery agent options struct | `go run ./tools/osquery-agent-options <output-file>` - macOS only |
| `run-scripts/` | Test Orbit script execution | `go run ./tools/run-scripts -exec-id <id> -content 'echo "Hello"'` |
| **Packaging & Installers** | | |
| `bomutils-docker/` | Docker image for BOM utils (macOS pkg) | Docker build for BOM utilities |
| `team-builder/` | Bulk team creation + installer generation | `./build_teams.sh -s teams.txt -u fleet.example.com` |
| `wix-docker/` | Docker image for WiX (Windows MSI) | Docker build for WiX toolset |
| **Release & Distribution** | | |
| `fleetctl-docker/` | Docker image for fleetctl packaging | `docker run fleetdm/fleetctl package --type=pkg` |
| `fleetctl-npm/` | NPM package for fleetctl | See [fleetctl-npm/README.md](fleetctl-npm/README.md) |
| `fleetd-linux/` | Linux fleetd packaging | Packaging scripts for Linux fleetd |
| `release/` | Fleet release automation | `./tools/release/publish_release.sh -m` - See [release/README.md](release/README.md) |
| `sign-fleetctl/` | Code signing for fleetctl | Signing utilities for fleetctl binaries |
| `tuf/` | TUF repository management for fleetd updates | `./tools/tuf/releaser.sh` - See [tuf/README.md](tuf/README.md) |
| `tuf/migrate/` | TUF migration tools | Migration scripts for TUF updates |
| `tuf/status/` | Query TUF repository status | `go run tools/tuf/status/tuf-status.go channel-version -channel stable` |
| `tuf/test/` | TUF testing scripts | `./tools/tuf/test/main.sh` - See [tuf/test/README.md](tuf/test/README.md) |
| **Security & Auth** | | |
| `app-sso-platform/` | Test app_sso_platform table (macOS) | `go run ./tools/app-sso-platform <extensionID> <realm>` |
| `inspect-cert/` | Certificate inspection | Certificate inspection utilities |
| `msal/` | Microsoft Entra Device ID sample app | Obj-C reference app for MSAL |
| `saml/` | SAML SSO testing config | Edit `users.php` for test users |
| **Software & Vulnerabilities** | | |
| `custom-package-parser/` | Parse custom software packages | See [custom-package-parser/README.md](custom-package-parser/README.md) |
| `nvd/` | NVD (National Vulnerability Database) tools | See [nvd/nvdvuln/README.md](nvd/nvdvuln/README.md) |
| `software/icons/` | Software icon management | See [software/icons/README.md](software/icons/README.md) |
| `software/packages/` | Software package utilities | See [software/packages/README.md](software/packages/README.md) |
| `software/vulnerabilities/` | Seed vulnerable software for dev | `go run ./tools/software/vulnerabilities/seed_vuln_data.go --ubuntu 1 --macos 1 --windows 1` |
| `software/vulnerabilities/performance_test/` | Vuln performance testing | See [software/vulnerabilities/performance_test/README.md](software/vulnerabilities/performance_test/README.md) |
| `vex-parser/` | Parse OpenVEX documents | `go run ./tools/vex-parser <vex-file>` |
| **Testing & Load Testing** | | |
| `desktop-rate-limit/` | Test Fleet Desktop rate limiting | `go run ./tools/desktop-rate-limit -fleet_url https://localhost:8080` |
| `kubequery/` | Kubequery + Fleet config | `kubectl apply -f kubequery-fleet.yml` |
| `loadtest/fleetd_labels/` | Apply manual labels for load testing | `go run ./tools/loadtest/fleetd_labels` |
| `loadtest/osquery/` | Load test osquery on macOS/Windows/Linux | See [loadtest/osquery/README.md](loadtest/osquery/README.md) |
| `loadtest/scripts_and_profiles/` | Load test scripts and profiles | `go run ./tools/loadtest/scripts_and_profiles` |
| `loadtest/unified_queue/` | Load test unified queue story | See [loadtest/unified_queue/README.md](loadtest/unified_queue/README.md) |
| `osquery/` | Containerized osqueryd testing | `docker-compose up` - See [osquery/README.md](osquery/README.md) |
| `osquery-testing/` | osquery integration tests | `docker-compose up` in directory |
| `test-certs/` | Fake certificate chain for TLS testing | See [test-certs/README.md](test-certs/README.md) |
| `test-orbit-mtls/` | Test Orbit mTLS | Scripts for mTLS testing |
| `test_extensions/` | Test osquery extensions (hello_world) | `./tools/test_extensions/hello_world/build.sh` |
| **Testing Data** | | |
| `seed_data/` | Seed test data | Test data seeding scripts |
| `testdata/` | Test fixtures and data | Static test fixtures |
