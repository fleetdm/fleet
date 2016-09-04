package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/kolide/kolide-ose/errors"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/kolide/kolide-ose/mock"
	"github.com/stretchr/testify/assert"
)

type MockOsqueryResultHandler struct{}

func (h *MockOsqueryResultHandler) HandleResultLog(log OsqueryResultLog, nodeKey string) error {
	return nil
}

type MockOsqueryStatusHandler struct{}

func (h *MockOsqueryStatusHandler) HandleStatusLog(log OsqueryStatusLog, nodeKey string) error {
	return nil
}

func TestGetAllQueries(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// try to get queries while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"GET",
		"/api/v1/kolide/queries",
		nil,
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// get queries from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"GET",
		"/api/v1/kolide/queries",
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var queries GetAllQueriesResponseBody
	err := json.NewDecoder(response.Body).Decode(&queries)
	assert.Nil(t, err)
	assert.Len(t, queries.Queries, 3)
}

func TestGetQuery(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)
	queries, err := ds.Queries()
	assert.Nil(t, err)
	assert.NotEmpty(t, queries)
	query := queries[0]

	////////////////////////////////////////////////////////////////////////////
	// try to get query while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"GET",
		fmt.Sprintf("/api/v1/kolide/queries/%d", query.ID),
		nil,
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// get query from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"GET",
		fmt.Sprintf("/api/v1/kolide/queries/%d", query.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var q GetQueryResponseBody
	err = json.NewDecoder(response.Body).Decode(&q)
	assert.Nil(t, err)
	assert.Equal(t, q.Name, query.Name)
}

func TestCreateQuery(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// try to create query while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/queries",
		CreateQueryRequestBody{
			Name:  "new query",
			Query: "select * from time;",
		},
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// create query from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/queries",
		CreateQueryRequestBody{
			Name:  "new query",
			Query: "select * from time;",
		},
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var q GetQueryResponseBody
	err := json.NewDecoder(response.Body).Decode(&q)
	assert.Nil(t, err)
	assert.Equal(t, q.Name, "new query")
}

func TestModifyQuery(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)
	queries, err := ds.Queries()
	assert.Nil(t, err)
	assert.NotEmpty(t, queries)
	query := queries[0]
	newName := "new name"

	////////////////////////////////////////////////////////////////////////////
	// try to modify query while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"PATCH",
		fmt.Sprintf("/api/v1/kolide/queries/%d", query.ID),
		ModifyQueryRequestBody{
			Name: &newName,
		},
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with a test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// modify query from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"PATCH",
		fmt.Sprintf("/api/v1/kolide/queries/%d", query.ID),
		ModifyQueryRequestBody{
			Name: &newName,
		},
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var q GetQueryResponseBody
	err = json.NewDecoder(response.Body).Decode(&q)
	assert.Nil(t, err)
	assert.Equal(t, q.Name, "new name")

	// ensure the result was persisted to the database
	query, err = ds.Query(query.ID)
	assert.Nil(t, err)
	assert.Equal(t, query.Name, "new name")
}

func TestDeleteQuery(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)
	queries, err := ds.Queries()
	assert.Nil(t, err)
	assert.NotEmpty(t, queries)
	query := queries[0]

	////////////////////////////////////////////////////////////////////////////
	// try to delete query while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"DELETE",
		fmt.Sprintf("/api/v1/kolide/queries/%d", query.ID),
		nil,
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// delete query from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"DELETE",
		fmt.Sprintf("/api/v1/kolide/queries/%d", query.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusNoContent, response.Code)

	// ensure result was persisted to the database
	query, err = ds.Query(query.ID)
	assert.NotNil(t, err)
}

func TestGetAllPacks(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// try to get packs while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"GET",
		"/api/v1/kolide/packs",
		nil,
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// get queries from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"GET",
		"/api/v1/kolide/packs",
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var packs GetAllPacksResponseBody
	err := json.NewDecoder(response.Body).Decode(&packs)
	assert.Nil(t, err)
	assert.Len(t, packs.Packs, 2)
}

func TestGetPack(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)
	packs, err := ds.Packs()
	assert.Nil(t, err)
	assert.NotEmpty(t, packs)
	pack := packs[0]

	////////////////////////////////////////////////////////////////////////////
	// try to get pack while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"GET",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		nil,
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// get pack from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"GET",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var p GetPackResponseBody
	err = json.NewDecoder(response.Body).Decode(&p)
	assert.Nil(t, err)
	assert.Equal(t, p.Name, pack.Name)
}

func TestCreatePack(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// try to create pack while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/packs",
		CreatePackRequestBody{
			Name: "new pack",
		},
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// create query from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/packs",
		CreateQueryRequestBody{
			Name: "new pack",
		},
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var p GetPackResponseBody
	err := json.NewDecoder(response.Body).Decode(&p)
	assert.Nil(t, err)
	assert.Equal(t, p.Name, "new pack")
}

