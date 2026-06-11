package runnerv2service

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBufferReadDrainsBufferedDataBeforeEOF(t *testing.T) {
	t.Parallel()

	buf := newBuffer(4)
	n, err := buf.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.NoError(t, buf.Close())

	p := make([]byte, 2)

	n, err = buf.Read(p)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "he", string(p[:n]))

	n, err = buf.Read(p)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "ll", string(p[:n]))

	n, err = buf.Read(p)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "o", string(p[:n]))

	n, err = buf.Read(p)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, 0, n)
}

func TestBufferReadClosedEmptyBufferReturnsEOF(t *testing.T) {
	t.Parallel()

	buf := newBuffer(4)
	require.NoError(t, buf.Close())

	p := make([]byte, 2)
	n, err := buf.Read(p)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, 0, n)
}
