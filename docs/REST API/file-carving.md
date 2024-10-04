# File carving

Fleet supports osquery's file carving functionality as of Fleet 3.3.0. This allows the Fleet server to request files (and sets of files) from Fleet's agent (fleetd), returning the full contents to Fleet.

To initiate a file carve using the Fleet API, you can use the [live query](#run-live-query) endpoint to run a query against the `carves` table.

For more information on executing a file carve in Fleet, go to the [File carving with Fleet docs](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/File-carving.md).

## List carves

Retrieves a list of the non expired carves. Carve contents remain available for 24 hours after the first data is provided from the osquery client.

`GET /api/v1/fleet/carves`

### Parameters

| Name            | Type    | In    | Description                                                                                                                    |
|-----------------|---------|-------|--------------------------------------------------------------------------------------------------------------------------------|
| page            | integer | query | Page number of the results to fetch.                                                                                           |
| per_page        | integer | query | Results per page.                                                                                                              |
| order_key       | string  | query | What to order results by. Can be any field listed in the `results` array example below.                                        |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Valid options are 'asc' or 'desc'. Default is 'asc'. |
| after           | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used.                       |
| expired         | boolean | query | Include expired carves (default: false)                                                                                        |

### Example

`GET /api/v1/fleet/carves`

#### Default response

`Status: 200`

```json
{
  "carves": [
    {
      "id": 1,
      "created_at": "2021-02-23T22:52:01Z",
      "host_id": 7,
      "name": "macbook-pro.local-2021-02-23T22:52:01Z-fleet_distributed_query_30",
      "block_count": 1,
      "block_size": 2000000,
      "carve_size": 2048,
      "carve_id": "c6958b5f-4c10-4dc8-bc10-60aad5b20dc8",
      "request_id": "fleet_distributed_query_30",
      "session_id": "065a1dc3-40ad-441c-afff-80c2ad7dac28",
      "expired": false,
      "max_block": 0
    },
    {
      "id": 2,
      "created_at": "2021-02-23T22:53:03Z",
      "host_id": 7,
      "name": "macbook-pro.local-2021-02-23T22:53:03Z-fleet_distributed_query_31",
      "block_count": 2,
      "block_size": 2000000,
      "carve_size": 3400704,
      "carve_id": "2b9170b9-4e11-4569-a97c-2f18d18bec7a",
      "request_id": "fleet_distributed_query_31",
      "session_id": "f73922ed-40a4-4e98-a50a-ccda9d3eb755",
      "expired": false,
      "max_block": 1,
      "error": "S3 multipart carve upload: EntityTooSmall: Your proposed upload is smaller than the minimum allowed object size"
    }
  ]
}
```

## Get carve

Retrieves the specified carve.

`GET /api/v1/fleet/carves/:id`

### Parameters

| Name | Type    | In   | Description                           |
| ---- | ------- | ---- | ------------------------------------- |
| id   | integer | path | **Required.** The desired carve's ID. |

### Example

`GET /api/v1/fleet/carves/1`

#### Default response

`Status: 200`

```json
{
  "carve": {
    "id": 1,
    "created_at": "2021-02-23T22:52:01Z",
    "host_id": 7,
    "name": "macbook-pro.local-2021-02-23T22:52:01Z-fleet_distributed_query_30",
    "block_count": 1,
    "block_size": 2000000,
    "carve_size": 2048,
    "carve_id": "c6958b5f-4c10-4dc8-bc10-60aad5b20dc8",
    "request_id": "fleet_distributed_query_30",
    "session_id": "065a1dc3-40ad-441c-afff-80c2ad7dac28",
    "expired": false,
    "max_block": 0
  }
}
```

## Get carve block

Retrieves the specified carve block. This endpoint retrieves the data that was carved.

`GET /api/v1/fleet/carves/:id/block/:block_id`

### Parameters

| Name     | Type    | In   | Description                                 |
| -------- | ------- | ---- | ------------------------------------------- |
| id       | integer | path | **Required.** The desired carve's ID.       |
| block_id | integer | path | **Required.** The desired carve block's ID. |

### Example

`GET /api/v1/fleet/carves/1/block/0`

#### Default response

`Status: 200`

```json
{
    "data": "aG9zdHMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA..."
}
```

---

<meta name="description" value="Documentation for Fleet's file carving REST API endpoints.">
<meta name="pageOrderInSection" value="50">