// Package azure implements fleet.SoftwareInstallerStore and friends backed by
// Azure Blob Storage. Azure Blob Storage has no S3-compatible API, so this is
// a separate implementation rather than another branch of the s3 package (see
// the s3 package's isGCS for how S3-compatible backends are handled instead).
package azure

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/cloudstorage"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type azureBlobStore struct {
	client    *azblob.Client
	container string
	// signedURL, when true, makes Sign() return a SAS-signed GET URL instead of
	// erroring out with fleet.ErrNotConfigured.
	signedURL bool
}

type fileNotFoundError struct{}

var _ fleet.NotFoundError = (*fileNotFoundError)(nil)

func (fileNotFoundError) Error() string {
	return "file not found"
}

func (fileNotFoundError) IsNotFound() bool {
	return true
}

// misconfiguredCodes are the Azure error codes that mean the store is
// misconfigured (credentials, permissions, network rules or container).
var misconfiguredCodes = map[bloberror.Code]struct{}{
	bloberror.AccountIsDisabled:                 {},
	bloberror.AuthenticationFailed:              {},
	bloberror.AuthorizationFailure:              {},
	bloberror.AuthorizationPermissionMismatch:   {},
	bloberror.AuthorizationProtocolMismatch:     {},
	bloberror.AuthorizationResourceTypeMismatch: {},
	bloberror.AuthorizationServiceMismatch:      {},
	bloberror.AuthorizationSourceIPMismatch:     {},
	bloberror.ContainerDisabled:                 {},
	bloberror.ContainerNotFound:                 {},
	bloberror.InsufficientAccountPermissions:    {},
	bloberror.InvalidAuthenticationInfo:         {},
}

// newAPIError returns err as a *cloudstorage.APIError if it is an Azure
// response error, otherwise it returns err unchanged. The APIError is flagged
// as misconfigured if the store's configuration caused the error.
func newAPIError(err error, container string) error {
	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) {
		return err
	}

	code := respErr.ErrorCode
	if code == "" {
		code = strconv.Itoa(respErr.StatusCode)
	}
	// HEAD requests (e.g. Exists) have no body, so fall back to the status text.
	msg := http.StatusText(respErr.StatusCode)
	if respErr.RawResponse != nil {
		if body, err := runtime.Payload(respErr.RawResponse); err == nil {
			var xmlErr struct {
				Message string `xml:"Message"`
			}
			if xml.Unmarshal(body, &xmlErr) == nil && strings.TrimSpace(xmlErr.Message) != "" {
				msg = xmlErr.Message
			}
		}
	}

	_, knownCode := misconfiguredCodes[bloberror.Code(code)]
	misconfigured := knownCode || respErr.StatusCode == http.StatusUnauthorized || respErr.StatusCode == http.StatusForbidden
	return cloudstorage.NewAPIError(err, "Azure", code, msg, "container: "+container, misconfigured)
}

// newAzureBlobStore initializes an Azure Blob Storage-backed store using
// shared-key authentication.
func newAzureBlobStore(cfg config.AzureConfigInternal) (*azureBlobStore, error) {
	if cfg.AccountName == "" || cfg.AccountKey == "" {
		return nil, errors.New("azure storage account name and account key are required")
	}
	if cfg.Container == "" {
		return nil, errors.New("azure storage container is required")
	}

	cred, err := azblob.NewSharedKeyCredential(cfg.AccountName, cfg.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("creating azure shared key credential: %w", err)
	}

	// ContainerURL overrides the default endpoint; used for Azurite/testing.
	serviceURL := cfg.ContainerURL
	if serviceURL == "" {
		serviceURL = fmt.Sprintf("https://%s.blob.core.windows.net/", cfg.AccountName)
	}

	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating azure blob client: %w", err)
	}

	return &azureBlobStore{
		client:    client,
		container: cfg.Container,
		signedURL: cfg.SignedURL,
	}, nil
}

// CreateTestContainer creates the container associated with this store.
// Only recommended for local testing.
func (s *azureBlobStore) CreateTestContainer(ctx context.Context, name string) error {
	_, err := s.client.CreateContainer(ctx, name, nil)
	if bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
		return nil
	}
	return err
}

// CleanupTestContainer empties and deletes the container associated with
// this store. Only recommended for local testing. If the container no
// longer exists, it returns nil.
func (s *azureBlobStore) CleanupTestContainer(ctx context.Context) error {
	pager := s.client.NewListBlobsFlatPager(s.container, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if bloberror.HasCode(err, bloberror.ContainerNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		if page.Segment == nil {
			continue
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name == nil {
				continue
			}
			if _, err := s.client.DeleteBlob(ctx, s.container, *item.Name, nil); err != nil {
				return err
			}
		}
	}

	_, err := s.client.DeleteContainer(ctx, s.container, nil)
	if bloberror.HasCode(err, bloberror.ContainerNotFound) {
		return nil
	}
	return err
}
