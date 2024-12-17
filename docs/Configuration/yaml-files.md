# YAML files

Use Fleet's best practice GitOps workflow to manage your computers as code.

To learn how to set up a GitOps workflow see the [Fleet GitOps repo](https://github.com/fleetdm/fleet-gitops).

The following are the required keys in the `default.yml` and any `teams/team-name.yml` files:

```yaml
name: # Only teams/team-name.yml. To edit a team's name, change `name` but don't change the filename.
policies:
queries:
agent_options:
controls: # Can be defined in teams/no-team.yml too.
org_settings: # Only default.yml
team_settings: # Only teams/team-name.yml
```

Currently, managing labels and users is only supported using Fleet's UI or [API](https://fleetdm.com/docs/rest-api/rest-api) (YAML coming soon).

## policies

Policies can be specified inline in your `default.yml`, `teams/team-name.yml`, or `teams/no-team.yml` files. They can also be specified in separate files in your `lib/` folder.

### Options

For possible options, see the parameters for the [Add policy API endpoint](https://fleetdm.com/docs/rest-api/rest-api#add-policy).

### Example

#### Inline
  
`default.yml`, `teams/team-name.yml`, or `teams/no-team.yml`

```yaml
policies:
  - name: macOS - Enable FileVault
    description: This policy checks if FileVault (disk encryption) is enabled.
    resolution: As an IT admin, turn on disk encryption in Fleet.
    query: SELECT 1 FROM filevault_status WHERE status = 'FileVault is On.';
    platform: darwin
    critical: false
```

#### Separate file
 
`lib/policies-name.policies.yml`

```yaml
- name: macOS - Enable FileVault
  description: This policy checks if FileVault (disk encryption) is enabled.
  resolution: As an IT admin, turn on disk encryption in Fleet.
  query: SELECT 1 FROM filevault_status WHERE status = 'FileVault is On.';
  platform: darwin
  critical: false
  calendar_event_enabled: false
- name: macOS - Disable guest account
  description: This policy checks if the guest account is disabled.
  resolution: An an IT admin, deploy a macOS, login window profile with the DisableGuestAccount option set to true.
  query: SELECT 1 FROM managed_policies WHERE domain='com.apple.loginwindow' AND username = '' AND name='DisableGuestAccount' AND CAST(value AS INT) = 1;
  platform: darwin
  critical: false
  calendar_event_enabled: false
  run_script:
    path: "../lib/disable-guest-account.sh"
- name: Install Firefox on macOS
  platform: darwin
  description: "This policy checks that Firefox is installed."
  resolution: "Install Firefox app if not installed."
  query: "SELECT 1 FROM apps WHERE name = 'Firefox.app'"
  install_software:
    package_path: "../lib/firefox.package.yml"
- name: [Install software] Logic Pro
  platform: darwin
  description: "This policy checks that Logic Pro is installed"
  resolution: "Install Logic Pro App Store app if not installed"
  query: "SELECT 1 FROM apps WHERE name = 'Logic Pro'"
  install_software:
    app_store_app_id: "1487937127"
```

`default.yml` (for policies that neither install software nor run scripts), `teams/team-name.yml`, or `teams/no-team.yml`

```yaml
policies:
  - path: ../lib/policies-name.policies.yml
# path is relative to default.yml, teams/team-name.yml, or teams/no-team.yml
```

> Currently, the `run_script` and `install_software` policy automations can only be configured for a team (`teams/team-name.yml`) or "No team" (`teams/no-team.yml`). The automations can only be added to policies in which the script (or software) is defined in the same team (or "No team"). `calendar_event_enabled` can only be configured for policies on a team.

## queries

Queries can be specified inline in your `default.yml` file or `teams/team-name.yml` files. They can also be specified in separate files in your `lib/` folder.

Note that the `team_id` option isn't supported in GitOps.

### Options

For possible options, see the parameters for the [Create query API endpoint](https://fleetdm.com/docs/rest-api/rest-api#create-query).

### Example

#### Inline
  
`default.yml` or `teams/team-name.yml`

```yaml
queries:
  - name: Collect failed login attempts
    description: Lists the users at least one failed login attempt and timestamp of failed login. Number of failed login attempts reset to zero after a user successfully logs in.
    query: SELECT users.username, account_policy_data.failed_login_count, account_policy_data.failed_login_timestamp FROM users INNER JOIN account_policy_data using (uid) WHERE account_policy_data.failed_login_count > 0;
    platform: darwin,linux,windows
    interval: 300
    observer_can_run: false
    automations_enabled: false
```

#### Separate file
 
`lib/queries-name.queries.yml`

```yaml
- name: Collect failed login attempts
  description: Lists the users at least one failed login attempt and timestamp of failed login. Number of failed login attempts reset to zero after a user successfully logs in.
  query: SELECT users.username, account_policy_data.failed_login_count, account_policy_data.failed_login_timestamp FROM users INNER JOIN account_policy_data using (uid) WHERE account_policy_data.failed_login_count > 0;
  platform: darwin,linux,windows
  interval: 300
  observer_can_run: false
  automations_enabled: false
- name: Collect USB devices
  description: Collects the USB devices that are currently connected to macOS and Linux hosts.
  query: SELECT model, vendor FROM usb_devices;
  platform: darwin,linux
  interval: 300
  observer_can_run: true
  automations_enabled: false
```

`default.yml` or `teams/team-name.yml`

```yaml
queries:
  - path: ../lib/queries-name.queries.yml
# path is relative to default.yml, teams/team-name.yml, or teams/no-team.yml
```

## agent_options

Agent options can be specified inline in your `default.yml` file or `teams/team-name.yml` files. They can also be specified in separate files in your `lib/` folder.

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" to find all possible options.

### Example

#### Inline
  
`default.yml` or `teams/team-name.yml`

```yaml
agent_options:
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
      logger_tls_endpoint: /api/osquery/log
      logger_tls_period: 10
      pack_delimiter: /
```

#### Separate file
 
`lib/agent-options.yml`

```yaml
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
    logger_tls_endpoint: /api/osquery/log
    logger_tls_period: 10
    pack_delimiter: /
```

`default.yml` or `teams/team-name.yml`

> We want `-` for policies and queries because it’s an array. Agent Options we do not use `-` for `path`.

```yaml
queries:
  path: ../lib/agent-options.yml
# path is relative to default.yml, teams/team-name.yml, or teams/no-team.yml
```

## controls

The `controls` section allows you to configure scripts and device management (MDM) features in Fleet.

- `scripts` is a list of paths to macOS, Windows, or Linux scripts.
- `windows_enabled_and_configured` specifies whether or not to turn on Windows MDM features (default: `false`). Can only be configured for all teams (`default.yml`).
- `enable_disk_encryption` specifies whether or not to enforce disk encryption on macOS, Windows, and Linux hosts (default: `false`).

#### Example

```yaml
controls:
  scripts: 
    - path: ../lib/macos-script.sh 
    - path: ../lib/windows-script.ps1
    - path: ../lib/linux-script.sh
  windows_enabled_and_configured: true
  enable_disk_encryption: true # Available in Fleet Premium
  macos_updates: # Available in Fleet Premium
    deadline: "2024-12-31"
    minimum_version: 15.1
  ios_updates: # Available in Fleet Premium
    deadline: "2024-12-31"
    minimum_version: 18.1
  ipados_updates: # Available in Fleet Premium
    deadline: "2024-12-31"
    minimum_version: 18.1
  windows_updates: # Available in Fleet Premium
    deadline_days: 5
    grace_period_days: 2
  macos_settings:
    custom_settings:
      - path: ../lib/macos-profile1.mobileconfig
        labels_exclude_any:
          - Macs on Sequoia
      - path: ../lib/macos-profile2.json
        labels_include_all:
          - Macs on Sonoma
      - path: ../lib/macos-profile3.mobileconfig
        labels_include_any:
          - Engineering
          - Product
  windows_settings:
    custom_settings:
      - path: ../lib/windows-profile.xml
  macos_setup: # Available in Fleet Premium
    bootstrap_package: https://example.org/bootstrap_package.pkg
    enable_end_user_authentication: true
    macos_setup_assistant: ../lib/dep-profile.json
    script: ../lib/macos-setup-script.sh
    software:
      - app_store_id: '1091189122'
      - package_path: ../lib/software/adobe-acrobat.software.yml
  macos_migration: # Available in Fleet Premium
    enable: true
    mode: voluntary
    webhook_url: https://example.org/webhook_handler
# paths are relative to default.yml or teams/team-name.yml 
```

### macos_updates

- `deadline` specifies the deadline in `YYYY-MM-DD` format. The exact deadline is set to noon local time for hosts on macOS 14 and above, 20:00 UTC for hosts on older macOS versions. (default: `""`).
- `minimum_version` specifies the minimum required macOS version (default: `""`).

### ios_updates

- `deadline` specifies the deadline in `YYYY-MM-DD` format; the exact deadline is set to noon local time. (default: `""`).
- `minimum_version` specifies the minimum required iOS version (default: `""`).

### ipados_updates

- `deadline` specifies the deadline in `YYYY-MM-DD` format; the exact deadline is set to noon local time. (default: `""`).
- `minimum_version` specifies the minimum required iPadOS version (default: `""`).

### windows_updates

- `deadline_days` specifies the number of days before Windows installs updates (default: `null`)
- `grace_period_days` specifies the number of days before Windows restarts to install updates (default: `null`)

### macos_settings and windows_settings

- `macos_settings.custom_settings` is a list of paths to macOS configuration profiles (.mobileconfig) or declaration profiles (.json).
- `windows_settings.custom_settings` is a list of paths to Windows configuration profiles (.xml).

Fleet supports adding [GitHub environment variables](https://docs.github.com/en/actions/learn-github-actions/variables#defining-environment-variables-for-a-single-workflow) in your configuration profiles. Use `$ENV_VARIABLE` format. Variables beginning with `$FLEET_VAR_` are reserved for Fleet server. The server will replace these variables with the actual values when profiles are sent to hosts. See supported variables in the guide [here](https://fleetdm.com/guides/ndes-scep-proxy).

Use `labels_include_all` to only apply (scope) profiles to hosts that have all those labels, `labels_include_any` to apply profiles to hosts that have any of those labels, or `labels_exclude_any` to apply profiles to hosts that don't have any of those labels.

### macos_setup

The `macos_setup` section lets you control the out-of-the-box macOS [setup experience](https://fleetdm.com/guides/macos-setup-experience) for hosts that use Automated Device Enrollment (ADE).

- `bootstrap_package` is the URL to a bootstap package. Fleet will download the bootstrap package (default: `""`).
- `enable_end_user_authentication` specifies whether or not to require end user authentication when the user first sets up their macOS host. 
- `macos_setup_assistant` is a path to a custom automatic enrollment (ADE) profile (.json).
- `script` is the path to a custom setup script to run after the host is first set up.
- `software` is a list of references to either a `package_path` matching a package in the `software` section below or an `app_store_id` to install when the host is first set up.

### macos_migration

The `macos_migration` section lets you control the [end user migration workflow](https://fleetdm.com/docs/using-fleet/mdm-migration-guide#end-user-workflow) for macOS hosts that enrolled to your old MDM solution.

- `enable` specifies whether or not to enable end user migration workflow (default: `false`)
- `mode` specifies whether the end user initiates migration (`voluntary`) or they're nudged every 15-20 minutes to migrate (`forced`) (default: `""`).
- `webhook_url` is the URL that Fleet sends a webhook to when the end user selects **Start**. Receive this webhook using your automation tool (ex. Tines) to unenroll your end users from your old MDM solution.

Can only be configured for all teams (`default.yml`).

## software

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

The `software` section allows you to configure packages and Apple App Store apps that you want to install on your hosts.

Currently, managing [Fleet-maintained apps](https://fleetdm.com/guides/install-fleet-maintained-apps-on-macos-hosts) is only supported using Fleet's UI or [API](https://fleetdm.com/docs/rest-api/rest-api) (YAML coming soon).

- `packages` is a list of paths to custom packages (.pkg, .msi, .exe, .rpm, or .deb).
- `app_store_apps` is a list of Apple App Store apps.

Currently, one app for each of an App Store app's supported platforms are added. For example, adding [Bear](https://apps.apple.com/us/app/bear-markdown-notes/id1016366447) (supported on iOS and iPadOS) adds both the iOS and iPadOS apps to your software that's available to install in Fleet. Specifying specific platforms is only supported using Fleet's UI or [API](https://fleetdm.com/docs/rest-api/rest-api) (YAML coming soon).

#### Example

`default.yml`, `teams/team-name.yml`, or `teams/no-team.yml`

```yaml
software:
  packages:
    - path: ../lib/software-name.package.yml
  # path is relative to default.yml, teams/team-name.yml, or teams/no-team.yml
  app_store_apps:
    - app_store_id: '1091189122'
      labels_include_any:
        - Product
        - Marketing
```

### packages

- `url` specifies the URL at which the software is located. Fleet will download the software and upload it to S3 (default: `""`).
- `pre_install_query.path` is the osquery query Fleet runs before installing the software. Software will be installed only if the [query returns results](https://fleetdm.com/tables) (default: `""`).
- `install_script.path` specifies the command Fleet will run on hosts to install software. The [default script](https://github.com/fleetdm/fleet/tree/main/pkg/file/scripts) is dependent on the software type (i.e. .pkg).
- `uninstall_script.path` is the script Fleet will run on hosts to uninstall software. The [default script](https://github.com/fleetdm/fleet/tree/main/pkg/file/scripts) is dependent on the software type (i.e. .pkg).
- `self_service` specifies whether or not end users can install from **Fleet Desktop > Self-service**.

#### Example

`lib/software-name.package.yml`:

```yaml
url: https://dl.tailscale.com/stable/tailscale-setup-1.72.0.exe
install_script:
  path: ../lib/software/tailscale-install-script.ps1
uninstall_script:
  path: ../lib/software/tailscale-uninstall-script.ps1
self_service: true
```

### app_store_apps

- `app_store_id` is the ID of the Apple App Store app. You can find this at the end of the app's App Store URL. For example, "Bear - Markdown Notes" URL is "https://apps.apple.com/us/app/bear-markdown-notes/id1016366447" and the `app_store_id` is `1016366447`.

> Make sure to include only the ID itself, and not the `id` prefix shown in the URL. The ID must be wrapped in quotes as shown in the example so that it is processed as a string.

- `self_service` only applies to macOS, and is ignored for other platforms. For example, if the app is supported on macOS, iOS, and iPadOS, and `self_service` is set to `true`, it will be self-service on macOS workstations but not iPhones or iPads.

## org_settings and team_settings

### features

The `features` section of the configuration YAML lets you define what predefined queries are sent to the hosts and later on processed by Fleet for different functionalities.
- `additional_queries` adds extra host details. This information will be updated at the same time as other host details and is returned by the API when host objects are returned (default: empty).
- `enable_host_users` specifies whether or not Fleet collects user data from hosts (default: `true`).
- `enable_software_inventory` specifies whether or not Fleet collects softwre inventory from hosts (default: `true`).

#### Example

```yaml
org_settings:
  features:
    additional_queries:
      time: SELECT * FROM time
      macs: SELECT mac FROM interface_details
    enable_host_users: true
    enable_software_inventory: true
```

### fleet_desktop

Direct end users to a custom URL when they select **About Fleet** in the Fleet Desktop dropdown (default: [https://fleetdm.com/transparency](https://fleetdm.com/transparency)).

Can only be configured for all teams (`org_settings`).

#### Example

```yaml
org_settings:
  fleet_desktop:
    transparency_url: "https://example.org/transparency"
```

### host_expiry_settings

The `host_expiry_settings` section lets you define if and when hosts should be automatically deleted from Fleet if they have not checked in.
- `host_expiry_enabled` (default: `false`)
- `host_expiry_window` if a host has not communicated with Fleet in the specified number of days, it will be removed. Must be > `0` when host expiry is enabled (default: `0`).

#### Example

```yaml
org_settings:
  host_expiry_settings:
  	host_expiry_enabled: true
    host_expiry_window: 10
```

### org_info

- `name` is the name of your organization (default: `""`)
- `logo_url` is a public URL of the logo for your organization (default: Fleet logo).
- `org_logo_url_light_background` is a public URL of the logo for your organization that can be used with light backgrounds (default: Fleet logo).
- `contact_url` is a URL that appears in error messages presented to end users (default: `"https://fleetdm.com/company/contact"`)

Can only be configured for all teams (`org_settings`).

#### Example

```yaml
org_settings:
  org_info:
    org_name: Fleet
    org_logo_url: https://example.com/logo.png
    org_logo_url_light_background: https://example.com/logo-light.png
    contact_url: https://fleetdm.com/company/contact
```

### secrets

The `secrets` section defines the valid secrets that hosts can use to enroll to Fleet. Supply one of these secrets when generating the fleetd agent you'll use to enroll hosts. Learn more [here](https://fleetdm.com/docs/using-fleet/enroll-hosts).

#### Example

```yaml
org_settings:
  secrets: 
  - secret: $ENROLL_SECRET
```

### server_settings

- `enable_analytics` specifies whether or not to enable Fleet's [usage statistics](https://fleetdm.com/docs/using-fleet/usage-statistics) (default: `true`).
- `live_query_disabled` disables the ability to run live queries (ad hoc queries executed via the UI or fleetctl) (default: `false`).
- `query_reports_disabled` disables query reports and deletes existing repors (default: `false`).
- `query_report_cap` sets the maximum number of results to store per query report before the report is clipped. If increasing this cap, we recommend enabling reports for one query at time and monitoring your infrastructure. (Default: `1000`)
- `scripts_disabled` blocks access to run scripts. Scripts may still be added in the UI and CLI (defaul: `false`).
- `server_url` is the base URL of the Fleet instance. If this URL changes and Apple (macOS, iOS, iPadOS) hosts already have MDM turned on, the end users will have to turn MDM off and back on to use MDM features. (default: provided during Fleet setup)


Can only be configured for all teams (`org_settings`).

#### Example

  ```yaml
org_settings:
  server_settings:
    enable_analytics: true
    live_query_disabled: false
    query_reports_disabled: false
    scripts_disabled: false
    server_url: https://instance.fleet.com
  ```


### sso_settings

The `sso_settings` section lets you define single sign-on (SSO) settings. Learn more about SSO in Fleet [here](https://fleetdm.com/docs/deploying/configuration#configuring-single-sign-on-sso).

- `enable_sso` (default: `false`)
- `idp_name` is the human-friendly name for the identity provider that will provide single sign-on authentication (default: `""`).
- `entity_id` is the entity ID: a Uniform Resource Identifier (URI) that you use to identify Fleet when configuring the identity provider. It must exactly match the Entity ID field used in identity provider configuration (default: `""`).
- `metadata` is the metadata (in XML format) provided by the identity provider. (default: `""`)
- `metadata_url` is the URL that references the identity provider metadata. Only one of  `metadata` or `metadata_url` is required (default: `""`).
- `enable_jit_provisioning` specified whether or not to allow single sign-on login initiated by identity provider (default: `false`). 
- `enable_sso_idp_login` specifies whether or not to enables [just-in-time user provisioning](https://fleetdm.com/docs/deploy/single-sign-on-sso#just-in-time-jit-user-provisioning) (default: `false`).

Can only be configured for all teams (`org_settings`).

#### Example

```yaml
org_settings:
  sso_settings:
    enable_sso: true
    idp_name: SimpleSAML
    entity_id: https://example.com
    metadata: $SSO_METADATA
    enable_jit_provisioning: true # Available in Fleet Premium
    enable_sso_idp_login: true
```

### integrations

The `integrations` section lets you configure your Google Calendar, Jira, and Zendesk. After configuration, you can enable [automations](https://fleetdm.com/docs/using-fleet/automations) like calendar event and ticket creation for failing policies. Currently, enabling ticket creation is only available using Fleet's UI or [API](https://fleetdm.com/docs/rest-api/rest-api) (YAML files coming soon).

In addition, you can configure your the SCEP server to help your end users connect to Wi-Fi. Learn more about SCEP and NDES in Fleet [here](https://fleetdm.com/guides/ndes-scep-proxy).

#### Example

```yaml
org_settings:
  integrations:
    google_calendar:
      - api_key_json: $GOOGLE_CALENDAR_API_KEY_JSON
        domain: fleetdm.com
    jira:
      - url: https://example.atlassian.net
        username: user1
        api_token: $JIRA_API_TOKEN
        project_key: PJ1
    zendesk:
      - url: https://example.zendesk.com
        email: user1@example.com
        api_token: $ZENDESK_API_TOKEN
        group_id: 1234
    ndes_scep_proxy:
      url: https://example.com/certsrv/mscep/mscep.dll
      admin_url: https://example.com/certsrv/mscep_admin/
      username: Administrator@example.com
      password: 'myPassword'
```

For secrets, you can add [GitHub environment variables](https://docs.github.com/en/actions/learn-github-actions/variables#defining-environment-variables-for-a-single-workflow)

#### google_calendar

- `api_key_json` is the contents of the JSON file downloaded when you create your Google Workspace service account API key (default: `""`).
- `domain` is the primary domain used to identify your end user's work calendar (default: `""`).

#### jira

- `url` is the URL of your Jira (default: `""`)
- `username` is the username of your Jira account (default: `""`).
- `api_token` is the Jira API token (default: `""`).
- `project_key` is the project key location in your Jira project's URL. For example, in "jira.example.com/projects/EXMPL," "EXMPL" is the project key (default: `""`).

#### zendesk

- `url` is the URL of your Zendesk (default: `""`)
- `username` is the username of your Zendesk account (default: `""`).
- `api_token` is the Zendesk API token (default: `""`).
- `group_id`is found by selecting **Admin > People > Groups** in Zendesk. Find your group and select it. The group ID will appear in the search field.

#### ndes_scep_proxy
- `url` is the URL of the NDES SCEP endpoint (default: `""`).
- `admin_url` is the URL of the NDES admin endpoint (default: `""`).
- `username` is the username of the NDES admin endpoint (default: `""`).
- `password` is the password of the NDES admin endpoint (default: `""`).

### webhook_settings

The `webhook_settings` section lets you define webhook settings for failing policy, vulnerability, and host status automations. Learn more about automations in Fleet [here](https://fleetdm.com/docs/using-fleet/automations).

#### failing_policies_webhook

- `enable_failing_policies_webhook` (default: `false`)
- `destination_url` is the URL to `POST` to when the condition for the webhook triggers (default: `""`).
- `policy_ids` is the list of policies that will trigger a webhook.
- `host_batch_size` is the maximum number of hosts to batch in each webhook. A value of `0` means no batching (default: `0`).

#### Example

```yaml
org_settings:
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: https://example.org/webhook_handler
      host_batch_size: 0
      policy_ids:
        - 1
        - 2
        - 3
```

#### host_status_webhook

- `enable_host_status_webhook` (default: `false`)
- `destination_url` is the URL to `POST` to when the condition for the webhook triggers (default: `""`).
- `days_count` is the number of days that hosts need to be offline to count as part of the percentage (default: `0`).
- `host_percentage` is the percentage of hosts that need to be offline to trigger the webhook. (default: `0`).

#### Example

```yaml
org_settings:
  webhook_settings:
    host_status_webhook:
      enable_host_status_webhook: true
      destination_url: https://example.org/webhook_handler
      days_count: 7
      host_percentage: 25
```

#### vulnerabilities_webhook

- `enable_vulnerabilities_webhook` (default: `false`)
- `destination_url` is the URL to `POST` to when the condition for the webhook triggers (default: `""`).
- `days_count` is the number of days that hosts need to be offline to count as part of the percentage (default: `0`).
- `host_batch_size` is the maximum number of hosts to batch in each webhook. A value of `0` means no batching (default: `0`).

#### Example

```yaml
org_settings:
  webhook_settings:
    vulnerabilities_webhook:
      enable_vulnerabilities_webhook: true
      destination_url: https://example.org/webhook_handler
      host_batch_size: 0
```

Can only be configured for all teams (`org_settings`).

### mdm

#### apple_business_manager

After you've uploaded an Apple Business Manager (ABM) token, the `apple_business_manager` section lets you configure the teams in Fleet new hosts in ABM are automatically added to. Currently, adding an ABM token is only available using Fleet's UI. Learn more [here](https://fleetdm.com/guides/macos-mdm-setup#automatic-enrollment).

Currently, managing labels and users, ticket destinations (Jira and Zendesk), Apple Business Manager (ABM) are only supported using Fleet's UI or [API](https://fleetdm.com/docs/rest-api/rest-api) (YAML files coming soon).

- `organization_name` is the organization name associated with the Apple Business Manager account.
- `macos_team` is the team where macOS hosts are automatically added when they appear in Apple Business Manager.
- `ios_team` is the the team where iOS hosts are automatically added when they appear in Apple Business Manager.
- `ipados_team` is the team where iPadOS hosts are automatically added when they appear in Apple Business Manager.

#### Example

```yaml
org_settings:
  mdm:
    apple_business_manager: # Available in Fleet Premium
    - organization_name: Fleet Device Management Inc.
      macos_team: "💻 Workstations" 
      ios_team: "📱🏢 Company-owned iPhones"
      ipados_team: "🔳🏢 Company-owned iPads"
```

> Apple Business Manager settings can only be configured for all teams (`org_settings`).

#### volume_purchasing_program

After you've uploaded a Volume Purchasing Program (VPP) token, the  `volume_purchasing_program` section lets you configure the teams in Fleet that have access to that VPP token's App Store apps. Currently, adding a VPP token is only available using Fleet's UI. Learn more [here](https://fleetdm.com/guides/macos-mdm-setup#volume-purchasing-program-vpp).

- `location` is the name of the location in the Apple Business Manager account.
- `teams` is a list of team names. If you choose specific teams, App Store apps in this VPP account will only be available to install on hosts in these teams. If not specified, App Store apps are available to install on hosts in all teams.

#### Example

```yaml
org_settings:
  mdm:
    volume_purchasing_program: # Available in Fleet Premium
    - location: Fleet Device Management Inc.
      teams: 
      - "💻 Workstations" 
      - "💻🐣 Workstations (canary)"
      - "📱🏢 Company-owned iPhones"
      - "🔳🏢 Company-owned iPads"
```

Can only be configured for all teams (`org_settings`).

#### end_user_authentication

The `end_user_authentication` section lets you define the identity provider (IdP) settings used for end user authentication during Automated Device Enrollment (ADE). Learn more about end user authentication in Fleet [here](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication-and-eula).

Once the IdP settings are configured, you can use the [`controls.macos_setup.enable_end_user_authentication`](#macos_setup) key to control the end user experience during ADE.

- `idp_name` is the human-friendly name for the identity provider that will provide single sign-on authentication (default: `""`).
- `entity_id` is the entity ID: a Uniform Resource Identifier (URI) that you use to identify Fleet when configuring the identity provider. It must exactly match the Entity ID field used in identity provider configuration (default: `""`).
- `metadata` is the metadata (in XML format) provided by the identity provider. (default: `""`)
- `metadata_url` is the URL that references the identity provider metadata. Only one of  `metadata` or `metadata_url` is required (default: `""`).

Can only be configured for all teams (`org_settings`).

#### end_user_authentication

The `end_user_authentication` section lets you define the identity provider (IdP) settings used for end user authentication during Automated Device Enrollment (ADE). Learn more about end user authentication in Fleet [here](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication-and-eula).

Once the IdP settings are configured, you can use the [`controls.macos_setup.enable_end_user_authentication`](#macos_setup) key to control the end user experience during ADE.

- `idp_name` is the human-friendly name for the identity provider that will provide single sign-on authentication (default: `""`).
- `entity_id` is the entity ID: a Uniform Resource Identifier (URI) that you use to identify Fleet when configuring the identity provider. It must exactly match the Entity ID field used in identity provider configuration (default: `""`).
- `metadata` is the metadata (in XML format) provided by the identity provider. (default: `""`)
- `metadata_url` is the URL that references the identity provider metadata. Only one of  `metadata` or `metadata_url` is required (default: `""`).

Can only be configured for all teams (`org_settings`).

##### apple_server_url

Update this URL if you're self-hosting Fleet and you want your hosts to talk to this URL for MDM features. (If not configured, hosts will use the base URL of the Fleet instance.)

If this URL changes and hosts already have MDM turned on, the end users will have to turn MDM off and back on to use MDM features.

##### Example

```yaml
org_settings:
  mdm:
    apple_server_url: https://instance.fleet.com
```

Can only be configured for all teams (`org_settings`).

#### yara_rules

The `yara_rules` section lets you define [YARA rules](https://virustotal.github.io/yara/) that will be served by Fleet's authenticated
YARA rule functionality. Learn more about authenticated YARA rules in Fleet
[here](https://fleetdm.com/guides/remote-yara-rules).

Each entry should be the relative path to a valid YARA rule file.

##### Example

```yaml
org_settings:
  yara_rules:
    - path: ./lib/rule1.yar
    - path: ./lib/rule2.yar
```

Can only be configured for all teams (`org_settings`). To target rules to specific teams, target the
queries referencing the rules to the desired teams.

<meta name="title" value="YAML files">
<meta name="description" value="Reference documentation for Fleet's GitOps workflow. See examples and configuration options.">
<meta name="pageOrderInSection" value="1500">
