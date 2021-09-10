//go:build !full
// +build !full

package bindata

import "os"

// This file contains placeholders for the asset functions that would normally
// allow Fleet to retrieve assets and templates. Providing these placeholders
// allows Fleet packages to be included as a library with `go get`.

func Asset(name string) ([]byte, error) {
	panic("Assets may not be used when running Fleet as a library")
}

func AssetDir(name string) ([]string, error) {
	panic("Assets may not be used when running Fleet as a library")
}

func AssetInfo(name string) (os.FileInfo, error) {
	panic("Assets may not be used when running Fleet as a library")
}
