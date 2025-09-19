package server

import (
	"cmp"
	"crypto/tls"
	"net"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/runmedev/runme/v3/internal/config"
	runmetls "github.com/runmedev/runme/v3/internal/tls"
)

type Server struct {
	cfg    *config.Config
	gs     *grpc.Server
	lis    net.Listener
	logger *zap.Logger
}

type ServiceRegistrar func(grpc.ServiceRegistrar)

func SplitAddress(raw string) (network, addr string, _ error) {
	if u, err := url.Parse(raw); err == nil {
		switch u.Scheme {
		case "tcp":
			return u.Scheme, cmp.Or(u.Opaque, u.Host), nil
		case "unix":
			return u.Scheme, cmp.Or(u.Opaque, u.Path), nil
		}
	}

	return "tcp", raw, nil
}

func New(
	cfg *config.Config,
	logger *zap.Logger,
	registrar ServiceRegistrar,
) (*Server, error) {
	logger = logger.Named("Server")

	tlsCfg, err := createTLSConfig(cfg, logger)
	if err != nil {
		return nil, err
	}

	grpcServer := createGRPCServer(cfg, tlsCfg)

	// Register runme services.
	registrar(grpcServer)

	// Register health service.
	healthcheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthcheck)
	// Setting SERVING for the whole system.
	healthcheck.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection service.
	reflection.Register(grpcServer)

	s := Server{
		cfg:    cfg,
		gs:     grpcServer,
		logger: logger,
	}

	return &s, nil
}

func (s *Server) Addr() string {
	if s.lis == nil {
		return "<nil>"
	}
	return s.lis.Addr().String()
}

func (s *Server) Serve() (err error) {
	s.lis, err = createListener(s.cfg.Server.Address)
	if err != nil {
		return err
	}
	s.logger.Info("starting gRPC server", zap.String("address", s.Addr()))
	return s.gs.Serve(s.lis)
}

func (s *Server) Shutdown() {
	s.logger.Info("stopping gRPC server")
	s.gs.GracefulStop()
}

func createListener(addr string) (net.Listener, error) {
	network, addr, err := SplitAddress(addr)
	if err != nil {
		return nil, err
	}

	if network == "unix" {
		if _, err := os.Stat(addr); err == nil {
			return nil, &os.PathError{Op: "listen", Path: addr, Err: os.ErrExist}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	lis, err := net.Listen(network, addr)
	return lis, errors.WithStack(err)
}

func createTLSConfig(cfg *config.Config, logger *zap.Logger) (*tls.Config, error) {
	if tls := cfg.Server.Tls; tls != nil && tls.Enabled {
		// TODO(adamb): redesign runmetls API.
		return runmetls.LoadOrGenerateConfig(
			*tls.CertFile, // guaranteed in [getRootConfig]
			*tls.KeyFile,  // guaranteed in [getRootConfig]
			logger,
		)
	}
	return nil, nil
}

func createGRPCServer(cfg *config.Config, tlsCfg *tls.Config) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.Server.MaxMessageSize),
		grpc.MaxSendMsgSize(cfg.Server.MaxMessageSize),
	}

	if tlsCfg != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}

	return grpc.NewServer(opts...)
}
