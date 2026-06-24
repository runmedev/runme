from __future__ import annotations

import tomllib
from pathlib import Path

import pytest

from runme_harbor import cli
from runme_harbor import metadata_sync


def test_main_sync_metadata_command_uses_default_jobs_dir(
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    synced: list[str] = []
    preflighted: list[bool] = []
    monkeypatch.setattr(cli, "_preflight_harbor_package", lambda: preflighted.append(True))
    monkeypatch.setattr(
        metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 3
    )

    assert cli.main(["sync-metadata"]) == 0

    assert preflighted == [True]
    assert synced == [".runme/evals/jobs"]
    assert capsys.readouterr().out == "Synced Harbor metadata for 3 job(s).\n"


def test_main_sync_metadata_command_accepts_jobs_dir(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    synced: list[str] = []
    monkeypatch.setattr(cli, "_preflight_harbor_package", lambda: None)
    monkeypatch.setattr(
        metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 1
    )

    assert cli.main(["sync-metadata", "--jobs-dir", "jobs"]) == 0

    assert synced == ["jobs"]


def test_main_sync_metadata_command_reports_preflight_errors(
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    def fail_preflight() -> None:
        raise SystemExit("Runme Harbor requires the `harbor` Python package.")

    synced: list[str] = []
    monkeypatch.setattr(cli, "_preflight_harbor_package", fail_preflight)
    monkeypatch.setattr(
        metadata_sync, "sync_jobs_metadata", lambda jobs_dir: synced.append(jobs_dir) or 1
    )

    assert cli.main(["sync-metadata"]) == 1

    assert synced == []
    assert capsys.readouterr().err == "Runme Harbor requires the `harbor` Python package.\n"


def test_run_command_is_not_exposed(capsys: pytest.CaptureFixture[str]) -> None:
    assert cli.main(["run", "dataset"]) == 2

    assert "invalid choice: 'run'" in capsys.readouterr().err


@pytest.mark.parametrize(
    ("version", "supported"),
    [
        ("0.14.0", False),
        ("0.15.0", True),
        ("0.15.9", True),
        ("0.16.0", False),
    ],
)
def test_preflight_harbor_version_range(
    monkeypatch: pytest.MonkeyPatch,
    version: str,
    supported: bool,
) -> None:
    monkeypatch.setattr(cli.importlib, "import_module", lambda _name: object())
    monkeypatch.setattr(cli.importlib.metadata, "version", lambda _name: version)

    if supported:
        cli._preflight_harbor_package()
    else:
        with pytest.raises(SystemExit, match="harbor>=0.15,<0.16"):
            cli._preflight_harbor_package()


def test_pyproject_exposes_runme_owned_scripts_only() -> None:
    pyproject = Path(__file__).parents[1] / "pyproject.toml"
    scripts = tomllib.loads(pyproject.read_text())["project"]["scripts"]

    assert scripts["runme-harbor"] == "runme_harbor.cli:main"
    assert scripts["runme-harbor-harbor"] == "harbor.cli.main:app"
    assert "harbor" not in scripts
    assert "hr" not in scripts
    assert "hb" not in scripts
