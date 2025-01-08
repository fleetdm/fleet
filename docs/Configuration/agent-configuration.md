# Agent configuration

Learn how to update the settings of the agent installed on all your hosts at once. In Fleet, we refer to these settings as **agent options**.

## Overview

The `agent_options` key in the `config` YAML file controls the settings applied to the agent on all your hosts. These settings are applied when each host checks in and may be configured using the fleetctl command line tool or Fleet UI.

See the [osquery documentation](https://osquery.readthedocs.io/en/stable/installation/cli-flags/#configuration-control-flags) for the available options. This document shows all examples in command line flag format. Remove the dashed lines (`--`) for Fleet to successfully update the setting. For example, use `distributed_interval` instead of `--distributed_interval`.

Agent options are validated using the latest version of osquery.

When updating agent options, you may see an error similar to this:

```sh
[...] unsupported key provided: "logger_plugin"
If you’re not using the latest osquery, use the fleetctl apply --force command to override validation.
```

This error indicates that you're providing a config option that isn't valid in the current version of osquery, typically because you're setting a command line flag through the configuration key. This has always been unsupported through the config plugin, but osquery has recently become more opinionated and Fleet now validates the configuration to make sure there aren't errors in the osquery agent.

If you are not using the latest version of osquery, you can create a config YAML file and apply it with `fleetctl` using the `--force` flag to override the validation:

```sh
fleetctl apply --force -f config.yaml
```

You can verify that your agent options are valid by using [the `fleetctl apply` command](https://fleetdm.com/docs/using-fleet/fleetctl-cli) with the `--dry-run` flag. This will report any error and do nothing if the configuration was valid. If you don't use the latest version of osquery, you can override validation using the `--force` flag. This will update agent options even if they are invalid.

Existing options will be overwritten by the application of this file.

### Example Agent options YAML

```yaml
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        distributed_interval: 3
        distributed_tls_max_attempts: 3
        logger_tls_endpoint: /api/osquery/log
        logger_tls_period: 10
      decorators:
        load:
          - "SELECT version FROM osquery_info"
          - "SELECT uuid AS host_uuid FROM system_info"
        always:
          - "SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1"
        interval:
          3600: "SELECT total_seconds AS uptime FROM uptime"
    overrides:
      # Note configs in overrides take precedence over the default config defined
      # under the config key above. Be aware that these overrides are NOT merged
      # with the top-level configuration!! This means that settings values defined
      # on the top-level config.options section will not be propagated to the platform
      # override sections. So for example, the config.options.distributed_interval value
      # will be discarded on a platform override section, and only the section value
      # for distributed_interval will be used. If the given setting is not specified
      # in the override section, its default value will be enforced.
      # Going back to the example, if the override section is windows,
      # overrides.platforms.windows.distributed_interval will have to be set again to 5
      # for this setting to be enforced as expected, otherwise the setting will get
      # its default value (60 in the case of distributed_interval).
      # Hosts receive overrides based on the platform returned by `SELECT platform FROM os_version`.
      # In this example, the base config would be used for Windows and CentOS hosts,
      # while Mac and Ubuntu hosts would receive their respective overrides.
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

        ubuntu:
          options:
            distributed_interval: 10
            distributed_tls_max_attempts: 3
            logger_tls_endpoint: /api/osquery/log
            logger_tls_period: 60
            schedule_timeout: 60
            docker_socket: /etc/run/docker.sock
          file_paths:
            homes:
              - /root/.ssh/%%
              - /home/%/.ssh/%%
            etc:
              - /etc/%%
            tmp:
              - /tmp/%%
          exclude_paths:
            homes:
              - /home/not_to_monitor/.ssh/%%
            tmp:
              - /tmp/too_many_events/
          decorators:
            load:
              - "SELECT * FROM cpuid"
              - "SELECT * FROM docker_info"
            interval:
              3600: "SELECT total_seconds AS uptime FROM uptime"
  host_expiry_settings:
    # ...
```

## Command line flags

> This feature requires [Fleetd, the Fleet agent manager](https://fleetdm.com/announcements/introducing-orbit-your-fleet-agent-manager).

The `command_line_flags` key inside of `agent_options` allows you to remotely manage the osquery command line flags. These command line flags are options that typically require osquery to restart for them to take effect. But with Fleetd, you can use the `command_line_flags` key to take care of that. Fleetd will write these to the flagfile on the host and pass it to osquery.

To see the full list of these osquery command line flags, please run `osquery` with the `--help` switch.

> YAML `command_line_flags` are not additive and will replace any osquery command line flags in the CLI.

Just like the other `agent_options` above, remove the dashed lines (`--`) for Fleet to successfully update them.

Here is an example of using the `command_line_flags` key:

```yaml
agent_options:
  command_line_flags: # requires Fleet's agent (fleetd)
    verbose: true
    disable_watchdog: false
    disable_tables: chrome_extensions
    logger_path: /path/to/logger
```

Note that the `command_line_flags` key does not support the `overrides` key, which is documented below.

You can verify that these flags have taken effect on the hosts by running a query against the `osquery_flags` table.

> If you revoked an old enroll secret, this feature won't update for hosts that were added to Fleet using this old enroll secret. This is because Fleetd uses the enroll secret to receive new flags from Fleet. For these hosts, all existing features will work as expected.

For further documentation on how to rotate enroll secrets, please see [this guide](https://fleetdm.com/docs/configuration/configuration-files#rotating-enroll-secrets).

If you prefer to deploy a new package with the updated enroll secret:

1. Check which hosts need a new enroll secret by running the following query: `SELECT * FROM orbit_info WHERE enrolled = false`.

> The hosts that don't have Fleetd installed will return an error because the `orbit_info` table doesn't exist. You can safely ignore these errors.

2. In Fleet, head to the Hosts page and select **Add hosts** to find the fleetctl package command with an active enroll secret.

3. Copy and run the fleetctl package command to create a new package. Distribute this package to the hosts that returned results in step 1.

4. Done!



> In order for these options to be applied to your hosts, the `osquery` agent must be configured to use the `tls` config plugin and pointed to the correct endpoint. If you are using Fleetd to enroll your hosts, this is done automatically.

```go
"--config_plugin=tls",
"--config_tls_endpoint=" + path.Join(prefix, "/api/v1/osquery/config")
```

```yaml
apiVersion: v1
kind: config
spec:
  agent_options:
```

## Extensions

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

### Configure fleetd update channels

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

## Config

The config key sets the osqueryd configuration options for your agents. In a plain osquery deployment, these would typically be set in `osquery.conf`. Each key below represents a corresponding key in the osquery documentation.

For detailed information on osquery configuration options, check out the [osquery configuration docs](https://osquery.readthedocs.io/en/stable/deployment/configuration/).

```yaml
agent_options:
    config:
      options: ~
      decorators: ~
      yara: ~

```

### Options

In the `options` key, you can set your osqueryd options and feature flags.

Any command line only flags must be set using the `command_line_flags` key for Orbit agents, or by modifying the osquery flags on your hosts if you're using plain osquery.

To see a full list of flags, broken down by the method you can use to set them (configuration options vs command line flags), you can run `osqueryd --help` on a plain osquery agent. For Orbit agents, run `sudo orbit osqueryd --help`. The options will be shown there in command line format as `--key value`. In `yaml` format, that would become `key: value`.

```yaml
agent_options:
    config:
      options:
        distributed_interval: 3
        distributed_tls_max_attempts: 3
        logger_tls_endpoint: /api/osquery/log
        logger_tls_period: 10
```
### Decorators

In the `decorators` key, you can specify queries to include additional information in your osquery results logs.

Use `load` for details you want to update values when the configuration loads, `always` to update every time a scheduled query is run, and `interval` if you want to update on a schedule.

```yaml
agent_options:
    config:
      options: ~
      decorators:
        load:
          - "SELECT version FROM osquery_info"
          - "SELECT uuid AS host_uuid FROM system_info"
        always:
          - "SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1"
        interval:
          3600: "SELECT total_seconds AS uptime FROM uptime"
```

### Yara

You can use Fleet to configure the `yara` and `yara_events` osquery tables. Fore more information on YARA configuration and continuous monitoring using the `yara_events` table, check out the [YARA-based scanning with osquery section](https://osquery.readthedocs.io/en/stable/deployment/yara/) of the osquery documentation.

The following is an example Fleet configuration file with YARA configuration. The values are taken from an example config supplied in the above link to the osquery documentation.

```yaml
agent_options:
  config:
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

## Overrides

The `overrides` key allows you to segment hosts, by their platform, and supply these groups with unique osquery configuration options. When you choose to use the overrides option for a specific platform, all options specified in the default configuration will be ignored for that platform.

In the example file below, all Darwin and Ubuntu hosts will **only** receive the options specified in their respective overrides sections.

> IMPORTANT: If a given option is not specified in a platform override section, its default value will be enforced.

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

## Script execution timeout

The `script_execution_timeout` allows you to change the default script execution timeout.

- Optional setting (integer)
- Default value: 300
- Maximum value: 3600
- Config file format:
  ```yaml
  agent_options:
    script_execution_timeout: 600
  ```

## Auto table construction

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

## Update agent options in Fleet UI

<!-- Heading is kept so that the link from the Fleet UI still works -->
<span id="configuring-agent-options" name="configuring-agent-options"></span>

Fleet allows you to update the settings of the agent installed on all your hosts at once. In Fleet, these settings are called "agent options."

The default agent options are good to start. 

How to update agent options:

1. In the top navigation, select your avatar and select **Settings**. Only users with the [admin role](https://fleetdm.com/docs/using-fleet/permissions) can access the pages in **Settings**.

2. On the Organization settings page, select **Agent options** on the left side of the page.

3. Use Fleet's YAML editor to configure your osquery options, decorators, or set command line flags.

4. Place your new setting one level below the `options` key. The new setting's key should be below and one tab to the right of `options`.

5. Select **Save**.

The agents may take several seconds to update because Fleet has to wait for the hosts to check in. Additionally, hosts enrolled with removed enroll secrets must properly rotate their secret to have the new changes take effect.

> When configuring a value for [`script_execution_timeout`](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout) in the UI, make sure to put the key at the top level of the YAML, _not_ as a child of `config`.  

<meta name="pageOrderInSection" value="300">
<meta name="description" value="Learn how to use configuration files and the fleetctl command line tool to configure agent options.">
