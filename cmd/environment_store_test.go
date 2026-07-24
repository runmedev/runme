package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	runnerv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/runner/v1"
)

func TestSnapshotEnvsFromProtoMarksExplicitRows(t *testing.T) {
	t.Parallel()

	envs := snapshotEnvsFromProto([]*runnerv1.MonitorEnvStoreResponseSnapshot_SnapshotEnv{
		{Name: "API_URL", Spec: "Plain"},
		{Name: "DATABASE_URL", Spec: "Opaque", Description: "Database URL"},
		{Name: "TOKEN", Spec: "Opaque", IsRequired: true},
		{Name: "PATH", Spec: "Opaque"},
		{Name: "HOME", Spec: "https://owl.runme.dev/v1/types/core/opaque"},
	})

	byName := make(map[string]bool, len(envs))
	for _, env := range envs {
		byName[env.Name] = env.Explicit
	}

	assert.True(t, byName["API_URL"])
	assert.True(t, byName["DATABASE_URL"])
	assert.True(t, byName["TOKEN"])
	assert.False(t, byName["PATH"])
	assert.False(t, byName["HOME"])
}
