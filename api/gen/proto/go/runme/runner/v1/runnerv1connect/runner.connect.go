// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: runme/runner/v1/runner.proto

package runnerv1connect

import (
	context "context"
	errors "errors"
	http "net/http"
	strings "strings"

	connect "connectrpc.com/connect"

	v1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v1"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// RunnerServiceName is the fully-qualified name of the RunnerService service.
	RunnerServiceName = "runme.runner.v1.RunnerService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// RunnerServiceCreateSessionProcedure is the fully-qualified name of the RunnerService's
	// CreateSession RPC.
	RunnerServiceCreateSessionProcedure = "/runme.runner.v1.RunnerService/CreateSession"
	// RunnerServiceGetSessionProcedure is the fully-qualified name of the RunnerService's GetSession
	// RPC.
	RunnerServiceGetSessionProcedure = "/runme.runner.v1.RunnerService/GetSession"
	// RunnerServiceListSessionsProcedure is the fully-qualified name of the RunnerService's
	// ListSessions RPC.
	RunnerServiceListSessionsProcedure = "/runme.runner.v1.RunnerService/ListSessions"
	// RunnerServiceDeleteSessionProcedure is the fully-qualified name of the RunnerService's
	// DeleteSession RPC.
	RunnerServiceDeleteSessionProcedure = "/runme.runner.v1.RunnerService/DeleteSession"
	// RunnerServiceMonitorEnvStoreProcedure is the fully-qualified name of the RunnerService's
	// MonitorEnvStore RPC.
	RunnerServiceMonitorEnvStoreProcedure = "/runme.runner.v1.RunnerService/MonitorEnvStore"
	// RunnerServiceExecuteProcedure is the fully-qualified name of the RunnerService's Execute RPC.
	RunnerServiceExecuteProcedure = "/runme.runner.v1.RunnerService/Execute"
	// RunnerServiceResolveProgramProcedure is the fully-qualified name of the RunnerService's
	// ResolveProgram RPC.
	RunnerServiceResolveProgramProcedure = "/runme.runner.v1.RunnerService/ResolveProgram"
)

// RunnerServiceClient is a client for the runme.runner.v1.RunnerService service.
type RunnerServiceClient interface {
	CreateSession(context.Context, *connect.Request[v1.CreateSessionRequest]) (*connect.Response[v1.CreateSessionResponse], error)
	GetSession(context.Context, *connect.Request[v1.GetSessionRequest]) (*connect.Response[v1.GetSessionResponse], error)
	ListSessions(context.Context, *connect.Request[v1.ListSessionsRequest]) (*connect.Response[v1.ListSessionsResponse], error)
	DeleteSession(context.Context, *connect.Request[v1.DeleteSessionRequest]) (*connect.Response[v1.DeleteSessionResponse], error)
	MonitorEnvStore(context.Context, *connect.Request[v1.MonitorEnvStoreRequest]) (*connect.ServerStreamForClient[v1.MonitorEnvStoreResponse], error)
	// Execute executes a program. Examine "ExecuteRequest" to explore
	// configuration options.
	//
	// It's a bidirectional stream RPC method. It expects the first
	// "ExecuteRequest" to contain details of a program to execute.
	// Subsequent "ExecuteRequest" should only contain "input_data" as
	// other fields will be ignored.
	Execute(context.Context) *connect.BidiStreamForClient[v1.ExecuteRequest, v1.ExecuteResponse]
	// ResolveProgram resolves variables from a script or a list of commands
	// using the provided sources, which can be a list of environment variables,
	// a session, or a project.
	// For now, the resolved variables are only the exported ones using `export`.
	ResolveProgram(context.Context, *connect.Request[v1.ResolveProgramRequest]) (*connect.Response[v1.ResolveProgramResponse], error)
}

