import os
import shlex
import shutil
from pathlib import Path

from harbor.agents.installed.base import with_prompt_template
from harbor.agents.installed.codex import Codex
from harbor.environments.base import BaseEnvironment
from harbor.models.agent.context import AgentContext
from harbor.models.trial.paths import EnvironmentPaths


class RunmeCodexAgent(Codex):
    """Codex-backed Runme agent without container bootstrap.

    Harbor's installed Codex agent assumes a disposable container and mutates
    that environment during setup by installing system packages, Node, and the
    Codex CLI. Runme Harbor executes against a local Runme runtime, so this
    wrapper expects `codex` to already be available and skips setup-time
    environment changes.
    """

    @staticmethod
    def name() -> str:
        return "runme-codex"

    async def install(self, environment: BaseEnvironment) -> None:
        return None

    async def setup(self, environment: BaseEnvironment) -> None:
        return None

    @staticmethod
    def _codex_sessions_dir() -> Path:
        codex_home = os.environ.get("CODEX_HOME")
        if codex_home:
            return Path(codex_home).expanduser() / "sessions"
        return Path.home() / ".codex" / "sessions"

    def _snapshot_session_files(self) -> set[Path]:
        sessions_dir = self._codex_sessions_dir()
        if not sessions_dir.exists():
            return set()
        return {path for path in sessions_dir.rglob("*.jsonl") if path.is_file()}

    def _collect_new_sessions(self, before: set[Path]) -> None:
        sessions_dir = self._codex_sessions_dir()
        if not sessions_dir.exists():
            return

        after = {path for path in sessions_dir.rglob("*.jsonl") if path.is_file()}
        new_sessions = after - before
        if not new_sessions:
            return

        target_dir = self.logs_dir / "sessions"
        if target_dir.exists():
            shutil.rmtree(target_dir)

        for session_file in new_sessions:
            target = target_dir / session_file.relative_to(sessions_dir)
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(session_file, target)

    @with_prompt_template
    async def run(
        self,
        instruction: str,
        environment: BaseEnvironment,
        context: AgentContext,
    ) -> None:
        escaped_instruction = shlex.quote(instruction)
        model = self.model_name.split("/")[-1] if self.model_name else None
        model_arg = f"--model {shlex.quote(model)} " if model else ""

        cli_flags = self.build_cli_flags()
        cli_flags_arg = f"{cli_flags} " if cli_flags else ""
        session_files_before = self._snapshot_session_files()

        try:
            await self.exec_as_agent(
                environment,
                command=(
                    "set -o pipefail\n"
                    "codex exec "
                    "--dangerously-bypass-approvals-and-sandbox "
                    "--skip-git-repo-check "
                    f"{model_arg}"
                    "--json "
                    "--enable unified_exec "
                    f"{cli_flags_arg}"
                    "-- "
                    f"{escaped_instruction} "
                    f"2>&1 </dev/null | tee "
                    f"{EnvironmentPaths.agent_dir / self._OUTPUT_FILENAME}"
                ),
            )
        finally:
            try:
                self._collect_new_sessions(session_files_before)
            except Exception:
                pass

            self.populate_context_post_run(context)
