package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

const (
	// defaultCarvesCleanupMaxPerRun bounds how many carves a single cleanup run
	// examines (and therefore the number of S3 HeadObject requests it makes) when
	// the s3.carves_cleanup_max_per_run config is unset. A larger backlog drains
	// across subsequent runs.
	defaultCarvesCleanupMaxPerRun = 1000
	// defaultCarvesCleanupConcurrency bounds how many HeadObject probes run at once
	// when the s3.carves_cleanup_concurrency config is unset. Kept modest to stay
	// well under S3's per-prefix request rate and avoid throttling.
	defaultCarvesCleanupConcurrency = 32
	// This is Golang's way of formatting timestrings, it's confusing, I know.
	// If you are used to more conventional timestrings, this is equivalent
	// to %Y/%m/%d/%H (year/month/day/hour)
	timePrefixFormat = "2006/01/02/15"
)

// s3HeadObjectAPI is the subset of the S3 client used to probe object existence.
// It is an interface so cleanup can be unit-tested without a live S3 backend.
type s3HeadObjectAPI interface {
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

// CarveStore is a type implementing the CarveStore interface
// relying on AWS S3 storage
type CarveStore struct {
	*s3store
	metadatadb fleet.CarveStore
	// headObjectAPI probes object existence during cleanup; defaults to the S3
	// client and is overridable in tests.
	headObjectAPI s3HeadObjectAPI
	// cleanupDisabled, when true, makes CleanupCarves a no-op so operators can
	// rely solely on the bucket lifecycle policy and skip S3 reconciliation.
	cleanupDisabled bool
	// maxPerRun and probeConcurrency tune CleanupCarves; when <= 0 the
	// defaultCarvesCleanup* constants are used.
	maxPerRun        int
	probeConcurrency int
}

// NewCarveStore creates a new store with the given config
func NewCarveStore(config config.S3Config, metadatadb fleet.CarveStore) (*CarveStore, error) {
	s3store, err := newS3Store(config.CarvesToInternalCfg())
	if err != nil {
		return nil, err
	}

	return &CarveStore{
		s3store:          s3store,
		metadatadb:       metadatadb,
		headObjectAPI:    s3store.s3Client,
		cleanupDisabled:  config.CarvesCleanupDisabled,
		maxPerRun:        config.CarvesCleanupMaxPerRun,
		probeConcurrency: config.CarvesCleanupConcurrency,
	}, nil
}

// generateS3Key builds S3 key from carve metadata
// all keys are prefixed by date so that they can easily be listed chronologically
func (c *CarveStore) generateS3Key(metadata *fleet.CarveMetadata) string {
	simpleDateHour := metadata.CreatedAt.Format(timePrefixFormat)
	return fmt.Sprintf("%s%s/%s", c.prefix, simpleDateHour, metadata.Name)
}

// NewCarve initializes a new file carving session
func (c *CarveStore) NewCarve(ctx context.Context, metadata *fleet.CarveMetadata) (*fleet.CarveMetadata, error) {
	objectKey := c.generateS3Key(metadata)

	var checksumAlgorithm types.ChecksumAlgorithm
	if c.gcs {
		checksumAlgorithm = types.ChecksumAlgorithmCrc32c // Required for GCS
	}

	res, err := c.s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: &c.bucket,
		Key:    &objectKey,

		ChecksumAlgorithm: checksumAlgorithm,
	})
	if err != nil {
		// even if we fail to create the multipart upload, we still want to create
		// the carve in the database and register an error, this way the user can
		// still fetch the carve and check its status
		metadata.Error = ptr.String(err.Error())
		if _, err := c.metadatadb.NewCarve(ctx, metadata); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "creating carve metadata")
		}
		return nil, ctxerr.Wrap(ctx, err, "creating multipart upload")
	}

	metadata.SessionId = *res.UploadId
	savedMetadata, err := c.metadatadb.NewCarve(ctx, metadata)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating carve metadata")
	}
	return savedMetadata, nil
}

// UpdateCarve updates carve definition in database
// Only max_block and expired are updatable
func (c *CarveStore) UpdateCarve(ctx context.Context, metadata *fleet.CarveMetadata) error {
	return c.metadatadb.UpdateCarve(ctx, metadata)
}

// ExpireCarves marks the given carves as expired via the metadata store.
func (c *CarveStore) ExpireCarves(ctx context.Context, ids []int64) error {
	return c.metadatadb.ExpireCarves(ctx, ids)
}

