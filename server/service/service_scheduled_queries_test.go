package service

import (
	"testing"

	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/datastore/inmem"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/kolide/kolide-ose/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestGetScheduledQueriesInPack(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)
	ctx := context.Background()

	u1 := test.NewUser(t, ds, "Admin", "admin", "admin@kolide.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	q2 := test.NewQuery(t, ds, "bar", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false)

	queries, err := svc.GetScheduledQueriesInPack(ctx, p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, queries, 1)
	assert.Equal(t, sq1.ID, queries[0].ID)

	test.NewScheduledQuery(t, ds, p1.ID, q2.ID, 60, false, false)
	test.NewScheduledQuery(t, ds, p1.ID, q2.ID, 60, true, false)

	queries, err = svc.GetScheduledQueriesInPack(ctx, p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, queries, 3)
}

func TestGetScheduledQuery(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)
	ctx := context.Background()

	u1 := test.NewUser(t, ds, "Admin", "admin", "admin@kolide.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false)

	query, err := svc.GetScheduledQuery(ctx, sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)
}

func TestModifyScheduledQuery(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)
	ctx := context.Background()

	u1 := test.NewUser(t, ds, "Admin", "admin", "admin@kolide.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false)

	query, err := svc.GetScheduledQuery(ctx, sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)

	query.Interval = uint(120)
	query, err = svc.ModifyScheduledQuery(ctx, query)
	assert.Equal(t, uint(120), query.Interval)

	queryVerify, err := svc.GetScheduledQuery(ctx, sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(120), queryVerify.Interval)
}

func TestDeleteScheduledQuery(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)
	ctx := context.Background()

	u1 := test.NewUser(t, ds, "Admin", "admin", "admin@kolide.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false)

	query, err := svc.GetScheduledQuery(ctx, sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)

	err = svc.DeleteScheduledQuery(ctx, sq1.ID)
	require.Nil(t, err)

	_, err = svc.GetScheduledQuery(ctx, sq1.ID)
	require.NotNil(t, err)
}
