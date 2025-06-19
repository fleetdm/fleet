//go:build linux

package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/tee"
	"github.com/remitly-oss/httpsig-go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tpmSigner wraps a TEE key and implements crypto.Signer for HTTP signatures
type tpmSigner struct {
	teeKey tee.Key
	keyID  string
}

// newTPMSigner creates a new TPM-based signer
func newTPMSigner(teeKey tee.Key, keyID string) *tpmSigner {
	return &tpmSigner{
		teeKey: teeKey,
		keyID:  keyID,
	}
}

// Public returns the public key from the TPM
func (s *tpmSigner) Public() crypto.PublicKey {
	pubKey, err := s.teeKey.Public()
	if err != nil {
		// In a real implementation, you might want to handle this differently
		panic(fmt.Sprintf("failed to get public key from TPM: %v", err))
	}
	return pubKey
}

// Sign signs the digest using the TPM key
func (s *tpmSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	teeSigner, err := s.teeKey.Signer()
	if err != nil {
		return nil, fmt.Errorf("failed to get TPM signer: %w", err)
	}

	return teeSigner.Sign(rand, digest, opts)
}

// KeyID returns the key identifier
func (s *tpmSigner) KeyID() string {
	return s.keyID
}

// tpmKeyStore holds TPM public keys in memory for HTTP signature verification
type tpmKeyStore struct {
	keys map[string]*httpsig.KeySpec // keyID -> KeySpec
}

func newTPMKeyStore() *tpmKeyStore {
	return &tpmKeyStore{
		keys: make(map[string]*httpsig.KeySpec),
	}
}

func (ks *tpmKeyStore) addKey(keyID string, keySpec *httpsig.KeySpec) {
	ks.keys[keyID] = keySpec
}

func (ks *tpmKeyStore) FetchByKeyID(ctx context.Context, _ http.Header, keyID string) (httpsig.KeySpecer, error) {
	keySpec, exists := ks.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found for keyID: %s", keyID)
	}
	return keySpec, nil
}

func (ks *tpmKeyStore) Fetch(_ context.Context, _ http.Header, _ httpsig.MetadataProvider) (httpsig.KeySpecer, error) {
	return nil, fmt.Errorf("not implemented")
}

// publicKeyToPEM converts a public key to PEM format for debugging
func publicKeyToPEM(pubKey crypto.PublicKey) (string, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	return string(pem.EncodeToMemory(pemBlock)), nil
}

// logHTTPSignatureHeaders logs the Signature and Signature-Input headers for debugging
func logHTTPSignatureHeaders(t *testing.T, req *http.Request, requestType string) {
	signature := req.Header.Get("Signature")
	signatureInput := req.Header.Get("Signature-Input")

	t.Logf("🔍 %s Request HTTP Signature Headers:", requestType)
	t.Logf("📝 Signature: %s", signature)
	t.Logf("📋 Signature-Input: %s", signatureInput)

	// Also log other relevant headers
	contentDigest := req.Header.Get("Content-Digest")
	if contentDigest != "" {
		t.Logf("🔐 Content-Digest: %s", contentDigest)
	}
}

