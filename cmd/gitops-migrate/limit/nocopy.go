package limit

import "sync"

var _ sync.Locker = noCopy{}

// noCopy impmlements the 'sync.Locker' interface. Its purpose is to inform the
// various Go tools that it, and anything it is nested within, should _not_ be
// copied.
//
// As a quick demonstration: if you change any method in 'limit.go' to a value
// receiver, rather than a pointer receiver, and run the following, you'll see
// an error logged about the copy:
//
//	$ go vet -copylocks ./cmd/gitops-migrate/...
type noCopy struct{}

func (noCopy) Lock()   {}
func (noCopy) Unlock() {}
