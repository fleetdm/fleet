package pubsub

import (
	"sync"
)

type SafeHostHostMap struct {
	HostMap map[uint]chan struct{}
	sync.Mutex
}

func NewSafeHostHostMap() *SafeHostHostMap {
	return &SafeHostHostMap{
		HostMap: make(map[uint]chan struct{}),
	}
}

func (hm *SafeHostHostMap) AddHost(hostID uint, ch chan struct{}) {
	hm.Lock()
	defer hm.Unlock()
	hm.HostMap[hostID] = ch
}

func (hm *SafeHostHostMap) SendMessageToHost(hostID uint) bool {
	hm.Lock()

	ch, ok := hm.HostMap[hostID]

	hm.Unlock()

	if !ok {
		return false
	}
	ch <- struct{}{}
	return true
}

func (hm *SafeHostHostMap) Len() int {
	hm.Lock()
	defer hm.Unlock()
	return len(hm.HostMap)
}

func (shm *SafeHostHostMap) BroadcastSignalToAllHosts() {
	shm.Lock()
	defer shm.Unlock()

	for _, ch := range shm.HostMap {
		ch <- struct{}{}
	}
}
