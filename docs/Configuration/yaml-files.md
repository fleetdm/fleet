# GitOps

Use Fleet's best practice GitOps workflow to manage your computers as code. To learn how to set up a GitOps workflow see the [Fleet GitOps repo](https://github.com/fleetdm/fleet-gitops).

Fleet GitOps workflow is designed to be applied to all teams at once. However, the flow can be customized to only modify specific teams and/or global settings.

Users that have global admin permissions may apply GitOps configurations globally and to all teams, while users whose permissions are scoped to specific teams may apply settings to only to teams they has permissions to modify.

Any settings not defined in your YAML files (including missing or mispelled keys) will be reset to the default values, which may include deleting assets such as software packages.

The following are the required keys in the `default.yml` and any `teams/team-name.yml` files:

```yaml
name: # Only teams/team-name.yml. To edit a team's name, change `name` but don't change the filename.
policies:
queries:
agent_options:
controls: # Can be defined in teams/no-team.yml too.
software: # Can be defined in teams/no-team.yml too
org_settings: # Only default.yml
team_settings: # Only teams/team-name.yml
```

You may also wish to create specialized API-Only users which may modify configurations through GitOps, but cannot access fleet through the UI. These specialized users can be created through `fleetctl user create` with the `--api-only` flag, and then assigned the `GitOps` role, and given global or team scope in the UI.

## labels

Labels can be specified in your `default.yml` file using inline configuration or references to separate files in your `lib/` folder.

