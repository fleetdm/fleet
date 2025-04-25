package swiftdialog

import (
	"image"
	"io"
	"net/http"

	// Packages below are not used explicitly, but are imported for initialization side-effects, which allows
	// image.Decode to understand gif, jpeg, and png formatted images
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const (
	DefaultIconSize = uint(80)
	WideIconSize    = uint(200)
	WideAspectRatio = 1.8 // 9:5 aspect ratio
)

func GetIconSize(url string) (uint, error) {
	resp, err := http.Get(url)
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
	if float64(ic.Width) >= float64(ic.Height)*WideAspectRatio {
		return WideIconSize, nil
	}

	return DefaultIconSize, nil
}
