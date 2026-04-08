package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int64         `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

type jsonRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCNotification struct {
	Method string
	Params json.RawMessage
}

type jsonRPCServerRequest struct {
	ID     int64
	Method string
	Params json.RawMessage
}

type jsonRPCServerRequestHandler func(jsonRPCServerRequest) (any, error)

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *jsonRPCError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("jsonrpc error (%d): %s", e.Code, e.Message)
}

// Client is a minimal JSON-RPC 2.0 client over io.Reader/io.Writer.
// It serializes calls to preserve request/response ordering on plain stdio streams.
type Client struct {
	mu                   sync.Mutex
	enc                  *json.Encoder
	dec                  *json.Decoder
	nextReq              atomic.Int64
	serverRequestHandler jsonRPCServerRequestHandler
}

func NewClient(reader io.Reader, writer io.Writer) *Client {
	c := &Client{
		enc: json.NewEncoder(writer),
		dec: json.NewDecoder(reader),
	}
	c.nextReq.Store(1)
	return c
}

func (c *Client) SetServerRequestHandler(handler jsonRPCServerRequestHandler) {
	c.mu.Lock()
	c.serverRequestHandler = handler
	c.mu.Unlock()
}

func (c *Client) Notify(ctx context.Context, method string, params any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enc.Encode(jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
}

func (c *Client) Call(ctx context.Context, method string, params any, result any) error {
	return c.CallUntil(ctx, method, params, result, nil, nil)
}

func (c *Client) CallUntil(
	ctx context.Context,
	method string,
	params any,
	result any,
	onNotification func(jsonRPCNotification) error,
	isDone func() bool,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	reqID := c.nextReq.Add(1)
	if err := c.enc.Encode(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  params,
	}); err != nil {
		return err
	}

	responseSeen := false
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		msg := &jsonRPCMessage{}
		if err := c.dec.Decode(msg); err != nil {
			return err
		}

		if msg.Method != "" && msg.ID == nil {
			if onNotification != nil {
				if err := onNotification(jsonRPCNotification{
					Method: msg.Method,
					Params: msg.Params,
				}); err != nil {
					return err
				}
			}
			if responseSeen && (isDone == nil || isDone()) {
				return nil
			}
			continue
		}

		if msg.Method != "" && msg.ID != nil {
			if err := c.handleServerRequest(*msg.ID, msg.Method, msg.Params); err != nil {
				return err
			}
			if responseSeen && (isDone == nil || isDone()) {
				return nil
			}
			continue
		}

		// Ignore unrelated messages while waiting for our response.
		if msg.ID == nil || *msg.ID != reqID {
			continue
		}
		if msg.Error != nil {
			return msg.Error
		}
		if result == nil {
			responseSeen = true
			if isDone == nil || isDone() {
				return nil
			}
			continue
		}
		if len(msg.Result) == 0 {
			return errors.New("jsonrpc response missing result")
		}
		if err := json.Unmarshal(msg.Result, result); err != nil {
			return err
		}
		responseSeen = true
		if isDone == nil || isDone() {
			return nil
		}
	}
}

func (c *Client) handleServerRequest(id int64, method string, params json.RawMessage) error {
	handler := c.serverRequestHandler
	if handler == nil {
		return c.enc.Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &jsonRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("unsupported server request method %q", method),
			},
		})
	}

	result, err := handler(jsonRPCServerRequest{
		ID:     id,
		Method: method,
		Params: params,
	})
	if err != nil {
		return c.enc.Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &jsonRPCError{
				Code:    -32000,
				Message: err.Error(),
			},
		})
	}
	return c.enc.Encode(jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}
