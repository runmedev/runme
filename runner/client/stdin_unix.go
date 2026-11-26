//go:build unix

package client

import (
	"os"
	"syscall"
)

func isNullInput(file *os.File) bool {
	info, err := file.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	if err != nil || null.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return info.Sys().(*syscall.Stat_t).Rdev == null.Sys().(*syscall.Stat_t).Rdev
}
