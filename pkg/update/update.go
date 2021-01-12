// package update contains the types and functions used by the update system.
package update

import (
	"bytes"
	"crypto/sha512"
	"crypto/tls"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fleetdm/orbit/pkg/constant"
	"github.com/pkg/errors"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/trustpinning"
	"github.com/theupdateframework/notary/tuf/data"
)

const (
	binDir     = "bin"
	osqueryDir = "osquery"
	orbitDir   = "orbit"

	notaryDir = "notary"
)

// Updater is responsible for managing update state.
type Updater struct {
	opt       Options
	transport *http.Transport
}

// Options are the options that can be provided when creating an Updater.
type Options struct {
	// RootDirectory is the root directory from which other directories should be referenced.
	RootDirectory string
	// ServerURL is the URL of the update server.
	ServerURL string
	// GUN is the Globally Unique Name to look up with the Notary server.
	GUN string
	// InsecureTransport skips TLS certificate verification in the transport if
	// set to true.
	InsecureTransport bool
}

// New creates a new updater given the provided options. All the necessary
// directories are initialized.
func New(opt Options) (*Updater, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: opt.InsecureTransport,
	}

	updater := &Updater{
		opt:       opt,
		transport: transport,
	}

	err := updater.initializeDirectories()
	if err != nil {
		return nil, err
	}

	return updater, nil
}

// Lookup returns the target metadata for the provided GUN and target name.
func (u *Updater) Lookup(GUN, target string) (*client.Target, error) {
	client, err := client.NewFileCachedRepository(
		u.pathFromRoot(notaryDir),
		data.GUN(GUN),
		u.opt.ServerURL,
		u.transport,
		nil,
		trustpinning.TrustPinConfig{},
	)
	if err != nil {
		return nil, errors.Wrap(err, "make notary client")
	}

	targetWithRole, err := client.GetTargetByName(target)
	if err != nil {
		return nil, errors.Wrap(err, "get target by name")
	}

	return &targetWithRole.Target, nil
}

func (u *Updater) pathFromRoot(parts ...string) string {
	return filepath.Join(append([]string{u.opt.RootDirectory}, parts...)...)
}

func (u *Updater) initializeDirectories() error {
	for _, dir := range []string{
		u.pathFromRoot(binDir),
		u.pathFromRoot(binDir, osqueryDir),
		u.pathFromRoot(binDir, orbitDir),
		u.pathFromRoot(notaryDir),
	} {
		err := os.MkdirAll(dir, constant.DefaultDirMode)
		if err != nil {
			return errors.Wrap(err, "initialize directories")
		}
	}

	return nil
}

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
	if !bytes.Equal(gotHash, expectedHash) {
		return errors.Errorf(
			"hash %s does not match expected %s",
			base64.StdEncoding.EncodeToString(gotHash),
			base64.StdEncoding.EncodeToString(expectedHash),
		)
	}

	return nil
}
