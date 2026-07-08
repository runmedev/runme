package telemetry

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

func ReportKernelStartup(logger *zap.Logger) bool {
	return newReporter(logger).report(kernelStartupEventFromEnv(lookupEnv))
}

func kernelStartupEventFromEnv(lookup lookupEnvFunc) event {
	props := map[string]string{}
	for _, prop := range []string{
		"extname",
		"extversion",
		"remotename",
		"appname",
		"product",
		"platform",
		"uikind",
	} {
		addEnvProp(lookup, props, prop)
	}

	return event{
		client:  clientKernel,
		props:   props,
		timeout: 30 * time.Second,
	}
}

func addEnvProp(lookup lookupEnvFunc, props map[string]string, prop string) {
	if v, ok := lookup(fmt.Sprintf("TELEMETRY_%s", strings.ToUpper(prop))); ok {
		props[strings.ToLower(prop)] = v
	}
}
