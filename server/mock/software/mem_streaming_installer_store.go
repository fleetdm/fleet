package mock

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

var (
	_ fleet.StreamingInstallerStore = (*MemStreamingInstallerStore)(nil)
	_ fleet.ETagStagingStore        = (*MemStreamingInstallerStore)(nil)
)

// MemStreamingInstallerStore is an in-memory fleet.StreamingInstallerStore
// and fleet.ETagStagingStore (and the Put/Exists part of
// fleet.SoftwareInstallerStore) for tests. Each write of a staging object
// gives it a new ETag.
type MemStreamingInstallerStore struct {
	SoftwareInstallerStore

	mu         sync.Mutex
	Staging    map[string][]byte
	Installers map[string][]byte
	// RangeRequests counts the GetStagingRange calls.
	RangeRequests int
	// PutStagingErr, if set, is returned by PutStaging after reading r.
	PutStagingErr error
	// NoETagPinning makes StagingETag return an empty ETag, like a store
	// that can't pin staging objects.
	NoETagPinning bool

	etags  map[string]string
	writes int
}

// NewMemStreamingInstallerStore returns an empty in-memory store.
func NewMemStreamingInstallerStore() *MemStreamingInstallerStore {
	s := &MemStreamingInstallerStore{
		Staging:    map[string][]byte{},
		Installers: map[string][]byte{},
		etags:      map[string]string{},
	}
	s.PutFunc = func(_ context.Context, installerID string, content io.ReadSeeker) error {
		b, err := io.ReadAll(content)
		if err != nil {
			return err
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.Installers[installerID] = b
		return nil
	}
	s.ExistsFunc = func(_ context.Context, installerID string) (bool, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, ok := s.Installers[installerID]
		return ok, nil
	}
	return s
}

type memNotFoundError struct{}

func (memNotFoundError) Error() string    { return "not found" }
func (memNotFoundError) IsNotFound() bool { return true }

func (s *MemStreamingInstallerStore) PutStaging(_ context.Context, stagingID string, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if s.PutStagingErr != nil {
		return s.PutStagingErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setStaging(stagingID, b)
	return nil
}

// setStaging must be called with s.mu held.
func (s *MemStreamingInstallerStore) setStaging(stagingID string, b []byte) {
	s.writes++
	s.Staging[stagingID] = b
	s.etags[stagingID] = fmt.Sprintf("%q", fmt.Sprintf("v%d", s.writes))
}

func (s *MemStreamingInstallerStore) GetStagingRange(_ context.Context, stagingID string, off, n int64) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RangeRequests++
	b, ok := s.Staging[stagingID]
	if !ok {
		return nil, memNotFoundError{}
	}
	end := min(off+n, int64(len(b)))
	return io.NopCloser(bytes.NewReader(b[off:end])), nil
}

func (s *MemStreamingInstallerStore) StagingSize(_ context.Context, stagingID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.Staging[stagingID]
	if !ok {
		return 0, memNotFoundError{}
	}
	return int64(len(b)), nil
}

func (s *MemStreamingInstallerStore) Promote(_ context.Context, stagingID, installerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.Staging[stagingID]
	if !ok {
		return memNotFoundError{}
	}
	if _, exists := s.Installers[installerID]; !exists {
		s.Installers[installerID] = b
	}
	delete(s.Staging, stagingID)
	delete(s.etags, stagingID)
	return nil
}

func (s *MemStreamingInstallerStore) CopyStaging(_ context.Context, srcStagingID, dstStagingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.Staging[srcStagingID]
	if !ok {
		return memNotFoundError{}
	}
	s.setStaging(dstStagingID, bytes.Clone(b))
	return nil
}

func (s *MemStreamingInstallerStore) DeleteStaging(_ context.Context, stagingID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Staging, stagingID)
	delete(s.etags, stagingID)
	return nil
}

func (s *MemStreamingInstallerStore) StagingETag(_ context.Context, stagingID string) (int64, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.Staging[stagingID]
	if !ok {
		return 0, "", memNotFoundError{}
	}
	if s.NoETagPinning {
		return int64(len(b)), "", nil
	}
	return int64(len(b)), s.etags[stagingID], nil
}

// checkETag must be called with s.mu held.
func (s *MemStreamingInstallerStore) checkETag(stagingID, etag string) error {
	if _, ok := s.Staging[stagingID]; !ok {
		return memNotFoundError{}
	}
	if s.etags[stagingID] != etag {
		return fmt.Errorf("staging %s: %w", stagingID, fleet.ErrStagingChanged)
	}
	return nil
}

func (s *MemStreamingInstallerStore) GetStagingRangeIfMatch(ctx context.Context, stagingID, etag string, off, n int64) (io.ReadCloser, error) {
	s.mu.Lock()
	err := s.checkETag(stagingID, etag)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return s.GetStagingRange(ctx, stagingID, off, n)
}

func (s *MemStreamingInstallerStore) PromoteIfMatch(_ context.Context, stagingID, etag, installerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.checkETag(stagingID, etag); err != nil {
		return err
	}
	if _, exists := s.Installers[installerID]; !exists {
		s.Installers[installerID] = s.Staging[stagingID]
	}
	return nil
}

// StagingCount returns the number of staging objects.
func (s *MemStreamingInstallerStore) StagingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Staging)
}
