package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/spf13/viper"
)

func Test_UpdateViperConfig(t *testing.T) {
	type testCase struct {
		name       string
		configFile string
		expression string
		expected   *Config
	}

	cases := []testCase{
		{
			name:       "model",
			configFile: "empty.yaml",
			expression: "SomeOption=some-value",
			expected: &Config{
				Logging: Logging{
					Level: "",
				},
			},
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory")
	}
	tDir := filepath.Join(cwd, "test_data")

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Create an empty configuration file and run various assertions on it
			v := viper.New()
			v.SetConfigFile(filepath.Join(tDir, c.configFile))

			if err := InitViperInstance(v, nil); err != nil {
				t.Fatalf("Failed to initialize the configuration.")
			}

			cfg, err := UpdateViperConfig(v, c.expression)
			if err != nil {
				t.Fatalf("Failed to update config; %+v", err)
			}

			opts := cmpopts.IgnoreUnexported(Config{})
			if d := cmp.Diff(c.expected, cfg, opts); d != "" {
				t.Fatalf("Unexpected diff:\n%+v", d)
			}
		})
	}
}
