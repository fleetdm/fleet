package keyidentifier

import (
	"bytes"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

const sshcomBegin = "---- BEGIN SSH2 ENCRYPTED PRIVATE KEY ----"
const sshcomMagicNumber = 1064303083 // This is 0x3f6ff9eb

// ParseSshComPrivateKey returns key information from an ssh.com
// private key.
//
// The underlying format was gleaned from various other code. Notably:
//
// Putty
// https://github.com/poderosaproject/poderosa/blob/da6a0512d510fc5f02f065a33727f4bbff19a8cb/Granados/Poderosa/KeyFormat/SSHComPrivateKeyLoader.cs

func ParseSshComPrivateKey(keyBytes []byte) (*KeyInfo, error) {

	if !bytes.HasPrefix(keyBytes, []byte(sshcomBegin)) {
		return nil, errors.New("missing sshcom header")
	}

	// ssh2 looks like a pem, but uses a different start and end block
	// designation. So we re-write them to look like a pem block and
	// then hand it to pem to decode
	keyBytes = bytes.Replace(keyBytes, []byte("---- BEGIN"), []byte("-----BEGIN"), 1)
	keyBytes = bytes.Replace(keyBytes, []byte("---- END"), []byte("-----END"), 1)
	keyBytes = bytes.Replace(keyBytes, []byte("KEY ----"), []byte("KEY-----"), -1)

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("pem could not parse block")
	}

	var sshData struct {
		Magic     uint32
		KeyLength uint32
	}

	blockReader := bytes.NewReader(block.Bytes)

	// TODO: Is this ever Little Endian?
	if err := binary.Read(blockReader, binary.BigEndian, &sshData); err != nil {
		return nil, fmt.Errorf("binary read: %w", err)
	}

	if sshData.Magic != sshcomMagicNumber {
		return nil, errors.New("missing magic number")
	}

	keyType, err := readSizedString(blockReader)
	if err != nil {
		return nil, fmt.Errorf("readstring keyType: %w", err)
	}

	cipherName, err := readSizedString(blockReader)
	if err != nil {
		return nil, fmt.Errorf("cipherName: %w", err)
	}

	ki := &KeyInfo{
		Format: "sshcom",
		Parser: "ParseSshComPrivateKey",
	}

	switch cipherName {
	case "none":
		ki.Encrypted = boolPtr(false)
	case "3des-cbc":
		ki.Encrypted = boolPtr(true)
		ki.Encryption = cipherName
	default:
		return nil, fmt.Errorf("sshcom bad cipher name: %s. Should be none or 3des-cbc", cipherName)
	}

	switch {
	case strings.HasPrefix(keyType, "if-modn{sign{rsa"):
		ki.Type = "ssh-rsa"
	case strings.HasPrefix(keyType, "dl-modp{sign{dsa"):
		ki.Type = "ssh-dss"
	default:
		return nil, fmt.Errorf("Unknown key type: %s", keyType)
	}

	return ki, nil

}
