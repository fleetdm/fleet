package e2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

type bstokenDevice interface {
	IDer
	DoGetBootstrapToken(ctx context.Context) (*mdm.BootstrapToken, error)
	DoEscrowBootstrapToken(ctx context.Context, token []byte) error
}

// bstoken assumes d is a new enrollment and can have or not have a BootstrapToken stored yet.
func bstoken(t *testing.T, ctx context.Context, d bstokenDevice) {
	input := []byte("hello world")

	err := d.DoEscrowBootstrapToken(ctx, input)
	if err != nil {
		t.Fatal(err)
	}

	tok, err := d.DoGetBootstrapToken(ctx)
	if err != nil {
		t.Fatal(err)
	}

	x, err := base64.StdEncoding.DecodeString(string(tok.BootstrapToken))
	if err != nil {
		t.Fatal(err)
	}

	if have, want := x, input; !bytes.Equal(have, want) {
		t.Errorf("bootstrap token: have: %v, want: %v", string(have), string(want))
	}
}
