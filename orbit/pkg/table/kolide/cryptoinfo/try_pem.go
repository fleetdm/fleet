package cryptoinfo

import (
	"encoding/pem"
	"fmt"
)

func tryPem(pemBytes []byte, _password string) ([]*KeyInfo, error) {
	expanded := []*KeyInfo{}

	// Loop over the bytes, reading pem blocks
	var block *pem.Block
	for len(pemBytes) > 0 {
		block, pemBytes = pem.Decode(pemBytes)
		if block == nil {
			// When pem.Decode finds no pem, it returns a nil block, and the input as rest.
			// In that case, we stop parsing, as anything else would land in an infinite loop
			break
		}

		expanded = append(expanded, expandPem(block))
	}

	if len(expanded) == 0 {
		return nil, fmt.Errorf("No pem decoded")
	}

	return expanded, nil
}

func expandPem(block *pem.Block) *KeyInfo {
	switch block.Type {
	case "CERTIFICATE":
		return NewCertificate(kiPEM).SetHeaders(block.Headers).SetData(parseCertificate(block.Bytes))
	}

	return NewError(kiPEM, fmt.Errorf("Unknown block type: %s", block.Type))
}
