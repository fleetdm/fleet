# Configuration for contributors

- [Integrations](#integrations)
- [SMTP-settings](#smtp-settings)
- [Environment variables](#environment-variables)

This document includes configuration files and settings that are helpful when developing or contributing to Fleet.

Unlike the [configuration files documentation](https://fleetdm.com/docs/using-fleet/configuration-files), the files and settings in this document are not recommended for production use. Each setting includes the best practice for being successful in production.
## Integrations

Integration settings in Fleet can be configured using the `integrations` section of the `config` YAML file. To see all settings in this file, check out the [configuration files documentation](https://fleetdm.com/docs/using-fleet/configuration-files#organization-settings).

> **Warning:** Be careful not to store your integration credentials in source control. The best practice is to configure integrations [via the Fleet UI](https://fleetdm.com/docs/using-fleet/automations).

### Jira

Jira integrations are configured under the `integrations.jira` field, which is an array of dictionaries.

#### integrations.jira[].url

This is the URL of the Jira server to use, including the scheme (e.g. "https://").

- Required setting (string)
- Default value: none
- Config file format:
  ```yaml
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
  ```

#### integrations.jira[].username

Use this username to authenticate API requests with the Jira server.

- Required setting (string)
- Default value: none
- Config file format:
  ```yaml
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
  ```

#### integrations.jira[].api_token

Use this API token to authenticate API requests with the Jira server.

- Required setting (string)
- Default value: none
- Config file format:
  ```yaml
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
  ```

#### integrations.jira[].project_key

Use this Jira project key to create tickets.

- Required setting (string)
- Default value: none
- Config file format:
  ```yaml
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
  ```

#### integrations.jira[].enable_failing_policies

Whether the integration is configured to create Jira tickets for failing policies.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
        enable_failing_policies: true
  ```

#### integrations.jira[].enable_software_vulnerabilities

Whether the integration is configured to create Jira tickets for recent software vulnerabilities.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
        enable_software_vulnerabilities: true
  ```

### Zendesk

Zendesk integrations are configured under the `integrations.zendesk` field, which is an array of dictionaries.

#### integrations.zendesk[].url

This is the URL of the Zendesk server to use, including the scheme (e.g. "https://").

- Required setting (string)
- Default value: none
- Config file format:
  ```yaml
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
  ```

#### integrations.zendesk[].email

Use this email address to authenticate API requests with the Zendesk server.

- Required setting (string)
- Default value: none
- Config file format:
  ```yaml
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
  ```

#### integrations.zendesk[].api_token

Use this API token to authenticate API requests with the Zendesk server.

- Required setting (string)
- Default value: none
- Config file format:
  ```yaml
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
  ```

#### integrations.zendesk[].group_id

Use this group ID to create tickets.

- Required setting (integer)
- Default value: none
- Config file format:
  ```yaml
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
  ```

#### integrations.zendesk[].enable_failing_policies

Whether the integration is configured to create Zendesk tickets for failing policies.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
        enable_failing_policies: true
  ```

#### integrations.zendesk[].enable_software_vulnerabilities

Whether the integration is configured to create Zendesk tickets for recent software vulnerabilities.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
        enable_software_vulnerabilities: true
  ```

## SMTP settings

SMTP settings in Fleet can be configured using the `smtp_settings` section of the `config` YAML file. To see all settings in this file, check out the [configuration files documentation](https://fleetdm.com/docs/using-fleet/configuration-files#organization-settings).

> **Warning:** Be careful not to store your SMTP credentials in source control. The best practice is to configure SMTP [via the Fleet UI](https://fleetdm.com/docs/deploying/configuration#configuring-single-sign-on-sso).

### smtp_settings.authentication_method

Use this authentication method when the authentication type is `authtype_username_password`.

- Optional setting (string)
- Default value: `authmethod_plain`
- Possible values:
  - `authmethod_cram_md5`
  - `authmethod_login`
  - `authmethod_plain`
- Config file format:
  ```yaml
  smtp_settings:
    authentication_method: authmethod_cram_md5
  ```

### smtp_settings.authentication_type

This is the type of authentication for the configured SMTP server.

- Optional setting (string)
- Default value: `authtype_username_password`
- Possible values:
  - `authtype_none` - use this if your SMTP server is open
  - `authtype_username_password` - use this if your SMTP server requires authentication with a username and password
- Config file format:
  ```yaml
  smtp_settings:
    authentication_type: authtype_none
  ```

### smtp_settings.enable_smtp

Whether SMTP support is enabled or not to send emails from Fleet.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  smtp_settings:
    enable_smtp: true
  ```

### smtp_settings.enable_ssl_tls

Whether to enable SSL/TLS for the SMTP connection.

- Optional setting (boolean)
- Default value: `true`
- Config file format:
  ```yaml
  smtp_settings:
    enable_ssl_tls: false
  ```

### smtp_settings.enable_start_tls

Whether to detect if TLS is used by the SMTP server and start using it if so.

- Optional setting (boolean)
- Default value: `true`
- Config file format:
  ```yaml
  smtp_settings:
    enable_start_tls: false
  ```

### smtp_settings.password

Use this password for SMTP authentication when the `authentication_type` is set to `authtype_username_password`.

- Optional setting (string)
- Default value: ""
- Config file format:
  ```yaml
  smtp_settings:
    password: supersekretsmtppass
  ```

### smtp_settings.port

Use this port to connect to the SMTP server.

- Optional setting (integer)
- Default value: `587` (the standard SMTP port)
- Config file format:
  ```yaml
  smtp_settings:
    port: 5870
  ```

### smtp_settings.sender_address

Use this email address as the sender for emails sent by Fleet.

- Optional setting (string)
- Default value: ""
- Config file format:
  ```yaml
  smtp_settings:
    sender_address: fleet@example.org
  ```

### smtp_settings.server

This is the server hostname for SMTP.

- Optional setting, required to properly configue SMTP (string)
- Default value: ""
- Config file format:
  ```yaml
  smtp_settings:
    server: mail.example.org
  ```

### smtp_settings.user_name

Use this username for SMTP authentication when the `authentication_type` is set to `authtype_username_password`.

- Optional setting (string)
- Default value: ""
- Config file format:
  ```yaml
  smtp_settings:
    user_name: test_user
  ```

### smtp_settings.verify_ssl_certs

Whether the SMTP server's SSL certificates should be verified. This can be turned off if self-signed certificates are used by the SMTP server.

- Optional setting (boolean)
- Default value: `true`
- Config file format:
  ```yaml
  smtp_settings:
    verify_ssl_certs: false
  ```

## Environment variables

### FLEET_ENABLE_POST_CLIENT_DEBUG_ERRORS

Use this environment variable to allow `fleetd` to report errors to the server using the [endpoint to report an agent error](./API-for-contributors.md#report-an-agent-error).

<meta name="pageOrderInSection" value="1100">
<meta name="description" value="Learn about the configuration files and settings that are helpful when developing or contributing to Fleet.">
