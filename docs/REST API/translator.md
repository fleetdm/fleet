# Translator

## Translate IDs

Transforms a host name into a host id. For example, the Fleet UI use this endpoint when sending live queries to a set of hosts.

`POST /api/v1/fleet/translate`

### Parameters

| Name  | Type  | In   | Description                              |
| ----- | ----- | ---- | ---------------------------------------- |
| array | array | body | **Required** list of items to translate. |

### Example

`POST /api/v1/fleet/translate`

#### Request body

```json
{
  "list": [
    {
      "type": "user",
      "payload": {
        "identifier": "some@email.com"
      }
    },
    {
      "type": "label",
      "payload": {
        "identifier": "labelA"
      }
    },
    {
      "type": "team",
      "payload": {
        "identifier": "team1"
      }
    },
    {
      "type": "host",
      "payload": {
        "identifier": "host-ABC"
      }
    }
  ]
}
```

#### Default response

`Status: 200`

```json
{
  "list": [
    {
      "type": "user",
      "payload": {
        "identifier": "some@email.com",
        "id": 32
      }
    },
    {
      "type": "label",
      "payload": {
        "identifier": "labelA",
        "id": 1
      }
    },
    {
      "type": "team",
      "payload": {
        "identifier": "team1",
        "id": 22
      }
    },
    {
      "type": "host",
      "payload": {
        "identifier": "host-ABC",
        "id": 45
      }
    }
  ]
}
```
---

---

<meta name="description" value="Documentation for Fleet's translator REST API endpoints.">
<meta name="pageOrderInSection" value="175">