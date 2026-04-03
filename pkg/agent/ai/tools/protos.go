package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/responses"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1/toolsv1mcp"
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
	case toolsv1mcp.NotebookService_ExecuteCodeToolOpenAI.Name:
		descriptor = (&toolsv1.ExecuteCodeRequest{}).ProtoReflect().Descriptor()
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
	}

	// Now that we've fixed the json to be proto compatible we can deserialize it as a proto
	var pbMessage proto.Message

	switch name {
	case toolsv1mcp.NotebookService_ExecuteCodeToolOpenAI.Name:
		callInput.Input = &toolsv1.ToolCallInput_ExecuteCode{
			ExecuteCode: &toolsv1.ExecuteCodeRequest{},
		}
		pbMessage = callInput.GetExecuteCode()
	default:
		return callInput, errors.Errorf("Unknown message type: %s", name)
	}
	if err := mapToProto(argsMap, pbMessage); err != nil {
		return callInput, errors.Wrap(err, "Failed to deserialize map arguments to proto")
	}

	return callInput, nil
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
	case *toolsv1.ToolCallOutput_ExecuteCode:
		m = toolCallOutput.GetExecuteCode()
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
