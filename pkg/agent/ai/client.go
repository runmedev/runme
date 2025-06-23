package ai

import (
	"os"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/runmedev/runme/v3/pkg/agent/config"

	"github.com/pkg/errors"
)

// NewClient helper function to create a new OpenAI client from  a config
func NewClient(cfg config.OpenAIConfig) (*openai.Client, error) {
	if cfg.APIKeyFile == "" {
		return nil, errors.New("OpenAI API key is empty")
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
