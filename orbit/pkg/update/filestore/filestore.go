package filestore

import (
	"encoding/json"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/secure"
	"github.com/pkg/errors"
	"github.com/theupdateframework/go-tuf/client"
)

type fileStore struct {
	filename string
	metadata metadataMap
}

type metadataMap map[string]json.RawMessage

func New(filename string) (client.LocalStore, error) {
	store := &fileStore{filename: filename}
	if err := store.readData(); err != nil {
		return nil, err
	}

	return store, nil
}

// SetMeta stores the provided metadata.
func (s *fileStore) SetMeta(name string, meta json.RawMessage) error {
	if s.metadata == nil {
		if err := s.readData(); err != nil {
			return err
		}
	}

	s.metadata[name] = meta
	if err := s.writeData(); err != nil {
		return err
	}

	return nil
}

// GetMeta returns all of the saved metadata.
func (s *fileStore) GetMeta() (map[string]json.RawMessage, error) {
	if s.metadata == nil {
		if err := s.readData(); err != nil {
			return nil, err
		}
	}

	return s.metadata, nil
}

func (s *fileStore) readData() error {
	stat, err := os.Stat(s.filename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Wrap(err, "stat file store")
	} else if errors.Is(err, os.ErrNotExist) {
		// initialize empty
		s.metadata = metadataMap{}
		return nil
	} else if !stat.Mode().IsRegular() {
		return errors.New("expected file store to be regular file")
	}

	f, err := os.Open(s.filename)
	if err != nil {
		return errors.Wrap(err, "open file store")
	}
	defer f.Close()

	var meta metadataMap
	if err := json.NewDecoder(f).Decode(&meta); err != nil {
		return errors.Wrap(err, "read file store")
	}

	s.metadata = meta
	return nil
}

func (s *fileStore) writeData() error {
	f, err := secure.OpenFile(s.filename, os.O_RDWR|os.O_CREATE, constant.DefaultFileMode)
	if err != nil {
		return errors.Wrap(err, "open file store")
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(s.metadata); err != nil {
		return errors.Wrap(err, "write file store")
	}
	if err := f.Sync(); err != nil {
		return errors.Wrap(err, "sync file store")
	}

	return nil
}
