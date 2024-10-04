# Labels

## Add label

Add a dynamic or manual label.

`POST /api/v1/fleet/labels`

### Parameters

| Name        | Type   | In   | Description                                                                                                                                                                                                                                  |
| ----------- | ------ | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| name        | string | body | **Required**. The label's name.                                                                                                                                                                                                              |
| description | string | body | The label's description.                                                                                                                                                                                                                     |
| query       | string | body | The query in SQL syntax used to filter the hosts. Only one of either `query` (to create a dynamic label) or `hosts` (to create a manual label) can be included in the request.  |
| hosts       | array | body | The list of host identifiers (`hardware_serial`, `uuid`, `osquery_host_id`, `hostname`, or `name`) the label will apply to. Only one of either `query` (to create a dynamic label) or `hosts` (to create a manual label)  can be included in the request. |
| platform    | string | body | The specific platform for the label to target. Provides an additional filter. Choices for platform are `darwin`, `windows`, `ubuntu`, and `centos`. All platforms are included by default and this option is represented by an empty string. |

If both `query` and `hosts` aren't specified, a manual label with no hosts will be created.

### Example

`POST /api/v1/fleet/labels`

#### Request body

```json
{
  "name": "Ubuntu hosts",
  "description": "Filters ubuntu hosts",
  "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
  "platform": ""
}
```

#### Default response

`Status: 200`

```json
{
  "label": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 1,
    "name": "Ubuntu hosts",
    "description": "Filters ubuntu hosts",
    "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
    "label_type": "regular",
    "label_membership_type": "dynamic",
    "display_text": "Ubuntu hosts",
    "count": 0,
    "host_ids": null
  }
}
```

## Update label

Updates the specified label. Note: Label queries and platforms are immutable. To change these, you must delete the label and create a new label.

`PATCH /api/v1/fleet/labels/:id`

### Parameters

| Name        | Type    | In   | Description                   |
| ----------- | ------- | ---- | ----------------------------- |
| id          | integer | path | **Required**. The label's id. |
| name        | string  | body | The label's name.             |
| description | string  | body | The label's description.      |
| hosts       | array   | body | If updating a manual label: the list of host identifiers (`hardware_serial`, `uuid`, `osquery_host_id`, `hostname`, or `name`) the label will apply to. |


### Example

`PATCH /api/v1/fleet/labels/1`

#### Request body

```json
{
  "name": "macOS label",
  "description": "Now this label only includes macOS machines",
  "platform": "darwin"
}
```

#### Default response

`Status: 200`

```json
{
  "label": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 1,
    "name": "Ubuntu hosts",
    "description": "Filters ubuntu hosts",
    "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
    "platform": "darwin",
    "label_type": "regular",
    "label_membership_type": "dynamic",
    "display_text": "Ubuntu hosts",
    "count": 0,
    "host_ids": null
  }
}
```

## Get label

Returns the specified label.

`GET /api/v1/fleet/labels/:id`

### Parameters

| Name | Type    | In   | Description                   |
| ---- | ------- | ---- | ----------------------------- |
| id   | integer | path | **Required**. The label's id. |

### Example

`GET /api/v1/fleet/labels/1`

#### Default response

`Status: 200`

```json
{
  "label": {
    "created_at": "2021-02-09T22:09:43Z",
    "updated_at": "2021-02-09T22:15:58Z",
    "id": 12,
    "name": "Ubuntu",
    "description": "Filters ubuntu hosts",
    "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
    "label_type": "regular",
    "label_membership_type": "dynamic",
    "display_text": "Ubuntu",
    "count": 0,
    "host_ids": null
  }
}
```

## Get labels summary

Returns a list of all the labels in Fleet.

`GET /api/v1/fleet/labels/summary`

### Example

`GET /api/v1/fleet/labels/summary`

#### Default response

`Status: 200`

```json
{
  "labels": [
    {
      "id": 6,
      "name": "All Hosts",
      "description": "All hosts which have enrolled in Fleet",
      "label_type": "builtin"
    },
    {
      "id": 7,
      "name": "macOS",
      "description": "All macOS hosts",
      "label_type": "builtin"
    },
    {
      "id": 8,
      "name": "Ubuntu Linux",
      "description": "All Ubuntu hosts",
      "label_type": "builtin"
    },
    {
      "id": 9,
      "name": "CentOS Linux",
      "description": "All CentOS hosts",
      "label_type": "builtin"
    },
    {
      "id": 10,
      "name": "MS Windows",
      "description": "All Windows hosts",
      "label_type": "builtin"
    }
  ]
}
```

## List labels

Returns a list of all the labels in Fleet.

`GET /api/v1/fleet/labels`

### Parameters

| Name            | Type    | In    | Description   |
| --------------- | ------- | ----- |------------------------------------- |
| order_key       | string  | query | What to order results by. Can be any column in the labels table.                                                  |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

### Example

`GET /api/v1/fleet/labels`

#### Default response

`Status: 200`

```json
{
  "labels": [
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 6,
      "name": "All Hosts",
      "description": "All hosts which have enrolled in Fleet",
      "query": "SELECT 1;",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 7,
      "display_text": "All Hosts",
      "count": 7,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 7,
      "name": "macOS",
      "description": "All macOS hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'darwin';",
      "platform": "darwin",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 1,
      "display_text": "macOS",
      "count": 1,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 8,
      "name": "Ubuntu Linux",
      "description": "All Ubuntu hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
      "platform": "ubuntu",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 3,
      "display_text": "Ubuntu Linux",
      "count": 3,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 9,
      "name": "CentOS Linux",
      "description": "All CentOS hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'centos' OR name LIKE '%centos%'",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 3,
      "display_text": "CentOS Linux",
      "count": 3,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 10,
      "name": "MS Windows",
      "description": "All Windows hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'windows';",
      "platform": "windows",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "display_text": "MS Windows",
      "count": 0,
      "host_ids": null
    }
  ]
}
```

