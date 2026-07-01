package s3

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// stubCarveMetadataStore implements fleet.CarveStore using only in-memory
// state. It is used by tests that do not require a real S3 backend.
type stubCarveMetadataStore struct {
	carves []*fleet.CarveMetadata
	// lastListOpts records the options of the most recent ListCarves call so
	// tests can assert how CleanupCarves queries the metadata store.
	lastListOpts fleet.CarveListOptions
	// listCarvesCalled records whether ListCarves was invoked.
	listCarvesCalled bool
	// expiredIDs accumulates the ids passed to ExpireCarves.
	expiredIDs []int64
	// updateCarveCalled records whether the per-carve UpdateCarve was invoked.
	updateCarveCalled bool
}

func (s *stubCarveMetadataStore) NewCarve(_ context.Context, metadata *fleet.CarveMetadata) (*fleet.CarveMetadata, error) {
	return metadata, nil
}
func (s *stubCarveMetadataStore) UpdateCarve(_ context.Context, _ *fleet.CarveMetadata) error {
	s.updateCarveCalled = true
	return nil
}
func (s *stubCarveMetadataStore) ExpireCarves(_ context.Context, ids []int64) error {
	s.expiredIDs = append(s.expiredIDs, ids...)
	return nil
}
func (s *stubCarveMetadataStore) Carve(_ context.Context, _ int64) (*fleet.CarveMetadata, error) {
	return nil, nil
}
func (s *stubCarveMetadataStore) CarveBySessionId(_ context.Context, _ string) (*fleet.CarveMetadata, error) {
	return nil, nil
}
func (s *stubCarveMetadataStore) CarveByName(_ context.Context, _ string) (*fleet.CarveMetadata, error) {
	return nil, nil
}
func (s *stubCarveMetadataStore) ListCarves(_ context.Context, opt fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
	s.listCarvesCalled = true
	s.lastListOpts = opt
	// Mirror the real datastore: honor ordering by created_at so the S3 listing
	// window CleanupCarves derives from the first/last elements is meaningful.
	carves := append([]*fleet.CarveMetadata(nil), s.carves...)
	if opt.OrderKey == "created_at" {
		sort.SliceStable(carves, func(i, j int) bool {
			less := carves[i].CreatedAt.Before(carves[j].CreatedAt)
			if opt.OrderDirection == fleet.OrderDescending {
				return !less
			}
			return less
		})
	}
	return carves, nil
}
func (s *stubCarveMetadataStore) NewBlock(_ context.Context, _ *fleet.CarveMetadata, _ int64, _ []byte) error {
	return nil
}
func (s *stubCarveMetadataStore) GetBlock(_ context.Context, _ *fleet.CarveMetadata, _ int64) ([]byte, error) {
	return nil, nil
}
func (s *stubCarveMetadataStore) CleanupCarves(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}

// TestCleanupCarvesEmptyNonExpired verifies that CleanupCarves returns (0, nil)
// and does not panic when there are no non-expired carves in the metadata store.
func TestCleanupCarvesEmptyNonExpired(t *testing.T) {
	store := &CarveStore{
		metadatadb: &stubCarveMetadataStore{carves: nil},
	}
	cleaned, err := store.CleanupCarves(t.Context(), time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, cleaned)
}

