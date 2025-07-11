//go:build linux

package hostidentity_test

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/hostidentity"
	"github.com/rs/zerolog"
)

const (
	scepURL       = "https://localhost:8080/api/fleet/orbit/scep"
	scepChallenge = "challenge"
	commonName    = "fleet-device"
)

// ExampleCreateOrLoadClientCertificate demonstrates how to use the SCEP client
// to obtain a certificate from an external SCEP server using a challenge password
// with actual TPM hardware for secure key operations.
//
// This example shows the complete workflow:
// 1. Initialize TPM 2.0 device for hardware-based cryptography.
// 2. Configure the SCEP client with server URL, challenge password, and other options.
// 3. Fetch a certificate using SCEP with a private key generated in the TPM.
//
// Prerequisites:
// - TPM 2.0 hardware available at /dev/tpmrm0
// - SCEP server URL and challenge password
func ExampleCreateOrLoadClientCertificate() {
	if os.Geteuid() != 0 {
		log.Panic("Example needs to be run as root")
	}
	if _, err := os.Stat("/dev/tpmrm0"); err != nil {
		log.Panic("Could not read TPM 2.0 device")
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	metadataDir, err := os.MkdirTemp("", "scep_tpm_example")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(metadataDir)

	clientCertificate, err := hostidentity.CreateOrLoadClientCertificate(
		context.Background(),
		metadataDir,
		scepURL,
		scepChallenge,
		commonName,
		logger,
	)
	if err != nil {
		log.Panicf("Failed to create or load client certificate: %v", err)
	}

	// Verify the certificate was successfully saved
	if clientCertificate.C == nil {
		log.Panic("missing certificate")
	}
	if clientCertificate.C.SerialNumber.Cmp(big.NewInt(0)) == 0 {
		log.Panicf("invalid serial number: %v", clientCertificate.C.SerialNumber)
	}

	fmt.Printf("✅ SCEP enrollment successful!\n")
	fmt.Printf("💁‍♂️ Certificate common name: %s\n", clientCertificate.C.Subject.CommonName)
	fmt.Printf("🔐 Certificate signed with TPM-generated ECC key\n")
	fmt.Printf("🛡️ Private key secured in TPM hardware\n")

	// In production, you would typically:
	// 1. Move the certificate to its final location (e.g., /etc/fleet/certs/)
	// 2. Configure your application to use the certificate
	// 3. Set up certificate renewal before expiration
	// 4. Monitor certificate validity and TPM health

	// Output: ✅ SCEP enrollment successful!
	// 💁‍♂️ Certificate common name: fleet-device
	// 🔐 Certificate signed with TPM-generated ECC key
	// 🛡️ Private key secured in TPM hardware
}
