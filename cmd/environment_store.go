package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	owlcmd "github.com/runmedev/owl/cmd"

	runnerv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v1"
	runmetls "github.com/runmedev/runme/v3/internal/tls"
	"github.com/runmedev/runme/v3/runner/client"
)

type runmeOwlStoreClient struct {
	storeFlags    envStoreFlags
	checkAddr     string
	getRunnerOpts func() ([]client.RunnerOption, error)
	stdin         io.Reader
	stdout        io.Writer
	stderr        io.Writer
}

func (c *runmeOwlStoreClient) Snapshot(ctx context.Context, _ owlcmd.SnapshotRequest) (*owlcmd.SnapshotResult, error) {
	runnerClient, closeConn, err := c.runnerClient()
	if err != nil {
		return nil, err
	}
	defer closeConn()

	sessionID, err := c.sessionID(ctx, runnerClient)
	if err != nil {
		return nil, err
	}

	req := &runnerv1.MonitorEnvStoreRequest{
		Session: &runnerv1.Session{Id: sessionID},
	}
	meClient, err := runnerClient.MonitorEnvStore(ctx, req)
	if err != nil {
		return nil, err
	}

	var msg runnerv1.MonitorEnvStoreResponse
	if err := meClient.RecvMsg(&msg); err != nil {
		return nil, err
	}

	msgData, ok := msg.Data.(*runnerv1.MonitorEnvStoreResponse_Snapshot)
	if !ok {
		return &owlcmd.SnapshotResult{}, nil
	}

	return &owlcmd.SnapshotResult{Envs: snapshotEnvsFromProto(msgData.Snapshot.Envs)}, nil
}

func (c *runmeOwlStoreClient) Source(ctx context.Context, _ owlcmd.SourceRequest) (*owlcmd.SourceResult, error) {
	runnerClient, closeConn, err := c.runnerClient()
	if err != nil {
		return nil, err
	}
	defer closeConn()

	sessionID, err := c.sessionID(ctx, runnerClient)
	if err != nil {
		return nil, err
	}

	resp, err := runnerClient.GetSession(ctx, &runnerv1.GetSessionRequest{Id: sessionID})
	if err != nil {
		return nil, err
	}

	return &owlcmd.SourceResult{Envs: resp.Session.Envs}, nil
}

func (c *runmeOwlStoreClient) Check(ctx context.Context, _ owlcmd.CheckRequest) (*owlcmd.CheckResult, error) {
	project, err := getProject()
	if err != nil {
		return nil, err
	}

	if c.getRunnerOpts == nil {
		return nil, errors.New("runner options are not configured")
	}
	runnerOpts, err := c.getRunnerOpts()
	if err != nil {
		return nil, err
	}

	runnerOpts = append(
		runnerOpts,
		client.WithinShellMaybe(),
		client.WithStdin(c.stdin),
		client.WithCleanupSession(true),
		client.WithStdout(c.stdout),
		client.WithStderr(c.stderr),
		client.WithProject(project),
		client.WithEnvStoreType(runnerv1.SessionEnvStoreType_SESSION_ENV_STORE_TYPE_OWL),
	)

	_, err = client.NewRemoteRunner(ctx, c.checkAddr, runnerOpts...)
	if err != nil {
		errStr := err.Error()
		parts := strings.Split(errStr, "Unknown desc = ")
		return &owlcmd.CheckResult{Message: fmt.Sprintf("Error: %s", parts[len(parts)-1])}, nil
	}

	return &owlcmd.CheckResult{Message: "Success"}, nil
}

func (c *runmeOwlStoreClient) runnerClient() (runnerv1.RunnerServiceClient, func(), error) {
	tlsConfig, err := runmetls.LoadClientConfigFromDir(c.storeFlags.tlsDir)
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.NewClient(
		c.storeFlags.serverAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to connect")
	}

	return runnerv1.NewRunnerServiceClient(conn), func() { _ = conn.Close() }, nil
}

func (c *runmeOwlStoreClient) sessionID(ctx context.Context, runnerClient runnerv1.RunnerServiceClient) (string, error) {
	sessionID := c.storeFlags.sessionID
	if strings.ToLower(c.storeFlags.sessionStrategy) != "recent" {
		return sessionID, nil
	}

	resp, err := runnerClient.ListSessions(ctx, &runnerv1.ListSessionsRequest{})
	if err != nil {
		return "", err
	}
	l := len(resp.Sessions)
	if l == 0 {
		return "", errors.New("no sessions found")
	}
	return resp.Sessions[l-1].Id, nil
}

func snapshotEnvsFromProto(envs []*runnerv1.MonitorEnvStoreResponseSnapshot_SnapshotEnv) []owlcmd.SnapshotEnv {
	result := make([]owlcmd.SnapshotEnv, 0, len(envs))
	for _, env := range envs {
		result = append(result, owlcmd.SnapshotEnv{
			Name:          env.GetName(),
			OriginalValue: env.GetOriginalValue(),
			ResolvedValue: env.GetResolvedValue(),
			Description:   env.GetDescription(),
			Spec:          env.GetSpec(),
			Origin:        env.GetOrigin(),
			Status:        snapshotStatusFromProto(env.GetStatus()),
			UpdateTime:    env.GetUpdateTime(),
		})
	}
	return result
}

func snapshotStatusFromProto(status runnerv1.MonitorEnvStoreResponseSnapshot_Status) string {
	switch status {
	case runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_HIDDEN:
		return "HIDDEN"
	case runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_MASKED:
		return "MASKED"
	case runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_LITERAL:
		return "LITERAL"
	default:
		return "UNSPECIFIED"
	}
}