// NewRunnerServiceClient constructs a client for the runme.runner.v1.RunnerService service. By
// default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses,
// and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewRunnerServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) RunnerServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	runnerServiceMethods := v1.File_runme_runner_v1_runner_proto.Services().ByName("RunnerService").Methods()
	return &runnerServiceClient{
		createSession: connect.NewClient[v1.CreateSessionRequest, v1.CreateSessionResponse](
			httpClient,
			baseURL+RunnerServiceCreateSessionProcedure,
			connect.WithSchema(runnerServiceMethods.ByName("CreateSession")),
			connect.WithClientOptions(opts...),
		),
		getSession: connect.NewClient[v1.GetSessionRequest, v1.GetSessionResponse](
			httpClient,
			baseURL+RunnerServiceGetSessionProcedure,
			connect.WithSchema(runnerServiceMethods.ByName("GetSession")),
			connect.WithClientOptions(opts...),
		),
		listSessions: connect.NewClient[v1.ListSessionsRequest, v1.ListSessionsResponse](
			httpClient,
			baseURL+RunnerServiceListSessionsProcedure,
			connect.WithSchema(runnerServiceMethods.ByName("ListSessions")),
			connect.WithClientOptions(opts...),
		),
		deleteSession: connect.NewClient[v1.DeleteSessionRequest, v1.DeleteSessionResponse](
			httpClient,
			baseURL+RunnerServiceDeleteSessionProcedure,
			connect.WithSchema(runnerServiceMethods.ByName("DeleteSession")),
			connect.WithClientOptions(opts...),
		),
		monitorEnvStore: connect.NewClient[v1.MonitorEnvStoreRequest, v1.MonitorEnvStoreResponse](
			httpClient,
			baseURL+RunnerServiceMonitorEnvStoreProcedure,
			connect.WithSchema(runnerServiceMethods.ByName("MonitorEnvStore")),
			connect.WithClientOptions(opts...),
		),
		execute: connect.NewClient[v1.ExecuteRequest, v1.ExecuteResponse](
			httpClient,
			baseURL+RunnerServiceExecuteProcedure,
			connect.WithSchema(runnerServiceMethods.ByName("Execute")),
			connect.WithClientOptions(opts...),
		),
		resolveProgram: connect.NewClient[v1.ResolveProgramRequest, v1.ResolveProgramResponse](
			httpClient,
			baseURL+RunnerServiceResolveProgramProcedure,
			connect.WithSchema(runnerServiceMethods.ByName("ResolveProgram")),
			connect.WithClientOptions(opts...),
		),
	}
}

// runnerServiceClient implements RunnerServiceClient.
type runnerServiceClient struct {
	createSession   *connect.Client[v1.CreateSessionRequest, v1.CreateSessionResponse]
	getSession      *connect.Client[v1.GetSessionRequest, v1.GetSessionResponse]
	listSessions    *connect.Client[v1.ListSessionsRequest, v1.ListSessionsResponse]
	deleteSession   *connect.Client[v1.DeleteSessionRequest, v1.DeleteSessionResponse]
	monitorEnvStore *connect.Client[v1.MonitorEnvStoreRequest, v1.MonitorEnvStoreResponse]
	execute         *connect.Client[v1.ExecuteRequest, v1.ExecuteResponse]
	resolveProgram  *connect.Client[v1.ResolveProgramRequest, v1.ResolveProgramResponse]
}

// CreateSession calls runme.runner.v1.RunnerService.CreateSession.
func (c *runnerServiceClient) CreateSession(ctx context.Context, req *connect.Request[v1.CreateSessionRequest]) (*connect.Response[v1.CreateSessionResponse], error) {
	return c.createSession.CallUnary(ctx, req)
}

// GetSession calls runme.runner.v1.RunnerService.GetSession.
func (c *runnerServiceClient) GetSession(ctx context.Context, req *connect.Request[v1.GetSessionRequest]) (*connect.Response[v1.GetSessionResponse], error) {
	return c.getSession.CallUnary(ctx, req)
}

// ListSessions calls runme.runner.v1.RunnerService.ListSessions.
func (c *runnerServiceClient) ListSessions(ctx context.Context, req *connect.Request[v1.ListSessionsRequest]) (*connect.Response[v1.ListSessionsResponse], error) {
	return c.listSessions.CallUnary(ctx, req)
}

