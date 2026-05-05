package fleet

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"

	_ "golang.org/x/image/webp"
)

const OrgLogoMaxFileSize = 100 * 1024

type OrgLogoMode string

const (
	OrgLogoModeLight OrgLogoMode = "light"
	OrgLogoModeDark  OrgLogoMode = "dark"
	OrgLogoModeAll   OrgLogoMode = "all"
)

func (m OrgLogoMode) IsValid() bool {
	return m == OrgLogoModeLight || m == OrgLogoModeDark || m == OrgLogoModeAll
}

func (m OrgLogoMode) IsStorable() bool {
	return m == OrgLogoModeLight || m == OrgLogoModeDark
}

func (m OrgLogoMode) Modes() []OrgLogoMode {
	switch m {
	case OrgLogoModeAll:
		return []OrgLogoMode{OrgLogoModeLight, OrgLogoModeDark}
	case OrgLogoModeLight, OrgLogoModeDark:
		return []OrgLogoMode{m}
	}
	return nil
}

type OrgLogoStore interface {
	Put(ctx context.Context, mode OrgLogoMode, content io.ReadSeeker) error
	Get(ctx context.Context, mode OrgLogoMode) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, mode OrgLogoMode) error
	Exists(ctx context.Context, mode OrgLogoMode) (bool, error)
}

// Magic-byte signatures used to identify accepted image formats. We compare
// against raw upload bytes rather than trusting the multipart Content-Type
// header.
var (
	orgLogoPNGMagic  = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	orgLogoJPEGMagic = []byte{0xFF, 0xD8, 0xFF}
)

// hasWebPMagic reports whether b begins with a WebP RIFF container header
// ("RIFF" at bytes 0-3, "WEBP" at bytes 8-11). WebP isn't a simple prefix
// check because the 4 bytes between the two markers carry the file size.
func hasWebPMagic(b []byte) bool {
	return len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP"))
}

// ContentTypeForOrgLogo returns the HTTP Content-Type for the accepted org
// logo formats (PNG, JPEG, WebP) based on the leading bytes, or "" for
// anything else.
func ContentTypeForOrgLogo(b []byte) string {
	switch {
	case bytes.HasPrefix(b, orgLogoPNGMagic):
		return "image/png"
	case bytes.HasPrefix(b, orgLogoJPEGMagic):
		return "image/jpeg"
	case hasWebPMagic(b):
		return "image/webp"
	}
	return ""
}

// ValidateOrgLogoBytes is the canonical org-logo validator. The HTTP upload
// handler uses it to gate uploads, and gitops apply uses it as a pre-flight
// before sending bytes to the server, so a YAML referencing an invalid
// image fails fast at apply time rather than mid-PATCH.
func ValidateOrgLogoBytes(b []byte) error {
	if int64(len(b)) > OrgLogoMaxFileSize {
		return &BadRequestError{Message: "logo must be 100KB or less"}
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return &BadRequestError{
			Message:     "logo must be a valid PNG, JPEG, or WebP image",
			InternalErr: err,
		}
	}
	switch format {
	case "png", "jpeg", "webp":
		return nil
	}
	return &BadRequestError{Message: "logo must be a PNG, JPEG, or WebP file"}
}
