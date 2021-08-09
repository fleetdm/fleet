package database

import (
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	compactionInterval = 5 * time.Minute
	// This is the discard ratio recommended in Badger docs
	// (https://pkg.go.dev/github.com/dgraph-io/badger#DB.RunValueLogGC)
	compactionDiscardRatio = 0.5
)

// BadgerDB is a wrapper around the standard badger.DB that provides a
// background compaction routine.
type BadgerDB struct {
	*badger.DB
	closeChan chan struct{}
}

// Open opens (initializing if necessary) a Badger database at the specified
// path. Users must close the DB with Close().
func Open(path string) (*BadgerDB, error) {
	// DefaultOptions sets synchronous writes to true (maximum data integrity).
	// TODO implement logging?
	db, err := badger.Open(badger.DefaultOptions(path).WithLogger(nil))
	if err != nil {
		return nil, errors.Wrapf(err, "open badger %s", path)
	}

	b := &BadgerDB{DB: db}
	b.startBackgroundCompaction()

	return b, nil
}

// OpenTruncate opens (initializing and/or truncating if necessary) a Badger
// database at the specified path. Users must close the DB with Close().
//
// Prefer Open in the general case, but after a bad shutdown it may be necessary
// to call OpenTruncate. This may cause data loss. Detect this situation by
// looking for badger.ErrTruncateNeeded.
func OpenTruncate(path string) (*BadgerDB, error) {
	// DefaultOptions sets synchronous writes to true (maximum data integrity).
	// TODO implement logging?
	db, err := badger.Open(badger.DefaultOptions(path).WithLogger(nil).WithTruncate(true))
	if err != nil {
		return nil, errors.Wrapf(err, "open badger with truncate %s", path)
	}

	b := &BadgerDB{DB: db}
	b.startBackgroundCompaction()

	return b, nil
}

// startBackgroundCompaction starts a background loop that will call the
// compaction method on the database. Badger does not do this automatically, so
// we need to be sure to do so here (or elsewhere).
func (b *BadgerDB) startBackgroundCompaction() {
	if b.closeChan != nil {
		panic("background compaction already running")
	}
	b.closeChan = make(chan struct{})

	go func() {
		ticker := time.NewTicker(compactionInterval)
		defer ticker.Stop()
		for {
			select {
			case <-b.closeChan:
				return

			case <-ticker.C:
				if err := b.DB.RunValueLogGC(compactionDiscardRatio); err != nil && !errors.Is(err, badger.ErrNoRewrite) {
					log.Error().Err(err).Msg("compact badger")
				}
			}
		}
	}()
}

// stopBackgroundCompaction stops the background compaction routine.
func (b *BadgerDB) stopBackgroundCompaction() {
	b.closeChan <- struct{}{}
	b.closeChan = nil
}

// Close closes the database connection and releases the associated resources.
func (b *BadgerDB) Close() error {
	b.stopBackgroundCompaction()
	return b.DB.Close()
}
