package client

import (
	"fmt"
	"path/filepath"
)

// gitOpsValidationError is an error that occurs during validating fields in the yaml spec.
type gitOpsValidationError struct {
	message string
}

func (e *gitOpsValidationError) Error() string {
	return e.message
}

func (e *gitOpsValidationError) WithFileContext(baseDir, filename string) error {
	fileFullPath := filepath.Join(baseDir, filename)
	return fmt.Errorf("Couldn't edit %q at: %q. %s", filename, fileFullPath, e.message)
}

func newGitOpsValidationError(message string) *gitOpsValidationError {
	return &gitOpsValidationError{message: message}
}
