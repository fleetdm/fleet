package main

import (
	"crypto/subtle"
	"net/http"
)

// bearerAuthMiddleware rejects requests whose Authorization header does not
// match "Bearer <token>", returning 401 Unauthorized. Uses constant-time
// comparison to prevent timing attacks on the token value.
func bearerAuthMiddleware(token string, next http.Handler) http.Handler {
	expected := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actual := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(actual, expected) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
