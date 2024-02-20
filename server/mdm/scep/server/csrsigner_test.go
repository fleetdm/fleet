package scepserver

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/scep"
)

func TestChallengeMiddleware(t *testing.T) {
	testPW := "RIGHT"
	signer := ChallengeMiddleware(testPW, NopCSRSigner())

	csrReq := &scep.CSRReqMessage{ChallengePassword: testPW}

	_, err := signer.SignCSR(csrReq)
	if err != nil {
		t.Error(err)
	}

	csrReq.ChallengePassword = "WRONG"

	_, err = signer.SignCSR(csrReq)
	if err == nil {
		t.Error("invalid challenge should generate an error")
	}
}
