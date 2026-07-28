package autoconfig

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/runmedev/runme/v3/internal/config"
	"github.com/runmedev/runme/v3/internal/server"
	"github.com/runmedev/runme/v3/runnerv2client"
)

func TestInvokeForCommand_Config(t *testing.T) {
	builder := NewBuilder()
	configRootFS := fstest.MapFS{
		"runme.yaml": {
			// It's ok that README.md does not exist as it's not used in this test.
			Data: []byte(fmt.Sprintf("version: v1alpha1\nproject:\n  filename: %s\n", "README.md")),
		},
	}
	err := builder.Decorate(
		func() (*config.Loader, error) {
			return config.NewLoader([]string{"runme.yaml"}, configRootFS), nil
		},
	)
	require.NoError(t, err)
	err = builder.Invoke(func(*config.Config) error { return nil })
	require.NoError(t, err)
}

func TestInvokeForCommand_ServerClient(t *testing.T) {
	t.Run("NoServerInConfig", func(t *testing.T) {
		builder := NewBuilder()
		temp := t.TempDir()

		err := os.WriteFile(filepath.Join(temp, "README.md"), []byte("Hello, World!"), 0o644)
		require.NoError(t, err)

		configRootFS := fstest.MapFS{
			"runme.yaml": {
				Data: []byte(`version: v1alpha1
project:
  filename: ` + filepath.Join(temp, "README.md") + `
server: null
`),
			},
		}
		err = builder.Decorate(
			func() (*config.Loader, error) {
				return config.NewLoader([]string{"runme.yaml"}, configRootFS), nil
			},
		)
		require.NoError(t, err)

		err = builder.Invoke(func(
			server *server.Server,
			client *runnerv2client.Client,
		) error {
			require.Nil(t, server)
			require.Nil(t, client)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("ServerInConfigWithoutTLS", func(t *testing.T) {
		builder := NewBuilder()
		temp := t.TempDir()

		readmePath := filepath.Join(temp, "README.md")
		err := os.WriteFile(readmePath, []byte("Hello, World!"), 0o644)
		require.NoError(t, err)

		configRootFS := testServerConfig(t, readmePath, false)
		err = builder.Decorate(
			func() (*config.Loader, error) {
				return config.NewLoader([]string{"runme.yaml"}, configRootFS), nil
			},
		)
		require.NoError(t, err)

		err = builder.Invoke(func(
			server *server.Server,
			client *runnerv2client.Client,
		) error {
			require.NotNil(t, server)
			require.NotNil(t, client)

			var g errgroup.Group

			g.Go(func() error {
				return server.Serve()
			})

			g.Go(func() error {
				defer server.Shutdown()
				return checkHealth(client)
			})

			return g.Wait()
		})
		require.NoError(t, err)
	})

	t.Run("ServerInConfigWithTLS", func(t *testing.T) {
		builder := NewBuilder()
		temp := t.TempDir()

		readmePath := filepath.Join(temp, "README.md")
		err := os.WriteFile(readmePath, []byte("Hello, World!"), 0o644)
		require.NoError(t, err)

		configRootFS := testServerConfig(t, readmePath, true)
		err = builder.Decorate(
			func() (*config.Loader, error) {
				return config.NewLoader([]string{"runme.yaml"}, configRootFS), nil
			},
		)
		require.NoError(t, err)

		err = builder.Invoke(func(
			server *server.Server,
			client *runnerv2client.Client,
		) error {
			require.NotNil(t, server)
			require.NotNil(t, client)

			var g errgroup.Group

			g.Go(func() error {
				return server.Serve()
			})

			g.Go(func() error {
				defer server.Shutdown()
				return errors.WithMessage(checkHealth(client), "failed to check health")
			})

			return g.Wait()
		})
		require.NoError(t, err)
	})
}

func testServerConfig(t *testing.T, readmePath string, tlsEnabled bool) fstest.MapFS {
	t.Helper()

	return fstest.MapFS{
		"runme.yaml": {
			Data: []byte(fmt.Sprintf(`version: v1alpha1
project:
  filename: %s
server:
  address: %s
  tls:
    enabled: %t
  max_message_size: 33554432
`, readmePath, testServerAddress(t), tlsEnabled)),
		},
	}
}

func testServerAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, listener.Close())
	}()

	return listener.Addr().String()
}

func checkHealth(client healthv1.HealthClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		resp *healthv1.HealthCheckResponse
		err  error
	)

	delay := 50 * time.Millisecond
	const maxDelay = 500 * time.Millisecond

	for {
		attemptCtx, attemptCancel := context.WithTimeout(ctx, time.Second)
		resp, err = client.Check(attemptCtx, &healthv1.HealthCheckRequest{})
		attemptCancel()
		if err == nil && resp.GetStatus() == healthv1.HealthCheckResponse_SERVING {
			return nil
		}

		jitter := time.Duration(rand.Int64N(int64(delay / 2)))
		timer := time.NewTimer(delay + jitter)

		select {
		case <-ctx.Done():
			timer.Stop()
			if err != nil {
				return errors.WithMessage(err, "timed out waiting for health")
			}
			return errors.Errorf("timed out waiting for health: status = %s", resp.GetStatus())
		case <-timer.C:
		}

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}
