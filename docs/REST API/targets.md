# Targets

In Fleet, targets are used to run queries against specific hosts or groups of hosts. Labels are used to create groups in Fleet.

## Search targets

The search targets endpoint returns two lists. The first list includes the possible target hosts in Fleet given the search query provided and the hosts already selected as targets. The second list includes the possible target labels in Fleet given the search query provided and the labels already selected as targets.

The returned lists are filtered based on the hosts the requesting user has access to.

`POST /api/v1/fleet/targets`

### Parameters

| Name     | Type    | In   | Description                                                                                                                                                                |
| -------- | ------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| query    | string  | body | The search query. Searchable items include a host's hostname or IPv4 address and labels.                                                                                   |
| query_id | integer | body | The saved query (if any) that will be run. The `observer_can_run` property on the query and the user's roles effect which targets are included.                            |
| selected | object  | body | The targets already selected. The object includes a `hosts` property which contains a list of host IDs, a `labels` with label IDs and/or a `teams` property with team IDs. |

### Example

`POST /api/v1/fleet/targets`

#### Request body

```json
{
  "query": "172",
  "selected": {
    "hosts": [],
    "labels": [7]
  },
  "include_observer": true
}
```

#### Default response

```json
{
  "targets": {
    "hosts": [
      {
        "created_at": "2021-02-03T16:11:43Z",
        "updated_at": "2021-02-03T21:58:19Z",
        "id": 3,
        "detail_updated_at": "2021-02-03T21:58:10Z",
        "label_updated_at": "2021-02-03T21:58:10Z",
        "policy_updated_at": "2023-06-26T18:33:15Z",
        "last_enrolled_at": "2021-02-03T16:11:43Z",
        "software_updated_at": "2020-11-05T05:09:44Z",
        "seen_time": "2021-02-03T21:58:20Z",
        "hostname": "7a2f41482833",
        "uuid": "a2064cef-0000-0000-afb9-283e3c1d487e",
        "platform": "rhel",
        "osquery_version": "4.5.1",
        "os_version": "CentOS 6.10.0",
        "build": "",
        "platform_like": "rhel",
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
        "computer_name": "7a2f41482833",
        "display_name": "7a2f41482833",
        "primary_ip": "172.20.0.3",
        "primary_mac": "02:42:ac:14:00:03",
        "distributed_interval": 10,
        "config_tls_refresh": 10,
        "logger_tls_period": 10,
        "additional": {},
        "status": "offline",
        "display_text": "7a2f41482833"
      },
      {
        "created_at": "2021-02-03T16:11:43Z",
        "updated_at": "2021-02-03T21:58:19Z",
        "id": 4,
        "detail_updated_at": "2021-02-03T21:58:10Z",
        "label_updated_at": "2021-02-03T21:58:10Z",
        "policy_updated_at": "2023-06-26T18:33:15Z",
        "last_enrolled_at": "2021-02-03T16:11:43Z",
        "software_updated_at": "2020-11-05T05:09:44Z",
        "seen_time": "2021-02-03T21:58:20Z",
        "hostname": "78c96e72746c",
        "uuid": "a2064cef-0000-0000-afb9-283e3c1d487e",
        "platform": "ubuntu",
        "osquery_version": "4.5.1",
        "os_version": "Ubuntu 16.4.0",
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
        "computer_name": "78c96e72746c",
        "display_name": "78c96e72746c",
        "primary_ip": "172.20.0.7",
        "primary_mac": "02:42:ac:14:00:07",
        "distributed_interval": 10,
        "config_tls_refresh": 10,
        "logger_tls_period": 10,
        "additional": {},
        "status": "offline",
        "display_text": "78c96e72746c"
      }
    ],
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
        "host_count": 5,
        "display_text": "All Hosts",
        "count": 5
      }
    ],
    "teams": [
      {
        "id": 1,
        "created_at": "2021-05-27T20:02:20Z",
        "name": "Client Platform Engineering",
        "description": "",
        "agent_options": null,
        "user_count": 4,
        "host_count": 2,
        "display_text": "Client Platform Engineering",
        "count": 2
      }
    ]
  },
  "targets_count": 1,
  "targets_online": 1,
  "targets_offline": 0,
  "targets_missing_in_action": 0
}
```

---

<meta name="description" value="Documentation for Fleet's targets REST API endpoints.">
<meta name="pageOrderInSection" value="160">