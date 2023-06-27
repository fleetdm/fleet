package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// pdfMagic is the [file signature][1] (or magic bytes) for PDF
//
// [1]: https://en.wikipedia.org/wiki/List_of_file_signatures
var pdfMagic = []byte{0x25, 0x50, 0x44, 0x46}

// CheckPDF checks if the provided bytes are a PDF file.
func CheckPDF(pdf io.Reader) error {
	buf := make([]byte, len(pdfMagic))
	if _, err := io.ReadFull(pdf, buf); err != nil {
		// ReadFull returns ErrUnexpectedEOF if it can't read enough bytes, or EOF
		// if it cannot read a single byte.
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return ErrInvalidType
		}
		return fmt.Errorf("reading magic bytes: %w", err)
	}
	if !bytes.Equal(buf, pdfMagic) {
		return ErrInvalidType
	}
	return nil
}
