package service

import (
	"context"
	"testing"

	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/contexts/viewer"
	"github.com/kolide/kolide/server/datastore/inmem"
	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListQueries(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	queries, err := svc.ListQueries(ctx, kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	name := "foo"
	query := "select * from time"
	_, err = svc.NewQuery(ctx, kolide.QueryPayload{
		Name:  &name,
		Query: &query,
	})
	assert.Nil(t, err)

	queries, err = svc.ListQueries(ctx, kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
}

func TestGetQuery(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	query := &kolide.Query{
		Name:  "foo",
		Query: "select * from time;",
	}
	query, err = ds.NewQuery(query)
	assert.Nil(t, err)
	assert.NotZero(t, query.ID)

	queryVerify, err := svc.GetQuery(ctx, query.ID)
	assert.Nil(t, err)
	assert.Equal(t, query.ID, queryVerify.ID)
}

func TestNewQuery(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	createTestUsers(t, ds)
	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	user, err := ds.User("admin1")
	require.Nil(t, err)

	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	name := "foo"
	query := "select * from time;"
	q, err := svc.NewQuery(ctx, kolide.QueryPayload{
		Name:  &name,
		Query: &query,
	})
	assert.Nil(t, err)
	assert.Equal(t, "Test Name admin1", q.AuthorName)
	assert.Equal(t, []kolide.Pack{}, q.Packs)

	queries, err := ds.ListQueries(kolide.ListOptions{})
	assert.Nil(t, err)
	if assert.Len(t, queries, 1) {
		assert.Equal(t, "Test Name admin1", queries[0].AuthorName)
	}
}

func TestModifyQuery(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	query := &kolide.Query{
		Name:  "foo",
		Query: "select * from time;",
	}
	query, err = ds.NewQuery(query)
	assert.Nil(t, err)
	assert.NotZero(t, query.ID)

	newName := "bar"
	queryVerify, err := svc.ModifyQuery(ctx, query.ID, kolide.QueryPayload{
		Name: &newName,
	})
	assert.Nil(t, err)

	assert.Equal(t, query.ID, queryVerify.ID)
	assert.Equal(t, "bar", queryVerify.Name)
}

func TestDeleteQuery(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	query := &kolide.Query{
		Name:  "foo",
		Query: "select * from time;",
	}
	query, err = ds.NewQuery(query)
	assert.Nil(t, err)
	assert.NotZero(t, query.ID)

	err = svc.DeleteQuery(ctx, query.ID)
	assert.Nil(t, err)

	queries, err := ds.ListQueries(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 0)
}
