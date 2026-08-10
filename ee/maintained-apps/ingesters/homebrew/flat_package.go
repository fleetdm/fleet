package homebrew

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const (
	// xarMagic and xarHeaderSize describe the fixed header every flat package
	// starts with: magic, header size (uint16), format version (uint16),
	// compressed and uncompressed table-of-contents lengths (uint64 each), and
	// the checksum algorithm (uint32).
	xarMagic      = "xar!"
	xarHeaderSize = 28

	// maxXarTOCSize and maxPkgDistributionSize bound how much of a remote
	// package we're willing to read. Real values are tens of KB; the caps keep
	// a malformed header from turning a metadata lookup into a huge download.
	maxXarTOCSize          = 8 << 20
	maxPkgDistributionSize = 8 << 20

	// errorBodySize is how much of an unexpected response body we read to
	// include in an error message.
	errorBodySize = 512
)

// pkgRefVersion returns the version a flat macOS installer package records for
// the given package reference id (e.g. "com.cisco.Jabber"), without downloading
// the package.
//
// A .pkg is a xar archive: a fixed header, a compressed table of contents
// listing every member's offset and length, then the members themselves. Its
// Distribution file (the XML install script) carries a <pkg-ref> per bundled
// component, each recording the version that component installs. Reading the
// header, the table of contents, and the Distribution file takes three HTTP
// range requests totaling a few tens of KB, against packages that run to
// hundreds of MB.
//
// This recovers the version osquery will report for casks that Homebrew
// versions by build number rather than by app version.
func (i *brewIngester) pkgRefVersion(ctx context.Context, pkgURL, pkgRefID string) (string, error) {
	header, err := i.rangeGet(ctx, pkgURL, 0, xarHeaderSize)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "read flat package header")
	}
	if string(header[:len(xarMagic)]) != xarMagic {
		return "", ctxerr.Errorf(ctx, "%s is not a flat package: xar magic missing", pkgURL)
	}

	headerSize := int64(binary.BigEndian.Uint16(header[4:6]))
	tocSize := int64(binary.BigEndian.Uint64(header[8:16]))
	if headerSize < xarHeaderSize || tocSize <= 0 || tocSize > maxXarTOCSize {
		return "", ctxerr.Errorf(ctx, "flat package header is implausible: header %d bytes, table of contents %d bytes", headerSize, tocSize)
	}

	compressedTOC, err := i.rangeGet(ctx, pkgURL, headerSize, tocSize)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "read flat package table of contents")
	}
	tocXML, err := inflate(compressedTOC, maxXarTOCSize)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "decompress flat package table of contents")
	}

	var toc xarTOC
	if err := xml.Unmarshal(tocXML, &toc); err != nil {
		return "", ctxerr.Wrap(ctx, err, "parse flat package table of contents")
	}

	distribution := findXarFile(toc.Files, "Distribution")
	if distribution == nil || distribution.Data == nil {
		return "", ctxerr.Errorf(ctx, "flat package %s has no Distribution file", pkgURL)
	}
	if distribution.Data.Length <= 0 || distribution.Data.Length > maxPkgDistributionSize {
		return "", ctxerr.Errorf(ctx, "flat package Distribution file has an implausible length of %d bytes", distribution.Data.Length)
	}

	// Member offsets are relative to the heap, which starts right after the
	// header and the compressed table of contents.
	heapStart := headerSize + tocSize
	member, err := i.rangeGet(ctx, pkgURL, heapStart+distribution.Data.Offset, distribution.Data.Length)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "read flat package Distribution file")
	}

	distributionXML, err := decodeXarMember(member, distribution.Data.Encoding.Style)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "decode flat package Distribution file")
	}

	version, err := pkgRefVersionFromDistribution(distributionXML, pkgRefID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "read version from flat package Distribution file")
	}

	return version, nil
}

// xarTOC and friends model the parts of a flat package's table of contents we
// need: where each member sits in the heap and how it's compressed. Directories
// nest their contents in child <file> elements.
type xarTOC struct {
	Files []xarFile `xml:"toc>file"`
}

type xarFile struct {
	Name     string    `xml:"name"`
	Data     *xarData  `xml:"data"`
	Children []xarFile `xml:"file"`
}

type xarData struct {
	// Offset and Length locate the member in the heap in its stored (encoded)
	// form; Size is its decoded length.
	Offset   int64       `xml:"offset"`
	Length   int64       `xml:"length"`
	Size     int64       `xml:"size"`
	Encoding xarEncoding `xml:"encoding"`
}

type xarEncoding struct {
	Style string `xml:"style,attr"`
}

// findXarFile looks up a member by name, descending into directories.
func findXarFile(files []xarFile, name string) *xarFile {
	for idx := range files {
		if files[idx].Name == name {
			return &files[idx]
		}
		if found := findXarFile(files[idx].Children, name); found != nil {
			return found
		}
	}
	return nil
}