// TestCleanupCarvesMarksS3AbsentCarvesExpired verifies the comparison path:
// carves whose S3 object no longer exists are marked expired, while carves
// that are still present in S3 are left untouched.
//
// Requires a running S3-compatible endpoint (set S3_STORAGE_TEST env var).
func TestCleanupCarvesMarksS3AbsentCarvesExpired(t *testing.T) {
	checkTestEnv(t)
	ctx := t.Context()

	const bucket = "carves-cleanup-test"
	const prefix = "carvetest/"

	// Two carves whose uploads have completed (MaxBlock == BlockCount-1).
	// carve1 will have a corresponding S3 object; carve2 will not.
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	carve1 := &fleet.CarveMetadata{ID: 1, Name: "session-with-s3", CreatedAt: baseTime, BlockCount: 1, MaxBlock: 0}
	carve2 := &fleet.CarveMetadata{ID: 2, Name: "session-without-s3", CreatedAt: baseTime.Add(30 * time.Minute), BlockCount: 1, MaxBlock: 0}

	stub := &stubCarveMetadataStore{carves: []*fleet.CarveMetadata{carve1, carve2}}

	store, err := NewCarveStore(config.S3Config{
		CarvesBucket:           bucket,
		CarvesPrefix:           prefix,
		CarvesRegion:           "localhost",
		CarvesEndpointURL:      testEndpoint,
		CarvesAccessKeyID:      testAccessKeyID,
		CarvesSecretAccessKey:  testSecretAccessKey,
		CarvesForceS3PathStyle: true,
		CarvesDisableSSL:       true,
	}, stub)
	require.NoError(t, err)

	require.NoError(t, store.CreateTestBucket(ctx, bucket))
	t.Cleanup(func() {
		if err := store.CleanupTestBucket(context.Background()); err != nil {
			t.Errorf("cleanup s3 bucket %q: %v", bucket, err)
		}
	})

	// Put carve1's object in S3 so it looks like it was uploaded.
	key1 := store.generateS3Key(carve1)
	_, err = store.s3Client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket: &store.bucket,
		Key:    &key1,
		Body:   strings.NewReader("dummy-carve-data"),
	})
	require.NoError(t, err)

	cleaned, err := store.CleanupCarves(ctx, time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, cleaned, "only carve2 (absent from S3) should be counted")

	require.False(t, carve1.Expired, "carve1 has an S3 object and must not be marked expired")
	require.True(t, carve2.Expired, "carve2 has no S3 object and must be marked expired")
}

// TestCleanupCarvesSkipsInFlightCarves verifies that carves whose multipart
// upload has not yet completed are not marked expired. An in-flight carve has
// no completed S3 object yet (ListObjectsV2 does not return in-progress
// multipart uploads), so without a completion guard it would be wrongly
// expired and become permanently undownloadable.
//
// Requires a running S3-compatible endpoint (set S3_STORAGE_TEST env var).
func TestCleanupCarvesSkipsInFlightCarves(t *testing.T) {
	checkTestEnv(t)
	ctx := t.Context()

	const bucket = "carves-inflight-test"
	const prefix = "carvetest/"

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	// completed: upload finished (MaxBlock == BlockCount-1), object absent from S3.
	completed := &fleet.CarveMetadata{ID: 1, Name: "completed-absent", CreatedAt: baseTime, BlockCount: 1, MaxBlock: 0}
	// inFlight: upload still in progress (MaxBlock < BlockCount-1), object not yet listable.
	inFlight := &fleet.CarveMetadata{ID: 2, Name: "in-flight", CreatedAt: baseTime.Add(30 * time.Minute), BlockCount: 3, MaxBlock: 0}

	stub := &stubCarveMetadataStore{carves: []*fleet.CarveMetadata{completed, inFlight}}

	store, err := NewCarveStore(config.S3Config{
		CarvesBucket:           bucket,
		CarvesPrefix:           prefix,
		CarvesRegion:           "localhost",
		CarvesEndpointURL:      testEndpoint,
		CarvesAccessKeyID:      testAccessKeyID,
		CarvesSecretAccessKey:  testSecretAccessKey,
		CarvesForceS3PathStyle: true,
		CarvesDisableSSL:       true,
	}, stub)
	require.NoError(t, err)

	require.NoError(t, store.CreateTestBucket(ctx, bucket))
	t.Cleanup(func() {
		if err := store.CleanupTestBucket(context.Background()); err != nil {
			t.Errorf("cleanup s3 bucket %q: %v", bucket, err)
		}
	})

	// Neither carve has a completed object in S3.
	cleaned, err := store.CleanupCarves(ctx, time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, cleaned, "only the completed carve absent from S3 should be expired")

	require.True(t, completed.Expired, "completed carve absent from S3 must be marked expired")
	require.False(t, inFlight.Expired, "in-flight carve must not be marked expired")
}

// TestCleanupCarvesQueriesOldestFirst verifies that CleanupCarves lists carves
// ordered by created_at ascending and capped, so a backlog drains oldest-first
// across runs without any single run making an unbounded number of requests.
func TestCleanupCarvesQueriesOldestFirst(t *testing.T) {
	// No carves: CleanupCarves returns before touching S3, so no backend needed.
	stub := &stubCarveMetadataStore{carves: nil}
	store := &CarveStore{metadatadb: stub}

	_, err := store.CleanupCarves(t.Context(), time.Now())
	require.NoError(t, err)

	require.Equal(t, "created_at", stub.lastListOpts.OrderKey, "cleanup must order carves by created_at")
	require.Equal(t, fleet.OrderAscending, stub.lastListOpts.OrderDirection, "cleanup must order carves ascending (oldest first)")
	require.False(t, stub.lastListOpts.Expired, "cleanup must only consider non-expired carves")
}

