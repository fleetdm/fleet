// package badgerstore implements the go-tuf LocalStore interface using Badger
// as a backing store.
package badgerstore

import (
	"encoding/json"
	"strings"

	"github.com/dgraph-io/badger/v2"
	"github.com/theupdateframework/go-tuf/client"
)

const (
	keyPrefix = ":tuf-metadata:"
)

type badgerStore struct {
	db *badger.DB
}

// New creates the new store given the badger DB instance.
func New(db *badger.DB) client.LocalStore {
	return &badgerStore{db: db}
}

// SetMeta stores the provided metadata.
func (b *badgerStore) SetMeta(name string, meta json.RawMessage) error {
	if err := b.db.Update(func(tx *badger.Txn) error {
		if err := tx.Set([]byte(keyPrefix+name), meta); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// GetMeta returns all of the saved metadata.
func (b *badgerStore) GetMeta() (map[string]json.RawMessage, error) {
	res := make(map[string]json.RawMessage)

	// Iterate all keys with matching prefix
	// Adapted from Badger docs
	if err := b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(keyPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()

			if err := item.Value(func(v []byte) error {
				// Remember to strip prefix
				strippedKey := strings.TrimPrefix(string(k), keyPrefix)
				res[strippedKey] = json.RawMessage(v)
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return res, err
	}

	return res, nil
}
