package fleet

import "io"

// Installer describes an installer in an S3 bucket
type Installer struct {
	EnrollSecret string
	Kind         string
	Desktop      bool
	Content      io.ReadSeeker
}
