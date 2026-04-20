package runnerv2service

// tailCapWriter is a non-blocking io.Writer that retains at most cap
// bytes, keeping the tail (most recent writes). It's used to capture a
// bounded suffix of a command's stdout for env-var storage without
// applying backpressure to the producer.
//
// Not safe for concurrent use. Callers must establish a happens-before
// relationship (e.g. closing the producer) before reading Bytes.
type tailCapWriter struct {
	buf []byte
	cap int
}

func (w *tailCapWriter) Write(p []byte) (int, error) {
	n := len(p)
	if w.cap <= 0 {
		return n, nil
	}
	if n >= w.cap {
		w.buf = append(w.buf[:0], p[n-w.cap:]...)
		return n, nil
	}
	w.buf = append(w.buf, p...)
	if len(w.buf) > w.cap {
		w.buf = w.buf[len(w.buf)-w.cap:]
	}
	return n, nil
}

func (w *tailCapWriter) Bytes() []byte {
	return w.buf
}
