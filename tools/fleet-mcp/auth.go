package main

import (
	"crypto/subtle"
	"net/http"
)

// bearerAuthMiddleware rejects requests whose Authorization header does not
// match "Bearer <token>", returning 401 Unauthorized. The comparison uses
// crypto/subtle.ConstantTimeCompare to prevent timing side-channel attacks.
func bearerAuthMiddleware(token string, next http.Handler) http.Handler {
	expected := []byte("Bearer " + token)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, expected) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
