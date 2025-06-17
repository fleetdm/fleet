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
	// Create a logger
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

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

	// The key blobs are automatically saved to files during CreateKey
	// Marshal is still available for compatibility but not needed for LoadKey
	_, err = key.Marshal()
	if err != nil {
		log.Fatalf("Failed to marshal key: %v", err)
	}

	// Later, load the key back from the saved blobs
	loadedKey, err := teeDevice.LoadKey(ctx)
	if err != nil {
		log.Fatalf("Failed to load key: %v", err)
	}
	defer loadedKey.Close()

	fmt.Println("Key successfully saved and loaded")
}

//func ExampleKey_Decrypter() {
//	// Create a logger
//	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
//
//	// Initialize TEE
//	teeDevice, err := tee.NewTPM2(
//		tee.WithLogger(logger),
//		tee.WithPublicBlobPath("/tmp/tpm_public2.blob"),
//		tee.WithPrivateBlobPath("/tmp/tpm_private2.blob"),
//	)
//	if err != nil {
//		log.Fatalf("Failed to initialize TEE: %v", err)
//	}
//	defer teeDevice.Close()
//
//	// Create a key
//	ctx := context.Background()
//	key, err := teeDevice.CreateKey(ctx)
//	if err != nil {
//		log.Fatalf("Failed to create key: %v", err)
//	}
//	defer key.Close()
//
//	// Note: ECC keys don't support decryption operations
//	// The Decrypter interface has been removed since ECC keys don't support decryption
//	fmt.Println("ECC key created successfully (decryption not supported)")
//}
