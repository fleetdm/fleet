package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/remitly-oss/httpsig-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// customSigner wraps an ECDSA private key and implements crypto.Signer
// This demonstrates how to create a custom signer implementation
type customSigner struct {
	privateKey *ecdsa.PrivateKey
	keyID      string
}

// newCustomSigner creates a new custom signer with the given private key
func newCustomSigner(privateKey *ecdsa.PrivateKey, keyID string) *customSigner {
	return &customSigner{
		privateKey: privateKey,
		keyID:      keyID,
	}
}

// Public returns the public key associated with this signer
func (signer *customSigner) Public() crypto.PublicKey {
	return &signer.privateKey.PublicKey
}

// Sign signs the given digest using the private key
// For P-384, we need to produce a 96-byte signature (48 bytes r + 48 bytes s)
func (signer *customSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	// Use the standard ECDSA signing
	r, s, err := ecdsa.Sign(rand, signer.privateKey, digest)
	if err != nil {
		return nil, err
	}

	// For P-384, each component should be 48 bytes
	keySize := (signer.privateKey.Curve.Params().BitSize + 7) / 8

	// Convert r and s to fixed-length byte arrays
	rBytes := r.FillBytes(make([]byte, keySize))
	sBytes := s.FillBytes(make([]byte, keySize))

	// Concatenate r and s for the signature format expected by httpsig
	signature := make([]byte, 2*keySize)
	copy(signature[:keySize], rBytes)
	copy(signature[keySize:], sBytes)

	return signature, nil
}

// KeyID returns the key identifier for this signer
func (signer *customSigner) KeyID() string {
	return signer.keyID
}

// inMemoryKeyStore holds keys in memory for HTTP signature verification
type inMemoryKeyStore struct {
	keys map[string]*httpsig.KeySpec // keyID -> KeySpec
}

func newInMemoryKeyStore() *inMemoryKeyStore {
	return &inMemoryKeyStore{
		keys: make(map[string]*httpsig.KeySpec),
	}
}

func (ks *inMemoryKeyStore) addKey(keyID string, keySpec *httpsig.KeySpec) {
	ks.keys[keyID] = keySpec
}

func (ks *inMemoryKeyStore) FetchByKeyID(ctx context.Context, _ http.Header, keyID string) (httpsig.KeySpecer, error) {
	keySpec, exists := ks.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found for keyID: %s", keyID)
	}
	return keySpec, nil
}

func (ks *inMemoryKeyStore) Fetch(_ context.Context, _ http.Header, _ httpsig.MetadataProvider) (httpsig.KeySpecer, error) {
	return nil, errors.New("not implemented")
}

