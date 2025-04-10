package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/runme/internal/dagger"

	"github.com/google/go-github/v71/github"
)

// A Dagger module for Runme
type Runme struct {
	// GhcrUsername is the GitHub Container Registry username for authentication
	GhcrUsername string
	// GhcrToken is the GitHub Container Registry token for authentication
	GhcrToken *dagger.Secret

	// Source is the local source directory for building Runme
	Source *dagger.Directory

	// TargetOS specifies the target operating system for the build
	TargetOS string
	// TargetArch specifies the target architecture for the build
	TargetArch string
}

func New(
	// GhcrUsername is the GitHub Container Registry username for authentication
	// +optional
	ghcrUsername string,
	// GhcrToken is the GitHub Container Registry token for authentication
	// +optional
	ghcrToken *dagger.Secret,

	// TargetPlatform specifies the target platform (os/arch combination) for the build
	// +optional
	targetPlatform string,
) *Runme {
	targetOS := "linux"
	targetArch := "amd64"

	if p, err := dag.DefaultPlatform(context.Background()); err == nil {
		parts := strings.Split(string(p), "/")
		if len(parts) == 2 {
			// keep Linux as OS, but set Arch
			targetArch = parts[1]
		}
	}

	if targetPlatform != "" {
		parts := strings.Split(targetPlatform, "/")
		if len(parts) == 2 {
			targetOS = parts[0]
			targetArch = parts[1]
		}
	}

	return &Runme{
		GhcrUsername: ghcrUsername,
		GhcrToken:    ghcrToken,

		TargetOS:   targetOS,
		TargetArch: targetArch,
	}
}

// TargetPlatform returns the target platform in the format "OS/ARCH".
func (m *Runme) TargetPlatform(ctx context.Context) string {
	return fmt.Sprintf("%s/%s", m.TargetOS, m.TargetArch)
}

// WithSource sets the source directory for the Runme module and returns the module
// for method chaining.
func (m *Runme) WithSource(source *dagger.Directory) *Runme {
	m.Source = source
	return m
}

// Container creates a container with Runme source and registry auth.
func (m *Runme) Container(ctx context.Context) *dagger.Container {
	// archs have to match for app-level unit tests to pass
	containerPlatform := dagger.Platform(fmt.Sprintf("linux/%s", m.TargetArch))
	ctr := dag.Container(dagger.ContainerOpts{Platform: containerPlatform}).
		From("ghcr.io/runmedev/runme-build-env:latest").
		WithWorkdir("/workspace")

	if m.GhcrToken != nil {
		ctr = ctr.WithRegistryAuth("ghcr.io", m.GhcrUsername, m.GhcrToken)
	}

	if m.Source != nil {
		ctr = ctr.WithMountedDirectory("/workspace", m.Source)
	} else {
		main := dag.Git("https://github.com/runmedev/runme").Branch("main").Tree()
		ctr = ctr.WithMountedDirectory("/workspace", main).
			// retagging to make version tests work
			WithExec([]string{"git", "tag", "-f", "v3.999.999"}).
			WithExec([]string{"git", "tag", "-d", "main"})
	}

	return ctr
}

// Build compiles the Runme binary for the target OS and architecture
// specified in the module and returns the container with the built binary.
func (m *Runme) Build(ctx context.Context) *dagger.Container {
	return m.Container(ctx).
		WithEnvVariable("GOOS", m.TargetOS).
		WithEnvVariable("GOARCH", m.TargetArch).
		WithExec([]string{"make", "build"})
}

// Binary returns the Runme binary as a file.
func (m *Runme) Binary(ctx context.Context) *dagger.File {
	return m.Build(ctx).
		File("/usr/local/bin/runme")
}

// Test runs the test suite for Runme and returns the container with test results.
func (m *Runme) Test(
	ctx context.Context,
	// pkgs is an optional golang package name to narrow down the test suite
	// +default="./..."
	pkgs string,
) *dagger.Container {
	return m.Container(ctx).
		WithEnvVariable("GOARCH", m.TargetArch).
		WithEnvVariable("RUNME_TEST_ENV", "docker").
		WithEnvVariable("PKGS", pkgs).
		WithExec([]string{"make", "test"})
}

// Release fetches a Runme release from GitHub and returns a directory with the release assets.
func (m *Runme) Release(ctx context.Context,
	// GithubToken is an optional authentication token for GitHub API access
	// +optional
	githubToken *dagger.Secret,
	// Version specifies the release version to fetch, defaults to "latest"
	// +optional
	// +default="latest"
	version string,
) *dagger.Directory {
	client := github.NewClient(nil)
	if githubToken != nil {
		plaintext, err := githubToken.Plaintext(ctx)
		if err == nil {
			client = github.NewClient(nil).WithAuthToken(plaintext)
		}
	}

	var release *github.RepositoryRelease
	var err error

	switch version {
	case "latest":
		release, _, err = client.Repositories.GetLatestRelease(ctx, "runmedev", "runme")
	default:
		release, _, err = client.Repositories.GetReleaseByTag(ctx, "runmedev", "runme", version)
	}

	if err != nil {
		panic(fmt.Sprintf("Failed to get release: %v", err))
	}

	ctr := m.Container(ctx)
	releaseDir := "/releases/" + version

	for _, asset := range release.Assets {
		if !strings.HasSuffix(asset.GetName(), ".tar.gz") {
			continue
		}
		ctr = ctr.WithFile(releaseDir+"/"+asset.GetName(), dag.HTTP(asset.GetBrowserDownloadURL()))
	}

	return ctr.Directory(releaseDir)
}

// ReleaseFiles fetches a Runme release from GitHub and returns a directory with the uncompressed release files.
func (m *Runme) ReleaseFiles(ctx context.Context,
	// Platform specifies the target OS and architecture in the format "os/arch"
	// e.g. "linux/amd64"
	platform dagger.Platform,
	// GithubToken is an optional authentication token for GitHub API access
	// +optional
	githubToken *dagger.Secret,
	// Version specifies the release version to fetch, defaults to "latest"
	// +optional
	// +default="latest"
	version string,
) *dagger.Directory {
	parts := strings.Split(string(platform), "/")
	os := parts[0]
	arch := parts[1]

	// Map architecture names to match release file naming
	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
		"wasm":  "wasm",
	}

	archName := arch
	if mapped, ok := archMap[arch]; ok {
		archName = mapped
	}

	filename := fmt.Sprintf("runme_%s_%s.tar.gz", os, archName)
	release := m.Release(ctx, githubToken, version)

	ctr := m.Container(ctx).
		WithFile("/tmp/release/runme.tar.gz", release.File(filename)).
		WithWorkdir("/tmp/release").
		WithExec([]string{"tar", "-xzf", "runme.tar.gz"})

	return ctr.Directory("/tmp/release").
		WithoutFile("runme.tar.gz")
}
