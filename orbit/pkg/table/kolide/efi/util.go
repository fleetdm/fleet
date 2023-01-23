package efi

import (
	"bytes"
	"fmt"
	"log"
	"unicode/utf16"
	"unicode/utf8"
)

// DecodeUTF16 decodes the input as a utf16 string.
// Code from https://github.com/u-root/u-root/blob/master/pkg/uefivars/vars.go
// https://gist.github.com/bradleypeabody/185b1d7ed6c0c2ab6cec
func decodeUTF16(b []byte) (string, error) {
	if len(b)%2 != 0 {
		return "", fmt.Errorf("Must have even length byte slice")
	}

	u16s := make([]uint16, 1)
	ret := &bytes.Buffer{}
	b8buf := make([]byte, 4)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[0] = bytesToU16(b[i : i+2])
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}

	return ret.String(), nil
}

// BytesToU16 converts a []byte of length 2 to a uint16.
func bytesToU16(b []byte) uint16 {
	if len(b) != 2 {
		log.Fatalf("bytesToU16: bad len %d (%x)", len(b), b)
	}
	return uint16(b[0]) + (uint16(b[1]) << 8)
}
