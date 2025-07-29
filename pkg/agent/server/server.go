package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	agentv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1/agentv1connect"

	"github.com/runmedev/runme/v3/pkg/agent/api"
	"github.com/runmedev/runme/v3/pkg/agent/iam"
	"github.com/runmedev/runme/v3/pkg/agent/runme"
	"github.com/runmedev/runme/v3/pkg/agent/runme/stream"

	"github.com/runmedev/runme/v3/pkg/agent/logs"
	"github.com/runmedev/runme/v3/pkg/agent/tlsbuilder"

	"connectrpc.com/otelconnect"

	"github.com/go-logr/zapr"
	"github.com/pkg/errors"

	"github.com/runmedev/runme/v3/pkg/agent/config"

	"github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1/parserv1connect"
	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	"github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2/runnerv2connect"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

// Server is the main server for the cloud assistant
type Server struct {
	telemetry        *config.TelemetryConfig
	serverConfig     *config.AssistantServerConfig
	webAppConfig     *agentv1.WebAppConfig
	hServer          *http.Server
	engine           http.Handler
	shutdownComplete chan bool
	runner           *runme.Runner
	parser           *runme.Parser
	agent            agentv1connect.MessagesServiceHandler
	checker          iam.Checker
	registerHandlers RegisterHandlers
}

type RegisterHandlers func(mux *AuthMux, checker iam.Checker, interceptors []connect.Interceptor) error
type Options struct {
	Telemetry *config.TelemetryConfig
	Server    *config.AssistantServerConfig
	WebApp    *agentv1.WebAppConfig
	IAMPolicy *api.IAMPolicy
	// RegisterHandlers is a callback that allows you to register additional handlers in the server.
	// These could be regular HTTP handlers or proto services.
	RegisterHandlers RegisterHandlers
}

// NewServer creates a new server
func NewServer(opts Options, agent agentv1connect.MessagesServiceHandler) (*Server, error) {
	log := zapr.NewLogger(zap.L())
	if agent == nil {
		if !opts.Server.RunnerService {
			return nil, errors.New("Agent and Runner service are both disabled")
		}
		log.Info("Agent is nil; continuing without AI service")
	}

	var runner *runme.Runner

	if opts.Server.RunnerService {
		var err error
		runner, err = runme.NewRunner(zap.L())
		if err != nil {
			return nil, err
		}
		ctx := context.Background()
		session, err := runner.Server.CreateSession(ctx, &runnerv2.CreateSessionRequest{
			Project: &runnerv2.Project{
				Root:         ".",
				EnvLoadOrder: []string{".env", ".env.local", ".env.development", ".env.dev"},
			},
			Config: &runnerv2.CreateSessionRequest_Config{
				EnvStoreSeeding: runnerv2.CreateSessionRequest_Config_SESSION_ENV_STORE_SEEDING_SYSTEM.Enum(),
			},
		})
		if err != nil {
			return nil, err
		}
		log.Info("Runner session created", "sessionID", session.GetSession().GetId())
	} else {
		log.Info("Runner service is disabled")
	}

	var parser *runme.Parser

	if opts.Server.ParserService {
		parser = runme.NewParser(zap.L())
	}

	if opts.Server.OIDC == nil && opts.IAMPolicy != nil {
		return nil, errors.New("IAM policy is set but OIDC is not configured")
	}

	if opts.Server.OIDC != nil && opts.IAMPolicy == nil {
		return nil, errors.New("IAM policy must be set if OIDC is configured")
	}

	var checker iam.Checker

	if opts.IAMPolicy != nil {
		c, err := iam.NewChecker(*opts.IAMPolicy)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create IAM policy checker")
		}
		checker = c
	} else {
		checker = &iam.AllowAllChecker{}
	}

	s := &Server{
		telemetry:        opts.Telemetry,
		serverConfig:     opts.Server,
		webAppConfig:     opts.WebApp,
		runner:           runner,
		parser:           parser,
		agent:            agent,
		checker:          checker,
		registerHandlers: opts.RegisterHandlers,
	}
	return s, nil
}