// TestTPMHTTPSignatureVerification tests HTTP signature creation and verification
// using TPM-generated ECC keys through the TEE interface
func TestTPMHTTPSignatureVerification(t *testing.T) {
	// Skip test if TPM is not available
	if !isTPMAvailable() {
		t.Skip("TPM device not available, skipping TPM HTTP signature test")
	}

	// Create a temporary directory for TPM key blobs
	tempDir := t.TempDir()

	// Create logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Initialize TPM 2.0 device
	teeDevice, err := tee.NewTPM2(
		tee.WithLogger(logger),
		tee.WithPublicBlobPath(filepath.Join(tempDir, "tpm_public.blob")),
		tee.WithPrivateBlobPath(filepath.Join(tempDir, "tpm_private.blob")),
	)
	require.NoError(t, err, "Failed to initialize TPM")
	defer teeDevice.Close()

	// Create ECC key in TPM (automatically selects best curve: P-384 or P-256)
	ctx := context.Background()
	teeKey, err := teeDevice.CreateKey(ctx)
	require.NoError(t, err, "Failed to create TPM key")
	defer teeKey.Close()

	// Get public key from TPM
	publicKey, err := teeKey.Public()
	require.NoError(t, err, "Failed to get public key from TPM")

	// Print public key in PEM format for debugging
	pubKeyPEM, err := publicKeyToPEM(publicKey)
	require.NoError(t, err, "Failed to convert public key to PEM")
	t.Logf("🔑 TPM Public Key (PEM format):\n%s", pubKeyPEM)

	// Determine the algorithm based on the key type
	var algo httpsig.Algorithm
	switch pub := publicKey.(type) {
	case *ecdsa.PublicKey:
		switch pub.Curve.Params().BitSize {
		case 256:
			algo = httpsig.Algo_ECDSA_P256_SHA256
		case 384:
			algo = httpsig.Algo_ECDSA_P384_SHA384
		default:
			t.Fatalf("Unsupported ECC curve bit size: %d", pub.Curve.Params().BitSize)
		}
	default:
		t.Fatalf("Unsupported public key type: %T", publicKey)
	}

	// Create a key ID for the signature
	keyID := "tpm-key-001"

	// Log key details for debugging
	if ecdsaKey, ok := publicKey.(*ecdsa.PublicKey); ok {
		t.Logf("🔐 TPM Key Details:")
		t.Logf("   Algorithm: %s", algo)
		t.Logf("   Curve: %s", ecdsaKey.Curve.Params().Name)
		t.Logf("   Bit Size: %d", ecdsaKey.Curve.Params().BitSize)
		t.Logf("   Key ID: %s", keyID)
	}

	// Create TPM signer that wraps the TEE key
	tpmSigner := newTPMSigner(teeKey, keyID)

	// Set up in-memory key store with the TPM public key
	keyStore := newTPMKeyStore()
	keyStore.addKey(keyID, &httpsig.KeySpec{
		KeyID:  keyID,
		Algo:   algo,
		PubKey: tpmSigner.Public(),
	})

	// Set up HTTP signature verifier (server-side)
	verifier, err := httpsig.NewVerifier(keyStore, httpsig.VerifyProfile{
		SignatureLabel:     httpsig.DefaultSignatureLabel,
		AllowedAlgorithms:  []httpsig.Algorithm{httpsig.Algo_ECDSA_P256_SHA256, httpsig.Algo_ECDSA_P384_SHA384},
		RequiredFields:     httpsig.Fields("@method", "@target-uri", "content-digest"),
		RequiredMetadata:   []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID, httpsig.MetaNonce},
		DisallowedMetadata: []httpsig.Metadata{httpsig.MetaAlgorithm},
	})
	require.NoError(t, err, "Failed to create verifier")

	// Create test server with HTTP signature verification
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log signature headers for debugging
		logHTTPSignatureHeaders(t, r, r.Method)

		// Verify the HTTP signature
		_, err := verifier.Verify(r)
		if err != nil {
			t.Logf("❌ Signature verification failed: %v", err)
			http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
			return
		}

		t.Logf("✅ Signature verification succeeded for %s %s", r.Method, r.URL.Path)

		// If verification succeeds, return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("TPM HTTP signature verified successfully!"))
	}))
	defer testServer.Close()

	// Set up HTTP signature signer (client-side) using TPM
	signer, err := httpsig.NewSigner(httpsig.SigningProfile{
		Algorithm: algo,
		Fields:    httpsig.Fields("@method", "@target-uri", "content-digest"),
		Metadata:  []httpsig.Metadata{httpsig.MetaKeyID, httpsig.MetaCreated, httpsig.MetaNonce},
	}, httpsig.SigningKey{
		Key:       tpmSigner, // Use the TPM signer that implements crypto.Signer
		MetaKeyID: keyID,
	})
	require.NoError(t, err, "Failed to create TPM-based signer")

	// Create HTTP client with signature support
	baseClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	signedClient := httpsig.NewHTTPClient(baseClient, signer, nil)

	// Test 1: Make a GET request to the test server
	resp, err := signedClient.Get(testServer.URL + "/test-endpoint")
	require.NoError(t, err, "Failed to make signed GET request with TPM")
	defer resp.Body.Close()

	// Verify the GET response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected successful GET response with TPM signature")

	// Read response body
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	responseText := string(body[:n])

	assert.Contains(t, responseText, "TPM HTTP signature verified successfully!", "Expected success message in GET response")

	t.Logf("✅ TPM GET request HTTP signature verification passed!")
	t.Logf("📝 GET Response: %s", responseText)

	// Test 2: Make a POST request with body content using TPM signature
	// Create JSON payload for POST request
	postData := map[string]interface{}{
		"message":   "Hello from TPM-signed POST request",
		"timestamp": time.Now().Unix(),
		"tpm_info":  "Hardware-based signature using TPM 2.0",
		"data":      []string{"tpm-item1", "tpm-item2", "tpm-item3"},
	}

	jsonData, err := json.Marshal(postData)
	require.NoError(t, err, "Failed to marshal JSON data")

	// Make POST request with JSON body signed by TPM
	postResp, err := signedClient.Post(testServer.URL+"/api/tpm-data", "application/json", bytes.NewReader(jsonData))
	require.NoError(t, err, "Failed to make signed POST request with TPM")
	defer postResp.Body.Close()

	// Verify the POST response
	assert.Equal(t, http.StatusOK, postResp.StatusCode, "Expected successful POST response with TPM signature")

	// Read POST response body
	postBody := make([]byte, 1024)
	postN, _ := postResp.Body.Read(postBody)
	postResponseText := string(postBody[:postN])

	assert.Contains(t, postResponseText, "TPM HTTP signature verified successfully!", "Expected success message in POST response")

	t.Logf("✅ TPM POST request HTTP signature verification passed!")
	t.Logf("📝 POST Response: %s", postResponseText)
	t.Logf("🔐 Used TPM-generated ECC key for signing both GET and POST")
	t.Logf("🛡️ TPM signatures verified successfully on server side")
	t.Logf("🔑 Key ID: %s", keyID)
	t.Logf("📊 POST payload size: %d bytes", len(jsonData))
	t.Logf("🔧 TPM signer wraps TEE key and implements crypto.Signer interface")
	t.Logf("🏭 Algorithm used: %s", algo)
}

