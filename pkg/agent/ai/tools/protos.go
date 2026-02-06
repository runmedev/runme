package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/openai/openai-go/responses"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1/toolsv1mcp"
	"github.com/runmedev/runme/v3/pkg/agent/logs"
)

// ArgsToToolCallInput converts the string representation of the toolcall arguments returned by OpenAI into a proto
func ArgsToToolCallInput(ctx context.Context, name string, callID string, args string) (*toolsv1.ToolCallInput, error) {
	callInput := &toolsv1.ToolCallInput{
		CallId: callID,
	}

	// OpenAI JSON needs to be converted to proto json
	// https://github.com/redpanda-data/protoc-gen-go-mcp?tab=readme-ov-file#openai-compatible
	var descriptor protoreflect.MessageDescriptor

	switch name {
	case toolsv1mcp.NotebookService_UpdateCellsToolOpenAI.Name:
		descriptor = (&toolsv1.UpdateCellsRequest{}).ProtoReflect().Descriptor()
	case toolsv1mcp.NotebookService_ListCellsToolOpenAI.Name:
		descriptor = (&toolsv1.ListCellsRequest{}).ProtoReflect().Descriptor()
	case toolsv1mcp.NotebookService_GetCellsToolOpenAI.Name:
		descriptor = (&toolsv1.GetCellsRequest{}).ProtoReflect().Descriptor()
	case toolsv1mcp.NotebookService_ExecuteCellsToolOpenAI.Name:
		descriptor = (&toolsv1.NotebookServiceExecuteCellsRequest{}).ProtoReflect().Descriptor()
	case toolsv1mcp.NotebookService_TerminateRunToolOpenAI.Name:
		descriptor = (&toolsv1.TerminateRunRequest{}).ProtoReflect().Descriptor()
	case toolsv1mcp.NotebookService_SendSlackMessageToolOpenAI.Name:
		descriptor = (&toolsv1.SendSlackMessageRequest{}).ProtoReflect().Descriptor()
	default:
		return nil, errors.Errorf("unrecognized toolcall: %s", name)
	}

	// Parse the json string into a map
	// This is necessary so that we can fix the JSON.
	argsMap, err := parseToolArguments(args)
	if err != nil {
		return callInput, err
	}
	if descriptor != nil {
		FixOpenAI(descriptor, argsMap)
	} else {
		logger := logs.FromContext(ctx).WithValues("tool", name)
		logger.Error(nil, "No message descriptor for tool; argument won't be converted to proto JSON")
	}

	// Now that we've fixed the json to be proto compatible we can deserialize it as a proto
	var pbMessage proto.Message

	switch name {
	case toolsv1mcp.NotebookService_UpdateCellsToolOpenAI.Name:
		callInput.Input = &toolsv1.ToolCallInput_UpdateCells{
			UpdateCells: &toolsv1.UpdateCellsRequest{},
		}
		pbMessage = callInput.GetUpdateCells()
	case toolsv1mcp.NotebookService_ListCellsToolOpenAI.Name:
		callInput.Input = &toolsv1.ToolCallInput_ListCells{
			ListCells: &toolsv1.ListCellsRequest{},
		}
		pbMessage = callInput.GetListCells()
	case toolsv1mcp.NotebookService_GetCellsToolOpenAI.Name:
		callInput.Input = &toolsv1.ToolCallInput_GetCells{
			GetCells: &toolsv1.GetCellsRequest{},
		}
		pbMessage = callInput.GetGetCells()
	case toolsv1mcp.NotebookService_ExecuteCellsToolOpenAI.Name:
		callInput.Input = &toolsv1.ToolCallInput_ExecuteCells{
			ExecuteCells: &toolsv1.NotebookServiceExecuteCellsRequest{},
		}
		pbMessage = callInput.GetExecuteCells()
	case toolsv1mcp.NotebookService_TerminateRunToolOpenAI.Name:
		callInput.Input = &toolsv1.ToolCallInput_TerminateRun{
			TerminateRun: &toolsv1.TerminateRunRequest{},
		}
		pbMessage = callInput.GetExecuteCells()
	default:
		return callInput, errors.Errorf("Unknown message type: %s", name)
	}

	if pbMessage == nil {
		return callInput, errors.Errorf("Message type: %s produced nil pbMessage; this is a bug", name)
	}
	if err := mapToProto(argsMap, pbMessage); err != nil {
		return callInput, errors.Wrap(err, "Failed to deserialize map arguments to proto")
	}

	switch name {
	case toolsv1mcp.NotebookService_UpdateCellsToolOpenAI.Name:
		if err := ensureValidUpdateCellsRequest(ctx, callInput.GetUpdateCells()); err != nil {
			return callInput, err
		}
	}

	return callInput, nil
}

// ensureValidUpdateCellsRequest applies some validation to the UpdateCellsRequest
func ensureValidUpdateCellsRequest(ctx context.Context, m *toolsv1.UpdateCellsRequest) error {
	// Add cell ids for any new cells
	for _, c := range m.GetCells() {
		if c.RefId == "" {
			c.RefId = uuid.NewString()
			logger := logs.FromContext(ctx).WithValues("id", c.RefId)
			logger.Info("Generated cellid")
		}
	}

	return nil
}

// mapToProto converts a map[string]any to a proto message
func mapToProto(m map[string]any, p proto.Message) error {
	if p == nil {
		return errors.WithStack(errors.New("p can't be nil"))
	}
	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshal map to json")
	}

	return protojson.Unmarshal(b, p)
}

func parseToolArguments(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return parsed, err
	}
	return parsed, nil
}

func ProtoToMap(m proto.Message) (map[string]any, error) {
	b, err := protojson.Marshal(m)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal message as proto")
	}

	result := make(map[string]any)
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal message as map[string]any")
	}
	return result, nil
}

// AddToolCallOutputToResponse takes toolCallOutput sent by chatkit and adds it to the response.
func AddToolCallOutputToResponse(_ context.Context, toolCallOutput *toolsv1.ToolCallOutput, resp *responses.ResponseNewParams) error {
	if toolCallOutput == nil {
		return nil
	}

	if toolCallOutput.CallId == "" {
		return errors.WithStack(errors.New("missing call id"))
	}

	var m proto.Message

	switch toolCallOutput.Output.(type) {
	case *toolsv1.ToolCallOutput_UpdateCells:
		m = toolCallOutput.GetUpdateCells()
	case *toolsv1.ToolCallOutput_ListCells:
		m = toolCallOutput.GetListCells()
	case *toolsv1.ToolCallOutput_GetCells:
		m = toolCallOutput.GetGetCells()
	case *toolsv1.ToolCallOutput_ExecuteCells:
		m = toolCallOutput.GetExecuteCells()
	default:
		return errors.WithStack(fmt.Errorf("unexpected type %T", toolCallOutput.Output))
	}

	b, err := protojson.Marshal(m)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal tool call output")
	}

	resp.Input.OfInputItemList = append(resp.Input.OfInputItemList, responses.ResponseInputItemUnionParam{
		OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
			// TODO(jlewi): What if the model didn't tell us to call that function?
			CallID: toolCallOutput.GetCallId(),
			Output: string(b),
		},
	})

	return nil
}
