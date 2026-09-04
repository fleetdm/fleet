# Osquery Server Performance Tester

This is a tool to generate realistic traffic to an osquery
management server (primarily, [Fleet](https://github.com/fleetdm/fleet)). With
this tool, many thousands of hosts can be simulated from a single host.

## Requirements

The only requirement for running this tool is a working installation of
[Go](https://golang.org/doc/install).

## Usage

Typically `go run` is used.

You can use `--help` to view the available configuration:

```
go run agent.go --help
```

The tool should be invoked with the appropriate enroll secret. A typical
invocation looks like:

```
go run agent.go --enroll_secret hgh4hk3434l2jjf
```

When starting many hosts, it is a good idea to extend the intervals, and also
the period over which the hosts are started:

```
go run agent.go --enroll_secret hgh4hk3434l2jjf --host_count 5000 --start_period 5m --query_interval 60s --config_interval 5m
```

This will start 5,000 hosts over a period of 5 minutes. Each host will check in
for live queries at a 1 minute interval, and for configuration at a 5 minute
interval. Starting over a 5 minute period ensures that the configuration
requests are spread evenly over the 5 minute interval.

It can be useful to start the "same" hosts. This can be achieved with the
`--seed` parameter:

```
go run agent.go --enroll_secret hgh4hk3434l2jjf --seed 0
```

By using the same seed, along with other values, we usually get hosts that look
the same to the server. This is not guaranteed, but it is a useful technique.

By default, all hosts will simulate macOS hosts (specifically, macOS 10.14). To simulate hosts using other operating systems, use the `--os_templates` flag. This flag takes a comma-separated list of host template names and will start hosts by alternating in the list of OS templates when multiple templates are specified. For example:

```
go run agent.go --enroll_secret hgh4hk3434l2jjf --os_templates ubuntu_22.04,windows_11 --host_count 6
```

would start 3 Ubuntu hosts and 3 Windows hosts.

Supported Linux templates: `ubuntu_22.04`, `rhel_8`, `rhel_9`, `rhel_10`. RHEL templates report `platform=rhel`, RPM-style kernels (e.g., `kernel-core 5.14.0-503.26.1.el9_5`), and (when `--software_db_path` points at a populated database) additional RPM packages from the software library. See the `os_templates` flag description in `go run agent.go --help` for the full list of supported template names.

The software database (`cmd/osquery-perf/software-library/software.db`) is optional — macOS, Windows, and Ubuntu have embedded fallback fixtures, and RHEL kernels are embedded too. The DB only adds non-kernel software variety. If the DB isn't present at `--software_db_path`, osquery-perf logs a warning and falls back to the embedded fixtures.

## Controlling Agent Behavior From the Fleet UI

### Specify Query Results

Using the naming convention `MyQuery_10` (name separated by `_number`) will instruct agents to
return 10 rows for that query

### Control policy pass/fail per policy

In the Policy SQL:

- `select 1` will instruct agents to send back only passing responses
- `select 0` will instruct agents to send back only failing responses

## Running Locally (Development Environment)

First, ensure your Fleet local development environment is up and running. Refer to [Building Fleet](../../docs/Contributing/getting-started/building-fleet.md) for details. Once this is done:

* navigate to the Hosts tab of your Fleet web interface (typically, this would be at https://localhost:8080/hosts/manage).
* click on "Manage enroll secret" and copy the enroll secret.
* start the `osquery-perf` agent (from the root of the Fleet repository, it would be `go run ./cmd/osquery-perf/agent.go --enroll_secret <paste-the-secret>`).

Alternatively, you can retrieve the enroll secret from the command-line using `fleetctl get enroll_secret` (you may have to login to `fleetctl` first).

The agent will start. You can connect to MySQL to view changes made to the development database by the agent (e.g., at the terminal, with `docker-compose exec mysql mysql -uroot -ptoor -Dfleet`). Remember that frequency of the reported data depends on the configuration of the Fleet instance, so you may want to start it with shorter delays for some cases and enable debug logging (e.g., `./build/fleet serve --dev --logging_debug --osquery_detail_update_interval 1m`).

## Resource Limits

On many systems, trying to simulate a large number of hosts will result in hitting system resource limits (such as number of open file descriptors).

If you see errors such as `dial tcp: lookup localhost: no such host` or `read: connection reset by peer`, try increasing these limits.

### macOS

Run the following command in the shell before running the Fleet server _and_ before running `agent.go` (run it once in each shell):

``` sh
ulimit -n 64000
```

## Running with MDM

Set up MDM on your server. To extract the SCEP challenge, you can use the [MDM asset extractor](https://github.com/fleetdm/fleet/tree/main/tools/mdm/assets).

For your server, configure a custom Apple push notifications URL since we will be using devices with fake UUIDs:

```
export FLEET_DEV_MDM_APPLE_PUSH_SERVER_URL=http://localhost:8378
```

Example of running the agent with MDM. Note that `enroll_secret` is not needed for iPhone/iPad devices:

```
go run agent.go --os_templates ipad_13.18,iphone_14.6 --host_count 10 --mdm_scep_challenge 0d53306e-6d7a-9d14-a372-f9e53f9d62db
```

`mdm_prob` determines the probability of MDM enrollment for each host. The default is 0 (0%). You can set it to 1.0 to ensure all hosts enroll in MDM.

`mdm_user_prob` determines the probability of MDM user enrollment for each host. The default is 0 (0%). You can set it to 1.0 to ensure all hosts enroll in MDM user enrollment. This probability stacks with `mdm_prob`. So this probability is based on the hosts who end up MDM enrolling.

`mdm_ios_byod_prob` determines the probability that a simulated iOS/iPadOS device (`iphone_14.6`, `ipad_13.18`, `iphone_17` templates) reports as a personal (BYOD) enrollment, which omits the newer device vitals fields from its `DeviceInformation` ack, matching what Fleet's server asks a real BYOD device for. The default is 0 (all simulated iOS/iPadOS devices report the full vitals set).

`mdm_apns_url` sets the mock APNs server URL for the simulated Apple MDM devices. It is required when using iPhone/iPad templates and required when using macOS templates with a non-zero `mdm_prob`.

`mdm_cancelable_command_ack_delay` makes simulated Apple devices log when they fetch a cancelable MDM command (lock, wipe, clear passcode, enable lost mode) and wait the given duration before acknowledging it. This opens a window to cancel an already-delivered command (`DELETE /api/v1/fleet/hosts/:id/commands/:command_uuid`) and observe the server restoring the host's lock/wipe state when the acknowledgment arrives anyway. The default is 0 (acknowledge immediately, no behavior change).

### Apple Platform SSO (PSSO)

A subset of macOS MDM agents can additionally exercise Apple Platform SSO: device
registration, password login (proxied through Fleet to your IdP), and the
offline-unlock key request/exchange. This requires a server that has account
provisioning configured (with a reachable ROPG IdP) and the PSSO configuration
profile assigned to the enrolled hosts — the agent obtains its Fleet-signed
registration token from the delivered profile, so nothing happens until that
profile reconciles onto the host.

Each selected agent registers once (staggered across `--mdm_psso_interval` to
avoid a thundering herd), does one login and one key request/exchange during
setup, and then, on each interval, performs a login and/or a key request/exchange
according to their probabilities — spread across the interval rather than on the
tick boundary.

- `--mdm_psso_prob`: default 0, probability an MDM-enrolled macOS host also simulates PSSO [0, 1]
- `--mdm_psso_client_id`: IdP/extension client ID, must match the server's account provisioning config (PSSO is skipped when empty)
- `--mdm_psso_username` / `--mdm_psso_password`: IdP credentials used for logins (must be accepted by the IdP Fleet proxies to)
- `--mdm_psso_interval`: default 4h, window for staggering registrations and recurring logins/key operations
- `--mdm_psso_login_prob`: default 1.0, probability of a login during each interval after registration [0, 1]
- `--mdm_psso_key_prob`: default 0.1, probability of a key request/exchange during each interval after registration [0, 1]

```
go run agent.go --host_count 100 --mdm_prob 1.0 --mdm_scep_challenge <challenge> \
  --mdm_psso_prob 0.5 --mdm_psso_client_id <client-id> \
  --mdm_psso_username loadtest@example.com --mdm_psso_password <password> \
  --mdm_psso_interval 4h --mdm_psso_login_prob 1.0 --mdm_psso_key_prob 0.1
```

### Synthetically reproducing MDM device protocol failures

#### NotNow'ing profiles

> Currently only supported for macOS and `InstallProfile` commands

To force an osquery-perf agent to respond with `NotNow` once to an `InstallProfile` command, the payload has to contain `NotNow` anywhere in the profile. It will NotNow once, then acknowledge it on next check-in. To force a new `NotNow` response, you have to change the `ProfileIdentifier`.

#### Forcing a certain error code and failure for InstallApplication

> Currently only supported for macOS.

To force a certain ErrorCode and failure for an `InstallApplication` command, the `iTunesStoreID` payload field has to have a value below 100_000. The agent will respond with a failure and the specified error code, which helps QA and repro logic scenarios on certain error codes.

## Conditional Config Request (ETag) Support

The agent can simulate the native osquery conditional config request
lifecycle. The validator travels in the JSON bodies: an enabled agent sends
an `"etag"` field in the config request body (empty on its first request),
stores the server-assigned `"etag"` value from each full config response, and
treats the constant `{"etag":"ok"}` response as "unchanged", retaining the
installed scheduled-query state.

- `--config_tls_etag`: default `false`, enable native osquery conditional config requests

This feature is **off by default** because osquery-perf is designed for load
testing, and enabling it reduces bandwidth without necessarily reducing
backend load. Opt in with `--config_tls_etag=true` to measure bandwidth
savings.

The etag is in-memory only — a restarted osquery-perf process starts with a
full fetch for every simulated host. The value is stored on receipt, before
the config is processed, mirroring the real osquery client: a config the
agent fails to process is confirmed unchanged on later check-ins rather than
re-downloaded. The server's value is authoritative and opaque; a local
SHA-256 of the canonical (etag-less) body is compared only as a diagnostic
(mismatches are counted but do not affect behavior).

Stats logged every 10 seconds include:

- `config full responses`: full config responses received
- `config not-modified responses`: `{"etag":"ok"}` responses received
- `conditional config requests`: requests that echoed a non-empty etag
- `config response body bytes`: total downloaded config body bytes
- `estimated config body bytes avoided`: body bytes saved by not-modified responses
- `estimated config body savings pct`: percentage of logical body bytes avoided
- `config etag drift`: times the server's etag disagreed with the locally calculated hash

### Example control/treatment run

Run a control (no etag) and treatment (etag enabled) against the same Fleet
server to measure bandwidth savings:

```
# Control: no etag, every request downloads the full config
go run agent.go --config_tls_etag=false --host_count 100 --config_interval 1m ...

# Treatment: etag enabled, unchanged configs get the minimal body
go run agent.go --config_tls_etag=true --host_count 100 --config_interval 1m ...
```

Compare the `config response body bytes` and `estimated config body savings pct`
from the stats logs. For a config-dominant profile, use:

```
--orbit_prob 0.0 --mdm_prob 0.0 --config_interval 1m --query_interval 24h --logger_tls_period 24h
```

## Installing software

The agent can install software for "macos", "ubuntu", and "windows" OSs when running with orbit agent. The following options control the installation behavior:

- `--software_installer_pre_install_fail_prob`: default 0.05, `select 1` always passes and `select 0` always fails
- `--software_installer_install_fail_prob`: default 0.05, `exit 0` always passes and `exit 1` always fails
- `--software_installer_post_install_fail_prob`: default 0.05, `exit 0` always passes and `exit 1` always fails
