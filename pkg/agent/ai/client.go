package ai

import (
	"os"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/runmedev/runme/v3/pkg/agent/logs"

	"github.com/runmedev/runme/v3/pkg/agent/config"

	"github.com/pkg/errors"
)

// NewClient helper function to create a new OpenAI client from  a config
func NewClient(cfg config.OpenAIConfig) (*openai.Client, error) {
	if cfg.APIKeyFile == "" {
		log := logs.NewLogger()
		log.Info("OpenAI client configured without APIKeyFile")
		return NewClientWithoutKey(), nil
	}

	b, err := os.ReadFile(cfg.APIKeyFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read OpenAI API key file: %s", cfg.APIKeyFile)
	}

	key := strings.TrimSpace(string(b))

	return NewClientWithKey(key)
}

func NewClientWithKey(key string) (*openai.Client, error) {
	// ************************************************************************
	// Setup middleware
	// ************************************************************************

	// Handle retryable errors
	// To handle retryable errors we use hashi corp's retryable client. This client will automatically retry on
	// retryable errors like 429; rate limiting
	retryClient := retryablehttp.NewClient()
	httpClient := retryClient.StandardClient()

	client := openai.NewClient(
		option.WithAPIKey(key), // defaults to os.LookupEnv("OPENAI_API_KEY")
		option.WithHTTPClient(httpClient),
	)
	return &client, nil
}

// NewClientWithoutKey returns an OpenAI client configured without an API key.
// This is intended for OAuth flows where the caller supplies the Authorization header per-request.
func NewClientWithoutKey() *openai.Client {
	retryClient := retryablehttp.NewClient()
	httpClient := retryClient.StandardClient()

	client := openai.NewClient(
		option.WithHTTPClient(httpClient),
	)
	return &client
}
