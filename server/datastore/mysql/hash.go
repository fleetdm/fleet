package mysql

import (
	"bytes"
	"crypto/md5" //nolint:gosec // md5 is used for non-cryptographic change detection and deduplication, not security
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

// These helpers compute the same md5 hashes that were historically produced by
// MySQL's MD5() SQL function. MySQL 9.6/9.7 LTS removed MD5()/SHA1(), so every
// hash that feeds a checksum/token column or a unique index is now computed in
// Go and bound as a query parameter. The values are byte-identical to MySQL's
// MD5() output, so existing stored values keep comparing equal.

// md5ChecksumBytes returns the uppercase hex md5 digest of b, suitable for
// binding into a statement as UNHEX(?). UNHEX is case-insensitive, so the
// uppercase form matches MySQL's lowercase MD5() output once decoded.
func md5ChecksumBytes(b []byte) string {
	rawChecksum := md5.Sum(b) //nolint:gosec
	return strings.ToUpper(hex.EncodeToString(rawChecksum[:]))
}

// md5ChecksumScriptContent returns the uppercase hex md5 digest of s.
func md5ChecksumScriptContent(s string) string {
	return md5ChecksumBytes([]byte(s))
}

// md5Checksum returns the raw 16-byte md5 digest of b, suitable for binding
// directly into a BINARY(16) column.
func md5Checksum(b []byte) []byte {
	rawChecksum := md5.Sum(b) //nolint:gosec
	return rawChecksum[:]
}

// mysqlDatetime6 renders t the way MySQL renders a DATETIME(6) value inside
// CONCAT(): the connection uses loc=UTC, so MySQL stores (and the driver sends)
// the UTC wall-clock; DATETIME(6) rounds the fractional part to microseconds
// (round half up, matching time.Round) and always renders 6 fractional digits.
// A nil time renders as "" to match IFNULL(col, ”).
func mysqlDatetime6(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Round(time.Microsecond).Format("2006-01-02 15:04:05.000000")
}

// md5ChecksumFromJSON computes an md5 checksum of the canonical JSON form of b:
// object keys are sorted and insignificant whitespace is removed, while array
// order is preserved. Numbers are kept as written (json.Number), so a value
// reformatted by a JSON engine (e.g. 1e3 vs 1000) counts as a change.
func md5ChecksumFromJSON(b json.RawMessage) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return md5Checksum(canonical), nil
}
