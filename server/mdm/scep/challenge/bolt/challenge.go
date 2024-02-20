package challengestore

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
)

type Depot struct {
	*bolt.DB
}

const challengeBucket = "scep_challenges"

// NewBoltDepot creates a depot.Depot backed by BoltDB.
func NewBoltDepot(db *bolt.DB) (*Depot, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(challengeBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Depot{db}, nil
}

func (db *Depot) SCEPChallenge() (string, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}

	challenge := base64.StdEncoding.EncodeToString(key)
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(challengeBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found!", challengeBucket)
		}
		return bucket.Put([]byte(challenge), []byte(challenge))
	})
	if err != nil {
		return "", err
	}
	return challenge, nil
}

func (db *Depot) HasChallenge(pw string) (bool, error) {
	tx, err := db.Begin(true)
	if err != nil {
		return false, errors.Join(err, errors.New("begin transaction"))
	}
	bkt := tx.Bucket([]byte(challengeBucket))
	if bkt == nil {
		return false, fmt.Errorf("bucket %q not found!", challengeBucket)
	}

	key := []byte(pw)
	var matches bool
	if chal := bkt.Get(key); chal != nil {
		if err := bkt.Delete(key); err != nil {
			return false, err
		}
		matches = true
	}

	return matches, tx.Commit()
}
