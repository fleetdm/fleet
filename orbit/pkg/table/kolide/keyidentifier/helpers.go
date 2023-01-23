package keyidentifier

import (
	"bytes"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"strings"
)

// boolPtr makes a pointer from a boolean. We use it to fake a ternary unknown/true/false
func boolPtr(b bool) *bool {
	return &b
}

// readSizedString expects an inputs stream in the form of
// <length><data>. It uses length to determine how much of the data to
// read. It returns as string. (This is used by the ssh.com format.)
func readSizedString(r *bytes.Reader) (string, error) {
	strLenBytes := make([]uint8, 4)
	r.Read(strLenBytes)

	strLen := binary.BigEndian.Uint32(strLenBytes)

	if strLen > uint32(r.Len()) {
		return "", fmt.Errorf("requsted %d, but only %d remain", strLen, r.Len())
	}

	str := make([]byte, strLen)
	if _, err := io.ReadFull(r, str); err != nil {
		return "", fmt.Errorf("error reading buffer: %w", err)
	}

	return string(str), nil
}

// pemEncryptedBlock tells whether a private key is encrypted by
// examining its Proc-Type header for a mention of ENCRYPTED.
// according to RFC 1421 Section 4.6.1.1.
func pemEncryptedBlock(block *pem.Block) bool {
	return strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED")
}
