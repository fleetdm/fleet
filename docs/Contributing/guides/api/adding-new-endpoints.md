# API endpoints how-to

## Steps to add a new endpoint

There are two main ways to add a new endpoint to the Fleet API:

1. Building the data layer first (datastore) and then going up the stack until the API endpoint.
2. Building the API endpoint and then going down the stack until the datastore.

For the sake of ease of writing this document, we'll cover option one. If you prefer to build in more of an option two style
you can simply read the documentation bottom up.

### Step 1: Datastore

Let's say you want to add an endpoint to count the total number of hosts enrolled in Fleet. A SQL query for
gathering this data could be the following:

```sql
SELECT COUNT(*) FROM hosts
```

So let's create a function for this within the mysql datastore in the [hosts.go file](https://github.com/fleetdm/fleet/blob/main/server/datastore/mysql/hosts.go):

```go
func (ds *Datastore) CountAllHosts(ctx context.Context) (int, error) {
    var hostCount int
    err := sqlx.GetContext(ctx, ds.reader, &hostCount, `SELECT COUNT(*) FROM hosts`)
    if err != nil {
        return 0, err
    }
    return hostCount, nil
}
```

Now, this is part of the `Datastore` struct. In order to use it in an endpoint, we need this to be exposed by the `Datastore`
interface. So we add this method [to it](https://github.com/fleetdm/fleet/blob/main/server/fleet/datastore.go#L25):

```go
type Datastore interface {
	// rest of the interface here
	
    CountAllHosts(ctx context.Context) (int, error)
}
```

After adding a function to the Datastore interface, you need to run `make generate-mock` to update the mock for it. And 
now we are ready to create a method in the service.

### Step 2: Service

In order to use this new Datastore function we created, the layer that is in communication with it is the `Service` 
which is both [an interface](https://github.com/fleetdm/fleet/blob/main/server/fleet/service.go#L41) and 
[a struct](https://github.com/fleetdm/fleet/blob/main/server/service/service.go#L25) that implements that interface.

Now at this point, we are not going to be jumping around files too much. Given that this new API will count the total 
amount of hosts, it makes sense to add it to the 
[hosts.go file within the service package](https://github.com/fleetdm/fleet/blob/main/server/service/hosts.go). If this 
was a totally new feature, we could've created new files instead of adding to existing ones (the same applies to the 
datastore portion).

If you scroll around the hosts.go file, you'll notice the pattern that we'll be following. Since we are doing it 
"datastore up," you'll notice that we'll skip a few things in this step, but we'll add them in the next one.

So we add to the bottom of the file the following:

```go
/////////////////////////////////////////////////////////////////////////////////
// Count total amount of hosts
/////////////////////////////////////////////////////////////////////////////////

func (svc *Service) CountAllHosts(ctx context.Context) (int, error) {
    if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
        return nil, err
    }

    return svc.ds.CountAllHosts(ctx)
}
```

As you can see, the only thing this method adds to the table is the authorization. We assume that if a user can list 
hosts, then they could list all hosts and count them, so they can have Fleet do that math for them.

Just like with the `Datastore`, we need to add this method in the `Service` interface as well because otherwise, we won't 
be able to call this from the endpoint function itself:

```go
type Service interface {
	// rest of the interface here

	CountAllHosts(ctx context.Context) (int, error)
}
```

Now we are ready to work on the endpoint itself.

### Step 3: Endpoint

We're going to be working in the same [hosts.go file from before](https://github.com/fleetdm/fleet/blob/main/server/service/hosts.go):

```go
/////////////////////////////////////////////////////////////////////////////////
// Count total amount of hosts
/////////////////////////////////////////////////////////////////////////////////

type countAllHostsRequest struct {}

type countAllHostsResponse struct {
    Err error `json:"error,omitempty"`
    Count int `json:"count"`
}

func (r countAllHostsResponse) error() error { return r.Err }

func countAllHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
    req := request.(*countAllHostsRequest)
    count, err := svc.CountAllHosts(ctx)
    if err != nil {
        return countAllHostsResponse{Err: err}, nil
    }
    return countAllHostsResponse{Count: count}, nil
}

func (svc *Service) CountAllHosts(ctx context.Context) (int, error) {
	// ...
}
```

We added four things above:

1. The struct that represents details about the request that we might receive. It would be defined here if the request could have query 
parameters, or a JSON body, etc.
2. The struct for the response. This struct has to implement the `errorer` [interface](https://github.com/fleetdm/fleet/blob/main/server/service/transport_error.go#L21).
3. The implementation of the only method in the `errorer` interface.
4. The endpoint function handler itself.

Now it's time to expose this to be used.

### Step 4: Exposing the new API

View the whole API in the [handler.go file](https://github.com/fleetdm/fleet/blob/main/server/service/handler.go).
Most of what there is to know about what API is being exposed lives in this file and particularly within the 
`attachFleetAPIRoutes`.

Since this endpoint is a user authenticated endpoint, we'll add it at the end of the `ue` (user authenticated) endpoints 
of the function mentioned:

```go
func attachFleetAPIRoutes(r *mux.Router, svc fleet.Service, config config.FleetConfig,
	logger kitlog.Logger, limitStore throttled.GCRAStore, opts []kithttp.ServerOption,
	extra extraHandlerOpts,
) {
	// ...

	ue.GET("/api/_version_/fleet/hosts/count_all", countAllHostsEndpoint, countAllHostsRequest)
	
	// ...
}
```

And that's it! (Besides tests and documentation, which are key parts of adding a new API).

Now that the endpoint is all connected in the right places, a few things happen automatically:

1. The [decoding of the request](https://github.com/fleetdm/fleet/blob/main/server/service/endpoint_utils.go#L90) data 
(body, query params, etc.). More on this below.
2. The [encoding of the response](https://github.com/fleetdm/fleet/blob/main/server/service/transport.go#L22), including 
[error encoding/handling](https://github.com/fleetdm/fleet/blob/main/server/service/transport.go#L32) among other things.
3. [User](https://github.com/fleetdm/fleet/blob/main/server/service/endpoint_utils.go#L311) or 
[host](https://github.com/fleetdm/fleet/blob/main/server/service/endpoint_utils.go#L318) or 
[device](https://github.com/fleetdm/fleet/blob/main/server/service/endpoint_utils.go#L295) token authentication. 
4. API versioning. You probably noticed the `_version_` portion of the URL above. More on this approach 
[here](./API-Versioning.md).

One thing to note is that while we used an empty struct `countAllHostsRequest`, we could've easily skipped defining it
and used `nil`, but it was added for the sake of this documentation.

## Recap of the responsibilities for each layer

At first, it's easy to feel like Fleet might have too many layers (and it still might!), but this is the minimum we've 
defined at the time of this writing to allow for the type of testing we want to have. You might've noticed we 
didn't discuss testing at all so far, and we won't entirely because that's out of the scope of this document, but we'll 
discuss the responsibilities of each layer and how it's meant to be used in testing.

### Datastore

This is the layer where Fleet talks directly to the database. If it has a database query, 
[this](https://github.com/fleetdm/fleet/tree/main/server/datastore) is where that code should live.

The reason this layer implements the `Datastore` interface is to mock it while testing any other layer with which it 
interacts.

### Service

The [service layer](https://github.com/fleetdm/fleet/tree/main/server/service) is what implements the data access 
authorization logic and connects the HTTP layer with the datastore layer. There is some translation of data here, 
but not a lot.

The reason this layer implements the `Service` interface is to allow for another implementation of this interface to 
exist: [the enterprise/premium service that holds all the premium features](https://github.com/fleetdm/fleet/tree/main/ee).

We don't use this to mock the service layer in tests.

### HTTP Handler

The HTTP layer is where all HTTP logic lives. Where structures go from query parameters or JSON bodies to structs that 
the service layer understands.

This layer is tested in the integrations tests:

- [Core](https://github.com/fleetdm/fleet/blob/main/server/service/integration_core_test.go)
- [License](https://github.com/fleetdm/fleet/blob/main/server/service/integration_ds_only_test.go)
- [Premium features](https://github.com/fleetdm/fleet/blob/main/server/service/integration_enterprise_test.go)
- [Live queries](https://github.com/fleetdm/fleet/blob/main/server/service/integration_live_queries_test.go)
- [Logger](https://github.com/fleetdm/fleet/blob/main/server/service/integration_logger_test.go)
- [SSO](https://github.com/fleetdm/fleet/blob/main/server/service/integration_sso_test.go)

## Queries, Request bodies, and other decoding facts

Before we dive into specifics, let's discuss some context around the framework we are using and other tools that have
been implemented.

The main thing to be aware of is that Fleet uses `go-kit` underneath. With it, we get all the concepts from the 
framework, such as decoders, transport, etc. We have decided `go-kit` is no longer the best framework for Fleet to use 
anymore, but the cost of replacing it is higher than the cost of building some of the tools we'll discuss here and 
maintaining it.

The tools that were built were meant to abstract away layers that `go-kit` leaves available to its users in a way that 
makes sense for our use case. 

For instance, we found ourselves implementing extremely similar request decoding code. It varied very slightly from one 
implementation to the next, and it differed in ways that Go wasn't capable of handling at the time. So we wrote a [generic
decoder](https://github.com/fleetdm/fleet/blob/main/server/service/endpoint_utils.go#L90) that uses Go's `reflect` to 
understand what kind of request it is and the destination and decodes it correctly.

Then the problem was specifying this decoder _every time a new endpoint is created_. And making new endpoints 
also involves creating a server, user or host authentication, etc. So we abstracted this away into a 
[handful of types](https://github.com/fleetdm/fleet/blob/main/server/service/endpoint_utils.go#L280) that handles this 
in a readable way.

So with that in mind, let's look at the different tools these new things add for us:

## How to add query parameters

In order to add query parameters, you have to specify them in the Request struct with a tag 
`query:"name_of_the_parameter"` and the decoder looks for that parameter there. If the query parameter is 
optional, you can add the `,optional` suffix to the parameter name: `query:"param1,optional"`. Otherwise, it will error out the request. Here's an [example of this error out](https://github.com/fleetdm/fleet/blob/main/server/service/hosts.go#L407) in 
use.

Optional parameters can be pointers, in which case it's `nil` if omitted. If the optional parameter is not a 
pointer, the zero value for the type is set.

## How to add default listing options

There are shortcuts for bundles of query parameters such as `list_options` which results in parameters such as page and 
order to be added, among others. Here's an 
[example of this](https://github.com/fleetdm/fleet/blob/main/server/service/labels.go#L171).

## How to add URL variables

To assign a part of a URL to a variable, such as an ID for an entity, define this by specifying 
`url:"id"` in the tag in the Request struct and in-between `{}` in the URL for the handler where that variable is placed. 
For instance: `"/api/_version_/fleet/labels/{id:[0-9]+}"` and can be found in 
[this example](https://github.com/fleetdm/fleet/blob/main/server/service/handler.go#L341). URL variables cannot be 
optional.

## How is the JSON body defined

The logic here is that if there are any parameters in the Request struct that have the `json` tag, then a JSON body is 
expected, and the absence of it results in an error.

<meta name="pageOrderInSection" value="400">
<meta name="description" value="Documentation about adding new API endpoints to Fleet.">
