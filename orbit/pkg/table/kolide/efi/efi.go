// Some simple EFI utilities. These are inspired and adapted from
// https://github.com/u-root/u-root/blob/master/pkg/uefivars
// https://github.com/Foxboron/go-uefi
//
// Also useful references
// https://www.kernel.org/doc/html/latest/filesystems/efivarfs.html
// https://github.com/rhboot/efivar/
// https://github.com/rhboot/efivar/blob/master/src/guids.txt
// https://www.kernel.org/doc/Documentation/ABI/stable/sysfs-firmware-efi-vars
package efi

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

const varDir = "/sys/firmware/efi/efivars"

type EfiVar struct {
	Uuid       string // consider a uuid type?
	Name       string
	Attributes Attributes
	Raw        []byte
}

// Per Linux docs, and uefi specs, there is a 4 byte attribute bitfield
type Attributes uint32

// From the UEFI spec
const (
	EFI_VARIABLE_NON_VOLATILE                          Attributes = 0x00000001
	EFI_VARIABLE_BOOTSERVICE_ACCESS                               = 0x00000002
	EFI_VARIABLE_RUNTIME_ACCESS                                   = 0x00000004
	EFI_VARIABLE_HARDWARE_ERROR_RECORD                            = 0x00000008
	EFI_VARIABLE_AUTHENTICATED_WRITE_ACCESS                       = 0x00000010
	EFI_VARIABLE_TIME_BASED_AUTHENTICATED_WRITE_ACCESS            = 0x00000020
	EFI_VARIABLE_APPEND_WRITE                                     = 0x00000040
	EFI_VARIABLE_ENHANCED_AUTHENTICATED_ACCESS                    = 0x00000080
)

// ReadVar reads a given uuid, name pair from the efivars filesystem
// and returns an EfiVar struct.
func ReadVar(uuid string, name string) (*EfiVar, error) {
	ev := &EfiVar{
		Name: name,
		Uuid: uuid,
	}

	if err := ev.ReadRaw(); err != nil {
		return nil, err
	}
	return ev, nil
}

// ReadRaw loads the raw data from the efivar filesystem.
func (ev *EfiVar) ReadRaw() error {
	filename := filepath.Join(varDir, fmt.Sprintf("%s-%s", ev.Name, ev.Uuid))

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening %s: %w", filename, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("statting file descriptor for %s: %w", filename, err)
	}

	if err := binary.Read(f, binary.LittleEndian, &ev.Attributes); err != nil {
		return fmt.Errorf("reading attributes from %s: %w", filename, err)
	}

	// -4 is for the attribute size
	ev.Raw = make([]byte, stat.Size()-4)

	if err = binary.Read(f, binary.LittleEndian, &ev.Raw); err != nil {
		return fmt.Errorf("reading data from %s: %w", filename, err)
	}

	return nil
}

// AsBool converts the raw data to a boolean value.
func (ev *EfiVar) AsBool() (bool, error) {
	return ev.Raw[0] == 1, nil
}

// AsUTF16 converts the raw data to a utf16 encoded string.
func (ev *EfiVar) AsUTF16() (string, error) {
	return decodeUTF16(ev.Raw)
}
