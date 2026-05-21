package types

import (
	"crypto/rand"
	"encoding/base64"
)

const nonceRawByteSize = 16 // 128 bits

// per the RFC (https://datatracker.ietf.org/doc/html/rfc8555/#section-6.5):
// > The precise method used to generate and track nonces is up to the
// > server.  For example, the server could generate a random 128-bit
// > value
func CreateRawNonce() string {
	nonce := make([]byte, nonceRawByteSize)
	// as per rand.Read documentation, never returns an error and always fills the entire slice
	_, _ = rand.Read(nonce)
	return string(nonce)
}

// per the RFC (https://datatracker.ietf.org/doc/html/rfc8555/#section-6.5.1):
// > The value of the Replay-Nonce header field MUST be an octet string
// > encoded according to the base64url encoding described in Section 2 of
// > [RFC7515].
func CreateNonceEncodedForHeader() string {
	nonce := CreateRawNonce()
	return base64.RawURLEncoding.EncodeToString([]byte(nonce))
}
