import io
import json

import pytest

from runme_harbor.environment import HarborProtocolError, RunmeEnvironment


class FakeProcess:
    def __init__(self, lines: list[dict[str, object]]) -> None:
        self.stdin = io.StringIO()
        self.stdout = io.StringIO("".join(json.dumps(line) + "\n" for line in lines))
        self.stderr = io.StringIO()
        self.killed = False
        self.returncode = None

    def wait(self, timeout: float | None = None) -> int:
        self.returncode = 0
        return 0

    def kill(self) -> None:
        self.killed = True


def test_environment_starts_runme_harbor_stdio(monkeypatch: pytest.MonkeyPatch) -> None:
    calls = []
    fake = FakeProcess([{"id": "1", "preflight": {"protocol": "runme.harbor.stdio"}}])

    def popen(command, **kwargs):
        calls.append((command, kwargs))
        return fake

    monkeypatch.setattr("subprocess.Popen", popen)

    env = RunmeEnvironment()
    assert env.preflight()["protocol"] == "runme.harbor.stdio"
    assert calls[0][0] == ["runme", "harbor", "stdio"]
    assert calls[0][1]["stdin"] == -1
    assert calls[0][1]["stdout"] == -1
    assert calls[0][1]["stderr"] == -1

    sent = json.loads(fake.stdin.getvalue().strip())
    assert sent == {"id": "1", "preflight": {}}


def test_protocol_error_raises_runtime_error(monkeypatch: pytest.MonkeyPatch) -> None:
    fake = FakeProcess([{"id": "custom", "error": {"code": "invalid_argument", "message": "bad"}}])
    monkeypatch.setattr("subprocess.Popen", lambda *_args, **_kwargs: fake)

    env = RunmeEnvironment()
    with pytest.raises(HarborProtocolError) as exc:
        env.request({"preflight": {}}, request_id="custom")

    assert exc.value.code == "invalid_argument"
    assert "bad" in str(exc.value)


def test_request_id_mismatch_raises(monkeypatch: pytest.MonkeyPatch) -> None:
    fake = FakeProcess([{"id": "other", "preflight": {}}])
    monkeypatch.setattr("subprocess.Popen", lambda *_args, **_kwargs: fake)

    env = RunmeEnvironment()
    with pytest.raises(HarborProtocolError) as exc:
        env.request({"preflight": {}}, request_id="expected")

    assert exc.value.code == "request_id_mismatch"
