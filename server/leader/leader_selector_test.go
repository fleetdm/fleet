package leader

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

type globalMutexLock struct {
	ch     chan struct{}
	locked bool
	m      sync.Mutex
}

func (m *globalMutexLock) Lock(name string, owner string, expiration time.Duration) (bool, error) {
	m.m.Lock()
	defer m.m.Unlock()
	if m.locked {
		return true, nil
	}
	select {
	case m.ch <- struct{}{}:
		m.locked = true
		go func() {
			time.Sleep(expiration)
		}()
		return true, nil
	default:
	}
	return false, errors.New("not locked")
}

func (m *globalMutexLock) Unlock(name string, owner string) error {
	m.m.Lock()
	defer m.m.Unlock()
	if !m.locked {
		return errors.New("unlocking but not locked")
	}
	select {
	case <-m.ch:
		m.locked = false
	}
	return nil
}

func TestOnlyOneIsLeader(t *testing.T) {
	ch := make(chan struct{}, 1)

	ls1 := NewLeaderSelector(&globalMutexLock{ch: ch}, 5*time.Second, 1*time.Second)
	defer ls1.ShutDown()
	ls2 := NewLeaderSelector(&globalMutexLock{ch: ch}, 5*time.Second, 2*time.Second)
	defer ls2.ShutDown()

	tries := 100
	for !ls1.AmILeader() && !ls2.AmILeader() && tries > 0 {
		tries--
		time.Sleep(1 * time.Second)
	}
	require.Greater(t, tries, 0)

	if ls1.AmILeader() {
		assert.False(t, ls2.AmILeader())
		ls1.ShutDown()
		time.Sleep(4 * time.Second)
		assert.True(t, ls2.AmILeader())
		assert.False(t, ls1.AmILeader())
	} else {
		assert.False(t, ls1.AmILeader())
		t.Error(t, "ls2 shouldn't have won")
	}
}
