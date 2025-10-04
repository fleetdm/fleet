//go:build darwin

package santa

type ringBuffer struct {
	buf   []logEntry
	start int
	size  int
}

func newRingBuffer(n int) *ringBuffer {
	return &ringBuffer{buf: make([]logEntry, n)}
}

func (r *ringBuffer) Add(e logEntry) {
	if len(r.buf) == 0 {
		return
	}
	if r.size < len(r.buf) {
		r.buf[(r.start+r.size)%len(r.buf)] = e
		r.size++
	} else {
		r.buf[r.start] = e
		r.start = (r.start + 1) % len(r.buf)
	}
}

func (r *ringBuffer) Len() int {
	return r.size
}

func (r *ringBuffer) SliceChrono() []logEntry {
	out := make([]logEntry, r.size)
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(r.start+i)%len(r.buf)]
	}
	return out
}

// SliceReverse returns entries newest → oldest.
func (r *ringBuffer) SliceReverse() []logEntry {
	out := make([]logEntry, r.size)
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(r.start+r.size-1-i+len(r.buf))%len(r.buf)]
	}
	return out
}
