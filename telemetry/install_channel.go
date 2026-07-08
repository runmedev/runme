package telemetry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	installChannelHomebrew     = "homebrew"
	installChannelNPM          = "npm"
	installChannelGoInstall    = "go_install"
	installChannelLinuxPackage = "linux_package"
	installChannelDocker       = "docker"
	installChannelSource       = "source"
	installChannelUnknown      = "unknown"
)

func installChannel() string {
	if runningInContainer() {
		return installChannelDocker
	}

	executable, err := os.Executable()
	if err != nil {
		return installChannelUnknown
	}

	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		resolved = executable
	}

	return installChannelFromPath(resolved)
}

func installChannelFromPath(path string) string {
	normalized := filepath.ToSlash(strings.ToLower(path))

	switch {
	case strings.Contains(normalized, "/cellar/runme/"),
		strings.Contains(normalized, "/homebrew/"),
		strings.Contains(normalized, "/linuxbrew/"):
		return installChannelHomebrew
	case strings.Contains(normalized, "/node_modules/"),
		strings.Contains(normalized, "/npm/"):
		return installChannelNPM
	case strings.HasSuffix(normalized, "/go/bin/runme"),
		strings.Contains(normalized, "/go/bin/runme"):
		return installChannelGoInstall
	case normalized == "/usr/bin/runme",
		normalized == "/bin/runme",
		strings.HasPrefix(normalized, "/usr/lib/runme/"),
		strings.HasPrefix(normalized, "/usr/share/runme/"):
		return installChannelLinuxPackage
	case strings.Contains(normalized, "/runme/") && strings.HasSuffix(normalized, "/runme"):
		return installChannelSource
	default:
		return installChannelUnknown
	}
}

func runningInContainer() bool {
	if runtime.GOOS == "windows" {
		return false
	}

	for _, path := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}
