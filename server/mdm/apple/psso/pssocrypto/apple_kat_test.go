package pssocrypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"math/big"
	"testing"

	josecipher "github.com/go-jose/go-jose/v3/cipher"
	"github.com/stretchr/testify/require"
)

// Known-answer test against Apple's published PSSO encryption-verification
// vectors: developer.apple.com/documentation/authenticationservices/performing-encryption-verification
//
// It exercises the exact ECDH + Concat-KDF path BuildPartyInfoJWE uses
// (josecipher.DeriveECDHES with an "APPLE"-labelled apu and the request's apv),
// proving Fleet derives the same shared secret and content-encryption key,
// byte for byte, that Apple's reference does. If go-jose, the party-info
// encoding, or the alg identifier ever drift, this fails.
func TestAppleEncryptionVerificationVectors(t *testing.T) {
	// Base64url (JWK / JOSE) — no padding.
	jwk := func(s string) []byte {
		b, err := base64.RawURLEncoding.DecodeString(s)
		require.NoError(t, err)
		return b
	}

	// Device encryption key (static recipient) from the doc.
	devPub := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(jwk("TvkPOH4yscrSC1rFYvnBVPYMqzR1vKck9ht4D7K_gAQ")),
		Y:     new(big.Int).SetBytes(jwk("4MlSuUf_7J6Ljv0FBT1jK0_sKGB4WYwdKCOtnTEAwz4")),
	}

	// Ephemeral key (sender) from the doc.
	eph := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(jwk("VIXdgu3x0eLgEVtROZ5YQ4GUS8WZQT-3HPqX2FPoY4I")),
			Y:     new(big.Int).SetBytes(jwk("erf9nEkEC8SiuwP-7f7udD7CnX5KEauVIfBPoqnmlYo")),
		},
		D: new(big.Int).SetBytes(jwk("okDfU7IYlFXoEKgqu-79iy-AR55omCKzVKlFCXy8z5c")),
	}

	// apu is what Fleet constructs; apv is echoed from the request.
	appleAPU := jwk("AAAABUFQUExFAAAAQQRUhd2C7fHR4uARW1E5nlhDgZRLxZlBP7cc-pfYU-hjgnq3_ZxJBAvEorsD_u3-7nQ-wp1-ShGrlSHwT6Kp5pWK")
	apvRaw := jwk("AAAABUFwcGxlAAAAQQRO-Q84fjKxytILWsVi-cFU9gyrNHW8pyT2G3gPsr-ABODJUrlH_-yei479BQU9YytP7ChgeFmMHSgjrZ0xAMM-AAAAJERERjY4MTcxLTQwOUQtNEUyQy05MUYwLTlFNDJENzc0NTM2NQ")

	wantZ := jwk("L87ywmD3aLpVlXsqAvq7udyr4s6M0y9MjQCytE71epA")
	wantCEK := jwk("kh36uWSGH25r09lLf3m5l3TLS5xKAs-h3UCdbTKheCY")

	// 1. Raw ECDH shared secret Z matches Apple's value. This is the same
	// crypto/ecdh path ComputeECDHShared uses.
	ephECDH, err := eph.ECDH()
	require.NoError(t, err)
	devPubECDH, err := devPub.ECDH()
	require.NoError(t, err)
	z, err := ephECDH.ECDH(devPubECDH)
	require.NoError(t, err)
	require.Equal(t, wantZ, z, "ECDH shared secret Z must match Apple's vector")

	// 2. Fleet's apu construction ("APPLE" || uncompressed epk) matches Apple's
	// apu byte-for-byte, via the same PublicKey().ECDH() path BuildPartyInfoJWE
	// uses (derives the point from the ephemeral X/Y, not the scalar).
	epkECDH, err := eph.PublicKey.ECDH()
	require.NoError(t, err)
	fleetAPU := EncodeApplePartyInfo([]byte(APUPartyLabel), epkECDH.Bytes())
	require.Equal(t, appleAPU, fleetAPU, "apu construction must match Apple's vector")

	// 3. The derived content-encryption key matches Apple's, using the identical
	// call BuildPartyInfoJWE makes: Concat KDF with alg id = A256GCM.
	cek := josecipher.DeriveECDHES(ContentEncryptionAlg, fleetAPU, apvRaw, eph, devPub, 32)
	require.Equal(t, wantCEK, cek, "Concat-KDF derived CEK must match Apple's vector")

	// 4. apv round-trips through Fleet's parser into Apple's documented fields:
	// "Apple" || 65-byte device point || 36-byte nonce UUID.
	fields, err := ParseApplePartyInfo(apvRaw)
	require.NoError(t, err)
	require.Len(t, fields, 3)
	require.Equal(t, APVPartyLabel, string(fields[0]))
	require.Len(t, fields[1], 65)
	require.Equal(t, "DDF68171-409D-4E2C-91F0-9E42D7745365", string(fields[2]))
}
