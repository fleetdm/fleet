package update

import "os"

// fileDestination wraps the standard os.File with a Delete method for
// compatibility with the go-tuf Destination interface.
// Adapted from
// https://github.com/theupdateframework/go-tuf/blob/master/cmd/tuf-client/get.go
type fileDestination struct {
	*os.File
}

func (f *fileDestination) Delete() error {
	_ = f.Close()
	return os.Remove(f.Name())
}
