// Package api implements HTTP handlers for the NanoDEP API.
package api

import (
	"encoding/json"
	"net/http"
)

// jsonError writes err as JSON and to w.
func jsonError(w http.ResponseWriter, err error) {
	jsonErr := &struct {
		Err string `json:"error"`
	}{Err: err.Error()}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(jsonErr)
}
