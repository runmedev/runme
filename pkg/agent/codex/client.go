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
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

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
	mu      sync.Mutex
	enc     *json.Encoder
	dec     *json.Decoder
	nextReq atomic.Int64
}

func NewClient(reader io.Reader, writer io.Writer) *Client {
	c := &Client{
		enc: json.NewEncoder(writer),
		dec: json.NewDecoder(reader),
	}
	c.nextReq.Store(1)
	return c
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

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		resp := &jsonRPCResponse{}
		if err := c.dec.Decode(resp); err != nil {
			return err
		}
		// Ignore unrelated messages while waiting for our response.
		if resp.ID != reqID {
			continue
		}
		if resp.Error != nil {
			return resp.Error
		}
		if result == nil {
			return nil
		}
		if len(resp.Result) == 0 {
			return errors.New("jsonrpc response missing result")
		}
		return json.Unmarshal(resp.Result, result)
	}
}
