package httpsig_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/remitly-oss/httpsig-go"
	"github.com/remitly-oss/httpsig-go/keyman"
	"github.com/remitly-oss/httpsig-go/keyutil"
	"github.com/remitly-oss/httpsig-go/sigtest"
)

func TestVerify(t *testing.T) {
	testcases := []struct {
		Name         string
		RequestFile  string
		Label        string
		AddDebugInfo bool
		Keys         httpsig.KeyFetcher
		Expected     httpsig.VerifyResult
	}{
		{
			Name:        "OneValid",
			Label:       "sig-b21",
			RequestFile: "verify_request1.txt",
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-rsa-pss": {
					KeyID:  "test-key-rsa-pss",
					Algo:   httpsig.Algo_RSA_PSS_SHA512,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-rsa-pss.pub"),
				},
			}),
			Expected: httpsig.VerifyResult{
				Verified: true,
				Label:    "sig-b21",
				MetadataProvider: &fixedMetadataProvider{map[httpsig.Metadata]any{
					httpsig.MetaKeyID:   "test-key-rsa-pss",
					httpsig.MetaCreated: int64(1618884473),
					httpsig.MetaNonce:   "b3k2pp5k7z-50gnwp.yemd",
				}},
				KeySpecer: httpsig.KeySpec{
					KeyID:  "test-key-rsa-pss",
					Algo:   httpsig.Algo_RSA_PSS_SHA512,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-rsa-pss.pub"),
				},
			},
		},
		{
			Name:         "OneValidDebug",
			Label:        "sig-b21",
			RequestFile:  "verify_request1.txt",
			AddDebugInfo: true,
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-rsa-pss": {
					KeyID:  "test-key-rsa-pss",
					Algo:   httpsig.Algo_RSA_PSS_SHA512,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-rsa-pss.pub"),
				},
			}),
			Expected: httpsig.VerifyResult{
				Verified: true,
				Label:    "sig-b21",
				MetadataProvider: &fixedMetadataProvider{map[httpsig.Metadata]any{
					httpsig.MetaKeyID:   "test-key-rsa-pss",
					httpsig.MetaCreated: int64(1618884473),
					httpsig.MetaNonce:   "b3k2pp5k7z-50gnwp.yemd",
				}},
				KeySpecer: httpsig.KeySpec{
					KeyID:  "test-key-rsa-pss",
					Algo:   httpsig.Algo_RSA_PSS_SHA512,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-rsa-pss.pub"),
				},
				DebugInfo: httpsig.VerifyDebugInfo{
					SignatureBase: `"@signature-params": ();created=1618884473;keyid="test-key-rsa-pss";nonce="b3k2pp5k7z-50gnwp.yemd"`,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := sigtest.ReadRequest(t, tc.RequestFile)
			if tc.AddDebugInfo {
				req = req.WithContext(httpsig.SetAddDebugInfo(req.Context()))
			}
			actual, err := httpsig.Verify(req, tc.Keys, httpsig.VerifyProfile{SignatureLabel: tc.Label})
			if err != nil {
				t.Fatal(err)
			}

			// VerifyResult is returned even when error is also returned.
			// Because VerifryResult embed Metadataprovider we first need diff ignoring the MetadataProvider
			sigtest.Diff(t, tc.Expected, actual, "Did not match",
				cmp.FilterPath(func(p cmp.Path) bool {
					return p.String() == "MetadataProvider"
				}, cmp.Ignore()))

			// Then diff the metadata provider
			sigtest.Diff(t, tc.Expected, actual, "Did not match", getCmdOpts()...)
		})
	}
}

func TestVerifyInvalid(t *testing.T) {
	testcases := []struct {
		Name        string
		RequestFile string
		Label       string
		Keys        httpsig.KeyFetcher
		Expected    httpsig.ErrCode
	}{
		{
			Name:        "SignatureVerificationFailure",
			RequestFile: "verify_request2.txt",
			Label:       "bad-sig",
			Keys: keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
				"test-key-rsa-pss": {
					KeyID:  "test-key-rsa-pss",
					Algo:   httpsig.Algo_RSA_PSS_SHA512,
					PubKey: keyutil.MustReadPublicKeyFile("testdata/test-key-rsa-pss.pub"),
				},
			}),
			Expected: httpsig.ErrSigVerification,
		},
		{
			Name:        "KeyFetchError",
			RequestFile: "verify_request2.txt",
			Label:       "sig-b21",
			Keys:        keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{}),
			Expected:    httpsig.ErrSigKeyFetch,
		},
		{
			Name:        "KeyFetchError2",
			RequestFile: "verify_request2.txt",
			Label:       "bad-sig",
			Keys:        keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{}),
			Expected:    httpsig.ErrSigKeyFetch,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := httpsig.Verify(sigtest.ReadRequest(t, tc.RequestFile), tc.Keys, httpsig.VerifyProfile{SignatureLabel: tc.Label})

			if err == nil {
				t.Fatal("Expected err")
			}
			if sigerr, ok := err.(*httpsig.SignatureError); ok {
				sigtest.Diff(t, tc.Expected, sigerr.Code, "Did not match")
			} else {
				sigtest.Diff(t, tc.Expected, sigerr, "Did not match")
			}
		})
	}
}

type fixedMetadataProvider struct {
	values map[httpsig.Metadata]any
}

func (fmp fixedMetadataProvider) Created() (int, error) {
	if val, ok := fmp.values[httpsig.MetaCreated]; ok {
		return int(val.(int64)), nil
	}
	return 0, fmt.Errorf("No created value")
}

func (fmp fixedMetadataProvider) Expires() (int, error) {
	if val, ok := fmp.values[httpsig.MetaExpires]; ok {
		return int(val.(int64)), nil
	}
	return 0, fmt.Errorf("No expires value")
}

func (fmp fixedMetadataProvider) Nonce() (string, error) {
	if val, ok := fmp.values[httpsig.MetaNonce]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No nonce value")
}

func (fmp fixedMetadataProvider) Alg() (string, error) {
	if val, ok := fmp.values[httpsig.MetaAlgorithm]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No alg value")
}

func (fmp fixedMetadataProvider) KeyID() (string, error) {
	if val, ok := fmp.values[httpsig.MetaKeyID]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No keyid value")
}

func (fmp fixedMetadataProvider) Tag() (string, error) {
	if val, ok := fmp.values[httpsig.MetaTag]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No tag value")
}

func metaVal[E comparable](f1 func() (E, error)) any {
	val, err := f1()
	if err != nil {
		return err.Error()
	}
	return val
}

func getCmdOpts() []cmp.Option {
	return []cmp.Option{
		// This gets used for *ANY* struct assignable to MetadataProvider including other structres
		// that embed it!
		cmp.Transformer("MetadataProvider", TransformMeta),
	}
}

func TransformMeta(md httpsig.MetadataProvider) map[string]any {
	out := map[string]any{}

	if md == nil {
		return out
	}
	out[string(httpsig.MetaCreated)] = metaVal(md.Created)
	out[string(httpsig.MetaExpires)] = metaVal(md.Expires)
	out[string(httpsig.MetaNonce)] = metaVal(md.Nonce)
	out[string(httpsig.MetaAlgorithm)] = metaVal(md.Alg)
	out[string(httpsig.MetaKeyID)] = metaVal(md.KeyID)
	out[string(httpsig.MetaTag)] = metaVal(md.Tag)
	return out
}
