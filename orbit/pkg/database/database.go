package database

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog/log"
)

// This is the discard ratio recommended in Badger docs
// (https://pkg.go.dev/github.com/dgraph-io/badger#DB.RunValueLogGC)
const compactionDiscardRatio = 0.5

var compactionInterval = 5 * time.Minute

// BadgerDB is a wrapper around the standard badger.DB that provides a
// background compaction routine.
type BadgerDB struct {
	*badger.DB
	closeChan chan struct{}
	m         sync.Mutex // synchronizes start/stop compaction.
}

// Open opens (initializing if necessary) a Badger database at the specified
// path. Users must close the DB with Close().
func Open(path string) (*BadgerDB, error) {
	// DefaultOptions sets synchronous writes to true (maximum data integrity).
	// TODO implement logging?
	db, err := badger.Open(badger.DefaultOptions(path).WithLogger(nil))
	if err != nil {
		return nil, fmt.Errorf("open badger %s: %w", path, err)
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
		return nil, fmt.Errorf("open badger with truncate %s: %w", path, err)
	}

	b := &BadgerDB{DB: db}
	b.startBackgroundCompaction()

	return b, nil
}

// startBackgroundCompaction starts a background loop that will call the
// compaction method on the database. Badger does not do this automatically, so
// we need to be sure to do so here (or elsewhere).
func (b *BadgerDB) startBackgroundCompaction() {
	b.m.Lock()
	defer b.m.Unlock()

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
				err := b.DB.RunValueLogGC(compactionDiscardRatio)
				if err == nil || errors.Is(err, badger.ErrNoRewrite) {
					continue
				}
				log.Error().Err(err).Msg("compact badger")
				if errors.Is(err, badger.ErrDBClosed) {
					return
				}
			}
		}
	}()
}

// stopBackgroundCompaction stops the background compaction routine.
func (b *BadgerDB) stopBackgroundCompaction() {
	b.m.Lock()
	defer b.m.Unlock()

	if b.closeChan != nil {
		b.closeChan <- struct{}{}
		b.closeChan = nil
	}
}

// Close closes the database connection and releases the associated resources.
func (b *BadgerDB) Close() error {
	b.stopBackgroundCompaction()
	return b.DB.Close()
}
