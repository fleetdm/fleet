# Advanced Configuration

TODO(noah): some context to explain those non-recommended settings.

## Integrations

### Jira

Jira integrations are configured under the `integrations.jira` field, which is an array of dictionaries.

#### integrations.jira[].url

The URL of the Jira server to use, including the scheme (e.g. "https://").

- Required setting (string).
- Default value: none.
- Config file format:
  ```
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
  ```

#### integrations.jira[].username

> **Warning:** Be careful not to store your Jira credentials in source control. It is recommended to configure integrations [via the Fleet UI](../Using-Fleet/Automations.md).

The username to use to authenticate with the Jira server for API requests.

- Required setting (string).
- Default value: none.
- Config file format:
  ```
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
  ```

#### integrations.jira[].api_token

> **Warning:** Be careful not to store your Jira credentials in source control. It is recommended to configure integrations [via the Fleet UI](../Using-Fleet/Automations.md).

The API token to use to authenticate with the Jira server for API requests.

- Required setting (string).
- Default value: none.
- Config file format:
  ```
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
  ```

#### integrations.jira[].project_key

The Jira project key to use to create tickets.

- Required setting (string).
- Default value: none.
- Config file format:
  ```
  integrations:
    jira:
      - url: "https://example.atlassian.net"
        username: "user1"
        api_token: "secret"
        project_key: "PJ1"
  ```

#### integrations.jira[].enable_failing_policies

Whether the integration is configured to create Jira tickets for failing policies.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
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

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
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

The URL of the Zendesk server to use, including the scheme (e.g. "https://").

- Required setting (string).
- Default value: none.
- Config file format:
  ```
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
  ```

#### integrations.zendesk[].email

> **Warning:** Be careful not to store your Zendesk credentials in source control. It is recommended to configure integrations [via the Fleet UI](../Using-Fleet/Automations.md).

The email address to use to authenticate with the Zendesk server for API requests.

- Required setting (string).
- Default value: none.
- Config file format:
  ```
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
  ```

#### integrations.zendesk[].api_token

> **Warning:** Be careful not to store your Zendesk credentials in source control. It is recommended to configure integrations [via the Fleet UI](../Using-Fleet/Automations.md).

The API token to use to authenticate with the Zendesk server for API requests.

- Required setting (string).
- Default value: none.
- Config file format:
  ```
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
  ```

#### integrations.zendesk[].group_id

The group ID to use to create tickets.

- Required setting (integer).
- Default value: none.
- Config file format:
  ```
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
  ```

#### integrations.zendesk[].enable_failing_policies

Whether the integration is configured to create Zendesk tickets for failing policies.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
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

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  integrations:
    zendesk:
      - url: "https://example.zendesk.com"
        email: "user1@example.com"
        api_token: "secret"
        group_id: 1234
        enable_software_vulnerabilities: true
  ```

## SMTP settings

### smtp_settings.authentication_method

The authentication method to use when the authentication type is `authtype_username_password`.

- Optional setting (string).
- Default value: `authmethod_plain`.
- Possible values:
  - `authmethod_cram_md5`
  - `authmethod_login`
  - `authmethod_plain`
- Config file format:
  ```
  smtp_settings:
    authentication_method: authmethod_cram_md5
  ```

### smtp_settings.authentication_type

The type of authentication for the configured SMTP server.

- Optional setting (string).
- Default value: `authtype_username_password`.
- Possible values:
  - `authtype_none` - use this if your SMTP server is open
  - `authtype_username_password` - use this if your SMTP server requires authentication with a username and password
- Config file format:
  ```
  smtp_settings:
    authentication_type: authtype_none
  ```

### smtp_settings.enable_smtp

Whether SMTP support is enabled or not to send emails from Fleet.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  smtp_settings:
    enable_smtp: true
  ```

### smtp_settings.enable_ssl_tls

Whether to enable SSL/TLS for the SMTP connection.

- Optional setting (boolean).
- Default value: `true`.
- Config file format:
  ```
  smtp_settings:
    enable_ssl_tls: false
  ```

### smtp_settings.enable_start_tls

Whether to detect if TLS is used by the SMTP server and start using it if so.

- Optional setting (boolean).
- Default value: `true`.
- Config file format:
  ```
  smtp_settings:
    enable_start_tls: false
  ```

### smtp_settings.password

> **Warning:** Be careful not to store your SMTP credentials in source control. It is recommended to set the password through the web UI or `fleetctl` and then remove the line from the checked in version. Fleet will leave the password as-is if the field is missing from the applied configuration.

The password to use for the SMTP authentication, when `authentication_type` is set to `authtype_username_password`.

- Optional setting (string).
- Default value: "".
- Config file format:
  ```
  smtp_settings:
    password: supersekretsmtppass
  ```

### smtp_settings.port

The port to use to connect to the SMTP server.

- Optional setting (integer).
- Default value: `587` (the standard SMTP port).
- Config file format:
  ```
  smtp_settings:
    port: 5870
  ```

### smtp_settings.sender_address

The email address to use as sender for emails sent by Fleet.

- Optional setting (string).
- Default value: "".
- Config file format:
  ```
  smtp_settings:
    sender_address: fleet@example.org
  ```

### smtp_settings.server

The server hostname for SMTP.

- Optional setting, required to properly configue SMTP (string).
- Default value: "".
- Config file format:
  ```
  smtp_settings:
    server: mail.example.org
  ```

### smtp_settings.user_name

> **Warning:** Be careful not to store your SMTP credentials in source control. It is recommended to set the password through the web UI or `fleetctl` and then remove the line from the checked in version. Fleet will leave the password as-is if the field is missing from the applied configuration.

The username to use for the SMTP authentication, when `authentication_type` is set to `authtype_username_password`.

- Optional setting (string).
- Default value: "".
- Config file format:
  ```
  smtp_settings:
    user_name: test_user
  ```

### smtp_settings.verify_ssl_certs

Whether the SMTP server's SSL certificates should be verified. Can be turned off if self-signed certificates are used by the SMTP server.

- Optional setting (boolean).
- Default value: `true`.
- Config file format:
  ```
  smtp_settings:
    verify_ssl_certs: false
  ```

<meta name="pageOrderInSection" value="1100">
