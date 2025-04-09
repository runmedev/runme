package main

import (
	"context"
	"strings"

	"dagger/runme/internal/dagger"
)

// A Dagger module for Runme
type Runme struct {
	// GhcrUsername is the GitHub Container Registry username for authentication
	GhcrUsername string
	// GhcrToken is the GitHub Container Registry token for authentication
	GhcrToken *dagger.Secret

	// ContainerPlatform specifies the target platform for container builds
	ContainerPlatform dagger.Platform
	// Source is the local source directory for building Runme
	Source *dagger.Directory

	// TargetOS specifies the target operating system for the build
	TargetOS string
	// TargetArch specifies the target architecture for the build
	TargetArch string
}

func New(
	// +optional
	ghcrUsername string,
	// +optional
	ghcrToken *dagger.Secret,

	// +optional
	targetPlatform string,
) *Runme {
	targetOS := "linux"
	targetArch := "amd64"

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

		// build env container only available for linux/amd64
		ContainerPlatform: dagger.Platform("linux/amd64"),

		TargetOS:   targetOS,
		TargetArch: targetArch,
	}
}

// WithSource sets the source directory for the Runme module and returns the module
// for method chaining.
func (m *Runme) WithSource(source *dagger.Directory) *Runme {
	m.Source = source
	return m
}

// Container creates a container with Runme source and registry auth.
func (m *Runme) Container(ctx context.Context) *dagger.Container {
	ctr := dag.Container(dagger.ContainerOpts{Platform: m.ContainerPlatform}).
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
			WithExec([]string{"git", "tag", "-f", "v99.9.9"}).
			WithExec([]string{"git", "tag", "-d", "main"})
	}

	return ctr.
		WithEnvVariable("GOOS", m.TargetOS).
		WithEnvVariable("GOARCH", m.TargetArch)
}

// Build compiles the Runme binary and returns the container with the built binary.
func (m *Runme) Build(ctx context.Context) *dagger.Container {
	return m.Container(ctx).
		WithExec([]string{"make", "build"})
}

// Binary returns the Runme binary as a file.
func (m *Runme) Binary(ctx context.Context) *dagger.File {
	return m.Build(ctx).
		File("/usr/local/bin/runme")
}

// Test runs the test suite for Runme and returns the container with test results.
func (m *Runme) Test(ctx context.Context) *dagger.Container {
	return m.Container(ctx).
		WithEnvVariable("RUNME_TEST_ENV", "docker").
		// short-cut tests for development
		// WithEnvVariable("PKGS", "github.com/runmedev/runme/v3/pkg/document/editor").
		WithExec([]string{"make", "test"})
}
