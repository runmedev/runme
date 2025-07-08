package testutil

import (
	"github.com/google/go-cmp/cmp/cmpopts"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
)

// TODO(jlewi): We should deprecate this and use
//  cmp.Diff(tc.expected, tc.request, protocmp.Transform());

var CellComparer = cmpopts.IgnoreUnexported(parserv1.Cell{}, parserv1.CellOutput{}, parserv1.CellOutputItem{}, parserv1.TextRange{})
