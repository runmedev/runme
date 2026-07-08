package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	baseURL = "https://home.runme.dev/"

	envDoNotTrack       = "DO_NOT_TRACK"
	envScarfNoAnalytics = "SCARF_NO_ANALYTICS"
)

type client string

const (
	clientKernel client = "Kernel"
	clientCLI    client = "CLI"
)

type event struct {
	client  client
	name    string
	props   map[string]string
	timeout time.Duration
}

type reporter interface {
	report(event event) bool
}

type scarfReporter struct {
	logger *zap.Logger
	client *http.Client
}

type noopReporter struct{}

func newReporter(logger *zap.Logger) reporter {
	if TelemetryDisabledByUserOptOut() {
		if logger != nil {
			logger.Info("Telemetry reporting is disabled by user opt-out")
		}
		return noopReporter{}
	}

	return newScarfReporter(logger, http.DefaultClient)
}

func newScarfReporter(logger *zap.Logger, client *http.Client) reporter {
	if logger == nil {
		logger = zap.NewNop()
	}
	if client == nil {
		client = http.DefaultClient
	}

	return &scarfReporter{
		logger: logger,
		client: client,
	}
}

func (r *scarfReporter) report(event event) bool {
	r.logger.Info("Telemetry reporting is enabled")

	go func() {
		if err := r.send(event); err != nil {
			r.logger.Warn("Error reporting telemetry", zap.Error(err))
		}
	}()

	return true
}

func (noopReporter) report(event) bool {
	return false
}

func TelemetryDisabledByUserOptOut() bool {
	for _, key := range []string{envDoNotTrack, envScarfNoAnalytics} {
		disabled, err := trackingDisabledForEnv(key)
		if err == nil && disabled {
			return true
		}
	}

	return false
}

func trackingDisabledForEnv(key string) (bool, error) {
	val, err := strconv.ParseBool(getenv(key))
	if err != nil {
		return false, err
	}

	return val, nil
}

func (r *scarfReporter) send(event event) error {
	timeout := event.timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	encodedURL, err := buildURL(baseURL, event)
	if err != nil {
		return errors.Wrapf(err, "Error building telemetry URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, encodedURL.String(), nil)
	if err != nil {
		return errors.Wrapf(err, "Error creating telemetry request")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "Error sending telemetry request")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("error sending telemetry request: status_code=%d, status=%s", resp.StatusCode, resp.Status)
	}

	return nil
}

func buildURL(base string, event event) (*url.URL, error) {
	if event.client == "" {
		return nil, fmt.Errorf("telemetry client is required")
	}

	dst, err := url.Parse(base + string(event.client))
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	if event.name != "" {
		params.Add("event", event.name)
	}
	for key, value := range event.props {
		if value != "" {
			params.Add(key, value)
		}
	}

	// Until we have a non-extension-bundled reporting strategy, let's error.
	if len(params) == 0 {
		return nil, fmt.Errorf("no telemetry properties provided")
	}

	dst.RawQuery = params.Encode()

	return dst, nil
}
