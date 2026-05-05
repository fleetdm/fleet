package fleet

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/url"
	"strings"

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

// looksLikeSVG only decides which validator to run; validateSVG is what
// actually rejects unsafe content.
func looksLikeSVG(b []byte) bool {
	// Editors may prepend a UTF-8 BOM (byte-order mark) or whitespace.
	if bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}) {
		b = b[3:]
	}
	b = bytes.TrimLeft(b, " \t\r\n")
	// 512B keeps routing cheap; the root <svg> is always near the top.
	const window = 512
	if len(b) > window {
		b = b[:window]
	}
	return bytes.Contains(bytes.ToLower(b), []byte("<svg"))
}

// ContentTypeForOrgLogo returns the HTTP Content-Type for the accepted org
// logo formats (PNG, JPEG, WebP, SVG) based on the leading bytes, or "" for
// anything else.
func ContentTypeForOrgLogo(b []byte) string {
	switch {
	case bytes.HasPrefix(b, orgLogoPNGMagic):
		return "image/png"
	case bytes.HasPrefix(b, orgLogoJPEGMagic):
		return "image/jpeg"
	case hasWebPMagic(b):
		return "image/webp"
	case looksLikeSVG(b):
		return "image/svg+xml"
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
	if looksLikeSVG(b) {
		return validateSVG(b)
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return &BadRequestError{
			Message:     "logo must be a valid PNG, JPEG, WebP, or SVG image",
			InternalErr: err,
		}
	}
	switch format {
	case "png", "jpeg", "webp":
		return nil
	}
	return &BadRequestError{Message: "logo must be a PNG, JPEG, WebP, or SVG file"}
}

func isSafeSVGURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme == "" {
		// Bare relative path or fragment is fine; protocol-relative
		// "//host/x" parses to empty Scheme + non-empty Host and a
		// browser resolves it against the page's scheme — reject.
		return u.Host == ""
	}
	s := strings.ToLower(u.Scheme)
	return s == "http" || s == "https"
}

// Elements that can run scripts or load foreign content. <img>-rendered
// SVGs are script-sandboxed, but pasting the URL loads it as a document
// — so reject structurally instead of trusting the renderer. SMIL
// animation tags are blocked because they can mutate href/xlink:href at
// runtime and bypass the static href allowlist.
var disallowedSVGElements = map[string]struct{}{
	"script":           {},
	"foreignobject":    {},
	"iframe":           {},
	"object":           {},
	"embed":            {},
	"set":              {},
	"animate":          {},
	"animatetransform": {},
	"animatemotion":    {},
}

// validateSVG rejects unsafe SVG content. Leaving decoder.Entity nil and
// rejecting DOCTYPE neutralizes XXE (external entities reading local
// files) and billion-laughs (DoS via recursive entity expansion).
func validateSVG(b []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(b))
	decoder.Strict = true

	sawRoot := false
	for {
		tok, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return &BadRequestError{
				Message:     "logo is not valid SVG",
				InternalErr: err,
			}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if !sawRoot {
				if name != "svg" {
					return &BadRequestError{Message: "logo SVG must have <svg> as the root element"}
				}
				sawRoot = true
			}
			if _, bad := disallowedSVGElements[name]; bad {
				return &BadRequestError{Message: fmt.Sprintf("SVG element <%s> is not allowed", name)}
			}
			for _, attr := range t.Attr {
				attrName := strings.ToLower(attr.Name.Local)
				// on* (onclick, onload, …) is SVG's main XSS vector.
				if strings.HasPrefix(attrName, "on") {
					return &BadRequestError{Message: "SVG event-handler attributes are not allowed"}
				}
				// Name.Local matches both href and xlink:href.
				if attrName == "href" || attrName == "src" {
					if !isSafeSVGURL(attr.Value) {
						return &BadRequestError{Message: "SVG href/src must be a fragment, relative path, or http(s):// URL"}
					}
				}
			}
		case xml.Directive:
			// DOCTYPE / ENTITY (XXE, billion-laughs vectors).
			return &BadRequestError{Message: "SVG DTD/DOCTYPE declarations are not allowed"}
		}
	}
	if !sawRoot {
		return &BadRequestError{Message: "logo SVG missing root <svg> element"}
	}
	return nil
}
