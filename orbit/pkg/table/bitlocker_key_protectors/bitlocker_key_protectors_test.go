//go:build windows
// +build windows

package bitlocker_key_protectors

import (
	"reflect"
	"testing"

	"github.com/rs/zerolog"
)

func TestTable_parseOutput(t *testing.T) {
	// Sample JSON output from Get-BitLockerVolume | ConvertTo-Json
	jsonOutput := `
[{
    "ComputerName":  "WIN-VM",
    "MountPoint":  "C:",
    "EncryptionMethod":  6,
    "EncryptionMethodFlags":  0,
    "AutoUnlockEnabled":  null,
    "AutoUnlockKeyStored":  false,
    "MetadataVersion":  2,
    "VolumeStatus":  1,
    "ProtectionStatus":  1,
    "LockStatus":  0,
    "EncryptionPercentage":  100,
    "WipePercentage":  0,
    "VolumeType":  0,
    "CapacityGB":  475.77832,
    "KeyProtector":  [
	 {
		 "KeyProtectorId":  "{8A696888-99A7-4FAC-92C7-6CC497EA7BDF}",
		 "AutoUnlockProtector":  null,
		 "KeyProtectorType":  3,
		 "KeyFileName":  "",
		 "RecoveryPassword":  "254793-568007-425557-712855-251119-474815-445082-696212",
		 "KeyCertificateType":  null,
		 "Thumbprint":  ""
	 },
	 {
		 "KeyProtectorId":  "{3B3E091C-ECB9-4B46-9FDD-A9E699EC3C8C}",
		 "AutoUnlockProtector":  null,
		 "KeyProtectorType":  1,
		 "KeyFileName":  "",
		 "RecoveryPassword":  "",
		 "KeyCertificateType":  null,
		 "Thumbprint":  ""
	 }
 ]
}, {
	"ComputerName":  "WIN-VM",
	"MountPoint":  "E:",
	"EncryptionMethod":  0,
	"EncryptionMethodFlags":  0,
	"AutoUnlockEnabled":  null,
	"AutoUnlockKeyStored":  null,
	"MetadataVersion":  0,
	"VolumeStatus":  0,
	"ProtectionStatus":  0,
	"LockStatus":  0,
	"EncryptionPercentage":  0,
	"WipePercentage":  0,
	"VolumeType":  1,
	"CapacityGB":  57.7006836,
	"KeyProtector":  []
}]`
	table := &Table{
		logger: zerolog.New(zerolog.NewTestWriter(t)),
	}

	expected := []map[string]string{
		{
			"drive_letter":       "C:",
			"key_protector_type": "3",
		},
		{
			"drive_letter":       "C:",
			"key_protector_type": "1",
		},
	}

	results, err := table.parseOutput([]byte(jsonOutput))
	if err != nil {
		t.Fatalf("parseOutput() error = %v", err)
	}

	if !reflect.DeepEqual(results, expected) {
		t.Errorf("parseOutput() = %v, want %v", results, expected)
	}
}

func TestTable_parseOutput_InvalidJSON(t *testing.T) {
	table := &Table{
		logger: zerolog.New(zerolog.NewTestWriter(t)),
	}

	_, err := table.parseOutput([]byte(`invalid json`))
	if err == nil {
		t.Error("parseOutput() with invalid JSON should return error")
	}
}

func TestTable_parseOutput_EmptyInput(t *testing.T) {
	table := &Table{
		logger: zerolog.New(zerolog.NewTestWriter(t)),
	}

	results, err := table.parseOutput([]byte(`[]`))
	if err != nil {
		t.Fatalf("parseOutput() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("parseOutput() with empty input should return empty results, got %v", results)
	}
}
