package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
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
		return &owlcmd.CheckResult{
			OK:          false,
			Diagnostics: []string{fmt.Sprintf("Error: %s", parts[len(parts)-1])},
		}, nil
	}

	return &owlcmd.CheckResult{OK: true}, nil
}

func (c *runmeOwlStoreClient) Type(ctx context.Context, req owlcmd.TypeRequest) (*owlcmd.TypeResult, error) {
	if req.SpecPath == "" {
		req.SpecPath = ".env.spec"
	}
	snapshot, err := c.Snapshot(ctx, owlcmd.SnapshotRequest{All: true})
	if err != nil {
		return nil, err
	}

	proposals := make([]owlcmd.TypeProposal, 0, len(snapshot.Envs))
	for _, env := range snapshot.Envs {
		if env.Explicit {
			continue
		}
		suggested, reason := suggestSnapshotPrimitiveType(env)
		proposals = append(proposals, owlcmd.TypeProposal{
			Key:           env.Name,
			CurrentType:   normalizeSnapshotType(env.Type),
			SuggestedType: suggested,
			Confidence:    "heuristic",
			Reason:        reason,
			Description:   descriptionForEnvKey(env.Name),
		})
	}
	sort.SliceStable(proposals, func(i, j int) bool {
		return proposals[i].Key < proposals[j].Key
	})

	rendered := renderDotenvSpecTypeProposals(proposals)
	if req.Output != "" {
		if err := os.WriteFile(req.Output, []byte(rendered), 0o600); err != nil {
			return nil, err
		}
	}
	if req.Fix {
		if err := appendDotenvSpecTypeProposals(req.SpecPath, rendered); err != nil {
			return nil, err
		}
	}

	return &owlcmd.TypeResult{Proposals: proposals, Rendered: rendered}, nil
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

func suggestSnapshotPrimitiveType(env owlcmd.SnapshotEnv) (string, string) {
	upper := strings.ToUpper(env.Name)
	switch {
	case strings.Contains(upper, "PASSWORD"),
		strings.Contains(upper, "SECRET"),
		strings.Contains(upper, "TOKEN"),
		strings.Contains(upper, "API_KEY"),
		strings.Contains(upper, "PRIVATE_KEY"):
		return "core/secret", "key name suggests sensitive value"
	case upper == "URL" || strings.HasSuffix(upper, "_URL") || strings.Contains(upper, "URL_"):
		return "core/url", "key name suggests URL"
	case upper == "HOST" || strings.HasSuffix(upper, "_HOST") || strings.Contains(upper, "HOST_"):
		return "core/host", "key name suggests host"
	case upper == "PORT" || strings.HasSuffix(upper, "_PORT") || strings.Contains(upper, "PORT_"):
		return "core/port", "key name suggests port"
	default:
		return "core/plain", "default primitive type"
	}
}

func renderDotenvSpecTypeProposals(proposals []owlcmd.TypeProposal) string {
	if len(proposals) == 0 {
		return ""
	}
	var b strings.Builder
	for _, proposal := range proposals {
		_, _ = b.WriteString(proposal.Key)
		_, _ = b.WriteString("=")
		_, _ = b.WriteString(quoteDotenvSpecDescription(proposal.Description))
		_, _ = b.WriteString(" # ")
		_, _ = b.WriteString(dotenvSpecTypeName(proposal.SuggestedType))
		if proposal.Required {
			_ = b.WriteByte('!')
		}
		_ = b.WriteByte('\n')
	}
	return b.String()
}

func appendDotenvSpecTypeProposals(path string, rendered string) error {
	if rendered == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var b strings.Builder
	_, _ = b.Write(raw)
	if len(raw) > 0 && !strings.HasSuffix(string(raw), "\n") {
		_ = b.WriteByte('\n')
	}
	_, _ = b.WriteString(rendered)
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func dotenvSpecTypeName(typeID string) string {
	switch normalizeSnapshotType(typeID) {
	case "core/secret":
		return "Secret"
	case "core/url":
		return "Url"
	case "core/host":
		return "Host"
	case "core/port":
		return "Port"
	case "core/plain":
		return "Plain"
	default:
		return "Opaque"
	}
}

func normalizeSnapshotType(typeID string) string {
	return strings.TrimPrefix(typeID, "https://owl.runme.dev/v1/types/")
}

func quoteDotenvSpecDescription(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

func descriptionForEnvKey(key string) string {
	words := strings.Split(strings.ToLower(strings.ReplaceAll(key, "_", " ")), " ")
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
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
			Name:        env.GetName(),
			Value:       env.GetResolvedValue(),
			Description: env.GetDescription(),
			Type:        env.GetSpec(),
			Source:      env.GetOrigin(),
			Explicit:    snapshotExplicitFromProto(env),
			Visibility:  snapshotVisibilityFromProto(env.GetStatus()),
			Diagnostics: snapshotDiagnosticsFromProto(env.GetErrors()),
		})
	}
	return result
}

func snapshotExplicitFromProto(env *runnerv1.MonitorEnvStoreResponseSnapshot_SnapshotEnv) bool {
	if env.GetDescription() != "" || env.GetIsRequired() {
		return true
	}
	spec := env.GetSpec()
	return spec != "" && spec != "Opaque" && spec != "https://owl.runme.dev/v1/types/core/opaque"
}

func snapshotVisibilityFromProto(status runnerv1.MonitorEnvStoreResponseSnapshot_Status) string {
	switch status {
	case runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_HIDDEN:
		return "hidden"
	case runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_MASKED:
		return "masked"
	case runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_LITERAL:
		return "literal"
	default:
		return "unspecified"
	}
}

func snapshotDiagnosticsFromProto(errors []*runnerv1.MonitorEnvStoreResponseSnapshot_Error) []string {
	if len(errors) == 0 {
		return nil
	}
	result := make([]string, 0, len(errors))
	for _, err := range errors {
		result = append(result, err.GetMessage())
	}
	return result
}
