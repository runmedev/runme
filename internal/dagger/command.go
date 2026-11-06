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
	LegacyCommand = "dagger shell"
	Command       = "dagger script"
)

var (
	v1Version       = semver.MustParse("1.0.0-0")
	versionPattern  = regexp.MustCompile(`v?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?)`)
	defaultResolver = newCommandResolver(probeVersion)
)

type commandKind uint8

const (
	commandLegacyScript commandKind = iota + 1
	commandScript
)

type invocation struct {
	kind            commandKind
	daggerIndex     int
	subcommandIndex int
}

type versionProbe func(context.Context, []string, []string) ([]byte, error)

type commandResolver struct {
	probe    versionProbe
	versions sync.Map
}

func newCommandResolver(probe versionProbe) *commandResolver {
	return &commandResolver{probe: probe}
}

// IsCommand reports whether command is a Dagger script interpreter. The exact
// legacy "dagger shell" spelling is supported, but shell invocations with
// arguments belong to Dagger v1's terminal command and are left alone.
func IsCommand(command string) bool {
	parsed, ok := parseInvocation(strings.Fields(command))
	return ok && (parsed.kind == commandLegacyScript || parsed.kind == commandScript)
}

// ResolveArgs selects the script-interpreter spelling supported by the installed
// Dagger CLI. Legacy input is preserved when the version probe fails.
func ResolveArgs(ctx context.Context, program string, args, env []string) ([]string, error) {
	fields := append([]string{program}, args...)
	resolved, err := defaultResolver.resolve(ctx, fields, env)
	if err != nil {
		return nil, err
	}
	if args == nil && len(resolved) == 1 {
		return nil, nil
	}
	return resolved[1:], nil
}

func (r *commandResolver) resolve(ctx context.Context, fields, env []string) ([]string, error) {
	parsed, ok := parseInvocation(fields)
	if !ok {
		return fields, nil
	}

	versionFields := append([]string{}, fields[:parsed.daggerIndex+1]...)
	for index := parsed.daggerIndex + 1; index < parsed.subcommandIndex; index++ {
		if fields[index] == "--x-release" && index+1 < parsed.subcommandIndex {
			versionFields = append(versionFields, fields[index], fields[index+1])
			index++
			continue
		}
		if strings.HasPrefix(fields[index], "--x-release=") {
			versionFields = append(versionFields, fields[index])
		}
	}
	versionFields = append(versionFields, "version")
	version, err := r.version(ctx, versionFields, env)
	if err != nil {
		if parsed.kind == commandLegacyScript {
			return fields, nil
		}
		return nil, fmt.Errorf("determine Dagger CLI version for %q: %w", strings.Join(fields, " "), err)
	}

	if version.LessThan(v1Version) {
		if parsed.kind == commandScript {
			return nil, fmt.Errorf("%q requires Dagger v1 or newer; installed version is %s", strings.Join(fields, " "), version)
		}
		return fields, nil
	}

	resolved := append([]string{}, fields...)
	resolved[parsed.subcommandIndex] = "script"
	return resolved, nil
}

func (r *commandResolver) version(ctx context.Context, fields, env []string) (*semver.Version, error) {
	key := versionCacheKey(fields, env)
	if cached, ok := r.versions.Load(key); ok {
		return cached.(*semver.Version), nil
	}

	output, err := r.probe(ctx, fields, env)
	if err != nil {
		return nil, err
	}
	version, err := parseVersion(string(output))
	if err != nil {
		return nil, err
	}
	r.versions.Store(key, version)
	return version, nil
}

func parseInvocation(fields []string) (invocation, bool) {
	if len(fields) < 2 {
		return invocation{}, false
	}

	daggerIndex := -1
	switch executableName(fields[0]) {
	case "dagger":
		daggerIndex = 0
	case "env":
		for index := 1; index < len(fields); index++ {
			if executableName(fields[index]) == "dagger" {
				daggerIndex = index
				break
			}
		}
	default:
		return invocation{}, false
	}
	if daggerIndex < 0 {
		return invocation{}, false
	}

	for index := daggerIndex + 1; index < len(fields); index++ {
		switch fields[index] {
		case "shell":
			if index != len(fields)-1 {
				return invocation{}, false
			}
			return invocation{kind: commandLegacyScript, daggerIndex: daggerIndex, subcommandIndex: index}, true
		case "script":
			return invocation{kind: commandScript, daggerIndex: daggerIndex, subcommandIndex: index}, true
		}
	}

	return invocation{}, false
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

func versionCacheKey(fields, env []string) string {
	return strings.Join(fields, "\x00") + "\x00PATH=" + envValue(env, "PATH") + "\x00DAGGER_X_RELEASE=" + envValue(env, "DAGGER_X_RELEASE")
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

func probeVersion(ctx context.Context, fields, env []string) ([]byte, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, fields[0], fields[1:]...)
	if len(env) > 0 {
		cmd.Env = env
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run %q: %w", strings.Join(fields, " "), err)
	}
	return output, nil
}
