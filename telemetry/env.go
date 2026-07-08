package telemetry

import "os"

type lookupEnvFunc func(key string) (string, bool)

var (
	getenv    = os.Getenv
	lookupEnv = os.LookupEnv
)
