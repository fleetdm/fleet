package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
)

func makeRequest(t *testing.T, server http.Handler, verb, endpoint string, body interface{}, cookie string) *httptest.ResponseRecorder {
	params, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	buff.Write(params)
	request, _ := http.NewRequest(verb, endpoint, buff)
	if cookie != "" {
		request.Header.Set("Cookie", cookie)
	}
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	return response
}

func createTestServer(ds kolide.Datastore) http.Handler {
	return createTestServerWithSMTP(ds, kolide.NewMockSMTPConnectionPool())
}

func createTestServerWithSMTP(ds kolide.Datastore, pool kolide.SMTPConnectionPool) http.Handler {
	return CreateServer(
		ds,
		pool,
		os.Stderr,
		&MockOsqueryResultHandler{},
		&MockOsqueryStatusHandler{},
	)
}

func createTestDatastore(t *testing.T) kolide.Datastore {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	return ds
}

func createTestPacksAndQueries(t *testing.T, ds kolide.Datastore) kolide.Datastore {
	var err error

	pack1 := &kolide.Pack{
		Name: "pack1",
	}
	err = ds.NewPack(pack1)
	assert.Nil(t, err)

	pack2 := &kolide.Pack{
		Name: "pack2",
	}
	err = ds.NewPack(pack2)
	assert.Nil(t, err)

	query1 := &kolide.Query{
		Name:  "query1",
		Query: "select * from time;",
	}
	err = ds.NewQuery(query1)
	assert.Nil(t, err)

	query2 := &kolide.Query{
		Name:  "query2",
		Query: "select * from time;",
	}
	err = ds.NewQuery(query2)
	assert.Nil(t, err)

	query3 := &kolide.Query{
		Name:  "query3",
		Query: "select * from time;",
	}
	err = ds.NewQuery(query3)
	assert.Nil(t, err)

	err = ds.AddQueryToPack(query1, pack1)
	assert.Nil(t, err)

	err = ds.AddQueryToPack(query2, pack1)
	assert.Nil(t, err)

	err = ds.AddQueryToPack(query3, pack2)
	assert.Nil(t, err)

	return ds
}

func createTestUsers(t *testing.T, ds kolide.Datastore) kolide.Datastore {
	type NewUserParams struct {
		Username           string
		Password           string
		Email              string
		Admin              bool
		NeedsPasswordReset bool
	}

	users := []NewUserParams{
		NewUserParams{
			Username:           "admin1",
			Password:           "foobar",
			Email:              "admin@kolide.co",
			Admin:              true,
			NeedsPasswordReset: false,
		},
		NewUserParams{
			Username:           "admin2",
			Password:           "foobar",
			Email:              "admin2@kolide.co",
			Admin:              true,
			NeedsPasswordReset: false,
		},
		NewUserParams{
			Username:           "user1",
			Password:           "foobar",
			Email:              "user1@kolide.co",
			Admin:              false,
			NeedsPasswordReset: false,
		},
		NewUserParams{
			Username:           "user2",
			Password:           "foobar",
			Email:              "user2@kolide.co",
			Admin:              false,
			NeedsPasswordReset: true,
		},
	}

	for _, user := range users {
		newUser, err := kolide.NewUser(
			user.Username,
			user.Password,
			user.Email,
			user.Admin,
			user.NeedsPasswordReset,
		)
		if err != nil {
			t.Fatal(err)
		}
		newUser, err = ds.NewUser(newUser)
		if err != nil {
			t.Fatal(err)
		}
	}
	return ds
}
