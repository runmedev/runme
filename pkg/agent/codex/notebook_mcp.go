package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	mcpruntime "github.com/redpanda-data/protoc-gen-go-mcp/pkg/runtime"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1/toolsv1mcp"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
)

const (
	defaultBridgeCallTimeout = 30 * time.Second
)

type bridgeCaller interface {
	Call(ctx context.Context, input *toolsv1.ToolCallInput) (*toolsv1.ToolCallOutput, error)
}

type NotebookMCPBridge struct {
	bridge      bridgeCaller
	callTimeout time.Duration
}

type executeCodeArgs struct {
	Code string `json:"code,omitempty"`
}

func NewNotebookMCPBridge(bridge bridgeCaller) *NotebookMCPBridge {
	return &NotebookMCPBridge{
		bridge:      bridge,
		callTimeout: defaultBridgeCallTimeout,
	}
}

func (b *NotebookMCPBridge) SetCallTimeout(timeout time.Duration) {
	if timeout > 0 {
		b.callTimeout = timeout
	}
}

func (b *NotebookMCPBridge) NewServer() *mcpserver.MCPServer {
	server := mcpserver.NewMCPServer(
		"runme-notebooks",
		"0.1.0",
		mcpserver.WithToolCapabilities(true),
	)

	server.AddTool(
		toMCPTool(toolsv1mcp.NotebookService_ExecuteCodeToolOpenAI),
		mcp.NewTypedToolHandler(b.handleExecuteCode),
	)
	return server
}

func toMCPTool(tool mcpruntime.Tool) mcp.Tool {
	return mcp.Tool{
		Name:           tool.Name,
		Description:    tool.Description,
		RawInputSchema: tool.RawInputSchema,
	}
}

func (b *NotebookMCPBridge) handleExecuteCode(ctx context.Context, _ mcp.CallToolRequest, args executeCodeArgs) (*mcp.CallToolResult, error) {
	logger := loggerForToolCall(ctx, "ExecuteCode")
	input := &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ExecuteCode{
			ExecuteCode: &toolsv1.ExecuteCodeRequest{
				Code: args.Code,
			},
		},
	}
	output, err := b.callBridgeForTool(ctx, "ExecuteCode", input)
	if err != nil {
		logger.Error(err, "failed to dispatch tool call")
		return mcp.NewToolResultErrorFromErr("failed to dispatch ExecuteCode over codex bridge", err), nil
	}
	logger.Info("completed tool call", "callID", output.GetCallId(), "status", output.GetStatus().String())
	return toolOutputToResult(output.GetExecuteCode(), output)
}

func (b *NotebookMCPBridge) callBridge(ctx context.Context, input *toolsv1.ToolCallInput) (*toolsv1.ToolCallOutput, error) {
	callCtx := ctx
	if b.callTimeout > 0 {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			callCtx, cancel = context.WithTimeout(ctx, b.callTimeout)
			defer cancel()
		}
	}
	return b.bridge.Call(callCtx, input)
}

func (b *NotebookMCPBridge) callBridgeForTool(ctx context.Context, tool string, input *toolsv1.ToolCallInput) (*toolsv1.ToolCallOutput, error) {
	started := time.Now()
	output, err := b.callBridge(ctx, input)
	observeMCPToolCall(tool, time.Since(started), toolCallOutcome(output, err))
	return output, err
}

func loggerForToolCall(ctx context.Context, tool string) logr.Logger {
	logger := logs.FromContextWithTrace(ctx).WithValues("tool", tool)
	if sid := SessionIDFromContext(ctx); sid != "" {
		logger = logger.WithValues("sessionID", sid)
	}
	if principal := obs.GetPrincipal(ctx); principal != "" {
		logger = logger.WithValues("principal", principal)
	}
	return logger
}

func toolOutputToResult(payload proto.Message, output *toolsv1.ToolCallOutput) (*mcp.CallToolResult, error) {
	if output == nil {
		return mcp.NewToolResultError("tool bridge returned an empty output"), nil
	}
	if output.GetStatus() == toolsv1.ToolCallOutput_STATUS_FAILED {
		return mcp.NewToolResultErrorf("tool call failed: %s", output.GetClientError()), nil
	}
	if output.GetClientError() != "" {
		return mcp.NewToolResultErrorf("tool call client_error: %s", output.GetClientError()), nil
	}
	if payload == nil {
		return mcp.NewToolResultError("tool bridge returned no payload"), nil
	}

	marshal := protojson.MarshalOptions{
		UseProtoNames: true,
	}
	payloadJSON, err := marshal.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal tool payload: %w", err)
	}

	var structured any
	if err := json.Unmarshal(payloadJSON, &structured); err != nil {
		return nil, fmt.Errorf("unmarshal tool payload: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(payloadJSON)),
		},
		StructuredContent: structured,
	}, nil
}
