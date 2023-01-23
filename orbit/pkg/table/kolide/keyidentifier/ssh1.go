package keyidentifier

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const ssh1LegacyBegin = "SSH PRIVATE KEY FILE FORMAT 1.1\n"

// ParseSsh1PrivateKey returns key information from an ssh1 private key.
//
// The underlying format was gleaned from various other code. Notably:
//
// https://github.com/openssh/openssh-portable/blob/c7670b091a7174760d619ef6738b4f26b2093301/sshkey.c
// https://github.com/KasperDeng/putty/blob/037a4ccb6e731fafc4cc77c0d16f80552fd69dce/putty-src/sshpubk.c#L176-L180
// https://github.com/chrber/pcells-maven/blob/bb7a1ef3aa5e9313c532c043a624bfb929962b48/modules/pcells-gui-core/src/main/java/dmg/security/cipher/SshPrivateKeyInputStream.java#L23
func ParseSsh1PrivateKey(keyBytes []byte) (*KeyInfo, error) {

	if !bytes.HasPrefix(keyBytes, []byte(ssh1LegacyBegin)) {
		return nil, errors.New("missing ssh1 header")
	}

	keyReader := bytes.NewReader(keyBytes)

	var sshData struct {
		Header     [len(ssh1LegacyBegin)]byte
		Zero       uint8  // null after header
		CipherType uint8  // Enc type (0 is none, 3 is encrypted)
		Reserved   uint32 // 4 bytes reserved
		Bits       uint32 // 4 bytes for the bit size
	}

	// TODO: Is this ever Little Endian!?
	if err := binary.Read(keyReader, binary.BigEndian, &sshData); err != nil {
		return nil, fmt.Errorf("failed binary read: %w", err)
	}

	ki := &KeyInfo{
		Bits:   int(sshData.Bits),
		Format: "ssh1",
		Parser: "ParseSsh1PrivateKey",
		Type:   "rsa1",
	}

	switch sshData.CipherType {
	case 0:
		ki.Encrypted = boolPtr(false)
	case 3:
		ki.Encrypted = boolPtr(true)
	default:
		return nil, fmt.Errorf("ssh1 bad cipher type: %d. Should be 0 or 3", sshData.CipherType)
	}

	return ki, nil

}
