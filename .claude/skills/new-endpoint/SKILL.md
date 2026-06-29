---
name: new-endpoint
description: Scaffold a new Fleet API endpoint with request/response structs, endpoint function, service method, datastore interface, handler registration, and test stubs.
allowed-tools: Read, Write, Edit, Grep, Glob
model: sonnet
effort: high
disable-model-invocation: true
---

# Scaffold a New Fleet API Endpoint

Create a new API endpoint for: $ARGUMENTS

## Process

### 1. Gather Requirements
- Resource name and HTTP method (GET/POST/PATCH/DELETE)
- URL path (e.g., `/api/_version_/fleet/resource`)
- Request body fields (if any)
- Response body fields
- Which API version (use `StartingAtVersion("2022-04")` for new endpoints)
- Does it need a datastore method?

### 2. Read Reference Patterns
Read `server/service/vulnerabilities.go` for the canonical request/response/endpoint pattern:
- Request struct with json tags
- Response struct with `Err error` field and `Error()` method
- Endpoint function with `(ctx, request, svc)` signature

Read `server/service/handler.go` to find where to register the new endpoint.

### 3. Create Request/Response Structs
```go
type myResourceRequest struct {
    ID   uint   `url:"id"`
    Name string `json:"name"`
}

type myResourceResponse struct {
    Resource *fleet.Resource `json:"resource,omitempty"`
    Err      error           `json:"error,omitempty"`
}

func (r myResourceResponse) Error() error { return r.Err }
```

### 4. Create Endpoint Function
```go
func myResourceEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
    req := request.(*myResourceRequest)
    result, err := svc.MyResource(ctx, req.ID)
    if err != nil {
        return myResourceResponse{Err: err}, nil
    }
    return myResourceResponse{Resource: result}, nil
}
```

### 5. Add Service Interface Method
In `server/fleet/service.go`, add the method to the `Service` interface.

### 6. Implement Service Method
In the appropriate `server/service/*.go` file:
- Start with `svc.authz.Authorize(ctx, &fleet.Entity{}, fleet.ActionRead)`
- Implement business logic
- Wrap errors with `ctxerr.Wrap`

### 7. Add Datastore Interface Method (if needed)
In `server/fleet/datastore.go`, add the method to the `Datastore` interface.

### 8. Register in handler.go
```go
ue.StartingAtVersion("2022-04").GET("/api/_version_/fleet/resource", myResourceEndpoint, myResourceRequest{})
```

### 9. Create Test Stubs
- Unit test with mock datastore in `server/service/*_test.go`
- Integration test stub if it touches the database

### 10. Verify
- Run `go build ./...` to check compilation
- Run `go test ./server/service/` to check mocks are satisfied
