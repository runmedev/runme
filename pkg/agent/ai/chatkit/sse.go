package chatkit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// chatKitSSE handles streaming events to the frontend.
type chatKitSSE struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

func newChatKitSSE(w http.ResponseWriter) (*chatKitSSE, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming unsupported by server")
	}

	headers := w.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache, no-store")
	headers.Set("Connection", "keep-alive")

	return &chatKitSSE{
		writer:  w,
		flusher: flusher,
	}, nil
}

func (s *chatKitSSE) send(ctx context.Context, payload ThreadStreamEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := s.writer.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := s.writer.Write(data); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte("\n\n")); err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}
