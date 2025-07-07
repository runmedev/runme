package testutil

import (
	"github.com/google/go-cmp/cmp/cmpopts"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
)

// TODO(jlewi): We should deprecate this and use
//  cmp.Diff(tc.expected, tc.request, protocmp.Transform());

var BlockComparer = cmpopts.IgnoreUnexported(agentv1.Block{}, agentv1.BlockOutput{}, agentv1.BlockOutputItem{})
