# Teams

## List teams

_Available in Fleet Premium_

`GET /api/v1/fleet/teams`

### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |
| order_key       | string  | query | What to order results by. Can be any column in the `teams` table.                                                             |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| query           | string  | query | Search query keywords. Searchable fields include `name`.                                                                      |

### Example

`GET /api/v1/fleet/teams`

#### Default response

`Status: 200`

```json
{
  "teams": [
    {
      "id": 1,
      "created_at": "2021-07-28T15:58:21Z",
      "name": "workstations",
      "description": "",
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
      "user_count": 0,
      "host_count": 0,
      "secrets": [
        {
          "secret": "",
          "created_at": "2021-07-28T15:58:21Z",
          "team_id": 10
        }
      ]
    },
    {
      "id": 2,
      "created_at": "2021-08-05T21:41:42Z",
      "name": "servers",
      "description": "",
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
        },
        "user_count": 0,
        "host_count": 0,
        "secrets": [
          {
            "secret": "+ncixtnZB+IE0OrbrkCLeul3U8LMVITd",
            "created_at": "2021-08-05T21:41:42Z",
            "team_id": 15
          }
        ]
      }
    }
  ]
}
```

## Get team

_Available in Fleet Premium_

`GET /api/v1/fleet/teams/:id`

### Parameters

| Name | Type    | In   | Description                          |
| ---- | ------  | ---- | ------------------------------------ |
| id   | integer | path | **Required.** The desired team's ID. |

### Example

`GET /api/v1/fleet/teams/1`

#### Default response

`Status: 200`

```json
{
  "team": {
    "name": "Workstations",
    "id": 1,
    "user_count": 4,
    "host_count": 0,
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
    "webhook_settings": {
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      }
    },
    "integrations": {
      "google_calendar": {
        "enable_calendar_events": true,
        "webhook_url": "https://server.com/example"
      }
    },
    "mdm": {
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
      "macos_setup": {
        "bootstrap_package": "",
        "enable_end_user_authentication": false,
        "macos_setup_assistant": "path/to/config.json"
      }
    }
  }
}
```

## Create team

_Available in Fleet Premium_

`POST /api/v1/fleet/teams`

### Parameters

| Name | Type   | In   | Description                    |
| ---- | ------ | ---- | ------------------------------ |
| name | string | body | **Required.** The team's name. |

### Example

`POST /api/v1/fleet/teams`

#### Request body

```json
{
  "name": "workstations"
}
```

#### Default response

`Status: 200`

```json
{
  "teams": [
    {
      "name": "workstations",
      "id": 1,
      "user_count": 0,
      "host_count": 0,
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
      "webhook_settings": {
        "failing_policies_webhook": {
          "enable_failing_policies_webhook": false,
          "destination_url": "",
          "policy_ids": null,
          "host_batch_size": 0
        }
      }
    }
  ]
}
```

## Modify team

_Available in Fleet Premium_

`PATCH /api/v1/fleet/teams/:id`

### Parameters