// Run starts the http server
// Cells until its shutdown.
func (s *Server) Run() error {
	s.shutdownComplete = make(chan bool, 1)
	trapInterrupt(s)

	logZ := zap.L()
	log := zapr.NewLogger(logZ)

	// Register the services
	if err := s.registerServices(); err != nil {
		return errors.Wrapf(err, "Failed to register services")
	}

	serverConfig := s.serverConfig
	if serverConfig == nil {
		serverConfig = &config.AssistantServerConfig{}
	}

	port := serverConfig.GetPort()

	address := fmt.Sprintf("%s:%d", serverConfig.GetBindAddress(), port)
	log.Info("Starting http server", "address", address)

	// N.B. We don't use an http2 server because we are using websockets and we were having some issues with
	// http2. Without http2 I'm not sure we can serve grpc.
	hServer := &http.Server{
		// Set timeouts to 0 to disable them because we are using websockets
		WriteTimeout: 0,
		ReadTimeout:  0,
		// We need to wrap it in h2c to support HTTP/2 without TLS
		// TODO(jlewi): Should we only enable h2c if tls isn't enabled?
		Handler: h2c.NewHandler(s.engine, &http2.Server{}),
	}
	// Enable HTTP/2 support
	if err := http2.ConfigureServer(hServer, &http2.Server{}); err != nil {
		return errors.Wrapf(err, "failed to configure http2 server")
	}

	s.hServer = hServer

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return errors.Wrapf(err, "Could not start listener")
	}

	// If TLS is enabled, we need to set up the TLS config
	if serverConfig.TLSConfig != nil {
		tlsConfig, err := tlsbuilder.LoadOrGenerateConfig(s.serverConfig.TLSConfig.CertFile, s.serverConfig.TLSConfig.KeyFile, logZ)
		if err != nil {
			return err
		}
		hServer.TLSConfig = tlsConfig
	}

	go func() {
		// TODO(jlewi): Should we support running TLS and non HTTP on two different ports?
		if serverConfig.TLSConfig != nil {
			log.Info("Starting TLS server", "certFile", serverConfig.TLSConfig.CertFile, "keyFile", serverConfig.TLSConfig.KeyFile)
			// If TLS is enabled, we need to set up the TLS config
			// We can pass empty strings for the keys here because we have configured TLSConfig
			if err := hServer.ServeTLS(lis, "", ""); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					log.Error(err, "There was an error with the http server")
				}
			}
		} else {
			log.Info("Starting non-TLS server")
			if err := hServer.Serve(lis); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					log.Error(err, "There was an error with the http server")
				}
			}
		}
	}()

	// Wait for the shutdown to complete
	// We use a channel to signal when the shutdown method has completed and then return.
	// This is necessary because shutdown() is running in a different go function from hServer.Serve. So if we just
	// relied on hServer.Serve to return and then returned from Run we might still be in the middle of calling shutdown.
	// That's because shutdown calls hServer.Shutdown which causes hserver.Serve to return.
	<-s.shutdownComplete
	return nil
}

