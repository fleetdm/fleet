package lock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeLocker struct {
	name       string
	owner      string
	expiration time.Duration
}

func (f *fakeLocker) Lock(name string, owner string, expiration time.Duration) (bool, error) {
	f.name = name
	f.owner = owner
	f.expiration = expiration
	return true, nil
}

func (*fakeLocker) Unlock(name string, owner string) error {
	return nil
}

func TestLeaderLock(t *testing.T) {
	f := &fakeLocker{}
	Leader(f, time.Second)
	assert.Equal(t, LockKeyLeader, f.name)
	assert.NotEmpty(t, f.owner)
	assert.Equal(t, time.Second, f.expiration)
}
