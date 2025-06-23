package testutil

import (
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/runmedev/runme/v3/api/gen/proto/go/agent"
)

// TODO(jlewi): We should deprecate this and use
//  cmp.Diff(tc.expected, tc.request, protocmp.Transform());

var BlockComparer = cmpopts.IgnoreUnexported(agent.Block{}, agent.BlockOutput{}, agent.BlockOutputItem{})
