package log

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type LogEntry struct {
	Msg       string           `json:"msg"`
	ID        string           `json:"_id"`
	RunID     string           `json:"id"`
	KnownID   string           `json:"knownID"`
	KnownName string           `json:"knownName"`
	Req       *json.RawMessage `json:"req"`
}

// readLogMessages reads the log messages
func ReadLogMessages(logger *zap.Logger, logFile string) ([]*LogEntry, error) {
	messages := make([]*LogEntry, 0, 100)
	// Flush the log messages
	if err := logger.Sync(); err != nil {
		ignoreError := false
		// N.B. we get a bad file descriptor error when calling Sync on a logger writing to stderr
		// We can just ignore that.
		if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == syscall.EBADF {
			ignoreError = true
		}
		if strings.Contains(err.Error(), "/dev/stderr") {
			ignoreError = true
		}
		if !ignoreError {
			return messages, errors.Wrapf(err, "failed to sync logger")
		}
	}

	// Read the log messages
	b, err := os.ReadFile(logFile)
	if err != nil {
		return messages, errors.Wrapf(err, "failed to read log file")
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	for {
		var entry LogEntry
		if err := dec.Decode(&entry); err != nil {
			if errors.Is(err, io.EOF) {
				return messages, nil
			}
			return messages, errors.Wrapf(err, "failed to decode log entry")
		}
		messages = append(messages, &entry)
	}
}
