//go:build windows
// +build windows

package term

import "os"

type CancelFn func()

func ResizeOnSig(tty *os.File) CancelFn {
	return func() {}
}
