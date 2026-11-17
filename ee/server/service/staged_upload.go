package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/installersize"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

const (
	stagedUploadUnavailableMsg = "Direct upload isn't available on this server."
	stagedUploadNotFoundMsg    = "Upload not found or expired. Please upload the file again."
)

func (svc *Service) CreateStagedUpload(ctx context.Context, target fleet.StagedUploadTarget, teamID uint, size int64) (*fleet.StagedUpload, error) {
	var err error
	switch target {
	case fleet.StagedUploadTargetBootstrapPackage:
		err = svc.authz.Authorize(ctx, &fleet.MDMAppleBootstrapPackage{TeamID: teamID}, fleet.ActionWrite)
	case fleet.StagedUploadTargetSoftwarePackage:
		err = svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: &teamID}, fleet.ActionWrite)
	default:
		svc.authz.SkipAuthorization(ctx)
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("Unsupported upload target %q.", target)}
	}
	if err != nil {
		return nil, err
	}

	if svc.stagedUploadStore == nil {
		return nil, &fleet.BadRequestError{Message: stagedUploadUnavailableMsg}
	}
	// The installer size middleware only runs on upload routes, so read the config.
	if size <= 0 {
		return nil, &fleet.BadRequestError{Message: "size is required and must be greater than 0."}
	}
	if maxSize := svc.config.Server.MaxInstallerSizeBytes; size > maxSize {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("The maximum file size is %s.", installersize.Human(maxSize))}
	}

	uploadID := uuid.NewString()
	url, err := svc.stagedUploadStore.PresignPut(ctx, uploadID, size, fleet.StagedUploadURLExpiry)
	if err != nil {
		if errors.Is(err, fleet.ErrNotConfigured) {
			return nil, &fleet.BadRequestError{Message: stagedUploadUnavailableMsg}
		}
		return nil, ctxerr.Wrap(ctx, err, "presign staged upload")
	}
	return &fleet.StagedUpload{UploadID: uploadID, URL: url, ExpiresAt: time.Now().Add(fleet.StagedUploadURLExpiry)}, nil
}

// openStagedUpload copies a staged upload into a temp file the caller must close.
func (svc *Service) openStagedUpload(ctx context.Context, uploadID string) (*fleet.TempFileReader, error) {
	if svc.stagedUploadStore == nil || !svc.config.StagedUploadAvailable() {
		return nil, &fleet.BadRequestError{Message: stagedUploadUnavailableMsg}
	}
	// The id becomes part of the object key, so only accept the ids Fleet hands out.
	parsed, err := uuid.Parse(uploadID)
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "Invalid upload_id."}
	}
	rc, _, err := svc.stagedUploadStore.Get(ctx, parsed.String())
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, &fleet.BadRequestError{Message: stagedUploadNotFoundMsg}
		}
		return nil, ctxerr.Wrap(ctx, err, "get staged upload")
	}
	defer rc.Close()
	tfr, err := fleet.NewTempFileReader(rc, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "copy staged upload to temp file")
	}
	return tfr, nil
}

