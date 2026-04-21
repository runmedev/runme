package rbuffer

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

var ErrClosed = errors.New("buffer closed")

// RingBuffer is a bounded, concurrent-safe byte buffer that applies
// backpressure on Write when full instead of overwriting unread data.
//
// Semantics:
//   - Write blocks until all bytes in p have been written, or until
//     Close is called (in which case it returns ErrClosed with the count
//     written so far). It never discards unread data.
//   - Read blocks until at least one byte is available or the buffer is
//     closed and drained (in which case it returns io.EOF).
//   - Close wakes all blocked readers and writers; subsequent Writes
//     return ErrClosed, Reads return any remaining data then io.EOF.
type RingBuffer struct {
	mu     sync.Mutex
	cond   *sync.Cond // signals data-available, space-available, or closed
	buf    []byte
	size   int
	r      int // next position to read
	w      int // next position to write
	isFull bool
	closed *atomic.Bool
}

func NewRingBuffer(size int) *RingBuffer {
	b := &RingBuffer{
		buf:    make([]byte, size),
		size:   size,
		closed: &atomic.Bool{},
	}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *RingBuffer) Close() error {
	if b.closed.CompareAndSwap(false, true) {
		b.mu.Lock()
		b.cond.Broadcast()
		b.mu.Unlock()
	}
	return nil
}

func (b *RingBuffer) Reset() {
	b.mu.Lock()
	b.r = 0
	b.w = 0
	b.isFull = false
	b.cond.Broadcast()
	b.mu.Unlock()
}

// Read blocks until at least one byte is available, the buffer is
// closed and drained, or p is empty.
func (b *RingBuffer) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for {
		if b.w != b.r || b.isFull {
			return b.readLocked(p), nil
		}
		if b.closed.Load() {
			return 0, io.EOF
		}
		b.cond.Wait()
	}
}

// read is the internal non-blocking read used by tests. It returns
// io.EOF when the buffer is empty.
func (b *RingBuffer) read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.w == b.r && !b.isFull {
		return 0, io.EOF
	}
	return b.readLocked(p), nil
}

// readLocked copies up to len(p) bytes into p and advances r. Caller
// must hold b.mu and ensure data is available.
func (b *RingBuffer) readLocked(p []byte) int {
	var n int
	if b.w > b.r {
		n = b.w - b.r
		if n > len(p) {
			n = len(p)
		}
		copy(p, b.buf[b.r:b.r+n])
	} else {
		n = b.size - b.r + b.w
		if n > len(p) {
			n = len(p)
		}
		if b.r+n <= b.size {
			copy(p, b.buf[b.r:b.r+n])
		} else {
			c1 := b.size - b.r
			copy(p, b.buf[b.r:b.size])
			copy(p[c1:], b.buf[0:n-c1])
		}
	}
	b.r = (b.r + n) % b.size
	b.isFull = false
	b.cond.Broadcast()
	return n
}

// Write blocks until all of p has been written, applying backpressure
// when the buffer is full instead of discarding unread data. If Close
// is called while Write is blocked, it returns ErrClosed along with
// the number of bytes successfully written so far.
func (b *RingBuffer) Write(p []byte) (int, error) {
	if b.closed.Load() {
		return 0, ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	total := 0
	for len(p) > 0 {
		if b.closed.Load() {
			return total, ErrClosed
		}

		avail := b.availableLocked()
		if avail == 0 {
			b.cond.Wait()
			continue
		}

		k := len(p)
		if k > avail {
			k = avail
		}

		if b.w+k <= b.size {
			copy(b.buf[b.w:], p[:k])
			b.w += k
			if b.w == b.size {
				b.w = 0
			}
		} else {
			c1 := b.size - b.w
			copy(b.buf[b.w:], p[:c1])
			copy(b.buf[0:], p[c1:k])
			b.w = k - c1
		}
		if b.w == b.r {
			b.isFull = true
		}

		total += k
		p = p[k:]
		b.cond.Broadcast()
	}
	return total, nil
}

// availableLocked returns free space in the buffer. Caller must hold b.mu.
func (b *RingBuffer) availableLocked() int {
	if b.isFull {
		return 0
	}
	if b.w >= b.r {
		return b.size - (b.w - b.r)
	}
	return b.r - b.w
}
