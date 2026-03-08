# fleetctl apply

The `fleectl apply` command and YAML interface is used for one-off imports and backwards compatibility GitOps.

To use Fleet's best practice GitOps, check out the [GitOps docs](https://fleetdm.com/docs/using-fleet/gitops).

## Queries

The `queries` YAML file controls queries in Fleet.

You can define one or more queries in the same file with `---`.

The following example file includes several queries:

```yaml
---
apiVersion: v1
kind: query
spec:
  name: osquery_info
  description: A heartbeat counter that reports general performance (CPU, memory) and version.
  query: select i.*, p.resident_size, p.user_time, p.system_time, time.minutes as counter from osquery_info i, processes p, time where p.pid = i.pid;
  team: ""
  interval: 3600 # 1 hour
  observer_can_run: true
  automations_enabled: true
  discard_data: false
---
apiVersion: v1 
kind: query 
spec: 
  name: Get serial number of a laptop 
  description: Returns the serial number of a laptop, which can be useful for asset tracking.
  query: SELECT hardware_serial FROM system_info; 
  team: Workstations
  interval: 0
  observer_can_run: true
  discard_data: false
--- 
apiVersion: v1 
kind: query 
spec: 
  name: Get recently added or removed USB drives 
  description: Report event publisher health and track event counters. 
  query: |-
    SELECT action, DATETIME(time, 'unixepoch') AS datetime, vendor, mounts.path 
    FROM disk_events 
    LEFT JOIN mounts 
      ON mounts.device = disk_events.device
    ;
  team: Workstations (Canary)
  interval: 86400 # 24 hours
  observer_can_run: false
  min_osquery_version: 5.4.0
  platform: darwin,windows
  automations_enabled: true
  logging: differential
  discard_data: true
```

Continued edits and applications to this file will update the queries.

If you want to change the name of a query, you must first create a new query with the new name and then delete the query with the old name via the UI or API.

## Labels

The following file describes the labels which hosts should be automatically grouped into. The label resource should include the actual SQL query so that the label is self-contained:

```yaml
apiVersion: v1
kind: label
spec:
  name: slack_not_running
  query: >
    SELECT * FROM system_info
    WHERE NOT EXISTS (
      SELECT *
      FROM processes
      WHERE name LIKE "%Slack%"
    );
```

Labels can also be manually managed. When defining a manual label, reference hosts
by hostname:

```yaml
apiVersion: v1
kind: label
spec:
  name: Manually Managed Example
  label_membership_type: manual
  hosts:
    - hostname1
    - hostname2
    - hostname3
```

## Enroll secrets

The following file shows how to configure enroll secrets. Enroll secrets are valid until you delete them.

```yaml
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
    - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
    - secret: YBh0n4pvRplKyWiowv9bf3zp6BBOJ13O
```

Osquery provides the enroll secret only during the enrollment process. Once a host is enrolled, the node key it receives remains valid for authentication independent from the enroll secret.

Currently enrolled hosts do not necessarily need enroll secrets updated, as the existing enrollment will continue to be valid as long as the host is not deleted from Fleet and the osquery store on the host remains valid. Any newly enrolling hosts must have the new secret.

Deploying a new enroll secret cannot be done centrally from Fleet.

> Enroll secrets must be alphanumeric and should not contain special characters. 

### Multiple enroll secrets

