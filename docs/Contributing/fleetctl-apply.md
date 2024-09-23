# fleetctl apply

The `fleectl apply` command and YAML interface is used for one-off imports and backwards compatibility GitOps.

To use Fleet's best practice GitOps, check out the GitOps docs [here](https://fleetdm.com/docs/using-fleet/gitops).

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

Rotating enroll secrets follows this process:

1. Add a new secret.
2. Transition existing clients to the new secret. Note that existing clients may not need to be
   updated, as the enroll secret is not used by already enrolled clients.
3. Remove the old secret.

To do this with `fleetctl` (assuming the existing secret is `oldsecret` and the new secret is `newsecret`):

Begin by retrieving the existing secret configuration:

```sh
$ fleetctl get enroll_secret
---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - created_at: "2021-11-17T00:39:50Z"
    secret: oldsecret
```

Apply the new configuration with both secrets:

```sh
$ echo '
---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - created_at: "2021-11-17T00:39:50Z"
    secret: oldsecret
  - secret: newsecret
' > secrets.yml
$ fleetctl apply -f secrets.yml
```

Now transition clients to using only the new secret. When the transition is completed, remove the
old secret:

```sh
$ echo '
---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - secret: newsecret
' > secrets.yml
$ fleetctl apply -f secrets.yml
```

At this point, the old secret will no longer be accepted for new enrollments and the rotation is
complete.

A similar process may be followed for rotating team-specific enroll secrets. For teams, the secrets
are managed in the team yaml.

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
    name: Client Platform Engineering
    features:
      enable_host_users: false
      enable_software_inventory: true
      additional_queries:
        time: SELECT * FROM time
        macs: SELECT mac FROM interface_details
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
          logger_tls_endpoint: /api/v1/osquery/log
          logger_tls_period: 10
          pack_delimiter: /
      overrides: {}
      command_line_flags: {}
    secrets:
      - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
      - secret: JZ/C/Z7ucq22dt/zjx2kEuDBN0iLjqfz
    webhook_settings:
      host_status_webhook:
        days_count: 0
        destination_url: ""
        enable_host_status_webhook: false
        host_percentage: 0
    host_expiry_settings: 
      host_expiry_enabled: true 
      host_expiry_window: 14
    mdm:
      macos_updates:
        minimum_version: "12.3.1"
        deadline: "2022-01-04"
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
    scripts:
        - path/to/script1.sh
        - path/to/script2.sh
```

### Team agent options

The team agent options specify options that only apply to this team. When team-specific agent options have been specified, the agent options specified at the organization level are ignored for this team.

The documentation for this section is identical to the [Agent options](#agent-options) documentation for the organization settings, except that the YAML section where it is set must be as follows. (Note the `kind: team` key and the location of the `agent_options` key under `team` must have a `name` key to identify the team to configure.)

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Client Platform Engineering
    agent_options:
      # the team-specific options go here
```

### Team secrets

The `secrets` section provides the list of enroll secrets that will be valid for this team. When a new team is created via `fleetctl apply`, an enroll secret is automatically generated for it. If the section is missing, the existing secrets are left unmodified. Otherwise, they are replaced with this list of secrets for this team.

- Optional setting (array of dictionaries)
- Default value: none (empty)
- Config file format:
  ```yaml
  team:
    name: Client Platform Engineering
    secrets:
      - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
      - secret: JZ/C/Z7ucq22dt/zjx2kEuDBN0iLjqfz
  ```

### Modify an existing team

You can modify an existing team by applying a new team configuration file with the same `name` as an existing team. The new team configuration will completely replace the previous configuration. In order to avoid overriding existing settings, we recommend retrieving the existing configuration and modifying it.

Retrieve the team configuration and output to a YAML file:

```sh
% fleetctl get teams --name Workstations --yaml > workstation_config.yml
```
After updating the generated YAML, apply the changes:

```sh
% fleetctl apply -f workstation_config.yml
```

Depending on your Fleet version, you may see `unsupported key` errors for the following keys when applying the new team configuration:

```text
id
user_count
host_count
integrations
webhook_settings
description
agent_options
created_at
user_count
host_count
integrations
webhook_settings
```

You can bypass these errors by removing the key from your YAML or adding the `--force` flag. This flag will apply the changes without validation and should be used with caution.

### Mobile device management (MDM) settings for teams

The `mdm` section of this configuration YAML lets you control MDM settings for each team in Fleet.

To specify Team MDM configuration, as opposed to [Organization-wide MDM configuration](#mobile-device-management-mdm-settings), follow the below YAML format. Note the `kind: team` field, as well as the  `name` and `mdm` fields under `team`.

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Client Platform Engineering
    mdm:
      # the team-specific mdm options go here
```

### Team scripts

List of saved scripts that can be run on hosts that are part of the team.

- Default value: none
- Config file format:

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Client Platform Engineering
    scripts:
      - path/to/script1.sh
      - path/to/script2.sh
```

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
