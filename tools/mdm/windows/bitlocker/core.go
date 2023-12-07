package main

import (
	"flag"
	"fmt"
)

func BitlockerEncryptionNumericalPassword(encryptionPassword string) error {

	// Connect to the volume
	vol, err := Connect("c:")
	if err != nil {
		return fmt.Errorf("there was an error connecting to the volume - error: %v", err)
	}
	defer vol.Close()

	// Prepare for encryption
	if err := vol.Prepare(VolumeTypeDefault, EncryptionTypeSoftware); err != nil {
		return fmt.Errorf("there was an error preparing the volume for encryption - error: %v", err)
	}

	// Add a recovery protector

	if err := vol.ProtectWithNumericalPassword(encryptionPassword); err != nil {
		return fmt.Errorf("there was an error adding a recovery protector - error: %v", err)
	}

	// Protect with TPM
	if err := vol.ProtectWithTPM(nil); err != nil {
		return fmt.Errorf("there was an error protecting with TPM - error: %v", err)
	}

	// Start encryption
	if err := vol.Encrypt(XtsAES256, EncryptDataOnly); err != nil {
		return fmt.Errorf("there was an error starting encryption - error: %v", err)
	}

	return nil
}

func BitlockerDecryption() error {

	// Connect to the volume
	vol, err := Connect("c:")
	if err != nil {
		return fmt.Errorf("there was an error connecting to the volume - error: %v", err)
	}
	defer vol.Close()

	// Start decryption
	if err := vol.Decrypt(); err != nil {
		return fmt.Errorf("there was an error starting decryption - error: %v", err)
	}

	return nil
}

func GetBitlockerStatus() (*EncryptionStatus, error) {

	// Connect to the volume
	vol, err := Connect("c:")
	if err != nil {
		return nil, fmt.Errorf("there was an error connecting to the volume - error: %v", err)
	}
	defer vol.Close()

	// Get volume status
	status, err := vol.GetBitlockerStatus()
	if err != nil {
		return nil, fmt.Errorf("there was an error starting decryption - error: %v", err)
	}

	return status, nil
}

func main() {

	enableBitlocker := flag.Bool("encrypt", false, "encrypt the drive")
	disableBitlocker := flag.Bool("decrypt", false, "decrypt the drive")
	statusBitlocker := flag.Bool("status", true, "get drive status")

	flag.Parse()

	if *enableBitlocker {
		fmt.Println("About to attempt enabling bitlocker")

		//This needs to be generated with algorithm defined at
		//https://learn.microsoft.com/en-us/windows/win32/secprov/getkeyprotectornumericalpassword-win32-encryptablevolume
		newPassword := "527230-472395-606199-107525-536789-168927-479336-471856"

		err := BitlockerEncryptionNumericalPassword(newPassword)
		if err != nil {
			fmt.Printf("bitlocker encryption error - %v\n", err)
			return
		}

		fmt.Println("Bitlocker encryption started!")

	} else if *disableBitlocker {
		fmt.Println("About to attempt disabling bitlocker")

		err := BitlockerDecryption()
		if err != nil {
			fmt.Printf("bitlocker decryption error - %v\n", err)
			return
		}

		fmt.Println("Bitlocker decryption started!")

	} else if *statusBitlocker {
		fmt.Println("About to get encryption status bitlocker")

		status, err := GetBitlockerStatus()
		if err != nil {
			fmt.Printf("bitlocker decryption error - %v\n", err)
			return
		}

		fmt.Println("Protection status: ", status.ProtectionStatusDesc)
		fmt.Println("Conversion status: ", status.ConversionStatusDesc)
		fmt.Println("Encryption Flags: ", status.EncryptionFlags)
		fmt.Println("Wiping Status description: ", status.WipingStatusDesc)
		fmt.Println("Encryption percentage complete: ", status.EncryptionPercentage)
		fmt.Println("Wiping percentage complete: ", status.WipingPercentage)

		fmt.Println("Bitlocker encryption status gathered!")

	} else {
		fmt.Println("You must specify either -encrypt, -decrypt or -status")
		return
	}
}
