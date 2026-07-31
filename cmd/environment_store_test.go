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
		{Name: "OPENAI_API_KEY", Spec: "github.com/runmedev/owl/types/core/opaque"},
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
	assert.False(t, byName["OPENAI_API_KEY"])
	assert.False(t, byName["PATH"])
	assert.False(t, byName["HOME"])
}

func TestSnapshotEnvsFromProtoMapsVisibilityAndDisplayValue(t *testing.T) {
	t.Parallel()

	envs := snapshotEnvsFromProto([]*runnerv1.MonitorEnvStoreResponseSnapshot_SnapshotEnv{
		{
			Name:   "RUNME_TEST_TOKEN",
			Spec:   "github.com/runmedev/owl/types/core/secret",
			Status: runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_UNSPECIFIED,
		},
		{
			Name:          "GITHUB_TOKEN",
			Spec:          "https://owl.runme.dev/v1/types/core/secret",
			ResolvedValue: "[masked]",
			Status:        runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_MASKED,
		},
		{
			Name:          "API_URL",
			Spec:          "https://owl.runme.dev/v1/types/core/plain",
			ResolvedValue: "https://api.example.com",
			Status:        runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_LITERAL,
		},
		{
			Name:   "DATABASE_URL",
			Spec:   "https://owl.runme.dev/v1/types/core/opaque",
			Status: runnerv1.MonitorEnvStoreResponseSnapshot_STATUS_HIDDEN,
		},
	})

	byName := make(map[string]owlcmd.SnapshotEnv, len(envs))
	for _, env := range envs {
		byName[env.Name] = env
	}

	assert.Equal(t, "[unset]", byName["RUNME_TEST_TOKEN"].Value)
	assert.Equal(t, "core/secret", byName["RUNME_TEST_TOKEN"].Type)
	assert.Equal(t, "unresolved", byName["RUNME_TEST_TOKEN"].Visibility)
	assert.Equal(t, "[masked]", byName["GITHUB_TOKEN"].Value)
	assert.Equal(t, "masked", byName["GITHUB_TOKEN"].Visibility)
	assert.Equal(t, "https://api.example.com", byName["API_URL"].Value)
	assert.Equal(t, "literal", byName["API_URL"].Visibility)
	assert.Equal(t, "[hidden]", byName["DATABASE_URL"].Value)
	assert.Equal(t, "core/opaque", byName["DATABASE_URL"].Type)
	assert.Equal(t, "hidden", byName["DATABASE_URL"].Visibility)
}

func TestSnapshotTypeProposalHelpers(t *testing.T) {
	t.Parallel()

	suggested, reason, ok := suggestSnapshotPrimitiveType(owlSnapshotEnv("API_KEY", "https://owl.runme.dev/v1/types/core/opaque"))
	assert.Equal(t, "core/secret", suggested)
	assert.Equal(t, "key name suggests sensitive value", reason)
	assert.True(t, ok)

	suggested, reason, ok = suggestSnapshotPrimitiveType(owlSnapshotEnv("SERVICE_HOST", "core/opaque"))
	assert.Empty(t, suggested)
	assert.Equal(t, "no primitive type heuristic matched", reason)
	assert.False(t, ok)

	suggested, reason, ok = suggestSnapshotPrimitiveType(owlSnapshotEnv("TARGET_PLATFORM", "core/opaque"))
	assert.Empty(t, suggested)
	assert.Equal(t, "no primitive type heuristic matched", reason)
	assert.False(t, ok)

	assert.Equal(t, "core/opaque", normalizeSnapshotType("https://owl.runme.dev/v1/types/core/opaque"))
	assert.Equal(t, "core/opaque", normalizeSnapshotType("github.com/runmedev/owl/types/core/opaque"))
	assert.Equal(t, "universe/anthropic", normalizeSnapshotType("github.com/runmedev/owl/types/universe/anthropic"))
	assert.Equal(t, "Opaque", dotenvSpecTypeName("core/host"))
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
		{Key: "SERVICE_HOST", SuggestedType: "", Description: "Service Host"},
	})

	assert.Equal(t, strings.Join([]string{
		`GITHUB_TOKEN="The GitHub token to use for API requests."              # Secret`,
		`RUNME_TEST_TOKEN="The Runme test token to use for integration tests." # Secret`,
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
	assert.Equal(t, "API_URL=\"API URL\" # Plain\n\nAPI_KEY=\"Api Key\" # Secret\n", materialized)

	assert.NoError(t, os.WriteFile(specFile, []byte("API_URL=\"API URL\" # Plain\n"), 0o600))
	materialized, err = materializeDotenvSpecTypeProposals(specFile, "API_KEY=\"Api Key\" # Secret\n")
	assert.NoError(t, err)
	assert.Equal(t, "API_URL=\"API URL\" # Plain\n\nAPI_KEY=\"Api Key\" # Secret\n", materialized)

	assert.NoError(t, os.WriteFile(specFile, []byte("API_URL=\"API URL\" # Plain\n\n"), 0o600))
	materialized, err = materializeDotenvSpecTypeProposals(specFile, "API_KEY=\"Api Key\" # Secret\n")
	assert.NoError(t, err)
	assert.Equal(t, "API_URL=\"API URL\" # Plain\n\nAPI_KEY=\"Api Key\" # Secret\n", materialized)
}

func owlSnapshotEnv(name string, typeID string) owlcmd.SnapshotEnv {
	return owlcmd.SnapshotEnv{Name: name, Type: typeID}
}
