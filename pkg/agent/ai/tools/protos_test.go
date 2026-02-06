package tools

import (
	"context"
	"testing"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1/toolsv1mcp"
)

func Test_ArgsToToolCallInput(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		toolName string
		args     string
		validate func(t *testing.T, result *toolsv1.ToolCallInput)
	}{
		{
			name:     "TerminateRun populates correct oneof",
			toolName: toolsv1mcp.NotebookService_TerminateRunToolOpenAI.Name,
			args:     "{}",
			validate: func(t *testing.T, result *toolsv1.ToolCallInput) {
				if result.GetTerminateRun() == nil {
					t.Fatal("expected TerminateRun to be set, got nil")
				}
				if result.GetExecuteCells() != nil {
					t.Fatal("ExecuteCells should not be set for TerminateRun")
				}
			},
		},
		{
			name:     "ExecuteCells populates correct oneof",
			toolName: toolsv1mcp.NotebookService_ExecuteCellsToolOpenAI.Name,
			args:     `{"ref_ids":["cell-1"]}`,
			validate: func(t *testing.T, result *toolsv1.ToolCallInput) {
				ec := result.GetExecuteCells()
				if ec == nil {
					t.Fatal("expected ExecuteCells to be set, got nil")
				}
				if len(ec.GetRefIds()) != 1 || ec.GetRefIds()[0] != "cell-1" {
					t.Fatalf("expected ref_ids=[cell-1], got %v", ec.GetRefIds())
				}
			},
		},
		{
			name:     "ListCells populates correct oneof",
			toolName: toolsv1mcp.NotebookService_ListCellsToolOpenAI.Name,
			args:     "{}",
			validate: func(t *testing.T, result *toolsv1.ToolCallInput) {
				if result.GetListCells() == nil {
					t.Fatal("expected ListCells to be set, got nil")
				}
			},
		},
		{
			name:     "GetCells populates correct oneof",
			toolName: toolsv1mcp.NotebookService_GetCellsToolOpenAI.Name,
			args:     `{"ref_ids":["abc"]}`,
			validate: func(t *testing.T, result *toolsv1.ToolCallInput) {
				gc := result.GetGetCells()
				if gc == nil {
					t.Fatal("expected GetCells to be set, got nil")
				}
			},
		},
		{
			name:     "SendSlackMessage populates correct oneof",
			toolName: toolsv1mcp.NotebookService_SendSlackMessageToolOpenAI.Name,
			args:     `{"channel":"C123","text":"hello","timestamp":"","file_ids":[]}`,
			validate: func(t *testing.T, result *toolsv1.ToolCallInput) {
				sm := result.GetSendSlackMessage()
				if sm == nil {
					t.Fatal("expected SendSlackMessage to be set, got nil")
				}
				if sm.GetChannel() != "C123" {
					t.Fatalf("expected channel=C123, got %s", sm.GetChannel())
				}
			},
		},
		{
			name:     "UpdateCells populates correct oneof",
			toolName: toolsv1mcp.NotebookService_UpdateCellsToolOpenAI.Name,
			args:     `{"cells":[{"kind":"CELL_KIND_CODE","language_id":"python","value":"print('hi')"}]}`,
			validate: func(t *testing.T, result *toolsv1.ToolCallInput) {
				uc := result.GetUpdateCells()
				if uc == nil {
					t.Fatal("expected UpdateCells to be set, got nil")
				}
				if len(uc.GetCells()) != 1 {
					t.Fatalf("expected 1 cell, got %d", len(uc.GetCells()))
				}
				if uc.GetCells()[0].GetRefId() == "" {
					t.Fatal("expected ref_id to be auto-generated")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ArgsToToolCallInput(ctx, tt.toolName, "call-123", tt.args)
			if err != nil {
				t.Fatalf("ArgsToToolCallInput returned error: %v", err)
			}
			if result.GetCallId() != "call-123" {
				t.Fatalf("expected call_id=call-123, got %s", result.GetCallId())
			}
			tt.validate(t, result)
		})
	}
}

func Test_ArgsToToolCallInput_UnrecognizedTool(t *testing.T) {
	ctx := context.Background()
	_, err := ArgsToToolCallInput(ctx, "nonexistent_tool", "call-1", "{}")
	if err == nil {
		t.Fatal("expected error for unrecognized tool")
	}
}
