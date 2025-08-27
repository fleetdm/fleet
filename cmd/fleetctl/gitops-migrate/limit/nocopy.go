package limit

import "sync"

var _ sync.Locker = noCopy{}

type noCopy struct{}

func (noCopy) Lock()   {}
func (noCopy) Unlock() {}
