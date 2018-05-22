package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestDecodeDeleteLabelRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/labels/{name}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeleteLabelRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deleteLabelRequest)
		assert.Equal(t, "foo", params.Name)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/kolide/labels/foo", nil),
	)
}

func TestDecodeGetLabelRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/labels/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetLabelRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getLabelRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/kolide/labels/1", nil),
	)
}
