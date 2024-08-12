// Command decrypt-disk-encryption-key decrypts a base64-encoded encrypted key
// using the provided X509 certificate and private key. This is typically used
// to manually decrypt a disk encryption key, e.g. BitLocker on Windows or
// FileVault on macOS. The certificate and private key used are the SCEP files
// for a macOS host and the WSTEP files for a Windows host.
//
// Example usage (running from the root of this repository):
//
//		go run ./tools/mdm/decrypt-disk-encryption-key/main.go -cert path/to/file.crt \
//	   -key path/to/file.key -value-to-decrypt base64-encoded-value
package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/apex/log"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mdm"
)

func main() {
	var (
		certFile       = flag.String("cert", "", "The path to the X509 certificate file (required).")
		keyFile        = flag.String("key", "", "The path to the X509 private key file (required).")
		valueToDecrypt = flag.String("value-to-decrypt", "", "The base64-encoded value to decrypt (required).")
	)
	flag.Parse()

	if *certFile == "" || *keyFile == "" || *valueToDecrypt == "" {
		flag.Usage()
		return
	}

	cfg := config.MDMConfig{
		WindowsWSTEPIdentityCert: *certFile,
		WindowsWSTEPIdentityKey:  *keyFile,
	}
	cert, _, _, err := cfg.MicrosoftWSTEP()
	if err != nil {
		// unwrap the error once to remove "Microsoft WSTEP" from the error
		// message, as we don't know in this tool if the cert is for WSTEP or SCEP
		// (it doesn't matter)
		if uerr := errors.Unwrap(err); uerr != nil {
			err = uerr
		}
		log.Fatalf("Error loading certificate: %v", err)
	}

	decrypted, err := mdm.DecryptBase64CMS(*valueToDecrypt, cert.Leaf, cert.PrivateKey)
	if err != nil {
		log.Fatalf("Error decrypting value: %v", err)
	}
	fmt.Printf("Decrypted value: %s\n", string(decrypted))
}
