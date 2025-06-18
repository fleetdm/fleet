//go:build linux

package tee_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/tee"
	"github.com/rs/zerolog"
)

func TestExampleTEE(t *testing.T) {
	t.Skip("For local development only, with a TPM.")

	// Create a logger
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	t.Run("CreateKey", func(t *testing.T) {
		// Initialize TEE (TPM 2.0 on Linux)
		teeDevice, err := tee.NewTPM2(
			tee.WithLogger(logger),
			tee.WithPublicBlobPath("./temp/tpm_public.blob"),
			tee.WithPrivateBlobPath("./temp/tpm_private.blob"),
		)
		if err != nil {
			log.Fatalf("Failed to initialize TEE: %v", err)
		}
		defer teeDevice.Close()

		// Create an ECC key in the TEE (automatically selects best curve)
		ctx := context.Background()
		key, err := teeDevice.CreateKey(ctx)
		if err != nil {
			log.Fatalf("Failed to create key: %v", err)
		}
		defer key.Close()

		// Get a signer for the key
		signer, err := key.Signer()
		if err != nil {
			log.Fatalf("Failed to get signer: %v", err)
		}

		// Sign some data
		message := []byte("Hello, TEE!")
		hash := sha256.Sum256(message)
		signature, err := signer.Sign(rand.Reader, hash[:], crypto.SHA256)
		if err != nil {
			log.Fatalf("Failed to sign: %v", err)
		}

		fmt.Printf("Signature created: %x\n", signature)
	})

	t.Run("LoadKey", func(t *testing.T) {
		// Initialize TEE (TPM 2.0 on Linux)
		teeDevice, err := tee.NewTPM2(
			tee.WithLogger(logger),
			tee.WithPublicBlobPath("./temp/tpm_public.blob"),
			tee.WithPrivateBlobPath("./temp/tpm_private.blob"),
		)
		if err != nil {
			log.Fatalf("Failed to initialize TEE: %v", err)
		}
		defer teeDevice.Close()

		// Later, load the key back from the saved blobs
		key, err := teeDevice.LoadKey(context.Background())
		if err != nil {
			log.Fatalf("Failed to load key: %v", err)
		}
		defer key.Close()

		fmt.Println("Key successfully loaded")

		// Get a signer for the key
		signer, err := key.Signer()
		if err != nil {
			log.Fatalf("Failed to get signer: %v", err)
		}

		// Sign some data
		message := []byte("Hello, TEE!")
		hash := sha256.Sum256(message)
		signature, err := signer.Sign(rand.Reader, hash[:], crypto.SHA256)
		if err != nil {
			log.Fatalf("Failed to sign: %v", err)
		}

		fmt.Printf("Signature created: %x\n", signature)

	})

}
