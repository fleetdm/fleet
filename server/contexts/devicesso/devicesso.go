// Package devicesso provides a context key for the Fleet Desktop device SSO
// session ID.
package devicesso

import "context"

type key int

const sessionIDKey key = 0

func NewContext(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

func FromContext(ctx context.Context) string {
	sessionID, _ := ctx.Value(sessionIDKey).(string)
	return sessionID
}
