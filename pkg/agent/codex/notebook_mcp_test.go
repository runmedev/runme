package codex

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
)

type fakeBridge struct {
	callCount   int
	lastCtx     context.Context
	nextOutput  *toolsv1.ToolCallOutput
	nextErr     error
	blockOnCall bool
}

func (f *fakeBridge) Call(ctx context.Context, _ *toolsv1.ToolCallInput) (*toolsv1.ToolCallOutput, error) {
	f.callCount++
	f.lastCtx = ctx
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
