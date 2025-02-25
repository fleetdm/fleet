package ctxerr

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregate(t *testing.T) {
	t.Run("returns an error if can't find a handler in the context", func(t *testing.T) {
		_, err := Aggregate(context.Background())
		require.Error(t, err)
	})

	t.Run("returns an error if it can't decode one of the errors stored", func(t *testing.T) {
		eh := MockHandler{}
		eh.RetrieveImpl = func(flush bool) ([]*StoredError, error) {
			return []*StoredError{{Chain: []byte("invalid")}}, nil
		}
		ctx := NewContext(context.Background(), eh)
		_, err := Aggregate(ctx)
		require.Error(t, err)
	})

	t.Run("returns an aggregation of the errors stored", func(t *testing.T) {
		eh := MockHandler{}
		eh.RetrieveImpl = func(flush bool) ([]*StoredError, error) {
			return []*StoredError{
				{Count: 10, Chain: []byte(`[{"stack": ["a", "b", "c", "d"]}]`)},
				{Count: 20, Chain: []byte(`[{"stack": ["x", "y"]}]`)},
				{Count: 30, Chain: []byte(`[{"stack": ["a", "b", "c", "d"]}, {"stack": ["x", "y"]}]`)},
				{Count: 40, Chain: []byte(`[{"stack": ["a"]}, {"stack": ["x", "y"]}]`)},
			}, nil
		}
		ctx := NewContext(context.Background(), eh)
		rawAgg, err := Aggregate(ctx)
		require.NoError(t, err)

		var aggs []ErrorAgg
		err = json.Unmarshal(rawAgg, &aggs)
		require.NoError(t, err)
		require.Equal(t, []ErrorAgg{
			{Count: 10, Loc: []string{"a", "b", "c"}},
			{Count: 20, Loc: []string{"x", "y"}},
			{Count: 30, Loc: []string{"a", "b", "c"}},
			{Count: 40, Loc: []string{"a", "x", "y"}},
		}, aggs)
	})
}
