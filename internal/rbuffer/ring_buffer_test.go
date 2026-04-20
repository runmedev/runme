package rbuffer

import (
	"bytes"
	"crypto/rand"
	"io"
	mathRand "math/rand"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

// assertStillBlocked asserts that done has not been closed/signaled
// after yielding the scheduler many times. Used to verify a goroutine
// is parked in cond.Wait without relying on wall-clock sleeps. Returns
// if no progress is observed; fails the test if done fires.
func assertStillBlocked(t *testing.T, done <-chan struct{}) {
	t.Helper()
	for range 200 {
		select {
		case <-done:
			t.Fatal("goroutine completed before expected — not blocked as intended")
		default:
		}
		runtime.Gosched()
	}
}

func assertRead(t *testing.T, b *RingBuffer, expected []byte) {
	t.Helper()

	got := make([]byte, len(expected))
	n, err := b.read(got)
	assert.Nil(t, err)
	assert.Equal(t, len(expected), n)
	assert.Equal(t, expected, got)
}

func assertWrite(t *testing.T, b *RingBuffer, data []byte) {
	t.Helper()

	n, err := b.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, len(data), n)
}

func TestRingBuffer(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		data := []byte("hello")
		buf := NewRingBuffer(10)
		assertWrite(t, buf, data)
		assertRead(t, buf, data)

		data = []byte("helloworld")
		assertWrite(t, buf, data)
		assertRead(t, buf, data)

		data = []byte("world")
		assertWrite(t, buf, data)
		assertRead(t, buf, data)

		data = []byte("HELLO123")
		assertWrite(t, buf, data)
		assertRead(t, buf, data)
	})

	t.Run("ExceedingInput", func(t *testing.T) {
		buf := NewRingBuffer(4567) // not a power of 2

		var g errgroup.Group

		g.Go(func() error {
			token := make([]byte, 8<<10)         // 8 KiB
			unwritten := int64((64 << 10) << 10) // 64 MiB

			for unwritten > 0 {
				c := mathRand.Intn(cap(token))
				if c > int(unwritten) {
					c = int(unwritten)
				}

				n, err := rand.Read(token[:c])
				if err != nil {
					return err
				}

				n, err = buf.Write(token[:n])
				if err != nil {
					return err
				}

				unwritten -= int64(n)
			}

			return buf.Close()
		})

		g.Go(func() error {
			_, err := io.Copy(io.Discard, buf)
			return err
		})

		assert.NoError(t, g.Wait())
	})
}

func TestRingBuffer_Parallel(t *testing.T) {
	buf := NewRingBuffer(10 * 1000 * 1000)

	var g errgroup.Group

	for i := 0; i < 100; i++ {
		g.Go(func() error {
			data := make([]byte, 1000)
			if _, err := rand.Read(data); err != nil {
				return err
			}
			_, err := buf.Write(data)
			return err
		})
	}

	for i := 0; i < 100; i++ {
		g.Go(func() error {
			data := make([]byte, 1000)
			_, err := buf.Read(data)
			return err
		})
	}

	assert.NoError(t, g.Wait())
}

func TestRingBuffer_Close(t *testing.T) {
	buf := NewRingBuffer(512)
	assert.NoError(t, buf.Close())
	assert.NoError(t, buf.Close())
	n, err := buf.Write([]byte{1})
	assert.Equal(t, 0, n)
	assert.ErrorIs(t, err, ErrClosed)
	p := make([]byte, 32)
	n, err = buf.Read(p)
	assert.Equal(t, 0, n)
	assert.Error(t, err)
}

// TestRingBuffer_BlocksOnFullInsteadOfTrimming asserts that when the
// buffer is full, Write blocks until a Reader drains space — instead of
// silently overwriting unread bytes as the prior implementation did.
// This is the regression test for the PTY byte-loss bug where the shell
// produced bytes faster than the runner consumed them and the ring
// buffer silently dropped unread data.
func TestRingBuffer_BlocksOnFullInsteadOfTrimming(t *testing.T) {
	buf := NewRingBuffer(16)

	// Fill the buffer completely.
	first := bytes.Repeat([]byte{0xAA}, 16)
	assertWrite(t, buf, first)

	// Kick off a write that must block: there is zero space available.
	done := make(chan struct{})
	var writeErr atomic.Pointer[error]
	var writeN atomic.Int64
	go func() {
		n, err := buf.Write([]byte{0xBB})
		writeN.Store(int64(n))
		if err != nil {
			writeErr.Store(&err)
		}
		close(done)
	}()

	// Confirm the goroutine parks in Write without completing.
	assertStillBlocked(t, done)

	// Drain one byte. That should unblock the pending Write.
	got := make([]byte, 1)
	n, err := buf.Read(got)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, byte(0xAA), got[0])

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Write still blocked after Reader drained space")
	}

	assert.Equal(t, int64(1), writeN.Load())
	assert.Nil(t, writeErr.Load(), "Write returned an unexpected error")

	// Drain the rest and assert the original 16 bytes are intact,
	// followed by the byte that was blocked.
	rest := make([]byte, 16)
	total := 0
	for total < 16 {
		n, err := buf.Read(rest[total:])
		assert.NoError(t, err)
		total += n
	}
	assert.Equal(t, append(first[1:], 0xBB), rest)
}

// TestRingBuffer_CloseUnblocksWriter asserts that Close wakes up a
// writer blocked on a full buffer and returns ErrClosed with the byte
// count written so far.
func TestRingBuffer_CloseUnblocksWriter(t *testing.T) {
	buf := NewRingBuffer(4)

	// Fill the buffer.
	assertWrite(t, buf, []byte{1, 2, 3, 4})

	// Start a writer that will block on the full buffer.
	type result struct {
		n   int
		err error
	}
	done := make(chan result, 1)
	blocked := make(chan struct{})
	go func() {
		close(blocked)
		n, err := buf.Write([]byte{5, 6, 7})
		done <- result{n: n, err: err}
	}()

	// Confirm the goroutine parks in Write without completing. We
	// observe via `blocked` that it reached the call site, then verify
	// `done` has not fired across many scheduler yields.
	<-blocked
	for range 200 {
		select {
		case r := <-done:
			t.Fatalf("Write returned before Close: n=%d err=%v", r.n, r.err)
		default:
		}
		runtime.Gosched()
	}

	assert.NoError(t, buf.Close())

	select {
	case r := <-done:
		assert.ErrorIs(t, r.err, ErrClosed)
		assert.Equal(t, 0, r.n, "no bytes should have been written while the buffer was full")
	case <-time.After(1 * time.Second):
		t.Fatal("Close did not unblock the pending Writer")
	}
}
