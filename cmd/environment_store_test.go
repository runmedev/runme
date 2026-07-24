package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	owlcmd "github.com/runmedev/owl/cmd"

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

func TestSnapshotTypeProposalHelpers(t *testing.T) {
	t.Parallel()

	suggested, reason := suggestSnapshotPrimitiveType(owlSnapshotEnv("API_KEY", "https://owl.runme.dev/v1/types/core/opaque"))
	assert.Equal(t, "core/secret", suggested)
	assert.Equal(t, "key name suggests sensitive value", reason)

	suggested, reason = suggestSnapshotPrimitiveType(owlSnapshotEnv("SERVICE_HOST", "core/opaque"))
	assert.Equal(t, "core/host", suggested)
	assert.Equal(t, "key name suggests host", reason)

	assert.Equal(t, "core/opaque", normalizeSnapshotType("https://owl.runme.dev/v1/types/core/opaque"))
	assert.Equal(t, "Host", dotenvSpecTypeName("core/host"))
}

func TestIncludeSnapshotTypeProposalSkipsDefaultPlainUnlessAll(t *testing.T) {
	t.Parallel()

	assert.False(t, includeSnapshotTypeProposal(owlcmd.TypeRequest{}, "core/plain"))
	assert.True(t, includeSnapshotTypeProposal(owlcmd.TypeRequest{All: true}, "core/plain"))
	assert.True(t, includeSnapshotTypeProposal(owlcmd.TypeRequest{}, "core/secret"))
}

func TestRenderDotenvSpecTypeProposals(t *testing.T) {
	t.Parallel()

	rendered := renderDotenvSpecTypeProposals([]owlcmd.TypeProposal{
		{Key: "API_KEY", SuggestedType: "core/secret", Description: "Api Key"},
		{Key: "SERVICE_HOST", SuggestedType: "core/host", Description: "Service Host"},
	})

	assert.Equal(t, "API_KEY=\"Api Key\" # Secret\nSERVICE_HOST=\"Service Host\" # Host\n", rendered)
}

func owlSnapshotEnv(name string, typeID string) owlcmd.SnapshotEnv {
	return owlcmd.SnapshotEnv{Name: name, Type: typeID}
}
