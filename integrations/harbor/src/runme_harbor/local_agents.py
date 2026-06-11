import os
import shlex
import shutil
from hashlib import sha256
from pathlib import Path

from harbor.agents.installed.base import CliFlag, with_prompt_template
from harbor.agents.installed.claude_code import ClaudeCode
from harbor.agents.installed.codex import Codex
from harbor.agents.installed.openclaw import OpenClaw
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


class LocalCodex(Codex):
    """Codex-backed local agent without container bootstrap.

    Harbor's installed Codex agent assumes a disposable container and mutates
    that environment during setup by installing system packages, Node, and the
    Codex CLI. Runme Harbor executes against a local runtime, so this
    wrapper expects `codex` to already be available and skips setup-time
    environment changes.
    """

    @staticmethod
    def name() -> str:
        return "codex"

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
                    "if [ -s ~/.nvm/nvm.sh ]; then . ~/.nvm/nvm.sh; fi\n"
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


class LocalOpenClaw(OpenClaw):
    """OpenClaw-backed local agent without container bootstrap.

    Harbor's installed OpenClaw agent installs Node/OpenClaw in a disposable
    container and writes container-local config before execution. Runme Harbor
    executes against a local runtime, so this wrapper expects `openclaw` to
    already be available and configured.
    """

    CLI_FLAGS = [
        *OpenClaw.CLI_FLAGS,
        CliFlag("session_id", cli="--session-id", type="str"),
        CliFlag("session_key", cli="--session-key", type="str"),
    ]

    @staticmethod
    def name() -> str:
        return "openclaw"

    async def install(self, environment: BaseEnvironment) -> None:
        return None

    async def setup(self, environment: BaseEnvironment) -> None:
        return None

    def _runtime_env(self) -> dict[str, str]:
        if not self.model_name:
            return {}

        if "/" not in self.model_name:
            raise ValueError("Model name must be in the format provider/model_name")

        provider, _ = self.model_name.split("/", 1)
        self._validate_provider(provider)

        env: dict[str, str] = {}
        for key in self._provider_env_keys(provider):
            val = self._get_env(key)
            if val:
                env[key] = val
        return env

    def _collect_session_file(self) -> None:
        envelope = self._parse_stdout()
        if not envelope:
            return

        meta = envelope.get("meta")
        if not isinstance(meta, dict):
            return
        agent_meta = meta.get("agentMeta")
        if not isinstance(agent_meta, dict):
            return
        session_file = agent_meta.get("sessionFile")
        if not isinstance(session_file, str) or not session_file.strip():
            return

        source = Path(session_file).expanduser()
        if not source.is_file():
            return

        target = self.logs_dir / "openclaw.session.jsonl"
        shutil.copy2(source, target)

    def _session_key_arg(self) -> str:
        if self._resolved_flags.get("session_id") or self._resolved_flags.get(
            "session_key"
        ):
            return ""

        digest = sha256(str(self.logs_dir.resolve()).encode()).hexdigest()[:16]
        return f"--session-key runme-harbor-{digest} "

    @with_prompt_template
    async def run(
        self,
        instruction: str,
        environment: BaseEnvironment,
        context: AgentContext,
    ) -> None:
        escaped_instruction = shlex.quote(instruction)
        env = self._runtime_env()

        try:
            instruction_path = self.logs_dir / "instruction.txt"
            instruction_path.write_text(instruction)
        except OSError:
            pass

        cli_flags = self.build_cli_flags()
        cli_flags_arg = f"{cli_flags} " if cli_flags else ""
        session_key_arg = self._session_key_arg()
        model_arg = (
            f"--model {shlex.quote(self.model_name)} " if self.model_name else ""
        )

        try:
            await self.exec_as_agent(
                environment,
                command=(
                    "set -o pipefail\n"
                    "openclaw agent --local --json "
                    f"{cli_flags_arg}"
                    f"{session_key_arg}"
                    f"{model_arg}"
                    f"--message {escaped_instruction} "
                    f"2>&1 </dev/null | tee "
                    f"{EnvironmentPaths.agent_dir / 'openclaw.txt'}"
                ),
                env=env,
            )
        finally:
            try:
                self._collect_session_file()
            except Exception:
                pass

            self.populate_context_post_run(context)
