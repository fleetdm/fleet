// Package multipartform provides helpers for parsing multipart/form-data
// request bodies. It lives in its own subpackage so it can depend on
// server/contexts/logging without causing an import cycle with
// server/platform/http.
package multipartform

import (
	"context"
	"net/http"
)

// Parse parses the multipart form on the request.
func Parse(ctx context.Context, r *http.Request, maxMemory int64) error {
	return r.ParseMultipartForm(maxMemory) //nolint:gosec // G120: bounded by the route limit
}