// TestCleanupCarvesExpiresAbsentAndBatchesWrites verifies that CleanupCarves
// probes each carve's object directly: carves whose object is present are kept,
// carves whose object is absent are expired, and the expirations are written in a
// single batched ExpireCarves call rather than one UpdateCarve per carve.
//
// Requires a running S3-compatible endpoint (set S3_STORAGE_TEST env var).
func TestCleanupCarvesExpiresAbsentAndBatchesWrites(t *testing.T) {
	checkTestEnv(t)
	ctx := t.Context()

	const bucket = "carves-batch-test"
	const prefix = "carvetest/"

	base := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	present := &fleet.CarveMetadata{ID: 1, Name: "present", CreatedAt: base, BlockCount: 1, MaxBlock: 0}
	absentA := &fleet.CarveMetadata{ID: 2, Name: "absent-a", CreatedAt: base.Add(time.Minute), BlockCount: 1, MaxBlock: 0}
	absentB := &fleet.CarveMetadata{ID: 3, Name: "absent-b", CreatedAt: base.Add(2 * time.Minute), BlockCount: 1, MaxBlock: 0}
	stub := &stubCarveMetadataStore{carves: []*fleet.CarveMetadata{present, absentA, absentB}}

	store, err := NewCarveStore(config.S3Config{
		CarvesBucket:           bucket,
		CarvesPrefix:           prefix,
		CarvesRegion:           "localhost",
		CarvesEndpointURL:      testEndpoint,
		CarvesAccessKeyID:      testAccessKeyID,
		CarvesSecretAccessKey:  testSecretAccessKey,
		CarvesForceS3PathStyle: true,
		CarvesDisableSSL:       true,
	}, stub)
	require.NoError(t, err)

	require.NoError(t, store.CreateTestBucket(ctx, bucket))
	t.Cleanup(func() {
		if err := store.CleanupTestBucket(context.Background()); err != nil {
			t.Errorf("cleanup s3 bucket %q: %v", bucket, err)
		}
	})

	// Only "present" has an object in S3; the other two are absent.
	key := store.generateS3Key(present)
	_, err = store.s3Client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket: &store.bucket,
		Key:    &key,
		Body:   strings.NewReader("x"),
	})
	require.NoError(t, err)

	cleaned, err := store.CleanupCarves(ctx, time.Now())
	require.NoError(t, err)
	require.Equal(t, 2, cleaned, "both carves absent from S3 should be expired")
	require.False(t, present.Expired, "carve with a present object must not be expired")
	require.True(t, absentA.Expired)
	require.True(t, absentB.Expired)

	require.ElementsMatch(t, []int64{absentA.ID, absentB.ID}, stub.expiredIDs, "expirations must be batched via ExpireCarves")
	require.False(t, stub.updateCarveCalled, "cleanup must not fall back to per-carve UpdateCarve")
}

