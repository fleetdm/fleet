package mdmcrypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSCEPVerifierVerifyEmptyCerts(t *testing.T) {
	v := &SCEPVerifier{}
	err := v.Verify(nil)
	require.ErrorContains(t, err, "no certificate provided")
}
