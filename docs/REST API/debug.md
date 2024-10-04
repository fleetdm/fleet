# Debug

The Fleet server exposes a handful of API endpoints to retrieve debug information about the server itself in order to help troubleshooting. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

## Get a summary of errors

Returns a set of all the errors that happened in the server during the interval of time defined by the [logging_error_retention_period](https://fleetdm.com/docs/deploying/configuration#logging-error-retention-period) configuration.

The server only stores and returns a single instance of each error.

`GET /debug/errors`

### Parameters

| Name  | Type    | In    | Description                                                                       |
| ----- | ------- | ----- | --------------------------------------------------------------------------------- |
| flush | boolean | query | Whether or not clear the errors from Redis after reading them. Default is `false` |

### Example

`GET /debug/errors?flush=true`

#### Default response

`Status: 200`

```json
[
  {
    "count": "3",
    "chain": [
      {
        "message": "Authorization header required"
      },
      {
        "message": "missing FleetError in chain",
        "data": {
          "timestamp": "2022-06-03T14:16:01-03:00"
        },
        "stack": [
          "github.com/fleetdm/fleet/v4/server/contexts/ctxerr.Handle (ctxerr.go:262)",
          "github.com/fleetdm/fleet/v4/server/service.encodeError (transport_error.go:80)",
          "github.com/go-kit/kit/transport/http.Server.ServeHTTP (server.go:124)"
        ]
      }
    ]
  }
]
```

## Get database information

Returns information about the current state of the database; valid keys are:

- `locks`: returns transaction locking information.
- `innodb-status`: returns InnoDB status information.
- `process-list`: returns running processes (queries, etc).

`GET /debug/db/:key`

### Parameters

None.

## Get profiling information

Returns runtime profiling data of the server in the format expected by `go tools pprof`. The responses are equivalent to those returned by the Go `http/pprof` package.

Valid keys are: `cmdline`, `profile`, `symbol` and `trace`.

`GET /debug/pprof/:key`

### Parameters
None.

---

<meta name="description" value="Documentation for Fleet's debug REST API endpoints.">
<meta name="pageOrderInSection" value="40">