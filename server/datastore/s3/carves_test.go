package s3

import (
	"context"
	"strings"
	"testing"
	"time"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// stubCarveMetadataStore implements fleet.CarveStore using only in-memory
// state. It is used by tests that do not require a real S3 backend.
type stubCarveMetadataStore struct {
	carves []*fleet.CarveMetadata
}

func (s *stubCarveMetadataStore) NewCarve(_ context.Context, metadata *fleet.CarveMetadata) (*fleet.CarveMetadata, error) {
	return metadata, nil
}
func (s *stubCarveMetadataStore) UpdateCarve(_ context.Context, _ *fleet.CarveMetadata) error {
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
func (s *stubCarveMetadataStore) ListCarves(_ context.Context, _ fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
	return s.carves, nil
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
