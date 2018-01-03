package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestDecodeGetScheduledQueriesInPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{id}/scheduled", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetScheduledQueriesInPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getScheduledQueriesInPackRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/kolide/packs/1/scheduled", nil),
	)
}
