package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func Test_ConfigDefaultConfig(t *testing.T) {
	type testCase struct {
		name       string
		configFile string
	}

	cases := []testCase{
		{
			name:       "empty-file",
			configFile: "empty.yaml",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := newViperWithTestDataConfigFile(c.configFile)
			if err != nil {
				t.Fatalf("Failed to create viper with test data config file: %v", err)
			}

			ac, err := NewAppConfig("runme-agent-test", WithViperInstance(v, nil))
			if err != nil {
				t.Fatalf("Failed to initialize the configuration.")
			}

			cfg := ac.GetConfig()
			if cfg == nil {
				t.Fatalf("Expected config to be non-nil")
			}

			if problems := cfg.IsValid(); len(problems) > 0 {
				t.Fatalf("Expected config to be valid, but got problems: %v", problems)
			}
		})
	}
}

func Test_ConfigAutomaticEnvVars(t *testing.T) {
	expected := "warn"
	unset := setEnvVars(map[string]string{
		"RUNME-AGENT-TEST_LOGGING_LEVEL": expected,
	})
	defer unset()

	v, err := newViperWithTestDataConfigFile("logging.yaml")
	if err != nil {
		t.Fatalf("Failed to create viper with test data config file: %v", err)
	}

	ac, err := NewAppConfig("runme-agent-test", WithViperInstance(v, nil))
	if err != nil {
		t.Fatalf("Failed to initialize the configuration.")
	}

	cfg := ac.GetConfig()
	if cfg == nil {
		t.Fatalf("Expected config to be non-nil")
	}

	if problems := cfg.IsValid(); len(problems) > 0 {
		t.Fatalf("Expected config to be valid, but got problems: %v", problems)
	}

	if cfg.Logging.Level != expected {
		t.Fatalf("Expected logging level to be %q, but got %q", expected, cfg.Logging.Level)
	}
}

// setEnvVars sets the provided environment variables and returns a function to restore the previous state.
func setEnvVars(env map[string]string) func() {
	if env == nil {
		return func() {}
	}
	oldEnv := make(map[string]*string)
	for k, v := range env {
		if old, ok := os.LookupEnv(k); ok {
			oldEnv[k] = &old
		} else {
			oldEnv[k] = nil
		}
		os.Setenv(k, v)
	}
	return func() {
		for k, old := range oldEnv {
			if old == nil {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, *old)
			}
		}
	}
}

// newViperWithTestDataConfigFile returns a new viper instance with the config file from test_data set.
func newViperWithTestDataConfigFile(configFile string) (*viper.Viper, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fullPath := filepath.Join(cwd, "test_data", configFile)
	v := viper.New()
	v.SetConfigFile(fullPath)
	return v, nil
}