Fleet allows the abiility to maintain multiple enroll secrets. Some organizations have internal goals  around rotating secrets. Having multiple secrets allows some of them to work at the same time the rotation is happening.
Another reason you might want to use multiple enroll secrets is to use a certain [team enroll secret](#team-enroll-secrets) to auto-enroll hosts into a specific [team](https://fleetdm.com/docs/using-fleet/teams) (Fleet Premium).

### Rotating enroll secrets

1. In Fleet, head to **Hosts > Manage enroll secret** and add a new secret.
2. Create a fleetd agent with the new enroll secret and install it on hosts.
3. Delete the old enroll secret.

## Teams

**Applies only to Fleet Premium**.

The `team` YAML file controls a team in Fleet.

You can define one or more teams in the same file with `---`.

The following example file includes one team:

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: "💻 Workstations"
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
      update_channels:
        desktop: edge
        orbit: edge
        osqueryd: edge
    features:
      enable_host_users: true
      enable_software_inventory: true
    host_expiry_settings:
      host_expiry_enabled: false
      host_expiry_window: 0
    integrations:
      conditional_access_enabled: null
      google_calendar:
        enable_calendar_events: false
        webhook_url: ""
    mdm:
      android_settings:
        certificates: null
        custom_settings: null
      enable_disk_encryption: true
      ios_updates:
        deadline: null
        minimum_version: null
        update_new_hosts: null
      ipados_updates:
        deadline: null
        minimum_version: null
        update_new_hosts: null
      macos_settings:
        custom_settings: []
      macos_setup:
        bootstrap_package: ""
        enable_end_user_authentication: true
        enable_release_device_manually: false
        macos_setup_assistant: ""
        manual_agent_install: false
        require_all_software_macos: false
        script: ""
        software: []
      macos_updates:
        deadline: ""
        minimum_version: ""
        update_new_hosts: null
      windows_require_bitlocker_pin: null
      windows_settings:
        custom_settings: null
      windows_updates:
        deadline_days: null
        grace_period_days: null
    scripts:
    - /home/runner/work/fleet/fleet/it-and-security/lib/macos/scripts/uninstall-fleetd-macos.sh
    - /home/runner/work/fleet/fleet/it-and-security/lib/windows/scripts/uninstall-fleetd-windows.ps1
    - /home/runner/work/fleet/fleet/it-and-security/lib/linux/scripts/uninstall-fleetd-linux.sh
    - /home/runner/work/fleet/fleet/it-and-security/lib/linux/scripts/install-fleet-desktop-required-extension.sh
    secrets:
    - created_at: "2026-02-08T05:25:21Z"
      secret: tTavYeEwmUYzdnRlPICwVcFtPszkIvkf
      team_id: 310
    software:
      app_store_apps: null
      fleet_maintained_apps:
      - categories: null
        icon:
          path: ""
        install_script:
          path: ""
        labels_exclude_any: null
        labels_include_any: null
        post_install_script:
          path: ""
        pre_install_query:
          path: ""
        self_service: true
        setup_experience: null
        slug: santa/darwin
        uninstall_script:
          path: ""
      - categories: null
        icon:
          path: ""
        install_script:
          path: ""
        labels_exclude_any: null
        labels_include_any: null
        post_install_script:
          path: ""
        pre_install_query:
          path: ""
        self_service: true
        setup_experience: null
        slug: vnc-viewer/darwin
        uninstall_script:
          path: ""
      - categories: null
        icon:
          path: ""
        install_script:
          path: ""
        labels_exclude_any: null
        labels_include_any: null
        post_install_script:
          path: ""
        pre_install_query:
          path: ""
        self_service: true
        setup_experience: null
        slug: beyond-compare/darwin
        uninstall_script:
          path: ""
      - categories: null
        icon:
          path: ""
        install_script:
          path: ""
        labels_exclude_any: null
        labels_include_any: null
        post_install_script:
          path: ""
        pre_install_query:
          path: ""
        self_service: true
        setup_experience: null
        slug: iterm2/darwin
        uninstall_script:
          path: ""
      packages: null
    webhook_settings:
      failing_policies_webhook: null
      host_status_webhook:
        days_count: 0
        destination_url: ""
        enable_host_status_webhook: false
        host_percentage: 0
```

During an import with `fleetctl apply`, if you have an empty `macos.custom_settings`, `windows.custome_settings`, or `android.custom_settings`, all OS settings (configuration profiles) will be removed. To import other options without touching configuration profiles, remove `macos.custom_settings`, `windows.custome_settings`, and `android.custom_settings` from your YAML.

## Organization settings

The `config` YAML file controls Fleet's organization settings and MDM features for hosts assigned to "No team."

The following example file shows the default organization settings:

```yaml
apiVersion: v1
kind: config
spec:
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
    overrides: {}
    command_line_flags: {}
  features:
    enable_host_users: true
    enable_software_inventory: true
  fleet_desktop:
    transparency_url: https://fleetdm.com/transparency
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  integrations:
    jira: null
    zendesk: null
  org_info:
    org_logo_url: ""
    org_logo_url_light_background: ""
    contact_url: ""
    org_name: Fleet
  server_settings:
    deferred_save_host: false
    enable_analytics: true
    live_query_disabled: false
    query_reports_disabled: false
    scripts_disabled: false
    server_url: ""
  smtp_settings:
    authentication_method: authmethod_plain
    authentication_type: authtype_username_password
    domain: ""
    enable_smtp: false
    enable_ssl_tls: true
    enable_start_tls: true
    password: ""
    port: 587
    sender_address: ""
    server: ""
    user_name: ""
    verify_ssl_certs: true
  sso_settings:
    enable_jit_provisioning: false
    enable_sso: false
    enable_sso_idp_login: false
    entity_id: ""
    idp_image_url: ""
    idp_name: ""
    issuer_uri: ""
    metadata: ""
    metadata_url: ""
  vulnerabilities:
    databases_path: "/tmp/vulndbs"
    periodicity: 1h
    cpe_database_url: ""
    cpe_translations_url: ""
    cve_feed_prefix_url: ""
    disable_schedule: false
    disable_data_sync: false
    recent_vulnerability_max_age: 30d
    disable_win_os_vulnerabilities: false
  webhook_settings:
    failing_policies_webhook:
      destination_url: ""
      enable_failing_policies_webhook: false
      host_batch_size: 0
      policy_ids: null
    host_status_webhook:
      days_count: 0
      destination_url: ""
      enable_host_status_webhook: false
      host_percentage: 0
    interval: 24h
    vulnerabilities_webhook:
      destination_url: ""
      enable_vulnerabilities_webhook: false
      host_batch_size: 0
  mdm:
    apple_bm_default_team: ""
    windows_enabled_and_configured: false
    macos_updates:
      minimum_version: ""
      deadline: ""
    macos_settings:
      custom_settings:
        - path: '/path/to/profile1.mobileconfig'
        - path: '/path/to/profile2.mobileconfig'
        - path: '/path/to/profile3.mobileconfig'
      enable_disk_encryption: true
    windows_settings:
      custom_settings:
        - path: '/path/to/profile4.xml'
        - path: '/path/to/profile5.xml'
```

### Settings

For possible options, see the parameter for the parameters of the [Modify configuration API endpoint](../REST%20API/rest-api.md#modify-configuration).

Each section's key must be one level below the `spec` key, indented with spaces (not `<tab>` characters) as required by the YAML format.

For example, when adding the `host_expiry_settings.host_expiry_enabled` setting, you'd specify the `host_expiry_settings` section one level below the `spec` key:

```yaml
apiVersion: v1
kind: config
spec:
  host_expiry_settings:
    host_expiry_enabled: true
```
