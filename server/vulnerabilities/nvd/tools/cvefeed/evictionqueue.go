// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cvefeed

import (
	"container/heap"
	"time"
)

type evictionData struct {
	key    string    // which key in Cache.data refers to it
	index  int       // the index of item on the heap
	access time.Time // last access time
}

// evictionQueue is a priority queue for LRU cache
type evictionQueue struct {
	q evictionHeap
}

// pop pops next key to evict
func (eq *evictionQueue) pop() string {
	if eq.q.Len() > 0 {
		return heap.Pop(&eq.q).(*evictionData).key
	}
	return ""
}

// push pushes a key onto heap, returns the index item ended up at.
func (eq *evictionQueue) push(key string) int {
	index := eq.q.Len()
	ed := &evictionData{
		key:    key,
		index:  index,
		access: time.Now(),
	}
	heap.Push(&eq.q, ed)
	return ed.index
}

// touch updates the access time of the item at index, returns the new index of that item.
func (eq *evictionQueue) touch(index int) int {
	ed := eq.q[index]
	ed.access = time.Now()
	heap.Fix(&eq.q, index)
	return ed.index
}

// evictionHeap is a slice of evictionData that implements heap.Interface
type evictionHeap []*evictionData

func (eh evictionHeap) Len() int { return len(eh) }

func (eh evictionHeap) Less(i, j int) bool { return eh[i].access.Before(eh[j].access) }

func (eh evictionHeap) Swap(i, j int) {
	eh[i], eh[j] = eh[j], eh[i]
	eh[i].index, eh[j].index = i, j
}

func (eh *evictionHeap) Push(x interface{}) {
	ed := x.(*evictionData)
	ed.index = len(*eh)
	*eh = append(*eh, ed)
}

func (eh *evictionHeap) Pop() interface{} {
	old := *eh
	ed := old[len(old)-1]
	*eh = old[:len(old)-1]
	return ed
}