> `labels` is an optional key: if included, existing labels not listed will be deleted. If the `label` key is omitted, existing labels will stay intact. For this reason, enabling [GitOps mode](https://fleetdm.com/learn-more-about/ui-gitops-mode) _does not_ restrict creating/editing labels via the UI.
>
> Any labels referenced in other sections (like [policies](https://fleetdm.com/docs/configuration/yaml-files#policies), [queries](https://fleetdm.com/docs/configuration/yaml-files#queries) or [software](https://fleetdm.com/docs/configuration/yaml-files#software)) _must_ be specified in the `labels` section.

### Options

For possible options, see the parameters for the [Add label API endpoint](https://fleetdm.com/docs/rest-api/rest-api#add-label).

### Example

#### Inline

`default.yml`

```yaml
labels:
  - name: Arm64
    description: Hosts on the Arm64 architecture
    query: "SELECT 1 FROM system_info WHERE cpu_type LIKE 'arm64%' OR cpu_type LIKE 'aarch64%'"
    label_membership_type: dynamic
  - name: C-Suite
    description: Hosts belonging to the C-Suite
    label_membership_type: manual
    hosts:
      - "ceo-laptop"
      - "the-CFOs-computer"
```

#### Separate file
 
`lib/labels-name.labels.yml`

```yaml
- name: Arm64
  description: Hosts on the Arm64 architecture
  query: SELECT 1 FROM system_info WHERE cpu_type LIKE "arm64%" OR cpu_type LIKE "aarch64%"
  label_membership_type: dynamic
- name: C-Suite
  description: Hosts belonging to the C-Suite
  label_membership_type: manual
  hosts:
    - "ceo-laptop"
    - "the-CFOs-computer"
```

`lib/default.yml`

```yaml
labels:
  path: ./lib/labels-name.labels.yml
```


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
    query: "SELECT 1 FROM filevault_status WHERE status = 'FileVault is On.';"
    platform: darwin
    critical: false
    calendar_events_enabled: false
    conditional_access_enabled: true
    labels_include_any:
      - Engineering
      - Customer Support
```

#### Separate file
 
`lib/policies-name.policies.yml`

```yaml
- name: macOS - Enable FileVault
  description: This policy checks if FileVault (disk encryption) is enabled.
  resolution: As an IT admin, turn on disk encryption in Fleet.
  query: "SELECT 1 FROM filevault_status WHERE status = 'FileVault is On.';"
  platform: darwin
  critical: false
  calendar_events_enabled: false
  conditional_access_enabled: true
- name: macOS - Disable guest account
  description: This policy checks if the guest account is disabled.
  resolution: As an IT admin, deploy a macOS, login window profile with the DisableGuestAccount option set to true.
  query: "SELECT 1 FROM managed_policies WHERE domain='com.apple.mcx' AND username = '' AND name='DisableGuestAccount' AND CAST(value AS INT) = 1;"
  platform: darwin
  critical: false
  calendar_events_enabled: false
  run_script:
    path: ./disable-guest-account.sh
- name: Install Firefox on macOS
  platform: darwin
  description: This policy checks that Firefox is installed.
  resolution: Install Firefox app if not installed.
  query: "SELECT 1 FROM apps WHERE name = 'Firefox.app'"
  install_software:
    package_path: ./firefox.package.yml
- name: [Install software] Logic Pro
  platform: darwin
  description: This policy checks that Logic Pro is installed
  resolution: Install Logic Pro App Store app if not installed
  query: "SELECT 1 FROM apps WHERE name = 'Logic Pro'"
  install_software:
    package_path: ./linux-firefox.deb.package.yml
    # app_store_id: "1487937127" (for App Store apps)
```

`default.yml` (for policies that neither install software nor run scripts), `teams/team-name.yml`, or `teams/no-team.yml`

```yaml
policies:
  - path: ../lib/policies-name.policies.yml
```

> Currently, the `run_script` and `install_software` policy automations can only be configured for a team (`teams/team-name.yml`) or "No team" (`teams/no-team.yml`). The automations can only be added to policies in which the script (or software) is defined in the same team (or "No team"). `calendar_events_enabled` can only be configured for policies on a team.

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
    labels_include_any:
      - Engineering
      - Customer Support
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
    labels_include_any:
      - Engineering
      - Customer Support
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
agent_options:
  path: ../lib/agent-options.yml
```

## controls

The `controls` section allows you to configure scripts and device management (MDM) features in Fleet.

- `scripts` is a list of paths to macOS, Windows, or Linux scripts.
- `windows_enabled_and_configured` specifies whether or not to turn on Windows MDM features (default: `false`). Can only be configured for all teams (`default.yml`).
- `windows_migration_enabled` specifies whether or not to automatically migrate Windows hosts connected to another MDM solution. If `false`, MDM is only turned on after hosts are unenrolled from your old MDM solution (default: `false`). Can only be configured for all teams (`default.yml`).
- `enable_disk_encryption` specifies whether or not to enforce disk encryption on macOS, Windows, and Linux hosts (default: `false`).
- `windows_require_bitlocker_pin` specifies whether or not to require end users on Windows hosts to set a BitLocker PIN. When set, this PIN is required to unlock Windows host during startup. `enable_disk_encryption` must be set to `true`. (default: `false`).

#### Example

```yaml
controls:
  scripts: 
    - path: ../lib/macos-script.sh 
    - path: ../lib/windows-script.ps1
    - path: ../lib/linux-script.sh
  windows_enabled_and_configured: true
  windows_migration_enabled: true # Available in Fleet Premium
  enable_disk_encryption: true # Available in Fleet Premium
  macos_updates: # Available in Fleet Premium
    deadline: "2024-12-31"
    minimum_version: "15.1"
  ios_updates: # Available in Fleet Premium
    deadline: "2024-12-31"
    minimum_version: "18.1"
  ipados_updates: # Available in Fleet Premium
    deadline: "2024-12-31"
    minimum_version: "18.1"
  windows_updates: # Available in Fleet Premium
    deadline_days: 5
    grace_period_days: 2
  macos_settings:
    custom_settings:
      - path: ../lib/macos-profile1.mobileconfig
        labels_exclude_any: # Available in Fleet Premium
          - Macs on Sequoia
      - path: ../lib/macos-profile2.json
        labels_include_all: # Available in Fleet Premium
          - Macs on Sonoma
      - path: ../lib/macos-profile3.mobileconfig
        labels_include_any: # Available in Fleet Premium
          - Engineering
          - Product
  windows_settings:
    custom_settings:
      - path: ../lib/windows-profile.xml
  macos_setup: # Available in Fleet Premium
    bootstrap_package: https://example.org/bootstrap_package.pkg
    enable_end_user_authentication: true
    enable_release_device_manually: true
    macos_setup_assistant: ../lib/dep-profile.json
    script: ../lib/macos-setup-script.sh
  macos_migration: # Available in Fleet Premium
    enable: true
    mode: voluntary
    webhook_url: https://example.org/webhook_handler
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

- `macos_settings.custom_settings` is a list of paths to macOS, iOS, and iPadOS configuration profiles (.mobileconfig) or declaration profiles (.json).
- `windows_settings.custom_settings` is a list of paths to Windows configuration profiles (.xml).

Use `labels_include_all` to target hosts that have all labels, `labels_include_any` to target hosts that have any label, or `labels_exclude_any` to target hosts that don't have any of the labels. Only one of `labels_include_all`, `labels_include_any`, or `labels_exclude_any` can be specified. If none are specified, all hosts are targeted.

For macOS configuration profiles, you can use any of Apple's [built-in variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0).

Fleet also supports adding [GitHub](https://docs.github.com/en/actions/learn-github-actions/variables#defining-environment-variables-for-a-single-workflow) or [GitLab](https://docs.gitlab.com/ci/variables/) environment variables in your configuration profiles. Use `$ENV_VARIABLE` format. 

In Fleet Premium, you can use reserved variables beginning with `$FLEET_VAR_` (currently available only for Apple profiles). Fleet will populate these variables when profiles are sent to hosts. Supported variables are:

- `$FLEET_VAR_NDES_SCEP_CHALLENGE`
- `$FLEET_VAR_NDES_SCEP_PROXY_URL`
- `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`: host's IdP username. When this changes, Fleet will automatically resend the profile.
- `$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART`: local part of the email (e.g. john from john@example.com). When this changes, Fleet will automatically resend the profile.
- `$FLEET_VAR_HOST_END_USER_IDP_GROUPS`: comma separated IdP groups that host belongs to. When these change, Fleet will automatically resend the profile.
- `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>` (`<CA_NAME>` should be replaced with name of the certificate authority configured in [scep_proxy](#scep-proxy).)
- `$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>`
- `$FLEET_VAR_DIGICERT_PASSWORD_<CA_NAME>` (`<CA_NAME>` should be replaced with name of the certificate authority configured in [digicert](#digicert).)
- `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>`
- `$FLEET_VAR_HYDRANT_DATA_<CA_NAME>` (`<CA_NAME>` should be replaced with name of the certificate authority configured in [hydrant](#hydrant).)

### macos_setup

The `macos_setup` section lets you control the out-of-the-box macOS [setup experience](https://fleetdm.com/guides/macos-setup-experience) for hosts that use Automated Device Enrollment (ADE).

> **Experimental feature.** The `manual_agent_install` feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

- `bootstrap_package` is the URL to a bootstrap package. Fleet will download the bootstrap package (default: `""`).
- `manual_agent_install` specifies whether Fleet's agent (fleetd) will be installed as part of setup experience. (default: `false`)
- `enable_end_user_authentication` specifies whether or not to require end user authentication when the user first sets up their macOS host. 
- `macos_setup_assistant` is a path to a custom automatic enrollment (ADE) profile (.json).
- `script` is the path to a custom setup script to run after the host is first set up.

### macos_migration

The `macos_migration` section lets you control the [end user migration workflow](https://fleetdm.com/docs/using-fleet/mdm-migration-guide#end-user-workflow) for macOS hosts that enrolled to your old MDM solution.

- `enable` specifies whether or not to enable end user migration workflow (default: `false`)
- `mode` specifies whether the end user initiates migration (`voluntary`) or they're nudged every 15-20 minutes to migrate (`forced`) (default: `""`).
- `webhook_url` is the URL that Fleet sends a webhook to when the end user selects **Start**. Receive this webhook using your automation tool (ex. Tines) to unenroll your end users from your old MDM solution.

Can only be configured for all teams (`default.yml`).

## software

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

The `software` section allows you to configure packages, Apple App Store apps, and Fleet-maintained apps that you want to install on your hosts.

- `packages` is a list of paths to custom packages (.pkg, .msi, .exe, .rpm, .deb, or .tar.gz).
- `app_store_apps` is a list of Apple App Store apps.
- `fleet_maintained_apps` is a list of Fleet-maintained apps.

Currently, you can specify `install_software` in the [`policies` YAML](#policies) to automatically install a custom package or App Store app when a host fails a policy. [Automatic install support for Fleet-maintained apps](https://github.com/fleetdm/fleet/issues/29584) is coming soon.

#### Example

`teams/team-name.yml`, or `teams/no-team.yml`

```yaml
software:
  packages:
    - path: ../lib/software-name.package.yml
      categories:
        - Browsers
      self_service: true
      setup_experience: true
    - path: ../lib/software-name2.package.yml
  app_store_apps:
    - app_store_id: "1091189122"
      labels_include_any: # Available in Fleet Premium
        - Product
        - Marketing
      categories:
        - Communication
  fleet_maintained_apps:
    - slug: slack/darwin
      self_service: true
      labels_include_any:
        - Design
        - Sales
      categories:
        - Communication
        - Productivity
```

#### self_service, labels, categories, and setup_experience

- `self-service` specifies whether end users can install from **Fleet Desktop > Self-service** (default: `false`). Currently, for App Store apps, this setting only applies to macOS and is ignored on other platforms. For example, if the app is supported on macOS, iOS, and iPadOS, and `self_service` is set to `true`, it will be available in self-service on macOS workstations but not on iPhones or iPads.
- `labels_include_any` targets hosts that have **any** of the specified labels. `labels_exclude_any` targets hosts that have **none** of the specified labels. Only one of these fields can be set. If neither is set, all hosts are targeted.
- `categories` groups self-service software on your end users' **Fleet Desktop > My device** page. If none are set, Fleet-maintained apps get their [default categories](https://github.com/fleetdm/fleet/tree/main/ee/maintained-apps/outputs) and all other software only appears in the **All** group. Supported values:
  - `Browsers`: shown as **🌎 Browsers**
  - `Communication`: shown as **👬 Communication**
  - `Developer tools`: shown as **🧰 Developer tools**
  - `Productivity`: shown as **🖥️ Productivity**
- `setup_experience` installs the software on macOS hosts that automatically enroll via [setup experience](https://fleetdm.com/guides/macos-setup-experience#software-and-script). This setting only applies to macOS and is ignored on other platforms (default: `false`).

### packages

- `url` specifies the URL at which the software is located. Fleet will download the software and upload it to S3.
- `hash_sha256` specifies the SHA256 hash of the package file. If provided, and if a software package with that hash has already been uploaded to Fleet, the existing package will be used and download will be skipped. If a package with that hash does not yet exist, Fleet will download the package, then verify that the hash matches, bailing out if it does not match.

> Without specifying a hash, Fleet downloads each installer for each team on each GitOps run.

> You can specify a hash alone to reference a software package that was previously uploaded to Fleet, whether via the UI or the API,. If a package with that hash isn't already in Fleet and visible to the user performing the GitOps run, the GitOps run will error.

- `pre_install_query.path` is the osquery query Fleet runs before installing the software. Software will be installed only if the [query returns results](https://fleetdm.com/tables).
- `install_script.path` specifies the command Fleet will run on hosts to install software. The [default script](https://github.com/fleetdm/fleet/tree/main/pkg/file/scripts) is dependent on the software type (i.e. .pkg).
- `uninstall_script.path` is the script Fleet will run on hosts to uninstall software. The [default script](https://github.com/fleetdm/fleet/tree/main/pkg/file/scripts) is dependent on the software type (i.e. .pkg).
- `post_install_script.path` is the script Fleet will run on hosts after the software install. There is no default.

> Without specifying a hash, Fleet downloads each installer for each team on each GitOps run.

#### Example

##### With URL

`lib/software-name.package.yml`:

```yaml
url: https://dl.tailscale.com/stable/tailscale-setup-1.72.0.exe
install_script:
  path: ../lib/software/tailscale-install-script.ps1
uninstall_script:
  path: ../lib/software/tailscale-uninstall-script.ps1
post_install_script:
  path: ../lib/software/tailscale-config-script.ps1
```

##### With hash

You can view the hash for existing software in the software detail page in the Fleet UI. It is also returned after uploading a new software item via the API.

```yaml
# Mozilla Firefox (Firefox 136.0.1.pkg) version 136.0.1
- hash_sha256: fd22528a87f3cfdb81aca981953aa5c8d7084581b9209bb69abf69c09a0afaaf
```

### app_store_apps

- `app_store_id` is the ID of the Apple App Store app. You can find this at the end of the app's App Store URL. For example, "Bear - Markdown Notes" URL is "https://apps.apple.com/us/app/bear-markdown-notes/id1016366447" and the `app_store_id` is `1016366447`.
  + Make sure to include only the ID itself, and not the `id` prefix shown in the URL. The ID must be wrapped in quotes as shown in the example so that it is processed as a string.
- `self_service` only applies to macOS, and is ignored for other platforms. For example, if the app is supported on macOS, iOS, and iPadOS, and `self_service` is set to `true`, it will be self-service on macOS workstations but not iPhones or iPads.
- `categories` is an array of categories. See [supported categories](#labels-and-categories).

Currently, one app for each of an App Store app's supported platforms are added. For example, adding [Bear](https://apps.apple.com/us/app/bear-markdown-notes/id1016366447) (supported on iOS and iPadOS) adds both the iOS and iPadOS apps to your software that's available to install in Fleet. Specifying specific platforms is only supported using Fleet's UI or [API](https://fleetdm.com/docs/rest-api/rest-api) (YAML coming soon).

### fleet_maintained_apps

- `fleet_maintained_apps` is a list of Fleet-maintained apps. Provide the `slug` field to include a Fleet-maintained app on a team. To find the `slug`, head to **Software > Add software** and select a Fleet-maintained app, then select **Show details**. You can also see the [list of app slugs on GitHub](https://github.com/fleetdm/fleet/blob/main/ee/maintained-apps/outputs/apps.json).

Currently, Fleet-maintained apps will be updated to the latest version published by Fleet when GitOps runs, [with the exception of Chrome](https://github.com/fleetdm/fleet/issues/30325).

## org_settings and team_settings

Currently, managing users and ticket destinations (Jira and Zendesk) are only supported using Fleet's UI or [API](https://fleetdm.com/docs/rest-api/rest-api).

### features

The `features` section of the configuration YAML lets you define what predefined queries are sent to the hosts and later on processed by Fleet for different functionalities.
- `additional_queries` adds extra host details. This information will be updated at the same time as other host details and is returned by the API when host objects are returned (default: empty).
- `enable_host_users` specifies whether or not Fleet collects user data from hosts (default: `true`).
- `enable_software_inventory` specifies whether or not Fleet collects software inventory from hosts (default: `true`).

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
    transparency_url: https://example.org/transparency
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

- `org_name` is the name of your organization (default: `""`)
- `org_logo_url` is a public URL of the logo for your organization (default: Fleet logo).
- `org_logo_url_light_background` is a public URL of the logo for your organization that can be used with light backgrounds (default: Fleet logo).
- `contact_url` is a URL that appears in error messages presented to end users (default: `"https://fleetdm.com/company/contact"`)

Can only be configured for all teams (`org_settings`).

To get the best results for your logos (`org_logo_url` and `org_logo_url_light_background`), use the following sizes:
- For square logos, use a PNG that's 256x256 pixels (px).
- For rectangular logos (wordmark), use a PNG that's 516x256 pixels (px).

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

The `secrets` section defines the valid secrets that hosts can use to enroll to Fleet. Supply one of these secrets when generating the fleetd agent you'll use to [enroll hosts](https://fleetdm.com/docs/using-fleet/enroll-hosts).

#### Example

```yaml
org_settings:
  secrets: 
  - secret: $ENROLL_SECRET
```

### server_settings

- `ai_features_disabled` disables AI-assisted policy descriptions and resolutions. (default: `false`)
- `enable_analytics` specifies whether or not to enable Fleet's [usage statistics](https://fleetdm.com/docs/using-fleet/usage-statistics). (default: `true`)
- `live_query_disabled` disables the ability to run live queries (ad hoc queries executed via the UI or fleetctl). (default: `false`)
- `query_reports_disabled` disables query reports and deletes existing reports. (default: `false`)
- `query_report_cap` sets the maximum number of results to store per query report before the report is clipped. If increasing this cap, we recommend enabling reports for one query at a time and monitoring your infrastructure. (default: `1000`)
- `scripts_disabled` blocks access to run scripts. Scripts may still be added in the UI and CLI. (default: `false`)
- `server_url` is the base URL of the Fleet instance. If this URL changes and Apple (macOS, iOS, iPadOS) hosts already have MDM turned on, the end users will have to turn MDM off and back on to use MDM features. (default: provided during Fleet setup)


Can only be configured for all teams (`org_settings`).

#### Example

```yaml
org_settings:
  server_settings:
    ai_features_disabled: false
    enable_analytics: true
    live_query_disabled: false
    query_reports_disabled: false
    scripts_disabled: false
    server_url: https://instance.fleet.com
```


### sso_settings

The `sso_settings` section lets you define [single sign-on (SSO)](https://fleetdm.com/docs/deploying/configuration#configuring-single-sign-on-sso) settings.

- `enable_sso` (default: `false`)
- `idp_name` is the human-friendly name for the identity provider that will provide single sign-on authentication (default: `""`).
- `idp_image_url` is an optional link to an image such as a logo for the identity provider. (default: `""`).
- `entity_id` is the entity ID: a Uniform Resource Identifier (URI) that you use to identify Fleet when configuring the identity provider. It must exactly match the Entity ID field used in identity provider configuration (default: `""`).
- `metadata` is the metadata (in XML format) provided by the identity provider. (default: `""`)
- `metadata_url` is the URL that references the identity provider metadata. Only one of  `metadata` or `metadata_url` is required (default: `""`).
- `enable_jit_provisioning` specifies whether or not to enable [just-in-time user provisioning](https://fleetdm.com/docs/deploy/single-sign-on-sso#just-in-time-jit-user-provisioning) (default: `false`).
- `enable_sso_idp_login` specifies whether or not to allow single sign-on login initiated by identity provider (default: `false`).

Can only be configured for all teams (`org_settings`).

#### Example

```yaml
org_settings:
  sso_settings:
    enable_sso: true
    idp_name: Okta
    idp_image_url: https://www.okta.com/favicon.ico
    entity_id: https://example.okta.com
    metadata: $SSO_METADATA
    enable_jit_provisioning: true # Available in Fleet Premium
    enable_sso_idp_login: true
```

### integrations

The `integrations` section lets you configure your Google Calendar, Conditional Access (for hosts in "No team"), Jira, and Zendesk. After configuration, you can enable [automations](https://fleetdm.com/docs/using-fleet/automations) like calendar event and ticket creation for failing policies. Currently, enabling ticket creation is only available using Fleet's UI or [API](https://fleetdm.com/docs/rest-api/rest-api) (YAML files coming soon).

In addition, you can configure your [certificate authorities (CA)](https://fleetdm.com/guides/certificate-authorities) to help your end users connect to Wi-Fi.

#### Example

`default.yml`

```yaml
org_settings:
  integrations:
    conditional_access_enabled: true
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
    digicert: # Available in Fleet Premium
      - name: DIGICERT_WIFI
        url: https://one.digicert.com
        api_token: $DIGICERT_API_TOKEN
        profile_id: 926dbcdd-41c4-4fe5-96c3-b6a7f0da81d8
        certificate_common_name: $FLEET_VAR_HOST_HARDWARE_SERIAL@example.com
        certificate_user_principal_names:
          - $FLEET_VAR_HOST_HARDWARE_SERIAL@example.com
        certificate_seat_id: $FLEET_VAR_HOST_HARDWARE_SERIAL@example.com
    ndes_scep_proxy: # Available in Fleet Premium
      url: https://example.com/certsrv/mscep/mscep.dll
      admin_url: https://example.com/certsrv/mscep_admin/
      username: Administrator@example.com
      password: myPassword
    custom_scep_proxy: # Available in Fleet Premium
      - name: SCEP_VPN
        url: https://example.com/scep
        challenge: $SCEP_VPN_CHALLENGE
    hydrant: # Available in Fleet Premium
      - name: HYDRANT_WIFI
        url: https://example.hydrantid.com/.well-known/est/abc123
        client_id: $HYDRANT_CLIENT_ID
        client_secret: $HYDRANT_CLIENT_SECRET
```

`/teams/team-name.yml`

At the team level, there is the additional option to enable conditional access, which blocks third party app sign-ins on hosts failing policies. (Available in Fleet Premium. Must have Microsoft Entra connected.)

```yaml
integrations:
  conditional_access_enabled: true
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

#### digicert

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

- `name` is the name of certificate authority that will be used in variables in configuration profiles. Only letters, numbers, and underscores are allowed.
- `url` is the URL to DigiCert One instance (default: `https://one.digicert.com`).
- `api_token` is the token used to authenticate requests to DigiCert.
- `profile_id` is the ID of certificate profile in DigiCert.
- `certificate_common_name` is the certificate's CN.
- `certificate_user_principal_names` is the certificate's user principal names (UPN) attribute in Subject Alternative Name (SAN).
- `certificate_seat_id` is the ID of the DigiCert's seat. Seats are license units in DigiCert.

#### ndes_scep_proxy
- `url` is the URL of the NDES SCEP endpoint (default: `""`).
- `admin_url` is the URL of the NDES admin endpoint (default: `""`).
- `username` is the username of the NDES admin endpoint (default: `""`).
- `password` is the password of the NDES admin endpoint (default: `""`).

#### scep_proxy

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

- `name` is the name of certificate authority that will be used in variables in configuration profiles. Only letters, numbers, and underscores are allowed.
- `url` is the URL of the Simple Certificate Enrollment Protocol (SCEP) server.
- `challenge` is the static challenge password used to authenticate requests to SCEP server.

#### hydrant

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

- `name` is the name of the certificate authority that will be used in variables in configuration profiles. Only letters, numbers, and underscores are allowed.
- `url` is the EST (Enrollment Over Secure Transport) endpoint provided by Hydrant.
- `client_id` is the client ID provided by Hydrant.
- `client_secret` is the client secret provided by Hydrant.

### webhook_settings

The `webhook_settings` section lets you define webhook settings for failing policy, vulnerability, and host status [automations](https://fleetdm.com/docs/using-fleet/automations).

#### activities_webhook

- `enable_activities_webhook` (default: `false`)
- `destination_url` is the URL to `POST` to when an activity is generated (default: `""`)

### Example

```yaml
org_settings:
  webhook_settings:
    activities_webhook:
      enable_activities_webhook: true
      destination_url: https://example.org/webhook_handler
```

#### failing_policies_webhook

> These settings can also be configured per-team when nested under `team_settings`. 

- `enable_failing_policies_webhook` (default: `false`)
- `destination_url` is the URL to `POST` to when the condition for the webhook triggers (default: `""`).
- `policy_ids` is the list of policies that will trigger a webhook.
- `host_batch_size` is the maximum number of host identifiers to send in one webhook request. A value of `0` means all host identifiers with a failing policy will be sent in a single request.

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
- `host_batch_size` is the maximum number of host identifiers to send in one webhook request. A value of `0` means all host identifiers with a detected vulnerability will be sent in a single request.

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

After [adding an Apple Business Manager (ABM) token via the UI](https://fleetdm.com/guides/macos-mdm-setup#automatic-enrollment), the `apple_business_manager` section lets you determine which team Apple devices are assigned to in Fleet when they appear in Apple Business Manager.

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
      macos_team: 💻 Workstations
      ios_team: 📱🏢 Company-owned iPhones
      ipados_team: 🔳🏢 Company-owned iPads
```

> Apple Business Manager settings can only be configured for all teams (`org_settings`).

#### volume_purchasing_program

After you've uploaded a [Volume Purchasing Program](https://fleetdm.com/guides/macos-mdm-setup#volume-purchasing-program-vpp) (VPP) token, the  `volume_purchasing_program` section lets you configure the teams in Fleet that have access to that VPP token's App Store apps. Currently, adding a VPP token is only available using Fleet's UI.

- `location` is the name of the location in the Apple Business Manager account.
- `teams` is a list of team names. If you choose specific teams, App Store apps in this VPP account will only be available to install on hosts in these teams. If not specified, App Store apps are available to install on hosts in all teams.

#### Example

```yaml
org_settings:
  mdm:
    volume_purchasing_program: # Available in Fleet Premium
    - location: Fleet Device Management Inc.
      teams: 
      - 💻 Workstations
      - 💻🐣 Workstations (canary)
      - 📱🏢 Company-owned iPhones
      - 🔳🏢 Company-owned iPads
```

Can only be configured for all teams (`org_settings`).

#### end_user_authentication

The `end_user_authentication` section lets you define the identity provider (IdP) settings used for [end user authentication](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication-and-eula) during Automated Device Enrollment (ADE).

Once the IdP settings are configured, you can use the [`controls.macos_setup.enable_end_user_authentication`](#macos-setup) key to control the end user experience during ADE.

Can only be configured for all teams (`org_settings`):

- `idp_name` is the human-friendly name for the identity provider that will provide single sign-on authentication (default: `""`).
- `entity_id` is the entity ID: a Uniform Resource Identifier (URI) that you use to identify Fleet when configuring the identity provider. It must exactly match the Entity ID field used in identity provider configuration (default: `""`).
- `metadata` is the metadata (in XML format) provided by the identity provider. (default: `""`)
- `metadata_url` is the URL that references the identity provider metadata. Only one of  `metadata` or `metadata_url` is required (default: `""`).

#### Example

```
org_settings:
  mdm:
    end_user_authentication:
      entity_id: https://example.okta.com
      idp_name: Okta
      metadata: $END_USER_SSO_METADATA
      metadata_url: ""
```

Can only be configured for all teams (`org_settings`).

##### end_user_license_agreement

You can require an end user to agree to an end user license agreement (EULA) before they can use their new Mac. `end_user_authentication` must be configured, and `controls.enable_end_user_authentication` must be set to `true`.

- `end_user_license_agreement` is the path to the PDF document.

##### Example

```yaml
org_settings:
  mdm:
    end_user_license_agreement: ./lib/eula.pdf
```

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

The `yara_rules` section lets you define [YARA rules](https://virustotal.github.io/yara/) that will be served by Fleet's [authenticated
YARA rule](https://fleetdm.com/guides/remote-yara-rules) functionality.

##### Example

```yaml
org_settings:
  yara_rules:
    - path: ./lib/rule1.yar
    - path: ./lib/rule2.yar
```

Can only be configured for all teams (`org_settings`). To target rules to specific teams, target the
queries referencing the rules to the desired teams.

<meta name="title" value="GitOps">
<meta name="description" value="Reference documentation for Fleet's GitOps workflow. See examples and configuration options.">
<meta name="pageOrderInSection" value="1500">
