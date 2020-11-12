# Fleet REST API endpoints

## Hosts

### List hosts

`GET /api/v1/kolide/hosts`

Parameters

| Name                    | Type    | In    | Description                                                                              |
|-------------------------|---------|-------|------------------------------------------------------------------------------------------|
| page                    | integer | query | Page number of the results to fetch.                                                     |
| per_page                | integer | query | Results per page.                                                                        |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                          |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be `new`, `online`, `offline`, or `mia`.|
| additional_info_filters | string  | query | A comma delimited list of fields to include in each host's additional information object. See [Fleet Configuration Options](https://github.com/fleetdm/fleet/blob/master/docs/cli/file-format.md#fleet-configuration-options) for an example configuration with hosts' additional information.|

Example

`GET /api/v1/kolide/hosts?page=0&per_page=100&order_key=host_name`

```
{
  "hosts": [
    {
      "created_at": "2020-11-05T05:09:44Z",
      "updated_at": "2020-11-05T06:03:39Z",
      "id": 1,
      "detail_updated_at": "2020-11-05T05:09:45Z",
      "label_updated_at": "2020-11-05T05:14:51Z",
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
      "primary_ip": "",
      "primary_mac": "",
      "distributed_interval": 10,
      "config_tls_refresh": 10,
      "logger_tls_period": 8,
      "additional": {},
      "enroll_secret_name": "default",
      "status": "offline",
      "display_text": "2ceca32fe484"
    },
    {
      "created_at": "2020-11-05T05:09:44Z",
      "updated_at": "2020-11-05T06:03:39Z",
      "id": 2,
      "detail_updated_at": "2020-11-05T05:09:45Z",
      "label_updated_at": "2020-11-05T05:14:52Z",
      "seen_time": "2020-11-05T06:03:40Z",
      "hostname": "4cc885c20110",
      "uuid": "392547dc-0000-0000-a87a-d701ff75bc65",
      "platform": "centos",
      "osquery_version": "2.7.0",
      "os_version": "CentOS 6.8.0",
      "build": "",
      "platform_like": "rhel",
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
      "computer_name": "4cc885c20110",
      "primary_ip": "",
      "primary_mac": "",
      "distributed_interval": 10,
      "config_tls_refresh": 10,
      "logger_tls_period": 8,
      "additional": {},
      "enroll_secret_name": "default",
      "status": "offline",
      "display_text": "4cc885c20110"
    },
  ]
}
```