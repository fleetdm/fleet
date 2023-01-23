package keyidentifier

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

const opensshMagic = "openssh-key-v1\x00"

// ParseOpenSSHPrivateKey returns key information from an openssh
// private key. It is adapted from
// https://github.com/golang/crypto/blob/master/ssh/keys.go
func ParseOpenSSHPrivateKey(keyBytes []byte) (*KeyInfo, error) {

	if !bytes.HasPrefix(keyBytes, []byte(opensshMagic)) {
		return nil, errors.New("missing openssh magic")
	}
	remaining := keyBytes[len([]byte(opensshMagic)):]

	var w struct {
		CipherName   string
		KdfName      string
		KdfOpts      string
		NumKeys      uint32
		PubKey       []byte
		PrivKeyBlock []byte
	}

	if err := ssh.Unmarshal(remaining, &w); err != nil {
		return nil, fmt.Errorf("ssh.Unmarshal: %w", err)
	}

	ki := &KeyInfo{
		Format: "openssh-new",
		Parser: "ParseOpenSSHPrivateKey",
	}

	if w.KdfName != "none" || w.CipherName != "none" {
		ki.Encrypted = boolPtr(true)
		ki.Encryption = fmt.Sprintf("%s-%s", w.CipherName, w.KdfName)
	} else {
		ki.Encrypted = boolPtr(false)
	}

	// If we can parse the public key. extract info
	if pubKey, err := ssh.ParsePublicKey(w.PubKey); err == nil {
		ki.Type = pubKey.Type()
		ki.FingerprintSHA256 = strings.TrimPrefix(ssh.FingerprintSHA256(pubKey), "SHA256:")
		ki.FingerprintMD5 = strings.TrimPrefix(ssh.FingerprintLegacyMD5(pubKey), "MD5:")
		// We ought be able to get the size of the key, but I don't see
		// how it's exposed. The ssh.PublicKey type is very bare.
		// ki.Bits = len(pubKey.Parameters().Y.Bytes()) * 8
	}

	return ki, nil

}
