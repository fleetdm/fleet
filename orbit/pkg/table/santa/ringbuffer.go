//go:build darwin

// ringBuffer is a fixed-size circular buffer for log entries.
package santa

type ringBuffer struct {
	buf   []logEntry
	cap   int
	start int
	size  int
}

func newRingBuffer(n int) *ringBuffer {
	// The backing array grows on demand: most hosts have far fewer events than the
	// cap, and the tables allocate a buffer on every query.
	return &ringBuffer{cap: n}
}

func (r *ringBuffer) Add(e logEntry) {
	if r.cap <= 0 {
		return
	}

	if r.size < r.cap {
		if r.size == len(r.buf) {
			r.grow()
		}
		r.buf[r.size] = e
		r.size++
		return
	}

	// Full: overwrite the oldest entry.
	r.buf[r.start] = e
	r.start = (r.start + 1) % len(r.buf)
}

func (r *ringBuffer) grow() {
	buf := make([]logEntry, min(max(2*len(r.buf), 64), r.cap))
	copy(buf, r.buf)
	r.buf = buf
}

func (r *ringBuffer) Len() int {
	return r.size
}

// Reset empties the buffer so it can be reused, releasing the entries it held.
func (r *ringBuffer) Reset() {
	clear(r.buf)
	r.start = 0
	r.size = 0
}

func (r *ringBuffer) SliceChrono() []logEntry {
	return r.AppendTo(make([]logEntry, 0, r.size))
}

// AppendTo appends the buffer's entries, oldest first, to dst. It lets several
// buffers be assembled into one result without a slice per buffer.
func (r *ringBuffer) AppendTo(dst []logEntry) []logEntry {
	for i := range r.size {
		dst = append(dst, r.buf[(r.start+i)%len(r.buf)])
	}
	return dst
}
