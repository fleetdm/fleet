package httpsig

import (
	"encoding/base64"
	"errors"
	"io"
	"testing"

	"github.com/remitly-oss/httpsig-go/sigtest"
)

func TestDigestCreate(t *testing.T) {
	testcases := []struct {
		Name            string
		Algo            Digest
		Body            io.ReadCloser
		ExpectedDigest  string // base64 encoded digest
		ExpectedHeader  string
		ExpectedErrCode ErrCode
	}{
		{
			Name:           "sha-256",
			Algo:           DigestSHA256,
			Body:           sigtest.MakeBody("hello world"),
			ExpectedDigest: "uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek=",
			ExpectedHeader: "sha-256=:uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek=:",
		},
		{
			Name:           "sha-512",
			Algo:           DigestSHA512,
			Body:           sigtest.MakeBody("hello world"),
			ExpectedDigest: "MJ7MSJwS1utMxA9QyQLytNDtd+5RGnx6m808qG1M2G+YndNbxf9JlnDaNCVbRbDP2DDoH2Bdz33FVC6TrpzXbw==",
			ExpectedHeader: "sha-512=:MJ7MSJwS1utMxA9QyQLytNDtd+5RGnx6m808qG1M2G+YndNbxf9JlnDaNCVbRbDP2DDoH2Bdz33FVC6TrpzXbw==:",
		},
		{
			Name:            "UnsupportedAlgorithm",
			Algo:            Digest("nope"),
			Body:            sigtest.MakeBody("hello world"),
			ExpectedErrCode: ErrNoSigUnsupportedDigest,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			actual, err := digestBody(tc.Algo, tc.Body)
			if err != nil {
				if tc.ExpectedErrCode != "" {
					diffErrorCode(t, err, tc.ExpectedErrCode)
					return
				}
				t.Fatal(err)
			}
			actualEncoded := base64.StdEncoding.EncodeToString(actual.Digest)
			sigtest.Diff(t, tc.ExpectedDigest, actualEncoded, "Wrong digest")

			actualHeader, err := createDigestHeader(tc.Algo, actual.Digest)
			if err != nil {
				t.Fatal(err)
			}
			sigtest.Diff(t, tc.ExpectedHeader, actualHeader, "Wrong digest header")
		})
	}
}

func TestDigestParse(t *testing.T) {
	testcases := []struct {
		Name            string
		Header          []string
		ExcepctedAlgo   Digest
		ExpectedDigest  string // base64 encoded digest
		ExpectedErrCode ErrCode
	}{
		{
			Name:           "sha-256",
			Header:         []string{"sha-256=:uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek=:"},
			ExcepctedAlgo:  DigestSHA256,
			ExpectedDigest: "uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek=",
		},
		{
			Name:           "sha-512",
			Header:         []string{"sha-512=:MJ7MSJwS1utMxA9QyQLytNDtd+5RGnx6m808qG1M2G+YndNbxf9JlnDaNCVbRbDP2DDoH2Bdz33FVC6TrpzXbw==:"},
			ExcepctedAlgo:  DigestSHA512,
			ExpectedDigest: "MJ7MSJwS1utMxA9QyQLytNDtd+5RGnx6m808qG1M2G+YndNbxf9JlnDaNCVbRbDP2DDoH2Bdz33FVC6TrpzXbw==",
		},
		{
			Name:           "Empty",
			Header:         []string{},
			ExpectedDigest: "",
		},
		{
			Name:            "BadHeader",
			Header:          []string{"bl===ah"},
			ExpectedErrCode: ErrNoSigInvalidHeader,
		},
		{
			Name:           "Unsupported",
			Header:         []string{"md5=:blah:"},
			ExpectedDigest: "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			actualAlgo, actualDigest, err := getSupportedDigestFromHeader(tc.Header)
			if err != nil {
				if tc.ExpectedErrCode != "" {
					diffErrorCode(t, err, tc.ExpectedErrCode)
					return
				}
				t.Fatal(err)
			} else if tc.ExpectedErrCode != "" {
				t.Fatal("Expected an err")
			}
			digestEncoded := base64.StdEncoding.EncodeToString(actualDigest)
			sigtest.Diff(t, tc.ExcepctedAlgo, actualAlgo, "Wrong digest algo")
			sigtest.Diff(t, tc.ExpectedDigest, digestEncoded, "Wrong digest")
		})
	}
}

func diffErrorCode(t *testing.T, err error, code ErrCode) bool {
	var sigerr *SignatureError
	if errors.As(err, &sigerr) {
		return sigtest.Diff(t, code, sigerr.Code, "Wrong error code")
	}
	return false
}
