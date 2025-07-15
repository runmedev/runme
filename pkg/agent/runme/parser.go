package runme

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	parserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
	connectparserv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1/parserv1connect"
	"github.com/runmedev/runme/v3/document/editor/editorservice"
)

type Parser struct {
	Server parserv1.ParserServiceServer
}

func NewParser(logger *zap.Logger) *Parser {
	server := editorservice.NewParserServiceServer(logger)
	return &Parser{
		Server: server,
	}
}

var _ connectparserv1.ParserServiceHandler = (*Parser)(nil)

func (e *Parser) Deserialize(ctx context.Context, req *connect.Request[parserv1.DeserializeRequest]) (*connect.Response[parserv1.DeserializeResponse], error) {
	resp, err := e.Server.Deserialize(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (e *Parser) Serialize(ctx context.Context, req *connect.Request[parserv1.SerializeRequest]) (*connect.Response[parserv1.SerializeResponse], error) {
	resp, err := e.Server.Serialize(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
