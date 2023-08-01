# Fleet configuration

- [Get certificate](#get-certificate)
- [Get configuration](#get-configuration)
- [Modify configuration](#modify-configuration)
- [Get global enroll secrets](#get-global-enroll-secrets)
- [Modify global enroll secrets](#modify-global-enroll-secrets)
- [Get enroll secrets for a team](#get-enroll-secrets-for-a-team)
- [Modify enroll secrets for a team](#modify-enroll-secrets-for-a-team)
- [Create invite](#create-invite)
- [List invites](#list-invites)
- [Delete invite](#delete-invite)
- [Verify invite](#verify-invite)
- [Update invite](#update-invite)
- [Version](#version)

The Fleet server exposes a handful of API endpoints that handle the configuration of Fleet as well as endpoints that manage invitation and enroll secret operations. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

## Get certificate

Returns the Fleet certificate.

`GET /api/v1/fleet/config/certificate`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/config/certificate`

##### Default response

`Status: 200`

```json
{
  "certificate_chain": <certificate_chain>
}
```

## Get configuration

Returns all information about the Fleet's configuration.

> NOTE: The `agent_options`, `sso_settings` and `smtp_settings` fields are only returned to Global Admin users.

`GET /api/v1/fleet/config`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/config`

##### Default response

`Status: 200`

```json
{
  "org_info": {
    "org_name": "fleet",
    "org_logo_url": "",
    "contact_url": "https://fleetdm.com/company/contact"
  },
  "server_settings": {
    "server_url": "https://localhost:8080",
    "live_query_disabled": false,
    "enable_analytics": true
  },
  "smtp_settings": {
    "enable_smtp": false,
    "configured": false,
    "sender_address": "",
    "server": "",
    "port": 587,
    "authentication_type": "authtype_username_password",
    "user_name": "",
    "password": "********",
    "enable_ssl_tls": true,
    "authentication_method": "authmethod_plain",
    "domain": "",
    "verify_ssl_certs": true,
    "enable_start_tls": true
  },
  "sso_settings": {
    "entity_id": "",
    "issuer_uri": "",
    "idp_image_url": "",
    "metadata": "",
    "metadata_url": "",
    "idp_name": "",
    "enable_sso": false,
    "enable_sso_idp_login": false,
    "enable_jit_provisioning": false
  },
  "host_expiry_settings": {
    "host_expiry_enabled": false,
    "host_expiry_window": 0
  },
  "features": {
    "additional_queries": null
  },
  "mdm": {
    "apple_bm_default_team": "",
    "apple_bm_terms_expired": false,
    "enabled_and_configured": true,
    "windows_enabled_and_configured": true,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "macos_settings": {
      "custom_settings": ["path/to/profile1.mobileconfig"],
      "enable_disk_encryption": true
    },
    "end_user_authentication": {
      "entity_id": "",
      "issuer_uri": "",
      "metadata": "",
      "metadata_url": "",
      "idp_name": ""
    },
    "macos_migration": {
      "enable": false,
      "mode": "voluntary",
      "webhook_url": "https://webhook.example.com"
    },
    "macos_setup": {
      "bootstrap_package": "",
      "enable_end_user_authentication": false,
      "macos_setup_assistant": "path/to/config.json"
    }
  },
  "agent_options": {
    "spec": {
      "config": {
        "options": {
          "pack_delimiter": "/",
          "logger_tls_period": 10,
          "distributed_plugin": "tls",
          "disable_distributed": false,
          "logger_tls_endpoint": "/api/v1/osquery/log",
          "distributed_interval": 10,
          "distributed_tls_max_attempts": 3
        },
        "decorators": {
          "load": [
            "SELECT uuid AS host_uuid FROM system_info;",
            "SELECT hostname AS hostname FROM system_info;"
          ]
        }
      },
      "overrides": {},
      "command_line_flags": {}
    }
  },
  "license": {
     "tier": "free",
     "expiration": "0001-01-01T00:00:00Z"
   },
  "logging": {
      "debug": false,
      "json": false,
      "result": {
          "plugin": "firehose",
          "config": {
              "region": "us-east-1",
              "status_stream": "",
              "result_stream": "result-topic"
          }
      },
      "status": {
          "plugin": "filesystem",
          "config": {
              "status_log_file": "foo_status",
              "result_log_file": "",
              "enable_log_rotation": false,
              "enable_log_compression": false
          }
      }
  },
  "vulnerability_settings": {
    "databases_path": ""
  },
  "webhook_settings": {
    "host_status_webhook": {
      "enable_host_status_webhook": true,
      "destination_url": "https://server.com",
      "host_percentage": 5,
      "days_count": 7
    },
    "failing_policies_webhook":{
      "enable_failing_policies_webhook":true,
      "destination_url": "https://server.com",
      "policy_ids": [1, 2, 3],
      "host_batch_size": 1000
    },
    "vulnerabilities_webhook":{
      "enable_vulnerabilities_webhook":true,
      "destination_url": "https://server.com",
      "host_batch_size": 1000
    }
  },
  "integrations": {
    "jira": null
  },
  "logging": {
    "debug": false,
    "json": false,
    "result": {
        "plugin": "filesystem",
        "config": {
          "status_log_file": "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_status",
          "result_log_file": "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_result",
          "enable_log_rotation": false,
          "enable_log_compression": false
        }
      },
    "status": {
      "plugin": "filesystem",
      "config": {
        "status_log_file": "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_status",
        "result_log_file": "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_result",
        "enable_log_rotation": false,
        "enable_log_compression": false
      }
    }
  },
  "update_interval": {
    "osquery_detail": 3600000000000,
    "osquery_policy": 3600000000000
  },
  "vulnerabilities": {
    "cpe_database_url": "",
    "current_instance_checks": "auto",
    "cve_feed_prefix_url": "",
    "databases_path": "",
    "disable_data_sync": false,
    "periodicity": 3600000000000,
    "recent_vulnerability_max_age": 2592000000000000
  }
}
```

## Modify configuration

Modifies the Fleet's configuration with the supplied information.

`PATCH /api/v1/fleet/config`

#### Parameters

| Name                              | Type    | In    | Description                                                                                                                                                                            |
| ---------------------             | ------- | ----  | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| org_name                          | string  | body  | _Organization information_. The organization name.                                                                                                                                     |
| org_logo_url                      | string  | body  | _Organization information_. The URL for the organization logo.                                                                                                                         |
| server_url                        | string  | body  | _Server settings_. The Fleet server URL.                                                                                                                                               |
| live_query_disabled               | boolean | body  | _Server settings_. Whether the live query capabilities are disabled.                                                                                                                   |
| enable_smtp                       | boolean | body  | _SMTP settings_. Whether SMTP is enabled for the Fleet app.                                                                                                                            |
| sender_address                    | string  | body  | _SMTP settings_. The sender email address for the Fleet app. An invitation email is an example of the emails that may use this sender address                                          |
| server                            | string  | body  | _SMTP settings_. The SMTP server for the Fleet app.                                                                                                                                    |
| port                              | integer | body  | _SMTP settings_. The SMTP port for the Fleet app.                                                                                                                                      |
| authentication_type               | string  | body  | _SMTP settings_. The authentication type used by the SMTP server. Options include `"authtype_username_and_password"` or `"none"`                                                       |
| username_name                     | string  | body  | _SMTP settings_. The username used to authenticate requests made to the SMTP server.                                                                                                   |
| password                          | string  | body  | _SMTP settings_. The password used to authenticate requests made to the SMTP server.                                                                                                   |
| enable_ssl_tls                    | boolean | body  | _SMTP settings_. Whether or not SSL and TLS are enabled for the SMTP server.                                                                                                           |
| authentication_method             | string  | body  | _SMTP settings_. The authentication method used to make authenticate requests to SMTP server. Options include `"authmethod_plain"`, `"authmethod_cram_md5"`, and `"authmethod_login"`. |
| domain                            | string  | body  | _SMTP settings_. The domain for the SMTP server.                                                                                                                                       |
| verify_ssl_certs                  | boolean | body  | _SMTP settings_. Whether or not SSL certificates are verified by the SMTP server. Turn this off (not recommended) if you use a self-signed certificate.                                |
| enabled_start_tls                 | boolean | body  | _SMTP settings_. Detects if STARTTLS is enabled in your SMTP server and starts to use it.                                                                                              |
| enabled_sso                       | boolean | body  | _SSO settings_. Whether or not SSO is enabled for the Fleet application. If this value is true, you must also include most of the SSO settings parameters below.                       |
| entity_id                         | string  | body  | _SSO settings_. The required entity ID is a URI that you use to identify Fleet when configuring the identity provider.                                                                 |
| issuer_uri                        | string  | body  | _SSO settings_. The URI you provide here must exactly match the Entity ID field used in the identity provider configuration.                                                           |
| idp_image_url                     | string  | body  | _SSO settings_. An optional link to an image such as a logo for the identity provider.                                                                                                 |
| metadata                          | string  | body  | _SSO settings_. Metadata provided by the identity provider. Either metadata or a metadata URL must be provided.                                                                        |
| metadata_url                      | string  | body  | _SSO settings_. A URL that references the identity provider metadata. If available from the identity provider, this is the preferred means of providing metadata.                      |
| host_expiry_enabled               | boolean | body  | _Host expiry settings_. When enabled, allows automatic cleanup of hosts that have not communicated with Fleet in some number of days.                                                  |
| host_expiry_window                | integer | body  | _Host expiry settings_. If a host has not communicated with Fleet in the specified number of days, it will be removed.                                                                 |
| agent_options                     | objects | body  | The agent_options spec that is applied to all hosts. In Fleet 4.0.0 the `api/v1/fleet/spec/osquery_options` endpoints were removed.                                                    |
| transparency_url                  | string  | body  | _Fleet Desktop_. The URL used to display transparency information to users of Fleet Desktop. **Requires Fleet Premium license**                                                           |
| enable_host_status_webhook        | boolean | body  | _webhook_settings.host_status_webhook settings_. Whether or not the host status webhook is enabled.                                                                 |
| destination_url                   | string  | body  | _webhook_settings.host_status_webhook settings_. The URL to deliver the webhook request to.                                                     |
| host_percentage                   | integer | body  | _webhook_settings.host_status_webhook settings_. The minimum percentage of hosts that must fail to check in to Fleet in order to trigger the webhook request.                                                              |
| days_count                        | integer | body  | _webhook_settings.host_status_webhook settings_. The minimum number of days that the configured `host_percentage` must fail to check in to Fleet in order to trigger the webhook request.                                |
| enable_failing_policies_webhook   | boolean | body  | _webhook_settings.failing_policies_webhook settings_. Whether or not the failing policies webhook is enabled. |
| destination_url                   | string  | body  | _webhook_settings.failing_policies_webhook settings_. The URL to deliver the webhook requests to.                                                     |
| policy_ids                        | array   | body  | _webhook_settings.failing_policies_webhook settings_. List of policy IDs to enable failing policies webhook.                                                              |
| host_batch_size                   | integer | body  | _webhook_settings.failing_policies_webhook settings_. Maximum number of hosts to batch on failing policy webhook requests. The default, 0, means no batching (all hosts failing a policy are sent on one request). |
| enable_vulnerabilities_webhook    | boolean | body  | _webhook_settings.vulnerabilities_webhook settings_. Whether or not the vulnerabilities webhook is enabled. |
| destination_url                   | string  | body  | _webhook_settings.vulnerabilities_webhook settings_. The URL to deliver the webhook requests to.                                                     |
| host_batch_size                   | integer | body  | _webhook_settings.vulnerabilities_webhook settings_. Maximum number of hosts to batch on vulnerabilities webhook requests. The default, 0, means no batching (all vulnerable hosts are sent on one request). |
| enable_software_vulnerabilities   | boolean | body  | _integrations.jira[] settings_. Whether or not Jira integration is enabled for software vulnerabilities. Only one vulnerability automation can be enabled at a given time (enable_vulnerabilities_webhook and enable_software_vulnerabilities). |
| enable_failing_policies           | boolean | body  | _integrations.jira[] settings_. Whether or not Jira integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| url                               | string  | body  | _integrations.jira[] settings_. The URL of the Jira server to integrate with. |
| username                          | string  | body  | _integrations.jira[] settings_. The Jira username to use for this Jira integration. |
| api_token                         | string  | body  | _integrations.jira[] settings_. The API token of the Jira username to use for this Jira integration. |
| project_key                       | string  | body  | _integrations.jira[] settings_. The Jira project key to use for this integration. Jira tickets will be created in this project. |
| enable_software_vulnerabilities   | boolean | body  | _integrations.zendesk[] settings_. Whether or not Zendesk integration is enabled for software vulnerabilities. Only one vulnerability automation can be enabled at a given time (enable_vulnerabilities_webhook and enable_software_vulnerabilities). |
| enable_failing_policies           | boolean | body  | _integrations.zendesk[] settings_. Whether or not Zendesk integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| url                               | string  | body  | _integrations.zendesk[] settings_. The URL of the Zendesk server to integrate with. |
| email                             | string  | body  | _integrations.zendesk[] settings_. The Zendesk user email to use for this Zendesk integration. |
| api_token                         | string  | body  | _integrations.zendesk[] settings_. The Zendesk API token to use for this Zendesk integration. |
| group_id                          | integer | body  | _integrations.zendesk[] settings_. The Zendesk group id to use for this integration. Zendesk tickets will be created in this group. |
| apple_bm_default_team             | string  | body  | _mdm settings_. The default team to use with Apple Business Manager. **Requires Fleet Premium license** |
| windows_enabled_and_configured    | boolean | body  | _mdm settings_. Enables Windows MDM support. |
| minimum_version                   | string  | body  | _mdm.macos_updates settings_. Hosts that belong to no team and are enrolled into Fleet's MDM will be nudged until their macOS is at or above this version. **Requires Fleet Premium license** |
| deadline                          | string  | body  | _mdm.macos_updates settings_. Hosts that belong to no team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past. **Requires Fleet Premium license** |
| enable                          | boolean  | body  | _mdm.macos_migration settings_. Whether to enable the end user migration workflow for devices migrating from your old MDM solution. **Requires Fleet Premium license** |
| mode                          | string  | body  | _mdm.macos_migration settings_. The end user migration workflow mode for devices migrating from your old MDM solution. Options are `"voluntary"` or `"forced"`. **Requires Fleet Premium license** |
| webhook_url                          | string  | body  | _mdm.macos_migration settings_. The webhook url configured to receive requests to unenroll devices migrating from your old MDM solution. **Requires Fleet Premium license** |
| custom_settings                   | list    | body  | _mdm.macos_settings settings_. Hosts that belong to no team and are enrolled into Fleet's MDM will have those custom profiles applied. |
| enable_disk_encryption            | boolean | body  | _mdm.macos_settings settings_. Hosts that belong to no team and are enrolled into Fleet's MDM will have disk encryption enabled if set to true. **Requires Fleet Premium license** |
| enable_end_user_authentication            | boolean | body  | _mdm.macos_setup settings_. If set to true, end user authentication will be required during automatic MDM enrollment of new macOS devices. Settings for your IdP provider must also be [configured](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#end-user-authentication). **Requires Fleet Premium license** |
| additional_queries                | boolean | body  | Whether or not additional queries are enabled on hosts.                                                                                                                                |
| force                             | bool    | query | Force apply the agent options even if there are validation errors.                                                                                                 |
| dry_run                           | bool    | query | Validate the configuration and return any validation errors, but do not apply the changes.                                                                         |

#### Example

`PATCH /api/v1/fleet/config`

##### Request body

```json
{
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png"
  },
  "smtp_settings": {
    "enable_smtp": true,
    "server": "localhost",
    "port": "1025"
  }
}
```

##### Default response

`Status: 200`

```json
{
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png",
    "contact_url": "https://fleetdm.com/company/contact"
  },
  "server_settings": {
    "server_url": "https://localhost:8080",
    "live_query_disabled": false
  },
  "smtp_settings": {
    "enable_smtp": true,
    "configured": true,
    "sender_address": "",
    "server": "localhost",
    "port": 1025,
    "authentication_type": "authtype_username_none",
    "user_name": "",
    "password": "********",
    "enable_ssl_tls": true,
    "authentication_method": "authmethod_plain",
    "domain": "",
    "verify_ssl_certs": true,
    "enable_start_tls": true
  },
  "sso_settings": {
    "entity_id": "",
    "issuer_uri": "",
    "idp_image_url": "",
    "metadata": "",
    "metadata_url": "",
    "idp_name": "",
    "enable_sso": false
  },
  "host_expiry_settings": {
    "host_expiry_enabled": false,
    "host_expiry_window": 0
  },
  "features": {
    "additional_queries": null
  },
  "license": {
    "tier": "free",
    "expiration": "0001-01-01T00:00:00Z"
  },
  "mdm": {
    "apple_bm_default_team": "",
    "apple_bm_terms_expired": false,
    "apple_bm_enabled_and_configured": false,
    "enabled_and_configured": false,
    "windows_enabled_and_configured": false,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "macos_settings": {
      "custom_settings": ["path/to/profile1.mobileconfig"],
      "enable_disk_encryption": true
    },
    "end_user_authentication": {
      "entity_id": "",
      "issuer_uri": "",
      "metadata": "",
      "metadata_url": "",
      "idp_name": ""
    },
    "macos_migration": {
      "enable": false,
      "mode": "voluntary",
      "webhook_url": "https://webhook.example.com"
    },
    "macos_setup": {
      "bootstrap_package": "",
      "enable_end_user_authentication": false,
      "macos_setup_assistant": "path/to/config.json"
    }
  },
  "agent_options": {
    "config": {
      "options": {
        "pack_delimiter": "/",
        "logger_tls_period": 10,
        "distributed_plugin": "tls",
        "disable_distributed": false,
        "logger_tls_endpoint": "/api/v1/osquery/log",
        "distributed_interval": 10,
        "distributed_tls_max_attempts": 3
      },
      "decorators": {
        "load": [
          "SELECT uuid AS host_uuid FROM system_info;",
          "SELECT hostname AS hostname FROM system_info;"
        ]
      }
    },
    "overrides": {},
    "command_line_flags": {}
  },
  "vulnerability_settings": {
    "databases_path": ""
  },
  "webhook_settings": {
    "host_status_webhook": {
      "enable_host_status_webhook": true,
      "destination_url": "https://server.com",
      "host_percentage": 5,
      "days_count": 7
    },
    "failing_policies_webhook":{
      "enable_failing_policies_webhook":true,
      "destination_url": "https://server.com",
      "policy_ids": [1, 2, 3],
      "host_batch_size": 1000
    },
    "vulnerabilities_webhook":{
      "enable_vulnerabilities_webhook":true,
      "destination_url": "https://server.com",
      "host_batch_size": 1000
    }
  },
  "integrations": {
    "jira": [
      {
        "url": "https://jiraserver.com",
        "username": "some_user",
        "password": "sec4et!",
        "project_key": "jira_project",
        "enable_software_vulnerabilities": false
      }
    ]
  },
  "logging": {
      "debug": false,
      "json": false,
      "result": {
          "plugin": "firehose",
          "config": {
              "region": "us-east-1",
              "status_stream": "",
              "result_stream": "result-topic"
          }
      },
      "status": {
          "plugin": "filesystem",
          "config": {
              "status_log_file": "foo_status",
              "result_log_file": "",
              "enable_log_rotation": false,
              "enable_log_compression": false
          }
      }
  }
}
```

## Get global enroll secrets

Returns the valid global enroll secrets.

`GET /api/v1/fleet/spec/enroll_secret`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/spec/enroll_secret`

##### Default response

`Status: 200`

```json
{
    "spec": {
        "secrets": [
            {
                "secret": "vhPzPOnCMOMoqSrLxKxzSADyqncayacB",
                "created_at": "2021-11-12T20:24:57Z"
            },
            {
                "secret": "jZpexWGiXmXaFAKdrdttFHdJBqEnqlVF",
                "created_at": "2021-11-12T20:24:57Z"
            }
        ]
    }
}
```

## Modify global enroll secrets

Replaces all existing global enroll secrets.

`POST /api/v1/fleet/spec/enroll_secret`

#### Parameters

| Name      | Type    | In   | Description                                                        |
| --------- | ------- | ---- | ------------------------------------------------------------------ |
| spec      | object  | body | **Required**. Attribute "secrets" must be a list of enroll secrets |

#### Example

Replace all global enroll secrets with a new enroll secret.

`POST /api/v1/fleet/spec/enroll_secret`

##### Request body

```json
{
    "spec": {
        "secrets": [
            {
                "secret": "KuSkYFsHBQVlaFtqOLwoUIWniHhpvEhP"
            }
        ]
    }
}
```

##### Default response

`Status: 200`

```json
{}
```

#### Example

Delete all global enroll secrets.

`POST /api/v1/fleet/spec/enroll_secret`

##### Request body

```json
{
    "spec": {
        "secrets": []
    }
}
```

##### Default response

`Status: 200`

```json
{}
```

## Get enroll secrets for a team

Returns the valid team enroll secrets.

`GET /api/v1/fleet/teams/{id}/secrets`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/teams/1/secrets`

##### Default response

`Status: 200`

```json
{
  "secrets": [
    {
      "created_at": "2021-06-16T22:05:49Z",
      "secret": "aFtH2Nq09hrvi73ErlWNQfa7M53D3rPR",
      "team_id": 1
    }
  ]
}
```


## Modify enroll secrets for a team

Replaces all existing team enroll secrets.

`PATCH /api/v1/fleet/teams/{id}/secrets`

#### Parameters

| Name      | Type    | In   | Description                            |
| --------- | ------- | ---- | -------------------------------------- |
| id        | integer | path | **Required**. The team's id.           |
| secrets   | array   | body | **Required**. A list of enroll secrets |

#### Example

Replace all of a team's existing enroll secrets with a new enroll secret

`PATCH /api/v1/fleet/teams/2/secrets`

##### Request body

```json
{
  "secrets": [
    {
      "secret": "n07v32y53c237734m3n201153c237"
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "secrets": [
    {
      "secret": "n07v32y53c237734m3n201153c237",
      "created_at": "0001-01-01T00:00:00Z"
    }
  ]
}
```

#### Example

Delete all of a team's existing enroll secrets

`PATCH /api/v1/fleet/teams/2/secrets`

##### Request body

```json
{
  "secrets": []
}
```

##### Default response

`Status: 200`

```json
{
  "secrets": null
}
```

## Create invite

`POST /api/v1/fleet/invites`

#### Parameters

| Name        | Type    | In   | Description                                                                                                                                           |
| ----------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| global_role | string  | body | Role the user will be granted. Either a global role is needed, or a team role.                                                                        |
| email       | string  | body | **Required.** The email of the invited user. This email will receive the invitation link.                                                             |
| name        | string  | body | **Required.** The name of the invited user.                                                                                                           |
| sso_enabled | boolean | body | **Required.** Whether or not SSO will be enabled for the invited user.                                                                                |
| teams       | list    | body | _Available in Fleet Premium_ A list of the teams the user is a member of. Each item includes the team's ID and the user's role in the specified team. |

#### Example

##### Request body

```json
{
  "email": "john_appleseed@example.com",
  "name": "John",
  "sso_enabled": false,
  "global_role": null,
  "teams": [
    {
      "id": 2,
      "role": "observer"
    },
    {
      "id": 3,
      "role": "maintainer"
    }
  ]
}
```

`POST /api/v1/fleet/invites`

##### Default response

`Status: 200`

```json
{
  "invite": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 3,
    "invited_by": 1,
    "email": "john_appleseed@example.com",
    "name": "John",
    "sso_enabled": false,
    "teams": [
      {
        "id": 10,
        "created_at": "0001-01-01T00:00:00Z",
        "name": "Apples",
        "description": "",
        "agent_options": null,
        "user_count": 0,
        "host_count": 0,
        "role": "observer"
      },
      {
        "id": 14,
        "created_at": "0001-01-01T00:00:00Z",
        "name": "Best of the Best Engineering",
        "description": "",
        "agent_options": null,
        "user_count": 0,
        "host_count": 0,
        "role": "maintainer"
      }
    ]
  }
}
```

## List invites

Returns a list of the active invitations in Fleet.

`GET /api/v1/fleet/invites`

#### Parameters

| Name            | Type   | In    | Description                                                                                                                   |
| --------------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| order_key       | string | query | What to order results by. Can be any column in the invites table.                                                             |
| order_direction | string | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| query           | string | query | Search query keywords. Searchable fields include `name` and `email`.                                                          |

#### Example

`GET /api/v1/fleet/invites`

##### Default response

`Status: 200`

```json
{
  "invites": [
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 3,
      "email": "john_appleseed@example.com",
      "name": "John",
      "sso_enabled": false,
      "global_role": "admin",
      "teams": []
    },
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 4,
      "email": "bob_marks@example.com",
      "name": "Bob",
      "sso_enabled": false,
      "global_role": "admin",
      "teams": []
    }
  ]
}
```

## Delete invite

Delete the specified invite from Fleet.

`DELETE /api/v1/fleet/invites/{id}`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required.** The user's id. |

#### Example

`DELETE /api/v1/fleet/invites/{id}`

##### Default response

`Status: 200`


## Verify invite

Verify the specified invite.

`GET /api/v1/fleet/invites/{token}`

#### Parameters

| Name  | Type    | In   | Description                            |
| ----- | ------- | ---- | -------------------------------------- |
| token | integer | path | **Required.** The user's invite token. |

#### Example

`GET /api/v1/fleet/invites/{token}`

##### Default response

`Status: 200`

```json
{
    "invite": {
        "created_at": "2021-01-15T00:58:33Z",
        "updated_at": "2021-01-15T00:58:33Z",
        "id": 4,
        "email": "steve@example.com",
        "name": "Steve",
        "sso_enabled": false,
        "global_role": "admin",
        "teams": []
    }
}
```

##### Not found

`Status: 404`

```json
{
    "message": "Resource Not Found",
    "errors": [
        {
            "name": "base",
            "reason": "Invite with token <token> was not found in the datastore"
        }
    ]
}
```

## Update invite

`PATCH /api/v1/fleet/invites/{id}`

#### Parameters

| Name        | Type    | In   | Description                                                                                                                                           |
| ----------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| global_role | string  | body | Role the user will be granted. Either a global role is needed, or a team role.                                                                        |
| email       | string  | body | The email of the invited user. Updates on the email won't resend the invitation.                                                             |
| name        | string  | body | The name of the invited user.                                                                                                           |
| sso_enabled | boolean | body | Whether or not SSO will be enabled for the invited user.                                                                                |
| teams       | list    | body | _Available in Fleet Premium_ A list of the teams the user is a member of. Each item includes the team's ID and the user's role in the specified team. |

#### Example

`PATCH /api/v1/fleet/invites/123`

##### Request body

```json
{
  "email": "john_appleseed@example.com",
  "name": "John",
  "sso_enabled": false,
  "global_role": null,
  "teams": [
    {
      "id": 2,
      "role": "observer"
    },
    {
      "id": 3,
      "role": "maintainer"
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "invite": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 3,
    "invited_by": 1,
    "email": "john_appleseed@example.com",
    "name": "John",
    "sso_enabled": false,
    "teams": [
      {
        "id": 10,
        "created_at": "0001-01-01T00:00:00Z",
        "name": "Apples",
        "description": "",
        "agent_options": null,
        "user_count": 0,
        "host_count": 0,
        "role": "observer"
      },
      {
        "id": 14,
        "created_at": "0001-01-01T00:00:00Z",
        "name": "Best of the Best Engineering",
        "description": "",
        "agent_options": null,
        "user_count": 0,
        "host_count": 0,
        "role": "maintainer"
      }
    ]
  }
}
```

## Version

Get version and build information from the Fleet server.

`GET /api/v1/fleet/version`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/version`

##### Default response

`Status: 200`

```json
{
  "version": "3.9.0-93-g1b67826f-dirty",
  "branch": "version",
  "revision": "1b67826fe4bf40b2f45ec53e01db9bf467752e74",
  "go_version": "go1.15.7",
  "build_date": "2021-03-27T00:28:48Z",
  "build_user": "zwass"
}
```

<meta name="pageOrderInSection" value="400">