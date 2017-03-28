package datastore

import (
	"testing"

	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDecorators(t *testing.T, ds kolide.Datastore) {
	decorator := &kolide.Decorator{
		Query:    "select from something",
		Type:     kolide.DecoratorInterval,
		Interval: 60,
	}
	decorator, err := ds.NewDecorator(decorator)
	require.Nil(t, err)
	require.True(t, decorator.ID > 0)
	result, err := ds.Decorator(decorator.ID)
	require.Nil(t, err)
	assert.Equal(t, decorator.Query, result.Query)
	results, err := ds.ListDecorators()
	require.Nil(t, err)
	assert.Len(t, results, 1)

	decorator.Query = "select foo from bar;"
	err = ds.SaveDecorator(decorator)
	require.Nil(t, err)
	result, err = ds.Decorator(decorator.ID)
	require.Nil(t, err)
	assert.Equal(t, "select foo from bar;", result.Query)

	err = ds.DeleteDecorator(decorator.ID)
	require.Nil(t, err)
	result, err = ds.Decorator(decorator.ID)
	assert.NotNil(t, err)

}
