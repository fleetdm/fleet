package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	kolideErr := New("Public message", "Private message")

	expect := &KolideError{
		Err:            nil,
		StatusCode:     http.StatusInternalServerError,
		PublicMessage:  "Public message",
		PrivateMessage: "Private message",
	}
	assert.Equal(t, expect, kolideErr)
}

func TestNewWithStatus(t *testing.T) {
	kolideErr := NewWithStatus(http.StatusUnauthorized, "Public message", "Private message")

	expect := &KolideError{
		Err:            nil,
		StatusCode:     http.StatusUnauthorized,
		PublicMessage:  "Public message",
		PrivateMessage: "Private message",
	}
	assert.Equal(t, expect, kolideErr)
}

func TestNewFromError(t *testing.T) {
	err := errors.New("Foo error")
	kolideErr := NewFromError(err, StatusUnprocessableEntity, "Public error")

	assert.Equal(t, "Public error", kolideErr.Error())

	expect := &KolideError{
		Err:            err,
		StatusCode:     StatusUnprocessableEntity,
		PublicMessage:  "Public error",
		PrivateMessage: "Foo error",
	}
	assert.Equal(t, expect, kolideErr)
}

// These types and functions for performing an unordered comparison on a
// []map[string]string] as parsed from the error JSON
type errorField map[string]string
type errorFields []errorField

func (e errorFields) Len() int {
	return len(e)
}

func (e errorFields) Less(i, j int) bool {
	return e[i]["field"] <= e[j]["field"] &&
		e[i]["code"] <= e[j]["code"] &&
		e[i]["message"] <= e[j]["message"]
}

func (e errorFields) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
