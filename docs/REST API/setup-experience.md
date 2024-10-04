# Setup experience

## Set custom MDM setup enrollment profile

_Available in Fleet Premium_

Sets the custom MDM setup enrollment profile for a team or no team.

`POST /api/v1/fleet/enrollment_profiles/automatic`

### Parameters

| Name                      | Type    | In    | Description                                                                   |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------     |
| team_id                   | integer | json  | The team ID this custom enrollment profile applies to, or no team if omitted. |
| name                      | string  | json  | The filename of the uploaded custom enrollment profile.                       |
| enrollment_profile        | object  | json  | The custom enrollment profile's json, as documented in https://developer.apple.com/documentation/devicemanagement/profile. |

### Example

`POST /api/v1/fleet/enrollment_profiles/automatic`

#### Default response

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

`GET /api/v1/fleet/enrollment_profiles/automatic`

### Parameters

| Name                      | Type    | In    | Description                                                                           |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------             |
| team_id                   | integer | query | The team ID for which to return the custom enrollment profile, or no team if omitted. |

### Example

`GET /api/v1/fleet/enrollment_profiles/automatic?team_id=123`

#### Default response

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

`DELETE /api/v1/fleet/enrollment_profiles/automatic`

### Parameters

| Name                      | Type    | In    | Description                                                                           |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------             |
| team_id                   | integer | query | The team ID for which to delete the custom enrollment profile, or no team if omitted. |

### Example

`DELETE /api/v1/fleet/enrollment_profiles/automatic?team_id=123`

#### Default response

`Status: 204`


## Get Over-the-Air (OTA) enrollment profile

`GET /api/v1/fleet/enrollment_profiles/ota`

The returned value is a signed `.mobileconfig` OTA enrollment profile. Install this profile on macOS, iOS, or iPadOS hosts to enroll them to a specific team in Fleet and turn on MDM features.

