package mysql

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChallenges(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NewChallengeAlphabet", testNewChallengeAlphabet},
		{"ConsumeChallenge", testConsumeChallenge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// Regression test for #46990: the Windows ClientCertificateInstall/SCEP CSP rejects base64url's '_' and '-' as
// non-printable characters, so generated challenges must be strictly alphanumeric.
func testNewChallengeAlphabet(t *testing.T, ds *Datastore) {
	// base32.StdEncoding of 20 random bytes: exactly 32 chars from the strictly alphanumeric A-Z2-7 alphabet.
	base32Alphabet := regexp.MustCompile(`^[A-Z2-7]{32}$`)
	for range 100 {
		challenge, err := ds.NewChallenge(t.Context())
		require.NoError(t, err)
		require.Regexp(t, base32Alphabet, challenge)
	}
}

func testConsumeChallenge(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	challenge, err := ds.NewChallenge(ctx)
	require.NoError(t, err)

	// First consumption succeeds, second fails since the challenge is one-time use.
	require.NoError(t, ds.ConsumeChallenge(ctx, challenge))
	require.Error(t, ds.ConsumeChallenge(ctx, challenge))

	require.Error(t, ds.ConsumeChallenge(ctx, "nonexistent-challenge"))
	require.Error(t, ds.ConsumeChallenge(ctx, ""))
}
