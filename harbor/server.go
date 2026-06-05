package harbor

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	harborv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/harbor/v1"
	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	"github.com/runmedev/runme/v3/command"
	"github.com/runmedev/runme/v3/pkg/agent/runme/stream"
)

const (
	ProtocolName    = "runme.harbor.stdio"
	ProtocolVersion = "v1"
)

type Options struct {
	Logger     *zap.Logger
	TapFactory stream.TapFactory
}

type Server struct {
	tapFactory stream.TapFactory
	factory    command.Factory

	mu       sync.Mutex
	root     string
	tempRoot bool
	env      []string
}

func NewServer(opts Options) (*Server, error) {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &Server{
		tapFactory: opts.TapFactory,
		factory: command.NewFactory(
			command.WithLogger(logger),
			command.WithProcessLifecycle(command.ProcessLifecycleLinked),
		),
		root: root,
	}, nil
}

func (s *Server) ServeStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 32*1024*1024)

	writer := bufio.NewWriter(stdout)
	defer writer.Flush()

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var req harborv1.Request
		if err := protojson.Unmarshal(scanner.Bytes(), &req); err != nil {
			if err := writeResponse(writer, errorResponse("", "invalid_json", err.Error())); err != nil {
				return err
			}
			continue
		}
		if err := writeResponse(writer, s.Handle(ctx, &req)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func writeResponse(w *bufio.Writer, resp *harborv1.Response) error {
	data, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := w.WriteByte('\n'); err != nil {
		return err
	}
	return w.Flush()
}

func (s *Server) Handle(ctx context.Context, req *harborv1.Request) *harborv1.Response {
	if req.GetId() == "" {
		return errorResponse("", "invalid_argument", "request id is required")
	}

	switch payload := req.GetPayload().(type) {
	case *harborv1.Request_Preflight:
		return &harborv1.Response{
			Id: req.GetId(),
			Payload: &harborv1.Response_Preflight{Preflight: &harborv1.PreflightResponse{
				Protocol: ProtocolName,
				Version:  ProtocolVersion,
				Capabilities: []string{
					"preflight",
					"start",
					"stop",
					"exec",
					"upload_file",
					"download_file",
					"upload_directory",
					"download_directory",
				},
			}},
		}
	case *harborv1.Request_Start:
		return s.handleStart(req.GetId(), payload.Start)
	case *harborv1.Request_Stop:
		return s.handleStop(req.GetId())
	case *harborv1.Request_Exec:
		return s.handleExec(ctx, req.GetId(), payload.Exec)
	case *harborv1.Request_UploadFile:
		return s.handleUploadFile(req.GetId(), payload.UploadFile)
	case *harborv1.Request_DownloadFile:
		return s.handleDownloadFile(req.GetId(), payload.DownloadFile)
	case *harborv1.Request_UploadDirectory:
		return s.handleUploadDirectory(req.GetId(), payload.UploadDirectory)
	case *harborv1.Request_DownloadDirectory:
		return s.handleDownloadDirectory(req.GetId(), payload.DownloadDirectory)
	default:
		return errorResponse(req.GetId(), "invalid_argument", "request payload is required")
	}
}

func (s *Server) handleStart(id string, req *harborv1.StartRequest) *harborv1.Response {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.cleanupLocked(); err != nil {
		return errorResponse(id, "internal", err.Error())
	}

	root := req.GetRoot()
	tempRoot := false
	if root == "" {
		var err error
		root, err = os.MkdirTemp("", "runme-harbor-*")
		if err != nil {
			return errorResponse(id, "internal", err.Error())
		}
		tempRoot = true
	} else {
		abs, err := filepath.Abs(root)
		if err != nil {
			return errorResponse(id, "invalid_argument", err.Error())
		}
		root = abs
		if err := os.MkdirAll(root, 0o755); err != nil {
			return errorResponse(id, "internal", err.Error())
		}
	}

	s.root = root
	s.tempRoot = tempRoot
	s.env = append([]string(nil), req.GetEnv()...)

	return &harborv1.Response{
		Id:      id,
		Payload: &harborv1.Response_Start{Start: &harborv1.StartResponse{Root: root}},
	}
}

func (s *Server) handleStop(id string) *harborv1.Response {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.cleanupLocked(); err != nil {
		return errorResponse(id, "internal", err.Error())
	}
	return &harborv1.Response{Id: id, Payload: &harborv1.Response_Stop{Stop: &harborv1.StopResponse{}}}
}

func (s *Server) cleanupLocked() error {
	if s.tempRoot && s.root != "" {
		if err := os.RemoveAll(s.root); err != nil {
			return err
		}
	}
	s.tempRoot = false
	s.env = nil
	return nil
}

func (s *Server) handleExec(ctx context.Context, id string, req *harborv1.ExecRequest) *harborv1.Response {
	if strings.TrimSpace(req.GetCommand()) == "" {
		return errorResponse(id, "invalid_argument", "exec command is required")
	}
	cwd, err := s.resolvePath(req.GetCwd())
	if err != nil {
		return errorResponse(id, "invalid_argument", err.Error())
	}

	runID := fmt.Sprintf("harbor_%s", ulid.Make().String())
	var tap stream.StreamTap
	if s.tapFactory != nil {
		tap = s.tapFactory(runID)
	}
	if tap != nil {
		tap.CommandStart(req.GetCommand(), cwd)
		defer tap.Close()
	}

	result, err := s.execute(ctx, runID, req.GetCommand(), cwd, req.GetEnv(), tap)
	if tap != nil {
		tap.CommandEnd(result.ExitCode)
	}
	if err != nil && result.ExitCode == 0 {
		return errorResponse(id, "exec_failed", err.Error())
	}
	return &harborv1.Response{
		Id: id,
		Payload: &harborv1.Response_Exec{Exec: &harborv1.ExecResponse{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: int32(result.ExitCode),
		}},
	}
}

type ExecResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

func (s *Server) Execute(ctx context.Context, command, cwd string, env []string) (ExecResult, error) {
	resolvedCWD, err := s.resolvePath(cwd)
	if err != nil {
		return ExecResult{}, err
	}
	runID := fmt.Sprintf("harbor_%s", ulid.Make().String())
	return s.execute(ctx, runID, command, resolvedCWD, env, nil)
}

func (s *Server) execute(ctx context.Context, runID, command, cwd string, env []string, tap stream.StreamTap) (ExecResult, error) {
	s.mu.Lock()
	baseEnv := append([]string(nil), s.env...)
	s.mu.Unlock()

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	execStream := newExecuteStream(streamCtx, &runnerv2.ExecuteRequest{
		Config: &runnerv2.ProgramConfig{
			ProgramName: "bash",
			Arguments:   []string{"-c", command},
			Directory:   cwd,
			Env:         append(baseEnv, env...),
			LanguageId:  "sh",
			Interactive: false,
			Mode:        runnerv2.CommandMode_COMMAND_MODE_INLINE,
			KnownId:     "harbor",
			KnownName:   "harbor-exec",
			RunId:       runID,
		},
	})

	err := execStream.Run(s.factory)
	result := execStream.result()
	if tap != nil {
		if len(result.Stdout) > 0 {
			tap.Output(result.Stdout)
		}
		if len(result.Stderr) > 0 {
			tap.Stderr(result.Stderr)
		}
	}
	return result, err
}

func (s *Server) handleUploadFile(id string, req *harborv1.UploadFileRequest) *harborv1.Response {
	target, err := s.resolvePath(req.GetPath())
	if err != nil {
		return errorResponse(id, "invalid_argument", err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return errorResponse(id, "internal", err.Error())
	}
	mode := fileMode(req.GetMode())
	if err := os.WriteFile(target, req.GetData(), mode); err != nil {
		return errorResponse(id, "internal", err.Error())
	}
	return &harborv1.Response{
		Id:      id,
		Payload: &harborv1.Response_UploadFile{UploadFile: &harborv1.UploadFileResponse{BytesWritten: uint64(len(req.GetData()))}},
	}
}

func (s *Server) handleDownloadFile(id string, req *harborv1.DownloadFileRequest) *harborv1.Response {
	target, err := s.resolvePath(req.GetPath())
	if err != nil {
		return errorResponse(id, "invalid_argument", err.Error())
	}
	info, err := os.Stat(target)
	if err != nil {
		return errorResponse(id, "not_found", err.Error())
	}
	if info.IsDir() {
		return errorResponse(id, "invalid_argument", "path is a directory")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return errorResponse(id, "internal", err.Error())
	}
	return &harborv1.Response{
		Id: id,
		Payload: &harborv1.Response_DownloadFile{DownloadFile: &harborv1.DownloadFileResponse{
			Data: data,
			Mode: uint32(info.Mode().Perm()),
		}},
	}
}

func (s *Server) handleUploadDirectory(id string, req *harborv1.UploadDirectoryRequest) *harborv1.Response {
	base, err := s.resolvePath(req.GetPath())
	if err != nil {
		return errorResponse(id, "invalid_argument", err.Error())
	}
	var written uint32
	for _, file := range req.GetFiles() {
		target, err := safeJoin(base, file.GetPath())
		if err != nil {
			return errorResponse(id, "invalid_argument", err.Error())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return errorResponse(id, "internal", err.Error())
		}
		if err := os.WriteFile(target, file.GetData(), fileMode(file.GetMode())); err != nil {
			return errorResponse(id, "internal", err.Error())
		}
		written++
	}
	return &harborv1.Response{
		Id:      id,
		Payload: &harborv1.Response_UploadDirectory{UploadDirectory: &harborv1.UploadDirectoryResponse{FilesWritten: written}},
	}
}

func (s *Server) handleDownloadDirectory(id string, req *harborv1.DownloadDirectoryRequest) *harborv1.Response {
	base, err := s.resolvePath(req.GetPath())
	if err != nil {
		return errorResponse(id, "invalid_argument", err.Error())
	}
	var files []*harborv1.FileEntry
	err = filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		files = append(files, &harborv1.FileEntry{
			Path: filepath.ToSlash(rel),
			Data: data,
			Mode: uint32(info.Mode().Perm()),
		})
		return nil
	})
	if err != nil {
		return errorResponse(id, "internal", err.Error())
	}
	return &harborv1.Response{
		Id:      id,
		Payload: &harborv1.Response_DownloadDirectory{DownloadDirectory: &harborv1.DownloadDirectoryResponse{Files: files}},
	}
}