// carveObjectExists reports whether the carve's object is present in S3. A missing
// object (NoSuchKey/NotFound) is reported as (false, nil); any other error —
// including a missing bucket — is returned so the caller does not treat a
// transient or configuration failure as a deleted object.
func (c *CarveStore) carveObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := c.headObjectAPI.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	})
	if err != nil {
		// AWS S3 signals a missing object on HeadObject with NotFound; NoSuchKey is
		// kept as a defensive fallback for S3-compatible backends (e.g. GCS) that may
		// surface it instead. Any other error (including NoSuchBucket, throttling, or
		// a network failure) is returned so the caller does not treat a transient or
		// configuration failure as a deleted object.
		if _, ok := errors.AsType[*types.NotFound](err); ok {
			return false, nil
		}
		if _, ok := errors.AsType[*types.NoSuchKey](err); ok {
			return false, nil
		}
		return false, ctxerr.Wrapf(ctx, err, "checking existence of carve %s in S3", key)
	}
	return true, nil
}

// CleanupCarves marks carves whose S3 object no longer exists as expired.
// Deletion of the objects themselves is delegated to the bucket lifecycle policy;
// this only reconciles the DB `expired` flag against S3. Carves created within the
// last 24h are not reconciled, since S3 lifecycle expiration has day granularity
// and cannot have removed them yet. Reconciliation can be disabled entirely via
// the s3.carves_cleanup_disabled config.
//
// Each candidate's object is probed directly with HeadObject (rather than listing
// the bucket), which is exact and independent of listing order or object counts.
// Probes run with bounded concurrency (s3.carves_cleanup_concurrency) and are
// capped per run (s3.carves_cleanup_max_per_run); expirations are then written in
// a single batched statement.
func (c *CarveStore) CleanupCarves(ctx context.Context, now time.Time) (int, error) {
	if c.cleanupDisabled {
		return 0, nil
	}
	maxPerRun := c.maxPerRun
	if maxPerRun <= 0 {
		maxPerRun = defaultCarvesCleanupMaxPerRun
	}
	concurrency := c.probeConcurrency
	if concurrency <= 0 {
		concurrency = defaultCarvesCleanupConcurrency
	}
	// Oldest-first and capped so a backlog drains deterministically across runs
	// without any single run making an unbounded number of S3 requests.
	nonExpiredCarves, err := c.ListCarves(ctx, fleet.CarveListOptions{
		ListOptions: fleet.ListOptions{
			PerPage:        uint(maxPerRun), //nolint:gosec // bounded small positive config value
			OrderKey:       "created_at",
			OrderDirection: fleet.OrderAscending,
		},
		Expired: false,
	})
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "s3 carve cleanup")
	}

	cutoff := now.Add(-24 * time.Hour)
	var candidates []*fleet.CarveMetadata
	for _, carve := range nonExpiredCarves {
		// Skip carves too new to have been lifecycle-deleted, and carves whose
		// multipart upload hasn't completed: their object isn't in S3 yet, so
		// expiring one would make it permanently undownloadable.
		if carve.CreatedAt.Before(cutoff) && carve.BlocksComplete() {
			candidates = append(candidates, carve)
		}
	}
	if len(candidates) == 0 {
		return 0, nil
	}

	// Probe each candidate's object with HeadObject, with bounded concurrency since
	// each is a small, latency-bound request. Only S3 is touched here; the DB write
	// below is a single batched statement, so there is no concurrent DB access.
	var (
		mu       sync.Mutex
		toExpire []int64
		probeErr error
	)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for _, carve := range candidates {
		wg.Add(1)
		sem <- struct{}{}
		go func(carve *fleet.CarveMetadata) {
			defer wg.Done()
			defer func() { <-sem }()
			exists, err := c.carveObjectExists(ctx, c.generateS3Key(carve))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				// Only expire on a definitive not-found; treat anything else (e.g. a
				// throttled or failed request) as transient and retry on a later run.
				probeErr = errors.Join(probeErr, err)
			case !exists:
				carve.Expired = true
				toExpire = append(toExpire, carve.ID)
			}
		}(carve)
	}
	wg.Wait()

	if len(toExpire) == 0 {
		return 0, probeErr
	}
	if err := c.ExpireCarves(ctx, toExpire); err != nil {
		return 0, errors.Join(probeErr, ctxerr.Wrap(ctx, err, "s3 carve cleanup"))
	}
	return len(toExpire), probeErr
}

// Carve returns carve metadata by ID
func (c *CarveStore) Carve(ctx context.Context, carveID int64) (*fleet.CarveMetadata, error) {
	return c.metadatadb.Carve(ctx, carveID)
}

