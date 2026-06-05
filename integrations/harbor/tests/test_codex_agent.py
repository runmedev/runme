from pathlib import Path

from runme_harbor.codex_agent import RunmeCodexAgent


def test_codex_agent_prefers_default_auth_json_over_openai_api_key(
    tmp_path: Path,
    monkeypatch,
) -> None:
    home = tmp_path / "home"
    auth_json = home / ".codex" / "auth.json"
    auth_json.parent.mkdir(parents=True)
    auth_json.write_text("{}")

    monkeypatch.setenv("HOME", str(home))
    monkeypatch.setenv("OPENAI_API_KEY", "ignored")
    monkeypatch.delenv("CODEX_AUTH_JSON_PATH", raising=False)
    monkeypatch.delenv("CODEX_FORCE_AUTH_JSON", raising=False)

    agent = RunmeCodexAgent(logs_dir=tmp_path)

    assert agent._resolve_preferred_auth_json_path() == auth_json


def test_codex_agent_explicit_auth_json_wins(
    tmp_path: Path,
    monkeypatch,
) -> None:
    home_auth_json = tmp_path / "home" / ".codex" / "auth.json"
    explicit_auth_json = tmp_path / "explicit" / "auth.json"
    home_auth_json.parent.mkdir(parents=True)
    explicit_auth_json.parent.mkdir(parents=True)
    home_auth_json.write_text("{}")
    explicit_auth_json.write_text("{}")

    monkeypatch.setenv("HOME", str(tmp_path / "home"))
    monkeypatch.setenv("CODEX_AUTH_JSON_PATH", str(explicit_auth_json))
    monkeypatch.setenv("OPENAI_API_KEY", "ignored")

    agent = RunmeCodexAgent(logs_dir=tmp_path)

    assert agent._resolve_preferred_auth_json_path() == explicit_auth_json
