package azure

import (
	"encoding/xml"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/fleetdm/fleet/v4/server/datastore/cloudstorage"
)

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
