package cmd

import (
	"os"
	"path/filepath"
	"strings"
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

	suggested, reason, ok := suggestSnapshotPrimitiveType(owlSnapshotEnv("API_KEY", "https://owl.runme.dev/v1/types/core/opaque"))
	assert.Equal(t, "core/secret", suggested)
	assert.Equal(t, "key name suggests sensitive value", reason)
	assert.True(t, ok)

	suggested, reason, ok = suggestSnapshotPrimitiveType(owlSnapshotEnv("SERVICE_HOST", "core/opaque"))
	assert.Equal(t, "core/host", suggested)
	assert.Equal(t, "key name suggests host", reason)
	assert.True(t, ok)

	suggested, reason, ok = suggestSnapshotPrimitiveType(owlSnapshotEnv("TARGET_PLATFORM", "core/opaque"))
	assert.Empty(t, suggested)
	assert.Equal(t, "no primitive type heuristic matched", reason)
	assert.False(t, ok)

	assert.Equal(t, "core/opaque", normalizeSnapshotType("https://owl.runme.dev/v1/types/core/opaque"))
	assert.Equal(t, "Host", dotenvSpecTypeName("core/host"))
}

func TestIncludeSnapshotTypeProposalSkipsNoSuggestionUnlessAll(t *testing.T) {
	t.Parallel()

	assert.False(t, includeSnapshotTypeProposal(owlcmd.TypeRequest{}, false))
	assert.True(t, includeSnapshotTypeProposal(owlcmd.TypeRequest{All: true}, false))
	assert.True(t, includeSnapshotTypeProposal(owlcmd.TypeRequest{}, true))
}

func TestRenderDotenvSpecTypeProposals(t *testing.T) {
	t.Parallel()

	rendered := renderDotenvSpecTypeProposals([]owlcmd.TypeProposal{
		{Key: "GITHUB_TOKEN", SuggestedType: "core/secret", Description: "The GitHub token to use for API requests."},
		{Key: "RUNME_TEST_TOKEN", SuggestedType: "core/secret", Description: "The Runme test token to use for integration tests."},
		{Key: "TARGET_PLATFORM", SuggestedType: "", Description: "Target Platform"},
		{Key: "SERVICE_HOST", SuggestedType: "core/host", Description: "Service Host"},
	})

	assert.Equal(t, strings.Join([]string{
		`GITHUB_TOKEN="The GitHub token to use for API requests."              # Secret`,
		`RUNME_TEST_TOKEN="The Runme test token to use for integration tests." # Secret`,
		`SERVICE_HOST="Service Host"                                           # Host`,
		"",
	}, "\n"), rendered)
}

func TestMaterializeDotenvSpecTypeProposals(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specFile := filepath.Join(dir, ".env.spec")
	assert.NoError(t, os.WriteFile(specFile, []byte(`API_URL="API URL" # Plain`), 0o600))

	materialized, err := materializeDotenvSpecTypeProposals(specFile, "API_KEY=\"Api Key\" # Secret\n")
	assert.NoError(t, err)
	assert.Equal(t, "API_URL=\"API URL\" # Plain\nAPI_KEY=\"Api Key\" # Secret\n", materialized)
}

func owlSnapshotEnv(name string, typeID string) owlcmd.SnapshotEnv {
	return owlcmd.SnapshotEnv{Name: name, Type: typeID}
}
