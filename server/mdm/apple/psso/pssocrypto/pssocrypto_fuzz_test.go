package pssocrypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	fuzzPEMPublicKey = "-----BEGIN PUBLIC KEY-----\n" +
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEaxfR8uEsQkf4vOblY6RA8ncDfYEt\n" +
		"6zOg9KE5RdiYwpZP40Li/hp/m47n60p8D54WK84zV2sxXs7LtkBoN79R9Q==\n" +
		"-----END PUBLIC KEY-----\n"
	fuzzRawPublicKey = "046b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c2964fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5"
	fuzzPartyInfoJWE = "eyJhbGciOiJFQ0RILUVTIiwiYXB1IjoiQUFBQUJVRlFVRXhGQUFBQVFRUjhmcGNaMy1RaUZEcmZDMWQ0NU9iWlZyQmEyU19wdTVPMUVsRVF5X0FfOW51T2ZnTFZJa1dET09DY3NGckh2dGZaM05leEJENlZXY2VzdTJiM2N2cWgiLCJhcHYiOiJBQUFBQlVGd2NHeGxBQUFBUVFSckY5SHk0U3hDUl9pODV1VmpwRUR5ZHdOOWdTM3JNNkQwb1RsRjJKakNsa19qUXVMLUduLWJqdWZyU253UG5oWXJ6ak5YYXpGZXpzdTJRR2czdjFIMUFBQUFEMjV2Ym1ObExXWjFlbm90YzJWbFpBIiwiZW5jIjoiQTI1NkdDTSIsImVwayI6eyJrdHkiOiJFQyIsImNydiI6IlAtMjU2IiwieCI6ImZINlhHZF9rSWhRNjN3dFhlT1RtMlZhd1d0a3Y2YnVUdFJKUkVNdndQX1kiLCJ5IjoiZTQ1LUF0VWlSWU00NEp5d1dzZS0xOW5jMTdFRVBwVlp4Nnk3WnZkeS1xRSJ9LCJ0eXAiOiJwbGF0Zm9ybXNzby1lbmNyeXB0ZWQtbG9naW4tYXNzZXJ0aW9uK2p3dCJ9..L3T6CxG6RI6v1HeM.FmXZqwZ0buPqLMPEw09uOxnO5FRjbcc2yryHRQ.li7URrHRofdwz1mpmc2R4A"
)

var fuzzAssertionPlaintext = []byte(`{"password":"fuzz-password"}`)

func fuzzP256PrivateKey(t testing.TB, scalar byte) *ecdsa.PrivateKey {
	t.Helper()
	raw := make([]byte, 32)
	raw[len(raw)-1] = scalar
	key, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), raw)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func rewriteJWEProtected(t testing.TB, compact string, mutate func(map[string]json.RawMessage)) string {
	t.Helper()
	parts := splitCompactJWE(compact)
	require.Len(t, parts, 5)
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(header, &fields))
	mutate(fields)
	header, err = json.Marshal(fields)
	require.NoError(t, err)
	parts[0] = base64.RawURLEncoding.EncodeToString(header)
	return joinCompactJWE(parts)
}

func splitCompactJWE(compact string) []string {
	parts := make([]string, 0, 5)
	start := 0
	for i := 0; i < len(compact); i++ {
		if compact[i] == '.' {
			parts = append(parts, compact[start:i])
			start = i + 1
		}
	}
	return append(parts, compact[start:])
}

func joinCompactJWE(parts []string) string {
	return parts[0] + "." + parts[1] + "." + parts[2] + "." + parts[3] + "." + parts[4]
}

func FuzzParseApplePartyInfo(f *testing.F) {
	valid := EncodeApplePartyInfo([]byte(APVPartyLabel), []byte{1, 2, 3}, []byte("nonce"))
	f.Add(valid)
	f.Add(EncodeApplePartyInfo([]byte{}, []byte("one")))
	f.Add([]byte{0, 0, 0})
	f.Add([]byte{0, 0, 0, 2, 1})

	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 8192 {
			t.Skip()
		}
		fields, err := ParseApplePartyInfo(raw)
		if err != nil {
			if bytes.Equal(raw, valid) {
				require.NoError(t, err, "fixed party-info seed must parse")
			}
			return
		}
		require.True(t, bytes.Equal(raw, EncodeApplePartyInfo(fields...)))
	})
}