| Name                                                    | Type    | In   | Description                                                                                                                                                                                               |
| ------------------------------------------------------- | ------- | ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id                                                      | integer | path | **Required.** The desired team's ID.                                                                                                                                                                      |
| name                                                    | string  | body | The team's name.                                                                                                                                                                                          |
| host_ids                                                | array    | body | A list of hosts that belong to the team.                                                                                                                                                                  |
| user_ids                                                | array    | body | A list of users on the team.                                                                                                                                                             |
| webhook_settings                                        | object  | body | Webhook settings contains for the team.                                                                                                                                                                   |
| &nbsp;&nbsp;failing_policies_webhook                    | object  | body | Failing policies webhook settings.                                                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies_webhook | boolean | body | Whether or not the failing policies webhook is enabled.                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;destination_url                 | string  | body | The URL to deliver the webhook requests to.                                                                                                                                                               |
| &nbsp;&nbsp;&nbsp;&nbsp;policy_ids                      | array   | body | List of policy IDs to enable failing policies webhook.                                                                                                                                                    |
| &nbsp;&nbsp;host_status_webhook                    | object  | body | Host status webhook settings. |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_host_status_webhook | boolean | body | Whether or not the host status webhook is enabled. |
| &nbsp;&nbsp;&nbsp;&nbsp;destination_url            | string | body | The URL to deliver the webhook request to. |
| &nbsp;&nbsp;&nbsp;&nbsp;host_percentage            | integer | body | The minimum percentage of hosts that must fail to check in to Fleet in order to trigger the webhook request. |
| &nbsp;&nbsp;&nbsp;&nbsp;days_count | integer | body | The minimum number of days that the configured `host_percentage` must fail to check in to Fleet in order to trigger the webhook request. |
| &nbsp;&nbsp;&nbsp;&nbsp;host_batch_size                 | integer | body | Maximum number of hosts to batch on failing policy webhook requests. The default, 0, means no batching (all hosts failing a policy are sent on one request).                                              |
| integrations                                            | object  | body | Integrations settings for the team. Note that integrations referenced here must already exist globally, created by a call to [Modify configuration](#modify-configuration).                               |
| &nbsp;&nbsp;jira                                        | array   | body | Jira integrations configuration.                                                                                                                                                                          |
| &nbsp;&nbsp;&nbsp;&nbsp;url                             | string  | body | The URL of the Jira server to use.                                                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;project_key                     | string  | body | The project key of the Jira integration to use. Jira tickets will be created in this project.                                                                                                             |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies         | boolean | body | Whether or not that Jira integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies).    |
| &nbsp;&nbsp;zendesk                                     | array   | body | Zendesk integrations configuration.                                                                                                                                                                       |
| &nbsp;&nbsp;&nbsp;&nbsp;url                             | string  | body | The URL of the Zendesk server to use.                                                                                                                                                                     |
| &nbsp;&nbsp;&nbsp;&nbsp;group_id                        | integer | body | The Zendesk group id to use. Zendesk tickets will be created in this group.                                                                                                                               |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies         | boolean | body | Whether or not that Zendesk integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| mdm                                                     | object  | body | MDM settings for the team.                                                                                                                                                                                |
| &nbsp;&nbsp;macos_updates                               | object  | body | macOS updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;minimum_version                 | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM will be nudged until their macOS is at or above this version.                                                                            |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline                        | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past.                                                                    |
| &nbsp;&nbsp;ios_updates                               | object  | body | iOS updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;minimum_version                 | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM will be nudged until their iOS is at or above this version.                                                                            |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline                        | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past.                                                                    |
| &nbsp;&nbsp;ipados_updates                               | object  | body | iPadOS updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;minimum_version                 | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM will be nudged until their iPadOS is at or above this version.                                                                            |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline                        | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past.                                                                    |
| &nbsp;&nbsp;windows_updates                             | object  | body | Windows updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline_days                   | integer | body | Hosts that belong to this team and are enrolled into Fleet's MDM will have this number of days before updates are installed on Windows.                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;grace_period_days               | integer | body | Hosts that belong to this team and are enrolled into Fleet's MDM will have this number of days before Windows restarts to install updates.                                                                    |
| &nbsp;&nbsp;macos_settings                              | object  | body | macOS-specific settings.                                                                                                                                                                                  |
| &nbsp;&nbsp;&nbsp;&nbsp;custom_settings                 | array    | body | The list of objects where each object includes .mobileconfig or JSON file (configuration profile) and label name to apply to macOS hosts that belong to this team and are members of the specified label.                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_disk_encryption          | boolean | body | Hosts that belong to this team and are enrolled into Fleet's MDM will have disk encryption enabled if set to true.                                                                                        |
| &nbsp;&nbsp;windows_settings                            | object  | body | Windows-specific settings.                                                                                                                                                                                |
| &nbsp;&nbsp;&nbsp;&nbsp;custom_settings                 | array    | body | The list of objects where each object includes XML file (configuration profile) and label name to apply to Windows hosts that belong to this team and are members of the specified label.                                                                                                                               |
| &nbsp;&nbsp;macos_setup                                 | object  | body | Setup for automatic MDM enrollment of macOS hosts.                                                                                                                                                      |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_end_user_authentication  | boolean | body | If set to true, end user authentication will be required during automatic MDM enrollment of new macOS hosts. Settings for your IdP provider must also be [configured](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula).                                                                                      |
| integrations                                            | object  | body | Integration settings for this team.                                                                                                                                                                   |
| &nbsp;&nbsp;google_calendar                             | object  | body | Google Calendar integration settings.                                                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_calendar_events          | boolean | body | Whether or not calendar events are enabled for this team.                                                                                                                                                  |
| &nbsp;&nbsp;&nbsp;&nbsp;webhook_url                     | string | body | The URL to send a request to during calendar events, to trigger auto-remediation.                |
| host_expiry_settings                                    | object  | body | Host expiry settings for the team.                                                                                                                                                                         |
| &nbsp;&nbsp;host_expiry_enabled                         | boolean | body | When enabled, allows automatic cleanup of hosts that have not communicated with Fleet in some number of days. When disabled, defaults to the global setting.                                               |
| &nbsp;&nbsp;host_expiry_window                          | integer | body | If a host has not communicated with Fleet in the specified number of days, it will be removed.                                                                                                             |

### Example (transfer hosts to a team)

`PATCH /api/v1/fleet/teams/1`

#### Request body

```json
{
  "host_ids": [3, 6, 7, 8, 9, 20, 32, 44]
}
```

#### Default response

`Status: 200`

```json
{
  "team": {
    "name": "Workstations",
    "id": 1,
    "user_count": 4,
    "host_count": 8,
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
    "webhook_settings": {
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      }
    }
  }
}
```

## Add users to a team

_Available in Fleet Premium_

`PATCH /api/v1/fleet/teams/:id/users`

### Parameters

