//go:build !unix

package client

import "os"

// The legacy runner does not support allocating a PTY on these platforms.
func isNullInput(_ *os.File) bool { return false }
