# Integrations

## Get Apple Push Notification service (APNs)

`GET /api/v1/fleet/apns`

### Parameters

None.

### Example

`GET /api/v1/fleet/apns`

#### Default response

`Status: 200`

```json
{
  "common_name": "APSP:04u52i98aewuh-xxxx-xxxx-xxxx-xxxx",
  "serial_number": "1234567890987654321",
  "issuer": "Apple Application Integration 2 Certification Authority",
  "renew_date": "2023-09-30T00:00:00Z"
}
```

## List Apple Business Manager (ABM) tokens

_Available in Fleet Premium_

`GET /api/v1/fleet/abm_tokens`

### Parameters

None.

### Example

`GET /api/v1/fleet/abm_tokens`

#### Default response

`Status: 200`

```json
"abm_tokens": [
  {
    "id": 1,
    "apple_id": "apple@example.com",
    "org_name": "Fleet Device Management Inc.",
    "mdm_server_url": "https://example.com/mdm/apple/mdm",
    "renew_date": "2023-11-29T00:00:00Z",
    "terms_expired": false,
    "macos_team": {
      "name": "💻 Workstations",
      "id" 1
    },
    "ios_team": {
      "name": "📱🏢 Company-owned iPhones",
      "id": 2
    },
    "ipados_team": {
      "name": "🔳🏢 Company-owned iPads",
      "id": 3
    }
  }
]
```

## List Volume Purchasing Program (VPP) tokens

_Available in Fleet Premium_

`GET /api/v1/fleet/vpp_tokens`

### Parameters

None.

### Example

`GET /api/v1/fleet/vpp_tokens`

#### Default response

`Status: 200`

```json
"vpp_tokens": [
  {
    "id": 1,
    "org_name": "Fleet Device Management Inc.",
    "location": "https://example.com/mdm/apple/mdm",
    "renew_date": "2023-11-29T00:00:00Z",
    "teams": [
      {
        "name": "💻 Workstations",
        "id": 1
      },
      {
        "name": "💻🐣 Workstations (canary)",
        "id": 2
      },
      {
        "name": "📱🏢 Company-owned iPhones",
        "id": 3
      },
      {
        "name": "🔳🏢 Company-owned iPads",
        "id" 4
      }
    ],
  }
]
```

## Get Volume Purchasing Program (VPP)


> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium_

`GET /api/v1/fleet/vpp`

### Example

`GET /api/v1/fleet/vpp`

#### Default response

`Status: 200`

```json
{
  "org_name": "Acme Inc.",
  "renew_date": "2023-11-29T00:00:00Z",
  "location": "Acme Inc. Main Address"
}
```

---

<meta name="description" value="Documentation for Fleet's integrations REST API endpoints.">
<meta name="pageOrderInSection" value="80">