package e2e

import (
	"net/http"
	"net/http/httptest"
)

// HandlerClient behaves like an HTTP client but merely routes to an http.Handler.
type HandlerClient struct {
	handler http.Handler
}

func NewHandlerClient(handler http.Handler) *HandlerClient {
	return &HandlerClient{handler: handler}
}

// Do routes HTTP requests to an http.Handler using an httptest.NewRecorder.
func (c *HandlerClient) Do(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, r)
	return rec.Result(), nil
}