func TestModifyPack(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)
	packs, err := ds.Packs()
	assert.Nil(t, err)
	assert.NotEmpty(t, packs)
	pack := packs[0]
	newName := "new name"

	////////////////////////////////////////////////////////////////////////////
	// try to modify pack while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"PATCH",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		ModifyPackRequestBody{
			Name: &newName,
		},
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with a test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// modify pack from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"PATCH",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		ModifyPackRequestBody{
			Name: &newName,
		},
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var p GetPackResponseBody
	err = json.NewDecoder(response.Body).Decode(&p)
	assert.Nil(t, err)
	assert.Equal(t, p.Name, "new name")

	// ensure the result was persisted to the database
	pack, err = ds.Pack(pack.ID)
	assert.Nil(t, err)
	assert.Equal(t, pack.Name, "new name")
}

func TestDeletePack(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)
	packs, err := ds.Packs()
	assert.Nil(t, err)
	assert.NotEmpty(t, packs)
	pack := packs[0]

	////////////////////////////////////////////////////////////////////////////
	// try to delete pack while logged out
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"DELETE",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		nil,
		"",
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// delete pack from a user account
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"DELETE",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusNoContent, response.Code)

	// ensure result was persisted to the database
	pack, err = ds.Pack(pack.ID)
	assert.NotNil(t, err)
}

func TestAddQueryToPack(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)

	packs, err := ds.Packs()
	assert.Nil(t, err)
	assert.NotEmpty(t, packs)
	pack := packs[0]

	queriesInPack, err := ds.GetQueriesInPack(pack)
	assert.Nil(t, err)
	assert.NotEmpty(t, queriesInPack)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// count queries in pack
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"GET",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var p GetPackResponseBody
	err = json.NewDecoder(response.Body).Decode(&p)
	assert.Nil(t, err)
	assert.Len(t, p.Queries, len(queriesInPack))

	////////////////////////////////////////////////////////////////////////////
	// add a query to the pack
	////////////////////////////////////////////////////////////////////////////
	query := &kolide.Query{
		Name:  "new query",
		Query: "select * from time;",
	}
	err = ds.NewQuery(query)
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"PUT",
		fmt.Sprintf("/api/v1/kolide/pack/%d/query/%d", pack.ID, query.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusNoContent, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// verify the number of queries in pack
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"GET",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	err = json.NewDecoder(response.Body).Decode(&p)
	assert.Nil(t, err)
	assert.Len(t, p.Queries, len(queriesInPack)+1)
}

func TestDeleteQueryFromPack(t *testing.T) {
	ds := createTestUsers(t, createTestPacksAndQueries(t, createTestDatastore(t)))
	server := createTestServer(ds)

	packs, err := ds.Packs()
	assert.Nil(t, err)
	assert.NotEmpty(t, packs)
	pack := packs[0]

	queriesInPack, err := ds.GetQueriesInPack(pack)
	assert.Nil(t, err)
	assert.NotEmpty(t, queriesInPack)
	query := queriesInPack[0]

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// count queries in pack
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"GET",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	var p GetPackResponseBody
	err = json.NewDecoder(response.Body).Decode(&p)
	assert.Nil(t, err)
	assert.Len(t, p.Queries, len(queriesInPack))

	////////////////////////////////////////////////////////////////////////////
	// remove a query from the pack
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"DELETE",
		fmt.Sprintf("/api/v1/kolide/pack/%d/query/%d", pack.ID, query.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusNoContent, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// verify the number of queries in pack
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"GET",
		fmt.Sprintf("/api/v1/kolide/packs/%d", pack.ID),
		nil,
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)
	err = json.NewDecoder(response.Body).Decode(&p)
	assert.Nil(t, err)
	assert.Len(t, p.Queries, len(queriesInPack)-1)
}

func TestHandleConfigDetail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mock.NewMockOsqueryStore(ctrl)

	detail := OsqueryConfigDetail{Platform: "darwin"}

	detailBytes, err := json.Marshal(detail)
	assert.NoError(t, err)
	detailJSON := json.RawMessage(detailBytes)

	host := &kolide.Host{
		NodeKey: "fake_key",
	}

	expectHost := &kolide.Host{
		NodeKey:  "fake_key",
		Platform: "darwin",
	}

	handler := OsqueryHandler{
		LabelQueryInterval: time.Minute,
	}

	expectQueries := map[string]string{
		"1": "query1",
		"3": "query3",
	}

	db.EXPECT().SaveHost(expectHost)
	db.EXPECT().LabelQueriesForHost(expectHost, gomock.Any()).
		Return(expectQueries, nil).
		Do(func(_ *kolide.Host, cutoff time.Time) {
			// Check that the cutoff is in the correct interval
			expectCutoff := time.Now().Add(-handler.LabelQueryInterval)
			allowedDelta := 5 * time.Second
			assert.WithinDuration(t, expectCutoff, cutoff, allowedDelta)
		})

	res, err := handler.handleConfigDetail(db, host, detailJSON)
	assert.NoError(t, err)
	assert.Equal(t, expectQueries, res)

}

