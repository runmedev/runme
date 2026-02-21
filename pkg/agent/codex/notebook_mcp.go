package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/go-logr/logr"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1/toolsv1mcp"
	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/obs"
)

const (
	defaultBridgeCallTimeout = 30 * time.Second
)

type bridgeCaller interface {
	Call(ctx context.Context, input *toolsv1.ToolCallInput) (*toolsv1.ToolCallOutput, error)
}

type executeApprover interface {
	AllowExecute(ctx context.Context, refIDs []string) error
}

type NotebookMCPBridge struct {
	bridge      bridgeCaller
	approver    executeApprover
	callTimeout time.Duration
}

type listCellsArgs struct{}

type getCellsArgs struct {
	RefIDs []string `json:"ref_ids,omitempty"`
}

type updateCellsArgs struct {
	Cells []*parserv1.Cell `json:"cells,omitempty"`
}

type executeCellsArgs struct {
	RefIDs []string `json:"ref_ids,omitempty"`
}

func NewNotebookMCPBridge(bridge bridgeCaller) *NotebookMCPBridge {
	return &NotebookMCPBridge{
		bridge:      bridge,
		callTimeout: defaultBridgeCallTimeout,
		approver:    denyExecuteApprover{},
	}
}

func (b *NotebookMCPBridge) SetExecuteApprover(approver executeApprover) {
	if approver != nil {
		b.approver = approver
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
		toolsv1mcp.NotebookService_ListCellsToolOpenAI,
		mcp.NewTypedToolHandler(b.handleListCells),
	)
	server.AddTool(
		toolsv1mcp.NotebookService_GetCellsToolOpenAI,
		mcp.NewTypedToolHandler(b.handleGetCells),
	)
	server.AddTool(
		toolsv1mcp.NotebookService_UpdateCellsToolOpenAI,
		mcp.NewTypedToolHandler(b.handleUpdateCells),
	)
	server.AddTool(
		toolsv1mcp.NotebookService_ExecuteCellsToolOpenAI,
		mcp.NewTypedToolHandler(b.handleExecuteCells),
	)
	return server
}

func (b *NotebookMCPBridge) handleListCells(ctx context.Context, _ mcp.CallToolRequest, _ listCellsArgs) (*mcp.CallToolResult, error) {
	logger := loggerForToolCall(ctx, "ListCells")
	input := &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ListCells{
			ListCells: &toolsv1.ListCellsRequest{},
		},
	}
	output, err := b.callBridgeForTool(ctx, "ListCells", input)
	if err != nil {
		logger.Error(err, "failed to dispatch tool call")
		return mcp.NewToolResultErrorFromErr("failed to dispatch ListCells over codex bridge", err), nil
	}
	logger.Info("completed tool call", "callID", output.GetCallId(), "status", output.GetStatus().String())
	return toolOutputToResult(output.GetListCells(), output)
}

func (b *NotebookMCPBridge) handleGetCells(ctx context.Context, _ mcp.CallToolRequest, args getCellsArgs) (*mcp.CallToolResult, error) {
	logger := loggerForToolCall(ctx, "GetCells")
	input := &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_GetCells{
			GetCells: &toolsv1.GetCellsRequest{RefIds: args.RefIDs},
		},
	}
	output, err := b.callBridgeForTool(ctx, "GetCells", input)
	if err != nil {
		logger.Error(err, "failed to dispatch tool call")
		return mcp.NewToolResultErrorFromErr("failed to dispatch GetCells over codex bridge", err), nil
	}
	logger.Info("completed tool call", "callID", output.GetCallId(), "status", output.GetStatus().String())
	return toolOutputToResult(output.GetGetCells(), output)
}

func (b *NotebookMCPBridge) handleUpdateCells(ctx context.Context, _ mcp.CallToolRequest, args updateCellsArgs) (*mcp.CallToolResult, error) {
	logger := loggerForToolCall(ctx, "UpdateCells")
	input := &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_UpdateCells{
			UpdateCells: &toolsv1.UpdateCellsRequest{
				Cells: args.Cells,
			},
		},
	}
	output, err := b.callBridgeForTool(ctx, "UpdateCells", input)
	if err != nil {
		logger.Error(err, "failed to dispatch tool call")
		return mcp.NewToolResultErrorFromErr("failed to dispatch UpdateCells over codex bridge", err), nil
	}
	logger.Info("completed tool call", "callID", output.GetCallId(), "status", output.GetStatus().String())
	return toolOutputToResult(output.GetUpdateCells(), output)
}

func (b *NotebookMCPBridge) handleExecuteCells(ctx context.Context, _ mcp.CallToolRequest, args executeCellsArgs) (*mcp.CallToolResult, error) {
	logger := loggerForToolCall(ctx, "ExecuteCells")
	if err := b.approver.AllowExecute(ctx, args.RefIDs); err != nil {
		logger.Error(err, "tool call denied by execute approval gate")
		observeMCPToolCall("ExecuteCells", 0, toolOutcomeToolErr)
		return mcp.NewToolResultErrorFromErr("ExecuteCells requires explicit user approval", err), nil
	}
	input := &toolsv1.ToolCallInput{
		Input: &toolsv1.ToolCallInput_ExecuteCells{
			ExecuteCells: &toolsv1.NotebookServiceExecuteCellsRequest{
				RefIds: args.RefIDs,
			},
		},
	}
	output, err := b.callBridgeForTool(ctx, "ExecuteCells", input)
	if err != nil {
		logger.Error(err, "failed to dispatch tool call")
		return mcp.NewToolResultErrorFromErr("failed to dispatch ExecuteCells over codex bridge", err), nil
	}
	logger.Info("completed tool call", "callID", output.GetCallId(), "status", output.GetStatus().String())
	return toolOutputToResult(output.GetExecuteCells(), output)
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

type denyExecuteApprover struct{}

func (denyExecuteApprover) AllowExecute(_ context.Context, _ []string) error {
	return errors.New("no execute approval has been granted")
}

type contextExecuteApprover struct{}

func (contextExecuteApprover) AllowExecute(ctx context.Context, refIDs []string) error {
	approved := approvedRefIDsFromContext(ctx)
	if len(approved) == 0 {
		return errors.New("missing execute approvals")
	}
	for _, id := range refIDs {
		if id == "" {
			continue
		}
		if !slices.Contains(approved, id) {
			return fmt.Errorf("cell %q is not approved for execution", id)
		}
	}
	return nil
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
