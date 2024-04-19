# GitOps

Fleet can be managed using configuration files (YAML) with GitOps workflow. To learn how to setup GitOps workflow see [Fleet GitOps repo](https://github.com/fleetdm/fleet-gitops).

> Old workflow with YAML configuration files is documented [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Configuration-files.md).  `fleetctl apply` can be still used for imports and backwards compatibility.

This page lists the options available in configuration files.

## Agent options

The `agent_options` key controls the settings applied to the agent on all your hosts. These settings are applied when each host checks in.

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" to find all possible options.

## Queries

The `lib/{name}.queries.yml` file controls saved queries in Fleet.

- Optional setting
- Array of dictionaries
- Config format:  
  ```yaml
  - name: Collect failed login attempts
    description: Lists the users at least one failed login attempt and timestamp of failed login. Number of failed login attempts reset to zero after a user successfully logs in.
    query: SELECT users.username, account_policy_data.failed_login_count, account_policy_data.failed_login_timestamp FROM users INNER JOIN account_policy_data using (uid) WHERE account_policy_data.failed_login_count > 0;
    interval: 300 # 5 minutes
    observer_can_run: false
    automations_enabled: false
    platform: darwin,linux,windows
  - name: Collect USB devices
    description: Collects the USB devices that are currently connected to macOS and Linux hosts.
    query: SELECT model, vendor FROM usb_devices;
    interval: 300
    observer_can_run: true
    automations_enabled: false
  ``` 
