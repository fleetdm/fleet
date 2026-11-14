package fleethttp

import (
	"context"

	"golang.org/x/oauth2"
)

// WithOAuth2Client returns a context that makes golang.org/x/oauth2 perform its
// token exchanges through a Fleet HTTP client. Without it, oauth2 builds its own
// client on http.DefaultTransport, which skips the private network blocking.
func WithOAuth2Client(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, NewClient())
}
