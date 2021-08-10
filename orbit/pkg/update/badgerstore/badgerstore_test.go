package badgerstore

import (
	"encoding/json"
	"testing"

	"github.com/dgraph-io/badger/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBadgerStore(t *testing.T) {
	badgerClient, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	require.NoError(t, err)

	store := New(badgerClient)

	expected := map[string]json.RawMessage{
		"test":      json.RawMessage("json"),
		"test2":     json.RawMessage("json2"),
		"root.json": json.RawMessage(`[{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"0994148e5242118d1d6a9a397a3646e0423545a37794a791c28aa39de3b0c523"}}]`),
	}

	for k, v := range expected {
		require.NoError(t, store.SetMeta(k, v))
	}

	res, err := store.GetMeta()
	require.NoError(t, err)
	assert.Equal(t, expected, res)
}
