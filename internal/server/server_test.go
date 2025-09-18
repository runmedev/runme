package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitAddress(t *testing.T) {
	tests := []struct {
		name, raw, network, addr string
	}{
		{
			"empty string defaults to tcp",
			"", "tcp", "",
		},
		{
			"no colon defaults to tcp",
			"no-colon", "tcp", "no-colon",
		},
		{
			"bare host:port",
			"localhost:8080", "tcp", "localhost:8080",
		},
		{
			"invalid URLs fall back to tcp",
			"tcp://%zz:1234", "tcp", "tcp://%zz:1234",
		},
		{
			"unknown scheme falls back to tcp",
			"udp://1.2.3.4:53", "tcp", "udp://1.2.3.4:53",
		},
		{
			"tcp without slashes",
			"tcp:127.0.0.1:80", "tcp", "127.0.0.1:80",
		},
		{
			"tcp with slashes",
			"tcp://127.0.0.1:80", "tcp", "127.0.0.1:80",
		},
		{
			"tcp with empty host",
			"tcp://", "tcp", "",
		},
		{
			"unix relative path",
			"unix:relative/path/to/socket", "unix", "relative/path/to/socket",
		},
		{
			"unix absolute path",
			"unix:/tmp/socket", "unix", "/tmp/socket",
		},
		{
			"unix absolute path with non-empty host",
			"unix://localhost/tmp/socket", "unix", "/tmp/socket",
		},
		{
			"unix absolute path with empty host",
			"unix:///tmp/socket", "unix", "/tmp/socket",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			network, addr, err := SplitAddress(test.raw)
			require.NoError(t, err)
			assert.Equal(t, test.network, network)
			assert.Equal(t, test.addr, addr)
		})
	}
}
