package keyidentifier

import (
	"bytes"
	"errors"
	"strings"
)

const ppkBegin = "PuTTY-User-Key-File-2"

// ParseSshComPrivateKey returns key information from a putty (ppk)
// formatted key file.
func ParsePuttyPrivateKey(keyBytes []byte) (*KeyInfo, error) {
	if !bytes.HasPrefix(keyBytes, []byte(ppkBegin)) {
		return nil, errors.New("missing ppk begin")
	}

	ki := &KeyInfo{
		Format: "putty",
		Parser: "ParsePuttyPrivateKey",
	}

	keyString := string(keyBytes)
	keyString = strings.Replace(keyString, "\r\n", "\n", -1)

	for _, line := range strings.Split(keyString, "\n") {
		components := strings.SplitN(line, ": ", 2)
		if len(components) != 2 {
			continue
		}
		switch components[0] {
		case ppkBegin:
			ki.Type = components[1]
		case "Encryption":
			if components[1] == "none" {
				ki.Encrypted = boolPtr(false)
			} else {
				ki.Encrypted = boolPtr(true)
				ki.Encryption = components[1]
			}
		case "Comment":
			ki.Comment = components[1]
		}
	}

	return ki, nil

}
