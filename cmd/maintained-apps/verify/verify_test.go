package main

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/ee/maintained-apps/sigverify"
	"github.com/stretchr/testify/require"
)

func TestClassifyAuthenticode(t *testing.T) {
	pin := &maintained_apps.FMASignature{SubjectCNs: []string{"Good Corp"}}
	unsignedPin := &maintained_apps.FMASignature{Unsigned: true, Justification: "vendor ships unsigned"}

	verified := &sigverify.AuthenticodeResult{Available: true, Verified: true, SubjectCNs: []string{"Good Corp"}}
	verifiedOtherCN := &sigverify.AuthenticodeResult{Available: true, Verified: true, SubjectCNs: []string{"Other Corp"}}
	untrustedChain := &sigverify.AuthenticodeResult{Available: true, SubjectCNs: []string{"Good Corp"}, Detail: "Signature verification: failed"}
	untrustedChainOtherCN := &sigverify.AuthenticodeResult{Available: true, SubjectCNs: []string{"Other Corp"}, Detail: "Signature verification: failed"}
	digestMismatch := &sigverify.AuthenticodeResult{Available: true, DigestMismatch: true, SubjectCNs: []string{"Good Corp"}, Detail: "Signature verification: failed"}
	unsigned := &sigverify.AuthenticodeResult{Available: true, NoSignature: true, Detail: "no Authenticode signature"}

	cases := []struct {
		name       string
		res        *sigverify.AuthenticodeResult
		pin        *maintained_apps.FMASignature
		wantStatus checkStatus
		wantFail   bool
		wantWarn   bool
	}{
		// No pin recorded yet.
		{"no pin, verified", verified, nil, statusRecorded, false, false},
		{"no pin, untrusted chain warns and defers", untrustedChain, nil, statusWarn, false, true},
		{"no pin, digest mismatch fails", digestMismatch, nil, statusFail, true, false},
		{"no pin, unsigned warns", unsigned, nil, statusWarn, false, true},

		// Identity pin present.
		{"pin, verified and CN matches", verified, pin, statusPass, false, false},
		{"pin, verified but CN changed", verifiedOtherCN, pin, statusFail, true, false},
		{"pin, untrusted chain but CN matches warns and defers", untrustedChain, pin, statusWarn, false, true},
		{"pin, untrusted chain and CN changed fails", untrustedChainOtherCN, pin, statusFail, true, false},
		{"pin, digest mismatch fails", digestMismatch, pin, statusFail, true, false},
		{"pin, unsigned fails", unsigned, pin, statusFail, true, false},

		// "unsigned" pin present.
		{"unsigned pin, unsigned", unsigned, unsignedPin, statusPass, false, false},
		{"unsigned pin, now validly signed warns", verified, unsignedPin, statusWarn, false, true},
		{"unsigned pin, now signed with untrusted chain warns and defers", untrustedChain, unsignedPin, statusWarn, false, true},
		{"unsigned pin, digest mismatch fails", digestMismatch, unsignedPin, statusFail, true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			av := &appVerification{}
			classifyAuthenticode(av, tc.res, tc.pin, "")
			require.Equal(t, tc.wantStatus, av.Signature.Status, "status; detail: %s", av.Signature.Detail)
			require.Equal(t, tc.wantFail, len(av.Failures) > 0, "failures: %v", av.Failures)
			require.Equal(t, tc.wantWarn, len(av.Warnings) > 0, "warnings: %v", av.Warnings)
			require.True(t, av.SignatureObservable)
			require.Equal(t, tc.res.SubjectCNs, av.ObservedSubjectCNs)
		})
	}
}
