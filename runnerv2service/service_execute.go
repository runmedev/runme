package runnerv2service

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	runnerv2 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v2"
	"github.com/runmedev/runme/v3/internal/ulid"
	rcontext "github.com/runmedev/runme/v3/runner/context"
)

func (r *runnerService) Execute(srv runnerv2.RunnerService_ExecuteServer) error {
	logger := r.logger.Named("Execute")

	// Get the initial request.
	req, err := srv.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			logger.Info("client closed the connection while getting initial request; exiting")
			return nil
		}
		logger.Info("failed to receive a request", zap.Error(err))
		return errors.WithStack(err)
	}

	runID := req.GetConfig().GetRunId()
	if runID == "" {
		runID = ulid.GenerateID()
	}
	logger = logger.Named("Execute").With(zap.String("id", runID))
	logger.Debug("received initial request", zap.Any("req", req))

	execInfo := getExecutionInfoFromExecutionRequest(req)
	execInfo.RunID = runID

	ctx := rcontext.WithExecutionInfo(srv.Context(), execInfo)

	// Load the project.
	// TODO(adamb): this should come from the runme.yaml in the future.
	proj, err := convertProtoProjectToProject(req.GetProject())
	if err != nil {
		return err
	}

	// Manage the session.
	session, existed, err := r.getOrCreateSessionFromRequest(req, proj)
	if err != nil {
		return err
	}
	if !existed {
		err := r.sessions.Add(session)
		if err != nil {
			return err
		}
	}

	cfg := req.GetConfig()
	if cfg == nil {
		return errors.New("request config cannot be nil")
	}
	if err := session.SetEnv(ctx, cfg.GetEnv()...); err != nil {
		return err
	}

	exec, err := newExecution(
		cfg,
		proj,
		session,
		logger,
		req.StoreStdoutInEnv,
	)
	if err != nil {
		return err
	}

	sender := newExecuteResponseStreamSender(srv, logger.Named("responseSender"))

	// Start the command and send the initial response with PID.
	if err := exec.Cmd.Start(ctx); err != nil {
		_ = sender.Close()
		return err
	}
	if err := sender.Send(ctx, &runnerv2.ExecuteResponse{
		Pid: &wrapperspb.UInt32Value{Value: uint32(exec.Cmd.Pid())},
	}); err != nil {
		_ = sender.Close()
		return err
	}

	// From the initial request, only the config is used to create a new execution.
	// The rest of fields like InputData, Winsize, Stop are handled in this goroutine,
	// and then the goroutine continues to read the next requests.
	go func(initialReq *runnerv2.ExecuteRequest) {
		for req, err := initialReq, error(nil); ; req, err = srv.Recv() {
			logger.Info("received request", zap.Any("req", req), zap.Error(err))

			switch {
			case err == nil:
				// continue
			case err == io.EOF:
				logger.Info("client closed its send direction; stopping the program")
				if err := exec.Cmd.Signal(os.Interrupt); err != nil {
					logger.Info("failed to stop the command with interrupt signal", zap.Error(err))
				}
				return
			case status.Convert(err).Code() == codes.Canceled || status.Convert(err).Code() == codes.DeadlineExceeded:
				if !exec.Cmd.Running() {
					logger.Info("stream canceled after the process finished; ignoring")
				} else {
					logger.Info("stream canceled while the process is still running; program will be stopped if non-background")
					if err := exec.Cmd.Signal(os.Kill); err != nil {
						logger.Info("failed to stop program with kill signal", zap.Error(err))
					}
				}
				return
			}

			if err := exec.SetWinsize(req.Winsize); err != nil {
				logger.Info("failed to set winsize; ignoring", zap.Error(err))
			}

			if _, err := exec.Write(req.InputData); err != nil {
				logger.Info("failed to write to stdin; ignoring", zap.Error(err))
			}

			if req.Stop > runnerv2.ExecuteStop_EXECUTE_STOP_UNSPECIFIED {
				if err := exec.Stop(req.Stop); err != nil {
					logger.Info("failed to stop program; ignoring", zap.Error(err))
				}
			}
		}
	}(req)

	exitCode, waitErr := exec.Wait(ctx, sender.Send)
	logger.Info("command finished", zap.Int("exitCode", exitCode), zap.Error(waitErr))

	var finalExitCode *wrapperspb.UInt32Value
	if exitCode > -1 {
		finalExitCode = wrapperspb.UInt32(uint32(exitCode))
	}

	currPwd, prevPwd := "", ""
	if v, ok := session.GetEnv("PWD"); ok {
		currPwd = v
	}
	if v, ok := session.GetEnv("OLDPWD"); ok {
		prevPwd = v
	}

	if err := sender.Send(ctx, &runnerv2.ExecuteResponse{
		ExitCode: finalExitCode,
		Pwd: &runnerv2.ExecuteResponse_Pwd{
			Current:  currPwd,
			Previous: prevPwd,
		},
	}); err != nil {
		logger.Info("failed to send exit code", zap.Error(err))
	}
	if err := sender.Close(); err != nil {
		logger.Info("response sender stopped with error", zap.Error(err))
		if waitErr == nil {
			waitErr = err
		}
	}

	return waitErr
}

type executeResponseStreamSender struct {
	responses chan *runnerv2.ExecuteResponse
	done      chan struct{}
	closeOnce sync.Once
	errMu     sync.Mutex
	err       error
}

func newExecuteResponseStreamSender(
	srv runnerv2.RunnerService_ExecuteServer,
	logger *zap.Logger,
) *executeResponseStreamSender {
	s := &executeResponseStreamSender{
		responses: make(chan *runnerv2.ExecuteResponse),
		done:      make(chan struct{}),
	}

	go func() {
		defer close(s.done)
		for resp := range s.responses {
			if err := srv.Send(resp); err != nil {
				logger.Warn("failed to send response", zap.Error(err))
				s.setErr(errors.WithStack(err))
				return
			}
		}
	}()

	return s
}

func (s *executeResponseStreamSender) Send(ctx context.Context, resp *runnerv2.ExecuteResponse) error {
	select {
	case s.responses <- resp:
		return nil
	case <-s.done:
		if err := s.Err(); err != nil {
			return err
		}
		return errors.New("response sender stopped")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *executeResponseStreamSender) Close() error {
	s.closeOnce.Do(func() {
		close(s.responses)
	})
	<-s.done
	return s.Err()
}

func (s *executeResponseStreamSender) setErr(err error) {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	s.err = err
}

func (s *executeResponseStreamSender) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

func getExecutionInfoFromExecutionRequest(req *runnerv2.ExecuteRequest) *rcontext.ExecutionInfo {
	return &rcontext.ExecutionInfo{
		ExecContext: "Execute",
		KnownID:     req.GetConfig().GetKnownId(),
		KnownName:   req.GetConfig().GetKnownName(),
	}
}
