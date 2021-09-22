package service

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniversalDecoderIDs(t *testing.T) {
	type universalStruct struct {
		ID1        uint `url:"some-id"`
		OptionalID uint `url:"some-other-id,optional"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)
	req = mux.SetURLVars(req, map[string]string{"some-id": "999"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(999), casted.ID1)
	assert.Equal(t, uint(0), casted.OptionalID)

	// fails if non optional IDs are not provided
	req = httptest.NewRequest("POST", "/target", nil)
	_, err = decoder(context.Background(), req)
	require.Error(t, err)
}

func TestUniversalDecoderIDsAndJSON(t *testing.T) {
	type universalStruct struct {
		ID1        uint   `url:"some-id"`
		SomeString string `json:"some_string"`
	}
	decoder := makeDecoder(universalStruct{})

	body := `{"some_string": "hello"}`
	req := httptest.NewRequest("POST", "/target", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"some-id": "999"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(999), casted.ID1)
	assert.Equal(t, "hello", casted.SomeString)
}

func TestUniversalDecoderIDsAndJSONEmbedded(t *testing.T) {
	type EmbeddedJSON struct {
		SomeString string `json:"some_string"`
	}
	type UniversalStruct struct {
		ID1 uint `url:"some-id"`
		EmbeddedJSON
	}
	decoder := makeDecoder(UniversalStruct{})

	body := `{"some_string": "hello"}`
	req := httptest.NewRequest("POST", "/target", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"some-id": "999"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*UniversalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(999), casted.ID1)
	assert.Equal(t, "hello", casted.SomeString)
}

func TestUniversalDecoderIDsAndListOptions(t *testing.T) {
	type universalStruct struct {
		ID1        uint              `url:"some-id"`
		Opts       fleet.ListOptions `url:"list_options"`
		SomeString string            `json:"some_string"`
	}
	decoder := makeDecoder(universalStruct{})

	body := `{"some_string": "bye"}`
	req := httptest.NewRequest("POST", "/target?per_page=77&page=4", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"some-id": "123"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(123), casted.ID1)
	assert.Equal(t, "bye", casted.SomeString)
	assert.Equal(t, uint(77), casted.Opts.PerPage)
	assert.Equal(t, uint(4), casted.Opts.Page)
}

func TestUniversalDecoderHandlersEmbeddedAndNot(t *testing.T) {
	type EmbeddedJSON struct {
		SomeString string `json:"some_string"`
	}
	type universalStruct struct {
		ID1  uint              `url:"some-id"`
		Opts fleet.ListOptions `url:"list_options"`
		EmbeddedJSON
	}
	decoder := makeDecoder(universalStruct{})

	body := `{"some_string": "o/"}`
	req := httptest.NewRequest("POST", "/target?per_page=77&page=4", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"some-id": "123"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(123), casted.ID1)
	assert.Equal(t, "o/", casted.SomeString)
	assert.Equal(t, uint(77), casted.Opts.PerPage)
	assert.Equal(t, uint(4), casted.Opts.Page)
}

func TestUniversalDecoderListOptions(t *testing.T) {
	type universalStruct struct {
		ID1  uint              `url:"some-id"`
		Opts fleet.ListOptions `url:"list_options"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)
	req = mux.SetURLVars(req, map[string]string{"some-id": "123"})

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	_, ok := decoded.(*universalStruct)
	require.True(t, ok)
}

func TestUniversalDecoderOptionalQueryParams(t *testing.T) {
	type universalStruct struct {
		ID1 *uint `query:"some_id,optional"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Nil(t, casted.ID1)

	req = httptest.NewRequest("POST", "/target?some_id=321", nil)

	decoded, err = decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok = decoded.(*universalStruct)
	require.True(t, ok)

	require.NotNil(t, casted.ID1)
	assert.Equal(t, uint(321), *casted.ID1)
}

func TestUniversalDecoderOptionalQueryParamString(t *testing.T) {
	type universalStruct struct {
		ID1 *string `query:"some_val,optional"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Nil(t, casted.ID1)

	req = httptest.NewRequest("POST", "/target?some_val=321", nil)

	decoded, err = decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok = decoded.(*universalStruct)
	require.True(t, ok)

	require.NotNil(t, casted.ID1)
	assert.Equal(t, "321", *casted.ID1)
}

func TestUniversalDecoderOptionalQueryParamNotPtr(t *testing.T) {
	type universalStruct struct {
		ID1 string `query:"some_val,optional"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target", nil)

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, "", casted.ID1)

	req = httptest.NewRequest("POST", "/target?some_val=321", nil)

	decoded, err = decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok = decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, "321", casted.ID1)
}

func TestUniversalDecoderQueryAndListPlayNice(t *testing.T) {
	type universalStruct struct {
		ID1  *uint             `query:"some_id"`
		Opts fleet.ListOptions `url:"list_options"`
	}
	decoder := makeDecoder(universalStruct{})

	req := httptest.NewRequest("POST", "/target?per_page=77&page=4&some_id=444", nil)

	decoded, err := decoder(context.Background(), req)
	require.NoError(t, err)
	casted, ok := decoded.(*universalStruct)
	require.True(t, ok)

	assert.Equal(t, uint(77), casted.Opts.PerPage)
	assert.Equal(t, uint(4), casted.Opts.Page)
	require.NotNil(t, casted.ID1)
	assert.Equal(t, uint(444), *casted.ID1)
}
