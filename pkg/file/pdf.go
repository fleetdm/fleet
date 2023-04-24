package file

import (
	"bytes"
	"fmt"
	"io"
)

// pdfMagic is the [file signature][1] (or magic bytes) for PDF
//
// [1]: https://en.wikipedia.org/wiki/List_of_file_signatures
var pdfMagic = []byte{0x25, 0x50, 0x44, 0x46}

// CheckPDF checks if the provided bytes are a PDF file.
func CheckPDF(pdf io.Reader) error {
	var s bytes.Buffer
	if _, err := io.CopyN(&s, pdf, int64(len(pdfMagic))); err != nil {
		return fmt.Errorf("copying bytes to read magic: %w", err)
	}
	sb := s.Bytes()

	for i := range pdfMagic {
		if sb[i] != pdfMagic[i] {
			return ErrInvalidType
		}
	}

	return nil
}
