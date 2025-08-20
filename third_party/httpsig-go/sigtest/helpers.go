package sigtest

import (
	"bufio"
	"bytes"
	"crypto"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/remitly-oss/httpsig-go/keyutil"
)

func ReadRequest(t testing.TB, reqFile string) *http.Request {
	reqtxt, err := os.Open(fmt.Sprintf("testdata/%s", reqFile))
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.ReadRequest(bufio.NewReader(reqtxt))
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func MustReadFile(file string) []byte {
	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return data
}

func MakeBody(body string) io.ReadCloser {
	return io.NopCloser(bytes.NewReader([]byte(body)))
}

func ReadSharedSecret(t *testing.T, sharedSecretFile string) []byte {
	secretBytes, err := os.ReadFile(fmt.Sprintf("testdata/%s", sharedSecretFile))
	if err != nil {
		t.Fatal(err)
	}
	secret, err := base64.StdEncoding.DecodeString(string(secretBytes))
	if err != nil {
		t.Fatal(err)
	}
	return secret
}

func ReadTestPubkey(t *testing.T, pubkeyFile string) crypto.PublicKey {
	keybytes, err := os.ReadFile(fmt.Sprintf("testdata/%s", pubkeyFile))
	if err != nil {
		t.Fatal(err)
	}
	pubkey, err := keyutil.ReadPublicKey(keybytes)
	if err != nil {
		t.Fatal(err)
	}
	return pubkey
}

func ReadTestPrivateKey(t testing.TB, pkFile string, hint ...string) crypto.PrivateKey {
	keybytes, err := os.ReadFile(fmt.Sprintf("testdata/%s", pkFile))
	if err != nil {
		t.Fatal(err)
	}
	pkey, err := keyutil.ReadPrivateKey(keybytes)
	if err != nil {
		t.Fatal(err)
	}
	return pkey
}

func Diff(t *testing.T, expected, actual any, msg string, opts ...cmp.Option) bool {
	if diff := cmp.Diff(expected, actual, opts...); diff != "" {
		t.Errorf("%s (-want +got):\n%s", msg, diff)
		return true
	}
	return false
}
