package config

import (
	"strings"

	"github.com/pkg/errors"
)

// UpdateViperConfig update the viper configuration with the given expression.
// expression should be a value such as "agent.model=gpt-4o-mini"
// The input is a viper configuration because we leverage viper to handle setting most keys.
// However, in some special cases we use custom functions. This is why we return a Config object.
func (ac *AppConfig) UpdateViperConfig(expression string) (*Config, error) {
	pieces := strings.Split(expression, "=")
	cfgName := pieces[0]

	var fConfig *Config

	switch cfgName {
	default:
		if len(pieces) < 2 {
			return fConfig, errors.New("Invalid usage; set expects an argument in the form <NAME>=<VALUE>")
		}
		cfgValue := pieces[1]
		ac.V.Set(cfgName, cfgValue)
	}

	return ac.GetConfig(), nil
}
