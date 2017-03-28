package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDecoratorTest(r *testResource) {
	decs := []kolide.Decorator{
		kolide.Decorator{
			Type:  kolide.DecoratorLoad,
			Query: "select something from foo;",
		},
		kolide.Decorator{
			Type:  kolide.DecoratorLoad,
			Query: "select bar from foo;",
		},
		kolide.Decorator{
			Type:  kolide.DecoratorAlways,
			Query: "select x from y;",
		},
		kolide.Decorator{
			Type:     kolide.DecoratorInterval,
			Query:    "select name from system_info;",
			Interval: 3600,
		},
	}
	for _, d := range decs {
		r.ds.NewDecorator(&d)
	}
}

func testModifyDecorator(t *testing.T, r *testResource) {
	dec := &kolide.Decorator{
		Type:  kolide.DecoratorLoad,
		Query: "select foo from bar;",
	}
	dec, err := r.ds.NewDecorator(dec)
	require.Nil(t, err)
	buffer := bytes.NewBufferString(`{
	"payload": {
			"query": "select baz from boom;"
		}
	}`)
	req, err := http.NewRequest("PATCH", r.server.URL+fmt.Sprintf("/api/v1/kolide/decorators/%d", dec.ID), buffer)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	var decResp decoratorResponse
	err = json.NewDecoder(resp.Body).Decode(&decResp)
	require.Nil(t, err)
	require.NotNil(t, decResp.Decorator)
	assert.Equal(t, "select baz from boom;", decResp.Decorator.Query)
}

// This test verifies that we can submit the same payload twice without
// raising an error
func testModifyDecoratorNoChanges(t *testing.T, r *testResource) {
	dec := &kolide.Decorator{
		Type:  kolide.DecoratorLoad,
		Query: "select foo from bar;",
	}
	dec, err := r.ds.NewDecorator(dec)
	require.Nil(t, err)
	buffer := bytes.NewBufferString(`{
	"payload": {
	    "decorator_type": "load",
			"query": "select foo from bar;"
		}
	}`)
	req, err := http.NewRequest("PATCH", r.server.URL+fmt.Sprintf("/api/v1/kolide/decorators/%d", dec.ID), buffer)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	var decResp decoratorResponse
	err = json.NewDecoder(resp.Body).Decode(&decResp)
	require.Nil(t, err)
	require.NotNil(t, decResp.Decorator)
	assert.Equal(t, "select foo from bar;", decResp.Decorator.Query)
	assert.Equal(t, kolide.DecoratorLoad, decResp.Decorator.Type)
}

func testListDecorator(t *testing.T, r *testResource) {
	setupDecoratorTest(r)
	req, err := http.NewRequest("GET", r.server.URL+"/api/v1/kolide/decorators", nil)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var decs listDecoratorResponse
	err = json.NewDecoder(resp.Body).Decode(&decs)
	require.Nil(t, err)

	assert.Len(t, decs.Decorators, 4)
}

func testNewDecorator(t *testing.T, r *testResource) {
	buffer := bytes.NewBufferString(
		`{
		 "payload": {
			"decorator_type": "load",
			"query": "select x from y;"
			}
		}`)
	req, err := http.NewRequest("POST", r.server.URL+"/api/v1/kolide/decorators", buffer)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var dec decoratorResponse
	err = json.NewDecoder(resp.Body).Decode(&dec)
	require.Nil(t, err)
	require.NotNil(t, dec.Decorator)
	assert.Equal(t, kolide.DecoratorLoad, dec.Decorator.Type)
	assert.Equal(t, "select x from y;", dec.Decorator.Query)
}

// invalid json
func testNewDecoratorFailType(t *testing.T, r *testResource) {
	buffer := bytes.NewBufferString(
		`{
		 "payload": {
			"decorator_type": "zip",
			"query": "select x from y;"
			}
		}`)

	req, err := http.NewRequest("POST", r.server.URL+"/api/v1/kolide/decorators", buffer)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var errStruct mockValidationError
	err = json.NewDecoder(resp.Body).Decode(&errStruct)
	require.Nil(t, err)
	require.Len(t, errStruct.Errors, 1)
	assert.Equal(t, "invalid value, must be load, always, or interval", errStruct.Errors[0].Reason)

}

func testNewDecoratorFailValidation(t *testing.T, r *testResource) {
	buffer := bytes.NewBufferString(
		`{
			"payload": {
				"decorator_type": "interval",
				"query": "select x from y;",
				"interval": 3601
			}
		}`)

	req, err := http.NewRequest("POST", r.server.URL+"/api/v1/kolide/decorators", buffer)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	var errStruct mockValidationError
	err = json.NewDecoder(resp.Body).Decode(&errStruct)
	require.Nil(t, err)
	require.Len(t, errStruct.Errors, 1)
	assert.Equal(t, "must be divisible by 60", errStruct.Errors[0].Reason)
}

func testDeleteDecorator(t *testing.T, r *testResource) {
	setupDecoratorTest(r)
	req, err := http.NewRequest("DELETE", r.server.URL+"/api/v1/kolide/decorators/1", nil)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	decs, _ := r.ds.ListDecorators()
	assert.Len(t, decs, 3)
}
