# Sessions

## Get session info

Returns the session information for the session specified by ID.

`GET /api/v1/fleet/sessions/:id`

### Parameters

| Name | Type    | In   | Description                                  |
| ---- | ------- | ---- | -------------------------------------------- |
| id   | integer | path | **Required**. The ID of the desired session. |

### Example

`GET /api/v1/fleet/sessions/1`

#### Default response

`Status: 200`

```json
{
  "session_id": 1,
  "user_id": 1,
  "created_at": "2021-03-02T18:41:34Z"
}
```

## Delete session

Deletes the session specified by ID. When the user associated with the session next attempts to access Fleet, they will be asked to log in.

`DELETE /api/v1/fleet/sessions/:id`

### Parameters

| Name | Type    | In   | Description                                  |
| ---- | ------- | ---- | -------------------------------------------- |
| id   | integer | path | **Required**. The id of the desired session. |

### Example

`DELETE /api/v1/fleet/sessions/1`

#### Default response

`Status: 200`


---

<meta name="description" value="Documentation for Fleet's sessions REST API endpoints.">
<meta name="pageOrderInSection" value="135">