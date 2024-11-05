# HTTP Handlers

## Routes

Handlers are the starting location of any request made to Fleet. The handlers file is located at [handlers.go](/fleet/server/service/handlers.go)

Handlers functions have the following signature `func (ctx context.Context, request interface{}, svc fleet.Service) (errorer, error)`.

## Request

The `request` parameter will receive a the struct specified in handler.go, with the fields decoded according to their struct tags.

The struct tags available are:
- `json`: Extracted from the request body, decoded as JSON
- `url`: Extracted from URL parameters
- `query`: Extracted from query parameters

The request struct may also implement either of the following methods to handle custom request decoding.

- `func (request) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error)`
  This method will gives full control over the decoding of the request. This function does not modify the request it is attached to, and instead returns a new instance.
- `func (r *request) DecodeBody(ctx context.Context, body io.Reader, u url.Values, c []*x509.Certificate) error`
  This method is similar to the previous method, but the decoding of URL and query parameters is done for you, and the request body is passed in as `body`. This method modifies the request it is attached to.

## Response

The handler functions returns a response struct implementing the `errorer` interface. The response will be marshalled according to `json` struct tags. Any error that is to be viewed by the user should be returned as part of the response. The `error` return value is only for internal server errors that cannot be handled.

Response structs must have a `func error() error` method that returns a possible error, and a standardized `Err error json ``json:"error,omitempty"`` ` field to contain the details of the error.

Response structs may implement the following methods to have more control over how HTTP responses are returned.

- `func (r response) Status() int` Return a custom HTTP default status code
- `func (r response) hijackRender(ctx context.Context, w http.ResponseWriter)` Custom rendering, useful for returning an `octet-stream` instead of JSON

If a service layer method needs to return a specific HTTP status code, it should return an error wrapped by the `fleet.NewUserMessageError` function. The middleware will automatically return the HTTP code specified in the error.

If a record cannot be found in the `datastore` layer, it is usually represented by returning a datastore `notFound()` error.
