import os
import shlex
import shutil
from pathlib import Path

from harbor.agents.installed.base import with_prompt_template
from harbor.agents.installed.claude_code import ClaudeCode
from harbor.environments.base import BaseEnvironment
from harbor.models.agent.context import AgentContext
from harbor.models.trial.paths import EnvironmentPaths


class LocalClaudeCode(ClaudeCode):
    """Claude Code-backed local agent without container bootstrap.

    Harbor's installed Claude Code agent assumes a disposable container and
    prepares Claude config, credentials, skills, memory, and MCP servers before
    running. Runme Harbor executes against a local runtime, so this wrapper
    expects `claude` to already be available and configured.
    """

    @staticmethod
    def name() -> str:
        return "claude-code"

    async def install(self, environment: BaseEnvironment) -> None:
        return None

    async def setup(self, environment: BaseEnvironment) -> None:
        return None

    @staticmethod
    def _claude_config_dir() -> Path:
        claude_config_dir = os.environ.get("CLAUDE_CONFIG_DIR")
        if claude_config_dir:
            return Path(claude_config_dir).expanduser()
        return Path.home() / ".claude"

    def _snapshot_session_files(self) -> set[Path]:
        config_dir = self._claude_config_dir()
        if not config_dir.exists():
            return set()
        return {path for path in config_dir.rglob("*.jsonl") if path.is_file()}

    def _collect_new_sessions(self, before: set[Path]) -> None:
        config_dir = self._claude_config_dir()
        if not config_dir.exists():
            return

        after = {path for path in config_dir.rglob("*.jsonl") if path.is_file()}
        new_sessions = after - before
        if not new_sessions:
            return

        target_dir = self.logs_dir / "sessions"
        if target_dir.exists():
            shutil.rmtree(target_dir)

        for session_file in new_sessions:
            target = target_dir / session_file.relative_to(config_dir)
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(session_file, target)

    def _model_arg(self) -> str:
        if not self.model_name:
            return ""
        if os.environ.get("ANTHROPIC_BASE_URL"):
            model = self.model_name
        elif self._is_bedrock_mode() and "/" in self.model_name:
            model = self.model_name.split("/", 1)[-1]
        else:
            model = self.model_name.split("/")[-1]
        return f"--model {shlex.quote(model)} "

    def _runtime_env(self) -> dict[str, str]:
        env = {
            "FORCE_AUTO_BACKGROUND_TASKS": "1",
            "ENABLE_BACKGROUND_TASKS": "1",
            "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
            "IS_SANDBOX": "1",
        }

        if max_output_tokens := os.environ.get("CLAUDE_CODE_MAX_OUTPUT_TOKENS"):
            env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = max_output_tokens

        if os.environ.get("CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING", "").strip() == "1":
            env["CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING"] = "1"

        env.update(self._resolved_env_vars)
        return env

    @with_prompt_template
    async def run(
        self,
        instruction: str,
        environment: BaseEnvironment,
        context: AgentContext,
    ) -> None:
        escaped_instruction = shlex.quote(instruction)
        model_arg = self._model_arg()
        cli_flags = self.build_cli_flags()
        cli_flags_arg = f"{cli_flags} " if cli_flags else ""
        session_files_before = self._snapshot_session_files()
        env = self._runtime_env()

        try:
            await self.exec_as_agent(
                environment,
                command=(
                    "set -o pipefail\n"
                    "claude --verbose --output-format=stream-json "
                    "--permission-mode=bypassPermissions "
                    f"{model_arg}"
                    f"{cli_flags_arg}"
                    f"--print -- {escaped_instruction} "
                    f"2>&1 </dev/null | tee "
                    f"{EnvironmentPaths.agent_dir / 'claude-code.txt'}"
                ),
                env=env,
            )
        finally:
            try:
                self._collect_new_sessions(session_files_before)
            except Exception:
                pass

            self.populate_context_post_run(context)
