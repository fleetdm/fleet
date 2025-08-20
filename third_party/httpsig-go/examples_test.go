package httpsig_test

import (
	"crypto"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"

	"github.com/remitly-oss/httpsig-go"
	"github.com/remitly-oss/httpsig-go/keyman"
	"github.com/remitly-oss/httpsig-go/keyutil"
)

func ExampleSign() {
	pkeyEncoded := `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgNTK6255ubaaj1i/c
ppuLouTgjAVyHGSxI0pYX8z1e2GhRANCAASkbVuWv1KXXs2H8b0ruFLyv2lKJWtT
BznPJ5sSI1Jn+srosJB/GbEZ3Kg6PcEi+jODF9fdpNEaHGbbGdaVhJi1
-----END PRIVATE KEY-----`

	pkey, _ := keyutil.ReadPrivateKey([]byte(pkeyEncoded))
	req := httptest.NewRequest("GET", "https://example.com/data", nil)

	profile := httpsig.SigningProfile{
		Algorithm: httpsig.Algo_ECDSA_P256_SHA256,
		Fields:    httpsig.DefaultRequiredFields,
		Metadata:  []httpsig.Metadata{httpsig.MetaKeyID},
	}
	skey := httpsig.SigningKey{
		Key:       pkey,
		MetaKeyID: "key123",
	}

	signer, _ := httpsig.NewSigner(profile, skey)
	signer.Sign(req)
}

func ExampleVerify() {
	pubkeyEncoded := `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEIUctKvU5L/eEYxua5Zlz0HIQJRQq
MTQ7eYQXwqpTvTJkuTffGXKLilT75wY2YZWfybv9flu5d6bCfw+4UB9+cg==
-----END PUBLIC KEY-----`

	pubkey, _ := keyutil.ReadPublicKey([]byte(pubkeyEncoded))
	req := httptest.NewRequest("GET", "https://example.com/data", nil)

	kf := keyman.NewKeyFetchInMemory(map[string]httpsig.KeySpec{
		"key123": {
			KeyID:  "key123",
			Algo:   httpsig.Algo_ECDSA_P256_SHA256,
			PubKey: pubkey,
		},
	})

	httpsig.Verify(req, kf, httpsig.DefaultVerifyProfile)
}

func ExampleNewHandler() {
	myhandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Lookup the results of verification
		if veriftyResult, ok := httpsig.GetVerifyResult(r.Context()); ok {
			keyid, _ := veriftyResult.KeyID()
			fmt.Fprintf(w, "Hello, %s", html.EscapeString(keyid))
		} else {
			fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
		}
	})

	// Create a verifier
	verifier, _ := httpsig.NewVerifier(nil, httpsig.DefaultVerifyProfile)

	mux := http.NewServeMux()
	// Wrap the handler with the a signature verification handler.
	mux.Handle("/", httpsig.NewHandler(myhandler, verifier))
}

func ExampleClient() {
	profile := httpsig.SigningProfile{
		Algorithm: httpsig.Algo_ECDSA_P256_SHA256,
		Fields:    httpsig.DefaultRequiredFields,
		Metadata:  []httpsig.Metadata{httpsig.MetaKeyID},
	}
	var privateKey crypto.PrivateKey // Get your private key

	sk := httpsig.SigningKey{
		Key:       privateKey,
		MetaKeyID: "key123",
	}
	// Create the signature signer
	signer, _ := httpsig.NewSigner(profile, sk)

	// Create a net/http Client that signs all requests
	signingClient := httpsig.NewHTTPClient(nil, signer, nil)

	// This call will be signed.
	signingClient.Get("https://example.com")

	verifier, _ := httpsig.NewVerifier(nil, httpsig.DefaultVerifyProfile)
	// Create a net/http Client that signs and verifies all requests
	signVerifyClient := httpsig.NewHTTPClient(nil, signer, verifier)

	signVerifyClient.Get("https://example.com")
}