// TestTPMHTTPSignatureVerificationFailure tests that invalid TPM signatures are rejected
func TestTPMHTTPSignatureVerificationFailure(t *testing.T) {
	// Skip test if TPM is not available
	if !isTPMAvailable() {
		t.Skip("TPM device not available, skipping TPM HTTP signature failure test")
	}

	// Create temporary directories for different TPM keys
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Initialize first TPM device for the "correct" key
	teeDevice1, err := tee.NewTPM2(
		tee.WithLogger(logger),
		tee.WithPublicBlobPath(filepath.Join(tempDir1, "tpm_public1.blob")),
		tee.WithPrivateBlobPath(filepath.Join(tempDir1, "tpm_private1.blob")),
	)
	require.NoError(t, err, "Failed to initialize first TPM")
	defer teeDevice1.Close()

	// Initialize second TPM device for the "wrong" key
	teeDevice2, err := tee.NewTPM2(
		tee.WithLogger(logger),
		tee.WithPublicBlobPath(filepath.Join(tempDir2, "tpm_public2.blob")),
		tee.WithPrivateBlobPath(filepath.Join(tempDir2, "tpm_private2.blob")),
	)
	require.NoError(t, err, "Failed to initialize second TPM")
	defer teeDevice2.Close()

	ctx := context.Background()

	// Create correct key
	correctKey, err := teeDevice1.CreateKey(ctx)
	require.NoError(t, err, "Failed to create correct TPM key")
	defer correctKey.Close()

	// Create wrong key
	wrongKey, err := teeDevice2.CreateKey(ctx)
	require.NoError(t, err, "Failed to create wrong TPM key")
	defer wrongKey.Close()

	keyID := "tpm-key-001"

	// Create signers
	correctSigner := newTPMSigner(correctKey, keyID)
	wrongSigner := newTPMSigner(wrongKey, keyID)

	// Determine algorithm from correct key
	correctPubKey, err := correctKey.Public()
	require.NoError(t, err, "Failed to get correct public key")

	var algo httpsig.Algorithm
	switch pub := correctPubKey.(type) {
	case *ecdsa.PublicKey:
		switch pub.Curve.Params().BitSize {
		case 256:
			algo = httpsig.Algo_ECDSA_P256_SHA256
		case 384:
			algo = httpsig.Algo_ECDSA_P384_SHA384
		}
	}

	// Set up key store with the correct public key
	keyStore := newInMemoryKeyStore()
	keyStore.addKey(keyID, &httpsig.KeySpec{
		KeyID:  keyID,
		Algo:   algo,
		PubKey: correctSigner.Public(),
	})

	// Set up verifier
	verifier, err := httpsig.NewVerifier(keyStore, httpsig.VerifyProfile{
		SignatureLabel:     httpsig.DefaultSignatureLabel,
		AllowedAlgorithms:  []httpsig.Algorithm{httpsig.Algo_ECDSA_P256_SHA256, httpsig.Algo_ECDSA_P384_SHA384},
		RequiredFields:     httpsig.Fields("@method", "@target-uri", "content-digest"),
		RequiredMetadata:   []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID, httpsig.MetaNonce},
		DisallowedMetadata: []httpsig.Metadata{httpsig.MetaAlgorithm},
	})
	require.NoError(t, err, "Failed to create verifier")

	// Create test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := verifier.Verify(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This should not be reached"))
	}))
	defer testServer.Close()

	// Create signer with wrong TPM key
	signer, err := httpsig.NewSigner(httpsig.SigningProfile{
		Algorithm: algo,
		Fields:    httpsig.Fields("@method", "@target-uri", "content-digest"),
		Metadata:  []httpsig.Metadata{httpsig.MetaKeyID, httpsig.MetaCreated, httpsig.MetaNonce},
	}, httpsig.SigningKey{
		Key:       wrongSigner, // Use wrong TPM signer
		MetaKeyID: keyID,       // But correct key ID
	})
	require.NoError(t, err, "Failed to create wrong TPM signer")

	// Create signed client
	baseClient := &http.Client{Timeout: 10 * time.Second}
	signedClient := httpsig.NewHTTPClient(baseClient, signer, nil)

	// Make request (should fail verification)
	resp, err := signedClient.Get(testServer.URL + "/test-endpoint")
	require.NoError(t, err, "Failed to make HTTP request")
	defer resp.Body.Close()

	// Verify that the signature verification failed
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Expected unauthorized response due to TPM signature mismatch")

	t.Logf("✅ TPM HTTP signature verification failure test passed!")
	t.Logf("🚫 Invalid TPM signature correctly rejected")
	t.Logf("📊 Response status: %d", resp.StatusCode)
}

// isTPMAvailable checks if a TPM device is available on the system
func isTPMAvailable() bool {
	// Check for TPM resource manager device first (preferred)
	if _, err := os.Stat("/dev/tpmrm0"); err == nil {
		return true
	}

	// Check for direct TPM device
	if _, err := os.Stat("/dev/tpm0"); err == nil {
		return true
	}

	return false
}
