package httpsig

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/remitly-oss/httpsig-go/sigtest"
)

/*
   Test cases from https://www.rfc-editor.org/rfc/rfc9421.pdf
*/

// TestSpecVerify ensures that requests from spec can be verified
func TestSpecVerify(t *testing.T) {
	cases := []struct {
		Name                       string
		IsResponse                 bool
		Key                        KeySpec
		SignedRequestOrResonseFile string
		Skip                       bool
	}{
		{
			Name: "b21",
			Key: KeySpec{
				KeyID:  "test-key-rsa-pss",
				Algo:   Algo_RSA_PSS_SHA512,
				PubKey: sigtest.ReadTestPubkey(t, "test-key-rsa-pss.pub"),
			},
			SignedRequestOrResonseFile: "b21_request_signed.txt",
		},
		{
			Name: "b22",
			Key: KeySpec{
				KeyID:  "test-key-rsa-pss",
				Algo:   Algo_RSA_PSS_SHA512,
				PubKey: sigtest.ReadTestPubkey(t, "test-key-rsa-pss.pub"),
			},
			SignedRequestOrResonseFile: "b22_request_signed.txt",
		},
		{
			Name: "b23",
			Key: KeySpec{
				KeyID:  "test-key-rsa-pss",
				Algo:   Algo_RSA_PSS_SHA512,
				PubKey: sigtest.ReadTestPubkey(t, "test-key-rsa-pss.pub"),
			},
			SignedRequestOrResonseFile: "b23_request_signed.txt",
		},
		{
			Name:       "b24",
			IsResponse: true,
			Key: KeySpec{
				KeyID:  "test-key-ecc-p256",
				Algo:   Algo_ECDSA_P256_SHA256,
				PubKey: sigtest.ReadTestPubkey(t, "test-key-ecc-p256.pub"),
			},
			SignedRequestOrResonseFile: "b24_response_signed.txt",
		},
		{
			Name: "b25",
			Key: KeySpec{
				KeyID:  "test-shared-secret",
				Algo:   Algo_HMAC_SHA256,
				Secret: sigtest.ReadSharedSecret(t, "test-shared-secret"),
			},
			SignedRequestOrResonseFile: "b25_request_signed.txt",
		},
		{
			Name: "b26",
			Key: KeySpec{
				KeyID:  "test-key-ed25519",
				Algo:   Algo_ED25519,
				PubKey: sigtest.ReadTestPubkey(t, "test-key-ed25519.pub"),
			},
			SignedRequestOrResonseFile: "b26_request_signed.txt",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Skip {
				t.Skip(fmt.Sprintf("Skipping test %s", tc.Name))
			}

			hrrtxt, err := os.Open(fmt.Sprintf("testdata/%s", tc.SignedRequestOrResonseFile))
			if err != nil {
				t.Fatal(err)
			}

			ver, err := NewVerifier(&fixedKeyFetch{
				requiredKeyID: tc.Key.KeyID,
				key:           tc.Key,
			}, VerifyProfile{
				SignatureLabel: fmt.Sprintf("sig-%s", tc.Name),
			})

			var verifyErr error
			if tc.IsResponse {
				resp, err := http.ReadResponse(bufio.NewReader(hrrtxt), nil)
				if err != nil {
					t.Fatal(err)
				}
				_, verifyErr = ver.VerifyResponse(resp)
			} else {
				req, err := http.ReadRequest(bufio.NewReader(hrrtxt))
				if err != nil {
					t.Fatal(err)
				}
				_, verifyErr = ver.Verify(req)
			}

			if verifyErr != nil {
				t.Fatalf("%#v\n", verifyErr)
			}
		})
	}
}

