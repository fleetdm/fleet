package httpsig

import (
	"bufio"
	"bytes"
	"net/http"
	"os"
	"testing"

	"github.com/remitly-oss/httpsig-go/sigtest"
)

// FuzzSigningOptions fuzzes the basic user input to SigningOptions
func FuzzSigningOptions1(f *testing.F) {
	testcases := [][]string{
		{"", "", "", ""},
		{"", "0", "0", "\xde"},
		{"", "\n", "0", "0"},
		{"", "", "0", "@"},
		{"", "@query-param", "0", "0"},
		{string(Algo_ECDSA_P256_SHA256), "@query", "0", "0"},
		{"any", "@query", "0", "0"},
		{string(Algo_ED25519), "@query", "0", "0"},
	}

	for _, tc := range testcases {
		f.Add(tc[0], tc[1], tc[2], tc[3])
	}

	reqtxt, err := os.ReadFile("testdata/rfc-test-request.txt")
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, algo, label, keyID, tag string) {
		t.Logf("Label: %s\n", label)
		t.Logf("keyid: %s\n", keyID)
		t.Logf("tag: %s\n", tag)

		fields := Fields(label, keyID, tag)
		fields = append(fields, SignedField{
			Name: label,
			Parameters: map[string]any{
				keyID: tag,
			},
		})
		privKey := sigtest.ReadTestPrivateKey(t, "test-key-ed25519.key")
		so := SigningProfile{
			Algorithm: Algo_ED25519,
			Fields:    Fields(label, keyID, tag),
			Metadata:  []Metadata{MetaKeyID, MetaTag},
			Label:     label,
		}
		sk := SigningKey{
			Key:       privKey,
			MetaKeyID: keyID,
			MetaTag:   tag,
		}
		if so.validate(sk) != nil {
			// Catching invalidate signing options is good.
			return
		}

		req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(reqtxt)))
		if err != nil {
			t.Fatal(err)
		}

		err = Sign(req, so, sk)
		if err != nil {
			if _, ok := err.(*SignatureError); ok {
				// Handled error
				return
			}
			// Unhandled error
			t.Error(err)
		}
	})
}

func FuzzSigningOptionsFields(f *testing.F) {
	testcases := [][]string{
		{"", "", ""},
		{"0", "0", "\xde"},
		{"\n", "0", "0"},
		{"", "0", "@"},
		{"@query-param", "name", "0"},
		{"@query", "0", "0"},
		{"@method", "", ""},
		{"@status", "", ""},
	}

	for _, tc := range testcases {
		f.Add(tc[0], tc[1], tc[2])
	}

	reqtxt, err := os.ReadFile("testdata/rfc-test-request.txt")
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, field, tagName, tagValue string) {
		t.Logf("field: %s\n", field)
		t.Logf("tag: %s:%s\n", tagName, tagValue)
		fields := []SignedField{}
		if tagName == "" {
			fields = append(fields, SignedField{
				Name: field,
			})
		} else {
			fields = append(fields, SignedField{
				Name: field,
				Parameters: map[string]any{
					tagName: tagValue,
				},
			})
		}

		so := SigningProfile{
			Algorithm: Algo_ED25519,
			Fields:    fields,
		}
		sk := SigningKey{
			Key: sigtest.ReadTestPrivateKey(t, "test-key-ed25519.key"),
		}
		if so.validate(sk) != nil {
			// Catching invalidate signing options is good.
			return
		}

		req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(reqtxt)))
		if err != nil {
			t.Fatal(err)
		}

		err = Sign(req, so, sk)
		if err != nil {
			if _, ok := err.(*SignatureError); ok {
				// Handled error
				return
			}
			// Unhandled error
			t.Error(err)
		}
	})
}

func FuzzExtractSignatures(f *testing.F) {
	testcases := []struct {
		SignatureHeader      string
		SignatureInputHeader string
	}{
		{
			SignatureHeader:      "",
			SignatureInputHeader: "",
		},
		{
			SignatureHeader:      "sig-b24=(\"@status\" \"content-type\" \"content-digest\" \"content-length\");created=1618884473;keyid=\"test-key-ecc-p256\"",
			SignatureInputHeader: "sig-b24=:wNmSUAhwb5LxtOtOpNa6W5xj067m5hFrj0XQ4fvpaCLx0NKocgPquLgyahnzDnDAUy5eCdlYUEkLIj+32oiasw==:",
		},
	}

	for _, tc := range testcases {
		f.Add(tc.SignatureHeader, tc.SignatureInputHeader)
	}

	reqtxt, err := os.ReadFile("testdata/rfc-test-request.txt")
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, sigHeader, sigInputHeader string) {
		t.Logf("signature header: %s\n", sigHeader)
		t.Logf("signature input header: %s\n", sigInputHeader)

		req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(reqtxt)))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("signature", sigHeader)
		req.Header.Set("signature-input", sigInputHeader)

		sigSFV, err := parseSignaturesFromRequest(req.Header)
		if err != nil {
			return
		}
		for _, label := range sigSFV.Sigs.Names() {
			_, err = unmarshalSignature(sigSFV, label)
			if err != nil {
				if _, ok := err.(*SignatureError); ok {
					// Handled error
					return
				}
				// Unhandled error
				t.Error(err)
			}
		}
	})
}