## List hosts in a label

Returns a list of the hosts that belong to the specified label.

`GET /api/v1/fleet/labels/:id/hosts`

### Parameters

| Name                     | Type    | In    | Description                                                                                                                                                                                                                |
| ---------------          | ------- | ----- | -----------------------------------------------------------------------------------------------------------------------------                                                                                              |
| id                       | integer | path  | **Required**. The label's id.                                                                                                                                                                                              |
| page                     | integer | query | Page number of the results to fetch.                                                                                                                                                                                       |
| per_page                 | integer | query | Results per page.                                                                                                                                                                                                          |
| order_key                | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                            |
| order_direction          | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include 'asc' and 'desc'. Default is 'asc'.                                                                                              |
| after                    | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used.                                                                                                                   |
| status                   | string  | query | Indicates the status of the hosts to return. Can either be 'new', 'online', 'offline', 'mia' or 'missing'.                                                                                                                 |
| query                    | string  | query | Search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, and `ipv4`.                                                                                                                         |
| team_id                  | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts in the specified team.                                                                                                                                |
| disable_failing_policies | boolean | query | If "true", hosts will return failing policies as 0 regardless of whether there are any that failed for the host. This is meant to be used when increased performance is needed in exchange for the extra information.      |
| mdm_id                   | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).      |
| mdm_name                 | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).      |
| mdm_enrollment_status    | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Valid options are 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| macos_settings           | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| low_disk_space           | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                 |
| macos_settings_disk_encryption | string | query | Filters the hosts by the status of the macOS disk encryption MDM profile on the host. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'. |
| bootstrap_package       | string | query | _Available in Fleet Premium_. Filters the hosts by the status of the MDM bootstrap package on the host. Valid options are 'installed', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings          | string  | query | Filters the hosts by the status of the operating system settings applied to the hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings_disk_encryption | string | query | Filters the hosts by the status of the disk encryption setting applied to the hosts. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'.  **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |

If `mdm_id`, `mdm_name`, `mdm_enrollment_status`, `os_settings`, or `os_settings_disk_encryption` is specified, then Windows Servers are excluded from the results.

### Example

`GET /api/v1/fleet/labels/6/hosts&query=floobar`

#### Default response

`Status: 200`

```json
{
  "hosts": [
    {
      "created_at": "2021-02-03T16:11:43Z",
      "updated_at": "2021-02-03T21:58:19Z",
      "id": 2,
      "detail_updated_at": "2021-02-03T21:58:10Z",
      "label_updated_at": "2021-02-03T21:58:10Z",
      "policy_updated_at": "2023-06-26T18:33:15Z",
      "last_enrolled_at": "2021-02-03T16:11:43Z",
      "software_updated_at": "2020-11-05T05:09:44Z",
      "seen_time": "2021-02-03T21:58:20Z",
      "refetch_requested": false,
      "hostname": "floobar42",
      "uuid": "a2064cef-0000-0000-afb9-283e3c1d487e",
      "platform": "ubuntu",
      "osquery_version": "4.5.1",
      "os_version": "Ubuntu 20.4.0",
      "build": "",
      "platform_like": "debian",
      "code_name": "",
      "uptime": 32688000000000,
      "memory": 2086899712,
      "cpu_type": "x86_64",
      "cpu_subtype": "142",
      "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
      "cpu_physical_cores": 4,
      "cpu_logical_cores": 4,
      "hardware_vendor": "",
      "hardware_model": "",
      "hardware_version": "",
      "hardware_serial": "",
      "computer_name": "e2e7f8d8983d",
      "display_name": "e2e7f8d8983d",
      "primary_ip": "172.20.0.2",
      "primary_mac": "02:42:ac:14:00:02",
      "distributed_interval": 10,
      "config_tls_refresh": 10,
      "logger_tls_period": 10,
      "team_id": null,
      "pack_stats": null,
      "team_name": null,
      "status": "offline",
      "display_text": "e2e7f8d8983d",
      "mdm": {
        "encryption_key_available": false,
        "enrollment_status": null,
        "name": "",
        "server_url": null
      }
    }
  ]
}
```

## Delete label

Deletes the label specified by name.

`DELETE /api/v1/fleet/labels/:name`

### Parameters

| Name | Type   | In   | Description                     |
| ---- | ------ | ---- | ------------------------------- |
| name | string | path | **Required**. The label's name. |

### Example

`DELETE /api/v1/fleet/labels/ubuntu_label`

#### Default response

`Status: 200`


## Delete label by ID

Deletes the label specified by ID.

`DELETE /api/v1/fleet/labels/id/:id`

### Parameters

| Name | Type    | In   | Description                   |
| ---- | ------- | ---- | ----------------------------- |
| id   | integer | path | **Required**. The label's id. |

### Example

`DELETE /api/v1/fleet/labels/id/13`

#### Default response

`Status: 200`


---

<meta name="description" value="Documentation for Fleet's labels REST API endpoints.">
<meta name="pageOrderInSection" value="90">