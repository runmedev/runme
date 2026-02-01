package tools

import (
	"encoding/json"
	"testing"

	aisreproto "go.openai.org/oaiproto/aisre"
	"google.golang.org/protobuf/encoding/protojson"
)

func Test_FixOpenAI(t *testing.T) {
	// Unittest to reproduce failure to fix json.
	data := "{\"cells\":[{\"call_id\":\"1\",\"doc_results\":[],\"execution_summary\":null,\"kind\":\"CELL_KIND_CODE\",\"language_id\":\"python\",\"metadata\":[{\"key\":\"agent/summary\",\"value\":\"A simple hello world program.\"}],\"outputs\":[],\"ref_id\":\"\",\"role\":\"CELL_ROLE_USER\",\"text_range\":null,\"value\":\"print('Hello, World!')\"}]}"

	args := make(map[string]any)
	if err := json.Unmarshal([]byte(data), &args); err != nil {
		t.Fatal(err)
	}

	req := &aisreproto.UpdateCellsRequest{}
	descriptor := req.ProtoReflect().Descriptor()
	//runtime.FixOpenAI(descriptor, args)
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
