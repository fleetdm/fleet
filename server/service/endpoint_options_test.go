package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOptionNotFound(t *testing.T, r *testResource) {
	// id 6000 is not an actual option
	inJson := `{"options":[
  {"id":6000,"name":"aws_access_key_id","type":"string","value":"foo","read_only":false},
  {"id":7,"name":"aws_firehose_period","type":"int","value":23,"read_only":false}]}`
	buff := bytes.NewBufferString(inJson)
	req, err := http.NewRequest("PATCH", r.server.URL+"/api/v1/kolide/options", buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func testGetOptions(t *testing.T, r *testResource) {
	req, err := http.NewRequest("GET", r.server.URL+"/api/v1/kolide/options", nil)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var optsResp optionsResponse
	err = json.NewDecoder(resp.Body).Decode(&optsResp)
	require.Nil(t, err)
	require.NotNil(t, optsResp.Options)
	assert.Equal(t, "aws_access_key_id", optsResp.Options[0].Name)
}

func testModifyOptions(t *testing.T, r *testResource) {
	inJson := `{"options":[
  {"id":6,"name":"aws_access_key_id","type":"string","value":"foo","read_only":false},
  {"id":7,"name":"aws_firehose_period","type":"int","value":23,"read_only":false}]}`
	buff := bytes.NewBufferString(inJson)
	req, err := http.NewRequest("PATCH", r.server.URL+"/api/v1/kolide/options", buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)

	var optsResp optionsResponse
	err = json.NewDecoder(resp.Body).Decode(&optsResp)
	require.Nil(t, err)
	require.NotNil(t, optsResp.Options)
	require.Len(t, optsResp.Options, 2)
	assert.Equal(t, "foo", optsResp.Options[0].GetValue())
	assert.Equal(t, float64(23), optsResp.Options[1].GetValue())
}

func testModifyOptionsValidationFail(t *testing.T, r *testResource) {
	inJson := `{"options":[
  {"id":6,"name":"aws_access_key_id","type":"string","value":"foo","read_only":false},
  {"id":7,"name":"aws_firehose_period","type":"int","value":"xxs","read_only":false}]}`
	buff := bytes.NewBufferString(inJson)
	req, err := http.NewRequest("PATCH", r.server.URL+"/api/v1/kolide/options", buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	var errStruct mockValidationError
	err = json.NewDecoder(resp.Body).Decode(&errStruct)
	require.Nil(t, err)
	require.Len(t, errStruct.Errors, 1)
	assert.Equal(t, "aws_firehose_period", errStruct.Errors[0].Name)
	assert.Equal(t, "type mismatch", errStruct.Errors[0].Reason)
}
