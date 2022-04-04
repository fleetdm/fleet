// Package externalsvc provides types and functions to communicate with
// external services, typically via REST APIs.
package externalsvc

import "net/http"

// HTTPDoer defines the method required for an HTTP client. The net/http.Client
// standard library type satisfies this interface.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
