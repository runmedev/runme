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

			_, err := getConfigFromViper(v)
			if err != nil {
				t.Fatalf("Failed to get config; %+v", err)
			}

			// Add additional assertions here
		})
	}
}
