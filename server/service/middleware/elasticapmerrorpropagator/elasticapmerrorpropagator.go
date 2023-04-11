// Package elasticapmerrorpropagator implements a middleware that propagates errors to elastic apm
package elasticapmerrorpropagator

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"go.elastic.co/apm/v2"
)

// Middleware is the authzcheck middleware type.
type Middleware struct{}

// NewMiddleware returns a new authzcheck middleware.
func NewMiddleware() *Middleware {
	return &Middleware{}
}

func (m *Middleware) ElasticAPMErrorPropagator() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			response, err := next(ctx, req)
			if err != nil {
				apm.CaptureError(ctx, err).Send()
			}
			return response, err
		}
	}
}
