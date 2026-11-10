from pathlib import Path
from types import SimpleNamespace

import pytest

from harbor.agents.factory import AgentFactory
from harbor.agents.options import compile_cli_from_options, compile_env_from_options
from harbor.models.trial.config import AgentConfig
from runme_harbor.runme_agents import RunmeAntigravityCli, RunmeClaudeCode, RunmeCodex


@pytest.fixture(params=[RunmeCodex, RunmeClaudeCode, RunmeAntigravityCli])
def agent_class(request):
    return request.param


@pytest.mark.parametrize(
    ("kwargs", "expected"),
    [
        ({}, False),
        ({"route_oauth_credentials": False}, False),
        ({"route_oauth_credentials": True}, True),
        ({"route_oauth_credentials": "false"}, False),
        ({"route_oauth_credentials": "true"}, True),
        ({"route_oauth_credentials": "  FaLsE\t"}, False),
        ({"route_oauth_credentials": "\nTrUe  "}, True),
    ],
)
def test_oauth_routing_options_match_runtime(agent_class, kwargs, expected, tmp_path: Path):
    options = agent_class.parse_options(kwargs)
    agent_class.preflight(kwargs)
    agent = agent_class(logs_dir=tmp_path, **kwargs)

    assert options.route_oauth_credentials is expected
    assert agent.route_oauth_credentials is expected
    assert agent.options.route_oauth_credentials is expected
    # The adapter policy must not produce native CLI flags or environment variables.
    defaults = agent_class.parse_options({})
    assert compile_cli_from_options(options) == compile_cli_from_options(defaults)
    assert compile_env_from_options(options) == compile_env_from_options(defaults)


@pytest.mark.parametrize("value", ["yes", "no", "0", "1", "", 0, 1, 0.0, None, [], {}])
def test_oauth_routing_rejects_invalid_values(agent_class, value, tmp_path: Path):
    kwargs = {"route_oauth_credentials": value}
    for validate in (agent_class.parse_options, agent_class.preflight):
        with pytest.raises(ValueError, match="route_oauth_credentials must be true or false"):
            validate(kwargs)
    with pytest.raises(ValueError, match="route_oauth_credentials must be true or false"):
        agent_class(logs_dir=tmp_path, **kwargs)


def test_oauth_routing_keeps_unknown_option_validation(agent_class, tmp_path: Path):
    kwargs = {"route_oauth_credentials": False, "unknown_option": True}
    for validate in (agent_class.parse_options, agent_class.preflight):
        with pytest.raises(ValueError, match="Unknown option 'unknown_option'"):
            validate(kwargs)
    with pytest.raises(ValueError, match="Unknown option 'unknown_option'"):
        agent_class(logs_dir=tmp_path, **kwargs)


def test_oauth_routing_schema_preserves_inherited_options(agent_class, tmp_path: Path):
    schema = agent_class.options_schema()
    field = schema["properties"]["route_oauth_credentials"]
    assert field["type"] == "boolean"
    assert field["default"] is False
    assert schema["additionalProperties"] is False

    # Exercise a parent-specific option and its CLI metadata, not only shared fields.
    inherited = {
        RunmeCodex: ("reasoning_summary", "concise", "model_reasoning_summary=concise"),
        RunmeClaudeCode: ("max_turns", 3, "--max-turns 3"),
        RunmeAntigravityCli: ("sandbox", True, "--sandbox"),
    }
    name, value, flag = inherited[agent_class]
    kwargs = {name: value, "route_oauth_credentials": True}
    agent_class.preflight(kwargs)
    agent = agent_class(logs_dir=tmp_path, **kwargs)
    assert getattr(agent.options, name) == value
    assert flag in agent.build_cli_flags()


@pytest.mark.parametrize("value", [False, True, "false", "true"])
def test_factory_preflight_and_construction_accept_routing_policy(
    agent_class, value, tmp_path: Path, monkeypatch
):
    config = AgentConfig(
        import_path=f"{agent_class.__module__}:{agent_class.__name__}",
        kwargs={"route_oauth_credentials": value},
    )
    monkeypatch.setattr(
        "harbor.agents.factory.resolve_agent_model_connection",
        lambda *_args, **_kwargs: SimpleNamespace(execution_base_url=None),
    )

    AgentFactory.run_preflight(config, skip_credential_checks=True)
    agent = AgentFactory.create_agent_from_config(config, logs_dir=tmp_path)

    expected = value is True or value == "true"
    assert agent.route_oauth_credentials is expected
    assert agent.options.route_oauth_credentials is expected
