# Agent configuration

Agent configuration options (agent options) update the settings of [Fleet's agent (fleed)](https://fleetdm.com/docs/get-started/anatomy#fleetd) installed on all your hosts.

You can modify agent options in **Settings > Organization settings > Agent options** or via Fleet's [API](https://fleetdm.com/docs/rest-api/rest-api#modify-configuration) or [YAML files](https://fleetdm.com/docs/configuration/yaml-files).

## config 

The `config` section allows you to update settings like performance and and how often the agent checks-in.

####Example

```yaml
config:
  options:
    distributed_interval: 3
    distributed_tls_max_attempts: 3
    logger_tls_endpoint: /api/osquery/log
    logger_tls_period: 10
  command_line_flags: # requires Fleet's agent (fleetd)
    verbose: true
    disable_watchdog: false
    disable_tables: chrome_extensions
    logger_path: /path/to/logger
  decorators:
    load:
      - "SELECT version FROM osquery_info"
      - "SELECT uuid AS host_uuid FROM system_info"
    always:
      - "SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1"
    interval:
      3600: "SELECT total_seconds AS uptime FROM uptime"
  yara:
    file_paths:
      system_binaries:
      - sig_group_1
      tmp:
      - sig_group_1
      - sig_group_2
    signatures:
      sig_group_1:
      - /Users/wxs/sigs/foo.sig
      - /Users/wxs/sigs/bar.sig
      sig_group_2:
      - /Users/wxs/sigs/baz.sig
```

- `decorators`
- `yara`
- `auto_table_contructions`

### options and command_line_flags

- `options` include the agent settings listed under `osqueryOptions` [here](https://github.com/fleetdm/fleet/blob/main/server/fleet/agent_options_generated.go). These can be updated without a fleetd restart.
- `command_line_flags` include the agent settings listed under osqueryCommandLineFlags [here](https://github.com/fleetdm/fleet/blob/main/server/fleet/agent_options_generated.go). These are only updated when fleetd restarts. 

To see a description for all available settings, first [enroll your host](https://fleetdm.com/guides/enroll-hosts) to Fleet. Then, open your **Terminal** app and run `sudo orbit shell` to open an interactive osquery shell. Then run the following osquery query:

```
osquery > SELECT name, value, description FROM osquery; 
```

You can also run this query to verify that the latest settings have been applied to your hosts.

> If you revoked an old enroll secret, the `command_line_flags` won't update for hosts that enrolled to Fleet using this old enroll secret. This is because fleetd uses the enroll secret to receive new flags from Fleet. For these hosts, all existing features will work as expected.

How to rotate enroll secrets:

1. Check which hosts need a new enroll secret by running the following query: `SELECT * FROM orbit_info WHERE enrolled = false`.

> The hosts that don't have Fleetd installed will return an error because the `orbit_info` table doesn't exist. You can safely ignore these errors.

2. In Fleet, head to the Hosts page and select **Add hosts** to find the fleetctl package command with an active enroll secret.

3. Copy and run the fleetctl package command to create a new package. Distribute this package to the hosts that returned results in step 1.

4. Done!

#### Advanced

`options` and `command_line_flags` are validated using the latest version of osquery. If you are not using the latest version of osquery, you can create a YAML file and apply it with `fleetctl apply --force` command to override the validation:

```sh
fleetctl apply --force -f config.yaml
```

### decorators

In the `decorators` key, you can specify queries to include additional information in your osquery results logs.

- `load` is are queries you want to update values when the configuration loads.
- `always` are queries to update every time a scheduled query is run.
- `interval` are queries you want to update on a schedule.

### yara

You can use Fleet to configure the `yara` and `yara_events` osquery tables. Fore more information on YARA configuration and continuous monitoring using the `yara_events` table, check out the [YARA-based scanning with osquery section](https://osquery.readthedocs.io/en/stable/deployment/yara/) of the osquery documentation.

### auto_table_construction

You can use Fleet to query local SQLite databases as tables. For more information on creating ATC configuration from a SQLite database, check out the [Automatic Table Construction section](https://osquery.readthedocs.io/en/stable/deployment/configuration/#automatic-table-construction) of the osquery documentation.

If you already know what your ATC configuration needs to look like, you can add it to an options config file:

```yaml
agent_options:
  config:
    options:
      # ...
  overrides:
    platforms:
      darwin:
        auto_table_construction:
          tcc_system_entries:
            # This query and columns are restricted for compatability.  Open TCC.db with sqlite on
            # your endpoints to expand this out.
            query: "SELECT service, client, last_modified FROM access"
            # Note that TCC.db requires fleetd to have full-disk access, ensure that endpoints have 
            # this enabled.
            path: "/Library/Application Support/com.apple.TCC/TCC.db"
            columns:
              - "service"
              - "client"
              - "last_modified"
```

If you're editing this directly from the UI consider copying and pasting the following at the end of your agent configuration block:

```
overrides:
  platforms:
    darwin:
      auto_table_construction:
        tcc_system_entries:
          # This query and columns are restricted for compatability.  Open TCC.db with sqlite on
          # your endpoints to expand this out.
          query: "SELECT service, client, last_modified FROM access"
          # Note that TCC.db requires Orbit to have full-disk access, ensure that endpoints have
          # this enabled.
          path: "/Library/Application Support/com.apple.TCC/TCC.db"
          columns:
            - "service"
            - "client"
            - "last_modified"
```

## extensions

> This feature requires [Fleetd, the Fleet agent manager](https://fleetdm.com/announcements/introducing-orbit-your-fleet-agent-manager), along with a custom TUF auto-update server (a Fleet Premium feature).

The `extensions` key inside of `agent_options` allows you to remotely manage and deploy osquery extensions. Just like other `agent_options` the `extensions` key can be applied either to a team specific one or the global one.

This is best illustrated with an example. Here is an example of using the `extensions` key:
```yaml
agent_options:
  extensions: # requires Fleet's agent (fleetd)
    hello_world_macos:
      channel: 'stable'
      platform: 'macos'
    hello_world_linux:
      channel: 'stable'
      platform: 'linux'
    hello_world_windows:
      channel: 'stable'
      platform: 'windows'
```

In the above example, we are configuring our `hello_world` extensions for all the supported operating systems. We do this by creating `hello_world_{macos|linux|windows}` subkeys under `extensions`, and then specifying the `channel` and `platform` keys for each extension entry.

Next, you will need to make sure to push the binary files of our `hello_world_*` extension as a target on your TUF server. This step needs to follow these conventions:
* The binary file of the extension must have the same name as the extension, followed by `.ext` for macOS and Linux extensions and by `.ext.exe` for Windows extensions.
In the above case, the filename for macOS should be `hello_world_macos.ext`, for Linux it should be `hello_world_linux.ext` and for Windows it should be `hello_world_windows.ext.exe`.
* The target name for the TUF server must be named as `extensions/<extension_name>`. For the above example, this would be `extensions/hello_world_{macos|linux|windows}`
* The `platform` field is one of `macos`, `linux`, or `windows`.

If you are using `fleetctl` to manage your TUF server, these same conventions apply. You can run the following command to add a new target:
```bash
fleetctl updates add \
  --path /path/to/local/TUF/repo \
  --target /path/to/extensions/binary/hello_world_macos.ext \
  --name extensions/hello_world_macos \
  --platform macos \
  --version 0.1

fleetctl updates add \
  --path /path/to/local/TUF/repo
  --target /path/to/extensions/binary/hello_world_linux.ext \
  --name extensions/hello_world_linux \
  --platform linux \
  --version 0.1

fleetctl updates add \
  --path /path/to/local/TUF/repo \
  --target /path/to/extensions/binary/hello_world_windows.ext.exe \
  --name extensions/hello_world_windows \
  --platform windows \
  --version 0.1
```

After successfully configuring the agent options, and pushing the extension as a target on your TUF server, Fleetd will periodically check with the TUF server for updates to these extensions.

If you are using a self-hosted TUF server, you must also manage all of Fleetd's versions, including osquery, Fleet Desktop and osquery extensions.

Fleet recommends deploying extensions created with osquery-go or natively with C++, instead of Python. Extensions written in Python require the user to compile it into a single packaged binary along with all the dependencies.

### Targeting extensions with labels

_Available in Fleet Premium v4.38.0_

Fleet allows you to target extensions to hosts that belong to specific labels. To set these labels, you'll need to define a `labels` list under the extension name.
The label names in the list:
- must already exist (otherwise the `/api/latest/fleet/config` request will fail).
- are case insensitive.
- must **all** apply to a host in order to deploy the extension to that host.

Example:
```yaml
agent_options:
  extensions: # requires Fleet's agent (fleetd)
    hello_world_macos:
      channel: 'stable'
      platform: 'macos'
      labels:
        - Zoom installed
    hello_world_linux:
      channel: 'stable'
      platform: 'linux'
      labels:
        - Ubuntu Linux
        - Zoom installed
    hello_world_windows:
      channel: 'stable'
      platform: 'windows'
```
In the above example:
- the `hello_world_macos` extension is deployed to macOS hosts that are members of the 'Zoom installed' label.
- the `hello_world_linux` extension is deployed to Linux hosts that are members of the 'Ubuntu Linux' **and** 'Zoom installed' labels.

## update_channels

_Available in Fleet Premium v4.43.0 and fleetd v1.20.0_

Users can configure fleetd component TUF auto-update channels from Fleet's agent options. The components that can be configured are `orbit`, `osqueryd` and `desktop` (Fleet Desktop). When one of these components is omitted in `update_channels` then `stable` is assumed as the value for such component. Available options for update channels can be viewed [here](https://fleetdm.com/docs/using-fleet/enroll-hosts#specifying-update-channels).

Examples:
```yaml
agent_options:
  update_channels: # requires Fleet's agent (fleetd)
    orbit: stable
    osqueryd: '5.10.2'
    desktop: edge
```
```yaml
agent_options:
  update_channels: # requires Fleet's agent (fleetd)
    orbit: edge
    osqueryd: '5.10.2'
    # in this configuration `desktop` is assumed to be "stable"
```

- If a configured channel doesn't exist in the TUF repository, then fleetd will log errors on the hosts and will not auto-update the component/s until the channel is changed to a valid value in Fleet's `update_channels` configuration or until the user pushes the component to the channel (which effectively creates the channel).
- If the `update_channels` setting is removed from the agent settings, the devices will continue to use the last configured channels.
- If Fleet Desktop is disabled in fleetd, then the `desktop` channel setting is ignored by the host.

#### Auto update startup loop

Following we document an edge case scenario that could happen when using this feature.

After upgrading `orbit` on your devices to `1.20.0` using this feature, beware of downgrading `orbit` by changing it to a channel that's older than `1.20.0`. The auto-update system in orbit could end up in an update startup loop (where orbit starts, changes its channel and restarts over and over).

Following are the conditions (to avoid) that lead to the auto-update loop:
1. fleetd with `orbit` < `1.20.0` was packaged/configured to run with orbit channel `A`.
2. `orbit`'s channel `A` is updated to >= `1.20.0`.
3. `orbit`'s channel in the Fleet agent settings is configured to `B`, where channel `B` has orbit version < `1.20.0`.

This update startup loop can be fixed by any one of these actions:
A. Downgrading channel `A` to < `1.20.0`.
B. Upgrading channel `B` to >= `1.20.0`.

## overrides

The `overrides` key allows you to segment hosts, by their platform, and supply these groups with unique osquery configuration options. When you choose to use the overrides option for a specific platform, all options specified in the default configuration will be ignored for that platform.

In the example file below, all Darwin and Ubuntu hosts will **only** receive the options specified in their respective overrides sections.

If a given option is not specified in a platform override section, its default value will be enforced.

```yaml
agent_options:
  overrides:
    # Note configs in overrides take precedence over the default config defined
    # under the config key above. Hosts receive overrides based on the platform
    # returned by `SELECT platform FROM os_version`. In this example, the base
    # config would be used for Windows and CentOS hosts, while Mac and Ubuntu
    # hosts would receive their respective overrides. Note, these overrides are
    # NOT merged with the top level configuration.
    platforms:
      darwin:
        options:
          distributed_interval: 10
          distributed_tls_max_attempts: 10
          logger_tls_endpoint: /api/osquery/log
          logger_tls_period: 300
          docker_socket: /var/run/docker.sock
        file_paths:
          users:
            - /Users/%/Library/%%
            - /Users/%/Documents/%%
          etc:
            - /etc/%%
```

Note that the `command_line_flags` key is not supported in the `overrides`.

## script_execution_timeout

The `script_execution_timeout` allows you to change the default script execution timeout.

- Optional setting (integer)
- Default value: 300
- Maximum value: 3600
- Config file format:
  ```yaml
  agent_options:
    script_execution_timeout: 600
  ```

<meta name="pageOrderInSection" value="300">
<meta name="description" value="Learn how to use configuration files and the fleetctl command line tool to configure agent options.">
