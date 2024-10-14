package scepserver

import (
	"context"
	"testing"

	"github.com/smallstep/scep"
)

func TestChallengeMiddleware(t *testing.T) {
	testPW := "RIGHT"
	signer := StaticChallengeMiddleware(testPW, NopCSRSigner())

	csrReq := &scep.CSRReqMessage{ChallengePassword: testPW}

	ctx := context.Background()

	_, err := signer.SignCSRContext(ctx, csrReq)
	if err != nil {
		t.Error(err)
	}

	csrReq.ChallengePassword = "WRONG"

	_, err = signer.SignCSRContext(ctx, csrReq)
	if err == nil {
		t.Error("invalid challenge should generate an error")
	}
}