// openStagedSoftwareUpload opens a staged upload of a software package. When
// streaming installer downloads is enabled, it is read with ranged requests
// (and later promoted to its installer key with a server-side copy) instead of
// being copied to a temp file. The caller must close the returned reader.
func (svc *Service) openStagedSoftwareUpload(ctx context.Context, uploadID string) (fleet.InstallerReadCloser, error) {
	store := svc.streamingInstallerStore()
	if store == nil {
		tfr, err := svc.openStagedUpload(ctx, uploadID)
		if err != nil {
			return nil, err
		}
		return tfr, nil
	}

	if svc.stagedUploadStore == nil || !svc.config.StagedUploadAvailable() {
		return nil, &fleet.BadRequestError{Message: stagedUploadUnavailableMsg}
	}
	parsed, err := uuid.Parse(uploadID)
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "Invalid upload_id."}
	}
	uploadID = parsed.String()

	// The client's presigned PUT URL stays valid after the upload, so different
	// bytes could be swapped in between hashing and promoting them. Either pin
	// the upload to its ETag, or read and promote a copy only Fleet can write.
	if pinStore, ok := store.(fleet.ETagStagingStore); ok {
		size, etag, err := pinStore.StagingETag(ctx, uploadID)
		if err != nil {
			if fleet.IsNotFound(err) {
				return nil, &fleet.BadRequestError{Message: stagedUploadNotFoundMsg}
			}
			return nil, ctxerr.Wrap(ctx, err, "get staged upload attributes")
		}
		if etag != "" {
			// The client's upload is only deleted by finishStagedUpload on
			// success, so the request can be retried.
			rir, err := openHashedStagingObject(ctx, store, uploadID, size, etag)
			if err != nil {
				return nil, stagedUploadChangedError(err)
			}
			return rir, nil
		}
	}

	stagingID := uuid.NewString()
	if err := store.CopyStaging(ctx, uploadID, stagingID); err != nil {
		if fleet.IsNotFound(err) {
			return nil, &fleet.BadRequestError{Message: stagedUploadNotFoundMsg}
		}
		return nil, ctxerr.Wrap(ctx, err, "copy staged upload")
	}
	deleteCopy := func() {
		if err := store.DeleteStaging(context.WithoutCancel(ctx), stagingID); err != nil {
			svc.logger.WarnContext(ctx, "deleting copy of staged upload", "upload_id", uploadID, "err", err)
		}
	}

	size, err := store.StagingSize(ctx, stagingID)
	if err != nil {
		deleteCopy()
		return nil, ctxerr.Wrap(ctx, err, "get staged upload size")
	}
	rir, err := openHashedStagingObject(ctx, store, stagingID, size, "")
	if err != nil {
		deleteCopy()
		return nil, err
	}
	// The copy is Fleet's own, delete it when done (a no-op once promoted). The
	// client's upload is only deleted by finishStagedUpload on success, so the
	// request can be retried.
	rir.SetDeleteOnClose(true)
	return rir, nil
}

// openHashedStagingObject reads the staging object once to compute its SHA256
// (validating it inline on the way), so the parsers don't each read it all,
// and returns a reader over it that knows its hash. A non-empty etag pins the
// object to that ETag.
func openHashedStagingObject(ctx context.Context, store fleet.StreamingInstallerStore, stagingID string, size int64, etag string) (*fleet.RemoteInstallerReader, error) {
	open := func(sha256Hex string) (*fleet.RemoteInstallerReader, error) {
		if etag != "" {
			return fleet.NewPinnedRemoteInstallerReader(ctx, store, stagingID, size, etag, sha256Hex)
		}
		return fleet.NewRemoteInstallerReader(ctx, store, stagingID, size, sha256Hex)
	}

	hashReader, err := open("")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "open staged upload")
	}
	h := sha256.New()
	validator := file.NewTarGzStreamValidator()
	_, err = io.Copy(io.MultiWriter(h, validator), hashReader)
	format := validator.Finish()
	_ = hashReader.Close()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "hash staged upload")
	}

	rir, err := open(hex.EncodeToString(h.Sum(nil)))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "open staged upload")
	}
	rir.SetPrevalidatedFormat(format)
	return rir, nil
}

// stagedUploadChangedError turns the error of a staged upload overwritten
// while Fleet was processing it into a client error.
func stagedUploadChangedError(err error) error {
	if errors.Is(err, fleet.ErrStagingChanged) {
		return &fleet.BadRequestError{
			Message:     "The upload changed while it was being processed. Please upload the file again.",
			InternalErr: err,
		}
	}
	return err
}

// finishStagedUpload closes the reader and, if *err is nil, deletes the
// staged upload. An error caused by the staged upload changing while it was
// processed is turned into a client error.
func (svc *Service) finishStagedUpload(ctx context.Context, uploadID string, tfr io.Closer, err *error) {
	tfr.Close()
	if *err == nil {
		svc.deleteStagedUpload(ctx, uploadID)
		return
	}
	*err = stagedUploadChangedError(*err)
}

// deleteStagedUpload removes a staged upload once its package is stored. The
// cleanup cron is the backstop, so a failure is only logged.
func (svc *Service) deleteStagedUpload(ctx context.Context, uploadID string) {
	parsed, err := uuid.Parse(uploadID)
	if err != nil {
		return
	}
	if err := svc.stagedUploadStore.Delete(ctx, parsed.String()); err != nil {
		svc.logger.WarnContext(ctx, "deleting staged upload", "upload_id", uploadID, "err", err)
	}
}
