# Scripts

## Run script

Run a script on a host.

The script will be added to the host's list of upcoming activities.

The new script will run after other activities finish. Failure of one activity won't cancel other activities.

`POST /api/v1/fleet/scripts/run`

### Parameters

| Name            | Type    | In   | Description                                                                                    |
| ----            | ------- | ---- | --------------------------------------------                                                   |
| host_id         | integer | body | **Required**. The ID of the host to run the script on.                                                |
| script_id       | integer | body | The ID of the existing saved script to run. Only one of either `script_id` or `script_contents` can be included in the request; omit this parameter if using `script_contents`.  |
| script_contents | string  | body | The contents of the script to run. Only one of either `script_id` or `script_contents` can be included in the request; omit this parameter if using `script_id`. |

> Note that if both `script_id` and `script_contents` are included in the request, this endpoint will respond with an error.

### Example

`POST /api/v1/fleet/scripts/run`

#### Default response

`Status: 202`

```json
{
  "host_id": 1227,
  "execution_id": "e797d6c6-3aae-11ee-be56-0242ac120002"
}
```

## Get script result

Gets the result of a script that was executed.

### Parameters

| Name         | Type   | In   | Description                                   |
| ----         | ------ | ---- | --------------------------------------------  |
| execution_id | string | path | **Required**. The execution id of the script. |

### Example

`GET /api/v1/fleet/scripts/results/:execution_id`

#### Default Response

`Status: 200`

```json
{
  "script_contents": "echo 'hello'",
  "exit_code": 0,
  "output": "hello",
  "message": "",
  "hostname": "Test Host",
  "host_timeout": false,
  "host_id": 1,
  "execution_id": "e797d6c6-3aae-11ee-be56-0242ac120002",
  "runtime": 20,
  "created_at": "2024-09-11T20:30:24Z"
}
```

> Note: `exit_code` can be `null` if Fleet hasn't heard back from the host yet.

> Note: `created_at` is the creation timestamp of the script execution request.

## Add script

Uploads a script, making it available to run on hosts assigned to the specified team (or no team).

`POST /api/v1/fleet/scripts`

### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| script          | file    | form | **Required**. The file containing the script.    |
| team_id         | integer | form | _Available in Fleet Premium_. The team ID. If specified, the script will only be available to hosts assigned to this team. If not specified, the script will only be available to hosts on **no team**.  |

### Example

`POST /api/v1/fleet/scripts`

#### Request headers

```http
Content-Length: 306
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

#### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="team_id"

1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="script"; filename="myscript.sh"
Content-Type: application/octet-stream

echo "hello"
--------------------------f02md47480und42y--

```

#### Default response

`Status: 200`

```json
{
  "script_id": 1227
}
```

## Delete script

Deletes an existing script.

`DELETE /api/v1/fleet/scripts/:id`

### Parameters

| Name            | Type    | In   | Description                                           |
| ----            | ------- | ---- | --------------------------------------------          |
| id              | integer | path | **Required**. The ID of the script to delete. |

### Example

`DELETE /api/v1/fleet/scripts/1`

#### Default response

`Status: 204`

## List scripts

`GET /api/v1/fleet/scripts`

### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| team_id         | integer | query | _Available in Fleet Premium_. The ID of the team to filter scripts by. If not specified, it will filter only scripts that are available to hosts with no team. |
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |

### Example

`GET /api/v1/fleet/scripts`

#### Default response

`Status: 200`

```json
{
  "scripts": [
    {
      "id": 1,
      "team_id": null,
      "name": "script_1.sh",
      "created_at": "2023-07-30T13:41:07Z",
      "updated_at": "2023-07-30T13:41:07Z"
    },
    {
      "id": 2,
      "team_id": null,
      "name": "script_2.sh",
      "created_at": "2023-08-30T13:41:07Z",
      "updated_at": "2023-08-30T13:41:07Z"
    }
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}

```

## Get or download script

`GET /api/v1/fleet/scripts/:id`

### Parameters

| Name | Type    | In    | Description                                                       |
| ---- | ------- | ----  | -------------------------------------                             |
| id   | integer | path  | **Required.** The desired script's ID.                            |
| alt  | string  | query | If specified and set to "media", downloads the script's contents. |

### Example (get script)

`GET /api/v1/fleet/scripts/123`

#### Default response

`Status: 200`

```json
{
  "id": 123,
  "team_id": null,
  "name": "script_1.sh",
  "created_at": "2023-07-30T13:41:07Z",
  "updated_at": "2023-07-30T13:41:07Z"
}

```

### Example (download script)

`GET /api/v1/fleet/scripts/123?alt=media`

#### Example response headers

```http
Content-Length: 13
Content-Type: application/octet-stream
Content-Disposition: attachment;filename="2023-09-27 script_1.sh"
```

##### Example response body

`Status: 200`

```
echo "hello"
```

# Sessions

- [Get session info](#get-session-info)
- [Delete session](#delete-session)

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

<meta name="description" value="Documentation for Fleet's scripts REST API endpoints.">
<meta name="pageOrderInSection" value="130">