func FuzzEncodeApplePartyInfo(f *testing.F) {
	f.Add([]byte("Apple"))
	f.Add([]byte{0, 1, 0, 255})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 8192 {
			t.Skip()
		}
		fields := make([][]byte, 0, len(input)+1)
		start := 0
		for i, b := range input {
			if b == 0 {
				fields = append(fields, append([]byte{}, input[start:i]...))
				start = i + 1
			}
		}
		fields = append(fields, append([]byte{}, input[start:]...))
		encoded := EncodeApplePartyInfo(fields...)
		decoded, err := ParseApplePartyInfo(encoded)
		require.NoError(t, err)
		require.Len(t, decoded, len(fields))
		for i := range fields {
			require.True(t, bytes.Equal(fields[i], decoded[i]))
		}
	})
}

func FuzzParseECPublicKeyPEM(f *testing.F) {
	f.Add([]byte(fuzzPEMPublicKey))
	raw, err := hex.DecodeString(fuzzRawPublicKey)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: raw}))
	f.Add([]byte("not PEM"))

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 4096 {
			t.Skip()
		}
		key, err := ParseECPublicKeyPEM(input)
		if err != nil {
			if bytes.Equal(input, []byte(fuzzPEMPublicKey)) ||
				bytes.Equal(input, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: raw})) {
				require.NoError(t, err, "fixed public-key seed must parse")
			}
			return
		}
		require.Equal(t, elliptic.P256(), key.Curve)
		raw, err := RawECPoint(key)
		require.NoError(t, err)
		roundTrip, err := ParseRawECPoint(raw)
		require.NoError(t, err)
		roundTripRaw, err := RawECPoint(roundTrip)
		require.NoError(t, err)
		require.Equal(t, raw, roundTripRaw)
	})
}

func FuzzParseRawECPoint(f *testing.F) {
	raw, err := hex.DecodeString(fuzzRawPublicKey)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(raw)
	f.Add([]byte{})
	f.Add([]byte{4})

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 256 {
			t.Skip()
		}
		key, err := ParseRawECPoint(input)
		if err != nil {
			if bytes.Equal(input, raw) {
				require.NoError(t, err, "fixed raw point seed must parse")
			}
			return
		}
		require.Equal(t, elliptic.P256(), key.Curve)
		got, err := RawECPoint(key)
		require.NoError(t, err)
		require.Equal(t, input, got)
	})
}

func FuzzDecryptPartyInfoJWE(f *testing.F) {
	key := fuzzP256PrivateKey(f, 1)
	var invalidHeaders = make(map[string]struct{})
	f.Add([]byte(fuzzPartyInfoJWE))
	invalidAlg := rewriteJWEProtected(f, fuzzPartyInfoJWE, func(header map[string]json.RawMessage) {
		header["alg"] = json.RawMessage(`"dir"`)
	})
	invalidHeaders[invalidAlg] = struct{}{}
	f.Add([]byte(invalidAlg))
	invalidEnc := rewriteJWEProtected(f, fuzzPartyInfoJWE, func(header map[string]json.RawMessage) {
		header["enc"] = json.RawMessage(`"A128GCM"`)
	})
	invalidHeaders[invalidEnc] = struct{}{}
	f.Add([]byte(invalidEnc))
	invalidTyp := rewriteJWEProtected(f, fuzzPartyInfoJWE, func(header map[string]json.RawMessage) {
		header["typ"] = json.RawMessage(`"unexpected+jwt"`)
	})
	invalidHeaders[invalidTyp] = struct{}{}
	f.Add([]byte(invalidTyp))
	wrongRecipient := fuzzP256PrivateKey(f, 2)

	f.Fuzz(func(t *testing.T, compact []byte) {
		if len(compact) > 4096 {
			t.Skip()
		}
		plaintext, err := DecryptPartyInfoJWE(compact, key, TypEncryptedLoginAssertion)
		if _, isInvalidHeaderSeed := invalidHeaders[string(compact)]; isInvalidHeaderSeed {
			require.Error(t, err, "mutated alg, enc, and typ headers must be rejected")
			return
		}
		if err != nil {
			if bytes.Equal(compact, []byte(fuzzPartyInfoJWE)) {
				require.NoError(t, err, "fixed valid JWE seed must decrypt")
			}
			return
		}
		require.Equal(t, fuzzAssertionPlaintext, plaintext)
		_, err = DecryptPartyInfoJWE(compact, wrongRecipient, TypEncryptedLoginAssertion)
		require.Error(t, err, "a different recipient must not decrypt the assertion")
	})
}

