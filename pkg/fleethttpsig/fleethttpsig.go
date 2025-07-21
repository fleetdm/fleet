// Package fleethttpsig is a common package to use by Fleet client and servers for HTTP signing/verification.
package fleethttpsig

import (
	"crypto"

	"github.com/remitly-oss/httpsig-go"
)

var (
	// requiredFields specifies the required fields in HTTP signed requests.
	// We are not using @target-uri in the signature so that we don't run into issues with HTTPS forwarding and proxies (http vs https).
	requiredFields = httpsig.Fields("@method", "@authority", "@path", "@query", "content-digest")

	requiredMetadata = []httpsig.Metadata{httpsig.MetaKeyID, httpsig.MetaCreated, httpsig.MetaNonce}
)

// Verifier returns a *httpsig.Verified configured for verifying signed HTTP requests from Fleet clients.
func Verifier(kf httpsig.KeyFetcher) (*httpsig.Verifier, error) {
	return httpsig.NewVerifier(kf, httpsig.VerifyProfile{
		SignatureLabel:    httpsig.DefaultSignatureLabel,
		AllowedAlgorithms: []httpsig.Algorithm{httpsig.Algo_ECDSA_P256_SHA256, httpsig.Algo_ECDSA_P384_SHA384},
		RequiredFields:    requiredFields,
		RequiredMetadata:  requiredMetadata,
		// The algorithm should be looked up from the keyid not an explicit setting.
		DisallowedMetadata: []httpsig.Metadata{httpsig.MetaAlgorithm},
	})
}

// Signer returns a *httpsig.Signer to sign HTTP requests to a Fleet server.
func Signer(metaKeyID string, signer crypto.Signer, signingAlgorithm httpsig.Algorithm) (*httpsig.Signer, error) {
	return httpsig.NewSigner(httpsig.SigningProfile{
		Algorithm: signingAlgorithm,
		Fields:    requiredFields,
		Metadata:  requiredMetadata,
	}, httpsig.SigningKey{
		Key:       signer,
		MetaKeyID: metaKeyID,
	})
}
