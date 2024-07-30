package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

const (
	defaultMaxS3Keys = 1000
	cleanupSize      = 1000
	// This is Golang's way of formatting timestrings, it's confusing, I know.
	// If you are used to more conventional timestrings, this is equivalent
	// to %Y/%m/%d/%H (year/month/day/hour)
	timePrefixFormat = "2006/01/02/15"
)

// CarveStore is a type implementing the CarveStore interface
// relying on AWS S3 storage
type CarveStore struct {
	*s3store
	metadatadb fleet.CarveStore
}

// NewCarveStore creates a new store with the given config
func NewCarveStore(config config.S3Config, metadatadb fleet.CarveStore) (*CarveStore, error) {
	s3store, err := newS3store(config.CarvesToInternalCfg())
	if err != nil {
		return nil, err
	}

	return &CarveStore{s3store, metadatadb}, nil
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
	res, err := c.s3client.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket: &c.bucket,
		Key:    &objectKey,
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

// listS3Carves lists all keys up to a given one or if the passed max number
// of keys has been reached; keys are returned in a set-like map
func (c *CarveStore) listS3Carves(lastPrefix string, maxKeys int) (map[string]bool, error) {
	var err error
	var continuationToken string
	result := make(map[string]bool)
	if maxKeys <= 0 {
		maxKeys = defaultMaxS3Keys
	}
	if !strings.HasPrefix(lastPrefix, c.prefix) {
		lastPrefix = c.prefix + lastPrefix
	}
	for {
		carveFilesPage, err := c.s3client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            &c.bucket,
			Prefix:            &c.prefix,
			ContinuationToken: &continuationToken,
		})
		if err != nil {
			return nil, err
		}
		for _, carveObject := range carveFilesPage.Contents {
			result[*carveObject.Key] = true
			if strings.HasPrefix(*carveObject.Key, lastPrefix) || len(result) >= maxKeys {
				return result, nil
			}
		}
		if !*carveFilesPage.IsTruncated {
			break
		}
		continuationToken = *carveFilesPage.ContinuationToken
	}
	return result, err
}

// CleanupCarves is a noop on the S3 side since users should rely on the bucket
// lifecycle configurations provided by AWS. This will compare a portion of the
// metadata present in the database and mark as expired the carves no longer
// available in S3 (ignores the `now` argument)
func (c *CarveStore) CleanupCarves(ctx context.Context, now time.Time) (int, error) {
	var err error
	// Get the 1000 oldest carves
	nonExpiredCarves, err := c.ListCarves(ctx, fleet.CarveListOptions{
		ListOptions: fleet.ListOptions{PerPage: cleanupSize},
		Expired:     false,
	})
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "s3 carve cleanup")
	}
	// List carves in S3 up to a hour+1 prefix
	lastCarveNextHour := nonExpiredCarves[len(nonExpiredCarves)-1].CreatedAt.Add(time.Hour)
	lastCarvePrefix := c.prefix + lastCarveNextHour.Format(timePrefixFormat)
	carveKeys, err := c.listS3Carves(lastCarvePrefix, 2*cleanupSize)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "s3 carve cleanup")
	}
	// Compare carve metadata in DB with S3 listing and update expiration flag
	cleanCount := 0
	for _, carve := range nonExpiredCarves {
		if _, ok := carveKeys[c.generateS3Key(carve)]; !ok {
			carve.Expired = true
			err = c.UpdateCarve(ctx, carve)
			cleanCount++
		}
	}
	return cleanCount, err
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
func (c *CarveStore) listCompletedParts(objectKey, uploadID string) ([]*s3.CompletedPart, error) {
	var res []*s3.CompletedPart
	var partMarker int64
	for {
		parts, err := c.s3client.ListParts(&s3.ListPartsInput{
			Bucket:           &c.bucket,
			Key:              &objectKey,
			UploadId:         &uploadID,
			PartNumberMarker: &partMarker,
		})
		if err != nil {
			return res, err
		}
		for _, p := range parts.Parts {
			res = append(res, &s3.CompletedPart{
				ETag:       p.ETag,
				PartNumber: p.PartNumber,
			})
		}
		if !*parts.IsTruncated {
			break
		}
		partMarker = *parts.NextPartNumberMarker
	}
	return res, nil
}

// NewBlock uploads a new block for a specific carve
func (c *CarveStore) NewBlock(ctx context.Context, metadata *fleet.CarveMetadata, blockID int64, data []byte) error {
	objectKey := c.generateS3Key(metadata)
	partNumber := blockID + 1 // PartNumber is 1-indexed
	_, err := c.s3client.UploadPart(&s3.UploadPartInput{
		Body:       bytes.NewReader(data),
		Bucket:     &c.bucket,
		Key:        &objectKey,
		PartNumber: &partNumber,
		UploadId:   &metadata.SessionId,
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
		parts, err := c.listCompletedParts(objectKey, metadata.SessionId)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "s3 multipart carve upload")
		}
		_, err = c.s3client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
			Bucket:          &c.bucket,
			Key:             &objectKey,
			UploadId:        &metadata.SessionId,
			MultipartUpload: &s3.CompletedMultipartUpload{Parts: parts},
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "s3 multipart carve upload")
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
	res, err := c.s3client.GetObject(&s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &objectKey,
		Range:  &rangeString,
	})
	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) && awsErr.Code() == s3.ErrCodeNoSuchKey {
			// The carve does not exists in S3, mark expired
			metadata.Expired = true
			if updateErr := c.UpdateCarve(ctx, metadata); err != nil {
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
