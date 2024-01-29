//go:build integration
// +build integration

package mysql

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func TestBSToken(t *testing.T) {
	if *flDSN == "" {
		t.Fatal("MySQL DSN flag not provided to test")
	}

	storage, err := New(WithDSN(*flDSN), WithDeleteCommands())
	if err != nil {
		t.Fatal(err)
	}

	var d Device
	d, err = enrollTestDevice(storage)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	t.Run("BSToken nil", func(t *testing.T) {
		tok, err := storage.RetrieveBootstrapToken(&mdm.Request{Context: ctx, EnrollID: d.EnrollID()}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if tok != nil {
			t.Fatal("Token for new device was nonnull")
		}
	})
	t.Run("BSToken set/get", func(t *testing.T) {
		data := []byte("test token")
		bsToken := mdm.BootstrapToken{BootstrapToken: make([]byte, base64.StdEncoding.EncodedLen(len(data)))}
		base64.StdEncoding.Encode(bsToken.BootstrapToken, data)
		testReq := &mdm.Request{Context: ctx, EnrollID: d.EnrollID()}
		err := storage.StoreBootstrapToken(testReq, &mdm.SetBootstrapToken{BootstrapToken: bsToken})
		if err != nil {
			t.Fatal(err)
		}

		tok, err := storage.RetrieveBootstrapToken(testReq, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(bsToken.BootstrapToken, tok.BootstrapToken) {
			t.Fatalf("Bootstap tokens disequal after roundtrip: %v!=%v", bsToken, tok)
		}
	})
}
