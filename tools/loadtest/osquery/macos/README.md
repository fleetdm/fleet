# Load test of osquery queries in macOS

Following are the steps to load test osquery on macOS.
The purpose is to know the impact of Fleet provided queries on real devices.

> At the time of writing, the changes that add watchdog logging needed for this script are
> merged but not released yet (https://github.com/osquery/osquery/pull/8070).
> You will have to download and extract the osqueryd executable from the PR: https://github.com/osquery/osquery/suites/14033523376/artifacts/783724086

## Requirements

- Install gnuplot and ripgrep:
```sh
brew install gnuplot ripgrep
```
- Tooling to build osqueryd from source (at the time of writing this is needed), see https://osquery.readthedocs.io/en/stable/development/building/.

## Build fleetd_tables

We are going to use the fleetd tables as an extension so that it is also monitored by the watchdog.

```sh
make fleetd-tables-darwin-universal
sudo cp fleetd_tables_darwin_universal.ext /usr/local/osquery_extensions/fleetd_tables.ext
echo "/usr/local/osquery_extensions/fleetd_tables.ext" > /tmp/extensions.load
```

## Run osquery

> The following assumes a Fleet server instance running and listening at `localhost:8080`.

```sh
mkdir -p /Users/luk/osqueryd/osquery_log
```

```sh
sudo ENROLL_SECRET=<...> ./osquery/osqueryd \
    --verbose=true \
    --tls_dump=true \
    --pidfile=/Users/luk/osqueryd/osquery.pid \
    --database_path=/Users/luk/osqueryd/osquery.db \
    --logger_path=/Users/luk/osqueryd/osquery_log \
    --host_identifier=instance \
    # /Users/luk/fleetdm/git/fleet is the location of the Fleet mono repository.
    --tls_server_certs=/Users/luk/fleetdm/git/fleet/tools/osquery/fleet.crt \
    --enroll_secret_env=ENROLL_SECRET \
    --tls_hostname=localhost:8080 \
    --enroll_tls_endpoint=/api/v1/osquery/enroll \
    --config_plugin=tls \
    --config_tls_endpoint=/api/v1/osquery/config \
    --config_refresh=60 \
    --disable_distributed=false \
    --distributed_plugin=tls \
    --distributed_tls_max_attempts=10 \
    --distributed_tls_read_endpoint=/api/v1/osquery/distributed/read \
    --distributed_tls_write_endpoint=/api/v1/osquery/distributed/write \
    --logger_plugin=tls,filesystem \
    --logger_tls_endpoint=/api/v1/osquery/log \
    --disable_carver=false \
    --carver_disable_function=false \
    --carver_start_endpoint=/api/v1/osquery/carve/begin \
    --carver_continue_endpoint=/api/v1/osquery/carve/block \
    --carver_block_size=2000000 \
    --extensions_autoload=/tmp/extensions.load \
    --allow_unsafe \
    --enable_watchdog_debug \
    --distributed_denylist_duration 0 \
    --enable_extensions_watchdog 2>&1 | tee /tmp/osqueryd.log
```

## Check that the watchdog didn't trigger a worker kill

The following commands should return no output:
```sh
rg "utilization limit" /tmp/osqueryd.log
rg "Memory limit" /tmp/osqueryd.log
```

## Render CPU and memory usage

(Nice to have.)

```sh
./tools/loadtest/osquery/macos/gnuplot_osqueryd_cpu_memory.sh
```

> The horizontal red line is the configured CPU usage limit (hardcoded to `1200ms` in the `gnuplot_osqueryd_cpu_memory.sh`)
