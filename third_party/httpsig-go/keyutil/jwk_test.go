package keyutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParseJWK(t *testing.T) {
	tests := []struct {
		Name                string
		InputFile           string // one of InputFile or Input is used
		Input               string
		Expected            JWK
		ExpectedErrContains string
	}{
		{
			Name:      "Valid EC JWK",
			InputFile: "testdata/test-jwk-ec.json",
			Expected: JWK{
				KeyType: "EC",
				KeyID:   "test-key-ecc-p256",
			},
		},
		{

			Name:      "Valid symmetric JWK",
			InputFile: "testdata/test-jwk-symmetric.json",
			Expected: JWK{

				KeyType: "oct",
				KeyID:   "test-symmetric-key",
			},
		},
		{
			Name:                "Invalid JSON",
			Input:               `{"kty": malformed`,
			ExpectedErrContains: "parse",
		},
		{
			Name:                "Empty input",
			Input:               "",
			ExpectedErrContains: "parse",
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			var actual JWK
			var actualErr error
			if tc.InputFile != "" {
				actual, actualErr = ReadJWKFile(tc.InputFile)
			} else {
				actual, actualErr = ReadJWK([]byte(tc.Input))
			}

			if actualErr != nil {
				if !strings.Contains(actualErr.Error(), tc.ExpectedErrContains) {
					Diff(t, tc.ExpectedErrContains, actualErr.Error(), "Wrong error")
				}
				return
			}

			Diff(t, tc.Expected, actual, "Wrong JWK", cmpopts.IgnoreUnexported(JWK{}))
		})
	}
}

func TestJWKMarshalRoundTrip(t *testing.T) {
	tests := []struct {
		name                string
		inputType           string
		expectedErrContains string
		keyid               string
		algorithm           string
	}{
		{
			name:      "EC Key Round Trip",
			inputType: "EC",
			keyid:     "mykey_123",
			algorithm: "myalgo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var pk crypto.PrivateKey
			switch tc.inputType {
			case "EC":
				var err error
				pk, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
			}

			original, err := FromPrivateKey(pk)
			original.KeyID = tc.keyid
			original.Algorithm = tc.algorithm
			if err != nil {
				if tc.expectedErrContains != "" {
					if !strings.Contains(err.Error(), tc.expectedErrContains) {
						t.Errorf("Expected error containing %q, got: %v", tc.expectedErrContains, err)
					}
					return
				}
				t.Fatalf("Failed to generate create JWK from private key: %v", err)
			}

			// Marshal JWK to JSON
			jsonBytes, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Failed to marshal JWK: %v", err)
			}

			// Unmarshal back to new JWK
			roundTripped, err := ReadJWK(jsonBytes)
			if err != nil {
				t.Fatalf("Failed to unmarshal round-tripped JWK: %v", err)
			}

			// Compare original and round-tripped JWKs
			Diff(t, original, roundTripped, "Round-tripped JWK differs from original", cmpopts.IgnoreUnexported(JWK{}))
			Diff(t, tc.keyid, roundTripped.KeyID, "Round-tripped JWK differs from original", cmpopts.IgnoreUnexported(JWK{}))
		})
	}
}

func Diff(t *testing.T, expected, actual any, msg string, opts ...cmp.Option) bool {
	if diff := cmp.Diff(expected, actual, opts...); diff != "" {
		t.Errorf("%s (-want +got):\n%s", msg, diff)
		return true
	}
	return false
}
