# Teams

- [List teams](#list-teams)
- [Get team](#get-team)
- [Create team](#create-team)
- [Modify team](#modify-team)
- [Modify team's agent options](#modify-teams-agent-options)
- [Delete team](#delete-team)

## List teams

_Available in Fleet Premium_

`GET /api/v1/fleet/teams`

#### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |
| order_key       | string  | query | What to order results by. Can be any column in the `teams` table.                                                             |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| query           | string  | query | Search query keywords. Searchable fields include `name`.                                                                      |

#### Example

`GET /api/v1/fleet/teams`

##### Default response

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

`GET /api/v1/fleet/teams/{id}`

#### Parameters

| Name | Type    | In   | Description                          |
| ---- | ------  | ---- | ------------------------------------ |
| id   | integer | path | **Required.** The desired team's ID. |

#### Example

`GET /api/v1/fleet/teams/1`

##### Default response

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
    "mdm": {
      "macos_updates": {
        "minimum_version": "12.3.1",
        "deadline": "2022-01-01"
      },
      "macos_settings": {
        "custom_settings": ["path/to/profile.mobileconfig"],
        "enable_disk_encryption": false
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

#### Parameters

| Name | Type   | In   | Description                    |
| ---- | ------ | ---- | ------------------------------ |
| name | string | body | **Required.** The team's name. |

#### Example

`POST /api/v1/fleet/teams`

##### Request body

```json
{
  "name": "workstations"
}
```

##### Default response

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

`PATCH /api/v1/fleet/teams/{id}`

#### Parameters

| Name                                                    | Type    | In   | Description                                                                                                                                                                                               |
| ------------------------------------------------------- | ------- | ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id                                                      | integer | path | **Required.** The desired team's ID.                                                                                                                                                                      |
| name                                                    | string  | body | The team's name.                                                                                                                                                                                          |
| host_ids                                                | list    | body | A list of hosts that belong to the team.                                                                                                                                                                  |
| user_ids                                                | list    | body | A list of users that are members of the team.                                                                                                                                                             |
| webhook_settings                                        | object  | body | Webhook settings contains for the team.                                                                                                                                                                   |
| &nbsp;&nbsp;failing_policies_webhook                    | object  | body | Failing policies webhook settings.                                                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies_webhook | boolean | body | Whether or not the failing policies webhook is enabled.                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;destination_url                 | string  | body | The URL to deliver the webhook requests to.                                                                                                                                                               |
| &nbsp;&nbsp;&nbsp;&nbsp;policy_ids                      | array   | body | List of policy IDs to enable failing policies webhook.                                                                                                                                                    |
| &nbsp;&nbsp;&nbsp;&nbsp;host_batch_size                 | integer | body | Maximum number of hosts to batch on failing policy webhook requests. The default, 0, means no batching (all hosts failing a policy are sent on one request).                                              |
| integrations                                            | object  | body | Integrations settings for the team. Note that integrations referenced here must already exist globally, created by a call to [Modify configuration](./Fleet-configuration.md#modify-configuration).                               |
| &nbsp;&nbsp;jira                                        | array   | body | Jira integrations configuration.                                                                                                                                                                          |
| &nbsp;&nbsp;&nbsp;&nbsp;url                             | string  | body | The URL of the Jira server to use.                                                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;project_key                     | string  | body | The project key of the Jira integration to use. Jira tickets will be created in this project.                                                                                                             |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies         | boolean | body | Whether or not that Jira integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies).    |
| &nbsp;&nbsp;zendesk                                     | array   | body | Zendesk integrations configuration.                                                                                                                                                                       |
| &nbsp;&nbsp;&nbsp;&nbsp;url                             | string  | body | The URL of the Zendesk server to use.                                                                                                                                                                     |
| &nbsp;&nbsp;&nbsp;&nbsp;group_id                        | integer | body | The Zendesk group id to use. Zendesk tickets will be created in this group.                                                                                                                               |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies         | boolean | body | Whether or not that Zendesk integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| mdm                                                     | object  | body | MDM settings for the team.                                                                                                                                                                                |
| &nbsp;&nbsp;macos_updates                               | object  | body | MacOS updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;minimum_version                 | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM will be nudged until their macOS is at or above this version.                                                                            |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline                        | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past.                                                                    |
| &nbsp;&nbsp;macos_settings                              | object  | body | MacOS-specific settings.                                                                                                                                                                                  |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_disk_encryption          | boolean | body | Hosts that belong to this team and are enrolled into Fleet's MDM will have disk encryption enabled if set to true.                                                                                        |
| &nbsp;&nbsp;macos_setup                                 | object  | body | Setup for automatic MDM enrollment of macOS devices.                                                                                                                                                                                  |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_end_user_authentication          | boolean | body | If set to true, end user authentication will be required during automatic MDM enrollment of new macOS devices. Settings for your IdP provider must also be [configured](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#end-user-authentication).                                                                                      |


#### Example (add users to a team)

`PATCH /api/v1/fleet/teams/1/users`

##### Request body

```json
{
  "user_ids": [1, 17, 22, 32]
}
```

##### Default response

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
    "mdm": {
      "macos_updates": {
        "minimum_version": "12.3.1",
        "deadline": "2022-01-01"
      },
      "macos_settings": {
        "custom_settings": ["path/to/profile.mobileconfig"],
        "enable_disk_encryption": false
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

#### Example (transfer hosts to a team)

`PATCH /api/v1/fleet/teams/1`

##### Request body

```json
{
  "host_ids": [3, 6, 7, 8, 9, 20, 32, 44]
}
```

##### Default response

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

## Modify team's agent options

_Available in Fleet Premium_

`POST /api/v1/fleet/teams/{id}/agent_options`

#### Parameters

| Name                             | Type    | In    | Description                                                                                                                                                  |
| ---                              | ---     | ---   | ---                                                                                                                                                          |
| id                               | integer | path  | **Required.** The desired team's ID.                                                                                                                         |
| force                            | bool    | query | Force apply the options even if there are validation errors.                                                                                                 |
| dry_run                          | bool    | query | Validate the options and return any validation errors, but do not apply the changes.                                                                         |
| _JSON data_                      | object  | body  | The JSON to use as agent options for this team. See [Agent options](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options) for details.                              |

#### Example

`POST /api/v1/fleet/teams/1/agent_options`

##### Request body

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

##### Default response

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

`DELETE /api/v1/fleet/teams/{id}`

#### Parameters

| Name | Type    | In   | Description                          |
| ---- | ------  | ---- | ------------------------------------ |
| id   | integer | path | **Required.** The desired team's ID. |

#### Example

`DELETE /api/v1/fleet/teams/1`

#### Default response

`Status: 200`

<meta name="description" value="Learn how to list, create, modify, and delete teams with Fleet's REST API.">
<meta name="pageOrderInSection" value="1500">