// TestCleanupCarvesSkipsCarvesYoungerThan24h verifies that carves created within
// the last 24h are not reconciled against S3 (and so never expired), even when
// their object is absent. S3 lifecycle expiration has day granularity and never
// deletes objects that recently, so checking them is wasted work and risks wrongly
// expiring a carve whose object simply isn't listable yet. Mirrors the 24h floor
// the MySQL-backed carve store already applies.
//
// Requires a running S3-compatible endpoint (set S3_STORAGE_TEST env var).
func TestCleanupCarvesSkipsCarvesYoungerThan24h(t *testing.T) {
	checkTestEnv(t)
	ctx := t.Context()
	now := time.Now()

	const bucket = "carves-age-floor-test"
	const prefix = "carvetest/"

	// Neither carve has an object in S3 (empty bucket). Only the old one is old
	// enough to be reconciled; the recent one must be left untouched.
	old := &fleet.CarveMetadata{ID: 1, Name: "old-gone", CreatedAt: now.Add(-48 * time.Hour), BlockCount: 1, MaxBlock: 0}
	recent := &fleet.CarveMetadata{ID: 2, Name: "recent-gone", CreatedAt: now.Add(-1 * time.Hour), BlockCount: 1, MaxBlock: 0}
	stub := &stubCarveMetadataStore{carves: []*fleet.CarveMetadata{old, recent}}

	store, err := NewCarveStore(config.S3Config{
		CarvesBucket:           bucket,
		CarvesPrefix:           prefix,
		CarvesRegion:           "localhost",
		CarvesEndpointURL:      testEndpoint,
		CarvesAccessKeyID:      testAccessKeyID,
		CarvesSecretAccessKey:  testSecretAccessKey,
		CarvesForceS3PathStyle: true,
		CarvesDisableSSL:       true,
	}, stub)
	require.NoError(t, err)

	require.NoError(t, store.CreateTestBucket(ctx, bucket))
	t.Cleanup(func() {
		if err := store.CleanupTestBucket(context.Background()); err != nil {
			t.Errorf("cleanup s3 bucket %q: %v", bucket, err)
		}
	})

	cleaned, err := store.CleanupCarves(ctx, now)
	require.NoError(t, err)
	require.Equal(t, 1, cleaned, "only the >24h carve should be reconciled/expired")
	require.True(t, old.Expired, "carve older than 24h with no S3 object must be expired")
	require.False(t, recent.Expired, "carve younger than 24h must not be reconciled or expired")
}

// TestCleanupCarvesDisabled verifies that when the S3 carve store is configured
// with cleanup disabled, CleanupCarves is a no-op: it neither queries the metadata
// store nor expires any carve. This must apply to the S3 store only (the MySQL
// carve store is unaffected because it does not carry this flag).
func TestCleanupCarvesDisabled(t *testing.T) {
	// An old carve with no S3 object would normally be expired; with cleanup
	// disabled it must be left alone. No S3 backend is needed because the store
	// returns before doing any work.
	old := &fleet.CarveMetadata{ID: 1, Name: "old-gone", CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), BlockCount: 1, MaxBlock: 0}
	stub := &stubCarveMetadataStore{carves: []*fleet.CarveMetadata{old}}
	store := &CarveStore{metadatadb: stub, cleanupDisabled: true}

	cleaned, err := store.CleanupCarves(t.Context(), time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, cleaned, "cleanup must be a no-op when disabled")
	require.False(t, stub.listCarvesCalled, "cleanup must not query the metadata store when disabled")
	require.False(t, old.Expired, "no carve may be expired when cleanup is disabled")
}

// fakeHeadObjectAPI lets CleanupCarves be unit-tested without a live S3 backend.
type fakeHeadObjectAPI struct {
	fn func(key string) (*awss3.HeadObjectOutput, error)
}

func (f fakeHeadObjectAPI) HeadObject(_ context.Context, in *awss3.HeadObjectInput, _ ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
	return f.fn(*in.Key)
}

// oldCompletedCarve returns a carve old enough to pass the 24h floor with a
// completed upload, so cleanup will probe it.
func oldCompletedCarve(id int64, name string) *fleet.CarveMetadata {
	return &fleet.CarveMetadata{
		ID:         id,
		Name:       name,
		CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		BlockCount: 1,
		MaxBlock:   0,
	}
}

// TestCleanupCarvesDoesNotExpireOnTransientProbeError verifies the critical safety
// property: a probe error that is NOT a definitive not-found (e.g. throttling, a
// 5xx, or a network failure) must never expire a carve, since its object may well
// still exist.
func TestCleanupCarvesDoesNotExpireOnTransientProbeError(t *testing.T) {
	carve := oldCompletedCarve(1, "maybe-gone")
	stub := &stubCarveMetadataStore{carves: []*fleet.CarveMetadata{carve}}
	store := &CarveStore{
		s3store:    &s3store{prefix: "carvetest/", bucket: "test-bucket"},
		metadatadb: stub,
		headObjectAPI: fakeHeadObjectAPI{fn: func(string) (*awss3.HeadObjectOutput, error) {
			return nil, errors.New("throttled: SlowDown")
		}},
	}

	cleaned, err := store.CleanupCarves(t.Context(), time.Now())
	require.Error(t, err, "a non-not-found probe error must surface")
	require.Equal(t, 0, cleaned)
	require.False(t, carve.Expired, "a carve must not be expired on a transient probe error")
	require.Empty(t, stub.expiredIDs, "no ids should be batched for expiry")
}

