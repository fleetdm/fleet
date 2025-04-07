package e2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

type bstokenDevice interface {
	IDer
	DoGetBootstrapToken(ctx context.Context) (*mdm.BootstrapToken, error)
	DoEscrowBootstrapToken(ctx context.Context, token []byte) error
}

// bstoken assumes d is a new enrollment and has had no BootstrapToken stored yet.
func bstoken(t *testing.T, ctx context.Context, d bstokenDevice) {
	tok, err := d.DoGetBootstrapToken(ctx)
	if err != nil {
		// should not error. newly enrolled devices should not error
		// if their BS token is requested.
		t.Fatal(fmt.Errorf("error retrieving not-yet-escrowed bootstrap token: %w", err))
	}

	if tok != nil {
		t.Errorf("token for supposedly freshly enrolled device %s was not nil", d.ID())
	}

	input := []byte("hello world")

	err = d.DoEscrowBootstrapToken(ctx, input)
	if err != nil {
		t.Fatal(err)
	}

	tok, err = d.DoGetBootstrapToken(ctx)
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
