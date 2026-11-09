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

type command struct {
	program string
	args    []string
}

func newCommand(program string, args []string) command {
	return command{program: program, args: append([]string(nil), args...)}
}

func commandFromFields(fields []string) command {
	if len(fields) == 0 {
		return command{}
	}
	return newCommand(fields[0], fields[1:])
}

func (c command) fields() []string {
	return append([]string{c.program}, c.args...)
}

func (c command) String() string {
	return strings.Join(c.fields(), " ")
}

// invocation owns the positional details of a parsed Dagger interpreter
// command. Resolution can therefore operate on intent instead of slice indices.
type invocation struct {
	command         command
	interpreter     interpreter
	daggerIndex     int
	subcommandIndex int
}

func parseInvocation(program string, args []string) (invocation, bool) {
	cmd := newCommand(program, args)
	fields := cmd.fields()
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

func (i invocation) versionCommand() command {
	fields := i.command.fields()
	versionFields := append([]string(nil), fields[:i.daggerIndex+1]...)
	for index := i.daggerIndex + 1; index < len(fields); index++ {
		switch {
		case fields[index] == "--":
			return commandFromFields(append(versionFields, "version"))
		case fields[index] == "--x-release" && index+1 < len(fields):
			versionFields = append(versionFields, fields[index], fields[index+1])
			index++
		case strings.HasPrefix(fields[index], "--x-release="):
			versionFields = append(versionFields, fields[index])
		}
	}
	return commandFromFields(append(versionFields, "version"))
}

func (i invocation) asScript() command {
	fields := i.command.fields()
	fields[i.subcommandIndex] = "script"
	return commandFromFields(fields)
}

type versionProbe func(context.Context, command, []string) ([]byte, error)

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

func (r *resolver) resolve(ctx context.Context, invocation invocation, env []string) (command, error) {
	version, err := r.version(ctx, invocation.versionCommand(), env)
	if err != nil {
		if invocation.interpreter == legacyInterpreter {
			return invocation.command, nil
		}
		return command{}, fmt.Errorf("determine Dagger CLI version for %q: %w", invocation.command, err)
	}

	if version.LessThan(v1Version) {
		if invocation.interpreter == scriptInterpreter {
			return command{}, fmt.Errorf("%q requires Dagger v1 or newer; installed version is %s", invocation.command, version)
		}
		return invocation.command, nil
	}

	return invocation.asScript(), nil
}

func (r *resolver) version(ctx context.Context, cmd command, env []string) (*semver.Version, error) {
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
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return false
	}
	_, ok := parseInvocation(fields[0], fields[1:])
	return ok
}

// ResolveArgs selects the interpreter spelling supported by the installed
// Dagger CLI. Legacy input is preserved when version detection fails.
func ResolveArgs(ctx context.Context, program string, args, env []string) ([]string, error) {
	invocation, ok := parseInvocation(program, args)
	if !ok {
		return args, nil
	}
	resolved, err := defaultResolver.resolve(ctx, invocation, env)
	if err != nil {
		return nil, err
	}
	return resolved.args, nil
}

func probeVersion(ctx context.Context, cmd command, env []string) ([]byte, error) {
	probeCtx, cancel := context.WithTimeout(ctx, versionProbeTimeout)
	defer cancel()

	executable := exec.CommandContext(probeCtx, cmd.program, cmd.args...)
	if len(env) > 0 {
		executable.Env = env
	}
	output, err := executable.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run %q: %w", cmd, err)
	}
	return output, nil
}

func newVersionCacheKey(cmd command, env []string) versionCacheKey {
	return versionCacheKey{
		program: cmd.program,
		args:    strings.Join(cmd.args, "\x00"),
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
