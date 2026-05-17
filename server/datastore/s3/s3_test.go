package s3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func TestNewS3StoreGCSIAMAuthRequiresGCSEndpoint(t *testing.T) {
	_, err := newS3Store(config.S3ConfigInternal{
		Bucket:      "bucket",
		Prefix:      "prefix",
		Region:      "us-east-1",
		EndpointURL: "https://s3.amazonaws.com",
		GCSIAMAuth:  true,
	})
	require.ErrorContains(t, err, "gcs iam auth requires endpoint_url to contain storage.googleapis.com")
	require.ErrorContains(t, err, `"https://s3.amazonaws.com"`)
}

func TestNewS3StoreGCSIAMAuthRequiresEndpointSet(t *testing.T) {
	_, err := newS3Store(config.S3ConfigInternal{
		Bucket:     "bucket",
		Prefix:     "prefix",
		Region:     "us-east-1",
		GCSIAMAuth: true,
	})
	require.ErrorContains(t, err, "gcs iam auth requires endpoint_url to be set")
}

func TestNewS3StoreGCSIAMAuthDisallowsStaticKeys(t *testing.T) {
	_, err := newS3Store(config.S3ConfigInternal{
		Bucket:          "bucket",
		Prefix:          "prefix",
		Region:          "us-east-1",
		EndpointURL:     "https://storage.googleapis.com",
		GCSIAMAuth:      true,
		AccessKeyID:     "id",
		SecretAccessKey: "secret",
	})
	require.ErrorContains(t, err, "gcs iam auth cannot be used with access key credentials")
}

func TestNewS3StoreGCSIAMAuthDisallowsSTSRole(t *testing.T) {
	_, err := newS3Store(config.S3ConfigInternal{
		Bucket:           "bucket",
		Prefix:           "prefix",
		Region:           "us-east-1",
		EndpointURL:      "https://storage.googleapis.com",
		GCSIAMAuth:       true,
		StsAssumeRoleArn: "arn:aws:iam::123456789012:role/test",
	})
	require.ErrorContains(t, err, "gcs iam auth cannot be used with sts assume role")
}

func TestNewS3StoreGCSIAMAuthCredentialLookupError(t *testing.T) {
	originalFindDefaultGoogleCredentials := findDefaultGoogleCredentials
	t.Cleanup(func() {
		findDefaultGoogleCredentials = originalFindDefaultGoogleCredentials
	})

	findDefaultGoogleCredentials = func(context.Context, ...string) (*google.Credentials, error) {
		return nil, errors.New("lookup failed")
	}

	_, err := newS3Store(config.S3ConfigInternal{
		Bucket:      "bucket",
		Prefix:      "prefix",
		Region:      "us-east-1",
		EndpointURL: "https://storage.googleapis.com",
		GCSIAMAuth:  true,
	})
	require.ErrorContains(t, err, "finding default google credentials")
	require.ErrorContains(t, err, "lookup failed")
}

func TestSoftwareInstallerStoreGCSIAMAuthUsesBearerToken(t *testing.T) {
	type requestInfo struct {
		AuthHeader string
		Method     string
		Path       string
	}

	reqCh := make(chan requestInfo, 1)
	testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCh <- requestInfo{
			AuthHeader: r.Header.Get("Authorization"),
			Method:     r.Method,
			Path:       r.URL.Path,
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer testSrv.Close()

	originalFindDefaultGoogleCredentials := findDefaultGoogleCredentials
	t.Cleanup(func() {
		findDefaultGoogleCredentials = originalFindDefaultGoogleCredentials
	})

	const token = "test-bearer-token"
	findDefaultGoogleCredentials = func(context.Context, ...string) (*google.Credentials, error) {
		return &google.Credentials{
			TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
		}, nil
	}

	store, err := NewSoftwareInstallerStore(config.S3Config{
		SoftwareInstallersBucket:           "bucket",
		SoftwareInstallersPrefix:           "prefix",
		SoftwareInstallersEndpointURL:      testSrv.URL + "/storage.googleapis.com",
		SoftwareInstallersForceS3PathStyle: true,
		SoftwareInstallersGCSIAMAuth:       true,
	})
	require.NoError(t, err)

	exists, err := store.Exists(context.Background(), "installer-id")
	require.NoError(t, err)
	require.True(t, exists)

	select {
	case req := <-reqCh:
		require.Equal(t, http.MethodHead, req.Method)
		require.Equal(t, "Bearer "+token, req.AuthHeader)
		require.NotContains(t, req.AuthHeader, "AWS4-HMAC-SHA256")
		require.Contains(t, req.Path, "/bucket/", "expected bucket in request path, got %s", req.Path)
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive request to test server")
	}
}

func TestCarveStoreGCSIAMAuthUsesBearerToken(t *testing.T) {
	type requestInfo struct {
		AuthHeader string
		Method     string
		Path       string
	}

	reqCh := make(chan requestInfo, 1)
	testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case reqCh <- requestInfo{
			AuthHeader: r.Header.Get("Authorization"),
			Method:     r.Method,
			Path:       r.URL.Path,
		}:
		default:
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>false</IsTruncated></ListBucketResult>`))
	}))
	defer testSrv.Close()

	originalFindDefaultGoogleCredentials := findDefaultGoogleCredentials
	t.Cleanup(func() {
		findDefaultGoogleCredentials = originalFindDefaultGoogleCredentials
	})

	const token = "carves-bearer-token"
	findDefaultGoogleCredentials = func(context.Context, ...string) (*google.Credentials, error) {
		return &google.Credentials{
			TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
		}, nil
	}

	store, err := NewCarveStore(config.S3Config{
		CarvesBucket:           "carves-bucket",
		CarvesPrefix:           "carves-prefix",
		CarvesEndpointURL:      testSrv.URL + "/storage.googleapis.com",
		CarvesForceS3PathStyle: true,
		CarvesGCSIAMAuth:       true,
	}, nil)
	require.NoError(t, err)

	_, err = store.listS3Carves(context.Background(), "", 10)
	require.NoError(t, err)

	select {
	case req := <-reqCh:
		require.Equal(t, "Bearer "+token, req.AuthHeader)
		require.NotContains(t, req.AuthHeader, "AWS4-HMAC-SHA256")
		require.Contains(t, req.Path, "/carves-bucket", "expected carves bucket in request path, got %s", req.Path)
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive request to test server")
	}
}