// TestSpecBase test recreation of the signature bases from the spec
func TestSpecBase(t *testing.T) {
	cases := []testcaseSigBase{
		{
			Name: "b21",
			Params: sigBaseInput{
				Components: []componentID{},
				MetadataParams: []Metadata{
					MetaCreated,
					MetaKeyID,
					MetaNonce,
				},
				MetadataValues: fixedMetadataProvider{
					values: map[Metadata]any{
						MetaCreated: int64(1618884473),
						MetaKeyID:   "test-key-rsa-pss",
						MetaNonce:   "b3k2pp5k7z-50gnwp.yemd",
					},
				},
			},
			ExpectedFile: "b21_request_sigbase.txt",
		},
		{
			Name: "b22",
			Params: sigBaseInput{
				Components: makeComponentIDs(
					SignedField{
						Name: "@authority",
					},
					SignedField{
						Name: "content-digest",
					},
					SignedField{
						Name: "@query-param",
						Parameters: map[string]any{
							"name": "Pet",
						},
					},
				),
				MetadataParams: []Metadata{
					MetaCreated,
					MetaKeyID,
					MetaTag,
				},
				MetadataValues: fixedMetadataProvider{
					values: map[Metadata]any{
						MetaCreated: int64(1618884473),
						MetaKeyID:   "test-key-rsa-pss",
						MetaTag:     "header-example",
					},
				},
			},
			ExpectedFile: "b22_request_sigbase.txt",
		},
		{
			Name: "b23",
			Params: sigBaseInput{
				Components: makeComponentIDs(
					SignedField{
						Name: "date",
					},
					SignedField{
						Name: "@method",
					},
					SignedField{
						Name: "@path",
					},
					SignedField{
						Name: "@query",
					},
					SignedField{
						Name: "@authority",
					},
					SignedField{
						Name: "content-type",
					},
					SignedField{
						Name: "content-digest",
					},
					SignedField{
						Name: "content-length",
					},
				),
				MetadataParams: []Metadata{
					MetaCreated,
					MetaKeyID,
				},
				MetadataValues: fixedMetadataProvider{
					values: map[Metadata]any{
						MetaCreated: int64(1618884473),
						MetaKeyID:   "test-key-rsa-pss",
					},
				},
			},
			ExpectedFile: "b23_request_sigbase.txt",
		},
		{
			Name:       "b24",
			IsResponse: true,
			Params: sigBaseInput{
				Components: componentsIDs(Fields("@status", "content-type", "content-digest", "content-length")),
				MetadataParams: []Metadata{
					MetaCreated,
					MetaKeyID,
				},
				MetadataValues: fixedMetadataProvider{
					values: map[Metadata]any{
						MetaCreated: int64(1618884473),
						MetaKeyID:   "test-key-ecc-p256",
					},
				},
			},
			ExpectedFile: "b24_response_sigbase.txt",
		},
		{
			Name: "b25",
			Params: sigBaseInput{
				Components: componentsIDs(Fields("date", "@authority", "content-type")),
				MetadataParams: []Metadata{
					MetaCreated,
					MetaKeyID,
				},
				MetadataValues: fixedMetadataProvider{
					values: map[Metadata]any{
						MetaCreated: int64(1618884473),
						MetaKeyID:   "test-shared-secret",
					},
				},
			},
			ExpectedFile: "b25_request_sigbase.txt",
		},
		{
			Name: "b26",
			Params: sigBaseInput{
				Components: componentsIDs(Fields("date", "@method", "@path", "@authority", "content-type", "content-length")),
				MetadataParams: []Metadata{
					MetaCreated,
					MetaKeyID,
				},
				MetadataValues: fixedMetadataProvider{
					values: map[Metadata]any{
						MetaCreated: int64(1618884473),
						MetaKeyID:   "test-key-ed25519",
					},
				},
			},
			ExpectedFile: "b26_request_sigbase.txt",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			runTestSigBase(t, tc)
		})
	}
}

type fixedKeyFetch struct {
	requiredKeyID string // if not empty, Fetch will check if the input keyID matches
	key           KeySpec
}

func (kf fixedKeyFetch) FetchByKeyID(ctx context.Context, rh http.Header, keyID string) (KeySpecer, error) {
	if kf.requiredKeyID != "" && keyID != kf.requiredKeyID {
		return nil, &KeyError{
			error: fmt.Errorf("Invalid key id. Wanted '%s' got '%s'", kf.requiredKeyID, keyID),
		}
	}
	return kf.key, nil
}

func (kf fixedKeyFetch) Fetch(ctx context.Context, rh http.Header, md MetadataProvider) (KeySpecer, error) {
	return nil, fmt.Errorf("Fetch without a key id not supported.")
}

// TestSpecRecreateSignature recreates the signature in the test cases.
// Algorithms that include randomness in the signing (each signature is unique) cannot be tested in this way.
func TestSpecRecreateSignature(t *testing.T) {
	cases := []struct {
		Name         string
		Params       sigParameters
		ExpectedFile string
	}{
		{
			Name: "b25",
			Params: sigParameters{
				Base: sigBaseInput{
					Components: componentsIDs(Fields("date", "@authority", "content-type")),
					MetadataParams: []Metadata{
						MetaCreated,
						MetaKeyID,
					},
					MetadataValues: fixedMetadataProvider{
						values: map[Metadata]any{
							MetaCreated: int64(1618884473),
							MetaKeyID:   "test-shared-secret",
						},
					},
				},
				Algo:   Algo_HMAC_SHA256,
				Label:  "sig-b25",
				Secret: sigtest.ReadSharedSecret(t, "test-shared-secret"),
			},

			ExpectedFile: "b25_request_signed.txt",
		},
		{
			Name: "b26",
			Params: sigParameters{
				Base: sigBaseInput{
					Components: componentsIDs(Fields("date", "@method", "@path", "@authority", "content-type", "content-length")),
					MetadataParams: []Metadata{
						MetaCreated,
						MetaKeyID,
					},
					MetadataValues: fixedMetadataProvider{
						values: map[Metadata]any{
							MetaCreated: int64(1618884473),
							MetaKeyID:   "test-key-ed25519",
						},
					},
				},

				Algo:       Algo_ED25519,
				Label:      "sig-b26",
				PrivateKey: sigtest.ReadTestPrivateKey(t, "test-key-ed25519.key"),
			},
			ExpectedFile: "b26_request_signed.txt",
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reqtxt, err := os.Open("testdata/rfc-test-request.txt")
			if err != nil {
				t.Fatal(err)
			}

			req, err := http.ReadRequest(bufio.NewReader(reqtxt))
			if err != nil {
				t.Fatal(err)
			}

			err = sign(httpMessage{
				Req: req,
			}, tc.Params)
			if err != nil {
				t.Fatalf("%#v\n", err)
			}
			expectedtxt, err := os.Open(fmt.Sprintf("testdata/%s", tc.ExpectedFile))
			if err != nil {
				t.Fatal(err)
			}

			expected, err := http.ReadRequest(bufio.NewReader(expectedtxt))
			if err != nil {
				t.Fatal(err)
			}
			sigtest.Diff(t, expected.Header, req.Header, "")
		})
	}
}
