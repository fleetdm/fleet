# Mobile device management (MDM)

These API endpoints are used to automate MDM features in Fleet. Read more about MDM features in Fleet [here](https://fleetdm.com/docs/using-fleet/mdm-setup).

- [Add custom macOS setting (configuration profile)](#add-custom-macos-setting-configuration-profile)
- [List custom macOS settings (configuration profiles)](#list-custom-macos-settings-configuration-profiles)
- [Download custom macOS setting (configuration profile)](#download-custom-macos-setting-configuration-profile)
- [Delete custom macOS setting (configuration profile)](#delete-custom-macos-setting-configuration-profile)
- [Update disk encryption enforcement](#update-disk-encryption-enforcement)
- [Get disk encryption statistics](#get-disk-encryption-statistics)
- [Get macOS settings statistics](#get-macos-settings-statistics)
- [Run custom MDM command](#run-custom-mdm-command)
- [Get custom MDM command results](#get-custom-mdm-command-results)
- [List custom MDM commands](#list-custom-mdm-commands)
- [Set custom MDM setup enrollment profile](#set-custom-mdm-setup-enrollment-profile)
- [Get custom MDM setup enrollment profile](#get-custom-mdm-setup-enrollment-profile)
- [Delete custom MDM setup enrollment profile](#delete-custom-mdm-setup-enrollment-profile)
- [Get Apple Push Notification service (APNs)](#get-apple-push-notification-service-apns)
- [Get Apple Business Manager (ABM)](#get-apple-business-manager-abm)
- [Turn off MDM for a host](#turn-off-mdm-for-a-host)
- [Upload a bootstrap package](#upload-a-bootstrap-package)
- [Get metadata about a bootstrap package](#get-metadata-about-a-bootstrap-package)
- [Delete a bootstrap package](#delete-a-bootstrap-package)
- [Download a bootstrap package](#download-a-bootstrap-package)
- [Get a summary of bootstrap package status](#get-a-summary-of-bootstrap-package-status)
- [Upload an EULA file](#upload-an-eula-file)
- [Get metadata about an EULA file](#get-metadata-about-an-eula-file)
- [Delete an EULA file](#delete-an-eula-file)
- [Download an EULA file](#download-an-eula-file)

## Add custom macOS setting (configuration profile)

Add a configuration profile to enforce custom settings on macOS hosts.

`POST /api/v1/fleet/mdm/apple/profiles`

#### Parameters

| Name                      | Type     | In   | Description                                                               |
| ------------------------- | -------- | ---- | ------------------------------------------------------------------------- |
| profile                   | file     | form | **Required**. The mobileconfig file containing the profile.               |
| team_id                   | string   | form | _Available in Fleet Premium_ The team id for the profile. If specified, the profile is applied to only hosts that are assigned to the specified team. If not specified, the profile is applied to only to hosts that are not assigned to any team. |

#### Example

Add a new configuration profile to be applied to macOS hosts enrolled to Fleet's MDM that are
assigned to a team. Note that in this example the form data specifies`team_id` in addition to
`profile`.

`POST /api/v1/fleet/mdm/apple/profiles`

##### Request headers

```
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```
--------------------------f02md47480und42y
Content-Disposition: form-data; name="team_id"

1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="profile"; filename="Foo.mobileconfig"
Content-Type: application/octet-stream

<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array/>
  <key>PayloadDisplayName</key>
  <string>Example profile</string>
  <key>PayloadIdentifier</key>
  <string>com.example.profile</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>0BBF3E23-7F56-48FC-A2B6-5ACC598A4A69</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
--------------------------f02md47480und42y--

```

##### Default response

`Status: 200`

```json
{
  "profile_id": 42
}
```

###### Additional notes
If the response is `Status: 409 Conflict`, the body may include additional error details in the case
of duplicate payload display name or duplicate payload identifier.


## List custom macOS settings (configuration profiles)

Get a list of the configuration profiles in Fleet.

For Fleet Premium, the list can
optionally be filtered by team ID. If no team ID is specified, team profiles are excluded from the
results (i.e., only profiles that are associated with "No team" are listed).

`GET /api/v1/fleet/mdm/apple/profiles`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_ The team id to filter profiles.              |

#### Example

List all configuration profiles for macOS hosts enrolled to Fleet's MDM that are not assigned to any team.

`GET /api/v1/fleet/mdm/apple/profiles`

##### Default response

`Status: 200`

```json
{
  "profiles": [
    {
      "profile_id": 1337,
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

## Download custom macOS setting (configuration profile)

`GET /api/v1/fleet/mdm/apple/profiles/{profile_id}`

#### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------- | ----- | ------------------------------------------------------------------------- |
| profile_id                | integer | url   | **Required** The id of the profile to download.                           |

#### Example

`GET /api/v1/fleet/mdm/apple/profiles/42`

##### Default response

`Status: 200`

**Note** To confirm success, it is important for clients to match content length with the response
header (this is done automatically by most clients, including the browser) rather than relying
solely on the response status code returned by this endpoint.

##### Example response headers

```
  Content-Length: 542
  Content-Type: application/octet-stream
  Content-Disposition: attachment;filename="2023-03-31 Example profile.mobileconfig"
```

###### Example response body
```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array/>
  <key>PayloadDisplayName</key>
  <string>Example profile</string>
  <key>PayloadIdentifier</key>
  <string>com.example.profile</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>0BBF3E23-7F56-48FC-A2B6-5ACC598A4A69</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

## Delete custom macOS setting (configuration profile)

`DELETE /api/v1/fleet/mdm/apple/profiles/{profile_id}`

#### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------- | ----- | ------------------------------------------------------------------------- |
| profile_id                | integer | url   | **Required** The id of the profile to delete.                             |

#### Example

`DELETE /api/v1/fleet/mdm/apple/profiles/42`

##### Default response

`Status: 200`

## Update disk encryption enforcement

_Available in Fleet Premium_

`PATCH /api/v1/fleet/mdm/apple/settings`

#### Parameters

| Name                   | Type    | In    | Description                                                                                 |
| -------------          | ------  | ----  | --------------------------------------------------------------------------------------      |
| team_id                | integer | body  | The team ID to apply the settings to. Settings applied to hosts in no team if absent.       |
| enable_disk_encryption | boolean | body  | Whether disk encryption should be enforced on devices that belong to the team (or no team). |

#### Example

`PATCH /api/v1/fleet/mdm/apple/settings`

##### Default response

`204`

## Get disk encryption statistics

_Available in Fleet Premium_

Get aggregate status counts of disk encryption enforced on hosts.

The summary can optionally be filtered by team id.

`GET /api/v1/fleet/mdm/apple/filevault/summary`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_ The team id to filter the summary.            |

#### Example

Get aggregate status counts of Apple disk encryption profiles applying to macOS hosts enrolled to Fleet's MDM that are not assigned to any team.

`GET /api/v1/fleet/mdm/apple/filevault/summary`

##### Default response

`Status: 200`

```json
{
  "verified": 123,
  "verifying": 123,
  "action_required": 123,
  "enforcing": 123,
  "failed": 123,
  "removing_enforcement": 123
}
```

## Get macOS settings statistics

Get aggregate status counts of all macOS settings (configuraiton profiles and disk encryption) enforced on hosts.

For Fleet Premium uses, the statistics can
optionally be filtered by team id. If no team id is specified, team profiles are excluded from the
results (i.e., only profiles that are associated with "No team" are listed).

`GET /api/v1/fleet/mdm/apple/profiles/summary`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_ The team id to filter profiles.              |

#### Example

Get aggregate status counts of MDM profiles applying to macOS hosts enrolled to Fleet's MDM that are not assigned to any team.

`GET /api/v1/fleet/mdm/apple/profiles/summary`

##### Default response

`Status: 200`

```json
{
  "verified": 123,
  "verifying": 123,
  "failed": 123,
  "pending": 123
}
```

## Run custom MDM command

This endpoint tells Fleet to run a custom an MDM command, on the targeted macOS hosts, the next time they come online.

`POST /api/v1/fleet/mdm/apple/enqueue`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| command                   | string | json  | A base64-encoded MDM command as described in [Apple's documentation](https://developer.apple.com/documentation/devicemanagement/commands_and_queries) |
| device_ids                | array  | json  | An array of host UUIDs enrolled in Fleet's MDM on which the command should run.                   |

Note that the `EraseDevice` and `DeviceLock` commands are _available in Fleet Premium_ only.

#### Example

`POST /api/v1/fleet/mdm/apple/enqueue`

##### Default response

`Status: 200`

```json
{
  "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
  "request_type": "ProfileList"
}
```

## Get custom MDM command results

This endpoint returns the results for a specific custom MDM command.

`GET /api/v1/fleet/mdm/apple/commandresults`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| command_uuid              | string | query | The unique identifier of the command.                                     |

#### Example

`GET /api/v1/fleet/mdm/apple/commandresults?command_uuid=a2064cef-0000-1234-afb9-283e3c1d487e`

##### Default response

`Status: 200`

```json
{
  "results": [
    {
      "device_id": "145cafeb-87c7-4869-84d5-e4118a927746",
      "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
      "status": "Acknowledged",
      "updated_at": "2023-04-04:00:00Z",
      "request_type": "ProfileList",
      "hostname": "mycomputer",
      "result": "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4KPCFET0NUWVBFIHBsaXN0IFBVQkxJQyAiLS8vQXBwbGUvL0RURCBQTElTVCAxLjAvL0VOIiAiaHR0cDovL3d3dy5hcHBsZS5jb20vRFREcy9Qcm9wZXJ0eUxpc3QtMS4wLmR0ZCI-CjxwbGlzdCB2ZXJzaW9uPSIxLjAiPgo8ZGljdD4KICAgIDxrZXk-Q29tbWFuZDwva2V5PgogICAgPGRpY3Q-CiAgICAgICAgPGtleT5NYW5hZ2VkT25seTwva2V5PgogICAgICAgIDxmYWxzZS8-CiAgICAgICAgPGtleT5SZXF1ZXN0VHlwZTwva2V5PgogICAgICAgIDxzdHJpbmc-UHJvZmlsZUxpc3Q8L3N0cmluZz4KICAgIDwvZGljdD4KICAgIDxrZXk-Q29tbWFuZFVVSUQ8L2tleT4KICAgIDxzdHJpbmc-MDAwMV9Qcm9maWxlTGlzdDwvc3RyaW5nPgo8L2RpY3Q-CjwvcGxpc3Q-"
    }
  ]
}
```

## List custom MDM commands

This endpoint returns the list of custom MDM commands that have been executed.

`GET /api/v1/fleet/mdm/apple/commands`

#### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------  | ----- | ------------------------------------------------------------------------- |
| page                      | integer | query | Page number of the results to fetch.                                      |
| per_page                  | integer | query | Results per page.                                                         |
| order_key                 | string  | query | What to order results by. Can be any field listed in the `results` array example below. |
| order_direction           | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

#### Example

`GET /api/v1/fleet/mdm/apple/commands?per_page=5`

##### Default response

`Status: 200`

```json
{
  "results": [
    {
      "device_id": "145cafeb-87c7-4869-84d5-e4118a927746",
      "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
      "status": "Acknowledged",
      "updated_at": "2023-04-04:00:00Z",
      "request_type": "ProfileList",
      "hostname": "mycomputer"
    }
  ]
}
```

## Set custom MDM setup enrollment profile

_Available in Fleet Premium_

Sets the custom MDM setup enrollment profile for a team or no team.

`POST /api/v1/fleet/mdm/apple/enrollment_profile`

#### Parameters

| Name                      | Type    | In    | Description                                                                   |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------     |
| team_id                   | integer | json  | The team id this custom enrollment profile applies to, or no team if omitted. |
| name                      | string  | json  | The filename of the uploaded custom enrollment profile.                       |
| enrollment_profile        | object  | json  | The custom enrollment profile's json, as documented in https://developer.apple.com/documentation/devicemanagement/profile. |

#### Example

`POST /api/v1/fleet/mdm/apple/enrollment_profile`

##### Default response

`Status: 200`

```json
{
  "team_id": 123,
  "name": "dep_profile.json",
  "uploaded_at": "2023-04-04:00:00Z",
  "enrollment_profile": {
    "is_mandatory": true,
    "is_mdm_removable": false
  }
}
```

## Get custom MDM setup enrollment profile

_Available in Fleet Premium_

Gets the custom MDM setup enrollment profile for a team or no team.

`GET /api/v1/fleet/mdm/apple/enrollment_profile`

#### Parameters

| Name                      | Type    | In    | Description                                                                           |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------             |
| team_id                   | integer | query | The team id for which to return the custom enrollment profile, or no team if omitted. |

#### Example

`GET /api/v1/fleet/mdm/apple/enrollment_profile?team_id=123`

##### Default response

`Status: 200`

```json
{
  "team_id": 123,
  "name": "dep_profile.json",
  "uploaded_at": "2023-04-04:00:00Z",
  "enrollment_profile": {
    "is_mandatory": true,
    "is_mdm_removable": false
  }
}
```

## Delete custom MDM setup enrollment profile

_Available in Fleet Premium_

Deletes the custom MDM setup enrollment profile assigned to a team or no team.

`DELETE /api/v1/fleet/mdm/apple/enrollment_profile`

#### Parameters

| Name                      | Type    | In    | Description                                                                           |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------             |
| team_id                   | integer | query | The team id for which to delete the custom enrollment profile, or no team if omitted. |

#### Example

`DELETE /api/v1/fleet/mdm/apple/enrollment_profile?team_id=123`

##### Default response

`Status: 204`

## Get Apple Push Notification service (APNs)

`GET /api/v1/fleet/mdm/apple`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/mdm/apple`

##### Default response

`Status: 200`

```json
{
  "common_name": "APSP:04u52i98aewuh-xxxx-xxxx-xxxx-xxxx",
  "serial_number": "1234567890987654321",
  "issuer": "Apple Application Integration 2 Certification Authority",
  "renew_date": "2023-09-30T00:00:00Z"
}
```

## Get Apple Business Manager (ABM)

_Available in Fleet Premium_

`GET /api/v1/fleet/mdm/apple_bm`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/mdm/apple_bm`

##### Default response

`Status: 200`

```json
{
  "apple_id": "apple@example.com",
  "org_name": "Fleet Device Management",
  "mdm_server_url": "https://example.com/mdm/apple/mdm",
  "renew_date": "2023-11-29T00:00:00Z",
  "default_team": ""
}
```

## Turn off MDM for a host

`PATCH /api/v1/fleet/mdm/hosts/{id}/unenroll`

#### Parameters

| Name | Type    | In   | Description                           |
| ---- | ------- | ---- | ------------------------------------- |
| id   | integer | path | **Required.** The host's ID in Fleet. |

#### Example

`PATCH /api/v1/fleet/mdm/hosts/42/unenroll`

##### Default response

`Status: 200`


## Upload a bootstrap package

_Available in Fleet Premium_

Upload a bootstrap package that will be automatically installed during DEP setup.

`POST /api/v1/fleet/mdm/apple/bootstrap`

#### Parameters

| Name    | Type   | In   | Description                                                                                                                                                                                                            |
| ------- | ------ | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| package | file   | form | **Required**. The bootstrap package installer. It must be a signed `pkg` file.                                                                                                                                         |
| team_id | string | form | The team id for the package. If specified, the package will be installed to hosts that are assigned to the specified team. If not specified, the package will be installed to hosts that are not assigned to any team. |

#### Example

Upload a bootstrap package that will be installed to macOS hosts enrolled to MDM that are
assigned to a team. Note that in this example the form data specifies `team_id` in addition to
`package`.

`POST /api/v1/fleet/mdm/apple/profiles`

##### Request headers

```
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```
--------------------------f02md47480und42y
Content-Disposition: form-data; name="team_id"
1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="package"; filename="bootstrap-package.pkg"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------f02md47480und42y--
```

##### Default response

`Status: 200`

## Get metadata about a bootstrap package

_Available in Fleet Premium_

Get information about a bootstrap package that was uploaded to Fleet.

`GET /api/v1/fleet/mdm/apple/bootstrap/{team_id}/metadata`

#### Parameters

| Name       | Type    | In    | Description                                                                                                                                                                                                        |
| -------    | ------  | ---   | ---------------------------------------------------------------------------------------------------------------------------------------------------------                                                          |
| team_id    | string  | url   | **Required** The team id for the package. Zero (0) can be specified to get information about the bootstrap package for hosts that don't belong to a team.                                                          |
| for_update | boolean | query | If set to `true`, the authorization will be for a `write` action instead of a `read`. Useful for the write-only `gitops` role when requesting the bootstrap metadata to check if the package needs to be replaced. |

#### Example

`GET /api/v1/fleet/mdm/apple/bootstrap/0/metadata`

##### Default response

`Status: 200`

```json
{
  "name": "bootstrap-package.pkg",
  "team_id": 0,
  "sha256": "6bebb4433322fd52837de9e4787de534b4089ac645b0692dfb74d000438da4a3",
  "token": "AA598E2A-7952-46E3-B89D-526D45F7E233",
  "created_at": "2023-04-20T13:02:05Z"
}
```

In the response above:

- `token` is the value you can use to [download a bootstrap package](#download-a-bootstrap-package)
- `sha256` is the SHA256 digest of the bytes of the bootstrap package file.

## Delete a bootstrap package

_Available in Fleet Premium_

Delete a team's bootstrap package.

`DELETE /api/v1/fleet/mdm/apple/bootstrap/{team_id}`

#### Parameters

| Name    | Type   | In  | Description                                                                                                                                               |
| ------- | ------ | --- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| team_id | string | url | **Required** The team id for the package. Zero (0) can be specified to get information about the bootstrap package for hosts that don't belong to a team. |


#### Example

`DELETE /api/v1/fleet/mdm/apple/bootstrap/1`

##### Default response

`Status: 200`

## Download a bootstrap package

_Available in Fleet Premium_

Download a bootstrap package.

`GET /api/v1/fleet/mdm/apple/bootstrap`

#### Parameters

| Name  | Type   | In    | Description                                      |
| ----- | ------ | ----- | ------------------------------------------------ |
| token | string | query | **Required** The token of the bootstrap package. |

#### Example

`GET /api/v1/fleet/mdm/apple/bootstrap?token=AA598E2A-7952-46E3-B89D-526D45F7E233`

##### Default response

`Status: 200`

```
Status: 200
Content-Type: application/octet-stream
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

## Get a summary of bootstrap package status

_Available in Fleet Premium_

Get aggregate status counts of bootstrap packages delivered to DEP enrolled hosts.

The summary can optionally be filtered by team id.

`GET /api/v1/fleet/mdm/apple/bootstrap/summary`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | The team id to filter the summary.                                        |

#### Example

`GET /api/v1/fleet/mdm/apple/bootstrap/summary`

##### Default response

`Status: 200`

```json
{
  "installed": 10,
  "failed": 1,
  "pending": 4
}
```

## Turn on end user authentication for macOS setup

_Available in Fleet Premium_

`PATCH /api/v1/fleet/mdm/apple/setup`

#### Parameters

| Name                           | Type    | In    | Description                                                                                 |
| -------------          | ------  | ----  | --------------------------------------------------------------------------------------      |
| team_id                        | integer | body  | The team ID to apply the settings to. Settings applied to hosts in no team if absent.       |
| enable_end_user_authentication | boolean | body  | Whether end user authentication should be enabled for new macOS devices that automatically enroll to the team (or no team). |

#### Example

`PATCH /api/v1/fleet/mdm/apple/setup`

##### Request body

```json
{
  "team_id": 1,
  "enabled_end_user_authentication": true
}
```

##### Default response

`Status: 204`



## Upload an EULA file

_Available in Fleet Premium_

Upload an EULA that will be shown during the DEP flow.

`POST /api/v1/fleet/mdm/apple/setup/eula`

#### Parameters

| Name | Type | In   | Description                                       |
| ---- | ---- | ---- | ------------------------------------------------- |
| eula | file | form | **Required**. A PDF document containing the EULA. |

#### Example

`POST /api/v1/fleet/mdm/apple/setup/eula`

##### Request headers

```
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```
--------------------------f02md47480und42y
Content-Disposition: form-data; name="eula"; filename="eula.pdf"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------f02md47480und42y--
```

##### Default response

`Status: 200`

## Get metadata about an EULA file

_Available in Fleet Premium_

Get information about the EULA file that was uploaded to Fleet. If no EULA was previously uploaded, this endpoint returns a `404` status code.

`GET /api/v1/fleet/mdm/apple/setup/eula/metadata`

#### Example

`GET /api/v1/fleet/mdm/apple/setup/eula/metadata`

##### Default response

`Status: 200`

```json
{
  "name": "eula.pdf",
  "token": "AA598E2A-7952-46E3-B89D-526D45F7E233",
  "created_at": "2023-04-20T13:02:05Z"
}
```

In the response above:

- `token` is the value you can use to [download an EULA](#download-an-eula-file)

## Delete an EULA file

_Available in Fleet Premium_

Delete an EULA file.

`DELETE /api/v1/fleet/mdm/apple/setup/eula/{token}`

#### Parameters

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| token | string | path  | **Required** The token of the EULA file. |

#### Example

`DELETE /api/v1/fleet/mdm/apple/setup/eula/AA598E2A-7952-46E3-B89D-526D45F7E233`

##### Default response

`Status: 200`

## Download an EULA file

_Available in Fleet Premium_

Download an EULA file

`GET /api/v1/fleet/mdm/apple/setup/eula/{token}`

#### Parameters

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| token | string | path  | **Required** The token of the EULA file. |

#### Example

`GET /api/v1/fleet/mdm/apple/setup/eula/AA598E2A-7952-46E3-B89D-526D45F7E233`

##### Default response

`Status: 200`

```
Status: 200
Content-Type: application/pdf
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

<meta name="description" value="Learn about the API endpoints that can be used to automate and modify mobile device management features in Fleet.">
<meta name="title" value="Mobile device management (MDM)">
<meta name="pageOrderInSection" value="800">