package s3

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

const (
	defaultMaxS3Keys = 1000
	cleanupSize      = 1000
	// This is Golang's way of formatting timestrings, it's confusing, I know.
	// If you are used to more conventional timestrings, this is equivalent
	// to %Y/%m/%d/%H (year/month/day/hour)
	timePrefixFormat = "2006/01/02/15"
)

// generateS3Key builds S3 key from carve metadata
// all keys are prefixed by date so that they can easily be listed chronologically
func (d *Datastore) generateS3Key(metadata *fleet.CarveMetadata) string {
	simpleDateHour := metadata.CreatedAt.Format(timePrefixFormat)
	return fmt.Sprintf("%s%s/%s", d.prefix, simpleDateHour, metadata.Name)
}

// NewCarve initializes a new file carving session
func (d *Datastore) NewCarve(metadata *fleet.CarveMetadata) (*fleet.CarveMetadata, error) {
	objectKey := d.generateS3Key(metadata)
	res, err := d.s3client.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket: &d.bucket,
		Key:    &objectKey,
	})
	if err != nil {
		return nil, errors.Wrap(err, "s3 multipart carve create")
	}
	metadata.SessionId = *res.UploadId
	return d.metadatadb.NewCarve(metadata)
}

// UpdateCarve updates carve definition in database
// Only max_block and expired are updatable
func (d *Datastore) UpdateCarve(metadata *fleet.CarveMetadata) error {
	return d.metadatadb.UpdateCarve(metadata)
}

// listS3Carves lists all keys up to a given one or if the passed max number
// of keys has been reached; keys are returned in a set-like map
func (d *Datastore) listS3Carves(lastPrefix string, maxKeys int) (map[string]bool, error) {
	var err error
	var continuationToken string
	result := make(map[string]bool)
	if maxKeys <= 0 {
		maxKeys = defaultMaxS3Keys
	}
	if !strings.HasPrefix(lastPrefix, d.prefix) {
		lastPrefix = d.prefix + lastPrefix
	}
	for {
		carveFilesPage, err := d.s3client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            &d.bucket,
			Prefix:            &d.prefix,
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
func (d *Datastore) CleanupCarves(now time.Time) (int, error) {
	var err error
	// Get the 1000 oldest carves
	nonExpiredCarves, err := d.ListCarves(fleet.CarveListOptions{
		ListOptions: fleet.ListOptions{PerPage: cleanupSize},
		Expired:     false,
	})
	if err != nil {
		return 0, errors.Wrap(err, "s3 carve cleanup")
	}
	// List carves in S3 up to a hour+1 prefix
	lastCarveNextHour := nonExpiredCarves[len(nonExpiredCarves)-1].CreatedAt.Add(time.Hour)
	lastCarvePrefix := d.prefix + lastCarveNextHour.Format(timePrefixFormat)
	carveKeys, err := d.listS3Carves(lastCarvePrefix, 2*cleanupSize)
	if err != nil {
		return 0, errors.Wrap(err, "s3 carve cleanup")
	}
	// Compare carve metadata in DB with S3 listing and update expiration flag
	cleanCount := 0
	for _, carve := range nonExpiredCarves {
		if _, ok := carveKeys[d.generateS3Key(carve)]; !ok {
			carve.Expired = true
			err = d.UpdateCarve(carve)
			cleanCount++
		}
	}
	return cleanCount, err
}

// Carve returns carve metadata by ID
func (d *Datastore) Carve(carveID int64) (*fleet.CarveMetadata, error) {
	return d.metadatadb.Carve(carveID)
}

// CarveBySessionId returns carve metadata by session ID
func (d *Datastore) CarveBySessionId(sessionID string) (*fleet.CarveMetadata, error) {
	return d.metadatadb.CarveBySessionId(sessionID)
}

// CarveByName returns carve metadata by name
func (d *Datastore) CarveByName(name string) (*fleet.CarveMetadata, error) {
	return d.metadatadb.CarveByName(name)
}

// ListCarves returns a list of the currently available carves
func (d *Datastore) ListCarves(opt fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
	return d.metadatadb.ListCarves(opt)
}

// listCompletedParts returns a list of the parts in a multipart updaload given a key and uploadID
// results are wrapped into the s3.CompletedPart struct
func (d *Datastore) listCompletedParts(objectKey, uploadID string) ([]*s3.CompletedPart, error) {
	var res []*s3.CompletedPart
	var partMarker int64
	for {
		parts, err := d.s3client.ListParts(&s3.ListPartsInput{
			Bucket:           &d.bucket,
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
func (d *Datastore) NewBlock(metadata *fleet.CarveMetadata, blockID int64, data []byte) error {
	objectKey := d.generateS3Key(metadata)
	partNumber := blockID + 1 // PartNumber is 1-indexed
	_, err := d.s3client.UploadPart(&s3.UploadPartInput{
		Body:       bytes.NewReader(data),
		Bucket:     &d.bucket,
		Key:        &objectKey,
		PartNumber: &partNumber,
		UploadId:   &metadata.SessionId,
	})
	if err != nil {
		return errors.Wrap(err, "s3 multipart carve upload")
	}
	if metadata.MaxBlock < blockID {
		metadata.MaxBlock = blockID
		if err = d.UpdateCarve(metadata); err != nil {
			return errors.Wrap(err, "s3 multipart carve upload")
		}
	}
	if blockID >= metadata.BlockCount-1 {
		// The last block was reached, multipart upload can be completed
		parts, err := d.listCompletedParts(objectKey, metadata.SessionId)
		if err != nil {
			return errors.Wrap(err, "s3 multipart carve upload")
		}
		_, err = d.s3client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
			Bucket:          &d.bucket,
			Key:             &objectKey,
			UploadId:        &metadata.SessionId,
			MultipartUpload: &s3.CompletedMultipartUpload{Parts: parts},
		})
		if err != nil {
			return errors.Wrap(err, "s3 multipart carve upload")
		}
	}
	return nil
}

// GetBlock returns a block of data for a carve
func (d *Datastore) GetBlock(metadata *fleet.CarveMetadata, blockID int64) ([]byte, error) {
	objectKey := d.generateS3Key(metadata)
	// blockID is 0-indexed and sequential so can be perfectly used for evaluating ranges
	// range extremes are inclusive as for RFC-2616 (section 14.35)
	// no need to cap the rangeEnd to the carve size as S3 will do that by itself
	rangeStart := blockID * metadata.BlockSize
	rangeString := fmt.Sprintf("bytes=%d-%d", rangeStart, rangeStart+metadata.BlockSize-1)
	res, err := d.s3client.GetObject(&s3.GetObjectInput{
		Bucket: &d.bucket,
		Key:    &objectKey,
		Range:  &rangeString,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == s3.ErrCodeNoSuchKey {
			// The carve does not exists in S3, mark expired
			metadata.Expired = true
			if updateErr := d.UpdateCarve(metadata); err != nil {
				err = errors.Wrap(err, updateErr.Error())
			}
		}
		return nil, errors.Wrap(err, "s3 carve get block")
	}
	defer res.Body.Close()
	carveData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "s3 carve get block")
	}
	return carveData, nil
}
