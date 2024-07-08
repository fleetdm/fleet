# Configuration for contributors

- [Integrations](#integrations)
- [SMTP-settings](#smtp-settings)
- [Server configuration](#server-configuration)
- [Environment variables](#environment-variables)

This document includes Fleet server configuration settings that are helpful when developing or contributing to Fleet.

Unlike the [fleetctl apply format](https://github.com/fleetdm/fleet/tree/main/docs/Contributing/fleetctl-apply.md), the files and settings in this document are not recommended for production use. Each setting includes the best practice for being successful in production.

## Server configuration

##### s3_software_installers_disable_ssl

AWS S3 Disable SSL. Useful for local testing.

- Default value: false
- Environment variable: `FLEET_S3_SOFTWARE_INSTALLERS_DISABLE_SSL`
- Config file format:
  ```yaml
  s3:
    software_installers_disable_ssl: false
  ```

##### s3_carves_disable_ssl

- Default value: false
- Environment variable: `FLEET_S3_CARVES_DISABLE_SSL`
- Config file format:
  ```yaml
  s3:
    carves_disable_ssl: false
  ```

<<<<<<< HEAD
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

## Server configuration

##### s3_software_installers_disable_ssl

AWS S3 Disable SSL. Useful for local testing.

- Default value: false
- Environment variable: `FLEET_S3_SOFTWARE_INSTALLERS_DISABLE_SSL`
- Config file format:
  ```yaml
  s3:
    software_installers_disable_ssl: false
  ```

##### s3_carves_disable_ssl

- Default value: false
- Environment variable: `FLEET_S3_CARVES_DISABLE_SSL`
- Config file format:
  ```yaml
  s3:
    carves_disable_ssl: false
  ```

=======
>>>>>>> 6f3d0fa3fb52ed920228404e47c3644c63929e3b
##### mdm.apple_apns_cert_bytes

The content of the Apple Push Notification service (APNs) certificate. An X.509 certificate, PEM-encoded. Typically generated via `fleetctl generate mdm-apple`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_APNS_CERT_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_apns_cert_bytes: |
      -----BEGIN CERTIFICATE-----
      ... PEM-encoded content ...
      -----END CERTIFICATE-----
  ```

##### mdm.apple_apns_key_bytes

The content of the PEM-encoded private key for the Apple Push Notification service (APNs). Typically generated via `fleetctl generate mdm-apple`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_APNS_KEY_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_apns_key_bytes: |
      -----BEGIN RSA PRIVATE KEY-----
      ... PEM-encoded content ...
      -----END RSA PRIVATE KEY-----
  ```

##### mdm.apple_scep_cert_bytes

The content of the Simple Certificate Enrollment Protocol (SCEP) certificate. An X.509 certificate, PEM-encoded. Typically generated via `fleetctl generate mdm-apple`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_SCEP_CERT_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_scep_cert_bytes: |
      -----BEGIN CERTIFICATE-----
      ... PEM-encoded content ...
      -----END CERTIFICATE-----
  ```

The SCEP certificate/key pair generated by Fleet expires every 10 years. It's recommended to never change these unless they were compromised.

If your certificate/key pair was compromised and you change the pair, the disk encryption keys will no longer be viewable on all macOS hosts' **Host details** page until you turn disk encryption off and back on and the keys are [reset by the end user](https://fleetdm.com/docs/using-fleet/MDM-migration-guide#how-to-turn-on-disk-encryption).

##### mdm.apple_scep_key_bytes

The content of the PEM-encoded private key for the Simple Certificate Enrollment Protocol (SCEP). Typically generated via `fleetctl generate mdm-apple`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_SCEP_KEY_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_scep_key_bytes: |
      -----BEGIN RSA PRIVATE KEY-----
      ... PEM-encoded content ...
      -----END RSA PRIVATE KEY-----
  ```

##### mdm.apple_scep_challenge

An alphanumeric secret for the Simple Certificate Enrollment Protocol (SCEP). Define a unique, static secret 32 characters in length and only include alphanumeric characters.

> SCEP is commonly applied to a number of certificate use cases. Notably, Mobile Device Management (MDM) systems like Microsoft Intune and Apple MDM use SCEP for PKI certificate enrollment.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_SCEP_CHALLENGE`
- Config file format:
  ```yaml
  mdm:
    apple_scep_challenge: scepchallenge
  ```

##### mdm.apple_bm_server_token_bytes

This is the content of the Apple Business Manager encrypted server token downloaded from Apple Business Manager.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_BM_SERVER_TOKEN_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_bm_server_token_bytes: |
      Content-Type: application/pkcs7-mime; name="smime.p7m"; smime-type=enveloped-data
      Content-Transfer-Encoding: base64
      ... rest of content ...
  ```

##### mdm.apple_bm_cert_bytes

This is the content of the Apple Business Manager certificate. The certificate is a PEM-encoded X.509 certificate that's typically generated via `fleetctl generate mdm-apple-bm`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_BM_CERT_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_bm_cert_bytes: |
      -----BEGIN CERTIFICATE-----
      ... PEM-encoded content ...
      -----END CERTIFICATE-----
  ```

##### mdm.apple_bm_key_bytes

This is the content of the PEM-encoded private key for the Apple Business Manager. It's typically generated via `fleetctl generate mdm-apple-bm`.

- Default value: ""
- Environment variable: `FLEET_MDM_APPLE_BM_KEY_BYTES`
- Config file format:
  ```yaml
  mdm:
    apple_bm_key_bytes: |
      -----BEGIN RSA PRIVATE KEY-----
      ... PEM-encoded content ...
      -----END RSA PRIVATE KEY-----
  ```

## Environment variables

### FLEET_ENABLE_POST_CLIENT_DEBUG_ERRORS

Use this environment variable to allow `fleetd` to report errors to the server using the [endpoint to report an agent error](./API-for-contributors.md#report-an-agent-error).

<meta name="pageOrderInSection" value="1100">
<meta name="description" value="Learn about the configuration files and settings that are helpful when developing or contributing to Fleet.">
