# Fleet configuration

The Fleet server exposes a handful of API endpoints that handle the configuration of Fleet as well as endpoints that manage invitation and enroll secret operations. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

## Get certificate

Returns the Fleet certificate.

`GET /api/v1/fleet/config/certificate`

### Parameters

None.

### Example

`GET /api/v1/fleet/config/certificate`

#### Default response

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

### Parameters

None.

### Example

`GET /api/v1/fleet/config`

#### Default response

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
    "enable_analytics": true,
    "live_query_disabled": false,
    "query_reports_disabled": false,
    "ai_features_disabled": false
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
  "activity_expiry_settings": {
    "activity_expiry_enabled": false,
    "activity_expiry_window": 0
  },
  "features": {
    "enable_host_users": true,
    "enable_software_inventory": true,
    "additional_queries": null
  },
  "mdm": {
    "windows_enabled_and_configured": true,
    "enable_disk_encryption": true,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "ios_updates": {
      "minimum_version": "17.0.1",
      "deadline": "2024-08-01"
    },
    "ipados_updates": {
      "minimum_version": "17.0.1",
      "deadline": "2024-08-01"
    },
    "windows_updates": {
      "deadline_days": 5,
      "grace_period_days": 1
    },
    "macos_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile1.mobileconfig",
          "labels": ["Label 1", "Label 2"]
        }
      ]
    },
    "windows_settings": {
      "custom_settings": [
        {
         "path": "path/to/profile2.xml",
         "labels": ["Label 3", "Label 4"]
        }
      ],
    },
    "scripts": ["path/to/script.sh"],
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
      "macos_setup_assistant": "path/to/config.json",
      "enable_release_device_manually": true
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
    },
    "activities_webhook":{
      "enable_activities_webhook":true,
      "destination_url": "https://server.com"
    }
  },
  "integrations": {
    "jira": null,
    "google_calendar": [
      {
        "domain": "example.com",
        "api_key_json": {
           "type": "service_account",
           "project_id": "fleet-in-your-calendar",
           "private_key_id": "<private key id>",
           "private_key": "-----BEGIN PRIVATE KEY-----\n<private key>\n-----END PRIVATE KEY-----\n",
           "client_email": "fleet-calendar-events@fleet-in-your-calendar.iam.gserviceaccount.com",
           "client_id": "<client id>",
           "auth_uri": "https://accounts.google.com/o/oauth2/auth",
           "token_uri": "https://oauth2.googleapis.com/token",
           "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
           "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/fleet-calendar-events%40fleet-in-your-calendar.iam.gserviceaccount.com",
           "universe_domain": "googleapis.com"
         }
      }
    ]
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
    "disable_schedule": false,
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

### Parameters

| Name                     | Type    | In    | Description   |
| -----------------------  | ------- | ----  | ------------------------------------------------------------------------------------------------------------------------------------ |
| org_info                 | object  | body  | See [org_info](#org-info).                                                                                                           |
| server_settings          | object  | body  | See [server_settings](#server-settings).                                                                                             |
| smtp_settings            | object  | body  | See [smtp_settings](#smtp-settings).                                                                                                 |
| sso_settings             | object  | body  | See [sso_settings](#sso-settings).                                                                                                   |
| host_expiry_settings     | object  | body  | See [host_expiry_settings](#host-expiry-settings).                                                                                   |
| activity_expiry_settings | object  | body  | See [activity_expiry_settings](#activity-expiry-settings).                                                                           |
| agent_options            | objects | body  | The agent_options spec that is applied to all hosts. In Fleet 4.0.0 the `api/v1/fleet/spec/osquery_options` endpoints were removed.  |
| fleet_desktop            | object  | body  | See [fleet_desktop](#fleet-desktop).                                                                                                 |
| webhook_settings         | object  | body  | See [webhook_settings](#webhook-settings).                                                                                           |
| integrations             | object  | body  | Includes `jira`, `zendesk`, and `google_calendar` arrays. See [integrations](#integrations) for details.                             |
| mdm                      | object  | body  | See [mdm](#mdm).                                                                                                                     |
| features                 | object  | body  | See [features](#features).                                                                                                           |
| scripts                  | array   | body  | A list of script files to add so they can be executed at a later time.                                                               |
| force                    | boolean | query | Whether to force-apply the agent options even if there are validation errors.                                                        |
| dry_run                  | boolean | query | Whether to validate the configuration and return any validation errors **without** applying changes.                                 |


### Example

`PATCH /api/v1/fleet/config`

#### Request body

```json
{
  "scripts": []
}
```

#### Default response

`Status: 200`

```json
{
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png",
    "org_logo_url_light_background": "https://fleetdm.com/logo-light.png",
    "contact_url": "https://fleetdm.com/company/contact"
  },
  "server_settings": {
    "server_url": "https://localhost:8080",
    "enable_analytics": true,
    "live_query_disabled": false,
    "query_reports_disabled": false,
    "ai_features_disabled": false
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
    "enable_sso": false,
    "enable_sso_idp_login": false,
    "enable_jit_provisioning": false
  },
  "host_expiry_settings": {
    "host_expiry_enabled": false,
    "host_expiry_window": 0
  },
  "activity_expiry_settings": {
    "activity_expiry_enabled": false,
    "activity_expiry_window": 0
  },
  "features": {
    "enable_host_users": true,
    "enable_software_inventory": true,
    "additional_queries": null
  },
  "license": {
    "tier": "free",
    "expiration": "0001-01-01T00:00:00Z"
  },
  "mdm": {
    "enabled_and_configured": false,
    "windows_enabled_and_configured": false,
    "enable_disk_encryption": true,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "ios_updates": {
      "minimum_version": "17.0.1",
      "deadline": "2024-08-01"
    },
    "ipados_updates": {
      "minimum_version": "17.0.1",
      "deadline": "2024-08-01"
    },
    "windows_updates": {
      "deadline_days": 5,
      "grace_period_days": 1
    },
    "macos_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile1.mobileconfig",
          "labels_exclude_any": ["Label 1", "Label 2"]
        },
        {
          "path": "path/to/profile2.json",
          "labels_include_all": ["Label 3", "Label 4"]
        },
      ]
    },
    "windows_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile3.xml",
          "labels_exclude_any": ["Label 1", "Label 2"]
        }
      ]
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
  "fleet_desktop": {
    "transparency_url": "https://fleetdm.com/better"
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
    },
    "activities_webhook":{
      "enable_activities_webhook":true,
      "destination_url": "https://server.com"
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
    ],
    "google_calendar": [
      {
        "domain": "",
        "api_key_json": null
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
  },
  "scripts": []
}
```


### org_info

| Name                              | Type    | Description   |
| ---------------------             | ------- | ----------------------------------------------------------------------------------- |
| org_name                          | string  | The organization name.                                                              |
| org_logo_url                      | string  | The URL for the organization logo.                                                  |
| org_logo_url_light_background     | string  | The URL for the organization logo displayed in Fleet on top of light backgrounds.   |
| contact_url                       | string  | A URL that can be used by end users to contact the organization.                    |

<br/>

#### Example request body

```json
{
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png",
    "org_logo_url_light_background": "https://fleetdm.com/logo-light.png",
    "contact_url": "https://fleetdm.com/company/contact"
  }
}
```

### server_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | ------------------------------------------------------------------------------------------- |
| server_url                        | string  | The Fleet server URL.                                                                       |
| enable_analytics                  | boolean | Whether to send anonymous usage statistics. Always enabled for Fleet Premium customers.     |
| live_query_disabled               | boolean | Whether the live query capabilities are disabled.                                           |
| query_reports_disabled            | boolean | Whether query report capabilities are disabled.                                             |
| ai_features_disabled              | boolean | Whether AI features are disabled.                                                           |
| query_report_cap                  | integer | The maximum number of results to store per query report before the report is clipped. If increasing this cap, we recommend enabling reports for one query at time and monitoring your infrastructure. (Default: `1000`) |

<br/>

#### Example request body

```json
{
  "server_settings": {
    "server_url": "https://localhost:8080",
    "enable_analytics": true, 
    "live_query_disabled": false,
    "query_reports_disabled": false,
    "ai_features_disabled": false
  }
}
```

### smtp_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_smtp                       | boolean | Whether SMTP is enabled for the Fleet app.                                                                                                                            |
| sender_address                    | string  | The sender email address for the Fleet app. An invitation email is an example of the emails that may use this sender address                                          |
| server                            | string  | The SMTP server for the Fleet app.                                                                                                                                    |
| port                              | integer | The SMTP port for the Fleet app.                                                                                                                                      |
| authentication_type               | string  | The authentication type used by the SMTP server. Options include `"authtype_username_and_password"` or `"none"`                                                       |
| user_name                         | string  | The username used to authenticate requests made to the SMTP server.                                                                                                   |
| password                          | string  | The password used to authenticate requests made to the SMTP server.                                                                                                   |
| enable_ssl_tls                    | boolean | Whether or not SSL and TLS are enabled for the SMTP server.                                                                                                           |
| authentication_method             | string  | The authentication method used to make authenticate requests to SMTP server. Options include `"authmethod_plain"`, `"authmethod_cram_md5"`, and `"authmethod_login"`. |
| domain                            | string  | The domain for the SMTP server.                                                                                                                                       |
| verify_ssl_certs                  | boolean | Whether or not SSL certificates are verified by the SMTP server. Turn this off (not recommended) if you use a self-signed certificate.                                |
| enabled_start_tls                 | boolean | Detects if STARTTLS is enabled in your SMTP server and starts to use it.                                                                                              |

<br/>

#### Example request body

```json
{
  "smtp_settings": {
    "enable_smtp": true,
    "sender_address": "",
    "server": "localhost",
    "port": 1025,
    "authentication_type": "authtype_username_none",
    "user_name": "",
    "password": "",
    "enable_ssl_tls": true,
    "authentication_method": "authmethod_plain",
    "domain": "",
    "verify_ssl_certs": true,
    "enable_start_tls": true
  }
}
```

### sso_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_sso                        | boolean | Whether or not SSO is enabled for the Fleet application. If this value is true, you must also include most of the SSO settings parameters below.                       |
| entity_id                         | string  | The required entity ID is a URI that you use to identify Fleet when configuring the identity provider. Must be 5 or more characters.                                   |
| issuer_uri                        | string  | The URI you provide here must exactly match the Entity ID field used in the identity provider configuration.                                                           |
| idp_image_url                     | string  | An optional link to an image such as a logo for the identity provider.                                                                                                 |
| metadata_url                      | string  | A URL that references the identity provider metadata. If available from the identity provider, this is the preferred means of providing metadata. Must be either https or http |
| metadata                          | string  |  Metadata provided by the identity provider. Either `metadata` or a `metadata_url` must be provided.                                                                   |
| enable_sso_idp_login              | boolean | Determines whether Identity Provider (IdP) initiated login for Single sign-on (SSO) is enabled for the Fleet application.                                              |
| enable_jit_provisioning           | boolean | _Available in Fleet Premium._ When enabled, allows [just-in-time user provisioning](https://fleetdm.com/docs/deploy/single-sign-on-sso#just-in-time-jit-user-provisioning). |

<br/>

#### Example request body

```json
{
  "sso_settings": {
    "enable_sso": false,
    "entity_id": "",
    "issuer_uri": "",
    "idp_image_url": "",
    "metadata_url": "",
    "metadata": "",
    "idp_name": "",
    "enable_sso_idp_login": false,
    "enable_jit_provisioning": false
  }
}
```

### host_expiry_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| host_expiry_enabled               | boolean | When enabled, allows automatic cleanup of hosts that have not communicated with Fleet in some number of days.                                                  |
| host_expiry_window                | integer | If a host has not communicated with Fleet in the specified number of days, it will be removed. Must be greater than 0 if host_expiry_enabled is set to true.   |

<br/>

#### Example request body

```json
{
  "host_expiry_settings": {
    "host_expiry_enabled": true,
    "host_expiry_window": 7
  }
}
```

### activity_expiry_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | --------------------------------------------------------------------------------------------------------------------------------- |
| activity_expiry_enabled           | boolean | When enabled, allows automatic cleanup of activities (and associated live query data) older than the specified number of days.    |
| activity_expiry_window            | integer | The number of days to retain activity records, if activity expiry is enabled.                                                     |

<br/>

#### Example request body

```json
{
  "activity_expiry_settings": {
    "activity_expiry_enabled": true,
    "activity_expiry_window": 90
  }
}
```

### fleet_desktop

_Available in Fleet Premium._

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------- |
| transparency_url                  | string  | The URL used to display transparency information to users of Fleet Desktop.      |

<br/>

#### Example request body

```json
{
  "fleet_desktop": {
    "transparency_url": "https://fleetdm.com/better"
  }
}
```

### webhook_settings

<!--
+ [`webhook_settings.host_status_webhook`](#webhook-settings-host-status-webhook)
+ [`webhook_settings.failing_policies_webhook`](#webhook-settings-failing-policies-webhook)
+ [`webhook_settings.vulnerabilities_webhook`](#webhook-settings-vulnerabilities-webhook)
+ [`webhook_settings.activities_webhook`](#webhook-settings-activities-webhook)
-->

| Name                              | Type  | Description   |
| ---------------------             | ----- | ---------------------------------------------------------------------------------------------- |
| host_status_webhook               | array | See [`webhook_settings.host_status_webhook`](#webhook-settings-host-status-webhook).           |
| failing_policies_webhook          | array | See [`webhook_settings.failing_policies_webhook`](#webhook-settings-failing-policies-webhook). |
| vulnerabilities_webhook           | array | See [`webhook_settings.vulnerabilities_webhook`](#webhook-settings-vulnerabilities-webhook).   |
| activities_webhook                | array | See [`webhook_settings.activities_webhook`](#webhook-settings-activities-webhook).             |

<br/>

#### webhook_settings.host_status_webhook

`webhook_settings.host_status_webhook` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_host_status_webhook        | boolean | Whether or not the host status webhook is enabled.                                                                                          |
| destination_url                   | string  | The URL to deliver the webhook request to.                                                                                                  |
| host_percentage                   | integer | The minimum percentage of hosts that must fail to check in to Fleet in order to trigger the webhook request.                                |
| days_count                        | integer | The minimum number of days that the configured `host_percentage` must fail to check in to Fleet in order to trigger the webhook request.    |

<br/>

#### webhook_settings.failing_policies_webhook

`webhook_settings.failing_policies_webhook` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | ------------------------------------------------------------------------------------------------------------------- |
| enable_failing_policies_webhook   | boolean | Whether or not the failing policies webhook is enabled.                                                             |
| destination_url                   | string  | The URL to deliver the webhook requests to.                                                                         |
| policy_ids                        | array   | List of policy IDs to enable failing policies webhook.                                                              |
| host_batch_size                   | integer | Maximum number of hosts to batch on failing policy webhook requests. The default, 0, means no batching (all hosts failing a policy are sent on one request). |

<br/>

#### webhook_settings.vulnerabilities_webhook

`webhook_settings.vulnerabilities_webhook` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_vulnerabilities_webhook    | boolean | Whether or not the vulnerabilities webhook is enabled.                                                                                                  |
| destination_url                   | string  | The URL to deliver the webhook requests to.                                                                                                             |
| host_batch_size                   | integer | Maximum number of hosts to batch on vulnerabilities webhook requests. The default, 0, means no batching (all vulnerable hosts are sent on one request). |

<br/>

#### webhook_settings.activities_webhook

`webhook_settings.activities_webhook` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | --------------------------------------------------------- |
| enable_activities_webhook         | boolean | Whether or not the activity feed webhook is enabled.      |
| destination_url                   | string  | The URL to deliver the webhook requests to.               |

<br/>

#### Example request body

```json
{
  "webhook_settings": {
    "host_status_webhook": {
      "enable_host_status_webhook": true,
      "destination_url": "https://server.com",
      "host_percentage": 5,
      "days_count": 7
    },
    "failing_policies_webhook":{
      "enable_failing_policies_webhook": true,
      "destination_url": "https://server.com",
      "policy_ids": [1, 2, 3],
      "host_batch_size": 1000
    },
    "vulnerabilities_webhook":{
      "enable_vulnerabilities_webhook":true,
      "destination_url": "https://server.com",
      "host_batch_size": 1000
    },
    "activities_webhook":{
      "enable_activities_webhook":true,
      "destination_url": "https://server.com"
    }
  }
}
```

### integrations

<!--
+ [`integrations.jira`](#integrations-jira)
+ [`integrations.zendesk`](#integrations-zendesk)
+ [`integrations.google_calendar`](#integrations-google-calendar)
-->

| Name                  | Type  | Description   |
| --------------------- | ----- | -------------------------------------------------------------------- |
| jira                  | array | See [`integrations.jira`](#integrations-jira).                       |
| zendesk               | array | See [`integrations.zendesk`](#integrations-zendesk).                 |
| google_calendar       | array | See [`integrations.google_calendar`](#integrations-google-calendar). |


> Note that when making changes to the `integrations` object, all integrations must be provided (not just the one being modified). This is because the endpoint will consider missing integrations as deleted.

<br/>

#### integrations.jira

`integrations.jira` is an array of objects with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_software_vulnerabilities   | boolean | Whether or not Jira integration is enabled for software vulnerabilities. Only one vulnerability automation can be enabled at a given time (enable_vulnerabilities_webhook and enable_software_vulnerabilities). |
| enable_failing_policies           | boolean | Whether or not Jira integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| url                               | string  | The URL of the Jira server to integrate with. |
| username                          | string  | The Jira username to use for this Jira integration. |
| api_token                         | string  | The API token of the Jira username to use for this Jira integration. |
| project_key                       | string  | The Jira project key to use for this integration. Jira tickets will be created in this project. |

<br/>

#### integrations.zendesk

`integrations.zendesk` is an array of objects with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_software_vulnerabilities   | boolean | Whether or not Zendesk integration is enabled for software vulnerabilities. Only one vulnerability automation can be enabled at a given time (enable_vulnerabilities_webhook and enable_software_vulnerabilities). |
| enable_failing_policies           | boolean | Whether or not Zendesk integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| url                               | string  | The URL of the Zendesk server to integrate with. |
| email                             | string  | The Zendesk user email to use for this Zendesk integration. |
| api_token                         | string  | The Zendesk API token to use for this Zendesk integration. |
| group_id                          | integer | The Zendesk group id to use for this integration. Zendesk tickets will be created in this group. |

<br/>

#### integrations.google_calendar

`integrations.google_calendar` is an array of objects with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | --------------------------------------------------------------------------------------------------------------------- |
| domain                            | string  | The domain for the Google Workspace service account to be used for this calendar integration.                         |
| api_key_json                      | object  | The private key JSON downloaded when generating the service account API key to be used for this calendar integration. |

<br/>

#### Example request body

```json
{
  "integrations": {
    "jira": [
      {
        "enable_software_vulnerabilities": false,
        "enable_failing_poilicies": true,
        "url": "https://jiraserver.com",
        "username": "some_user",
        "api_token": "<TOKEN>",
        "project_key": "jira_project",
      }
    ],
    "zendesk": [],
    "google_calendar": [
      {
        "domain": "https://domain.com",
        "api_key_json": "<API KEY JSON>"
      }
    ]
  }
}
```

### mdm

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| windows_enabled_and_configured    | boolean | Enables Windows MDM support. |
| enable_disk_encryption            | boolean | _Available in Fleet Premium._ Hosts that belong to no team will have disk encryption enabled if set to true. |
| macos_updates         | object  | See [`mdm.macos_updates`](#mdm-macos-updates). |
| ios_updates         | object  | See [`mdm.ios_updates`](#mdm-ios-updates). |
| ipados_updates         | object  | See [`mdm.ipados_updates`](#mdm-ipados-updates). |
| windows_updates         | object  | See [`mdm.window_updates`](#mdm-windows-updates). |
| macos_migration         | object  | See [`mdm.macos_migration`](#mdm-macos-migration). |
| macos_setup         | object  | See [`mdm.macos_setup`](#mdm-macos-setup). |
| macos_settings         | object  | See [`mdm.macos_settings`](#mdm-macos-settings). |
| windows_settings         | object  | See [`mdm.windows_settings`](#mdm-windows-settings). |

<br/>

#### mdm.macos_updates

_Available in Fleet Premium._

`mdm.macos_updates` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| minimum_version                   | string  | Hosts that belong to no team will be nudged until their macOS is at or above this version. |
| deadline                          | string  | Hosts that belong to no team won't be able to dismiss the Nudge window once this deadline is past. |

<br/>

#### mdm.ios_updates

_Available in Fleet Premium._

`mdm.ios_updates` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| minimum_version                   | string  | Hosts that belong to no team and are enrolled into Fleet's MDM will be nudged until their iOS is at or above this version. |
| deadline                          | string  | Hosts that belong to no team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past. |

<br/>

#### mdm.ipados_updates

_Available in Fleet Premium._

`mdm.ipados_updates` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| minimum_version                   | string  | Hosts that belong to no team and are enrolled into Fleet's MDM will be nudged until their iPadOS is at or above this version. |
| deadline                          | string  | Hosts that belong to no team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past. |

<br/>

#### mdm.windows_updates

_Available in Fleet Premium._

`mdm.windows_updates` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| deadline_days                     | integer | Hosts that belong to no team will have this number of days before updates are installed on Windows. |
| grace_period_days                 | integer | Hosts that belong to no team will have this number of days before Windows restarts to install updates. |

<br/>

#### mdm.macos_migration

_Available in Fleet Premium._

`mdm.macos_migration` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable                            | boolean | Whether to enable the end user migration workflow for devices migrating from your old MDM solution. |
| mode                              | string  | The end user migration workflow mode for devices migrating from your old MDM solution. Options are `"voluntary"` or `"forced"`. |
| webhook_url                       | string  | The webhook url configured to receive requests to unenroll devices migrating from your old MDM solution. |

<br/>

#### mdm.macos_setup

_Available in Fleet Premium._

`mdm.macos_setup` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_end_user_authentication    | boolean | If set to true, end user authentication will be required during automatic MDM enrollment of new macOS devices. Settings for your IdP provider must also be [configured](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula). |

<br/>

#### mdm.macos_settings

`mdm.macos_settings` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| custom_settings                   | array   | macOS hosts that belong to no team will have custom profiles applied. |

<br/>

#### mdm.windows_settings

`mdm.windows_settings` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| custom_settings                   | array   | Windows hosts that belong to no team will have custom profiles applied. |

<br/>

#### Example request body

```json
{
  "mdm": {
    "windows_enabled_and_configured": false,
    "enable_disk_encryption": true,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "windows_updates": {
      "deadline_days": 5,
      "grace_period_days": 1
    },
    "macos_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile1.mobileconfig",
          "labels": ["Label 1", "Label 2"]
        },
        {
          "path": "path/to/profile2.json",
          "labels": ["Label 3", "Label 4"]
        },
      ]
    },
    "windows_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile3.xml",
          "labels": ["Label 1", "Label 2"]
        }
      ]     
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
  }
}
```

### Features

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_host_users                 | boolean | Whether to enable the users feature in Fleet. (Default: `true`)                                                                          |
| enable_software_inventory         | boolean | Whether to enable the software inventory feature in Fleet. (Default: `true`)                                                             |
| additional_queries                | boolean | Whether to enable additional queries on hosts. (Default: `null`)                                                                         |

<br/>

#### Example request body

```json
{
  "features": {
    "enable_host_users": true,
    "enable_software_inventory": true,
    "additional_queries": null
  }
}
```



## Get global enroll secrets

Returns the valid global enroll secrets.

`GET /api/v1/fleet/spec/enroll_secret`

### Parameters

None.

### Example

`GET /api/v1/fleet/spec/enroll_secret`

#### Default response

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

### Parameters

| Name      | Type    | In   | Description                                                        |
| --------- | ------- | ---- | ------------------------------------------------------------------ |
| spec      | object  | body | **Required**. Attribute "secrets" must be a list of enroll secrets |

### Example

Replace all global enroll secrets with a new enroll secret.

`POST /api/v1/fleet/spec/enroll_secret`

#### Request body

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

#### Default response

`Status: 200`

```json
{}
```

### Example

Delete all global enroll secrets.

`POST /api/v1/fleet/spec/enroll_secret`

#### Request body

```json
{
    "spec": {
        "secrets": []
    }
}
```

#### Default response

`Status: 200`

```json
{}
```

## Get team enroll secrets

Returns the valid team enroll secrets.

`GET /api/v1/fleet/teams/:id/secrets`

### Parameters

None.

### Example

`GET /api/v1/fleet/teams/1/secrets`

#### Default response

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


## Modify team enroll secrets

Replaces all existing team enroll secrets.

`PATCH /api/v1/fleet/teams/:id/secrets`

### Parameters

| Name      | Type    | In   | Description                            |
| --------- | ------- | ---- | -------------------------------------- |
| id        | integer | path | **Required**. The team's id.           |
| secrets   | array   | body | **Required**. A list of enroll secrets |

### Example

Replace all of a team's existing enroll secrets with a new enroll secret

`PATCH /api/v1/fleet/teams/2/secrets`

#### Request body

```json
{
  "secrets": [
    {
      "secret": "n07v32y53c237734m3n201153c237"
    }
  ]
}
```

#### Default response

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

### Example

Delete all of a team's existing enroll secrets

`PATCH /api/v1/fleet/teams/2/secrets`

#### Request body

```json
{
  "secrets": []
}
```

#### Default response

`Status: 200`

```json
{
  "secrets": null
}
```

## Create invite

`POST /api/v1/fleet/invites`

### Parameters

| Name        | Type    | In   | Description                                                                                                                                           |
| ----------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| global_role | string  | body | Role the user will be granted. Either a global role is needed, or a team role.                                                                        |
| email       | string  | body | **Required.** The email of the invited user. This email will receive the invitation link.                                                             |
| name        | string  | body | **Required.** The name of the invited user.                                                                                                           |
| sso_enabled | boolean | body | **Required.** Whether or not SSO will be enabled for the invited user.                                                                                |
| teams       | array   | body | _Available in Fleet Premium_. A list of the teams the user is a member of. Each item includes the team's ID and the user's role in the specified team. |

### Example

#### Request body

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

#### Default response

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

### Parameters

| Name            | Type   | In    | Description                                                                                                                   |
| --------------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| order_key       | string | query | What to order results by. Can be any column in the invites table.                                                             |
| order_direction | string | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| query           | string | query | Search query keywords. Searchable fields include `name` and `email`.                                                          |

### Example

`GET /api/v1/fleet/invites`

#### Default response

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

`DELETE /api/v1/fleet/invites/:id`

### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required.** The user's id. |

### Example

`DELETE /api/v1/fleet/invites/123`

#### Default response

`Status: 200`


## Verify invite

Verify the specified invite.

`GET /api/v1/fleet/invites/:token`

### Parameters

| Name  | Type    | In   | Description                            |
| ----- | ------- | ---- | -------------------------------------- |
| token | integer | path | **Required.** The user's invite token. |

### Example

`GET /api/v1/fleet/invites/abcdef012456789`

#### Default response

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

#### Not found

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

`PATCH /api/v1/fleet/invites/:id`

### Parameters

| Name        | Type    | In   | Description                                                                                                                                           |
| ----------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| global_role | string  | body | Role the user will be granted. Either a global role is needed, or a team role.                                                                        |
| email       | string  | body | The email of the invited user. Updates on the email won't resend the invitation.                                                             |
| name        | string  | body | The name of the invited user.                                                                                                           |
| sso_enabled | boolean | body | Whether or not SSO will be enabled for the invited user.                                                                                |
| teams       | array   | body | _Available in Fleet Premium_. A list of the teams the user is a member of. Each item includes the team's ID and the user's role in the specified team. |

### Example

`PATCH /api/v1/fleet/invites/123`

#### Request body

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

#### Default response

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

### Parameters

None.

### Example

`GET /api/v1/fleet/version`

#### Default response

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

---

---

<meta name="description" value="Documentation for Fleet's configuration REST API endpoints.">
<meta name="pageOrderInSection" value="60">