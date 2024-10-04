# Hosts


## List hosts

`GET /api/v1/fleet/hosts`

### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                                                                        |
| per_page                | integer | query | Results per page.                                                                                                                                                                                                                                                                                                                           |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                                                             |
| after                   | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used. **Note:** Use `page` instead of `after`                                                                                                                                                                                                                                    |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include 'asc' and 'desc'. Default is 'asc'.                                                                                                                                                                                                               |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be 'new', 'online', 'offline', 'mia' or 'missing'.                                                                                                                                                                                                                                  |
| query                   | string  | query | Search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, `ipv4` and the hosts' email addresses (only searched if the query looks like an email address, i.e. contains an '@', no space, etc.).                                                                                                                |
| additional_info_filters | string  | query | A comma-delimited list of fields to include in each host's `additional` object.                                              |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters to only include hosts in the specified team. Use `0` to filter by hosts assigned to "No team".                                                                                                                                                                                                                                                |
| policy_id               | integer | query | The ID of the policy to filter hosts by.                                                                                                                                                                                                                                                                                                    |
| policy_response         | string  | query | **Requires `policy_id`**. Valid options are 'passing' or 'failing'.                                                                                                                                                                                                                                       |
| software_version_id     | integer | query | The ID of the software version to filter hosts by.                                                                                                                                                                                                                                                                                                  |
| software_title_id       | integer | query | The ID of the software title to filter hosts by.                                                                                                                                                                                                                                                                                                  |
| software_status       | string | query | The status of the software install to filter hosts by.                                                                                                                                                                                                                                                                                                  |
| os_version_id | integer | query | The ID of the operating system version to filter hosts by. |
| os_name                 | string  | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                                                                                                                                                                                                     |
| os_version              | string  | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                                                                                                                                                                                                  |
| vulnerability           | string  | query | The cve to filter hosts by (including "cve-" prefix, case-insensitive).                                                                                                                                                                                                                                                                     |
| device_mapping          | boolean | query | Indicates whether `device_mapping` should be included for each host. See ["Get host's Google Chrome profiles](#get-hosts-google-chrome-profiles) for more information about this feature.                                                                                                                                                  |
| mdm_id                  | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).                                                                                                                                                                                                |
| mdm_name                | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).                                                                                                                                                                                                |
| mdm_enrollment_status   | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Valid options are 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| connected_to_fleet   | boolean  | query | Filter hosts that are talking to this Fleet server for MDM features. In rare cases, hosts can be enrolled to one Fleet server but talk to a different Fleet server for MDM features. In this case, the value would be `false`. Always `false` for Linux hosts.                                                                                                                           |
| macos_settings          | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| munki_issue_id          | integer | query | The ID of the _munki issue_ (a Munki-reported error or warning message) to filter hosts by (that is, filter hosts that are affected by that corresponding error or warning message).                                                                                                                                                        |
| low_disk_space          | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                                                                                                                                  |
| disable_failing_policies| boolean | query | If `true`, hosts will return failing policies as 0 regardless of whether there are any that failed for the host. This is meant to be used when increased performance is needed in exchange for the extra information.                                                                                                                       |
| macos_settings_disk_encryption | string | query | Filters the hosts by the status of the macOS disk encryption MDM profile on the host. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'. |
| bootstrap_package       | string | query | _Available in Fleet Premium_. Filters the hosts by the status of the MDM bootstrap package on the host. Valid options are 'installed', 'pending', or 'failed'. |
| os_settings          | string  | query | Filters the hosts by the status of the operating system settings applied to the hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings_disk_encryption | string | query | Filters the hosts by the status of the disk encryption setting applied to the hosts. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'.  **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| populate_software     | boolean | query | If `true`, the response will include a list of installed software for each host, including vulnerability data. |
| populate_policies     | boolean | query | If `true`, the response will include policy data for each host. |

> `software_id` is deprecated as of Fleet 4.42. It is maintained for backwards compatibility. Please use the `software_version_id` instead.

