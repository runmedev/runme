package runme

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
)

var (
	tripleTick  = strings.Repeat("`", 3)
	rawMarkdown = []byte(
		"" +
			tripleTick + "sh {\"id\":\"fc_6876a065ed948191944d4d42c29a519f08eb2d11327b5efb\"}" + "\n" +
			"gcloud config set project rando-app && gcloud container clusters list\n" +
			tripleTick + "\n\n" +
			"It appears that the Kubernetes Engine API is not enabled (or has never been used) in the rando-app project, so I can't list running workloads there from GKE.\n\n" +
			"Would you like me to check for other resources running in this project (such as Compute Engine VMs, Cloud Run, App Engine, etc.), or do you want help with enabling Kubernetes? Let me know your focus!\n\n" +
			"what about functions?\n\n" +
			tripleTick + "sh {\"id\":\"fc_6876a0761fc08191a8dc494f891758c208eb2d11327b5efb\"}" + "\n" +
			"gcloud config set project rando-app && gcloud functions list\n" +
			tripleTick + "\n\n" +
			"There are currently two Google Cloud Functions running in the rando-app project:\n\n" +
			"1. **beforeCreate** (HTTP Trigger, us-central1, 2nd gen, ACTIVE)\n" +
			"2. **beforeSignedIn** (HTTP Trigger, us-central1, 2nd gen, ACTIVE)\n\n" +
			"Let me know if you want to see details, source, or logs for either function.\n",
	)
)

func Test_DeserializeSerializeRoundtrip(t *testing.T) {
	logger := zap.NewNop()
	parser := NewParser(logger)

	desReq := &parserv1.DeserializeRequest{
		Source: rawMarkdown,
	}
	resp, err := parser.Deserialize(context.Background(), connect.NewRequest(desReq))
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	notebook := resp.Msg.Notebook
	serReq := &parserv1.SerializeRequest{
		Notebook: notebook,
	}
	serResp, err := parser.Serialize(context.Background(), connect.NewRequest(serReq))
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	got := string(serResp.Msg.Result)
	want := string(rawMarkdown)
	if got != want {
		t.Errorf("Markdown parser roundtrip mismatch.\nGot:\n%s\n\nWant:\n%s", got, want)
	}
}
