package server

import (
	"context"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/runmedev/runme/v3/internal/config"
	"github.com/runmedev/runme/v3/internal/server"
	runmetls "github.com/runmedev/runme/v3/internal/tls"
)

// defaultGRPCurlFormat indicates the default format for the grpcurl commands.
var defaultGRPCurlFormat = grpcurl.Format("json")

func getDescriptorSource(ctx context.Context, cfg *config.Config) (grpcurl.DescriptorSource, error) {
	cc, err := dialServer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	client := grpcreflect.NewClientAuto(ctx, cc)
	return grpcurl.DescriptorSourceFromServer(ctx, client), nil
}

func dialServer(ctx context.Context, cfg *config.Config) (*grpc.ClientConn, error) {
	tlsConf, err := runmetls.LoadClientConfig(*cfg.Server.Tls.CertFile, *cfg.Server.Tls.KeyFile)
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(tlsConf)

	network, addr, err := server.SplitAddress(cfg.Server.Address)
	if err != nil {
		return nil, err
	}

	cc, err := grpcurl.BlockingDial(ctx, network, addr, creds)
	return cc, errors.WithStack(err)
}
