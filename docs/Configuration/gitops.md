Fleet can be managed with configuration files (YAML) with GitOps workflow. To learn how to setup GitOps workflow see [Fleet GitOps repo](https://github.com/fleetdm/fleet-gitops).

> Old workflow with YAML configuration files is documented [here](https://github.com/fleetdm/fleet/blob/main/docs/Configuration/configuration-files/README.md).  `fleetctl apply` can be still used for imports and backwards compatibility.

On this page, you can learn how to write configuration files.

## Default configuration


The `default.yml` file defines the queries, policies, controls, and agent options for all hosts. If you're using Fleet Premium, this file updates queries and policies that run on all hosts ("All teams"). Controls and agent options are defined for hosts on "No team." 

Queries, policies, configuration profiles, scripts, and agent options can be referenced from `lib/` folder. Learn more about it in the [Library section](https://#library-lib).

```yaml
controls: # Controls added to "No team"
  macos_settings:
    custom_settings:
      - path: ./lib/macos-password.mobileconfig
  windows_enabled_and_configured: true
  windows_settings:
    custom_settings:
      - path: ./lib/windows-screenlock.xml
  scripts:
    - path: ./lib/collect-fleetd-logs.sh
queries:
  - path: ./lib/collect-fleetd-update-channels.queries.yml
policies:
agent_options:
  path: ./lib/agent-options.yml
org_settings:
  server_settings:
    debug_host_ids:
      - 1
      - 3
    enable_analytics: true
    live_query_disabled: false
    query_reports_disabled: false
    scripts_disabled: false
    server_url: https://dogfood.fleetdm.com
  org_info:
    contact_url: https://fleetdm.com/company/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: Fleet Device Management
  smtp_settings:
  sso_settings:
    enable_jit_provisioning: false
    enable_jit_role_sync: false
    enable_sso: true
    enable_sso_idp_login: false
    idp_name: Google Workspace
    entity_id: dogfood.fleetdm.com
    metadata: $FLEET_SSO_METADATA
  integrations:
  mdm:
    apple_bm_default_team: "Workstations"
  webhook_settings:
    vulnerabilities_webhook:
      enable_vulnerabilities_webhook: true
      destination_url: https://example.tines.com/webhook
  fleet_desktop: # Applies to Fleet Premium only
    transparency_url: https://fleetdm.com/transparency
  host_expiry_settings: # Applies to all teams
    host_expiry_enabled: false
  features: # Features added to all teams
  secrets: # These secrets are used to enroll hosts to the "All teams" team
    - secret: "$FLEET_GLOBAL_ENROLL_SECRET"
```


### Agent options


The `agent_options` key controls the settings applied to the agent on all your hosts. These settings are applied when each host checks in.

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" for more information.

### Features


...
----

## Team configuration


`team/{team_name}.yml` file updates controls, queries, policies, and agent options for hosts assigned to the specified team. Below the example file you can find each option explained.

Queries, policies, configuration profiles, scripts and agent options can be referenced from `lib/` folder. Learn more about it in the [Library section](https://#library-lib).

```yaml
name: Workstations
controls:
  enable_disk_encryption: true
  macos_updates:
    deadline: "2023-08-11"
    minimum_version: "13.5"
  windows_updates:
    deadline_days: 5
    grace_period_days: 2
  macos_settings:
    custom_settings:
      # - path: ../lib/macos-os-updates.ddm.json (DDM coming soon)
      - path: ../lib/macos-password.mobileconfig
  windows_settings:
    custom_settings:
    - path: ../lib/windows-screenlock.xml
  macos_setup:
      # bootstrap_package: https://github.com/organinzation/repository/bootstrap-package.pkg (example URL)
      # enable_end_user_authentication: true
      macos_setup_assistant: ../lib/automatic-enrollment.dep.json
  scripts:
    - path: ../lib/remove-zoom-artifacts.script.sh
    - path: ../lib/set-timezone.script.sh
queries:
  - path: ../lib/collect-usb-devices.queries.yml
  - path: ../lib/collect-failed-login-attempts.queries.yml
policies:
  - path: ../lib/macos-device-health.policies.yml
  - path: ../lib/windows-device-health.policies.yml
agent_options:
  path: ../lib/agent-options.yml
team_settings:
  secrets:
    - secret: "$FLEET_WORKSTATIONS_ENROLL_SECRET"
```


### Agent options


...

### Controls


...
----

## Library (`lib/`) 


Library is used to store files that define policies, queries, configuration profiles, scripts, and agent options. These files can be referenced in default configuration (`default.yml`) and team configurations inside `teams/` folder in GitOps repo.

### Policies


The `lib/{name}.policies.yml` files define set of policies that can be referenced in a default and team configurations.

```yaml
- name: Windows - Enable BitLocker
  platform: windows
  description: "This policy checks if BitLocker (disk encryption) is enabled on the C: volume."
  resolution: As an IT admin, turn on disk encryption in Fleet.
  query: SELECT * FROM bitlocker_info WHERE drive_letter='C:' AND protection_status = 1;
- name: Windows - Disable guest account
  platform: windows
  description: This policy checks if the guest account is disabled. The Guest account allows unauthenticated network users to gain access to the system.
  resolution: "As an IT admin, deploy a Windows profile with the Accounts_EnableGuestAccountStatus option documented here: https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-localpoliciessecurityoptions#accounts_enableguestaccountstatus"
  query: SELECT 1 FROM mdm_bridge where mdm_command_input = "<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/LocalPoliciesSecurityOptions/Accounts_EnableGuestAccountStatus</LocURI></Target></Item></Get></SyncBody>" and CAST(mdm_command_output AS INT) = 0;
- name: Windows - Require 10 character password
  platform: windows
  description: This policy checks if the end user is required to enter a password, with at least 10 characters, to unlock the host.
  resolution: "As an IT admin, deploy a Windows profile with the DevicePasswordEnabled and MinDevicePasswordLength option documented here: https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-devicelock"
  query: SELECT 1 FROM mdm_bridge where mdm_command_input = "<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/DeviceLock/DevicePasswordEnabled</LocURI></Target></Item></Get></SyncBody>" and CAST(mdm_command_output AS INT) = 0;
- name: Windows - Enable screen saver after 20 minutes
  platform: windows
  description: This policy checks if maximum amount of time (in minutes) the device is allowed to sit idle before the screen is locked. End users can select any value less than the specified maximum.
  resolution: "As an IT admin, to deploy a Windows profile with the MaxInactivityTimeDeviceLock option documented here: https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-devicelock#maxinactivitytimedevicelock"
  query: SELECT 1 FROM mdm_bridge where mdm_command_input = "<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/DeviceLock/MaxInactivityTimeDeviceLock</LocURI></Target></Item></Get></SyncBody>" and CAST(mdm_command_output AS INT) <= 20;
```


### Queries


The `lib/{name}.queries.yml` files define set of policies that can be referenced in a default and team configurations.

```yaml
name: Collect USB devices
  description: Collects the USB devices that are currently connected to macOS and Linux hosts.
  query: SELECT model, vendor FROM usb_devices;
  interval: 300 # 5 minutes
  observer_can_run: true
  automations_enabled: false
```


### Agent options


The `lib/agent-options.yml` define agent options.

```yaml
command_line_flags:
config:
  decorators:
    load:
      - SELECT uuid AS host_uuid FROM system_info;
      - SELECT hostname AS hostname FROM system_info;
  options:
    disable_distributed: false
    distributed_interval: 10
    distributed_plugin: tls
    distributed_tls_max_attempts: 3
    logger_tls_endpoint: /api/v1/osquery/log
    pack_delimiter: /
```


### Configuration profiles


The `lib/`folder can be used to add configuration profiles that can be referenced in a default and team configurations. You can add macOS profiles (.json), declaration (DDM) profiles (.json) and Windows profiles (.xml)

### Scripts


The `lib/`folder can be used to add scripts that can be referenced in a default and team configurations. You can add shell scripts (.sh) for macOS and Linux and PowerShell scripts (.ps1) for Windows hosts.
