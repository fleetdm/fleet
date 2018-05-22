package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestDecodeDeletePackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{name}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeletePackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deletePackRequest)
		assert.Equal(t, "packaday", params.Name)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/kolide/packs/packaday", nil),
	)
}

func TestDecodeGetPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getPackRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/kolide/packs/1", nil),
	)
}
