# OS settings

## Add custom OS setting (configuration profile)

> [Add custom macOS setting](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#add-custom-macos-setting-configuration-profile) (`POST /api/v1/fleet/mdm/apple/profiles`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the below API endpoint instead.

Add a configuration profile to enforce custom settings on macOS and Windows hosts.

`POST /api/v1/fleet/configuration_profiles`

### Parameters

| Name                      | Type     | In   | Description                                                                                                   |
| ------------------------- | -------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| profile                   | file     | form | **Required.** The .mobileconfig and JSON for macOS or XML for Windows file containing the profile. |
| team_id                   | string   | form | _Available in Fleet Premium_. The team ID for the profile. If specified, the profile is applied to only hosts that are assigned to the specified team. If not specified, the profile is applied to only to hosts that are not assigned to any team. |
| labels_include_all        | array     | form | _Available in Fleet Premium_. Profile will only be applied to hosts that have all of these labels. Only one of either `labels_include_all` or `labels_exclude_any` can be included in the request. |
| labels_exclude_any | array | form | _Available in Fleet Premium_. Profile will be applied to hosts that don’t have any of these labels. Only one of either `labels_include_all` or `labels_exclude_any` can be included in the request. |

### Example

Add a new configuration profile to be applied to macOS hosts
assigned to a team. Note that in this example the form data specifies`team_id` in addition to
`profile`.

`POST /api/v1/fleet/configuration_profiles`

#### Request headers

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

#### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="team_id"

1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="labels_include_all"

Label name 1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="labels_include_all"

Label name 2
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

#### Default response

`Status: 200`

```json
{
  "profile_uuid": "954ec5ea-a334-4825-87b3-937e7e381f24"
}
```

##### Additional notes
If the response is `Status: 409 Conflict`, the body may include additional error details in the case
of duplicate payload display name or duplicate payload identifier (macOS profiles).


## List custom OS settings (configuration profiles)

> [List custom macOS settings](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#list-custom-macos-settings-configuration-profiles) (`GET /api/v1/fleet/mdm/apple/profiles`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the below API endpoint instead.

Get a list of the configuration profiles in Fleet.

For Fleet Premium, the list can
optionally be filtered by team ID. If no team ID is specified, team profiles are excluded from the
results (i.e., only profiles that are associated with "No team" are listed).

`GET /api/v1/fleet/configuration_profiles`

### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_. The team id to filter profiles.              |
| page                      | integer | query | Page number of the results to fetch.                                     |
| per_page                  | integer | query | Results per page.                                                        |

### Example

List all configuration profiles for macOS and Windows hosts enrolled to Fleet's MDM that are not assigned to any team.

`GET /api/v1/fleet/configuration_profiles`

#### Default response

`Status: 200`

```json
{
  "profiles": [
    {
      "profile_uuid": "39f6cbbc-fe7b-4adc-b7a9-542d1af89c63",
      "team_id": 0,
      "name": "Example macOS profile",
      "platform": "darwin",
      "identifier": "com.example.profile",
      "created_at": "2023-03-31T00:00:00Z",
      "updated_at": "2023-03-31T00:00:00Z",
      "checksum": "dGVzdAo=",
      "labels_exclude_any": [
       {
        "name": "Label name 1",
        "id": 1
       }
      ]
    },
    {
      "profile_uuid": "f5ad01cc-f416-4b5f-88f3-a26da3b56a19",
      "team_id": 0,
      "name": "Example Windows profile",
      "platform": "windows",
      "created_at": "2023-04-31T00:00:00Z",
      "updated_at": "2023-04-31T00:00:00Z",
      "checksum": "aCLemVr)",
      "labels_include_all": [
        {
          "name": "Label name 2",
          "broken": true,
        },
        {
          "name": "Label name 3",
          "id": 3
        }
      ]
    }
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

If one or more assigned labels are deleted the profile is considered broken (`broken: true`). It won’t be applied to new hosts.

## Get or download custom OS setting (configuration profile)

> [Download custom macOS setting](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#download-custom-macos-setting-configuration-profile) (`GET /api/v1/fleet/mdm/apple/profiles/:profile_id`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the API endpoint below instead.

`GET /api/v1/fleet/configuration_profiles/:profile_uuid`

### Parameters

| Name                      | Type    | In    | Description                                             |
| ------------------------- | ------- | ----- | ------------------------------------------------------- |
| profile_uuid              | string | url   | **Required** The UUID of the profile to download.  |
| alt                       | string  | query | If specified and set to "media", downloads the profile. |

### Example (get a profile metadata)

`GET /api/v1/fleet/configuration_profiles/f663713f-04ee-40f0-a95a-7af428c351a9`

#### Default response

`Status: 200`

```json
{
  "profile_uuid": "f663713f-04ee-40f0-a95a-7af428c351a9",
  "team_id": 0,
  "name": "Example profile",
  "platform": "darwin",
  "identifier": "com.example.profile",
  "created_at": "2023-03-31T00:00:00Z",
  "updated_at": "2023-03-31T00:00:00Z",
  "checksum": "dGVzdAo=",
  "labels_include_all": [
    {
      "name": "Label name 1",
      "id": 1
      "broken": true
    },
    {
      "name": "Label name 2",
      "id": 2
    }
  ]
}
```

### Example (download a profile)

`GET /api/v1/fleet/configuration_profiles/f663713f-04ee-40f0-a95a-7af428c351a9?alt=media`

#### Default response

`Status: 200`

**Note** To confirm success, it is important for clients to match content length with the response
header (this is done automatically by most clients, including the browser) rather than relying
solely on the response status code returned by this endpoint.

#### Example response headers

```http
  Content-Length: 542
  Content-Type: application/octet-stream
  Content-Disposition: attachment;filename="2023-03-31 Example profile.mobileconfig"
```

##### Example response body

```xml
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

## Delete custom OS setting (configuration profile)

> [Delete custom macOS setting](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#delete-custom-macos-setting-configuration-profile) (`DELETE /api/v1/fleet/mdm/apple/profiles/:profile_id`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the below API endpoint instead.

`DELETE /api/v1/fleet/configuration_profiles/:profile_uuid`

### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------- | ----- | ------------------------------------------------------------------------- |
| profile_uuid              | string  | url   | **Required** The UUID of the profile to delete. |

### Example

`DELETE /api/v1/fleet/configuration_profiles/f663713f-04ee-40f0-a95a-7af428c351a9`

#### Default response

`Status: 200`


## Update disk encryption enforcement

> `PATCH /api/v1/fleet/mdm/apple/settings` API endpoint is deprecated as of Fleet 4.45. It is maintained for backward compatibility. Please use the new API endpoint below. See old API endpoint docs [here](https://github.com/fleetdm/fleet/blob/main/docs/REST%20API/rest-api.md?plain=1#L4296C29-L4296C29).

_Available in Fleet Premium_

`POST /api/v1/fleet/disk_encryption`

### Parameters

| Name                   | Type    | In    | Description                                                                                 |
| -------------          | ------  | ----  | --------------------------------------------------------------------------------------      |
| team_id                | integer | body  | The team ID to apply the settings to. Settings applied to hosts in no team if absent.       |
| enable_disk_encryption | boolean | body  | Whether disk encryption should be enforced on devices that belong to the team (or no team). |

### Example

`POST /api/v1/fleet/disk_encryption`

#### Default response

`204`


## Get disk encryption statistics

_Available in Fleet Premium_

Get aggregate status counts of disk encryption enforced on macOS and Windows hosts.

The summary can optionally be filtered by team ID.

`GET /api/v1/fleet/disk_encryption`

### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_. The team ID to filter the summary.           |

### Example

Get aggregate disk encryption status counts of macOS and Windows hosts enrolled to Fleet's MDM that are not assigned to any team.

`GET /api/v1/fleet/disk_encryption`

#### Default response

`Status: 200`

```json
{
  "verified": {"macos": 123, "windows": 123},
  "verifying": {"macos": 123, "windows": 0},
  "action_required": {"macos": 123, "windows": 0},
  "enforcing": {"macos": 123, "windows": 123},
  "failed": {"macos": 123, "windows": 123},
  "removing_enforcement": {"macos": 123, "windows": 0},
}
```


## Get OS settings status

> [Get macOS settings statistics](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#get-macos-settings-statistics) (`GET /api/v1/fleet/mdm/apple/profiles/summary`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the below API endpoint instead.

Get aggregate status counts of all OS settings (configuration profiles and disk encryption) enforced on hosts.

For Fleet Premium users, the counts can
optionally be filtered by `team_id`. If no `team_id` is specified, team profiles are excluded from the results (i.e., only profiles that are associated with "No team" are listed).

`GET /api/v1/fleet/configuration_profiles/summary`

### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_. The team ID to filter profiles.              |

### Example

Get aggregate status counts of profiles for to macOS and Windows hosts that are assigned to "No team".

`GET /api/v1/fleet/configuration_profiles/summary`

#### Default response

`Status: 200`

```json
{
  "verified": 123,
  "verifying": 123,
  "failed": 123,
  "pending": 123
}
```

---

<meta name="title" value="OS settings">
<meta name="description" value="Documentation for Fleet's OS settings REST API endpoints.">
<meta name="pageOrderInSection" value="100">