func FuzzLoginAssertionNonce(f *testing.F) {
	key := fuzzP256PrivateKey(f, 1)
	invalidSeeds := make(map[string]struct{})
	validSeeds := map[string]string{
		fuzzPartyInfoJWE: "nonce-fuzz-seed",
	}
	f.Add([]byte(fuzzPartyInfoJWE))
	padded := fuzzPartyInfoJWE + "="
	invalidSeeds[padded] = struct{}{}
	f.Add([]byte(padded))
	wrongAPV := rewriteJWEProtected(f, fuzzPartyInfoJWE, func(header map[string]json.RawMessage) {
		wrong, err := BuildAPV(&fuzzP256PrivateKey(f, 2).PublicKey, []byte("nonce-fuzz-seed"))
		if err != nil {
			f.Fatal(err)
		}
		encoded, err := json.Marshal(wrong)
		if err != nil {
			f.Fatal(err)
		}
		header["apv"] = encoded
	})
	invalidSeeds[wrongAPV] = struct{}{}
	f.Add([]byte(wrongAPV))
	changedNonceAPV, err := BuildAPV(&key.PublicKey, []byte("nonce-fuzz-seeX"))
	if err != nil {
		f.Fatal(err)
	}
	encodedChangedNonceAPV, err := json.Marshal(changedNonceAPV)
	if err != nil {
		f.Fatal(err)
	}
	// Nonce extraction parses the protected APV; it does not authenticate the JWE tag.
	changedNonce := rewriteJWEProtected(f, fuzzPartyInfoJWE, func(header map[string]json.RawMessage) {
		header["apv"] = encodedChangedNonceAPV
	})
	validSeeds[changedNonce] = "nonce-fuzz-seeX"
	f.Add([]byte(changedNonce))

	f.Fuzz(func(t *testing.T, compact []byte) {
		if len(compact) > 4096 {
			t.Skip()
		}
		nonce, err := LoginAssertionNonce(compact, &key.PublicKey)
		if _, isInvalidSeed := invalidSeeds[string(compact)]; isInvalidSeed {
			require.Error(t, err, "non-canonical encoding and wrong-recipient APV must be rejected")
			return
		}
		if err != nil {
			if _, isValidSeed := validSeeds[string(compact)]; isValidSeed {
				require.NoError(t, err, "fixed nonce seed must yield its APV nonce")
			}
			return
		}
		apvNonce, err := decodeAssertionAPVNonce(compact)
		require.NoError(t, err)
		require.Equal(t, string(apvNonce), nonce)
		if expected, isValidSeed := validSeeds[string(compact)]; isValidSeed {
			require.Equal(t, expected, nonce)
		}
	})
}

func decodeAssertionAPVNonce(compact []byte) ([]byte, error) {
	parts := strings.Split(string(compact), ".")
	if len(parts) != 5 {
		return nil, errors.New("not a compact JWE")
	}
	protected, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var header struct {
		APV string `json:"apv"`
	}
	if err := json.Unmarshal(protected, &header); err != nil {
		return nil, err
	}
	apv, err := DecodeJOSEB64(header.APV)
	if err != nil {
		return nil, err
	}
	fields, err := ParseApplePartyInfo(apv)
	if err != nil {
		return nil, err
	}
	if len(fields) != 3 {
		return nil, errors.New("APV must have three fields")
	}
	return fields[2], nil
}

func FuzzParseEmbeddedAssertionPassword(f *testing.F) {
	f.Add([]byte(`{"password":"fuzz-password","username":"ignored"}`))
	f.Add([]byte(`eyJhbGciOiJub25lIn0.eyJwYXNzd29yZCI6ImZ1enotcGFzc3dvcmQifQ.`))
	f.Add([]byte("not JSON or JWT"))

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 4096 {
			t.Skip()
		}
		password, err := ParseEmbeddedAssertionPassword(input)
		if err != nil {
			if bytes.Equal(input, []byte(`{"password":"fuzz-password","username":"ignored"}`)) ||
				bytes.Equal(input, []byte(`eyJhbGciOiJub25lIn0.eyJwYXNzd29yZCI6ImZ1enotcGFzc3dvcmQifQ.`)) {
				require.NoError(t, err, "fixed assertion seeds must parse")
			}
			return
		}
		encoded, err := BuildEmbeddedAssertionPlaintext(password)
		require.NoError(t, err)
		roundTrip, err := ParseEmbeddedAssertionPassword(encoded)
		require.NoError(t, err)
		require.Equal(t, password, roundTrip)
	})
}