| Name             | Type    | In   | Description                                  |
|------------------|---------|------|----------------------------------------------|
| id               | integer | path | **Required.** The desired team's ID.         |
| users            | string  | body | Array of users to add.                       |
| &nbsp;&nbsp;id   | integer | body | The id of the user.                          |
| &nbsp;&nbsp;role | string  | body | The team role that the user will be granted. Options are: "admin", "maintainer", "observer", "observer_plus", and "gitops". |

### Example

`PATCH /api/v1/fleet/teams/1/users`

#### Request body

```json
{
  "users": [
    {
      "id": 1,
      "role": "admin"
    },
    {
      "id": 17,
      "role": "observer"
    }
  ]
}
```

#### Default response

`Status: 200`

```json
{
  "team": {
    "name": "Workstations",
    "id": 1,
    "user_count": 2,
    "host_count": 0,
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
    "webhook_settings": {
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      }
    },
    "mdm": {
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
      "macos_setup": {
        "bootstrap_package": "",
        "enable_end_user_authentication": false,
        "macos_setup_assistant": "path/to/config.json"
      }
    },
    "users": [
      {
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "id": 1,
        "name": "Example User1",
        "email": "user1@example.com",
        "force_password_reset": false,
        "gravatar_url": "",
        "sso_enabled": false,
        "global_role": null,
        "api_only": false,
        "teams": null,
        "role": "admin"
      },
      {
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "id": 17,
        "name": "Example User2",
        "email": "user2@example.com",
        "force_password_reset": false,
        "gravatar_url": "",
        "sso_enabled": false,
        "global_role": null,
        "api_only": false,
        "teams": null,
        "role": "observer"
      }
    ]
  }
}
```

## Modify team's agent options

_Available in Fleet Premium_

`POST /api/v1/fleet/teams/:id/agent_options`

### Parameters

| Name                             | Type    | In    | Description                                                                                                                                                  |
| ---                              | ---     | ---   | ---                                                                                                                                                          |
| id                               | integer | path  | **Required.** The desired team's ID.                                                                                                                         |
| force                            | boolean | query | Force apply the options even if there are validation errors.                                                                                                 |
| dry_run                          | boolean | query | Validate the options and return any validation errors, but do not apply the changes.                                                                         |
| _JSON data_                      | object  | body  | The JSON to use as agent options for this team. See [Agent options](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options) for details.                              |

### Example

`POST /api/v1/fleet/teams/1/agent_options`

#### Request body

```json
{
  "config": {
    "options": {
      "pack_delimiter": "/",
      "logger_tls_period": 20,
      "distributed_plugin": "tls",
      "disable_distributed": false,
      "logger_tls_endpoint": "/api/v1/osquery/log",
      "distributed_interval": 60,
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
```

#### Default response

`Status: 200`

```json
{
  "team": {
    "name": "Workstations",
    "id": 1,
    "user_count": 4,
    "host_count": 8,
    "agent_options": {
      "config": {
        "options": {
          "pack_delimiter": "/",
          "logger_tls_period": 20,
          "distributed_plugin": "tls",
          "disable_distributed": false,
          "logger_tls_endpoint": "/api/v1/osquery/log",
          "distributed_interval": 60,
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
    "webhook_settings": {
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      }
    }
  }
}
```

## Delete team

_Available in Fleet Premium_

`DELETE /api/v1/fleet/teams/:id`

### Parameters

| Name | Type    | In   | Description                          |
| ---- | ------  | ---- | ------------------------------------ |
| id   | integer | path | **Required.** The desired team's ID. |

### Example

`DELETE /api/v1/fleet/teams/1`

### Default response

`Status: 200`

---

# Translator

- [Translate IDs](#translate-ids)

## Translate IDs

Transforms a host name into a host id. For example, the Fleet UI use this endpoint when sending live queries to a set of hosts.

`POST /api/v1/fleet/translate`

### Parameters

| Name  | Type  | In   | Description                              |
| ----- | ----- | ---- | ---------------------------------------- |
| array | array | body | **Required** list of items to translate. |

### Example

`POST /api/v1/fleet/translate`

#### Request body

```json
{
  "list": [
    {
      "type": "user",
      "payload": {
        "identifier": "some@email.com"
      }
    },
    {
      "type": "label",
      "payload": {
        "identifier": "labelA"
      }
    },
    {
      "type": "team",
      "payload": {
        "identifier": "team1"
      }
    },
    {
      "type": "host",
      "payload": {
        "identifier": "host-ABC"
      }
    }
  ]
}
```

#### Default response

`Status: 200`

```json
{
  "list": [
    {
      "type": "user",
      "payload": {
        "identifier": "some@email.com",
        "id": 32
      }
    },
    {
      "type": "label",
      "payload": {
        "identifier": "labelA",
        "id": 1
      }
    },
    {
      "type": "team",
      "payload": {
        "identifier": "team1",
        "id": 22
      }
    },
    {
      "type": "host",
      "payload": {
        "identifier": "host-ABC",
        "id": 45
      }
    }
  ]
}
```
---

<meta name="description" value="Documentation for Fleet's teams REST API endpoints.">
<meta name="pageOrderInSection" value="170">