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

would start 3 Ubuntu hosts and 3 Windows hosts. See the `os_templates` flag description in `go run agent.go --help` for the list of supported template names.

## Controlling Agent Behavior From the Fleet UI

### Specify Query Results

Using the naming convention `MyQuery_10` (name separated by `_number`) will instruct agents to
return 10 rows for that query

### Control policy pass/fail per policy

In the Policy SQL:

- `select 1` will instruct agents to send back only passing responses
- `select 0` will instruct agents to send back only failing responses

## Running Locally (Development Environment)

First, ensure your Fleet local development environment is up and running. Refer to [Building Fleet](../../docs/Contributing/Building-Fleet.md) for details. Once this is done:

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

For your server, disable Apple push notifications since we will be using devices with fake UUIDs:

```
export FLEET_DEV_MDM_APPLE_DISABLE_PUSH=1
```

Example of running the agent with MDM. Note that `enroll_secret` is not needed for iPhone/iPad devices:

```
go run agent.go --os_templates ipad_13.18,iphone_14.6 --host_count 10 --mdm_scep_challenge 0d53306e-6d7a-9d14-a372-f9e53f9d62db
```

## Installing software

The agent can install software for "macos", "ubuntu", and "windows" OSs when running with orbit agent. The following options control the installation behavior:

- `--software_installer_pre_install_fail_prob`: default 0.05, `select 1` always passes and `select 0` always fails
- `--software_installer_install_fail_prob`: default 0.05, `exit 0` always passes and `exit 1` always fails
- `--software_installer_post_install_fail_prob`: default 0.05, `exit 0` always passes and `exit 1` always fails
