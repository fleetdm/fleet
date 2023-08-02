# Translator

- [Translate IDs](#translate-ids)

## Translate IDs

Transforms a host name into a host id. For example, the Fleet UI uses this endpoint when sending live queries to a set of hosts.

`POST /api/v1/fleet/translate`

#### Parameters

| Name | Type  | In   | Description                              |
| ---- | ----- | ---- | ---------------------------------------- |
| list | array | body | **Required** list of items to translate. |

#### Example

`POST /api/v1/fleet/translate`

##### Request body

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

##### Default response

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

<meta name="description" value="Learn how to retrieve Fleet database IDs for hosts, labels, teams, and users with Fleet's REST API.">
<meta name="pageOrderInSection" value="1600">