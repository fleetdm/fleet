# API versioning

## Why do we need to version the API?

The API is a product, just like fleetctl, and the web UI. It has its users, mostly fleetctl and the web UI but there are
also third party developers working with it.

Evolving a product inherently needs versioning. Most products create a new version with any addition to the product, but 
the API will work differently in that regard as new additions to the API won't increase the version.

New versions will be used for breaking changes and deprecating APIs. Only when a breaking change is introduced we will 
release a new version of the API.

## What kind of versioning will we use for the API?

The format for the API version we've chosen is that of a date with the following format:

```
<year>-<month>
```

The date is chosen based on the month the breaking change was introduced.

## Why is v1 still available at the time of this writing?

`v1` is the first version of the API. It existed before this text, and so it doesn't follow the versioning schema 
explained here. We still need to support it for a few months (see below on deprecation). So it'll be treated as an 
exception in the logic in the Go code while it exists.

## Why not semantic versioning?

Semantic versioning is great and we are using it in Fleet itself. However, it doesn't necessarily work for APIs since we 
are not going to be releasing a new version with every addition, just with breaking changes. So it doesn't align with our 
needs in the API level.

## How are API releases aligned with regular Fleet releases?

New versions are deployed when Fleet is released, given the nature of the product. However, not all new versions of 
Fleet will have a new release for the API.

## How long do I have until you remove a deprecated API?

6 months after the new release has been available.

## How are breaking changes introduced? (Mostly for developers)

Let's use an example. In `handler.go` we have the following endpoint:

```go
e := NewUserAuthenticatedEndpointer(svc, opts, r, "v1", "2021-11")

// other endpoints here

e.GET("/api/latest/fleet/carves/{id:[0-9]+}/block/{block_id}", getCarveBlockEndpoint, getCarveBlockRequest{})
```

The versions available are `v1` and `2021-11`. This means that the following are valid API paths:

```
/api/latest/fleet/carves/1/block/1234
/api/2021-11/fleet/carves/1/block/1234
```

Now let's say we want to introduce a breaking change to this API, so we have to specify the version this particular API 
is being supported and then add the new one that will only be available starting in the new version:

```go
e := NewUserAuthenticatedEndpointer(svc, opts, r, "v1", "2021-11", "2021-12")

// other endpoints here

e.EndingAtVersion("2021-11").GET("/api/latest/fleet/carves/{id:[0-9]+}/block/{block_id}", getCarveBlockEndpointDeprecated, getCarveBlockRequestDeprecated{})
e.StartingAtVersion("2021-12").GET("/api/latest/fleet/carves/{id:[0-9]+}/block/{block_id}", getCarveBlockEndpoint, getCarveBlockRequest{})
```

This will mean that the following are all valid paths:

```
/api/latest/fleet/carves/1/block/1234
/api/2021-11/fleet/carves/1/block/1234
/api/2021-12/fleet/carves/1/block/1234
```

However, `/api/2021-11/fleet/carves/1/block/1234` will be the old API format, and `/api/2021-12/fleet/carves/1/block/1234` 
will be the new API format.

After the date `2022-03`, version `2021-11` (and `v1` in this case) will be removed:


```go
e := NewUserAuthenticatedEndpointer(svc, opts, r, "2021-12")

// other endpoints here

e.GET("/api/latest/fleet/carves/{id:[0-9]+}/block/{block_id}", getCarveBlockEndpoint, getCarveBlockRequest{})
```

This will mean that the following are the only valid paths after this point:

```
/api/2021-12/fleet/carves/1/block/1234
```

And the code doesn't have to specify `.StartingAtVersion("2021-12")` anymore.

<meta name="pageOrderInSection" value="900">
