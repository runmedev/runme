package dagger

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"

	"github.com/runmedev/runme/v3/internal/execution"
)

const (
	LegacyCommand       = "dagger shell"
	Command             = "dagger script"
	versionProbeTimeout = 5 * time.Second
)

var (
	v1Version       = semver.MustParse("1.0.0-0")
	versionPattern  = regexp.MustCompile(`v?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?)`)
	defaultResolver = newResolver(probeVersion)
)

type interpreter uint8

const (
	legacyInterpreter interpreter = iota + 1
	scriptInterpreter
)

func invocationFromFields(fields []string) execution.Invocation {
	if len(fields) == 0 {
		return execution.Invocation{}
	}
	return execution.NewInvocation(fields[0], fields[1:])
}

// invocation owns the positional details of a parsed Dagger interpreter
// command. Resolution can therefore operate on intent instead of slice indices.
type invocation struct {
	command         execution.Invocation
	interpreter     interpreter
	daggerIndex     int
	subcommandIndex int
}

func parseInvocation(program string, args []string) (invocation, bool) {
	cmd := execution.NewInvocation(program, args)
	fields := cmd.Fields()
	if len(fields) < 2 {
		return invocation{}, false
	}

	daggerIndex := findDaggerExecutable(fields)
	if daggerIndex < 0 {
		return invocation{}, false
	}

	for index := daggerIndex + 1; index < len(fields); index++ {
		switch fields[index] {
		case "shell":
			// In interpreter configuration, `shell` denotes the legacy script
			// interpreter. Leave argument validation to Dagger.
			return invocation{cmd, legacyInterpreter, daggerIndex, index}, true
		case "script":
			return invocation{cmd, scriptInterpreter, daggerIndex, index}, true
		}
	}

	return invocation{}, false
}

func (i invocation) versionCommand() execution.Invocation {
	fields := i.command.Fields()
	versionFields := append([]string(nil), fields[:i.daggerIndex+1]...)
	for index := i.daggerIndex + 1; index < len(fields); index++ {
		switch {
		case fields[index] == "--":
			return invocationFromFields(append(versionFields, "version"))
		case fields[index] == "--x-release" && index+1 < len(fields):
			versionFields = append(versionFields, fields[index], fields[index+1])
			index++
		case strings.HasPrefix(fields[index], "--x-release="):
			versionFields = append(versionFields, fields[index])
		}
	}
	return invocationFromFields(append(versionFields, "version"))
}

func (i invocation) asScript() execution.Invocation {
	fields := i.command.Fields()
	fields[i.subcommandIndex] = "script"
	return invocationFromFields(fields)
}

type versionProbe func(context.Context, execution.Invocation, []string) ([]byte, error)

type versionCacheKey struct {
	program string
	args    string
	path    string
	release string
}

type resolver struct {
	probe versionProbe
	mu    sync.Mutex
	// +checklocks:mu
	cache map[versionCacheKey]*semver.Version
}

func newResolver(probe versionProbe) *resolver {
	return &resolver{
		probe: probe,
		cache: make(map[versionCacheKey]*semver.Version),
	}
}

func (r *resolver) resolve(ctx context.Context, invocation invocation, env []string) (execution.Invocation, error) {
	version, err := r.version(ctx, invocation.versionCommand(), env)
	if err != nil {
		if invocation.interpreter == legacyInterpreter {
			return invocation.command, nil
		}
		return execution.Invocation{}, fmt.Errorf("determine Dagger CLI version for %q: %w", invocation.command, err)
	}

	if version.LessThan(v1Version) {
		if invocation.interpreter == scriptInterpreter {
			return execution.Invocation{}, fmt.Errorf("%q requires Dagger v1 or newer; installed version is %s", invocation.command, version)
		}
		return invocation.command, nil
	}

	return invocation.asScript(), nil
}

func (r *resolver) version(ctx context.Context, cmd execution.Invocation, env []string) (*semver.Version, error) {
	key := newVersionCacheKey(cmd, env)
	r.mu.Lock()
	version, ok := r.cache[key]
	r.mu.Unlock()
	if ok {
		return version, nil
	}

	output, err := r.probe(ctx, cmd, env)
	if err != nil {
		return nil, err
	}
	version, err = parseVersion(string(output))
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.cache[key] = version
	r.mu.Unlock()
	return version, nil
}

// IsCommand reports whether command is a Dagger script interpreter.
func IsCommand(value string) bool {
	tokens, valid := headerTokens(value)
	if !valid || len(tokens) == 0 {
		return false
	}
	fields := make([]string, len(tokens))
	for i, token := range tokens {
		fields[i] = token.value
	}
	_, ok := parseInvocation(fields[0], fields[1:])
	return ok
}

// ResolveInvocation selects the interpreter spelling supported by the installed
// Dagger CLI. Legacy input is preserved when version detection fails.
func ResolveInvocation(ctx context.Context, candidate execution.Invocation, env []string) (execution.Invocation, error) {
	invocation, ok := parseInvocation(candidate.Executable(), candidate.Arguments())
	if !ok {
		return candidate, nil
	}
	resolved, err := defaultResolver.resolve(ctx, invocation, env)
	if err != nil {
		return execution.Invocation{}, err
	}
	return resolved, nil
}

func probeVersion(ctx context.Context, cmd execution.Invocation, env []string) ([]byte, error) {
	probeCtx, cancel := context.WithTimeout(ctx, versionProbeTimeout)
	defer cancel()

	executable := exec.CommandContext(probeCtx, cmd.Executable(), cmd.Arguments()...)
	if len(env) > 0 {
		executable.Env = env
	}
	output, err := executable.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run %q: %w", cmd, err)
	}
	return output, nil
}

func newVersionCacheKey(cmd execution.Invocation, env []string) versionCacheKey {
	return versionCacheKey{
		program: cmd.Executable(),
		args:    strings.Join(cmd.Arguments(), "\x00"),
		path:    envValue(env, "PATH"),
		release: envValue(env, "DAGGER_X_RELEASE"),
	}
}

func findDaggerExecutable(fields []string) int {
	switch executableName(fields[0]) {
	case "dagger":
		return 0
	case "env":
		for index := 1; index < len(fields); index++ {
			if executableName(fields[index]) == "dagger" {
				return index
			}
		}
	}
	return -1
}

func executableName(path string) string {
	path = strings.ReplaceAll(path, `\`, "/")
	name := filepath.Base(path)
	return strings.TrimSuffix(strings.ToLower(name), ".exe")
}

func parseVersion(output string) (*semver.Version, error) {
	match := versionPattern.FindStringSubmatch(output)
	if len(match) < 2 {
		return nil, fmt.Errorf("dagger version output did not contain a semantic version: %q", strings.TrimSpace(output))
	}
	version, err := semver.NewVersion(match[1])
	if err != nil {
		return nil, fmt.Errorf("parse Dagger version %q: %w", match[1], err)
	}
	return version, nil
}

func envValue(env []string, name string) string {
	prefix := name + "="
	for index := len(env) - 1; index >= 0; index-- {
		if strings.HasPrefix(env[index], prefix) {
			return strings.TrimPrefix(env[index], prefix)
		}
	}
	return ""
}
