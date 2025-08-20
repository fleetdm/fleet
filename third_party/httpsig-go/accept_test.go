package httpsig

import (
	"testing"

	"github.com/remitly-oss/httpsig-go/sigtest"
)

func TestAcceptParseSignature(t *testing.T) {
	testcases := []struct {
		Name            string
		Desc            string
		AcceptHeader    string
		Expected        AcceptSignature
		ExpectedErrCode ErrCode
	}{
		{
			Name:         "FromSpecification",
			Desc:         "Accept header used in the RFC",
			AcceptHeader: `sig1=("@method" "@target-uri" "@authority" "content-digest" "cache-control");keyid="test-key-rsa-pss";created;tag="app-123"`,
			Expected: AcceptSignature{
				MetaKeyID: "test-key-rsa-pss",
				MetaTag:   "app-123",
				Profile: SigningProfile{
					Fields:   Fields("@method", "@target-uri", "@authority", "content-digest", "cache-control"),
					Metadata: []Metadata{"keyid", "created", "tag"},
					Label:    "sig1",
				},
			},
		},
		{
			Name:         "InvalidAcceptSig",
			AcceptHeader: `("@method" "@target-uri" "@authority" "content-digest" "cache-control");keyid="test-key-rsa-pss";created;tag="app-123"`,

			ExpectedErrCode: ErrInvalidAcceptSignature,
		},
		{
			Name:            "NoAcceptSig",
			AcceptHeader:    "",
			ExpectedErrCode: ErrMissingAcceptSignature,
		},
		{
			Name:            "NotAList",
			AcceptHeader:    `sig1="@method"`,
			ExpectedErrCode: ErrInvalidAcceptSignature,
		},
		{
			Name:            "BadComponent",
			AcceptHeader:    `sig1=("@method" 1 "@authority" "content-digest" "cache-control");keyid="test-key-rsa-pss";created;tag="app-123"`,
			ExpectedErrCode: ErrInvalidAcceptSignature,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			actual, err := ParseAcceptSignature(tc.AcceptHeader)
			if sigtest.Diff(t, tc.ExpectedErrCode, errCode(err), "Wrong error code") {
				t.Logf("%+v\n", err)
				return
			}
			sigtest.Diff(t, tc.Expected, actual, "Wrong signature options")
		})
	}
}