func (s *Server) registerServices() error {
	log := zapr.NewLogger(zap.L())

	// Create OIDC instance if configured
	var oidc *iam.OIDC
	var err error
	if s.serverConfig.OIDC != nil {
		oidc, err = iam.NewOIDC(s.serverConfig.OIDC)
		if err != nil {
			return errors.Wrapf(err, "Failed to create OIDC instance")
		}
	}

	// Create auth mux
	mux, err := NewAuthMux(s.serverConfig, oidc)
	if err != nil {
		return errors.Wrapf(err, "Failed to create auth mux")
	}

	// Create the OTEL interceptor
	otelInterceptor, err := otelconnect.NewInterceptor(
		otelconnect.WithTracerProvider(otel.GetTracerProvider()),
		otelconnect.WithMeterProvider(otel.GetMeterProvider()),
	)
	if err != nil {
		return errors.Wrapf(err, "Failed to create otel interceptor")
	}

	interceptors := []connect.Interceptor{otelInterceptor}

	origins := s.serverConfig.CorsOrigins
	if len(origins) == 0 {
		log.Info("No additional CORS origins specified for protected routes")
	} else {
		log.Info("Adding CORS support for protected routes", "origins", origins)
	}

	// Register auth routes if OIDC is configured
	if oidc != nil {
		log.Info("OIDC is configured; registering auth routes")
		if err := RegisterAuthRoutes(oidc, mux); err != nil {
			return errors.Wrapf(err, "Failed to register auth routes")
		}
	} else {
		log.Info("OIDC is not configured; auth routes will not be registered")
	}

	if s.agent != nil {
		aiSvcPath, aiSvcHandler := agentv1connect.NewMessagesServiceHandler(s.agent, connect.WithInterceptors(interceptors...))
		log.Info("Setting up AI service", "path", aiSvcPath)
		// Protect the AI service
		mux.HandleProtected(aiSvcPath, aiSvcHandler, s.checker, api.AgentUserRole)
	} else {
		log.Info("Agent is nil; AI service is disabled")
	}

	if s.parser != nil {
		parserSvcPath, parserSvcHandler := parserv1connect.NewParserServiceHandler(s.parser, connect.WithInterceptors(interceptors...))
		log.Info("Setting up parser service", "path", parserSvcPath)
		mux.HandleProtected(parserSvcPath, parserSvcHandler, s.checker, api.ParserUserRole)
	} else {
		log.Info("Parser is nil; parser service is disabled")
	}

	if s.runner != nil {
		wsHandler := stream.NewWebSocketHandler(s.runner, &iam.AuthContext{
			OIDC:    oidc,
			Checker: s.checker,
			Role:    api.RunnerUserRole,
		})
		// Unprotected WebSockets handler since socket protection is done on the app-level (messages)
		mux.Handle("/ws", otelhttp.NewHandler(http.HandlerFunc(wsHandler.Handler), "/ws"))
		log.Info("Setting up runner websocket handler", "path", "/ws")

		runnerSvcPath, runnerSvcHandler := runnerv2connect.NewRunnerServiceHandler(s.runner, connect.WithInterceptors(interceptors...))
		log.Info("Setting up runner service", "path", runnerSvcPath)
		mux.HandleProtected(runnerSvcPath, runnerSvcHandler, s.checker, api.RunnerUserRole)
	}

	// Health check should be public
	checker := grpchealth.NewStaticChecker()
	mux.Handle(grpchealth.NewHandler(checker))

	mux.HandleFunc("/trailerstest", trailersTest)
	mux.Handle("/metrics", promhttp.Handler())

	if s.registerHandlers != nil {
		log.Info("Registering additional handlers")
		if err := s.registerHandlers(mux, s.checker, interceptors); err != nil {
			return err
		}
	}

	// The single page app is currently only enabled in the agent not the runner.
	if s.agent != nil {
		// Handle the single page app and assets unprotected
		log.Info("Single page app is enabled")
		singlePageApp, err := s.singlePageAppHandler()
		if err != nil {
			return errors.Wrapf(err, "Failed to serve single page app")
		}
		mux.Handle("/", singlePageApp)
	} else {
		log.Info("Single page app is disabled")
	}
	s.engine = mux

	return nil
}

// trailersTest is a function to test returning trailers.
// This method is useful for testing whether a proxy (e.g. envoyproxy) is filtering out trailers.
// If trailers are being filtered out it will prevent grpc and grpcweb from working.
// To test it use
// curl -k -v --http2 https://<server_url>/trailerstest -d ”
// The output should show the trailers in the response headers.
func trailersTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/grpc+proto")
	w.Header().Set("Trailer", "Grpc-Status, Grpc-Message")

	w.WriteHeader(http.StatusOK)

	log := logs.FromContext(r.Context())
	// Write response body
	if _, err := w.Write([]byte("... this is the response ...")); err != nil {
		log.Error(err, "Failed to write response body")
	}

	// Now set trailers
	w.Header().Set("Grpc-Status", "0")
	w.Header().Set("Grpc-Message", "OK")
}

func (s *Server) shutdown() {
	log := zapr.NewLogger(zap.L())
	log.Info("Shutting down the cloud-assistant server")

	if s.hServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.hServer.Shutdown(ctx); err != nil {
			log.Error(err, "Error shutting down http server")
		}
		log.Info("HTTP Server shutdown complete")
	}
	log.Info("Shutdown complete")
	s.shutdownComplete <- true
}

// trapInterrupt shutdowns the server if the appropriate signals are sent
func trapInterrupt(s *Server) {
	log := zapr.NewLogger(zap.L())
	sigs := make(chan os.Signal, 10)
	// Note SIGSTOP and SIGTERM can't be caught
	// We can trap SIGINT which is what ctl-z sends to interrupt the process
	// to interrupt the process
	signal.Notify(sigs, syscall.SIGINT)

	go func() {
		msg := <-sigs
		log.Info("Received signal", "signal", msg)
		s.shutdown()
	}()
}