// CarveBySessionId returns carve metadata by session ID
func (c *CarveStore) CarveBySessionId(ctx context.Context, sessionID string) (*fleet.CarveMetadata, error) {
	return c.metadatadb.CarveBySessionId(ctx, sessionID)
}

// CarveByName returns carve metadata by name
func (c *CarveStore) CarveByName(ctx context.Context, name string) (*fleet.CarveMetadata, error) {
	return c.metadatadb.CarveByName(ctx, name)
}

// ListCarves returns a list of the currently available carves
func (c *CarveStore) ListCarves(ctx context.Context, opt fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
	return c.metadatadb.ListCarves(ctx, opt)
}

// listCompletedParts returns a list of the parts in a multipart updaload given a key and uploadID
// results are wrapped into the s3.CompletedPart struct
func (c *CarveStore) listCompletedParts(ctx context.Context, objectKey, uploadID string) ([]types.CompletedPart, error) {
	var res []types.CompletedPart
	var partMarker int32

	for {
		partNumberMarker := fmt.Sprint(partMarker)
		parts, err := c.s3Client.ListParts(ctx, &s3.ListPartsInput{
			Bucket:           &c.bucket,
			Key:              &objectKey,
			UploadId:         &uploadID,
			PartNumberMarker: &partNumberMarker,
		})
		if err != nil {
			return nil, err
		}
		for _, p := range parts.Parts {
			res = append(res, types.CompletedPart{
				ETag:       p.ETag,
				PartNumber: p.PartNumber,
			})
		}
		if !*parts.IsTruncated {
			break
		}
		pm, err := strconv.ParseInt(*parts.NextPartNumberMarker, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse next part number marker: %q: %w", *parts.NextPartNumberMarker, err)
		}
		partMarker = int32(pm)
	}
	return res, nil
}

const maxPartNumber = 10_000

// NewBlock uploads a new block for a specific carve
func (c *CarveStore) NewBlock(ctx context.Context, metadata *fleet.CarveMetadata, blockID int64, data []byte) error {
	if blockID < 0 || blockID >= maxPartNumber {
		return ctxerr.Errorf(ctx, "invalid blockID (must be 0-9_999): %d", blockID)
	}

	objectKey := c.generateS3Key(metadata)
	partNumber := int32(blockID) + 1 // PartNumber is 1-indexed

	var checksumAlgorithm types.ChecksumAlgorithm
	if c.gcs {
		checksumAlgorithm = types.ChecksumAlgorithmCrc32c // Required for GCS
	}

	_, err := c.s3Client.UploadPart(ctx, &s3.UploadPartInput{
		Body:       bytes.NewReader(data),
		Bucket:     &c.bucket,
		Key:        &objectKey,
		PartNumber: &partNumber,
		UploadId:   &metadata.SessionId,

		ChecksumAlgorithm: checksumAlgorithm,
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "s3 multipart carve upload")
	}
	if metadata.MaxBlock < blockID {
		metadata.MaxBlock = blockID
		if err = c.UpdateCarve(ctx, metadata); err != nil {
			return ctxerr.Wrap(ctx, err, "s3 multipart carve upload")
		}
	}
	if blockID >= metadata.BlockCount-1 {
		// The last block was reached, multipart upload can be completed
		parts, err := c.listCompletedParts(ctx, objectKey, metadata.SessionId)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "s3 multipart carve upload")
		}
		_, err = c.s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
			Bucket:   &c.bucket,
			Key:      &objectKey,
			UploadId: &metadata.SessionId,
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: parts,
			},
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "s3 complete multipart carve upload")
		}
	}
	return nil
}

// GetBlock returns a block of data for a carve
func (c *CarveStore) GetBlock(ctx context.Context, metadata *fleet.CarveMetadata, blockID int64) ([]byte, error) {
	objectKey := c.generateS3Key(metadata)
	// blockID is 0-indexed and sequential so can be perfectly used for evaluating ranges
	// range extremes are inclusive as for RFC-2616 (section 14.35)
	// no need to cap the rangeEnd to the carve size as S3 will do that by itself
	rangeStart := blockID * metadata.BlockSize
	rangeString := fmt.Sprintf("bytes=%d-%d", rangeStart, rangeStart+metadata.BlockSize-1)
	res, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &objectKey,
		Range:  &rangeString,
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			// The carve does not exists in S3, mark expired
			metadata.Expired = true
			if updateErr := c.UpdateCarve(ctx, metadata); updateErr != nil {
				err = ctxerr.Wrap(ctx, err, updateErr.Error())
			}
		}
		return nil, ctxerr.Wrap(ctx, err, "s3 carve get block")
	}
	defer res.Body.Close()
	carveData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "s3 carve get block")
	}
	return carveData, nil
}