// TestHTTPSignatureVerification tests HTTP signature creation and verification
// using ECC P-384 keys with everything kept in memory
func TestHTTPSignatureVerification(t *testing.T) {
	// Create ECC P-384 private key in memory
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err, "Failed to generate ECC P-384 private key")

	// Create a key ID for the signature
	keyID := "test-key-001"

	// Create custom signer that wraps the private key
	customSigner := newCustomSigner(privateKey, keyID)

	// Set up in-memory key store with the public key
	keyStore := newInMemoryKeyStore()
	keyStore.addKey(keyID, &httpsig.KeySpec{
		KeyID:  keyID,
		Algo:   httpsig.Algo_ECDSA_P384_SHA384,
		PubKey: customSigner.Public(), // Use the signer's public key method
	})

	// Set up HTTP signature verifier (server-side)
	verifier, err := httpsig.NewVerifier(keyStore, httpsig.VerifyProfile{
		SignatureLabel:     httpsig.DefaultSignatureLabel,
		AllowedAlgorithms:  []httpsig.Algorithm{httpsig.Algo_ECDSA_P256_SHA256, httpsig.Algo_ECDSA_P384_SHA384},
		RequiredFields:     httpsig.Fields("@method", "@target-uri", "content-digest"),
		RequiredMetadata:   []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID, httpsig.MetaNonce},
		DisallowedMetadata: []httpsig.Metadata{httpsig.MetaAlgorithm}, // Algorithm should be looked up from keyID
	})
	require.NoError(t, err, "Failed to create verifier")

	// Create test server with HTTP signature verification
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the HTTP signature
		_, err := verifier.Verify(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
			return
		}

		// If verification succeeds, return success response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("HTTP signature verified successfully!"))
	}))
	defer testServer.Close()

	// Set up HTTP signature signer (client-side) - similar to orbit.go
	signer, err := httpsig.NewSigner(httpsig.SigningProfile{
		Algorithm: httpsig.Algo_ECDSA_P384_SHA384, // Use P-384 as requested
		Fields:    httpsig.Fields("@method", "@target-uri", "content-digest"),
		Metadata:  []httpsig.Metadata{httpsig.MetaKeyID, httpsig.MetaCreated, httpsig.MetaNonce},
	}, httpsig.SigningKey{
		Key:       customSigner, // Use the custom signer that implements crypto.Signer
		MetaKeyID: keyID,        // Use the key ID
	})
	require.NoError(t, err, "Failed to create signer")

	// Create HTTP client with signature support - similar to orbit.go
	baseClient := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	signedClient := httpsig.NewHTTPClient(baseClient, signer, nil)

	// Test 1: Make a GET request to the test server
	resp, err := signedClient.Get(testServer.URL + "/test-endpoint")
	require.NoError(t, err, "Failed to make signed GET request")
	defer resp.Body.Close()

	// Verify the GET response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected successful GET response")

	// Read response body
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	responseText := string(body[:n])

	assert.Contains(t, responseText, "HTTP signature verified successfully!", "Expected success message in GET response")

	t.Logf("✅ GET request HTTP signature verification passed!")
	t.Logf("📝 GET Response: %s", responseText)

	// Test 2: Make a POST request with body content
	// Create JSON payload for POST request
	postData := map[string]interface{}{
		"message":   "Hello from signed POST request",
		"timestamp": time.Now().Unix(),
		"data":      []string{"item1", "item2", "item3"},
	}

	jsonData, err := json.Marshal(postData)
	require.NoError(t, err, "Failed to marshal JSON data")

	// Make POST request with JSON body
	postResp, err := signedClient.Post(testServer.URL+"/api/data", "application/json", bytes.NewReader(jsonData))
	require.NoError(t, err, "Failed to make signed POST request")
	defer postResp.Body.Close()

	// Verify the POST response
	assert.Equal(t, http.StatusOK, postResp.StatusCode, "Expected successful POST response")

	// Read POST response body
	postBody := make([]byte, 1024)
	postN, _ := postResp.Body.Read(postBody)
	postResponseText := string(postBody[:postN])

	assert.Contains(t, postResponseText, "HTTP signature verified successfully!", "Expected success message in POST response")

	t.Logf("✅ POST request HTTP signature verification passed!")
	t.Logf("📝 POST Response: %s", postResponseText)
	t.Logf("🔐 Used custom crypto.Signer with ECC P-384 key for signing both GET and POST")
	t.Logf("🛡️ Signatures verified successfully on server side")
	t.Logf("🔑 Key ID: %s", keyID)
	t.Logf("📊 POST payload size: %d bytes", len(jsonData))
	t.Logf("🔧 Custom signer wraps private key and implements crypto.Signer interface")
}

// TestHTTPSignatureVerificationFailure tests that invalid signatures are rejected
func TestHTTPSignatureVerificationFailure(t *testing.T) {
	// Create ECC P-384 private key for the key store
	correctPrivateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err, "Failed to generate correct ECC P-384 private key")

	// Create a different private key for signing (this should cause verification to fail)
	wrongPrivateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err, "Failed to generate wrong private key")

	keyID := "test-key-001"

	// Create custom signers
	correctSigner := newCustomSigner(correctPrivateKey, keyID)
	wrongSigner := newCustomSigner(wrongPrivateKey, keyID)

	// Set up key store with the correct public key
	keyStore := newInMemoryKeyStore()
	keyStore.addKey(keyID, &httpsig.KeySpec{
		KeyID:  keyID,
		Algo:   httpsig.Algo_ECDSA_P384_SHA384,
		PubKey: correctSigner.Public(), // Store correct public key from signer
	})

	// Set up verifier
	verifier, err := httpsig.NewVerifier(keyStore, httpsig.VerifyProfile{
		SignatureLabel:     httpsig.DefaultSignatureLabel,
		AllowedAlgorithms:  []httpsig.Algorithm{httpsig.Algo_ECDSA_P384_SHA384},
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
		_, _ = w.Write([]byte("This should not be reached"))
	}))
	defer testServer.Close()

	// Create signer with wrong private key
	signer, err := httpsig.NewSigner(httpsig.SigningProfile{
		Algorithm: httpsig.Algo_ECDSA_P384_SHA384,
		Fields:    httpsig.Fields("@method", "@target-uri", "content-digest"),
		Metadata:  []httpsig.Metadata{httpsig.MetaKeyID, httpsig.MetaCreated, httpsig.MetaNonce},
	}, httpsig.SigningKey{
		Key:       wrongSigner, // Use wrong custom signer
		MetaKeyID: keyID,       // But correct key ID
	})
	require.NoError(t, err, "Failed to create signer")

	// Create signed client
	baseClient := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	signedClient := httpsig.NewHTTPClient(baseClient, signer, nil)

	// Make request (should fail verification)
	resp, err := signedClient.Get(testServer.URL + "/test-endpoint")
	require.NoError(t, err, "Failed to make HTTP request")
	defer resp.Body.Close()

	// Verify that the signature verification failed
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Expected unauthorized response due to signature mismatch")

	t.Logf("✅ HTTP signature verification failure test passed!")
	t.Logf("🚫 Invalid signature correctly rejected")
	t.Logf("📊 Response status: %d", resp.StatusCode)
}
