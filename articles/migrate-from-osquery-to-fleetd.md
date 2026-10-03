# Migrate from osquery to fleetd

Organizations using [osquery](https://www.osquery.io/) with [Fleet](https://fleetdm.com/docs/get-started/anatomy#osquery) may want to migrate to Fleet's agent (fleetd), which includes osquery.

fleetd includes the core osquery table schema, plus additional tables added specifically for endpoint ops and device management with Fleet. The exact set of core tables depends on the osquery version fleetd ships, so compare it with the osquery version you run today before migrating.

The workflow below is intended to minimize data loss from hosts and preserve customized Fleet server-side configuration when migrating to fleetd.

## Workflow

The high-level steps to complete the migration are:

1. **Inventory your tables and extensions.** Confirm that every table you rely on today is available in fleetd. Compare your current osquery version with the version fleetd ships, and check the osquery release notes for tables added or removed between them (this matters most if you're on an older 4.x release). Account for any Automatic Table Construction (ATC) definitions and custom extensions.

   Deploying extensions through fleetd's `extensions` agent option requires a custom TUF auto-update server, which is available in Fleet Premium. If hosts depend on custom extensions, plan for this before migrating them. See:

   - [Automatic Table Construction | osquery](https://osquery.readthedocs.io/en/stable/deployment/configuration/#automatic-table-construction)
   - [Auto table construction | Fleet](https://fleetdm.com/docs/configuration/agent-configuration#auto-table-construction) (configured under `overrides`; see the [overrides example](https://fleetdm.com/docs/configuration/agent-configuration#overrides))
   - [Extensions | Fleet](https://fleetdm.com/docs/configuration/agent-configuration#extensions)
   - [osquery Extensions](https://osquery.readthedocs.io/en/stable/development/osquery-sdk/#extensions)

2. **Translate your osquery configuration to Fleet agent options.** This includes any "configuration as code" objects, such as flagfiles. Keep in mind:

   - `options` are applied without restarting fleetd.
   - `command_line_flags` are only applied when fleetd restarts.
   - Leaving `command_line_flags` out of agent options preserves each host's existing osquery flags. Setting it to `{}` or `null` **clears all local osquery flags** on hosts. Don't use an empty value as a placeholder while translating.
   - `command_line_flags` isn't supported inside `overrides`.
   - Agent options are validated against the latest osquery version. If you pin an older osquery version, you may need `fleetctl apply --force` to apply it.

   See:

   - [Configuring an osquery deployment](https://osquery.readthedocs.io/en/stable/deployment/configuration/)
   - [osquery command-line flags](https://osquery.readthedocs.io/en/stable/installation/cli-flags/)
   - [Agent configuration | Fleet](https://fleetdm.com/docs/configuration/agent-configuration)

   For a worked example, see [Example: migrating into the Workstations fleet](#example-migrating-into-the-workstations-fleet).

3. **Confirm how migrated hosts will match existing host records.** When a host re-enrolls through fleetd, Fleet should match it to its existing host record rather than create a duplicate. Review the Fleet server's `osquery.host_identifier` setting and verify the behavior on your test host before rolling out. See:

   - [Fleet server configuration | Fleet](https://fleetdm.com/docs/configuration/fleet-server-configuration)

4. **Pilot on a test host.** Ideally, enroll the test host into a dedicated [fleet](https://fleetdm.com/guides/fleets) (a Fleet Premium feature) so you can apply the translated agent options without affecting other hosts (see [Set up a pilot fleet](#set-up-a-pilot-fleet)). On the test host:

   - Confirm osquery has sent its buffered results (see [Avoiding data loss during cutover](#avoiding-data-loss-during-cutover)).
   - Stop and uninstall osquery.
   - Install fleetd using the fleet's enroll secret.
   - Confirm the host checks in, appears as a single host record, and returns report and policy results. Then confirm logs arrive at your log destination.

   See:

   - [Enroll hosts | Fleet](https://fleetdm.com/guides/enroll-hosts)

5. **Roll out in waves.** Once the pilot is healthy, repeat the cutover for the remaining hosts in batches. Osquery uninstallation steps differ by platform. See:

   - [Installing osquery on Linux](https://osquery.readthedocs.io/en/stable/installation/install-linux/)
   - [Installing osquery on macOS](https://osquery.readthedocs.io/en/stable/installation/install-macos/) (includes a "Removing osquery" section)
   - [Installing osquery on Windows](https://osquery.readthedocs.io/en/stable/installation/install-windows/)

## Example: migrating into the Workstations fleet

This example uses Fleet's own [Workstations fleet config](https://github.com/fleetdm/fleet/blob/main/it-and-security/fleets/workstations.yml) as the migration target. It shows how a typical osquery deployment's flags, config, and schedule map into a fleet YAML file managed with GitOps. Relative paths such as `../lib/...` follow the layout of that repository. Secrets use environment variables (the Workstations file uses `$DOGFOOD_...` names; substitute your own).

### Starting point: the existing osquery deployment

Assume hosts currently run osquery with this flagfile (`/etc/osquery/osquery.flags`):

```
--tls_hostname=fleet.example.com
--enroll_secret_path=/etc/osquery/enroll_secret
--tls_server_certs=/etc/osquery/fleet.pem
--config_plugin=tls
--logger_plugin=tls
--distributed_plugin=tls
--disable_events=false
--enable_file_events=true
--events_expiry=3600
--watchdog_memory_limit=500
```

And this local config (`/etc/osquery/osquery.conf`), used for settings not served by Fleet:

```json
{
  "decorators": {
    "load": [
      "SELECT uuid AS host_uuid FROM system_info;"
    ]
  },
  "file_paths": {
    "ssh_config": ["/etc/ssh/%%"]
  },
  "schedule": {
    "crontab": {
      "query": "SELECT command, path, minute, hour FROM crontab;",
      "interval": 3600,
      "platform": "posix",
      "removed": false
    },
    "listening_ports": {
      "query": "SELECT pid, port, protocol, address FROM listening_ports WHERE port != 0;",
      "interval": 3600,
      "snapshot": true
    }
  }
}
```

### Translate command-line flags

Sort the flagfile into two groups:

- **Connection and enrollment flags** (`--tls_hostname`, `--enroll_secret_path`, `--tls_server_certs`, `--config_plugin`, `--logger_plugin`, `--distributed_plugin`) are managed by fleetd. Don't carry them over. fleetd sets them from the options you pass to `fleetctl package` (see [Set up a pilot fleet](#set-up-a-pilot-fleet)).
- **Behavior flags** (eventing, watchdog, and similar) go under `agent_options.command_line_flags`.

The Workstations fleet already sets several behavior flags, so merge into its existing `command_line_flags` block rather than adding a second one. Where both set the same flag, decide which value wins:

```yaml
agent_options:
  command_line_flags:
    disable_events: false          # already set in workstations.yml
    enable_file_events: true       # already set in workstations.yml
    events_expiry: 86400           # workstations.yml uses 86400; the old flagfile used 3600
    events_max: 50000
    events_optimize: true
    watchdog_memory_limit: 500     # raised from 350 to match the old deployment
    watchdog_utilization_limit: 130
    # ...keep the remaining existing flags (EndpointSecurity, audit, ETW) unchanged
```

> `command_line_flags` only takes effect when fleetd restarts. Don't set it to `{}` while you work on it, because that clears every local osquery flag on hosts in the fleet.

### Translate decorators and file integrity monitoring

Decorators and `file_paths` go under `agent_options.config`, alongside the existing `options`:

```yaml
agent_options:
  config:
    options:
      # existing workstations.yml options, unchanged
      pack_delimiter: /
      logger_tls_period: 10
      distributed_plugin: tls
      disable_distributed: false
      logger_tls_endpoint: /api/osquery/log
      distributed_interval: 10
      distributed_tls_max_attempts: 3
    decorators:
      load:
        - SELECT uuid AS host_uuid FROM system_info;    # already present; don't duplicate
        - SELECT hostname AS hostname FROM system_info;
    file_paths:
      system_binaries:
        - /usr/bin/%%
        - /usr/sbin/%%
        - /bin/%%
        - /sbin/%%
      launch_items:
        - /Library/LaunchAgents/%%
        - /Library/LaunchDaemons/%%
        - /Users/%/Library/LaunchAgents/%%
      etc:
        - /etc/%%
        - /private/etc/%%
      ssh_config:                  # migrated from the old osquery.conf
        - /etc/ssh/%%
    exclude_paths:
      etc:
        - /private/etc/cups/%%
        - /private/etc/newsyslog.d/%%
```

`/etc/ssh/%%` already falls under the existing `etc` category, so this entry is only worth adding if you want SSH changes reported under a separate `category` value in `file_events`. Otherwise, drop it.

File integrity monitoring only produces results when `disable_events: false` and `enable_file_events: true` are set in `command_line_flags`, which the Workstations fleet already does.

> If you later add platform `overrides` (for example, to add an ATC table on macOS), remember that an override replaces the entire `config` for that platform. The macOS hosts would lose the `file_paths` and `decorators` above unless you repeat them inside the override.

### Translate the schedule to reports

Each entry in the osquery `schedule` becomes a Fleet report. The fields map as follows:

| osquery schedule | Fleet report |
| --- | --- |
| key (e.g. `crontab`) | `name` |
| `query` | `query` |
| `interval` | `interval` (seconds; `0` means the report isn't scheduled) |
| `snapshot: true` | `logging: snapshot` |
| `removed: false` | `logging: differential_ignore_removals` |
| neither set | `logging: differential` |
| `platform` | `platform` (comma-separated: `darwin`, `linux`, `windows`) |
| `version` | `min_osquery_version` |

To keep sending scheduled results to your log destination, as the old osquery schedule did, set `automations_enabled: true`.

Before creating a report, check whether the Workstations fleet already has an equivalent. It already includes `../lib/all/reports/collect-listening-ports.yml`, so the old `listening_ports` entry can likely be dropped once you confirm that report's columns, interval, logging type, and automations setting meet your needs.

The `crontab` entry has no equivalent, so add it as a new report file, for example `lib/all/reports/collect-crontab-entries.yml`:

```yaml
- name: Collect crontab entries
  description: Migrated from the osquery schedule entry "crontab".
  query: SELECT command, path, minute, hour FROM crontab;
  interval: 3600
  platform: darwin,linux
  logging: differential_ignore_removals
  automations_enabled: true
  observer_can_run: false
```

Then reference it from `workstations.yml`:

```yaml
reports:
  # ...existing reports
  - path: ../lib/all/reports/collect-listening-ports.yml
  - path: ../lib/all/reports/collect-crontab-entries.yml
```

In result logs, Fleet names each result after the report, not the old osquery schedule key. Update any downstream parsers, dashboards, or alerts that match on the old names, and check the exact `name` format in your pilot host's logs.

### Optionally pin the osquery version during the pilot

The Workstations fleet tracks the `stable` channel for every fleetd component. If you need table behavior to match your current osquery release while you validate the migration, pin `osqueryd` in the pilot fleet and return it to `stable` afterward:

```yaml
agent_options:
  update_channels:
    osqueryd: '5.x.y'   # replace with a version that exists in the update server's osqueryd channel
    orbit: stable
    desktop: stable
```

### Set up a pilot fleet

Rather than enrolling the test host straight into Workstations, create a pilot fleet from a copy of `workstations.yml` with its own name and enroll secret. For example, `fleets/workstations-pilot.yml`:

```yaml
name: "💻 Workstations (pilot)"
settings:
  secrets:
    - secret: $WORKSTATIONS_PILOT_ENROLL_SECRET
  # ...remaining settings copied from workstations.yml
agent_options:
  # translated agent options from the sections above
reports:
  # translated reports from the section above
```

Preview the change, then apply it:

```sh
fleetctl gitops -f default.yml -f fleets/workstations-pilot.yml --dry-run
fleetctl gitops -f default.yml -f fleets/workstations-pilot.yml
```

Build a fleetd package that enrolls into the pilot fleet. Use `--type=msi`, `deb`, or `rpm` for other platforms. If the old deployment relied on `--tls_server_certs` for a private CA, pass the certificate with `--fleet-certificate` instead:

```sh
fleetctl package --type=pkg \
  --fleet-url=https://fleet.example.com \
  --enroll-secret="$WORKSTATIONS_PILOT_ENROLL_SECRET"
```

After the pilot host checks in and its data looks correct, merge the translated `agent_options` and `reports` into `workstations.yml`, apply it, and roll fleetd out to the remaining hosts using the Workstations enroll secret.

## Best practices

### Service downtime

Migrating from osquery to fleetd is similar to an upgrade, and osquery doesn't provide its own mechanism to stop an old service, swap versions, and restart (see the [osquery note on upgrading from 4.x to 5.x](https://osquery.readthedocs.io/en/stable/installation/install-macos/#note-on-upgrading-from-osquery-4x-to-5x) on macOS). Handle the order of operations yourself.

Once your agent options are translated, on each host:

- Confirm osquery has sent its buffered results.
- Stop the osquery service.
- Remove osquery files and dependencies.
- Install and start fleetd.

This order prevents a race condition in which both agents check in to Fleet and overwrite each other's host data on the server.

### Avoiding data loss during cutover

Osquery buffers scheduled results locally before sending them to its logger. If you stop osquery and delete its database directory before the buffer is sent, those results are lost. Before removing osquery's files, wait at least one logger period (`logger_tls_period`, or the equivalent for your logger plugin) after the last scheduled run. Then confirm the host's most recent results have reached your log destination.

### Running osquery alongside fleetd

Fleet doesn't recommend running an osquery service and fleetd at the same time if both are configured to send data to the same Fleet instance.

Running both may be necessary if:

- Osquery is being used to collect data for an application other than Fleet.
- Logging data during the migration must be captured for security or compliance purposes.

In these cases, we recommend the following:

- Back up your existing osquery flags and configuration.
- Create a new osquery configuration that includes only critical functionality and doesn't communicate with Fleet:
  - Write osquery logs directly to your logging solution or to a local file.
  - Schedule only critical queries.
  - Load only critical extensions.
- Give the standalone osquery instance its own `database_path`, `pidfile`, and `extensions_socket` so it doesn't collide with fleetd's osquery. On Windows, make sure the two services have distinct names.
- On Linux, only one process can own the kernel audit netlink socket. Audit-based event tables (for example, `process_events` using audit) won't work in both instances at once, and they also conflict with `auditd`. Decide which agent owns audit during the migration. For example, the Workstations fleet sets `disable_audit: false` and `audit_persist: true`, so fleetd's osquery claims audit on Linux hosts. A standalone osquery running alongside it must keep `--disable_audit=true`.
- Run osquery with this configuration until the migration is complete, then remove it.


<meta name="articleTitle" value="Migrate from osquery to fleetd">
<meta name="authorFullName" value="Ryn Satterlee">
<meta name="authorGitHubUsername" value="rynsatterlee">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-30">
<meta name="description" value="Learn how to move hosts enrolled with plain osquery to Fleet's agent (fleetd) without losing host data or configuration.">
