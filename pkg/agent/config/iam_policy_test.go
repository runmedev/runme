package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/viper"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestAddIAMPolicyBinding_AddAndPreserve(t *testing.T) {
	tDir := t.TempDir()
	configPath := filepath.Join(tDir, "config.yaml")
	input := `
commands:
  - name: setup
    command: echo setup
`
	if err := os.WriteFile(configPath, []byte(strings.TrimLeft(input, "\n")), 0o600); err != nil {
		t.Fatalf("failed to write config fixture: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	ac, err := NewAppConfig("runme-agent-test", WithViperInstance(v, nil))
	if err != nil {
		t.Fatalf("failed to initialize app config: %v", err)
	}

	added, err := ac.AddIAMPolicyBinding("role/runner.user", "user:jlewi@openai.com")
	if err != nil {
		t.Fatalf("failed to add IAM binding: %v", err)
	}
	if !added {
		t.Fatalf("expected binding to be added")
	}

	root, err := yaml.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read resulting yaml: %v", err)
	}

	commandName, err := root.GetString("commands.[0].name")
	if err != nil {
		t.Fatalf("failed to lookup preserved command name: %v", err)
	}
	if commandName != "setup" {
		t.Fatalf("expected commands to be preserved; got %q", commandName)
	}

	roleNode, err := root.Pipe(yaml.Lookup("iamPolicy", "bindings", "0", "role"))
	if err != nil {
		t.Fatalf("failed to lookup role: %v", err)
	}
	role := roleNode.YNode().Value
	if role != "role/runner.user" {
		t.Fatalf("unexpected role: %q", role)
	}

	memberKindNode, err := root.Pipe(yaml.Lookup("iamPolicy", "bindings", "0", "members", "0", "kind"))
	if err != nil {
		t.Fatalf("failed to lookup member kind: %v", err)
	}
	memberKind := memberKindNode.YNode().Value
	if memberKind != "user" {
		t.Fatalf("unexpected member kind: %q", memberKind)
	}

	memberNameNode, err := root.Pipe(yaml.Lookup("iamPolicy", "bindings", "0", "members", "0", "name"))
	if err != nil {
		t.Fatalf("failed to lookup member name: %v", err)
	}
	memberName := memberNameNode.YNode().Value
	if memberName != "jlewi@openai.com" {
		t.Fatalf("unexpected member name: %q", memberName)
	}
}

func TestAddIAMPolicyBinding_AppendsAndDedupes(t *testing.T) {
	tDir := t.TempDir()
	configPath := filepath.Join(tDir, "config.yaml")
	input := `
iamPolicy:
  bindings:
    - role: role/runner.user
      members:
        - kind: user
          name: jlewi@openai.com
`
	if err := os.WriteFile(configPath, []byte(strings.TrimLeft(input, "\n")), 0o600); err != nil {
		t.Fatalf("failed to write config fixture: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	ac, err := NewAppConfig("runme-agent-test", WithViperInstance(v, nil))
	if err != nil {
		t.Fatalf("failed to initialize app config: %v", err)
	}

	added, err := ac.AddIAMPolicyBinding("role/runner.user", "domain:openai.com")
	if err != nil {
		t.Fatalf("failed to append member: %v", err)
	}
	if !added {
		t.Fatalf("expected domain member to be appended")
	}

	added, err = ac.AddIAMPolicyBinding("role/runner.user", "domain:openai.com")
	if err != nil {
		t.Fatalf("failed to check dedupe: %v", err)
	}
	if added {
		t.Fatalf("expected duplicate member not to be added")
	}

	root, err := yaml.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read resulting yaml: %v", err)
	}

	bindingsNode, err := root.Pipe(yaml.Lookup("iamPolicy", "bindings"))
	if err != nil {
		t.Fatalf("failed to find bindings: %v", err)
	}
	bindings, err := bindingsNode.Elements()
	if err != nil {
		t.Fatalf("failed to read bindings: %v", err)
	}

	memberSet := map[string]bool{}
	for _, binding := range bindings {
		roleNode, err := binding.Pipe(yaml.Lookup("role"))
		if err != nil {
			t.Fatalf("failed to lookup role in binding: %v", err)
		}
		if roleNode == nil || roleNode.YNode().Value != "role/runner.user" {
			continue
		}

		membersNode, err := binding.Pipe(yaml.Lookup("members"))
		if err != nil {
			t.Fatalf("failed to lookup members in binding: %v", err)
		}
		members, err := membersNode.Elements()
		if err != nil {
			t.Fatalf("failed to read members in binding: %v", err)
		}

		for _, member := range members {
			kindNode, err := member.Pipe(yaml.Lookup("kind"))
			if err != nil {
				t.Fatalf("failed to lookup member kind: %v", err)
			}
			nameNode, err := member.Pipe(yaml.Lookup("name"))
			if err != nil {
				t.Fatalf("failed to lookup member name: %v", err)
			}
			memberSet[kindNode.YNode().Value+":"+nameNode.YNode().Value] = true
		}
	}

	if len(memberSet) != 2 {
		t.Fatalf("expected exactly 2 unique members, got %d", len(memberSet))
	}
	if !memberSet["user:jlewi@openai.com"] || !memberSet["domain:openai.com"] {
		t.Fatalf("expected user and domain members; got %+v", memberSet)
	}
}

func TestParseIAMMember(t *testing.T) {
	type testCase struct {
		name      string
		memberArg string
		want      string
		wantErr   string
	}

	cases := []testCase{
		{
			name:      "user",
			memberArg: "user:test@example.com",
			want:      "user:test@example.com",
		},
		{
			name:      "domain",
			memberArg: "domain:openai.com",
			want:      "domain:openai.com",
		},
		{
			name:      "invalid-kind",
			memberArg: "group:eng@example.com",
			wantErr:   "invalid member kind",
		},
		{
			name:      "invalid-format",
			memberArg: "jlewi@openai.com",
			wantErr:   "expected <kind>:<name>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			member, err := parseIAMMember(tc.memberArg)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("unexpected error; got %q want substring %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := string(member.Kind) + ":" + member.Name
			if d := cmp.Diff(tc.want, got); d != "" {
				t.Fatalf("unexpected member (-want +got):\n%s", d)
			}
		})
	}
}