func TestHandleConfigDetailNoSave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mock.NewMockOsqueryStore(ctrl)

	detail := OsqueryConfigDetail{Platform: "darwin"}

	detailBytes, err := json.Marshal(detail)
	assert.NoError(t, err)
	detailJSON := json.RawMessage(detailBytes)

	host := &kolide.Host{
		NodeKey:  "fake_key",
		Platform: "darwin",
	}

	expectHost := &kolide.Host{
		NodeKey:  "fake_key",
		Platform: "darwin",
	}

	handler := OsqueryHandler{
		LabelQueryInterval: time.Hour,
	}

	expectQueries := map[string]string{}

	// Note that we don't expect a call to save because the platform did
	// not change
	db.EXPECT().LabelQueriesForHost(expectHost, gomock.Any()).
		Return(expectQueries, nil).
		Do(func(_ *kolide.Host, cutoff time.Time) {
			// Check that the cutoff is in the correct interval
			expectCutoff := time.Now().Add(-handler.LabelQueryInterval)
			allowedDelta := 5 * time.Second
			assert.WithinDuration(t, expectCutoff, cutoff, allowedDelta)
		})

	res, err := handler.handleConfigDetail(db, host, detailJSON)
	assert.NoError(t, err)
	assert.Equal(t, expectQueries, res)

}

func marshalRawMessage(t *testing.T, obj interface{}) json.RawMessage {
	objBytes, err := json.Marshal(obj)
	assert.NoError(t, err)
	objJSON := json.RawMessage(objBytes)
	return objJSON
}

func TestHandleConfigDetailError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mock.NewMockOsqueryStore(ctrl)

	detail := OsqueryConfigDetail{Platform: "darwin"}

	detailBytes, err := json.Marshal(detail)
	assert.NoError(t, err)
	detailJSON := json.RawMessage(detailBytes)

	host := &kolide.Host{
		NodeKey:  "fake_key",
		Platform: "darwin",
	}

	expectHost := &kolide.Host{
		NodeKey:  "fake_key",
		Platform: "darwin",
	}

	handler := OsqueryHandler{
		LabelQueryInterval: time.Hour,
	}

	// The DB call should error in this test
	db.EXPECT().LabelQueriesForHost(expectHost, gomock.Any()).
		Return(nil, errors.New("public", "private")).
		Do(func(_ *kolide.Host, cutoff time.Time) {
			// Check that the cutoff is in the correct interval
			expectCutoff := time.Now().Add(-handler.LabelQueryInterval)
			allowedDelta := 5 * time.Second
			assert.WithinDuration(t, expectCutoff, cutoff, allowedDelta)
		})

	res, err := handler.handleConfigDetail(db, host, detailJSON)
	assert.Error(t, err)
	assert.Nil(t, res)

}

func TestHandleConfigQueryResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mock.NewMockOsqueryStore(ctrl)

	results := OsqueryConfigQueryResults{
		Results: map[string]bool{
			"1": true,
			"3": false,
			"4": true,
		},
	}

	resultsJSON := marshalRawMessage(t, results)

	host := &kolide.Host{
		NodeKey:  "fake_key",
		Platform: "darwin",
	}

	handler := OsqueryHandler{
		LabelQueryInterval: time.Hour,
	}

	db.EXPECT().RecordLabelQueryExecutions(host, results.Results, gomock.Any()).
		Return(nil).
		Do(func(_ *kolide.Host, _ map[string]bool, recordTime time.Time) {
			// Check that the cutoff is in the correct interval
			allowedDelta := 5 * time.Second
			assert.WithinDuration(t, time.Now(), recordTime, allowedDelta)
		})

	assert.NoError(t, handler.handleConfigQueryResults(db, host, resultsJSON))
}

func TestHandleConfigQueryResultsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mock.NewMockOsqueryStore(ctrl)

	results := OsqueryConfigQueryResults{
		Results: map[string]bool{
			"1": true,
			"3": false,
			"4": true,
		},
	}

	resultsJSON := marshalRawMessage(t, results)

	host := &kolide.Host{
		NodeKey:  "fake_key",
		Platform: "darwin",
	}

	handler := OsqueryHandler{
		LabelQueryInterval: time.Hour,
	}

	// DB errors this time
	db.EXPECT().RecordLabelQueryExecutions(host, results.Results, gomock.Any()).
		Return(errors.New("public", "private")).
		Do(func(_ *kolide.Host, _ map[string]bool, recordTime time.Time) {
			// Check that the cutoff is in the correct interval
			allowedDelta := 5 * time.Second
			assert.WithinDuration(t, time.Now(), recordTime, allowedDelta)
		})

	assert.Error(t, handler.handleConfigQueryResults(db, host, resultsJSON))
}
