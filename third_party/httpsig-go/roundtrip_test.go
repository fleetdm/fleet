package httpsig_test

import (
	"crypto"
	"testing"

	"github.com/remitly-oss/httpsig-go"
	"github.com/remitly-oss/httpsig-go/keyman"
	"github.com/remitly-oss/httpsig-go/keyutil"
	"github.com/remitly-oss/httpsig-go/sigtest"
)

// TestRoundTrip tests that the signing code can be verified by the verify code.
func TestRoundTrip(t *testing.T) {

	testcases := []struct {
		Name                  string
		PrivateKey            crypto.PrivateKey
		MetaKeyID             string
		Secret                []byte
		SignProfile           httpsig.SigningProfile
		RequestFile           string
		Keys                  httpsig.KeyFetcher
		Profile               httpsig.VerifyProfile
		ExpectedErrCodeVerify httpsig.ErrCode
	}{
		{
			Name:       "RSA-PSS",
			PrivateKey: keyutil.MustReadPrivateKeyFile("testdata/test-key-rsa-pss.key"),
			MetaKeyID:  "test-key-rsa",
			SignProfile: httpsig.SigningProfile{
				Algorithm: httpsig.Algo_RSA_PSS_SHA512,
				Fields:    httpsig.DefaultRequiredFields,
				Metadata:  []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID},
				Label:     "tst-rsa-pss",
			},
			RequestFile: "rfc-test-request.txt",
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-rsa": {
					KeyID:  "test-key-rsa",
					Algo:   httpsig.Algo_RSA_PSS_SHA512,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-rsa-pss.pub"),
				},
			}),
			Profile: createVerifyProfile("tst-rsa-pss"),
		},
		{
			Name:       "RSA-v15",
			PrivateKey: keyutil.MustReadPrivateKeyFile("testdata/key-rsa-v15.key"),
			MetaKeyID:  "test-key-rsa",
			SignProfile: httpsig.SigningProfile{
				Algorithm: httpsig.Algo_RSA_v1_5_sha256,
				Fields:    httpsig.DefaultRequiredFields,
				Metadata:  []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID},
				Label:     "tst-rsa-pss",
			},
			RequestFile: "rfc-test-request.txt",
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-rsa": {
					KeyID:  "test-key-rsa",
					Algo:   httpsig.Algo_RSA_v1_5_sha256,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/key-rsa-v15.pub"),
				},
			}),
			Profile: createVerifyProfile("tst-rsa-pss"),
		},
		{
			Name:      "HMAC_SHA256",
			Secret:    sigtest.MustReadFile("testdata/test-shared-secret"),
			MetaKeyID: "test-key-shared",
			SignProfile: httpsig.SigningProfile{
				Algorithm: httpsig.Algo_HMAC_SHA256,
				Fields:    httpsig.DefaultRequiredFields,
				Metadata:  []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID},
			},
			RequestFile: "rfc-test-request.txt",
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-shared": {
					KeyID:  "test-key-shared",
					Algo:   httpsig.Algo_HMAC_SHA256,
					Secret: sigtest.MustReadFile("testdata/test-shared-secret"),
				},
			}),
			Profile: createVerifyProfile("sig1"),
		},
		{
			Name:       "ECDSA-p265",
			PrivateKey: keyutil.MustReadPrivateKeyFile("testdata/test-key-ecc-p256.key"),
			MetaKeyID:  "test-key-ecdsa",
			SignProfile: httpsig.SigningProfile{
				Algorithm: httpsig.Algo_ECDSA_P256_SHA256,
				Fields:    httpsig.DefaultRequiredFields,
				Metadata:  []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID},
				Label:     "tst-ecdsa",
			},
			RequestFile: "rfc-test-request.txt",
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-ecdsa": {
					KeyID:  "test-key-ecds",
					Algo:   httpsig.Algo_ECDSA_P256_SHA256,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-ecc-p256.pub"),
				},
			}),
			Profile: createVerifyProfile("tst-ecdsa"),
		},
		{
			Name:       "ECDSA-p384",
			PrivateKey: keyutil.MustReadPrivateKeyFile("testdata/test-key-ecc-p384.key"),
			MetaKeyID:  "test-key-ecdsa",
			SignProfile: httpsig.SigningProfile{
				Algorithm: httpsig.Algo_ECDSA_P384_SHA384,
				Fields:    httpsig.DefaultRequiredFields,
				Metadata:  []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID},
				Label:     "tst-ecdsa",
			},
			RequestFile: "rfc-test-request.txt",
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-ecdsa": {
					KeyID:  "test-key-ecdsa",
					Algo:   httpsig.Algo_ECDSA_P384_SHA384,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-ecc-p384.pub"),
				},
			}),
			Profile: createVerifyProfile("tst-ecdsa"),
		},
		{
			Name:       "ED25519",
			PrivateKey: keyutil.MustReadPrivateKeyFile("testdata/test-key-ed25519.key"),
			MetaKeyID:  "test-key-ed",
			SignProfile: httpsig.SigningProfile{
				Algorithm: httpsig.Algo_ED25519,
				Fields:    httpsig.DefaultRequiredFields,
				Metadata:  []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID},
				Label:     "tst-ed",
			},
			RequestFile: "rfc-test-request.txt",
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-ed": {
					KeyID:  "test-key-ed",
					Algo:   httpsig.Algo_ED25519,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-ed25519.pub"),
				},
			}),
			Profile: createVerifyProfile("tst-ed"),
		},
		{
			Name:       "BadDigest",
			PrivateKey: keyutil.MustReadPrivateKeyFile("testdata/test-key-ed25519.key"),
			MetaKeyID:  "test-key-ed",
			SignProfile: httpsig.SigningProfile{

				Algorithm: httpsig.Algo_ED25519,
				Fields:    httpsig.DefaultRequiredFields,
				Metadata:  []httpsig.Metadata{httpsig.MetaCreated, httpsig.MetaKeyID},
				Label:     "tst-content-digest",
			},
			RequestFile: "request_bad_digest.txt",
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-ed": {
					KeyID:  "test-key-ed",
					Algo:   httpsig.Algo_ED25519,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-ed25519.pub"),
				},
			}),
			Profile:               createVerifyProfile("tst-content-digest"),
			ExpectedErrCodeVerify: httpsig.ErrNoSigWrongDigest,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var signer *httpsig.Signer
			sk := httpsig.SigningKey{
				Key:       tc.PrivateKey,
				Secret:    tc.Secret,
				MetaKeyID: tc.MetaKeyID,
			}

			signer, err := httpsig.NewSigner(tc.SignProfile, sk)
			if err != nil {
				t.Fatal(err)
			}

			req := sigtest.ReadRequest(t, tc.RequestFile)
			err = signer.Sign(req)
			if err != nil {
				t.Fatalf("%#v", err)
			}
			t.Log(req.Header.Get("Signature-Input"))
			t.Log(req.Header.Get("Signature"))
			ver, err := httpsig.NewVerifier(tc.Keys, tc.Profile)
			if err != nil {
				t.Fatal(err)
			}
			vf, err := ver.Verify(req)
			if err != nil {
				if tc.ExpectedErrCodeVerify != "" {
					if sigerr, ok := err.(*httpsig.SignatureError); ok {
						sigtest.Diff(t, tc.ExpectedErrCodeVerify, sigerr.Code, "Wrong err code")
					}
				} else {
					t.Fatalf("%#v", err)
				}
			} else if tc.ExpectedErrCodeVerify != "" {
				t.Fatal("Expected error")
			}
			t.Logf("%+v\n", vf)
		})

	}
}

func createVerifyProfile(label string) httpsig.VerifyProfile {
	vp := httpsig.DefaultVerifyProfile
	vp.SignatureLabel = label
	return vp
}
