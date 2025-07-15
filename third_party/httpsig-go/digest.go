package httpsig

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"
	"net/http"

	sfv "github.com/dunglas/httpsfv"
)

const (
	digestAlgoSHA256 = "sha-256"
	digestAlgoSHA512 = "sha-512"
)

var (
	emptySHA256 = sha256.Sum256([]byte{})
	emptySHA512 = sha512.Sum512([]byte{})
)

// digestBody reads the entire body to calculate the digest and returns a new io.ReaderCloser which can be set as the new request body.
type digestInfo struct {
	Digest  []byte
	NewBody io.ReadCloser // NewBody is intended as the http.Request Body replacement. Calculating the digest requires reading the body.
}

func digestBody(digAlgo Digest, body io.ReadCloser) (digestInfo, error) {
	var digest []byte
	// client GET requests have a nil body
	// received/server GET requests have a body but its NoBody
	if body == nil || body == http.NoBody {
		switch digAlgo {
		case DigestSHA256:
			digest = emptySHA256[:]
		case DigestSHA512:
			digest = emptySHA512[:]
		default:
			return digestInfo{}, newError(ErrNoSigUnsupportedDigest, fmt.Sprintf("Unsupported digest algorithm '%s'", digAlgo))
		}
		return digestInfo{
			Digest:  digest,
			NewBody: body,
		}, nil
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(body); err != nil {
		return digestInfo{}, newError(ErrNoSigMessageBody, "Failed to read message body to calculate digest", err)
	}
	if err := body.Close(); err != nil {
		return digestInfo{}, newError(ErrNoSigMessageBody, "Failed to close message body to calculate digest", err)
	}

	switch digAlgo {
	case DigestSHA256:
		d := sha256.Sum256(buf.Bytes())
		digest = d[:]
	case DigestSHA512:
		d := sha512.Sum512(buf.Bytes())
		digest = d[:]
	default:
		return digestInfo{}, newError(ErrNoSigUnsupportedDigest, fmt.Sprintf("Unsupported digest algorithm '%s'", digAlgo))
	}

	return digestInfo{
		Digest:  digest,
		NewBody: io.NopCloser(bytes.NewReader(buf.Bytes())),
	}, nil
}

func createDigestHeader(algo Digest, digest []byte) (string, error) {
	sfValue := sfv.NewItem(digest)
	header := sfv.NewDictionary()
	switch algo {
	case DigestSHA256:
		header.Add(digestAlgoSHA256, sfValue)
	case DigestSHA512:
		header.Add(digestAlgoSHA512, sfValue)
	default:
		return "", newError(ErrNoSigUnsupportedDigest, fmt.Sprintf("Unsupported digest algorithm '%s'", algo))
	}
	value, err := sfv.Marshal(header)
	if err != nil {
		return "", newError(ErrInternal, "Failed to marshal digest", err)
	}
	return value, nil

}

// getSupportedDigestFromHeader returns the first supported digest from the supplied header. If no supported header is found a nil digest is returned.
func getSupportedDigestFromHeader(contentDigestHeader []string) (algo Digest, digest []byte, err error) {
	digestDict, err := sfv.UnmarshalDictionary(contentDigestHeader)
	if err != nil {
		return "", nil, newError(ErrNoSigInvalidHeader, "Could not parse Content-Digest header", err)
	}

	for _, algo := range digestDict.Names() {
		switch Digest(algo) {
		case DigestSHA256:
			fallthrough
		case DigestSHA512:
			member, ok := digestDict.Get(algo)
			if !ok {
				continue
			}
			item, ok := member.(sfv.Item)
			if !ok {
				// If not a an Item it's not a valid header value. Skip
				continue
			}
			if digest, ok := item.Value.([]byte); ok {
				return Digest(algo), digest, nil
			}
		default:
			// Unsupported
			continue
		}
	}

	return "", nil, nil
}