To enroll macOS hosts, turn on MDM features, and add [human-device mapping](#get-human-device-mapping), install the [manual enrollment profile](#get-manual-enrollment-profile) instead.

Learn more about OTA profiles [here](https://developer.apple.com/library/archive/documentation/NetworkingInternet/Conceptual/iPhoneOTAConfiguration/OTASecurity/OTASecurity.html).

### Parameters

| Name              | Type    | In    | Description                                                                      |
|-------------------|---------|-------|----------------------------------------------------------------------------------|
| enroll_secret     | string  | query | **Required**. The enroll secret of the team this host will be assigned to.       |

### Example

`GET /api/v1/fleet/enrollment_profiles/ota?enroll_secret=foobar`

#### Default response

`Status: 200`

> **Note:** To confirm success, it is important for clients to match content length with the response header (this is done automatically by most clients, including the browser) rather than relying solely on the response status code returned by this endpoint.

#### Example response headers

```http
  Content-Length: 542
  Content-Type: application/x-apple-aspen-config; charset=urf-8
  Content-Disposition: attachment;filename="fleet-mdm-enrollment-profile.mobileconfig"
  X-Content-Type-Options: nosniff
```

##### Example response body

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Inc//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>PayloadContent</key>
    <dict>
      <key>URL</key>
      <string>https://foo.example.com/api/fleet/ota_enrollment?enroll_secret=foobar</string>
      <key>DeviceAttributes</key>
      <array>
        <string>UDID</string>
        <string>VERSION</string>
        <string>PRODUCT</string>
        <string>SERIAL</string>
      </array>
    </dict>
    <key>PayloadOrganization</key>
    <string>Acme Inc.</string>
    <key>PayloadDisplayName</key>
    <string>Acme Inc. enrollment</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>PayloadUUID</key>
    <string>fdb376e5-b5bb-4d8c-829e-e90865f990c9</string>
    <key>PayloadIdentifier</key>
    <string>com.fleetdm.fleet.mdm.apple.ota</string>
    <key>PayloadType</key>
    <string>Profile Service</string>
  </dict>
</plist>
```


## Get manual enrollment profile

Retrieves an unsigned manual enrollment profile for macOS hosts. Install this profile on macOS hosts to turn on MDM features manually.

To add [human-device mapping](#get-human-device-mapping), add the end user's email to the enrollment profle. Learn how [here](https://fleetdm.com/guides/config-less-fleetd-agent-deployment#basic-article).

`GET /api/v1/fleet/enrollment_profiles/manual`

#### Example

`GET /api/v1/fleet/enrollment_profiles/manual`

#### Default response

`Status: 200`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<!-- ... -->
</plist>
```

## Upload a bootstrap package

_Available in Fleet Premium_

Upload a bootstrap package that will be automatically installed during DEP setup.

`POST /api/v1/fleet/bootstrap`

### Parameters

| Name    | Type   | In   | Description                                                                                                                                                                                                            |
| ------- | ------ | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| package | file   | form | **Required**. The bootstrap package installer. It must be a signed `pkg` file.                                                                                                                                         |
| team_id | string | form | The team ID for the package. If specified, the package will be installed to hosts that are assigned to the specified team. If not specified, the package will be installed to hosts that are not assigned to any team. |

### Example

Upload a bootstrap package that will be installed to macOS hosts enrolled to MDM that are
assigned to a team. Note that in this example the form data specifies `team_id` in addition to
`package`.

`POST /api/v1/fleet/bootstrap`

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
Content-Disposition: form-data; name="package"; filename="bootstrap-package.pkg"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------f02md47480und42y--
```

#### Default response

`Status: 200`


## Get metadata about a bootstrap package

_Available in Fleet Premium_

Get information about a bootstrap package that was uploaded to Fleet.

`GET /api/v1/fleet/bootstrap/:team_id/metadata`

### Parameters

| Name       | Type    | In    | Description                                                                                                                                                                                                        |
| -------    | ------  | ---   | ---------------------------------------------------------------------------------------------------------------------------------------------------------                                                          |
| team_id    | string  | url   | **Required** The team ID for the package. Zero (0) can be specified to get information about the bootstrap package for hosts that don't belong to a team.                                                          |
| for_update | boolean | query | If set to `true`, the authorization will be for a `write` action instead of a `read`. Useful for the write-only `gitops` role when requesting the bootstrap metadata to check if the package needs to be replaced. |

### Example

`GET /api/v1/fleet/bootstrap/0/metadata`

#### Default response

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

`DELETE /api/v1/fleet/bootstrap/:team_id`

### Parameters

| Name    | Type   | In  | Description                                                                                                                                               |
| ------- | ------ | --- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| team_id | string | url | **Required** The team ID for the package. Zero (0) can be specified to get information about the bootstrap package for hosts that don't belong to a team. |


### Example

`DELETE /api/v1/fleet/bootstrap/1`

#### Default response

`Status: 200`


## Download a bootstrap package

_Available in Fleet Premium_

Download a bootstrap package.

`GET /api/v1/fleet/bootstrap`

### Parameters

| Name  | Type   | In    | Description                                      |
| ----- | ------ | ----- | ------------------------------------------------ |
| token | string | query | **Required** The token of the bootstrap package. |

### Example

`GET /api/v1/fleet/bootstrap?token=AA598E2A-7952-46E3-B89D-526D45F7E233`

#### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/octet-stream
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

## Get a summary of bootstrap package status

_Available in Fleet Premium_

Get aggregate status counts of bootstrap packages delivered to DEP enrolled hosts.

The summary can optionally be filtered by team ID.

`GET /api/v1/fleet/bootstrap/summary`

### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | The team ID to filter the summary.                                        |

### Example

`GET /api/v1/fleet/bootstrap/summary`

#### Default response

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

`PATCH /api/v1/fleet/setup_experience`

### Parameters

| Name                           | Type    | In    | Description                                                                                 |
| -------------          | ------  | ----  | --------------------------------------------------------------------------------------      |
| team_id                        | integer | body  | The team ID to apply the settings to. Settings applied to hosts in no team if absent.       |
| enable_end_user_authentication | boolean | body  | When enabled, require end users to authenticate with your identity provider (IdP) when they set up their new macOS hosts. |
| enable_release_device_manually | boolean | body  | When enabled, you're responsible for sending the DeviceConfigured command.|

### Example

`PATCH /api/v1/fleet/setup_experience`

#### Request body

```json
{
  "team_id": 1,
  "enabled_end_user_authentication": true
}
```

#### Default response

`Status: 204`


## Upload an EULA file

_Available in Fleet Premium_

Upload an EULA that will be shown during the DEP flow.

`POST /api/v1/fleet/setup_experience/eula`

### Parameters

| Name | Type | In   | Description                                       |
| ---- | ---- | ---- | ------------------------------------------------- |
| eula | file | form | **Required**. A PDF document containing the EULA. |

### Example

`POST /api/v1/fleet/setup_experience/eula`

#### Request headers

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

#### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="eula"; filename="eula.pdf"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------f02md47480und42y--
```

#### Default response

`Status: 200`


## Get metadata about an EULA file

_Available in Fleet Premium_

Get information about the EULA file that was uploaded to Fleet. If no EULA was previously uploaded, this endpoint returns a `404` status code.

`GET /api/v1/fleet/setup_experience/eula/metadata`

### Example

`GET /api/v1/fleet/setup_experience/eula/metadata`

#### Default response

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

`DELETE /api/v1/fleet/setup_experience/eula/:token`

### Parameters

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| token | string | path  | **Required** The token of the EULA file. |

### Example

`DELETE /api/v1/fleet/setup_experience/eula/AA598E2A-7952-46E3-B89D-526D45F7E233`

#### Default response

`Status: 200`


## Download an EULA file

_Available in Fleet Premium_

Download an EULA file

`GET /api/v1/fleet/setup_experience/eula/:token`

### Parameters

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| token | string | path  | **Required** The token of the EULA file. |

### Example

`GET /api/v1/fleet/setup_experience/eula/AA598E2A-7952-46E3-B89D-526D45F7E233`

#### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/pdf
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

---

<meta name="description" value="Documentation for Fleet's setup experience REST API endpoints.">
<meta name="pageOrderInSection" value="140">