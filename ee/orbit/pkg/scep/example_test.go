//go:build linux

package scep_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/scep"
	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/tee"
	"github.com/rs/zerolog"
)

const (
	scepURL       = "https://example.com/scep"
	scepChallenge = "challenge"
	commonName    = "fleet-device"
)

// ExampleClient_FetchAndSaveCert demonstrates how to use the SCEP client
// to obtain a certificate from an external SCEP server using a challenge password
// with actual TPM hardware for secure key operations.
//
// This example shows the complete workflow:
// 1. Initialize TPM 2.0 device for hardware-based cryptography
// 2. Configure the SCEP client with server URL, challenge password, and other options
// 3. Fetch and save the certificate using ECC keys generated in the TPM
//
// Prerequisites:
// - TPM 2.0 hardware available at /dev/tpmrm0 or /dev/tpm0
// - SCEP server URL and challenge password
// - Write permissions to certificate directory
func TestExampleSCEPWithTPM(t *testing.T) {
	// t.Skip("For local development only, with a TPM.")

	// Create a logger for monitoring the enrollment process
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create a directory for storing certificates and TPM key blobs
	//certDir, err := os.MkdirTemp("", "scep_tpm_example")
	//if err != nil {
	//	log.Fatalf("Failed to create temp dir: %v", err)
	//}
	//defer os.RemoveAll(certDir)
	certDir := "./victor"

	// Initialize TPM 2.0 device for hardware-based cryptography
	// The TPM will automatically select the best ECC curve (P-384 preferred, P-256 fallback)
	teeDevice, err := tee.NewTPM2(
		// Enable detailed logging for TPM operations
		tee.WithLogger(logger),

		// Specify paths for storing TPM key blobs
		// These files will contain the encrypted key material that can only be used by this TPM
		tee.WithPublicBlobPath(filepath.Join(certDir, "tpm_public.blob")),
		tee.WithPrivateBlobPath(filepath.Join(certDir, "tpm_private.blob")),
	)
	if err != nil {
		log.Fatalf("Failed to initialize TPM: %v", err)
	}
	defer teeDevice.Close()

	// Configure SCEP client with all required parameters
	client, err := scep.NewClient(
		// Required: TPM device for hardware-based cryptographic operations
		scep.WithTEE(teeDevice),

		// Required: SCEP server URL
		// Replace with your actual SCEP server endpoint
		scep.WithURL(scepURL),

		// Required: Challenge password for SCEP authentication
		// This is typically provided by your SCEP server administrator
		// In production, consider using environment variables: os.Getenv("SCEP_CHALLENGE")
		scep.WithChallenge(scepChallenge),

		// Required: Directory where the certificate will be saved
		scep.WithCertDestDir(certDir),

		// Required: Common name for the certificate
		// This should uniquely identify your device/client
		scep.WithCommonName(commonName),

		// Optional: Logger for debugging and monitoring
		scep.WithLogger(logger),

		// Optional: Timeout for SCEP operations (default: 30 seconds)
		// Increase for slower networks or servers
		scep.WithTimeout(2*time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create SCEP client: %v", err)
	}

	// Create context with timeout for the entire enrollment operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Perform SCEP enrollment
	// This will:
	// 1. Create an ECC key pair in the TPM (P-384 or P-256)
	// 2. Generate a Certificate Signing Request (CSR) using the TPM key
	// 3. Send the CSR to the SCEP server with the challenge password
	// 4. Receive and decrypt the certificate response
	// 5. Save the certificate to the specified directory
	fmt.Println("Starting SCEP enrollment with TPM-based keys...")
	if err := client.FetchAndSaveCert(ctx); err != nil {
		// Handle common error scenarios
		switch {
		case err.Error() == "context deadline exceeded":
			log.Fatalf("SCEP enrollment timed out - check network connectivity and server availability: %v", err)
		case err.Error() == "PKIMessage CSR request failed":
			log.Fatalf("SCEP authentication failed - check challenge password: %v", err)
		case err.Error() == "get CA cert":
			log.Fatalf("Failed to connect to SCEP server - check server URL: %v", err)
		case err.Error() == "initialize TEE":
			log.Fatalf("TPM initialization failed - check TPM hardware availability: %v", err)
		default:
			log.Fatalf("SCEP enrollment failed: %v", err)
		}
	}

	// Verify the certificate was successfully saved
	certPath := filepath.Join(certDir, "fleet_client.crt")
	if _, err := os.Stat(certPath); err != nil {
		log.Fatalf("Certificate file not found after enrollment: %v", err)
	}

	// Verify TPM key blobs were created
	publicBlobPath := filepath.Join(certDir, "tmp_public.blob")
	privateBlobPath := filepath.Join(certDir, "tpm_private.blob")

	if _, err := os.Stat(publicBlobPath); err != nil {
		fmt.Printf("Note: TPM public blob not found at %s (this is normal for transient keys)\n", publicBlobPath)
	}

	if _, err := os.Stat(privateBlobPath); err != nil {
		fmt.Printf("Note: TPM private blob not found at %s (this is normal for transient keys)\n", privateBlobPath)
	}

	fmt.Printf("✅ SCEP enrollment successful!\n")
	fmt.Printf("📄 Certificate saved to: %s\n", certPath)
	fmt.Printf("🔐 Certificate signed with TPM-generated ECC key\n")
	fmt.Printf("🛡️  Private key secured in TPM hardware\n")

	// In production, you would typically:
	// 1. Move the certificate to its final location (e.g., /etc/fleet/certs/)
	// 2. Configure your application to use the certificate
	// 3. Set up certificate renewal before expiration
	// 4. Monitor certificate validity and TPM health

	// Output: ✅ SCEP enrollment successful!
	// 📄 Certificate saved to: /tmp/scep_tpm_example123456/fleet_client.crt
	// 🔐 Certificate signed with TPM-generated ECC key
	// 🛡️  Private key secured in TPM hardware
}