// DeleteSession calls runme.runner.v1.RunnerService.DeleteSession.
func (c *runnerServiceClient) DeleteSession(ctx context.Context, req *connect.Request[v1.DeleteSessionRequest]) (*connect.Response[v1.DeleteSessionResponse], error) {
	return c.deleteSession.CallUnary(ctx, req)
}

// MonitorEnvStore calls runme.runner.v1.RunnerService.MonitorEnvStore.
func (c *runnerServiceClient) MonitorEnvStore(ctx context.Context, req *connect.Request[v1.MonitorEnvStoreRequest]) (*connect.ServerStreamForClient[v1.MonitorEnvStoreResponse], error) {
	return c.monitorEnvStore.CallServerStream(ctx, req)
}

// Execute calls runme.runner.v1.RunnerService.Execute.
func (c *runnerServiceClient) Execute(ctx context.Context) *connect.BidiStreamForClient[v1.ExecuteRequest, v1.ExecuteResponse] {
	return c.execute.CallBidiStream(ctx)
}

// ResolveProgram calls runme.runner.v1.RunnerService.ResolveProgram.
func (c *runnerServiceClient) ResolveProgram(ctx context.Context, req *connect.Request[v1.ResolveProgramRequest]) (*connect.Response[v1.ResolveProgramResponse], error) {
	return c.resolveProgram.CallUnary(ctx, req)
}

// RunnerServiceHandler is an implementation of the runme.runner.v1.RunnerService service.
type RunnerServiceHandler interface {
	CreateSession(context.Context, *connect.Request[v1.CreateSessionRequest]) (*connect.Response[v1.CreateSessionResponse], error)
	GetSession(context.Context, *connect.Request[v1.GetSessionRequest]) (*connect.Response[v1.GetSessionResponse], error)
	ListSessions(context.Context, *connect.Request[v1.ListSessionsRequest]) (*connect.Response[v1.ListSessionsResponse], error)
	DeleteSession(context.Context, *connect.Request[v1.DeleteSessionRequest]) (*connect.Response[v1.DeleteSessionResponse], error)
	MonitorEnvStore(context.Context, *connect.Request[v1.MonitorEnvStoreRequest], *connect.ServerStream[v1.MonitorEnvStoreResponse]) error
	// Execute executes a program. Examine "ExecuteRequest" to explore
	// configuration options.
	//
	// It's a bidirectional stream RPC method. It expects the first
	// "ExecuteRequest" to contain details of a program to execute.
	// Subsequent "ExecuteRequest" should only contain "input_data" as
	// other fields will be ignored.
	Execute(context.Context, *connect.BidiStream[v1.ExecuteRequest, v1.ExecuteResponse]) error
	// ResolveProgram resolves variables from a script or a list of commands
	// using the provided sources, which can be a list of environment variables,
	// a session, or a project.
	// For now, the resolved variables are only the exported ones using `export`.
	ResolveProgram(context.Context, *connect.Request[v1.ResolveProgramRequest]) (*connect.Response[v1.ResolveProgramResponse], error)
}

// NewRunnerServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewRunnerServiceHandler(svc RunnerServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	runnerServiceMethods := v1.File_runme_runner_v1_runner_proto.Services().ByName("RunnerService").Methods()
	runnerServiceCreateSessionHandler := connect.NewUnaryHandler(
		RunnerServiceCreateSessionProcedure,
		svc.CreateSession,
		connect.WithSchema(runnerServiceMethods.ByName("CreateSession")),
		connect.WithHandlerOptions(opts...),
	)
	runnerServiceGetSessionHandler := connect.NewUnaryHandler(
		RunnerServiceGetSessionProcedure,
		svc.GetSession,
		connect.WithSchema(runnerServiceMethods.ByName("GetSession")),
		connect.WithHandlerOptions(opts...),
	)
	runnerServiceListSessionsHandler := connect.NewUnaryHandler(
		RunnerServiceListSessionsProcedure,
		svc.ListSessions,
		connect.WithSchema(runnerServiceMethods.ByName("ListSessions")),
		connect.WithHandlerOptions(opts...),
	)
	runnerServiceDeleteSessionHandler := connect.NewUnaryHandler(
		RunnerServiceDeleteSessionProcedure,
		svc.DeleteSession,
		connect.WithSchema(runnerServiceMethods.ByName("DeleteSession")),
		connect.WithHandlerOptions(opts...),
	)
	runnerServiceMonitorEnvStoreHandler := connect.NewServerStreamHandler(
		RunnerServiceMonitorEnvStoreProcedure,
		svc.MonitorEnvStore,
		connect.WithSchema(runnerServiceMethods.ByName("MonitorEnvStore")),
		connect.WithHandlerOptions(opts...),
	)
	runnerServiceExecuteHandler := connect.NewBidiStreamHandler(
		RunnerServiceExecuteProcedure,
		svc.Execute,
		connect.WithSchema(runnerServiceMethods.ByName("Execute")),
		connect.WithHandlerOptions(opts...),
	)
	runnerServiceResolveProgramHandler := connect.NewUnaryHandler(
		RunnerServiceResolveProgramProcedure,
		svc.ResolveProgram,
		connect.WithSchema(runnerServiceMethods.ByName("ResolveProgram")),
		connect.WithHandlerOptions(opts...),
	)
	return "/runme.runner.v1.RunnerService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case RunnerServiceCreateSessionProcedure:
			runnerServiceCreateSessionHandler.ServeHTTP(w, r)
		case RunnerServiceGetSessionProcedure:
			runnerServiceGetSessionHandler.ServeHTTP(w, r)
		case RunnerServiceListSessionsProcedure:
			runnerServiceListSessionsHandler.ServeHTTP(w, r)
		case RunnerServiceDeleteSessionProcedure:
			runnerServiceDeleteSessionHandler.ServeHTTP(w, r)
		case RunnerServiceMonitorEnvStoreProcedure:
			runnerServiceMonitorEnvStoreHandler.ServeHTTP(w, r)
		case RunnerServiceExecuteProcedure:
			runnerServiceExecuteHandler.ServeHTTP(w, r)
		case RunnerServiceResolveProgramProcedure:
			runnerServiceResolveProgramHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedRunnerServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedRunnerServiceHandler struct{}

func (UnimplementedRunnerServiceHandler) CreateSession(context.Context, *connect.Request[v1.CreateSessionRequest]) (*connect.Response[v1.CreateSessionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("runme.runner.v1.RunnerService.CreateSession is not implemented"))
}

func (UnimplementedRunnerServiceHandler) GetSession(context.Context, *connect.Request[v1.GetSessionRequest]) (*connect.Response[v1.GetSessionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("runme.runner.v1.RunnerService.GetSession is not implemented"))
}

func (UnimplementedRunnerServiceHandler) ListSessions(context.Context, *connect.Request[v1.ListSessionsRequest]) (*connect.Response[v1.ListSessionsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("runme.runner.v1.RunnerService.ListSessions is not implemented"))
}

func (UnimplementedRunnerServiceHandler) DeleteSession(context.Context, *connect.Request[v1.DeleteSessionRequest]) (*connect.Response[v1.DeleteSessionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("runme.runner.v1.RunnerService.DeleteSession is not implemented"))
}

func (UnimplementedRunnerServiceHandler) MonitorEnvStore(context.Context, *connect.Request[v1.MonitorEnvStoreRequest], *connect.ServerStream[v1.MonitorEnvStoreResponse]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("runme.runner.v1.RunnerService.MonitorEnvStore is not implemented"))
}

func (UnimplementedRunnerServiceHandler) Execute(context.Context, *connect.BidiStream[v1.ExecuteRequest, v1.ExecuteResponse]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("runme.runner.v1.RunnerService.Execute is not implemented"))
}

func (UnimplementedRunnerServiceHandler) ResolveProgram(context.Context, *connect.Request[v1.ResolveProgramRequest]) (*connect.Response[v1.ResolveProgramResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("runme.runner.v1.RunnerService.ResolveProgram is not implemented"))
}
