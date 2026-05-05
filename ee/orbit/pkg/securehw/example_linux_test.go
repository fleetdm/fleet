//go:build linux

package securehw_test

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/ee/orbit/pkg/securehw"
	"github.com/rs/zerolog"
)

func TestExampleTPM20Linux(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Test needs to be run as root")
	}
	if _, err := os.Stat("/dev/tpmrm0"); err != nil {
		t.Skip("Could not read TPM 2.0 device")
	}

	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	tmpDir := t.TempDir()

	t.Run("CreateKey", func(t *testing.T) {
		teeDevice, err := securehw.New(tmpDir, logger)
		if err != nil {
			log.Fatalf("Failed to initialize SecureHW: %v", err)
		}
		defer teeDevice.Close()

		// Create an ECC key in the SecureHW (automatically selects best curve)
		key, err := teeDevice.CreateKey()
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
		message := []byte("Hello, SecureHW!")
		hash := sha256.Sum256(message)
		signature, err := signer.Sign(rand.Reader, hash[:], crypto.SHA256)
		if err != nil {
			log.Fatalf("Failed to sign: %v", err)
		}

		fmt.Printf("Signature created: %x\n", signature)
	})

	t.Run("LoadKey", func(t *testing.T) {
		teeDevice, err := securehw.New(tmpDir, logger)
		if err != nil {
			log.Fatalf("Failed to initialize SecureHW: %v", err)
		}
		defer teeDevice.Close()

		// Later, load the key back from the saved blobs
		key, err := teeDevice.LoadKey()
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
		message := []byte("Hello, SecureHW!")
		hash := sha256.Sum256(message)
		signature, err := signer.Sign(rand.Reader, hash[:], crypto.SHA256)
		if err != nil {
			log.Fatalf("Failed to sign: %v", err)
		}

		fmt.Printf("Signature created: %x\n", signature)
	})
}
