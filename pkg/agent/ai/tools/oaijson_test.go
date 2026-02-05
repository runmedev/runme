package tools

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
)

func Test_FixOpenAI(t *testing.T) {
	// Unittest to reproduce failure to fix json.
	data := "{\"cells\":[{\"call_id\":\"1\",\"doc_results\":[],\"execution_summary\":null,\"kind\":\"CELL_KIND_CODE\",\"language_id\":\"python\",\"metadata\":[{\"key\":\"agent/summary\",\"value\":\"A simple hello world program.\"}],\"outputs\":[],\"ref_id\":\"\",\"role\":\"CELL_ROLE_USER\",\"text_range\":null,\"value\":\"print('Hello, World!')\"}]}"

	args := make(map[string]any)
	if err := json.Unmarshal([]byte(data), &args); err != nil {
		t.Fatal(err)
	}

	req := &toolsv1.UpdateCellsRequest{}
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
