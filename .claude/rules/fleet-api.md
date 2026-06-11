---
paths:
  - "server/service/**/*.go"
---

# Fleet API endpoint conventions

These conventions apply when working on API endpoints in the service layer. Not every file in `server/service/` defines endpoints, but the patterns below should be followed whenever you create or modify one.

## Endpoint registration
Register endpoints in `server/service/handler.go`:
```go
ue.POST("/api/_version_/fleet/{resource}", endpointFunc, requestType{})
ue.GET("/api/_version_/fleet/{resource}", endpointFunc, nil)
```
`_version_` is replaced with the actual API version at runtime.

## API versioning
- `ue.EndingAtVersion("v1")` — endpoint only available in v1 and earlier
- `ue.StartingAtVersion("2022-04")` — endpoint available from 2022-04 onward
- Current versions: `v1`, `2022-04`
- New endpoints should use `StartingAtVersion("2022-04")`

## Request body size limits
Use `ue.WithRequestBodySizeLimit(N)` for endpoints accepting large payloads (e.g., bootstrap packages, installers).

## Error response pattern
Return errors in the response body, not as the second return:
```go
return xResponse{Err: err}, nil  // correct
return nil, err                   // WRONG for Fleet endpoints
```
Every response struct needs: `func (r xResponse) Error() error { return r.Err }`

## Reference example
See `server/service/vulnerabilities.go` for a complete example of the request/response/endpoint/service pattern.
