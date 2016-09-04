package kitserver

import (
	"testing"

	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestGetAllQueries(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(testConfig(ds))
	assert.Nil(t, err)

	ctx := context.Background()

	queries, err := svc.GetAllQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	err = ds.NewQuery(&kolide.Query{
		Name:  "foo",
		Query: "select * from time;",
	})
	assert.Nil(t, err)

	queries, err = svc.GetAllQueries(ctx)
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
}

func TestGetQuery(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(testConfig(ds))
	assert.Nil(t, err)

	ctx := context.Background()

	query := &kolide.Query{
		Name:  "foo",
		Query: "select * from time;",
	}
	err = ds.NewQuery(query)
	assert.Nil(t, err)
	assert.NotZero(t, query.ID)

	queryVerify, err := svc.GetQuery(ctx, query.ID)
	assert.Nil(t, err)

	assert.Equal(t, query.ID, queryVerify.ID)
}

func TestNewQuery(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(testConfig(ds))
	assert.Nil(t, err)

	ctx := context.Background()

	name := "foo"
	query := "select * from time;"
	_, err = svc.NewQuery(ctx, kolide.QueryPayload{
		Name:  &name,
		Query: &query,
	})

	assert.Nil(t, err)

	queries, err := ds.Queries()
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
}

func TestModifyQuery(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(testConfig(ds))
	assert.Nil(t, err)

	ctx := context.Background()

	query := &kolide.Query{
		Name:  "foo",
		Query: "select * from time;",
	}
	err = ds.NewQuery(query)
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
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(testConfig(ds))
	assert.Nil(t, err)

	ctx := context.Background()

	query := &kolide.Query{
		Name:  "foo",
		Query: "select * from time;",
	}
	err = ds.NewQuery(query)
	assert.Nil(t, err)
	assert.NotZero(t, query.ID)

	err = svc.DeleteQuery(ctx, query.ID)
	assert.Nil(t, err)

	queries, err := ds.Queries()
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

}
