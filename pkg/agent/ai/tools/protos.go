package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/openai/openai-go/v2/responses"
	"github.com/pkg/errors"
	aisreproto "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	"go.openai.org/lib/oaigo/telemetry/oailog"
	"go.openai.org/project/aisre/toolsgen/aisremcp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ArgsToToolCallInput converts the string representation of the toolcall arguments returned by OpenAI into a proto
func ArgsToToolCallInput(ctx context.Context, name string, callID string, args string) (*aisreproto.ToolCallInput, error) {
	callInput := &aisreproto.ToolCallInput{
		CallId: callID,
	}

	// OpenAI JSON needs to be converted to proto json
	// https://github.com/redpanda-data/protoc-gen-go-mcp?tab=readme-ov-file#openai-compatible
	var descriptor protoreflect.MessageDescriptor

	switch name {
	case aisremcp.NotebookService_UpdateCellsToolOpenAI.Name:
		descriptor = (&aisreproto.UpdateCellsRequest{}).ProtoReflect().Descriptor()
	case aisremcp.NotebookService_ListCellsToolOpenAI.Name:
		descriptor = (&aisreproto.ListCellsRequest{}).ProtoReflect().Descriptor()
	case aisremcp.NotebookService_GetCellsToolOpenAI.Name:
		descriptor = (&aisreproto.GetCellsRequest{}).ProtoReflect().Descriptor()
	case aisremcp.NotebookService_ExecuteCellsToolOpenAI.Name:
		descriptor = (&aisreproto.NotebookServiceExecuteCellsRequest{}).ProtoReflect().Descriptor()
	case aisremcp.NotebookService_TerminateRunToolOpenAI.Name:
		descriptor = (&aisreproto.TerminateRunRequest{}).ProtoReflect().Descriptor()
	case aisremcp.NotebookService_SendSlackMessageToolOpenAI.Name:
		descriptor = (&aisreproto.SendSlackMessageRequest{}).ProtoReflect().Descriptor()
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
		oailog.Error(ctx, "No message descriptor for tool; argument won't be converted to proto JSON", oailog.String("tool", name))
	}

	// Now that we've fixed the json to be proto compatible we can deserialize it as a proto
	var pbMessage proto.Message

	switch name {
	case aisremcp.NotebookService_UpdateCellsToolOpenAI.Name:
		callInput.Input = &aisreproto.ToolCallInput_UpdateCells{
			UpdateCells: &aisreproto.UpdateCellsRequest{},
		}
		pbMessage = callInput.GetUpdateCells()
	case aisremcp.NotebookService_ListCellsToolOpenAI.Name:
		callInput.Input = &aisreproto.ToolCallInput_ListCells{
			ListCells: &aisreproto.ListCellsRequest{},
		}
		pbMessage = callInput.GetListCells()
	case aisremcp.NotebookService_GetCellsToolOpenAI.Name:
		callInput.Input = &aisreproto.ToolCallInput_GetCells{
			GetCells: &aisreproto.GetCellsRequest{},
		}
		pbMessage = callInput.GetGetCells()
	case aisremcp.NotebookService_ExecuteCellsToolOpenAI.Name:
		callInput.Input = &aisreproto.ToolCallInput_ExecuteCells{
			ExecuteCells: &aisreproto.NotebookServiceExecuteCellsRequest{},
		}
		pbMessage = callInput.GetExecuteCells()
	case aisremcp.NotebookService_TerminateRunToolOpenAI.Name:
		callInput.Input = &aisreproto.ToolCallInput_TerminateRun{
			TerminateRun: &aisreproto.TerminateRunRequest{},
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
	case aisremcp.NotebookService_UpdateCellsToolOpenAI.Name:
		if err := ensureValidUpdateCellsRequest(ctx, callInput.GetUpdateCells()); err != nil {
			return callInput, err
		}
	}

	return callInput, nil
}

// ensureValidUpdateCellsRequest applies some validation to the UpdateCellsRequest
func ensureValidUpdateCellsRequest(ctx context.Context, m *aisreproto.UpdateCellsRequest) error {
	// Add cell ids for any new cells
	for _, c := range m.GetCells() {
		if c.RefId == "" {
			c.RefId = uuid.NewString()
			oailog.Info(ctx, "Generated cellid", oailog.String("id", c.RefId))
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
func AddToolCallOutputToResponse(_ context.Context, toolCallOutput *aisreproto.ToolCallOutput, resp *responses.ResponseNewParams) error {
	if toolCallOutput == nil {
		return nil
	}

	if toolCallOutput.CallId == "" {
		return errors.WithStack(errors.New("missing call id"))
	}

	var m proto.Message

	switch toolCallOutput.Output.(type) {
	case *aisreproto.ToolCallOutput_UpdateCells:
		m = toolCallOutput.GetUpdateCells()
	case *aisreproto.ToolCallOutput_ListCells:
		m = toolCallOutput.GetListCells()
	case *aisreproto.ToolCallOutput_GetCells:
		m = toolCallOutput.GetGetCells()
	case *aisreproto.ToolCallOutput_ExecuteCells:
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
