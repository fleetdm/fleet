# Commands

## Run MDM command

> `POST /api/v1/fleet/mdm/apple/enqueue` API endpoint is deprecated as of Fleet 4.40. It is maintained for backward compatibility. Please use the new API endpoint below. See old API endpoint docs [here](https://github.com/fleetdm/fleet/blob/fleet-v4.39.0/docs/REST%20API/rest-api.md#run-custom-mdm-command).

This endpoint tells Fleet to run a custom MDM command, on the targeted macOS or Windows hosts, the next time they come online.

`POST /api/v1/fleet/commands/run`

### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| command                   | string | json  | A Base64 encoded MDM command as described in [Apple's documentation](https://developer.apple.com/documentation/devicemanagement/commands_and_queries) or [Windows's documentation](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mdm/0353f3d6-dbe2-42b6-b8d5-50db9333bba4). Supported formats are standard and raw (unpadded). You can paste your Base64 code to the [online decoder](https://devpal.co/base64-decode/) to check if you're using the valid format. |
| host_uuids                | array  | json  | An array of host UUIDs enrolled in Fleet on which the command should run. |

Note that the `EraseDevice` and `DeviceLock` commands are _available in Fleet Premium_ only.

### Example

`POST /api/v1/fleet/commands/run`

#### Default response

`Status: 200`

```json
{
  "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
  "request_type": "ProfileList"
}
```


## Get MDM command results

> `GET /api/v1/fleet/mdm/apple/commandresults` API endpoint is deprecated as of Fleet 4.40. It is maintained for backward compatibility. Please use the new API endpoint below. See old API endpoint docs [here](https://github.com/fleetdm/fleet/blob/fleet-v4.39.0/docs/REST%20API/rest-api.md#get-custom-mdm-command-results).

This endpoint returns the results for a specific custom MDM command.

In the reponse, the possible `status` values for macOS, iOS, and iPadOS hosts are the following:

* Pending: the command has yet to run on the host. The host will run the command the next time it comes online.
* NotNow: the host responded with "NotNow" status via the MDM protocol: the host received the command, but couldn’t execute it. The host will try to run the command the next time it comes online.
* Acknowledged: the host responded with "Acknowledged" status via the MDM protocol: the host processed the command successfully.
* Error: the host responded with "Error" status via the MDM protocol: an error occurred. Run the `fleetctl get mdm-command-results --id=<insert-command-id` to view the error.
* CommandFormatError: the host responded with "CommandFormatError" status via the MDM protocol: a protocol error occurred, which can result from a malformed command. Run the `fleetctl get mdm-command-results --id=<insert-command-id` to view the error.

The possible `status` values for Windows hosts are documented in Microsoft's documentation [here](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes).

`GET /api/v1/fleet/commands/results`

### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| command_uuid              | string | query | The unique identifier of the command.                                     |

### Example

`GET /api/v1/fleet/commands/results?command_uuid=a2064cef-0000-1234-afb9-283e3c1d487e`

#### Default response

`Status: 200`

```json
{
  "results": [
    {
      "host_uuid": "145cafeb-87c7-4869-84d5-e4118a927746",
      "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
      "status": "Acknowledged",
      "updated_at": "2023-04-04:00:00Z",
      "request_type": "ProfileList",
      "hostname": "mycomputer",
      "payload": "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4NCjwhRE9DVFlQRSBwbGlzdCBQVUJMSUMgIi0vL0FwcGxlLy9EVEQgUExJU1QgMS4wLy9FTiIgImh0dHA6Ly93d3cuYXBwbGUuY29tL0RURHMvUHJvcGVydHlMaXN0LTEuMC5kdGQiPg0KPHBsaXN0IHZlcnNpb249IjEuMCI+DQo8ZGljdD4NCg0KCTxrZXk+UGF5bG9hZERlc2NyaXB0aW9uPC9rZXk+DQoJPHN0cmluZz5UaGlzIHByb2ZpbGUgY29uZmlndXJhdGlvbiBpcyBkZXNpZ25lZCB0byBhcHBseSB0aGUgQ0lTIEJlbmNobWFyayBmb3IgbWFjT1MgMTAuMTQgKHYyLjAuMCksIDEwLjE1ICh2Mi4wLjApLCAxMS4wICh2Mi4wLjApLCBhbmQgMTIuMCAodjEuMC4wKTwvc3RyaW5nPg0KCTxrZXk+UGF5bG9hZERpc3BsYXlOYW1lPC9rZXk+DQoJPHN0cmluZz5EaXNhYmxlIEJsdWV0b290aCBzaGFyaW5nPC9zdHJpbmc+DQoJPGtleT5QYXlsb2FkRW5hYmxlZDwva2V5Pg0KCTx0cnVlLz4NCgk8a2V5PlBheWxvYWRJZGVudGlmaWVyPC9rZXk+DQoJPHN0cmluZz5jaXMubWFjT1NCZW5jaG1hcmsuc2VjdGlvbjIuQmx1ZXRvb3RoU2hhcmluZzwvc3RyaW5nPg0KCTxrZXk+UGF5bG9hZFNjb3BlPC9rZXk+DQoJPHN0cmluZz5TeXN0ZW08L3N0cmluZz4NCgk8a2V5PlBheWxvYWRUeXBlPC9rZXk+DQoJPHN0cmluZz5Db25maWd1cmF0aW9uPC9zdHJpbmc+DQoJPGtleT5QYXlsb2FkVVVJRDwva2V5Pg0KCTxzdHJpbmc+NUNFQkQ3MTItMjhFQi00MzJCLTg0QzctQUEyOEE1QTM4M0Q4PC9zdHJpbmc+DQoJPGtleT5QYXlsb2FkVmVyc2lvbjwva2V5Pg0KCTxpbnRlZ2VyPjE8L2ludGVnZXI+DQogICAgPGtleT5QYXlsb2FkUmVtb3ZhbERpc2FsbG93ZWQ8L2tleT4NCiAgICA8dHJ1ZS8+DQoJPGtleT5QYXlsb2FkQ29udGVudDwva2V5Pg0KCTxhcnJheT4NCgkJPGRpY3Q+DQoJCQk8a2V5PlBheWxvYWRDb250ZW50PC9rZXk+DQoJCQk8ZGljdD4NCgkJCQk8a2V5PmNvbS5hcHBsZS5CbHVldG9vdGg8L2tleT4NCgkJCQk8ZGljdD4NCgkJCQkJPGtleT5Gb3JjZWQ8L2tleT4NCgkJCQkJPGFycmF5Pg0KCQkJCQkJPGRpY3Q+DQoJCQkJCQkJPGtleT5tY3hfcHJlZmVyZW5jZV9zZXR0aW5nczwva2V5Pg0KCQkJCQkJCTxkaWN0Pg0KCQkJCQkJCQk8a2V5PlByZWZLZXlTZXJ2aWNlc0VuYWJsZWQ8L2tleT4NCgkJCQkJCQkJPGZhbHNlLz4NCgkJCQkJCQk8L2RpY3Q+DQoJCQkJCQk8L2RpY3Q+DQoJCQkJCTwvYXJyYXk+DQoJCQkJPC9kaWN0Pg0KCQkJPC9kaWN0Pg0KCQkJPGtleT5QYXlsb2FkRGVzY3JpcHRpb248L2tleT4NCgkJCTxzdHJpbmc+RGlzYWJsZXMgQmx1ZXRvb3RoIFNoYXJpbmc8L3N0cmluZz4NCgkJCTxrZXk+UGF5bG9hZERpc3BsYXlOYW1lPC9rZXk+DQoJCQk8c3RyaW5nPkN1c3RvbTwvc3RyaW5nPg0KCQkJPGtleT5QYXlsb2FkRW5hYmxlZDwva2V5Pg0KCQkJPHRydWUvPg0KCQkJPGtleT5QYXlsb2FkSWRlbnRpZmllcjwva2V5Pg0KCQkJPHN0cmluZz4wMjQwREQxQy03MERDLTQ3NjYtOTAxOC0wNDMyMkJGRUVBRDE8L3N0cmluZz4NCgkJCTxrZXk+UGF5bG9hZFR5cGU8L2tleT4NCgkJCTxzdHJpbmc+Y29tLmFwcGxlLk1hbmFnZWRDbGllbnQucHJlZmVyZW5jZXM8L3N0cmluZz4NCgkJCTxrZXk+UGF5bG9hZFVVSUQ8L2tleT4NCgkJCTxzdHJpbmc+MDI0MEREMUMtNzBEQy00NzY2LTkwMTgtMDQzMjJCRkVFQUQxPC9zdHJpbmc+DQoJCQk8a2V5PlBheWxvYWRWZXJzaW9uPC9rZXk+DQoJCQk8aW50ZWdlcj4xPC9pbnRlZ2VyPg0KCQk8L2RpY3Q+DQoJPC9hcnJheT4NCjwvZGljdD4NCjwvcGxpc3Q+",
      "result": "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4NCjwhRE9DVFlQRSBwbGlzdCBQVUJMSUMgIi0vL0FwcGxlLy9EVEQgUExJU1QgMS4wLy9FTiIgImh0dHA6Ly93d3cuYXBwbGUuY29tL0RURHMvUHJvcGVydHlMaXN0LTEuMC5kdGQiPg0KPHBsaXN0IHZlcnNpb249IjEuMCI+DQo8ZGljdD4NCiAgICA8a2V5PkNvbW1hbmRVVUlEPC9rZXk+DQogICAgPHN0cmluZz4wMDAxX0luc3RhbGxQcm9maWxlPC9zdHJpbmc+DQogICAgPGtleT5TdGF0dXM8L2tleT4NCiAgICA8c3RyaW5nPkFja25vd2xlZGdlZDwvc3RyaW5nPg0KICAgIDxrZXk+VURJRDwva2V5Pg0KICAgIDxzdHJpbmc+MDAwMDgwMjAtMDAwOTE1MDgzQzgwMDEyRTwvc3RyaW5nPg0KPC9kaWN0Pg0KPC9wbGlzdD4="
    }
  ]
}
```

> Note: If the server has not yet received a result for a command, it will return an empty object (`{}`).

## List MDM commands

> `GET /api/v1/fleet/mdm/apple/commands` API endpoint is deprecated as of Fleet 4.40. It is maintained for backward compatibility. Please use the new API endpoint below. See old API endpoint docs [here](https://github.com/fleetdm/fleet/blob/fleet-v4.39.0/docs/REST%20API/rest-api.md#list-custom-mdm-commands).

This endpoint returns the list of custom MDM commands that have been executed.

`GET /api/v1/fleet/commands`

### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------  | ----- | ------------------------------------------------------------------------- |
| page                      | integer | query | Page number of the results to fetch.                                      |
| per_page                  | integer | query | Results per page.                                                         |
| order_key                 | string  | query | What to order results by. Can be any field listed in the `results` array example below. |
| order_direction           | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| host_identifier           | string  | query | The host's `hostname`, `uuid`, or `hardware_serial`. |
| request_type              | string  | query | The request type to filter commands by. |

### Example

`GET /api/v1/fleet/commands?per_page=5`

#### Default response

`Status: 200`

```json
{
  "results": [
    {
      "host_uuid": "145cafeb-87c7-4869-84d5-e4118a927746",
      "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
      "status": "Acknowledged",
      "updated_at": "2023-04-04:00:00Z",
      "request_type": "ProfileList",
      "hostname": "mycomputer"
    },
    {
      "host_uuid": "322vghee-12c7-8976-83a1-e2118a927342",
      "command_uuid": "d76d69b7-d806-45a9-8e49-9d6dc533485c",
      "status": "200",
      "updated_at": "2023-05-04:00:00Z",
      "request_type": "./Device/Vendor/MSFT/Reboot/RebootNow",
      "hostname": "myhost"
    }
  ]
}
```

---

<meta name="description" value="Documentation for Fleet's mdm command REST API endpoints.">
<meta name="pageOrderInSection" value="30">