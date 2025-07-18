package httpsig

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/remitly-oss/httpsig-go/sigtest"
)

// testcaseSigBase is a test case for signature bases
type testcaseSigBase struct {
	Name         string
	Params       sigBaseInput
	IsResponse   bool
	SourceFile   string // defaults to the specification request or response file
	ExpectedFile string
	ExpectedErr  ErrCode
}

func TestSignatureBase(t *testing.T) {
	cases := []testcaseSigBase{
		{
			Name: "RepeatedComponents",
			Params: sigBaseInput{
				Components:     makeComponents("one", "two", "one", "three"),
				MetadataParams: []Metadata{},
				MetadataValues: emptyMeta,
			},
			SourceFile:  "request_repeated_components.txt",
			ExpectedErr: ErrInvalidSignatureOptions,
		},
		{
			Name: "BadComponentName",
			Params: sigBaseInput{
				Components:     makeComponents("\xd3", "two", "one", "three"),
				MetadataParams: []Metadata{},
				MetadataValues: emptyMeta,
			},
			ExpectedErr: ErrInvalidComponent,
		},
		{
			Name: "NoMultiValueSuport",
			Params: sigBaseInput{
				Components:     makeComponents("one"),
				MetadataParams: []Metadata{},
				MetadataValues: emptyMeta,
			},
			ExpectedErr: ErrUnsupported,
			SourceFile:  "request_multivalue.txt",
		},
		{
			Name: "BadMeta-Created",
			Params: sigBaseInput{
				Components: makeComponents(),
				MetadataParams: []Metadata{
					MetaCreated,
				},
				MetadataValues: errorMetadataProvider{},
			},
			ExpectedErr: ErrInvalidMetadata,
		},
		{
			Name: "BadMeta-Expires",
			Params: sigBaseInput{
				Components: makeComponents(),
				MetadataParams: []Metadata{
					MetaExpires,
				},
				MetadataValues: errorMetadataProvider{},
			},
			ExpectedErr: ErrInvalidMetadata,
		},
		{
			Name: "BadMeta-Nonce",
			Params: sigBaseInput{
				Components: makeComponents(),
				MetadataParams: []Metadata{
					MetaNonce,
				},
				MetadataValues: errorMetadataProvider{},
			},
			ExpectedErr: ErrInvalidMetadata,
		},
		{
			Name: "BadMeta-Algorithm",
			Params: sigBaseInput{
				Components: makeComponents(),
				MetadataParams: []Metadata{
					MetaAlgorithm,
				},
				MetadataValues: errorMetadataProvider{},
			},
			ExpectedErr: ErrInvalidMetadata,
		},
		{
			Name: "BadMeta-KeyID",
			Params: sigBaseInput{
				Components: makeComponents(),
				MetadataParams: []Metadata{
					MetaKeyID,
				},
				MetadataValues: errorMetadataProvider{},
			},
			ExpectedErr: ErrInvalidMetadata,
		},
		{
			Name: "BadMeta-Tag",
			Params: sigBaseInput{
				Components: makeComponents(),
				MetadataParams: []Metadata{
					MetaTag,
				},
				MetadataValues: errorMetadataProvider{},
			},
			ExpectedErr: ErrInvalidMetadata,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			runTestSigBase(t, tc)
		})
	}

}

func runTestSigBase(t *testing.T, tc testcaseSigBase) {
	sourceFile := tc.SourceFile
	hrr := httpMessage{
		IsResponse: tc.IsResponse,
	}
	if tc.IsResponse {
		if sourceFile == "" {
			sourceFile = "rfc-test-response.txt"
		}
		resptxt, err := os.Open(fmt.Sprintf("testdata/%s", sourceFile))
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.ReadResponse(bufio.NewReader(resptxt), nil)
		if err != nil {
			t.Fatal(err)
		}
		hrr.Resp = resp
	} else {
		if sourceFile == "" {
			sourceFile = "rfc-test-request.txt"
		}
		// request
		reqtxt, err := os.Open(fmt.Sprintf("testdata/%s", sourceFile))
		if err != nil {
			t.Fatal(err)
		}

		req, err := http.ReadRequest(bufio.NewReader(reqtxt))
		if err != nil {
			t.Fatal(err)
		}
		hrr.Req = req
	}

	actualBase, err := calculateSignatureBase(hrr, tc.Params)
	if sigtest.Diff(t, tc.ExpectedErr, errCode(err), "Wrong error code") {
		return
	} else if tc.ExpectedErr != "" {
		// If an error is expected and the err Diff check has passed then don't continue on to test the result
		return
	}

	t.Log(string(actualBase.base))
	expectedBase, err := os.ReadFile(fmt.Sprintf("testdata/%s", tc.ExpectedFile))
	if err != nil {
		t.Fatal(err)
	}
	if sigtest.Diff(t, string(expectedBase), string(actualBase.base), "Signature base did not match") {
		t.FailNow()
	}
}

type errorMetadataProvider struct{}

func (fmp errorMetadataProvider) Created() (int, error) {
	return 0, fmt.Errorf("No created value")
}

func (fmp errorMetadataProvider) Expires() (int, error) {
	return 0, fmt.Errorf("No expires value")
}

func (fmp errorMetadataProvider) Nonce() (string, error) {
	return "", fmt.Errorf("No nonce value")
}

func (fmp errorMetadataProvider) Alg() (string, error) {
	return "", fmt.Errorf("No alg value")
}

func (fmp errorMetadataProvider) KeyID() (string, error) {
	return "", fmt.Errorf("No keyid value")
}

func (fmp errorMetadataProvider) Tag() (string, error) {
	return "", fmt.Errorf("No tag value")
}

var emptyMeta = fixedMetadataProvider{
	values: map[Metadata]any{},
}

type fixedMetadataProvider struct {
	values map[Metadata]any
}

func (fmp fixedMetadataProvider) Created() (int, error) {
	if val, ok := fmp.values[MetaCreated]; ok {
		return int(val.(int64)), nil
	}
	return 0, fmt.Errorf("No created value")
}

func (fmp fixedMetadataProvider) Expires() (int, error) {
	if val, ok := fmp.values[MetaExpires]; ok {
		return int(val.(int64)), nil
	}
	return 0, fmt.Errorf("No expires value")
}

func (fmp fixedMetadataProvider) Nonce() (string, error) {
	if val, ok := fmp.values[MetaNonce]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No nonce value")
}

func (fmp fixedMetadataProvider) Alg() (string, error) {
	if val, ok := fmp.values[MetaAlgorithm]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No alg value")
}

func (fmp fixedMetadataProvider) KeyID() (string, error) {
	if val, ok := fmp.values[MetaKeyID]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No keyid value")
}

func (fmp fixedMetadataProvider) Tag() (string, error) {
	if val, ok := fmp.values[MetaTag]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("No tag value")
}

func makeComponents(ids ...string) []componentID {
	cids := []componentID{}
	for _, id := range ids {
		cids = append(cids, SignedField{
			Name: id,
		}.componentID())
	}
	return cids
}
func makeComponentIDs(sfs ...SignedField) []componentID {

	cids := []componentID{}
	for _, sf := range sfs {
		cids = append(cids, sf.componentID())
	}
	return cids
}
