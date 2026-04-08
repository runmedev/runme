package tools

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
)

func Test_FixOpenAI(t *testing.T) {
	data := `{"code":"console.log('Hello, World!')"}`

	args := make(map[string]any)
	if err := json.Unmarshal([]byte(data), &args); err != nil {
		t.Fatal(err)
	}

	req := &toolsv1.ExecuteCodeRequest{}
	descriptor := req.ProtoReflect().Descriptor()
	// runtime.FixOpenAI(descriptor, args)
	FixOpenAI(descriptor, args)

	fixedJson, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	if err := protojson.Unmarshal(fixedJson, req); err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", req)
}
