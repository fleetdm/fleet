// Package tokenpki includes helpers and utilities for exchanging certificates
// and parsing token PKCS#7 S/MIME messages from the Apple ABM/ASM/BE portals.
package tokenpki

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"io"
	"net/textproto"

	"github.com/smallstep/pkcs7"
)

// UnwrapSMIME removes the S/MIME-like header wrapper around the raw encrypted
// CMS/PKCS#7 data in the downloaded token ".p7m" file from the ABM/ASM/BE
// portal.
func UnwrapSMIME(smime []byte) ([]byte, error) {
	r := textproto.NewReader(bufio.NewReader(bytes.NewReader(smime)))
	if _, err := r.ReadMIMEHeader(); err != nil {
		return nil, err
	}
	d := base64.NewDecoder(base64.StdEncoding, r.DotReader())
	b := new(bytes.Buffer)
	_, _ = io.Copy(b, d) // writes to bytes.Buffer never fail
	return b.Bytes(), nil
}

// UnwrapTokenJSON removes the S/MIME-like header wrapper around the
// the decrypted JSON tokens from the token header.
func UnwrapTokenJSON(wrapped []byte) ([]byte, error) {
	r := textproto.NewReader(bufio.NewReader(bytes.NewReader(wrapped)))
	if _, err := r.ReadMIMEHeader(); err != nil {
		return nil, err
	}
	tokenJSON := new(bytes.Buffer)
	for {
		line, err := r.ReadLineBytes()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		line = bytes.TrimPrefix(line, []byte("-----BEGIN MESSAGE-----"))
		line = bytes.TrimPrefix(line, []byte("-----END MESSAGE-----"))
		_, err = tokenJSON.Write(line)
		if err != nil {
			return nil, err
		}
	}
	return tokenJSON.Bytes(), nil
}

// DecryptTokenJSON decrypts and decodes the downloaded token ".p7m" file from
// the ABM/ASM/BE portal to return the actual JSON contained within.
func DecryptTokenJSON(tokenBytes []byte, cert *x509.Certificate, key crypto.PrivateKey) ([]byte, error) {
	p7Bytes, err := UnwrapSMIME(tokenBytes)
	if err != nil {
		return nil, err
	}
	p7, err := pkcs7.Parse(p7Bytes)
	if err != nil {
		return nil, err
	}
	decrypted, err := p7.Decrypt(cert, key)
	if err != nil {
		return nil, err
	}
	return UnwrapTokenJSON(decrypted)
}
