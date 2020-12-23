// package update contains the types and functions used by the update system.
package update

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// DownloadWithSHA512Hash downloads the contents of the given URL, writing
// results to the provided writer. The size is used as an upper limit on the
// amount of data read. An error is returned if the hash of the data received
// does not match the expected hash.
func DownloadWithSHA512Hash(url string, out io.Writer, size int64, expectedHash []byte) error {
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "make get request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	hash := sha512.New()

	// Limit size of response read to expected size
	limitReader := &io.LimitedReader{
		R: resp.Body,
		N: size + 1,
	}

	// Tee the bytes through the hash function
	teeReader := io.TeeReader(limitReader, hash)

	n, err := io.Copy(out, teeReader)
	if err != nil {
		return errors.Wrap(err, "copy response body")
	}
	// Technically these cases would be caught by the hash, but these errors are
	// hopefully a bit more helpful.
	if n < size {
		return errors.New("response smaller than expected")
	}
	if n > size {
		return errors.New("response larger than expected")
	}

	// Validate the hash matches
	gotHash := hash.Sum(nil)
	if bytes.Compare(gotHash, expectedHash) != 0 {
		return errors.Errorf(
			"hash %s does not match expected %s",
			base64.StdEncoding.EncodeToString(gotHash),
			base64.StdEncoding.EncodeToString(expectedHash),
		)
	}

	return nil
}
