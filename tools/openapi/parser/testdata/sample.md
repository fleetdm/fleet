# REST API

## Widgets

### List widgets

Returns all widgets.

`GET /api/v1/fleet/widgets`

#### Parameters

| Name     | Type    | In    | Description                       |
| -------- | ------- | ----- | --------------------------------- |
| page     | integer | query | Page number of the results.       |
| query    | string  | query | Search query keywords.            |

#### Example

`GET /api/v1/fleet/widgets?page=0`

##### Default response

`Status: 200`

```json
{
  "widgets": [
    {
      "id": 1,
      "name": "wrench",
      "created_at": "2024-01-01T12:00:00Z",
      "tags": ["a", "b"],
      "active": true,
      "score": 1.5,
      "parent": null
    }
  ]
}
```

### Get widget

> `GET /api/v1/fleet/old/widgets/:id` is deprecated as of Fleet 4.0.

`GET /api/v1/fleet/widgets/:id`

#### Parameters

| Name | Type    | In   | Description                 |
| ---- | ------- | ---- | --------------------------- |
| id   | integer | path | **Required**. The widget ID. |

#### Example

`GET /api/v1/fleet/widgets/1`

##### Default response

`Status: 200`

```json
{
  "widget": { "id": 1, "name": "wrench" }
}
```

### Make widget

Send a body.

`POST /api/v1/fleet/widgets`

#### Parameters

| Name  | Type   | In   | Description               |
| ----- | ------ | ---- | ------------------------- |
| name  | string | json | **Required**. The name.   |
| count | integer | body | How many.                |

##### Default response

`Status: 200`

```json
{
  "widget": { "id": 2, "name": "hammer" }
}
```

## Prose

### Just words

This section has no request line and must be skipped, not fail the parse.