// decodeXarMember undoes the encoding a member is stored with. xar labels zlib
// streams "application/x-gzip"; uncompressed members are "application/octet-stream".
func decodeXarMember(member []byte, encoding string) ([]byte, error) {
	switch encoding {
	case "application/x-gzip":
		return inflate(member, maxPkgDistributionSize)
	case "application/octet-stream", "":
		return member, nil
	default:
		return nil, fmt.Errorf("unsupported encoding %q", encoding)
	}
}

// inflate decompresses a zlib stream, refusing to expand beyond max bytes.
func inflate(compressed []byte, max int64) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// Read one byte past the cap so an oversized stream is an error rather than
	// silently truncated content.
	out, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(out)) > max {
		return nil, fmt.Errorf("decompressed to more than %d bytes", max)
	}
	return out, nil
}

// pkgRefVersionFromDistribution pulls the version out of the <pkg-ref> element
// for pkgRefID. A Distribution lists each component several times — in the
// choices outline, in its <choice>, and once as the definition that points at
// the component package — and only the definition carries a version, so scan
// every pkg-ref at any depth and take the first versioned match.
func pkgRefVersionFromDistribution(distribution []byte, pkgRefID string) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(distribution))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "pkg-ref" {
			continue
		}

		var id, version string
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "id":
				id = attr.Value
			case "version":
				version = attr.Value
			}
		}
		if id == pkgRefID && version != "" {
			return version, nil
		}
	}

	return "", fmt.Errorf("no versioned pkg-ref for %q", pkgRefID)
}

// truncateVersion keeps at most the first segments dot-separated components of
// a version ("15.3.0.311163" with 3 -> "15.3.0"). Flat packages record the full
// build version, while osquery reports CFBundleShortVersionString.
func truncateVersion(version string, segments int) string {
	parts := strings.Split(version, ".")
	if len(parts) <= segments {
		return version
	}
	return strings.Join(parts[:segments], ".")
}

// rangeGet fetches length bytes starting at offset, retrying transient
// failures the same way the brew API calls do.
func (i *brewIngester) rangeGet(ctx context.Context, url string, offset, length int64) ([]byte, error) {
	interval := i.retryInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	maxAttempts := i.retryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	var body []byte
	attempt := 0
	err := retry.Do(func() error {
		attempt++

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "create range request")
		}
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))

		res, err := i.client.Do(req)
		if err != nil {
			// Caller cancellation/deadline is not transient; stop retrying.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			i.logger.WarnContext(ctx, "range request failed, retrying", "url", url, "attempt", attempt, "err", err.Error())
			return &transientErr{ctxerr.Wrap(ctx, err, "execute range request")}
		}
		defer res.Body.Close()

		// Anything but 206 means we didn't get the slice we asked for. Notably a
		// server that ignores Range answers 200 with the whole package, so read
		// only enough of the body to describe the failure.
		if res.StatusCode != http.StatusPartialContent {
			errBody, _ := io.ReadAll(io.LimitReader(res.Body, errorBodySize))
			switch res.StatusCode {
			case http.StatusTooManyRequests,
				http.StatusInternalServerError,
				http.StatusBadGateway,
				http.StatusServiceUnavailable,
				http.StatusGatewayTimeout:
				i.logger.WarnContext(ctx, "range request returned transient error, retrying", "url", url, "attempt", attempt, "status", res.StatusCode)
				return &transientErr{ctxerr.Errorf(ctx, "range request returned status %d: %s", res.StatusCode, truncateBody(errBody))}
			default:
				return ctxerr.Errorf(ctx, "range request returned status %d, want 206: %s", res.StatusCode, truncateBody(errBody))
			}
		}

		body, err = io.ReadAll(io.LimitReader(res.Body, length))
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			i.logger.WarnContext(ctx, "reading range response failed, retrying", "url", url, "attempt", attempt, "err", err.Error())
			return &transientErr{ctxerr.Wrap(ctx, err, "read range response body")}
		}
		if int64(len(body)) != length {
			return &transientErr{ctxerr.Errorf(ctx, "range request returned %d bytes, want %d", len(body), length)}
		}

		return nil
	},
		retry.WithInterval(interval),
		retry.WithBackoffMultiplier(2),
		retry.WithMaxAttempts(maxAttempts),
		retry.WithErrorFilter(func(err error) retry.ErrorOutcome {
			if _, ok := errors.AsType[*transientErr](err); ok {
				return retry.ErrorOutcomeNormalRetry
			}
			return retry.ErrorOutcomeDoNotRetry
		}),
	)
	if err != nil {
		return nil, err
	}

	return body, nil
}
