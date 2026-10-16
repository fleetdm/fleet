package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/installersize"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

const stagedUploadUnavailableMsg = "Direct upload isn't available on this server."

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
	if svc.stagedUploadStore == nil {
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
			return nil, &fleet.BadRequestError{Message: "Upload not found or expired. Please upload the file again."}
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

// finishStagedUpload closes the temp file and, if *err is nil, deletes the
// staged upload.
func (svc *Service) finishStagedUpload(ctx context.Context, uploadID string, tfr *fleet.TempFileReader, err *error) {
	tfr.Close()
	if *err == nil {
		svc.deleteStagedUpload(ctx, uploadID)
	}
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
