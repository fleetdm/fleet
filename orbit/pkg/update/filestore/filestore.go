package filestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/secure"
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

func (s *fileStore) maybeInit() error {
	if s.metadata == nil {
		return s.readData()
	}
	return nil
}

// SetMeta stores the provided metadata.
func (s *fileStore) SetMeta(name string, meta json.RawMessage) error {
	if err := s.maybeInit(); err != nil {
		return err
	}

	s.metadata[name] = meta
	if err := s.writeData(); err != nil {
		return err
	}

	return nil
}

// GetMeta returns all of the saved metadata.
func (s *fileStore) GetMeta() (map[string]json.RawMessage, error) {
	if err := s.maybeInit(); err != nil {
		return nil, err
	}

	return s.metadata, nil
}

func (s *fileStore) DeleteMeta(name string) error {
	if err := s.maybeInit(); err != nil {
		return err
	}

	delete(s.metadata, name)
	if err := s.writeData(); err != nil {
		return err
	}

	return nil
}

func (s *fileStore) Close() error {
	// Files are already closed after each operation.
	return nil
}

func (s *fileStore) readData() error {
	stat, err := os.Stat(s.filename)
	switch {
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("stat file store: %w", err)
	case errors.Is(err, os.ErrNotExist):
		// initialize empty
		s.metadata = metadataMap{}
		return nil
	case !stat.Mode().IsRegular():
		return errors.New("expected file store to be regular file")
	}

	f, err := secure.OpenFile(s.filename, os.O_RDWR|os.O_CREATE, constant.DefaultFileMode)
	if err != nil {
		return fmt.Errorf("open file store: %w", err)
	}
	defer f.Close()

	var meta metadataMap
	if err := json.NewDecoder(f).Decode(&meta); err != nil {
		return fmt.Errorf("read file store: %w", err)
	}

	s.metadata = meta
	return nil
}

func (s *fileStore) writeData() error {
	f, err := secure.OpenFile(s.filename, os.O_RDWR|os.O_CREATE, constant.DefaultFileMode)
	if err != nil {
		return fmt.Errorf("open file store: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(s.metadata); err != nil {
		return fmt.Errorf("write file store: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync file store: %w", err)
	}

	return nil
}
