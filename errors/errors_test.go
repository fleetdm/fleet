package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"gopkg.in/go-playground/validator.v8"
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

func TestDatabaseError(t *testing.T) {
	err := errors.New("Foo error")
	kolideErr := DatabaseError(err)

	expect := &KolideError{
		Err:            err,
		StatusCode:     http.StatusInternalServerError,
		PublicMessage:  "Database error",
		PrivateMessage: "Foo error",
	}
	assert.Equal(t, expect, kolideErr)
}

func TestReturnErrorUnspecified(t *testing.T) {
	r := gin.New()
	r.POST("/foo", func(c *gin.Context) {
		ReturnError(c, errors.New("foo"))
	})

	req, _ := http.NewRequest("POST", "/foo", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("Should respond with 500, got %d", resp.Code)
	}

	expect := `{"message": "Unspecified error"}`
	assert.JSONEq(t, expect, resp.Body.String())
}

func TestReturnErrorKolideError(t *testing.T) {
	r := gin.New()
	r.POST("/foo", func(c *gin.Context) {
		ReturnError(c, &KolideError{
			StatusCode:    http.StatusUnauthorized,
			PublicMessage: "Some error",
		})
	})

	req, _ := http.NewRequest("POST", "/foo", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Errorf("Should respond with 403, got %d", resp.Code)
	}

	expect := `{"message": "Some error"}`
	assert.JSONEq(t, expect, resp.Body.String())
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

func TestReturnErrorValidationError(t *testing.T) {
	r := gin.New()

	type Foo struct {
		Email    string `json:"email_foo" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	validate := validator.New(&validator.Config{TagName: "validate", FieldNameTag: "json"})

	r.POST("/foo", func(c *gin.Context) {
		ReturnError(c, validate.Struct(&Foo{Email: "foo", Password: ""}))
	})

	req, _ := http.NewRequest("POST", "/foo", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != StatusUnprocessableEntity {
		t.Errorf("Should respond with 422, got %d", resp.Code)
	}

	var bodyJson map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &bodyJson); err != nil {
		t.Errorf("Error unmarshaling JSON: %s", err.Error())
	}

	assert.Equal(t, "Validation error", bodyJson["message"])

	fields, ok := bodyJson["errors"].([]interface{})
	if !ok {
		t.Errorf("Unexpected type for errors")
	}

	// The error fields must be copied from []interface{} to
	// []map[string][string] before we can sort
	compFields := make(errorFields, 0, 0)
	for _, field := range fields {
		field := field.(map[string]interface{})
		compFields = append(
			compFields,
			errorField{
				"code":    field["code"].(string),
				"field":   field["field"].(string),
				"message": field["message"].(string),
			})
	}

	expect := errorFields{
		{"code": "invalid", "field": "email_foo", "message": "email"},
		{"code": "invalid", "field": "password", "message": "required"},
	}

	// Sort to standardize ordering before comparison
	sort.Sort(compFields)
	sort.Sort(expect)

	assert.Equal(t, expect, compFields)
}

func TestReturnErrorGormError(t *testing.T) {
	r := gin.New()

	r.POST("/foo", func(c *gin.Context) {
		err := gorm.Errors{}
		err.Add(gorm.ErrInvalidSQL)
		ReturnError(c, err)
	})

	req, _ := http.NewRequest("POST", "/foo", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("Should respond with 500, got %d", resp.Code)
	}

	assert.JSONEq(t, `{"message": "Database error"}`, resp.Body.String())
}
