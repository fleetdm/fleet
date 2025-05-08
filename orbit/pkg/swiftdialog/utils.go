package swiftdialog

import (
	"image"
	"io"
	"time"

	// Packages below are not used explicitly, but are imported for initialization side-effects, which allows
	// image.Decode to understand gif, jpeg, and png formatted images
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

const (
	// DefaultIconSize is the default size of the icon to be used in the dialog. It is used when the
	// image has a square or narrow aspect ratio.
	DefaultIconSize = uint(80)
	// wideIconSize is the size of the icon to be used in the dialog when the image has a wide aspect ratio.
	wideIconSize = uint(200)
	// wideAspectRatio is the aspect ratio used to determine if an image is wide. It is used to determine if the
	wideAspectRatio = 1.8 // 9:5 aspect ratio
)

// GetIconSize returns the size of the icon to be used in the dialog. If the image has a wide aspect ratio, it returns
// WideIconSize, otherwise it returns DefaultIconSize.
// It fetches the image from the given URL and decodes it to get the dimensions.
// The URL is expected to be a valid image URL.
// The function returns an error if the image cannot be fetched or decoded.
//
// NOTE: The caller is responsible for ensuring the URL is trusted (e.g., set by an admin user or a default value).
func GetIconSize(url string) (uint, error) {
	resp, err := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)).Get(url) //nolint:gosec
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return decodeIconSize(resp.Body)
}

func decodeIconSize(b io.Reader) (uint, error) {
	// use image.DecodeConfig to get the dimensions of the image
	ic, _, err := image.DecodeConfig(b)
	if err != nil {
		return 0, err
	}
	// if image has wide aspect ratio, use wideIconSize
	if float64(ic.Width) >= float64(ic.Height)*wideAspectRatio {
		return wideIconSize, nil
	}

	return DefaultIconSize, nil
}