// TestCleanupCarvesPartialFailureExpiresOnlyConfirmedAbsent verifies that within a
// single run, a confirmed-absent carve is still expired even when another carve's
// probe fails transiently, and the run surfaces the error.
func TestCleanupCarvesPartialFailureExpiresOnlyConfirmedAbsent(t *testing.T) {
	present := oldCompletedCarve(1, "present")
	absent := oldCompletedCarve(2, "absent")
	flaky := oldCompletedCarve(3, "flaky")
	stub := &stubCarveMetadataStore{carves: []*fleet.CarveMetadata{present, absent, flaky}}
	store := &CarveStore{
		s3store:    &s3store{prefix: "carvetest/", bucket: "test-bucket"},
		metadatadb: stub,
		headObjectAPI: fakeHeadObjectAPI{fn: func(key string) (*awss3.HeadObjectOutput, error) {
			switch {
			case strings.Contains(key, "present"):
				return &awss3.HeadObjectOutput{}, nil
			case strings.Contains(key, "absent"):
				return nil, &types.NotFound{}
			default: // flaky
				return nil, errors.New("throttled")
			}
		}},
	}

	cleaned, err := store.CleanupCarves(t.Context(), time.Now())
	require.Error(t, err, "the transient failure must surface")
	require.Equal(t, 1, cleaned, "only the confirmed-absent carve is expired")
	require.False(t, present.Expired)
	require.True(t, absent.Expired)
	require.False(t, flaky.Expired, "a carve with a transient probe error must not be expired")
	require.Equal(t, []int64{absent.ID}, stub.expiredIDs, "only the absent carve is batched for expiry")
	require.False(t, stub.updateCarveCalled, "cleanup must not fall back to per-carve UpdateCarve")
}

// TestCleanupCarvesConcurrentProbesAllAbsent exercises the bounded-concurrency
// probe fan-out (more candidates than the concurrency limit) and asserts no lost
// updates when collecting results. Run with -race to catch data races.
func TestCleanupCarvesConcurrentProbesAllAbsent(t *testing.T) {
	const n = 100
	carves := make([]*fleet.CarveMetadata, n)
	for i := range carves {
		carves[i] = oldCompletedCarve(int64(i+1), fmt.Sprintf("gone-%03d", i))
	}
	stub := &stubCarveMetadataStore{carves: carves}
	store := &CarveStore{
		s3store:    &s3store{prefix: "carvetest/", bucket: "test-bucket"},
		metadatadb: stub,
		headObjectAPI: fakeHeadObjectAPI{fn: func(string) (*awss3.HeadObjectOutput, error) {
			return nil, &types.NotFound{}
		}},
	}

	cleaned, err := store.CleanupCarves(t.Context(), time.Now())
	require.NoError(t, err)
	require.Equal(t, n, cleaned)
	require.Len(t, stub.expiredIDs, n, "every absent carve must be batched with no lost updates under concurrency")
	for _, c := range carves {
		require.True(t, c.Expired)
	}
}

// TestCleanupCarvesRespectsConfiguredMaxPerRun verifies the per-run cap is taken
// from the store's configured value (surfaced as the ListCarves page size), and
// falls back to the default when unset.
func TestCleanupCarvesRespectsConfiguredMaxPerRun(t *testing.T) {
	stub := &stubCarveMetadataStore{carves: nil}
	store := &CarveStore{metadatadb: stub, maxPerRun: 7}
	_, err := store.CleanupCarves(t.Context(), time.Now())
	require.NoError(t, err)
	require.Equal(t, uint(7), stub.lastListOpts.PerPage, "configured max-per-run should set the ListCarves page size")

	stubDefault := &stubCarveMetadataStore{carves: nil}
	storeDefault := &CarveStore{metadatadb: stubDefault}
	_, err = storeDefault.CleanupCarves(t.Context(), time.Now())
	require.NoError(t, err)
	require.Equal(t, uint(defaultCarvesCleanupMaxPerRun), stubDefault.lastListOpts.PerPage, "unset max-per-run should fall back to the default")
}
