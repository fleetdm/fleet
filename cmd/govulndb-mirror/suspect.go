package main

import (
	"errors"
	"fmt"
)

// suspectError marks a condition that makes the collected database untrustworthy rather than the
// run broken: a failed fetch, an incomplete download, a report shaped in a way the transform
// cannot read without guessing, or a snapshot that shrank against the last published one.
//
// These skip the publish and alert instead of failing the workflow. A bad upstream day should
// not page anyone, and it must never reach Fleet: an artifact missing advisories is read as
// those advisories having been remediated.
type suspectError struct{ msg string }

func (e *suspectError) Error() string { return e.msg }

func suspectf(format string, args ...any) error {
	return &suspectError{msg: fmt.Sprintf(format, args...)}
}

func isSuspect(err error) bool {
	var s *suspectError
	return errors.As(err, &s)
}
