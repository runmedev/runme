package command

import "strings"

const (
	// StoreStdoutEnvName is the session environment variable containing the
	// output of the previous command.
	StoreStdoutEnvName = "__"

	// EnvNameKnownName is the process environment variable containing the
	// known name of the code block being executed.
	EnvNameKnownName = "RUNME_NAME"
)

func CreateEnv(key, value string) string {
	return createEnv(key, value)
}

func createEnv(key, value string) string {
	return key + "=" + value
}

// AppendKnownNameEnv appends the known code block name to env when present.
// Existing values are removed first so Runme-managed execution metadata has
// deterministic precedence across native, virtual, and container runtimes.
func AppendKnownNameEnv(env []string, knownName string) []string {
	if knownName == "" {
		return env
	}

	result := make([]string, 0, len(env)+1)
	for _, item := range env {
		key, _, ok := strings.Cut(item, "=")
		if ok && strings.EqualFold(key, EnvNameKnownName) {
			continue
		}
		result = append(result, item)
	}

	return append(result, CreateEnv(EnvNameKnownName, knownName))
}
