//go:build linux

package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/hostidentity"
	"github.com/rs/zerolog"
)

const commonName = "fleet-device"

func main() {
	if os.Geteuid() != 0 {
		log.Panic("Example needs to be run as root")
	}
	if _, err := os.Stat("/dev/tpmrm0"); err != nil {
		log.Panic("Could not read TPM 2.0 device")
	}

	fleetURL := os.Getenv("FLEET_URL")
	scepChallenge := os.Getenv("ENROLL_SECRET")

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	metadataDir, err := os.MkdirTemp("", "scep_tpm_example")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(metadataDir)

	clientCertificate, err := hostidentity.CreateOrLoadClientCertificate(
		context.Background(),
		metadataDir,
		fleetURL+"/api/fleet/orbit/host_identity/scep",
		scepChallenge,
		commonName,
		"",
		true,
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
	fmt.Printf("💁 Certificate common name: %s\n", clientCertificate.C.Subject.CommonName)
	fmt.Printf("🔐 Certificate signed with TPM-generated ECC key\n")
	fmt.Printf("🛡️ Private key secured in TPM hardware\n")
}
