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
			return []*StoredError{{Error: []byte("invalid")}}, nil
		}
		ctx := NewContext(context.Background(), eh)
		_, err := Aggregate(ctx)
		require.Error(t, err)
	})

	t.Run("returns an aggregation of the errors stored", func(t *testing.T) {
		eh := MockHandler{}
		eh.RetrieveImpl = func(flush bool) ([]*StoredError, error) {
			return []*StoredError{
				{Count: 10, Error: []byte(`{"cause": {"stack": ["a", "b", "c", "d"]}}`)},
				{Count: 20, Error: []byte(`{"cause": {"stack": ["x", "y"]}}`)},
			}, nil
		}
		ctx := NewContext(context.Background(), eh)
		rawAgg, err := Aggregate(ctx)
		require.NoError(t, err)

		var aggs []errorAgg
		err = json.Unmarshal(rawAgg, &aggs)
		require.NoError(t, err)
		require.Len(t, aggs, 2)
		require.Equal(t, aggs[0].Count, 10)
		require.Equal(t, aggs[0].Loc, []string{"a", "b", "c"})
		require.Equal(t, aggs[1].Count, 20)
		require.Equal(t, aggs[1].Loc, []string{"x", "y"})
	})
}
