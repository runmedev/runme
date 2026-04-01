package codex

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1/toolsv1mcp"
)

type fakeBridge struct {
	callCount   int
	lastCtx     context.Context
	lastInput   *toolsv1.ToolCallInput
	nextOutput  *toolsv1.ToolCallOutput
	nextErr     error
	blockOnCall bool
}

func (f *fakeBridge) Call(ctx context.Context, input *toolsv1.ToolCallInput) (*toolsv1.ToolCallOutput, error) {
	f.callCount++
	f.lastCtx = ctx
	f.lastInput = input
	if f.blockOnCall {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.nextOutput, f.nextErr
}

func TestNotebookMCPBridge_ExecuteCellsRequiresApproval(t *testing.T) {
	bridge := &fakeBridge{
		nextOutput: &toolsv1.ToolCallOutput{
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
			Output: &toolsv1.ToolCallOutput_ExecuteCells{
				ExecuteCells: &toolsv1.NotebookServiceExecuteCellsResponse{},
			},
		},
	}
	nb := NewNotebookMCPBridge(bridge)
	nb.SetExecuteApprover(contextExecuteApprover{})

	result, err := nb.handleExecuteCells(context.Background(), mcp.CallToolRequest{}, executeCellsArgs{RefIDs: []string{"cell-1"}})
	if err != nil {
		t.Fatalf("handleExecuteCells returned error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected ExecuteCells to be rejected without approval")
	}
	if bridge.callCount != 0 {
		t.Fatalf("bridge call count = %d, want 0", bridge.callCount)
	}
}

func TestNotebookMCPBridge_ExecuteCellsWithApprovalCallsBridge(t *testing.T) {
	bridge := &fakeBridge{
		nextOutput: &toolsv1.ToolCallOutput{
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
			Output: &toolsv1.ToolCallOutput_ExecuteCells{
				ExecuteCells: &toolsv1.NotebookServiceExecuteCellsResponse{},
			},
		},
	}
	nb := NewNotebookMCPBridge(bridge)
	nb.SetExecuteApprover(contextExecuteApprover{})
	ctx := context.WithValue(context.Background(), approvedRefIDsContextKey, []string{"cell-1"})

	result, err := nb.handleExecuteCells(ctx, mcp.CallToolRequest{}, executeCellsArgs{RefIDs: []string{"cell-1"}})
	if err != nil {
		t.Fatalf("handleExecuteCells returned error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("expected ExecuteCells to succeed with approval")
	}
	if bridge.callCount != 1 {
		t.Fatalf("bridge call count = %d, want 1", bridge.callCount)
	}
}

func TestNotebookMCPBridge_AppliesDefaultCallTimeout(t *testing.T) {
	bridge := &fakeBridge{blockOnCall: true}
	nb := NewNotebookMCPBridge(bridge)
	nb.SetCallTimeout(20 * time.Millisecond)

	result, err := nb.handleListCells(context.Background(), mcp.CallToolRequest{}, listCellsArgs{})
	if err != nil {
		t.Fatalf("handleListCells returned error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected timeout to be surfaced as tool error")
	}
	if bridge.lastCtx == nil {
		t.Fatalf("bridge context was not captured")
	}
	if _, hasDeadline := bridge.lastCtx.Deadline(); !hasDeadline {
		t.Fatalf("bridge context should have a deadline")
	}
}

func TestNotebookMCPBridge_ExecuteCellsUsesApprovalManager(t *testing.T) {
	bridge := &fakeBridge{
		nextOutput: &toolsv1.ToolCallOutput{
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
			Output: &toolsv1.ToolCallOutput_ExecuteCells{
				ExecuteCells: &toolsv1.NotebookServiceExecuteCellsResponse{},
			},
		},
	}
	approvalManager := NewExecuteApprovalManager(10 * time.Minute)
	nb := NewNotebookMCPBridge(bridge)
	nb.SetExecuteApprover(executeApprovalApprover{manager: approvalManager})
	ctx := context.WithValue(context.Background(), sessionIDContextKey, "session-1")

	result, err := nb.handleExecuteCells(ctx, mcp.CallToolRequest{}, executeCellsArgs{RefIDs: []string{"cell-1"}})
	if err != nil {
		t.Fatalf("handleExecuteCells returned error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected ExecuteCells to be rejected before approval")
	}
	if bridge.callCount != 0 {
		t.Fatalf("bridge call count = %d, want 0", bridge.callCount)
	}
	if pending := approvalManager.ListPending("session-1"); len(pending) != 1 {
		t.Fatalf("pending approvals = %d, want 1", len(pending))
	}

	if err := approvalManager.Approve("session-1", []string{"cell-1"}); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	result, err = nb.handleExecuteCells(ctx, mcp.CallToolRequest{}, executeCellsArgs{RefIDs: []string{"cell-1"}})
	if err != nil {
		t.Fatalf("handleExecuteCells returned error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("expected ExecuteCells to succeed after approval")
	}
	if bridge.callCount != 1 {
		t.Fatalf("bridge call count = %d, want 1", bridge.callCount)
	}

	result, err = nb.handleExecuteCells(ctx, mcp.CallToolRequest{}, executeCellsArgs{RefIDs: []string{"cell-1"}})
	if err != nil {
		t.Fatalf("handleExecuteCells returned error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected one-time approval to be consumed")
	}
	if bridge.callCount != 1 {
		t.Fatalf("bridge call count = %d, want 1 after consumed approval", bridge.callCount)
	}
}

func TestNotebookMCPBridge_ExecuteCodeCallsBridge(t *testing.T) {
	bridge := &fakeBridge{
		nextOutput: &toolsv1.ToolCallOutput{
			Status: toolsv1.ToolCallOutput_STATUS_SUCCESS,
			Output: &toolsv1.ToolCallOutput_ExecuteCode{
				ExecuteCode: &toolsv1.ExecuteCodeResponse{
					Output: "ok\n",
				},
			},
		},
	}
	nb := NewNotebookMCPBridge(bridge)

	result, err := nb.handleExecuteCode(context.Background(), mcp.CallToolRequest{}, executeCodeArgs{Code: "console.log('ok')"}) //nolint:forbidigo
	if err != nil {
		t.Fatalf("handleExecuteCode returned error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("expected ExecuteCode to succeed")
	}
	if bridge.callCount != 1 {
		t.Fatalf("bridge call count = %d, want 1", bridge.callCount)
	}
	if bridge.lastInput == nil || bridge.lastInput.GetExecuteCode() == nil {
		t.Fatalf("expected ExecuteCode input to be dispatched")
	}
	if bridge.lastInput.GetExecuteCode().GetCode() != "console.log('ok')" {
		t.Fatalf("execute code = %q, want %q", bridge.lastInput.GetExecuteCode().GetCode(), "console.log('ok')")
	}
}

func TestNotebookMCPBridge_NewServerRegistersExecuteCodeOnly(t *testing.T) {
	nb := NewNotebookMCPBridge(&fakeBridge{})
	server := nb.NewServer()
	tools := server.ListTools()
	if len(tools) != 1 {
		t.Fatalf("tool count = %d, want 1", len(tools))
	}
	if _, ok := tools[toolsv1mcp.NotebookService_ExecuteCodeToolOpenAI.Name]; !ok {
		t.Fatalf("expected ExecuteCode tool to be registered; got keys=%v", mapKeys(tools))
	}
}

func mapKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	return keys
}
