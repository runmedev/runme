package harbor

import (
	"fmt"
	"os"
	"strings"
)

const (
	runmeHarborSkipMetadataSyncEnv = "RUNME_HARBOR_SKIP_METADATA_SYNC"
	runmeEnvironmentImportPath     = "runme_harbor.environment:RunmeEnvironment"
)

var runmeAgentSpecs = []runmeAgentSpec{
	{name: "oracle"},
	{name: "antigravity-cli", importPath: "runme_harbor.runme_agents:RunmeAntigravityCli"},
	{name: "codex", importPath: "runme_harbor.runme_agents:RunmeCodex"},
	{name: "claude-code", importPath: "runme_harbor.runme_agents:RunmeClaudeCode"},
	{name: "cursor-cli", importPath: "runme_harbor.runme_agents:RunmeCursorCli"},
	{name: "openclaw", importPath: "runme_harbor.runme_agents:RunmeOpenClaw"},
}

var (
	modelHarborFlag       = harborFlag{names: []string{"--model"}}
	agentKwargHarborFlag  = harborFlag{names: []string{"--agent-kwarg", "--ak"}}
	agentEnvHarborFlag    = harborFlag{names: []string{"--agent-env", "--ae"}}
	environmentHarborFlag = harborFlag{
		names:            []string{"--env", "-e", "--environment-import-path"},
		allowJoinedShort: true,
	}
	concurrencyHarborFlag = harborFlag{
		names:            []string{"--n-concurrent", "-n"},
		allowJoinedShort: true,
	}
)

type harborRunArgsBuilder struct {
	datasetPath string
	jobsDir     string
	opts        EvalOptions
	passthrough []string
}

func (b harborRunArgsBuilder) Build() ([]string, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	args := b.baseArgs()
	envArgs, err := b.environmentArgs()
	if err != nil {
		return nil, err
	}
	args = append(args, envArgs...)
	if b.opts.TaskDir != "" {
		args = append(args, "--include-task-name", b.opts.TaskDir)
	}
	if !b.opts.Ask {
		args = append(args, "-y")
	}

	extraArgs := b.extraArgs()
	if !concurrencyHarborFlag.Present(extraArgs) {
		args = append(args, "--n-concurrent", "1")
	}
	args = append(args, extraArgs...)
	return args, nil
}

func (b harborRunArgsBuilder) validate() error {
	if b.opts.Model != "" && modelHarborFlag.Present(b.passthrough) {
		return fmt.Errorf("--model cannot be used together with passthrough --model; use only runme eval --model")
	}
	if len(b.opts.AgentKwargs) > 0 && agentKwargHarborFlag.Present(b.passthrough) {
		return fmt.Errorf("--agent-kwarg cannot be used together with passthrough --agent-kwarg/--ak; use only runme eval --agent-kwarg")
	}
	if len(b.opts.AgentEnv) > 0 && agentEnvHarborFlag.Present(b.passthrough) {
		return fmt.Errorf("--agent-env cannot be used together with passthrough --agent-env/--ae; use only runme eval --agent-env")
	}
	if environmentHarborFlag.Present(b.passthrough) {
		return fmt.Errorf("use runme eval --env instead of passing Harbor environment flags after --")
	}
	return nil
}

func (b harborRunArgsBuilder) baseArgs() []string {
	return []string{
		"run",
		"--path",
		b.datasetPath,
		"--jobs-dir", b.jobsDir,
	}
}

func (b harborRunArgsBuilder) environmentArgs() ([]string, error) {
	if usesRunmeEnvironment(b.opts.Env) {
		spec, ok := runmeAgentByName(b.opts.Agent)
		if !ok {
			return nil, fmt.Errorf("invalid --agent %q: expected %s", b.opts.Agent, runmeAgentNames())
		}
		return append([]string{"--env", runmeEnvironmentImportPath}, spec.HarborArgs()...), nil
	}
	return []string{"--env", b.opts.Env, "--agent", b.opts.Agent}, nil
}

func (b harborRunArgsBuilder) extraArgs() []string {
	args := append([]string(nil), b.passthrough...)
	for _, kwarg := range b.opts.AgentKwargs {
		args = append(args, "--agent-kwarg", kwarg)
	}
	for _, env := range b.opts.AgentEnv {
		args = append(args, "--agent-env", env)
	}
	if b.opts.Model != "" {
		args = append(args, "--model", b.opts.Model)
	}
	return args
}

type runmeAgentSpec struct {
	name       string
	importPath string
}

func (s runmeAgentSpec) HarborArgs() []string {
	if s.importPath != "" {
		return []string{"--agent", s.importPath}
	}
	return []string{"--agent", s.name}
}

func runmeAgentByName(name string) (runmeAgentSpec, bool) {
	for _, spec := range runmeAgentSpecs {
		if spec.name == name {
			return spec, true
		}
	}
	return runmeAgentSpec{}, false
}

func runmeAgentNames() string {
	names := make([]string, 0, len(runmeAgentSpecs))
	for _, spec := range runmeAgentSpecs {
		names = append(names, spec.name)
	}
	return strings.Join(names, ", ")
}

type harborFlag struct {
	names            []string
	allowJoinedShort bool
}

func (f harborFlag) Present(args []string) bool {
	for _, arg := range args {
		for _, name := range f.names {
			if arg == name || strings.HasPrefix(arg, name+"=") {
				return true
			}
			if f.allowJoinedShort && isShortFlag(name) && strings.HasPrefix(arg, name) && len(arg) > len(name) {
				return true
			}
		}
	}
	return false
}

func isShortFlag(name string) bool {
	return strings.HasPrefix(name, "-") && !strings.HasPrefix(name, "--")
}

func usesRunmeEnvironment(env string) bool {
	return env == "" || env == "runme"
}

func skipMetadataSync() bool {
	switch strings.ToLower(os.Getenv(runmeHarborSkipMetadataSyncEnv)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func usesHarborDockerEnvironment(env string) bool {
	return env == "docker"
}
