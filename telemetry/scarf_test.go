package telemetry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewReporter(t *testing.T) {
	t.Run("Scarf", func(t *testing.T) {
		t.Setenv(envDoNotTrack, "false")
		t.Setenv(envScarfNoAnalytics, "false")

		reporter := newReporter(zap.NewNop())
		require.IsType(t, &scarfReporter{}, reporter)
	})

	t.Run("DO_NOT_TRACK", func(t *testing.T) {
		t.Setenv(envDoNotTrack, "true")
		reporter := newReporter(zap.NewNop())
		require.IsType(t, noopReporter{}, reporter)
	})

	t.Run("SCARF_NO_ANALYTICS", func(t *testing.T) {
		t.Setenv(envScarfNoAnalytics, "true")
		reporter := newReporter(zap.NewNop())
		require.IsType(t, noopReporter{}, reporter)
	})
}

func TestNoopReporter(t *testing.T) {
	reporter := noopReporter{}
	require.False(t, reporter.report(event{client: clientCLI, name: cliInvocationEvent}))
}

func TestScarfReporter(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "https", req.URL.Scheme)
		require.Equal(t, "home.runme.dev", req.URL.Host)
		require.Equal(t, "/CLI", req.URL.Path)
		require.Equal(t, cliInvocationEvent, req.URL.Query().Get("event"))

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		}, nil
	})

	reporter := &scarfReporter{
		logger: zap.NewNop(),
		client: &http.Client{Transport: transport},
	}

	err := reporter.send(event{
		client: clientCLI,
		name:   cliInvocationEvent,
		props:  map[string]string{"component": "cli"},
	})
	require.NoError(t, err)
}

func TestBuildURL(t *testing.T) {
	t.Parallel()

	t.Run("KernelFull", func(t *testing.T) {
		t.Parallel()

		event := kernelStartupEventFromEnv(createLookup(map[string]string{
			"TELEMETRY_EXTNAME":    "stateful.runme",
			"TELEMETRY_EXTVERSION": "3.7.7-dev.10",
			"TELEMETRY_REMOTENAME": "none",
			"TELEMETRY_APPNAME":    "Visual Studio Code",
			"TELEMETRY_PRODUCT":    "desktop",
			"TELEMETRY_PLATFORM":   "darwin_arm64",
			"TELEMETRY_UIKIND":     "desktop",
		}))
		dst, err := buildURL(baseURL, event)
		require.NoError(t, err)
		require.Equal(t, "https://home.runme.dev/Kernel?appname=Visual+Studio+Code&extname=stateful.runme&extversion=3.7.7-dev.10&platform=darwin_arm64&product=desktop&remotename=none&uikind=desktop", dst.String())
	})

	t.Run("KernelPartial", func(t *testing.T) {
		t.Parallel()

		event := kernelStartupEventFromEnv(createLookup(map[string]string{
			"TELEMETRY_EXTNAME":  "stateful.runme",
			"TELEMETRY_PLATFORM": "linux_x64",
		}))
		dst, err := buildURL(baseURL, event)
		require.NoError(t, err)
		require.Equal(t, "https://home.runme.dev/Kernel?extname=stateful.runme&platform=linux_x64", dst.String())
	})

	t.Run("KernelEmpty", func(t *testing.T) {
		t.Parallel()

		event := kernelStartupEventFromEnv(createLookup(map[string]string{}))
		_, err := buildURL(baseURL, event)
		require.ErrorContains(t, err, "no telemetry properties provided")
	})

	t.Run("CLI", func(t *testing.T) {
		t.Parallel()

		dst, err := buildURL(baseURL, event{
			client: clientCLI,
			name:   cliInvocationEvent,
			props: map[string]string{
				"component":       "cli",
				"version":         "1.2.3",
				"os":              "darwin",
				"arch":            "arm64",
				"command":         "run",
				"command_path":    "run",
				"install_channel": "homebrew",
			},
		})
		require.NoError(t, err)
		require.Equal(t, "https://home.runme.dev/CLI?arch=arm64&command=run&command_path=run&component=cli&event=cli_invocation&install_channel=homebrew&os=darwin&version=1.2.3", dst.String())
		require.NotContains(t, dst.String(), "/opt/homebrew")
	})
}

func createLookup(fixture map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := fixture[key]
		return value, ok
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