If `software_title_id` is specified, an additional top-level key `"software_title"` is returned with the software title object corresponding to the `software_title_id`. See [List software](#list-software) response payload for details about this object.

If `software_version_id` is specified, an additional top-level key `"software"` is returned with the software object corresponding to the `software_version_id`. See [List software versions](#list-software-versions) response payload for details about this object.

If `additional_info_filters` is not specified, no `additional` information will be returned.

If `mdm_id` is specified, an additional top-level key `"mobile_device_management_solution"` is returned with the information corresponding to the `mdm_id`.

If `mdm_id`, `mdm_name`, `mdm_enrollment_status`, `os_settings`, or `os_settings_disk_encryption` is specified, then Windows Servers are excluded from the results.

If `munki_issue_id` is specified, an additional top-level key `munki_issue` is returned with the information corresponding to the `munki_issue_id`.

If `after` is being used with `created_at` or `updated_at`, the table must be specified in `order_key`. Those columns become `h.created_at` and `h.updated_at`.

### Example

`GET /api/v1/fleet/hosts?page=0&per_page=100&order_key=hostname&query=2ce&populate_software=true&populate_policies=true`

#### Request query parameters

```json
{
  "page": 0,
  "per_page": 100,
  "order_key": "hostname"
}
```

#### Default response

`Status: 200`

```json
{
  "hosts": [
    {
      "created_at": "2020-11-05T05:09:44Z",
      "updated_at": "2020-11-05T06:03:39Z",
      "id": 1,
      "detail_updated_at": "2020-11-05T05:09:45Z",
      "last_restarted_at": "2020-11-01T03:01:45Z",
      "software_updated_at": "2020-11-05T05:09:44Z",
      "label_updated_at": "2020-11-05T05:14:51Z",
      "policy_updated_at": "2023-06-26T18:33:15Z",
      "last_enrolled_at": "2023-02-26T22:33:12Z",
      "seen_time": "2020-11-05T06:03:39Z",
      "hostname": "2ceca32fe484",
      "uuid": "392547dc-0000-0000-a87a-d701ff75bc65",
      "platform": "centos",
      "osquery_version": "2.7.0",
      "os_version": "CentOS Linux 7",
      "build": "",
      "platform_like": "rhel fedora",
      "code_name": "",
      "uptime": 8305000000000,
      "memory": 2084032512,
      "cpu_type": "6",
      "cpu_subtype": "142",
      "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
      "cpu_physical_cores": 4,
      "cpu_logical_cores": 4,
      "hardware_vendor": "",
      "hardware_model": "",
      "hardware_version": "",
      "hardware_serial": "",
      "computer_name": "2ceca32fe484",
      "display_name": "2ceca32fe484",
      "public_ip": "",
      "primary_ip": "",
      "primary_mac": "",
      "distributed_interval": 10,
      "config_tls_refresh": 10,
      "logger_tls_period": 8,
      "additional": {},
      "status": "offline",
      "display_text": "2ceca32fe484",
      "team_id": null,
      "team_name": null,
      "gigs_disk_space_available": 174.98,
      "percent_disk_space_available": 71,
      "gigs_total_disk_space": 246,
      "pack_stats": [
        {
          "pack_id": 0,
          "pack_name": "Global",
          "type": "global",
          "query_stats": [
            {
            "scheduled_query_name": "Get recently added or removed USB drives",
            "scheduled_query_id": 5535,
            "query_name": "Get recently added or removed USB drives",
            "discard_data": false,
            "last_fetched": null,
            "automations_enabled": false,
            "description": "Returns a record every time a USB device is plugged in or removed",
            "pack_name": "Global",
            "average_memory": 434176,
            "denylisted": false,
            "executions": 2,
            "interval": 86400,
            "last_executed": "2023-11-28T00:02:07Z",
            "output_size": 891,
            "system_time": 10,
            "user_time": 6,
            "wall_time": 0
            }
          ]
        }
      ],
      "issues": {
        "failing_policies_count": 1,
        "critical_vulnerabilities_count": 2, // Fleet Premium only
        "total_issues_count": 3
      },
      "geolocation": {
        "country_iso": "US",
        "city_name": "New York",
        "geometry": {
          "type": "point",
          "coordinates": [40.6799, -74.0028]
        }
      },
      "mdm": {
        "encryption_key_available": false,
        "enrollment_status": "Pending",
        "dep_profile_error": true,
        "name": "Fleet",
        "server_url": "https://example.fleetdm.com/mdm/apple/mdm"
      },
      "software": [
        {
          "id": 1,
          "name": "glibc",
          "version": "2.12",
          "source": "rpm_packages",
          "generated_cpe": "cpe:2.3:a:gnu:glibc:2.12:*:*:*:*:*:*:*",
          "vulnerabilities": [
            {
              "cve": "CVE-2009-5155",
              "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2009-5155",
              "cvss_score": 7.5, // Fleet Premium only
              "epss_probability": 0.01537, // Fleet Premium only
              "cisa_known_exploit": false, // Fleet Premium only
              "cve_published": "2022-01-01 12:32:00", // Fleet Premium only
              "cve_description": "In the GNU C Library (aka glibc or libc6) before 2.28, parse_reg_exp in posix/regcomp.c misparses alternatives, which allows attackers to cause a denial of service (assertion failure and application exit) or trigger an incorrect result by attempting a regular-expression match.", // Fleet Premium only
              "resolved_in_version": "2.28" // Fleet Premium only
            }
          ],
          "installed_paths": ["/usr/lib/some-path-1"]
        }
      ],
      "policies": [
        {
          "id": 1,
          "name": "Gatekeeper enabled",
          "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
          "description": "Checks if gatekeeper is enabled on macOS devices",
          "resolution": "Fix with these steps...",
          "platform": "darwin",
          "response": "fail",
          "critical": false
        }
      ]
    }
  ]
}
```

> Note: the response above assumes a [GeoIP database is configured](https://fleetdm.com/docs/deploying/configuration#geoip), otherwise the `geolocation` object won't be included.

Response payload with the `mdm_id` filter provided:

```json
{
  "hosts": [...],
  "mobile_device_management_solution": {
    "server_url": "http://some.url/mdm",
    "name": "MDM Vendor Name",
    "id": 999
  }
}
```

Response payload with the `munki_issue_id` filter provided:

```json
{
  "hosts": [...],
  "munki_issue": {
    "id": 1,
    "name": "Could not retrieve managed install primary manifest",
    "type": "error"
  }
}
```

## Count hosts

`GET /api/v1/fleet/hosts/count`

### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                                                             |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include 'asc' and 'desc'. Default is 'asc'.                                                                                                                                                                                                               |
| after                   | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used.                                                                                                                                                                                                                                    |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be 'new', 'online', 'offline', 'mia' or 'missing'.                                                                                                                                                                                                                                  |
| query                   | string  | query | Search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, `ipv4` and the hosts' email addresses (only searched if the query looks like an email address, i.e. contains an '@', no space, etc.).                                                                                                                |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts in the specified team.                                                                                                                                                                                                                                                 |
| policy_id               | integer | query | The ID of the policy to filter hosts by.                                                                                                                                                                                                                                                                                                    |
| policy_response         | string  | query | **Requires `policy_id`**. Valid options are 'passing' or 'failing'.                                                                                                                                                                                                                                       |
| software_version_id     | integer | query | The ID of the software version to filter hosts by.                                                                                                            |
| software_title_id       | integer | query | The ID of the software title to filter hosts by.                                                                                                              |
| os_version_id | integer | query | The ID of the operating system version to filter hosts by. |
| os_name                 | string  | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                                                                                                                                                                                                     |
| os_version              | string  | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                                                                                                                                                                                                  |
| vulnerability           | string  | query | The cve to filter hosts by (including "cve-" prefix, case-insensitive).                                                                                                                                                                                                                                                                     |
| label_id                | integer | query | A valid label ID. Can only be used in combination with `order_key`, `order_direction`, `after`, `status`, `query` and `team_id`.                                                                                                                                                                                                            |
| mdm_id                  | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).                                                                                                                                                                                                |
| mdm_name                | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).                                                                                                                                                                                                |
| mdm_enrollment_status   | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Valid options are 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| macos_settings          | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| munki_issue_id          | integer | query | The ID of the _munki issue_ (a Munki-reported error or warning message) to filter hosts by (that is, filter hosts that are affected by that corresponding error or warning message).                                                                                                                                                        |
| low_disk_space          | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                                                                                                                                  |
| macos_settings_disk_encryption | string | query | Filters the hosts by the status of the macOS disk encryption MDM profile on the host. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'. |
| bootstrap_package       | string | query | _Available in Fleet Premium_. Filters the hosts by the status of the MDM bootstrap package on the host. Valid options are 'installed', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings          | string  | query | Filters the hosts by the status of the operating system settings applied to the hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings_disk_encryption | string | query | Filters the hosts by the status of the disk encryption setting applied to the hosts. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'.  **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |

If `additional_info_filters` is not specified, no `additional` information will be returned.

If `mdm_id`, `mdm_name` or `mdm_enrollment_status` is specified, then Windows Servers are excluded from the results.

### Example

`GET /api/v1/fleet/hosts/count?page=0&per_page=100&order_key=hostname&query=2ce`

#### Request query parameters

```json
{
  "page": 0,
  "per_page": 100,
  "order_key": "hostname"
}
```

#### Default response

`Status: 200`

```json
{
  "count": 123
}
```

## Get hosts summary

Returns the count of all hosts organized by status. `online_count` includes all hosts currently enrolled in Fleet. `offline_count` includes all hosts that haven't checked into Fleet recently. `mia_count` includes all hosts that haven't been seen by Fleet in more than 30 days. `new_count` includes the hosts that have been enrolled to Fleet in the last 24 hours.

`GET /api/v1/fleet/host_summary`

### Parameters

| Name            | Type    | In    | Description                                                                     |
| --------------- | ------- | ----  | ------------------------------------------------------------------------------- |
| team_id         | integer | query | _Available in Fleet Premium_. The ID of the team whose host counts should be included. Defaults to all teams. |
| platform        | string  | query | Platform to filter by when counting. Defaults to all platforms.                 |
| low_disk_space  | integer | query | _Available in Fleet Premium_. Returns the count of hosts with less GB of disk space available than this value. Must be a number between 1-100. |

### Example

`GET /api/v1/fleet/host_summary?team_id=1&low_disk_space=32`

#### Default response

`Status: 200`

```json
{
  "team_id": 1,
  "totals_hosts_count": 2408,
  "online_count": 2267,
  "offline_count": 141,
  "mia_count": 0,
  "missing_30_days_count": 0,
  "new_count": 0,
  "all_linux_count": 1204,
  "low_disk_space_count": 12,
  "builtin_labels": [
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
    },
    {
      "id": 11,
      "name": "Red Hat Linux",
      "description": "All Red Hat Enterprise Linux hosts",
      "label_type": "builtin"
    },
    {
      "id": 12,
      "name": "All Linux",
      "description": "All Linux distributions",
      "label_type": "builtin"
    },
    {
      "id": 13,
      "name": "iOS",
      "description": "All iOS hosts",
      "label_type": "builtin"
    },
    {
      "id": 14,
      "name": "iPadOS",
      "description": "All iPadOS hosts",
      "label_type": "builtin"
    }
  ],
  "platforms": [
    {
      "platform": "chrome",
      "hosts_count": 1234
    },
    {
      "platform": "darwin",
      "hosts_count": 1234
    },
    {
      "platform": "ios",
      "hosts_count": 1234
    },
    {
      "platform": "ipados",
      "hosts_count": 1234
    },
    {
      "platform": "rhel",
      "hosts_count": 1234
    },
    {
      "platform": "ubuntu",
      "hosts_count": 12044
    },
    {
      "platform": "windows",
      "hosts_count": 12044
    }

  ]
}
```

## Get host

Returns the information of the specified host.

`GET /api/v1/fleet/hosts/:id`

### Parameters

| Name             | Type    | In    | Description                                                                         |
|------------------|---------|-------|-------------------------------------------------------------------------------------|
| id               | integer | path  | **Required**. The host's id.                                                        |
| exclude_software | boolean | query | If `true`, the response will not include a list of installed software for the host. |

### Example

`GET /api/v1/fleet/hosts/121`

#### Default response

`Status: 200`

```json
{
  "host": {
    "created_at": "2021-08-19T02:02:22Z",
    "updated_at": "2021-08-19T21:14:58Z",
    "software": [
      {
        "id": 408,
        "name": "osquery",
        "version": "4.5.1",
        "source": "rpm_packages",
        "browser": "",
        "generated_cpe": "",
        "vulnerabilities": null,
        "installed_paths": ["/usr/lib/some-path-1"]
      },
      {
        "id": 1146,
        "name": "tar",
        "version": "1.30",
        "source": "rpm_packages",
        "browser": "",
        "generated_cpe": "",
        "vulnerabilities": null
      },
      {
        "id": 321,
        "name": "SomeApp.app",
        "version": "1.0",
        "source": "apps",
        "browser": "",
        "bundle_identifier": "com.some.app",
        "last_opened_at": "2021-08-18T21:14:00Z",
        "generated_cpe": "",
        "vulnerabilities": null,
        "installed_paths": ["/usr/lib/some-path-2"]
      }
    ],
    "id": 1,
    "detail_updated_at": "2021-08-19T21:07:53Z",
    "last_restarted_at": "2020-11-01T03:01:45Z",
    "software_updated_at": "2020-11-05T05:09:44Z",
    "label_updated_at": "2021-08-19T21:07:53Z",
    "policy_updated_at": "2023-06-26T18:33:15Z",
    "last_enrolled_at": "2021-08-19T02:02:22Z",
    "seen_time": "2021-08-19T21:14:58Z",
    "refetch_requested": false,
    "hostname": "23cfc9caacf0",
    "uuid": "309a4b7d-0000-0000-8e7f-26ae0815ede8",
    "platform": "rhel",
    "osquery_version": "5.12.0",
    "orbit_version": "1.22.0",
    "fleet_desktop_version": "1.22.0",
    "scripts_enabled": true,
    "os_version": "CentOS Linux 8.3.2011",
    "build": "",
    "platform_like": "rhel",
    "code_name": "",
    "uptime": 210671000000000,
    "memory": 16788398080,
    "cpu_type": "x86_64",
    "cpu_subtype": "158",
    "cpu_brand": "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz",
    "cpu_physical_cores": 12,
    "cpu_logical_cores": 12,
    "hardware_vendor": "",
    "hardware_model": "",
    "hardware_version": "",
    "hardware_serial": "",
    "computer_name": "23cfc9caacf0",
    "display_name": "23cfc9caacf0",
    "public_ip": "",
    "primary_ip": "172.27.0.6",
    "primary_mac": "02:42:ac:1b:00:06",
    "distributed_interval": 10,
    "config_tls_refresh": 10,
    "logger_tls_period": 10,
    "team_id": null,
    "pack_stats": null,
    "team_name": null,
    "additional": {},
    "gigs_disk_space_available": 46.1,
    "percent_disk_space_available": 74,
    "gigs_total_disk_space": 160,
    "disk_encryption_enabled": true,
    "users": [
      {
        "uid": 0,
        "username": "root",
        "type": "",
        "groupname": "root",
        "shell": "/bin/bash"
      },
      {
        "uid": 1,
        "username": "bin",
        "type": "",
        "groupname": "bin",
        "shell": "/sbin/nologin"
      }
    ],
    "labels": [
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 6,
        "name": "All Hosts",
        "description": "All hosts which have enrolled in Fleet",
        "query": "SELECT 1;",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      },
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 9,
        "name": "CentOS Linux",
        "description": "All CentOS hosts",
        "query": "SELECT 1 FROM os_version WHERE platform = 'centos' OR name LIKE '%centos%'",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      },
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 12,
        "name": "All Linux",
        "description": "All Linux distributions",
        "query": "SELECT 1 FROM osquery_info WHERE build_platform LIKE '%ubuntu%' OR build_distro LIKE '%centos%';",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      }
    ],
    "packs": [],
    "status": "online",
    "display_text": "23cfc9caacf0",
    "policies": [
      {
        "id": 2,
        "name": "SomeQuery2",
        "query": "SELECT * FROM bar;",
        "description": "this is another query",
        "resolution": "fix with these other steps...",
        "platform": "darwin",
        "response": "fail",
        "critical": false
      },
      {
        "id": 3,
        "name": "SomeQuery3",
        "query": "SELECT * FROM baz;",
        "description": "",
        "resolution": "",
        "platform": "",
        "response": "",
        "critical": false
      },
      {
        "id": 1,
        "name": "SomeQuery",
        "query": "SELECT * FROM foo;",
        "description": "this is a query",
        "resolution": "fix with these steps...",
        "platform": "windows,linux",
        "response": "pass",
        "critical": false
      }
    ],
    "issues": {
        "failing_policies_count": 1,
        "critical_vulnerabilities_count": 2, // Fleet Premium only
        "total_issues_count": 3
    },
    "batteries": [
      {
        "cycle_count": 999,
        "health": "Normal"
      }
    ],
    "geolocation": {
      "country_iso": "US",
      "city_name": "New York",
      "geometry": {
        "type": "point",
        "coordinates": [40.6799, -74.0028]
      }
    },
    "maintenance_window": {
      "starts_at": "2024-06-18T13:27:18−04:00",
      "timezone": "America/New_York"
    },
    "mdm": {
      "encryption_key_available": true,
      "enrollment_status": "On (manual)",
      "name": "Fleet",
      "connected_to_fleet": true,
      "server_url": "https://acme.com/mdm/apple/mdm",
      "device_status": "unlocked",
      "pending_action": "",
      "macos_settings": {
        "disk_encryption": null,
        "action_required": null
      },
      "macos_setup": {
        "bootstrap_package_status": "installed",
        "detail": "",
        "bootstrap_package_name": "test.pkg"
      },
      "os_settings": {
        "disk_encryption": {
          "status": null,
          "detail": ""
        }
      },
      "profiles": [
        {
          "profile_uuid": "954ec5ea-a334-4825-87b3-937e7e381f24",
          "name": "profile1",
          "status": "verifying",
          "operation_type": "install",
          "detail": ""
        }
      ]
    }
  }
}
```

> Note: the response above assumes a [GeoIP database is configured](https://fleetdm.com/docs/deploying/configuration#geoip), otherwise the `geolocation` object won't be included.

> Note: `installed_paths` may be blank depending on installer package. For example, on Linux, RPM-installed packages do not provide installed path information.

> Note:
> - `orbit_version: null` means this agent is not a fleetd agent
> - `fleet_desktop_version: null` means this agent is not a fleetd agent, or this agent is version <=1.23.0 which is not collecting the desktop version
> - `fleet_desktop_version: ""` means this agent is a fleetd agent but does not have fleet desktop
> - `scripts_enabled: null` means this agent is not a fleetd agent, or this agent is version <=1.23.0 which is not collecting the scripts enabled info

## Get host by identifier

Returns the information of the host specified using the `hostname`, `uuid`, or `hardware_serial` as an identifier.

If `hostname` is specified when there is more than one host with the same hostname, the endpoint returns the first matching host. In Fleet, hostnames are fully qualified domain names (FQDNs). `hostname` (e.g. johns-macbook-air.local) is not the same as `display_name` (e.g. John's MacBook Air).

`GET /api/v1/fleet/hosts/identifier/:identifier`

### Parameters

| Name       | Type              | In   | Description                                                        |
| ---------- | ----------------- | ---- | ------------------------------------------------------------------ |
| identifier | string | path | **Required**. The host's `hostname`, `uuid`, or `hardware_serial`. |
| exclude_software | boolean | query | If `true`, the response will not include a list of installed software for the host. |

### Example

`GET /api/v1/fleet/hosts/identifier/392547dc-0000-0000-a87a-d701ff75bc65`

#### Default response

`Status: 200`

```json
{
  "host": {
    "created_at": "2022-02-10T02:29:13Z",
    "updated_at": "2022-10-14T17:07:11Z",
    "software": [
      {
        "id": 16923,
        "name": "Automat",
        "version": "0.8.0",
        "source": "python_packages",
        "browser": "",
        "generated_cpe": "",
        "vulnerabilities": null,
        "installed_paths": ["/usr/lib/some_path/"]
      }
    ],
    "id": 33,
    "detail_updated_at": "2022-10-14T17:07:12Z",
    "label_updated_at": "2022-10-14T17:07:12Z",
    "policy_updated_at": "2022-10-14T17:07:12Z",
    "last_enrolled_at": "2022-02-10T02:29:13Z",
    "software_updated_at": "2020-11-05T05:09:44Z",
    "seen_time": "2022-10-14T17:45:41Z",
    "refetch_requested": false,
    "hostname": "23cfc9caacf0",
    "uuid": "392547dc-0000-0000-a87a-d701ff75bc65",
    "platform": "ubuntu",
    "osquery_version": "5.5.1",
    "os_version": "Ubuntu 20.04.3 LTS",
    "build": "",
    "platform_like": "debian",
    "code_name": "focal",
    "uptime": 20807520000000000,
    "memory": 1024360448,
    "cpu_type": "x86_64",
    "cpu_subtype": "63",
    "cpu_brand": "DO-Regular",
    "cpu_physical_cores": 1,
    "cpu_logical_cores": 1,
    "hardware_vendor": "",
    "hardware_model": "",
    "hardware_version": "",
    "hardware_serial": "",
    "computer_name": "23cfc9caacf0",
    "public_ip": "",
    "primary_ip": "172.27.0.6",
    "primary_mac": "02:42:ac:1b:00:06",
    "distributed_interval": 10,
    "config_tls_refresh": 60,
    "logger_tls_period": 10,
    "team_id": 2,
    "pack_stats": [
      {
        "pack_id": 1,
        "pack_name": "Global",
        "type": "global",
        "query_stats": [
          {
            "scheduled_query_name": "Get running processes (with user_name)",
            "scheduled_query_id": 49,
            "query_name": "Get running processes (with user_name)",
            "pack_name": "Global",
            "pack_id": 1,
            "average_memory": 260000,
            "denylisted": false,
            "executions": 1,
            "interval": 86400,
            "last_executed": "2022-10-14T10:00:01Z",
            "output_size": 198,
            "system_time": 20,
            "user_time": 80,
            "wall_time": 0
          }
        ]
      }
    ],
    "team_name": null,
    "gigs_disk_space_available": 19.29,
    "percent_disk_space_available": 74,
    "gigs_total_disk_space": 192,
    "issues": {
        "failing_policies_count": 1,
        "critical_vulnerabilities_count": 2, // Fleet Premium only
        "total_issues_count": 3
    },
    "labels": [
      {
        "created_at": "2021-09-14T05:11:02Z",
        "updated_at": "2021-09-14T05:11:02Z",
        "id": 12,
        "name": "All Linux",
        "description": "All Linux distributions",
        "query": "SELECT 1 FROM osquery_info WHERE build_platform LIKE '%ubuntu%' OR build_distro LIKE '%centos%';",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      }
    ],
    "packs": [
      {
        "created_at": "2021-09-17T05:28:54Z",
        "updated_at": "2021-09-17T05:28:54Z",
        "id": 1,
        "name": "Global",
        "description": "Global pack",
        "disabled": false,
        "type": "global",
        "labels": null,
        "label_ids": null,
        "hosts": null,
        "host_ids": null,
        "teams": null,
        "team_ids": null
      }
    ],
    "policies": [
      {
        "id": 142,
        "name": "Full disk encryption enabled (macOS)",
        "query": "SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT '' AND filevault_status = 'on' LIMIT 1;",
        "description": "Checks to make sure that full disk encryption (FileVault) is enabled on macOS devices.",
        "author_id": 31,
        "author_name": "",
        "author_email": "",
        "team_id": null,
        "resolution": "To enable full disk encryption, on the failing device, select System Preferences > Security & Privacy > FileVault > Turn On FileVault.",
        "platform": "darwin,linux",
        "created_at": "2022-09-02T18:52:19Z",
        "updated_at": "2022-09-02T18:52:19Z",
        "response": "fail",
        "critical": false
      }
    ],
    "batteries": [
      {
        "cycle_count": 999,
        "health": "Normal"
      }
    ],
    "geolocation": {
      "country_iso": "US",
      "city_name": "New York",
      "geometry": {
        "type": "point",
        "coordinates": [40.6799, -74.0028]
      }
    },
    "status": "online",
    "display_text": "dogfood-ubuntu-box",
    "display_name": "dogfood-ubuntu-box",
    "mdm": {
      "encryption_key_available": false,
      "enrollment_status": null,
      "name": "",
      "server_url": null,
      "device_status": "unlocked",
      "pending_action": "lock",
      "macos_settings": {
        "disk_encryption": null,
        "action_required": null
      },
      "macos_setup": {
        "bootstrap_package_status": "installed",
        "detail": ""
      },
      "os_settings": {
        "disk_encryption": {
          "status": null,
          "detail": ""
        }
      },
      "profiles": [
        {
          "profile_uuid": "954ec5ea-a334-4825-87b3-937e7e381f24",
          "name": "profile1",
          "status": "verifying",
          "operation_type": "install",
          "detail": ""
        }
      ]
    }
  }
}
```

> Note: the response above assumes a [GeoIP database is configured](https://fleetdm.com/docs/deploying/configuration#geoip), otherwise the `geolocation` object won't be included.

> Note: `installed_paths` may be blank depending on installer package. For example, on Linux, RPM-installed packages do not provide installed path information.

### Get host by device token

Returns a subset of information about the host specified by `token`. To get all information about a host, use the "Get host" endpoint [here](#get-host).

This is the API route used by the **My device** page in Fleet desktop to display information about the host to the end user.

`GET /api/v1/fleet/device/:token`

#### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

#### Example

`GET /api/v1/fleet/device/abcdef012456789`

#### Default response

`Status: 200`

```json
{
  "host": {
    "created_at": "2021-08-19T02:02:22Z",
    "updated_at": "2021-08-19T21:14:58Z",
    "software": [
      {
        "id": 408,
        "name": "osquery",
        "version": "4.5.1",
        "source": "rpm_packages",
        "browser": "",
        "generated_cpe": "",
        "vulnerabilities": null
      },
      {
        "id": 1146,
        "name": "tar",
        "version": "1.30",
        "source": "rpm_packages",
        "browser": "",
        "generated_cpe": "",
        "vulnerabilities": null
      },
      {
        "id": 321,
        "name": "SomeApp.app",
        "version": "1.0",
        "source": "apps",
        "browser": "",
        "bundle_identifier": "com.some.app",
        "last_opened_at": "2021-08-18T21:14:00Z",
        "generated_cpe": "",
        "vulnerabilities": null
      }
    ],
    "id": 1,
    "detail_updated_at": "2021-08-19T21:07:53Z",
    "label_updated_at": "2021-08-19T21:07:53Z",
    "last_enrolled_at": "2021-08-19T02:02:22Z",
    "seen_time": "2021-08-19T21:14:58Z",
    "refetch_requested": false,
    "hostname": "23cfc9caacf0",
    "uuid": "309a4b7d-0000-0000-8e7f-26ae0815ede8",
    "platform": "rhel",
    "osquery_version": "4.5.1",
    "os_version": "CentOS Linux 8.3.2011",
    "build": "",
    "platform_like": "rhel",
    "code_name": "",
    "uptime": 210671000000000,
    "memory": 16788398080,
    "cpu_type": "x86_64",
    "cpu_subtype": "158",
    "cpu_brand": "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz",
    "cpu_physical_cores": 12,
    "cpu_logical_cores": 12,
    "hardware_vendor": "",
    "hardware_model": "",
    "hardware_version": "",
    "hardware_serial": "",
    "computer_name": "23cfc9caacf0",
    "display_name": "23cfc9caacf0",
    "public_ip": "",
    "primary_ip": "172.27.0.6",
    "primary_mac": "02:42:ac:1b:00:06",
    "distributed_interval": 10,
    "config_tls_refresh": 10,
    "logger_tls_period": 10,
    "team_id": null,
    "pack_stats": null,
    "team_name": null,
    "additional": {},
    "gigs_disk_space_available": 46.1,
    "percent_disk_space_available": 74,
    "gigs_total_disk_space": 160,
    "disk_encryption_enabled": true,
    "dep_assigned_to_fleet": false,
    "users": [
      {
        "uid": 0,
        "username": "root",
        "type": "",
        "groupname": "root",
        "shell": "/bin/bash"
      },
      {
        "uid": 1,
        "username": "bin",
        "type": "",
        "groupname": "bin",
        "shell": "/sbin/nologin"
      }
    ],
    "labels": [
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 6,
        "name": "All Hosts",
        "description": "All hosts which have enrolled in Fleet",
        "query": "SELECT 1;",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      },
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 9,
        "name": "CentOS Linux",
        "description": "All CentOS hosts",
        "query": "SELECT 1 FROM os_version WHERE platform = 'centos' OR name LIKE '%centos%'",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      },
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 12,
        "name": "All Linux",
        "description": "All Linux distributions",
        "query": "SELECT 1 FROM osquery_info WHERE build_platform LIKE '%ubuntu%' OR build_distro LIKE '%centos%';",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      }
    ],
    "packs": [],
    "status": "online",
    "display_text": "23cfc9caacf0",
    "batteries": [
      {
        "cycle_count": 999,
        "health": "Good"
      }
    ],
    "mdm": {
      "encryption_key_available": true,
      "enrollment_status": "On (manual)",
      "name": "Fleet",
      "connected_to_fleet": true,
      "server_url": "https://acme.com/mdm/apple/mdm",
      "macos_settings": {
        "disk_encryption": null,
        "action_required": null
      },
      "macos_setup": {
        "bootstrap_package_status": "installed",
        "detail": "",
        "bootstrap_package_name": "test.pkg"
      },
      "os_settings": {
        "disk_encryption": {
          "status": null,
          "detail": ""
        }
      },
      "profiles": [
        {
          "profile_uuid": "954ec5ea-a334-4825-87b3-937e7e381f24",
          "name": "profile1",
          "status": "verifying",
          "operation_type": "install",
          "detail": ""
        }
      ]
    }
  },
  "self_service": true,
  "org_logo_url": "https://example.com/logo.jpg",
  "license": {
    "tier": "free",
    "expiration": "2031-01-01T00:00:00Z"
  },
  "global_config": {
    "mdm": {
      "enabled_and_configured": false
    }
  }
}
```

## Delete host

Deletes the specified host from Fleet. Note that a deleted host will fail authentication with the previous node key, and in most osquery configurations will attempt to re-enroll automatically. If the host still has a valid enroll secret, it will re-enroll successfully.

`DELETE /api/v1/fleet/hosts/:id`

### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's id. |

### Example

`DELETE /api/v1/fleet/hosts/121`

#### Default response

`Status: 200`


## Refetch host

Flags the host details, labels and policies to be refetched the next time the host checks in for distributed queries. Note that we cannot be certain when the host will actually check in and update the query results. Further requests to the host APIs will indicate that the refetch has been requested through the `refetch_requested` field on the host object.

`POST /api/v1/fleet/hosts/:id/refetch`

### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's id. |

### Example

`POST /api/v1/fleet/hosts/121/refetch`

#### Default response

`Status: 200`


## Transfer hosts to a team

_Available in Fleet Premium_

`POST /api/v1/fleet/hosts/transfer`

### Parameters

| Name    | Type    | In   | Description                                                             |
| ------- | ------- | ---- | ----------------------------------------------------------------------- |
| team_id | integer | body | **Required**. The ID of the team you'd like to transfer the host(s) to. |
| hosts   | array   | body | **Required**. A list of host IDs.                                       |

### Example

`POST /api/v1/fleet/hosts/transfer`

#### Request body

```json
{
  "team_id": 1,
  "hosts": [3, 2, 4, 6, 1, 5, 7]
}
```

#### Default response

`Status: 200`


## Transfer hosts to a team by filter

_Available in Fleet Premium_

`POST /api/v1/fleet/hosts/transfer/filter`

### Parameters

| Name    | Type    | In   | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| team_id | integer | body | **Required**. The ID of the team you'd like to transfer the host(s) to.                                                                                                                                                                                                                                                            |
| filters | object  | body | **Required** Contains any of the following four properties: `query` for search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, and `ipv4`. `status` to indicate the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`. `label_id` to indicate the selected label. `team_id` to indicate the selected team. Note: `label_id` and `status` cannot be used at the same time. |

### Example

`POST /api/v1/fleet/hosts/transfer/filter`

#### Request body

```json
{
  "team_id": 1,
  "filters": {
    "status": "online",
    "team_id": 2,
  }
}
```

#### Default response

`Status: 200`


## Turn off MDM for a host

`DELETE /api/v1/fleet/hosts/:id/mdm`

### Parameters

| Name | Type    | In   | Description                           |
| ---- | ------- | ---- | ------------------------------------- |
| id   | integer | path | **Required.** The host's ID in Fleet. |

### Example

`DELETE /api/v1/fleet/hosts/42/mdm`

#### Default response

`Status: 200`


## Bulk delete hosts by filter or ids

`POST /api/v1/fleet/hosts/delete`

### Parameters

| Name    | Type    | In   | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ids     | array   | body | A list of the host IDs you'd like to delete. If `ids` is specified, `filters` cannot be specified.                                                                                                                                                                                                                                                           |
| filters | object  | body | Contains any of the following four properties: `query` for search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, and `ipv4`. `status` to indicate the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`. `label_id` to indicate the selected label. `team_id` to indicate the selected team. If `filters` is specified, `id` cannot be specified. `label_id` and `status` cannot be used at the same time. |

Either ids or filters are required.

Request (`ids` is specified):

```json
{
  "ids": [1]
}
```

Request (`filters` is specified):
```json
{
  "filters": {
    "status": "online",
    "label_id": 1,
    "team_id": 1,
    "query": "abc"
  }
}
```

Request (`filters` is specified and empty, to delete all hosts):
```json
{
  "filters": {}
}
```

### Example

`POST /api/v1/fleet/hosts/delete`

#### Request body

```json
{
  "filters": {
    "status": "online",
    "team_id": 1
  }
}
```

#### Default response

`Status: 200`

## Get human-device mapping

Returns the end user's email(s) they use to log in to their Identity Provider (IdP) and Google Chrome profile.

Also returns the custom email that's set via the `PUT /api/v1/fleet/hosts/:id/device_mapping` endpoint (docs [here](#update-custom-human-device-mapping))

Note that IdP email is only supported on macOS hosts. It's collected once, during automatic enrollment (DEP), only if the end user authenticates with the IdP and the DEP profile has `await_device_configured` set to `true`.

`GET /api/v1/fleet/hosts/:id/device_mapping`

### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id         | integer           | path | **Required**. The host's `id`.                                                |

### Example

`GET /api/v1/fleet/hosts/1/device_mapping`

#### Default response

`Status: 200`

```json
{
  "host_id": 1,
  "device_mapping": [
    {
      "email": "user@example.com",
      "source": "identity_provider"
    },
    {
      "email": "user@example.com",
      "source": "google_chrome_profiles"
    },
    {
      "email": "user@example.com",
      "source": "custom"
    }
  ]
}
```

---

## Update custom human-device mapping

`PUT /api/v1/fleet/hosts/:id/device_mapping`

Updates the email for the `custom` data source in the human-device mapping. This source can only have one email.

### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id         | integer           | path | **Required**. The host's `id`.                                                |
| email      | string            | body | **Required**. The custom email.                                               |

### Example

`PUT /api/v1/fleet/hosts/1/device_mapping`

#### Request body

```json
{
  "email": "user@example.com"
}
```

#### Default response

`Status: 200`

```json
{
  "host_id": 1,
  "device_mapping": [
    {
      "email": "user@example.com",
      "source": "identity_provider"
    },
    {
      "email": "user@example.com",
      "source": "google_chrome_profiles"
    },
    {
      "email": "user@example.com",
      "source": "custom"
    }
  ]
}
```

## Get host's device health report

Retrieves information about a single host's device health.

This report includes a subset of host vitals, and simplified policy and vulnerable software information. Data is cached to preserve performance. To get all up-to-date information about a host, use the "Get host" endpoint [here](#get-host).


`GET /api/v1/fleet/hosts/:id/health`

### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id         | integer           | path | **Required**. The host's `id`.                                                |

### Example

`GET /api/v1/fleet/hosts/1/health`

#### Default response

`Status: 200`

```json
{
  "host_id": 1,
  "health": {
    "updated_at": "2023-09-16T18:52:19Z",
    "os_version": "CentOS Linux 8.3.2011",
    "disk_encryption_enabled": true,
    "failing_policies_count": 1,
    "failing_critical_policies_count": 1, // Fleet Premium only
    "failing_policies": [
      {
        "id": 123,
        "name": "Google Chrome is up to date",
        "critical": true, // Fleet Premium only
        "resolution": "Follow the Update Google Chrome instructions here: https://support.google.com/chrome/answer/95414?sjid=6534253818042437614-NA"
      }
    ],
    "vulnerable_software": [
      {
        "id": 321,
        "name": "Firefox.app",
        "version": "116.0.3",
      }
    ]
  }
}
```

---

## Get host's mobile device management (MDM) information

Currently supports Windows and MacOS. On MacOS this requires the [macadmins osquery
extension](https://github.com/macadmins/osquery-extension) which comes bundled
in [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd).

Retrieves a host's MDM enrollment status and MDM server URL.

If the host exists but is not enrolled to an MDM server, then this API returns `null`.

`GET /api/v1/fleet/hosts/:id/mdm`

### Parameters

| Name    | Type    | In   | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ---- | -------------------------------------------------------------------------------- |
| id      | integer | path | **Required** The id of the host to get the details for                           |

### Example

`GET /api/v1/fleet/hosts/32/mdm`

#### Default response

`Status: 200`

```json
{
  "enrollment_status": "On (automatic)",
  "server_url": "some.mdm.com",
  "name": "Some MDM",
  "id": 3
}
```

---

## Get mobile device management (MDM) summary

Currently supports Windows and MacOS. On MacOS this requires the [macadmins osquery
extension](https://github.com/macadmins/osquery-extension) which comes bundled
in [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd).

Retrieves MDM enrollment summary. Windows servers are excluded from the aggregated data.

`GET /api/v1/fleet/hosts/summary/mdm`

### Parameters

| Name     | Type    | In    | Description                                                                                                                                                                                                                                                                                                                        |
| -------- | ------- | ----- | -------------------------------------------------------------------------------- |
| team_id  | integer | query | _Available in Fleet Premium_. Filter by team                                      |
| platform | string  | query | Filter by platform ("windows" or "darwin")                                       |

A `team_id` of `0` returns the statistics for hosts that are not part of any team. A `null` or missing `team_id` returns statistics for all hosts regardless of the team.

### Example

`GET /api/v1/fleet/hosts/summary/mdm?team_id=1&platform=windows`

#### Default response

`Status: 200`

```json
{
  "counts_updated_at": "2021-03-21T12:32:44Z",
  "mobile_device_management_enrollment_status": {
    "enrolled_manual_hosts_count": 0,
    "enrolled_automated_hosts_count": 2,
    "unenrolled_hosts_count": 0,
    "hosts_count": 2
  },
  "mobile_device_management_solution": [
    {
      "id": 2,
      "name": "Solution1",
      "server_url": "solution1.com",
      "hosts_count": 1
    },
    {
      "id": 3,
      "name": "Solution2",
      "server_url": "solution2.com",
      "hosts_count": 1
    }
  ]
}
```

---

## Get host's mobile device management (MDM) and Munki information

Retrieves a host's MDM enrollment status, MDM server URL, and Munki version.

`GET /api/v1/fleet/hosts/:id/macadmins`

### Parameters

| Name    | Type    | In   | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ---- | -------------------------------------------------------------------------------- |
| id      | integer | path | **Required** The id of the host to get the details for                           |

### Example

`GET /api/v1/fleet/hosts/32/macadmins`

#### Default response

`Status: 200`

```json
{
  "macadmins": {
    "munki": {
      "version": "1.2.3"
    },
    "munki_issues": [
      {
        "id": 1,
        "name": "Could not retrieve managed install primary manifest",
        "type": "error",
        "created_at": "2022-08-01T05:09:44Z"
      },
      {
        "id": 2,
        "name": "Could not process item Figma for optional install. No pkginfo found in catalogs: release",
        "type": "warning",
        "created_at": "2022-08-01T05:09:44Z"
      }
    ],
    "mobile_device_management": {
      "enrollment_status": "On (automatic)",
      "server_url": "http://some.url/mdm",
      "name": "MDM Vendor Name",
      "id": 999
    }
  }
}
```

---

## Get aggregated host's macadmin mobile device management (MDM) and Munki information

Requires the [macadmins osquery
extension](https://github.com/macadmins/osquery-extension) which comes bundled
in [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd).
Currently supported only on macOS.


Retrieves aggregated host's MDM enrollment status and Munki versions.

`GET /api/v1/fleet/macadmins`

### Parameters

| Name    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ----- | ---------------------------------------------------------------------------------------------------------------- |
| team_id | integer | query | _Available in Fleet Premium_. Filters the aggregate host information to only include hosts in the specified team. |                           |

A `team_id` of `0` returns the statistics for hosts that are not part of any team. A `null` or missing `team_id` returns statistics for all hosts regardless of the team.

### Example

`GET /api/v1/fleet/macadmins`

#### Default response

`Status: 200`

```json
{
  "macadmins": {
    "counts_updated_at": "2021-03-21T12:32:44Z",
    "munki_versions": [
      {
        "version": "5.5",
        "hosts_count": 8360
      },
      {
        "version": "5.4",
        "hosts_count": 1700
      },
      {
        "version": "5.3",
        "hosts_count": 400
      },
      {
        "version": "5.2.3",
        "hosts_count": 112
      },
      {
        "version": "5.2.2",
        "hosts_count": 50
      }
    ],
    "munki_issues": [
      {
        "id": 1,
        "name": "Could not retrieve managed install primary manifest",
        "type": "error",
        "hosts_count": 2851
      },
      {
        "id": 2,
        "name": "Could not process item Figma for optional install. No pkginfo found in catalogs: release",
        "type": "warning",
        "hosts_count": 1983
      }
    ],
    "mobile_device_management_enrollment_status": {
      "enrolled_manual_hosts_count": 124,
      "enrolled_automated_hosts_count": 124,
      "unenrolled_hosts_count": 112
    },
    "mobile_device_management_solution": [
      {
        "id": 1,
        "name": "SimpleMDM",
        "hosts_count": 8360,
        "server_url": "https://a.simplemdm.com/mdm"
      },
      {
        "id": 2,
        "name": "Intune",
        "hosts_count": 1700,
        "server_url": "https://enrollment.manage.microsoft.com"
      }
    ]
  }
}
```

## Resend host's configuration profile

Resends a configuration profile for the specified host.

`POST /api/v1/fleet/hosts/:id/configuration_profiles/resend/:profile_uuid`

### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id   | integer | path | **Required.** The host's ID. |
| profile_uuid   | string | path | **Required.** The UUID of the configuration profile to resend to the host. |

### Example

`POST /api/v1/fleet/hosts/233/configuration_profiles/resend/fc14a20-84a2-42d8-9257-a425f62bb54d`

#### Default response

`Status: 202`

## Get host's scripts

`GET /api/v1/fleet/hosts/:id/scripts`

### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's id. |
| page | integer | query | Page number of the results to fetch.|
| per_page | integer | query | Results per page.|

### Example

`GET /api/v1/fleet/hosts/123/scripts`

#### Default response

`Status: 200`

```json
"scripts": [
  {
    "script_id": 3,
    "name": "remove-zoom-artifacts.sh",
    "last_execution": {
      "execution_id": "e797d6c6-3aae-11ee-be56-0242ac120002",
      "executed_at": "2021-12-15T15:23:57Z",
      "status": "error"
    }
  },
  {
    "script_id": 5,
    "name": "set-timezone.sh",
    "last_execution": {
      "id": "e797d6c6-3aae-11ee-be56-0242ac120002",
      "executed_at": "2021-12-15T15:23:57Z",
      "status": "pending"
    }
  },
  {
    "script_id": 8,
    "name": "uninstall-zoom.sh",
    "last_execution": {
      "id": "e797d6c6-3aae-11ee-be56-0242ac120002",
      "executed_at": "2021-12-15T15:23:57Z",
      "status": "ran"
    }
  }
],
"meta": {
  "has_next_results": false,
  "has_previous_results": false
}

```

## Get host's software

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

`GET /api/v1/fleet/hosts/:id/software`

### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's ID. |
| query   | string | query | Search query keywords. Searchable fields include `name`. |
| available_for_install | boolean | query | If `true` or `1`, only list software that is available for install (added by the user). Default is `false`.  
| page | integer | query | Page number of the results to fetch.|
| per_page | integer | query | Results per page.|

### Example

`GET /api/v1/fleet/hosts/123/software`

#### Default response

`Status: 200`

```json
{
  "count": 3,
  "software": [
    {
      "id": 121,
      "name": "Google Chrome.app",
      "software_package": {
        "name": "GoogleChrome.pkg",
        "version": "125.12.0.3",
        "self_service": true,
        "last_install": {
          "install_uuid": "8bbb8ac2-b254-4387-8cba-4d8a0407368b",
          "installed_at": "2024-05-15T15:23:57Z"
        }
      },
      "app_store_app": null
      "source": "apps",
      "status": "failed",
      "installed_versions": [
        {
          "version": "121.0",
          "last_opened_at": "2024-04-01T23:03:07Z",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"],
          "installed_paths": ["/Applications/Google Chrome.app"]
        }
      ]
    },
    {
      "id": 134,
      "name": "Falcon.app",
      "software_package": {
        "name": "FalconSensor-6.44.pkg"
        "self_service": false,
        "last_install": null
        "last_install": null,
        "last_uninstall": {
          "script_execution_id": "ed579e73-0f41-46c8-aaf4-3c1e5880ed27",
          "uninstalled_at": "2024-05-15T15:23:57Z"
        }
      },
      "app_store_app": null    
      "source": "",
      "status": null,
      "status": "pending_uninstall",
      "installed_versions": [],
    },
    {
      "id": 147,
      "name": "Logic Pro",
      "software_package": null
      "app_store_app": {
        "app_store_id": "1091189122",
        "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/f4/25/1f/f4251f60-e27a-6f05-daa7-9f3a63aac929/AppIcon-0-0-85-220-0-0-4-0-0-2x-0-0-0-0-0.png/512x512bb.png"
        "version": "2.04",
        "self_service": false,
        "last_install": {
          "command_uuid": "0aa14ae5-58fe-491a-ac9a-e4ee2b3aac40",
          "installed_at": "2024-05-15T15:23:57Z"
        },
      },
      "source": "apps",
      "status": "installed",
      "installed_versions": [
        {
          "version": "118.0",
          "last_opened_at": "2024-04-01T23:03:07Z",
          "vulnerabilities": ["CVE-2023-1234"],
          "installed_paths": ["/Applications/Logic Pro.app"]
        }
      ]
    },
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

## Get hosts report in CSV

Returns the list of hosts corresponding to the search criteria in CSV format, ready for download when
requested by a web browser.

`GET /api/v1/fleet/hosts/report`

### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| format                  | string  | query | **Required**, must be "csv" (only supported format for now).                                                                                                                                                                                                                                                                                |
| columns                 | string  | query | Comma-delimited list of columns to include in the report (returns all columns if none is specified).                                                                                                                                                                                                                                        |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                                                             |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include 'asc' and 'desc'. Default is 'asc'.                                                                                                                                                                                                               |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be 'new', 'online', 'offline', 'mia' or 'missing'.                                                                                                                                                                                                                                  |
| query                   | string  | query | Search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, `ipv4` and the hosts' email addresses (only searched if the query looks like an email address, i.e. contains an `@`, no space, etc.).                                                                                                               |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts in the specified team.                                                                                                                                                                                                                                                |
| policy_id               | integer | query | The ID of the policy to filter hosts by.                                                                                                                                                                                                                                                                                                    |
| policy_response         | string  | query | **Requires `policy_id`**. Valid options are 'passing' or 'failing'. **Note: If `policy_id` is specified _without_ including `policy_response`, this will also return hosts where the policy is not configured to run or failed to run.** |
| software_version_id     | integer | query | The ID of the software version to filter hosts by.                                                                                                            |
| software_title_id       | integer | query | The ID of the software title to filter hosts by.                                                                                                              |
| os_version_id | integer | query | The ID of the operating system version to filter hosts by. |
| os_name                 | string  | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                                                                                                                                                                                                     |
| os_version              | string  | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                                                                                                                                                                                                  |
| vulnerability           | string  | query | The cve to filter hosts by (including "cve-" prefix, case-insensitive).                                                                                                                                                                                                                                                                     |
| mdm_id                  | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).                                                                                                                                                                                                |
| mdm_name                | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).                                                                                                                                                                                                      |
| mdm_enrollment_status   | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Valid options are 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                 |
| macos_settings          | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| munki_issue_id          | integer | query | The ID of the _munki issue_ (a Munki-reported error or warning message) to filter hosts by (that is, filter hosts that are affected by that corresponding error or warning message).                                                                                                                                                        |
| low_disk_space          | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                                                                                                                                 |
| label_id                | integer | query | A valid label ID. Can only be used in combination with `order_key`, `order_direction`, `status`, `query` and `team_id`.                                                                                                                                                                                                                     |
| bootstrap_package       | string | query | _Available in Fleet Premium_. Filters the hosts by the status of the MDM bootstrap package on the host. Valid options are 'installed', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| disable_failing_policies | boolean | query | If `true`, hosts will return failing policies as 0 (returned as the `issues` column) regardless of whether there are any that failed for the host. This is meant to be used when increased performance is needed in exchange for the extra information.      |

If `mdm_id`, `mdm_name` or `mdm_enrollment_status` is specified, then Windows Servers are excluded from the results.

### Example

`GET /api/v1/fleet/hosts/report?software_id=123&format=csv&columns=hostname,primary_ip,platform`

#### Default response

`Status: 200`

```csv
created_at,updated_at,id,detail_updated_at,label_updated_at,policy_updated_at,last_enrolled_at,seen_time,refetch_requested,hostname,uuid,platform,osquery_version,os_version,build,platform_like,code_name,uptime,memory,cpu_type,cpu_subtype,cpu_brand,cpu_physical_cores,cpu_logical_cores,hardware_vendor,hardware_model,hardware_version,hardware_serial,computer_name,primary_ip_id,primary_ip,primary_mac,distributed_interval,config_tls_refresh,logger_tls_period,team_id,team_name,gigs_disk_space_available,percent_disk_space_available,gigs_total_disk_space,issues,device_mapping,status,display_text
2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,1,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,false,foo.local0,a4fc55a1-b5de-409c-a2f4-441f564680d3,debian,,,,,,0s,0,,,,0,0,,,,,,,,,0,0,0,,,0,0,0,0,,,,
2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:22:56Z,false,foo.local1,689539e5-72f0-4bf7-9cc5-1530d3814660,rhel,,,,,,0s,0,,,,0,0,,,,,,,,,0,0,0,,,0,0,0,0,,,,
2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,3,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:21:56Z,false,foo.local2,48ebe4b0-39c3-4a74-a67f-308f7b5dd171,linux,,,,,,0s,0,,,,0,0,,,,,,,,,0,0,0,,,0,0,0,0,,,,
```

## Get host's disk encryption key

Retrieves the disk encryption key for a host.

Requires that disk encryption is enforced and the host has MDM turned on.

`GET /api/v1/fleet/hosts/:id/encryption_key`

### Parameters

| Name | Type    | In   | Description                                                        |
| ---- | ------- | ---- | ------------------------------------------------------------------ |
| id   | integer | path | **Required** The id of the host to get the disk encryption key for |


### Example

`GET /api/v1/fleet/hosts/8/encryption_key`

#### Default response

`Status: 200`

```json
{
  "host_id": 8,
  "encryption_key": {
    "key": "5ADZ-HTZ8-LJJ4-B2F8-JWH3-YPBT",
    "updated_at": "2022-12-01T05:31:43Z"
  }
}
```

## Get configuration profiles assigned to a host

Requires Fleet's MDM properly [enabled and configured](https://fleetdm.com/docs/using-fleet/mdm-setup).

Retrieves a list of the configuration profiles assigned to a host.

`GET /api/v1/fleet/hosts/:id/configuration_profiles`

### Parameters

| Name | Type    | In   | Description                      |
| ---- | ------- | ---- | -------------------------------- |
| id   | integer | path | **Required**. The ID of the host  |


### Example

`GET /api/v1/fleet/hosts/8/configuration_profiles`

#### Default response

`Status: 200`

```json
{
  "host_id": 8,
  "profiles": [
    {
      "profile_uuid": "bc84dae7-396c-4e10-9d45-5768bce8b8bd",
      "team_id": 0,
      "name": "Example profile",
      "identifier": "com.example.profile",
      "created_at": "2023-03-31T00:00:00Z",
      "updated_at": "2023-03-31T00:00:00Z",
      "checksum": "dGVzdAo="
    }
  ]
}
```

## Lock host

_Available in Fleet Premium_

Sends a command to lock the specified macOS, Linux, or Windows host. The host is locked once it comes online.

To lock a macOS host, the host must have MDM turned on. To lock a Windows or Linux host, the host must have [scripts enabled](https://fleetdm.com/docs/using-fleet/scripts).


`POST /api/v1/fleet/hosts/:id/lock`

### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id | integer | path | **Required**. ID of the host to be locked. |
| view_pin | boolean | query | For macOS hosts, whether to return the unlock PIN. |

### Example

`POST /api/v1/fleet/hosts/123/lock`

#### Default response

`Status: 204`

### Example

`POST /api/v1/fleet/hosts/123/lock?view_pin=true`

#### Default response (macOS hosts)

`Status: 200`

```json
{
  "unlock_pin": "123456"
}
```

## Unlock host

_Available in Fleet Premium_

Sends a command to unlock the specified Windows or Linux host, or retrieves the unlock PIN for a macOS host.

To unlock a Windows or Linux host, the host must have [scripts enabled](https://fleetdm.com/docs/using-fleet/scripts).

`POST /api/v1/fleet/hosts/:id/unlock`

### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id | integer | path | **Required**. ID of the host to be unlocked. |

### Example

`POST /api/v1/fleet/hosts/:id/unlock`

#### Default response (Windows or Linux hosts)

`Status: 204`

#### Default response (macOS hosts)

`Status: 200`

```json
{
  "host_id": 8,
  "unlock_pin": "123456"
}
```

## Wipe host

Sends a command to wipe the specified macOS, iOS, iPadOS, Windows, or Linux host. The host is wiped once it comes online.

To wipe a macOS, iOS, iPadOS, or Windows host, the host must have MDM turned on. To lock a Linux host, the host must have [scripts enabled](https://fleetdm.com/docs/using-fleet/scripts).

`POST /api/v1/fleet/hosts/:id/wipe`

### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id | integer | path | **Required**. ID of the host to be wiped. |

### Example

`POST /api/v1/fleet/hosts/123/wipe`

#### Default response

`Status: 204`


## Get host's past activity

`GET /api/v1/fleet/hosts/:id/activities`

### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's ID. |
| page | integer | query | Page number of the results to fetch.|
| per_page | integer | query | Results per page.|

### Example

`GET /api/v1/fleet/hosts/12/activities`

#### Default response

`Status: 200`

```json
{
  "activities": [
    {
      "created_at": "2023-07-27T14:35:08Z",
      "actor_id": 1,
      "actor_full_name": "Anna Chao",
      "id": 4,
      "actor_gravatar": "",
      "actor_email": "",
      "type": "uninstalled_software",
      "details": {
        "host_id": 1,
        "host_display_name": "Marko’s MacBook Pro",
        "software_title": "Adobe Acrobat.app",
        "script_execution_id": "ecf22dba-07dc-40a9-b122-5480e948b756",
        "status": "failed"
      }
    }, 
    {
      "created_at": "2023-07-27T14:35:08Z",
      "actor_id": 1,
      "actor_full_name": "Anna Chao",
      "id": 3,
      "actor_gravatar": "",
      "actor_email": "",
      "type": "uninstalled_software",
      "details": {
        "host_id": 1,
        "host_display_name": "Marko’s MacBook Pro",
        "software_title": "Adobe Acrobat.app",
        "script_execution_id": "ecf22dba-07dc-40a9-b122-5480e948b756",
        "status": "uninstalled"
      }
    },
    {
      "created_at": "2023-07-27T14:35:08Z",
      "id": 2,
      "actor_full_name": "Anna",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "anna@example.com",
      "type": "ran_script",
      "details": {
        "host_id": 1,
        "host_display_name": "Steve's MacBook Pro",
        "script_name": "set-timezones.sh",
        "script_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
        "async": true
      },
    },
    {
      "created_at": "2021-07-27T13:25:21Z",
      "id": 1,
      "actor_full_name": "Bob",
      "actor_id": 2,
      "actor_gravatar": "",
      "actor_email": "bob@example.com",
      "type": "ran_script",
      "details": {
        "host_id": 1,
        "host_display_name": "Steve's MacBook Pro",
        "script_name": "",
        "script_execution_id": "y3cffa75-b5b5-41ef-9230-15073c8a88cf",
        "async": false
      },
    },
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

## Get host's upcoming activity

`GET /api/v1/fleet/hosts/:id/activities/upcoming`

### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's id. |
| page | integer | query | Page number of the results to fetch.|
| per_page | integer | query | Results per page.|

### Example

`GET /api/v1/fleet/hosts/12/activities/upcoming`

#### Default response

`Status: 200`

```json
{
  "count": 3,
  "activities": [
    {
      "created_at": "2023-07-27T14:35:08Z",
      "actor_id": 1,
      "actor_full_name": "Anna Chao",
      "uuid": "cc081637-fdf9-4d44-929f-96dfaec00f67",
      "actor_gravatar": "",
      "actor_email": "",
      "type": "uninstalled_software",
      "fleet_initiated_activity": false,
      "details": {
        "host_id": 1,
        "host_display_name": "Marko's MacBook Pro",
        "software_title": "Adobe Acrobat.app",
        "script_execution_id": "ecf22dba-07dc-40a9-b122-5480e948b756",
        "status": "pending_uninstall",
      }
    },
    {
      "created_at": "2023-07-27T14:35:08Z",
      "uuid": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
      "actor_full_name": "Marko",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "marko@example.com",
      "type": "ran_script",
      "details": {
        "host_id": 1,
        "host_display_name": "Steve's MacBook Pro",
        "script_name": "set-timezones.sh",
        "script_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
        "async": true
      },
    },
    {
      "created_at": "2021-07-27T13:25:21Z",
      "uuid": "y3cffa75-b5b5-41ef-9230-15073c8a88cf",
      "actor_full_name": "Rachael",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "rachael@example.com",
      "type": "ran_script",
      "details": {
        "host_id": 1,
        "host_display_name": "Steve's MacBook Pro",
        "script_name": "",
        "script_execution_id": "y3cffa75-b5b5-41ef-9230-15073c8a88cf",
        "async": false
      },
    },
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

## Add labels to host

Adds manual labels to a host.

`POST /api/v1/fleet/hosts/:id/labels`

### Parameters

| Name   | Type    | In   | Description                  |
| ------ | ------- | ---- | ---------------------------- |
| labels | array   | body | The list of label names to add to the host. |


### Example

`POST /api/v1/fleet/hosts/12/labels`

#### Request body

```json
{
  "labels": ["label1", "label2"]
}
```

#### Default response

`Status: 200`

## Remove labels from host

Removes manual labels from a host.

`DELETE /api/v1/fleet/hosts/:id/labels`

### Parameters

| Name   | Type    | In   | Description                  |
| ------ | ------- | ---- | ---------------------------- |
| labels | array   | body | The list of label names to delete from the host. |


### Example

`DELETE /api/v1/fleet/hosts/12/labels`

#### Request body

```json
{
  "labels": ["label3", "label4"]
}
```

#### Default response

`Status: 200`

## Live query one host (ad-hoc)

Runs an ad-hoc live query against the specified host and responds with the results.

The live query will stop if the targeted host is offline, or if the query times out. Timeouts happen if the host hasn't responded after the configured `FLEET_LIVE_QUERY_REST_PERIOD` (default 25 seconds) or if the `distributed_interval` agent option (default 10 seconds) is higher than the `FLEET_LIVE_QUERY_REST_PERIOD`.


`POST /api/v1/fleet/hosts/:id/query`

### Parameters

| Name      | Type  | In   | Description                                                                                                                                                        |
|-----------|-------|------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| id        | integer  | path | **Required**. The target host ID. |
| query     | string   | body | **Required**. The query SQL. |


### Example

`POST /api/v1/fleet/hosts/123/query`

#### Request body

```json
{
  "query": "SELECT model, vendor FROM usb_devices;"
}
```

#### Default response

`Status: 200`

```json
{
  "host_id": 123,
  "query": "SELECT model, vendor FROM usb_devices;",
  "status": "online", // "online" or "offline"
  "error": null,
  "rows": [
    {
      "model": "USB2.0 Hub",
      "vendor": "VIA Labs, Inc."
    }
  ]
}
```

Note that if the host is online and the query times out, this endpoint will return an error and `rows` will be `null`. If the host is offline, no error will be returned, and `rows` will be`null`.

## Live query host by identifier (ad-hoc)

Runs an ad-hoc live query against a host identified using `uuid` and responds with the results.

The live query will stop if the targeted host is offline, or if the query times out. Timeouts happen if the host hasn't responded after the configured `FLEET_LIVE_QUERY_REST_PERIOD` (default 25 seconds) or if the `distributed_interval` agent option (default 10 seconds) is higher than the `FLEET_LIVE_QUERY_REST_PERIOD`.


`POST /api/v1/fleet/hosts/identifier/:identifier/query`

### Parameters

| Name      | Type  | In   | Description                                                                                                                                                        |
|-----------|-------|------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| identifier       | integer or string   | path | **Required**. The host's `hardware_serial`, `uuid`, `osquery_host_id`, `hostname`, or `node_key`. |
| query            | string   | body | **Required**. The query SQL. |


### Example

`POST /api/v1/fleet/hosts/identifier/392547dc-0000-0000-a87a-d701ff75bc65/query`

#### Request body

```json
{
  "query": "SELECT model, vendor FROM usb_devices;"
}
```

#### Default response

`Status: 200`

```json
{
  "host_id": 123,
  "query": "SELECT model, vendor FROM usb_devices;",
  "status": "online", // "online" or "offline"
  "error": null,
  "rows": [
    {
      "model": "USB2.0 Hub",
      "vendor": "VIA Labs, Inc."
    }
  ]
}
```

Note that if the host is online and the query times out, this endpoint will return an error and `rows` will be `null`. If the host is offline, no error will be returned, and `rows` will be `null`.

---

<meta name="description" value="Documentation for Fleet's hosts REST API endpoints.">
<meta name="pageOrderInSection" value="70">