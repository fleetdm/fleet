package execuser

const bufSize = 4096

// TransientWriter keeps the last bufSize bytes written to it.
type TransientWriter struct {
	buf [bufSize]byte
	len int
}

// Write writes p to the buffer. If the buffer is full, it will overwrite the oldest bytes.
func (w *TransientWriter) Write(p []byte) (n int, err error) {
	lenToWrite := len(p)
	switch {
	case lenToWrite >= bufSize:
		copy(w.buf[:], p[lenToWrite-bufSize:])
		w.len = bufSize
	case bufSize-w.len < lenToWrite:
		remainingLen := bufSize - lenToWrite
		copy(w.buf[0:remainingLen], w.buf[w.len-remainingLen:w.len])
		copy(w.buf[remainingLen:], p)
		w.len = bufSize
	default:
		n = copy(w.buf[w.len:], p)
		w.len += n
	}
	return lenToWrite, nil
}

func (w TransientWriter) String() string {
	return string(w.buf[:w.len])
}
