package server

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"

	"github.com/runmedev/runme/v3/internal/server"
)

func pidFileNameFromAddr(addr string) string {
	network, addr, err := server.SplitAddress(addr)
	if err != nil || network != "unix" {
		return ""
	}

	return filepath.Join(filepath.Dir(addr), "runme.pid")
}

func createFileWithPID(path string) error {
	return errors.WithStack(
		os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o600),
	)
}
