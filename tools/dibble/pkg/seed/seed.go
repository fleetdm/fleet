// Package seed contains the per-entity seeding logic that backs both the
// dibble subcommands and the interactive wizard. Keeping the logic here
// (instead of in the cmd_*.go cobra wrappers) makes it possible for the
// wizard to call the same functions and guarantees the two paths can't drift.
package seed

import (
	"errors"
	"fmt"
)

// MultipartFile is one file part of a multipart upload. The FieldName is the
// form field (e.g. "script", "profile"); the Filename is what the server sees
// when it inspects the upload's name — important because Fleet uses extensions
// like .mobileconfig / .xml to detect MDM profile platforms.
type MultipartFile struct {
	FieldName string
	Filename  string
	Content   []byte
}

// Client is the subset of the dibble HTTP client that seeders need. Defined
// as an interface so tests can swap in a fake.
type Client interface {
	Get(path string, out any) error
	Post(path string, body any, out any) error
	Patch(path string, body any, out any) error
	Delete(path string) error
	PostMultipart(path string, fields map[string]string, files []MultipartFile, out any) error
}

// AlreadyExists is the sentinel returned by Client when the server reports a
// conflict. Seeders treat this as soft success.
type AlreadyExistsError interface {
	error
	IsAlreadyExists() bool
}

// IsAlreadyExists reports whether err is an "already exists"-shaped error.
// Decoupled from the concrete client type via duck typing on Error() text so
// the seed package doesn't import the main package.
func IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	type ae interface{ IsAlreadyExists() bool }
	var x ae
	if errors.As(err, &x) {
		return x.IsAlreadyExists()
	}
	return false
}

// Result is what each seeder reports back so cmd_all and the wizard can
// print a tidy summary.
type Result struct {
	Entity  string
	Created int
	Skipped int // already-exists
	Errors  []error
}

// Summary turns a Result into a one-line printable status.
func (r Result) Summary() string {
	if len(r.Errors) > 0 {
		return fmt.Sprintf("%s: %d created, %d skipped, %d errors", r.Entity, r.Created, r.Skipped, len(r.Errors))
	}
	return fmt.Sprintf("%s: %d created, %d skipped", r.Entity, r.Created, r.Skipped)
}

// Logger is a minimal writer interface so seeders can print progress lines
// without coupling to the main package.
type Logger interface {
	Printf(format string, a ...any)
}
