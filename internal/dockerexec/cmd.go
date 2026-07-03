package dockerexec

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Cmd struct {
	Path   string
	Args   []string
	Env    []string
	Dir    string
	TTY    bool
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Process      *Process
	ProcessState *ProcessState

	// Fields set by the constructor.
	ctx    context.Context
	docker *Docker
	name   string

	// containerID will be set in Start().
	containerID string
	autoRemove  bool

	ioErrC chan error

	waitErrC  <-chan error
	waitRespC <-chan container.WaitResponse

	logger *zap.Logger
}

func (c *Cmd) Start() error {
	containerConfig := &container.Config{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          c.TTY,
		OpenStdin:    true,
		StdinOnce:    true,
		Env:          c.Env,
		Entrypoint:   []string{c.Path},
		Cmd:          c.Args,
		Image:        c.docker.image,
		WorkingDir:   "/workspace",
	}
	c.autoRemove = !c.docker.debug

	hostConfig := &container.HostConfig{
		RestartPolicy:  container.RestartPolicy{Name: container.RestartPolicyDisabled},
		AutoRemove:     false,
		ConsoleSize:    [2]uint{80, 24},
		ReadonlyRootfs: true,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: c.Dir,
				Target: "/workspace",
			},
		},
	}
	networkConfig := &network.NetworkingConfig{}
	platformConfig := &v1.Platform{}

	resp, err := c.docker.client.ContainerCreate(
		c.ctx,
		containerConfig,
		hostConfig,
		networkConfig,
		platformConfig,
		c.name,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	c.containerID = resp.ID

	// Wait() owns container cleanup, but callers do not call Wait() when Start()
	// returns an error, so remove the container here on any failed start.
	started := false
	defer func() {
		if !started {
			if rmErr := c.removeContainer(); rmErr != nil {
				c.logger.Warn("failed to remove container after Start error", zap.Error(rmErr))
			}
		}
	}()

	c.waitRespC, c.waitErrC = c.docker.client.ContainerWait(
		c.ctx,
		c.containerID,
		// Wait for the "next exit" state; the container is removed manually by
		// Wait()/removeContainer() rather than by daemon auto-remove.
		container.WaitConditionNextExit,
	)

	hijack, err := c.docker.client.ContainerAttach(
		c.ctx,
		c.containerID,
		container.AttachOptions{
			Stream: true,
			Stdin:  true,
			Stdout: true,
			Stderr: true,
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	err = c.docker.client.ContainerStart(
		c.ctx,
		resp.ID,
		container.StartOptions{},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	c.ioErrC = make(chan error, 2)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		if c.TTY {
			_, err = io.Copy(c.Stdout, hijack.Reader)
		} else {
			_, err = stdcopy.StdCopy(c.Stdout, c.Stderr, hijack.Reader)
		}
		c.ioErrC <- errors.WithStack(err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		if c.Stdin != nil {
			_, err = io.Copy(hijack.Conn, c.Stdin)
		}
		c.ioErrC <- errors.WithStack(err)
	}()

	go func() {
		wg.Wait()
		close(c.ioErrC)
	}()

	inspect, err := c.docker.client.ContainerInspect(c.ctx, c.containerID)
	if err != nil {
		return errors.WithStack(err)
	}

	c.Process = &Process{
		Pid: inspect.State.Pid,
	}

	started = true
	return nil
}

func (c *Cmd) Signal() error {
	err := c.docker.client.ContainerStop(
		c.ctx,
		c.containerID,
		container.StopOptions{
			Signal:  "SIGTERM",
			Timeout: nil,
		},
	)
	return errors.WithStack(err)
}

func (c *Cmd) Wait() error {
	var err error

	select {
	case resp := <-c.waitRespC:
		c.logger.Debug("container wait response", zap.Any("response", resp))
		c.ProcessState = &ProcessState{
			ExitCode: int(resp.StatusCode),
		}
		if resp.Error != nil {
			c.ProcessState.ErrorMessage = resp.Error.Message
		}

		if c.ProcessState.ExitCode > 0 {
			err = errors.Errorf("exit code %d due to error %q", c.ProcessState.ExitCode, c.ProcessState.ErrorMessage)
		}
	case err = <-c.waitErrC:
		err = errors.WithStack(err)
	}

	for errIO := range c.ioErrC {
		if err == nil {
			err = errIO
		} else {
			c.logger.Info("ignoring IO error as there was an earlier error", zap.Error(errIO))
		}
	}

	// Wait owns cleanup for non-debug Docker commands. Callers must call Wait
	// after Start so short-lived containers can still be inspected for process
	// metadata before they are removed.
	if errCleanup := c.removeContainer(); errCleanup != nil {
		if err == nil {
			err = errCleanup
		} else {
			c.logger.Info("ignoring container cleanup error as there was an earlier error", zap.Error(errCleanup))
		}
	}

	return err
}

func (c *Cmd) removeContainer() error {
	if !c.autoRemove || c.containerID == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := c.docker.client.ContainerRemove(
		ctx,
		c.containerID,
		container.RemoveOptions{
			Force: true,
		},
	)
	return errors.WithStack(err)
}

type Process struct {
	Pid int
}

type ProcessState struct {
	ErrorMessage string
	ExitCode     int
}
