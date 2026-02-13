package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/runmedev/runme/v3/pkg/agent/api"
)

// AddIAMPolicyBinding adds an IAM policy binding to the config file.
// It uses kyaml to preserve unrecognized sections in the YAML.
func (ac *AppConfig) AddIAMPolicyBinding(role string, memberExpr string) (bool, error) {
	member, err := parseIAMMember(memberExpr)
	if err != nil {
		return false, err
	}

	cfgFile := ac.GetConfigFile()
	if cfgFile == "" {
		return false, errors.New("no config file specified")
	}

	root, err := readYAMLConfig(cfgFile)
	if err != nil {
		return false, err
	}

	bindingsNode, err := root.Pipe(yaml.LookupCreate(yaml.SequenceNode, "iamPolicy", "bindings"))
	if err != nil {
		return false, errors.Wrap(err, "failed to access iamPolicy.bindings")
	}

	existingBindings, err := bindingsNode.Elements()
	if err != nil {
		return false, errors.Wrap(err, "failed to read iamPolicy.bindings")
	}

	for _, binding := range existingBindings {
		existingRole, err := stringFieldValue(binding, "role")
		if err != nil {
			return false, err
		}
		if existingRole != role {
			continue
		}

		membersNode, err := binding.Pipe(yaml.LookupCreate(yaml.SequenceNode, "members"))
		if err != nil {
			return false, errors.Wrap(err, "failed to access members for existing role")
		}
		existingMembers, err := membersNode.Elements()
		if err != nil {
			return false, errors.Wrap(err, "failed to read members for existing role")
		}

		for _, m := range existingMembers {
			kind, err := stringFieldValue(m, "kind")
			if err != nil {
				return false, err
			}
			name, err := stringFieldValue(m, "name")
			if err != nil {
				return false, err
			}
			if kind == string(member.Kind) && name == member.Name {
				return false, nil
			}
		}

		memberNode, err := newMemberNode(member)
		if err != nil {
			return false, err
		}
		if err := membersNode.PipeE(yaml.Append(memberNode.YNode())); err != nil {
			return false, errors.Wrap(err, "failed to append member to existing role binding")
		}

		return true, writeYAMLConfig(cfgFile, root)
	}

	bindingNode, err := newBindingNode(role, member)
	if err != nil {
		return false, err
	}
	if err := bindingsNode.PipeE(yaml.Append(bindingNode.YNode())); err != nil {
		return false, errors.Wrap(err, "failed to append role binding")
	}

	return true, writeYAMLConfig(cfgFile, root)
}

func readYAMLConfig(cfgFile string) (*yaml.RNode, error) {
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		if os.IsNotExist(err) {
			return yaml.MustParse("{}"), nil
		}
		return nil, errors.Wrapf(err, "failed to read config file %q", cfgFile)
	}

	if strings.TrimSpace(string(data)) == "" {
		return yaml.MustParse("{}"), nil
	}

	root, err := yaml.Parse(string(data))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse config file %q", cfgFile)
	}
	return root, nil
}

func writeYAMLConfig(cfgFile string, root *yaml.RNode) error {
	configDir := filepath.Dir(cfgFile)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return errors.Wrapf(err, "failed to create config directory %q", configDir)
	}

	out, err := root.String()
	if err != nil {
		return errors.Wrap(err, "failed to serialize config")
	}

	return os.WriteFile(cfgFile, []byte(out), 0o600)
}

func parseIAMMember(memberExpr string) (api.Member, error) {
	parts := strings.SplitN(strings.TrimSpace(memberExpr), ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return api.Member{}, errors.Errorf("invalid --member value %q; expected <kind>:<name>", memberExpr)
	}

	kindRaw := strings.ToLower(strings.TrimSpace(parts[0]))
	name := strings.TrimSpace(parts[1])
	if name == "" {
		return api.Member{}, errors.Errorf("invalid --member value %q; member name must be non-empty", memberExpr)
	}

	var kind api.MemberKind
	switch api.MemberKind(kindRaw) {
	case api.UserKind:
		kind = api.UserKind
	case api.DomainKind:
		kind = api.DomainKind
	default:
		return api.Member{}, errors.Errorf("invalid member kind %q; must be one of: %q, %q", kindRaw, api.UserKind, api.DomainKind)
	}

	return api.Member{
		Kind: kind,
		Name: name,
	}, nil
}

func newMemberNode(member api.Member) (*yaml.RNode, error) {
	memberNode := yaml.MustParse("{}")
	if err := memberNode.PipeE(yaml.SetField("kind", yaml.NewStringRNode(string(member.Kind)))); err != nil {
		return nil, errors.Wrap(err, "failed to set member kind")
	}
	if err := memberNode.PipeE(yaml.SetField("name", yaml.NewStringRNode(member.Name))); err != nil {
		return nil, errors.Wrap(err, "failed to set member name")
	}
	return memberNode, nil
}

func newBindingNode(role string, member api.Member) (*yaml.RNode, error) {
	memberNode, err := newMemberNode(member)
	if err != nil {
		return nil, err
	}

	bindingNode := yaml.MustParse("{}")
	if err := bindingNode.PipeE(yaml.SetField("role", yaml.NewStringRNode(role))); err != nil {
		return nil, errors.Wrap(err, "failed to set role")
	}

	membersNode := yaml.NewRNode(&yaml.Node{Kind: yaml.SequenceNode})
	if err := membersNode.PipeE(yaml.Append(memberNode.YNode())); err != nil {
		return nil, errors.Wrap(err, "failed to build members list")
	}
	if err := bindingNode.PipeE(yaml.SetField("members", membersNode)); err != nil {
		return nil, errors.Wrap(err, "failed to set members")
	}

	return bindingNode, nil
}

func stringFieldValue(node *yaml.RNode, field string) (string, error) {
	fieldNode, err := node.Pipe(yaml.Lookup(field))
	if err != nil {
		return "", errors.Wrapf(err, "failed to lookup field %q", field)
	}
	if fieldNode == nil || fieldNode.YNode() == nil {
		return "", nil
	}

	if fieldNode.YNode().Kind != yaml.ScalarNode {
		return "", fmt.Errorf("expected field %q to be scalar", field)
	}

	return strings.TrimSpace(fieldNode.YNode().Value), nil
}