func (s *Server) resolvePath(path string) (string, error) {
	s.mu.Lock()
	root := s.root
	s.mu.Unlock()

	if root == "" {
		return "", errors.New("runtime root is not configured")
	}
	if strings.TrimSpace(path) == "" {
		return root, nil
	}
	if filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return "", err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("path %q escapes root %q", path, root)
		}
		return abs, nil
	}
	return safeJoin(root, path)
}

func safeJoin(base, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path %q must be relative", rel)
	}
	clean := filepath.Clean(rel)
	if clean == "." {
		return base, nil
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes root", rel)
	}
	return filepath.Join(base, clean), nil
}

func fileMode(mode uint32) os.FileMode {
	if mode == 0 {
		return 0o644
	}
	return os.FileMode(mode).Perm()
}

func errorResponse(id, code, message string) *harborv1.Response {
	return &harborv1.Response{
		Id:    id,
		Error: &harborv1.Error{Code: code, Message: message},
	}
}

type executeStream struct {
	ctx     context.Context
	initial *runnerv2.ExecuteRequest

	mu       sync.Mutex
	stdout   bytes.Buffer
	stderr   bytes.Buffer
	exitCode int
}

func newExecuteStream(ctx context.Context, initial *runnerv2.ExecuteRequest) *executeStream {
	return &executeStream{ctx: ctx, initial: initial}
}

func (s *executeStream) Run(factory command.Factory) error {
	cmd, err := factory.Build(s.initial.GetConfig(), command.CommandOptions{
		NoShell: true,
		Stdout:  &s.stdout,
		Stderr:  &s.stderr,
	})
	if err != nil {
		return err
	}
	if err := cmd.Start(s.ctx); err != nil {
		return err
	}
	err = cmd.Wait(s.ctx)
	s.exitCode = commandExitCode(err)
	return err
}

func (s *executeStream) result() ExecResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ExecResult{
		Stdout:   append([]byte(nil), s.stdout.Bytes()...),
		Stderr:   append([]byte(nil), s.stderr.Bytes()...),
		ExitCode: s.exitCode,
	}
}

func commandExitCode(err error) int {
	var exit interface{ ExitCode() int }
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	if err != nil {
		return 1
	}
	return 0
}
