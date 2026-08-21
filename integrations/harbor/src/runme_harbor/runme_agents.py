import json
import os
import shlex
import shutil
import tempfile
from hashlib import sha256
from pathlib import Path
from typing import Any, Literal

from harbor.agents.installed.antigravity_cli import AntigravityCli
from harbor.agents.installed.base import CliFlag, with_prompt_template
from harbor.agents.installed.claude_code import ClaudeCode
from harbor.agents.installed.codex import Codex
from harbor.agents.installed.cursor_cli import CursorCli
from harbor.agents.installed.openclaw import OpenClaw
from harbor.environments.base import BaseEnvironment
from harbor.models.agent.context import AgentContext
from harbor.models.trajectories.trajectory import Trajectory
from harbor.models.trial.paths import EnvironmentPaths

_OPENAI_FALLBACK_ENV = "RUNME_ROUTER_FALLBACK_OPENAI_API_KEY"
_ANTHROPIC_FALLBACK_ENV = "RUNME_ROUTER_FALLBACK_ANTHROPIC_API_KEY"
_GOOGLE_FALLBACK_ENV = "RUNME_ROUTER_FALLBACK_GOOGLE_API_KEY"
_CODEX_ROUTER_PROVIDER = "runme_router"

_ProviderAuthSource = Literal[
    "explicit",
    "native",
    "fallback",
    "indeterminate",
    "unconfigured",
]


async def _command_succeeds(
    environment: BaseEnvironment,
    command: str,
    env: dict[str, str] | None = None,
) -> bool | None:
    try:
        result = await environment.exec(command=command, env=env)
    except Exception:
        return None
    return result.return_code == 0


def _toml_string(value: str) -> str:
    return json.dumps(value)


def _config_arg(key: str, value: str) -> str:
    return shlex.quote(f"{key}={_toml_string(value)}")


def _copy_if_present(
    source,
    env: dict[str, str],
    key: str,
) -> None:
    value = source._get_env(key)
    if value:
        env[key] = value


def _parse_bool_kwarg(name: str, value: bool | str) -> bool:
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        normalized = value.strip().lower()
        if normalized == "true":
            return True
        if normalized == "false":
            return False
    raise ValueError(f"{name} must be true or false")


async def _promote_fallback_auth(
    source,
    environment: BaseEnvironment,
    env: dict[str, str],
    provider_key: str,
    fallback_key: str,
    native_auth_probe,
    *,
    native_auth_first: bool = False,
    detect_native_auth: bool = False,
) -> _ProviderAuthSource:
    fallback = source._get_env(fallback_key)
    if source._resolve_env(provider_key) is not None and not (native_auth_first and fallback):
        return "explicit"

    if fallback or detect_native_auth:
        native_auth = await native_auth_probe(environment, env)
        if native_auth is True:
            env.pop(provider_key, None)
            return "native"
        if native_auth is None:
            env.pop(provider_key, None)
            return "indeterminate"

    if source._resolve_env(provider_key) is not None:
        return "explicit"

    if not fallback:
        return "unconfigured"

    env[provider_key] = fallback
    return "fallback"


class RunmeAntigravityCli(AntigravityCli):
    """Antigravity CLI-backed Runme agent without container bootstrap.

    Harbor's installed Antigravity agent assumes a disposable container and
    installs Google's `agy` CLI during setup. Runme Harbor executes through
    Runme's runtime, so this wrapper expects `agy` to already be available and
    configured.
    """

    @staticmethod
    def name() -> str:
        return "runme-antigravity-cli"

    def __init__(self, *args, route_oauth_credentials: bool | str = False, **kwargs):
        self.route_oauth_credentials = _parse_bool_kwarg(
            "route_oauth_credentials", route_oauth_credentials
        )
        super().__init__(*args, **kwargs)

    async def install(self, environment: BaseEnvironment) -> None:
        return None

    async def setup(self, environment: BaseEnvironment) -> None:
        return None

    @staticmethod
    def _make_collect_trajectory_command_portable(command: str) -> str:
        """Replace Harbor's GNU find selector for containerless macOS runs."""
        gnu_newest = "-printf '%T@\\t%p\\n' 2>/dev/null | sort -nr | head -n1 | cut -f2-"
        portable_newest = (
            "-exec sh -c 'latest=; for candidate do "
            'if [ -z "$latest" ] || [ "$candidate" -nt "$latest" ]; then '
            'latest=$candidate; fi; done; printf "%s\\n" "$latest"\' sh {} +'
        )
        if command.count(gnu_newest) != 3:
            raise RuntimeError("Harbor trajectory collector has an unexpected shape")
        return command.replace(gnu_newest, portable_newest)

    @with_prompt_template
    async def run(
        self,
        instruction: str,
        environment: BaseEnvironment,
        context: AgentContext,
    ) -> None:
        escaped_instruction = shlex.quote(instruction)

        if self.model_name and "/" not in self.model_name:
            raise ValueError("Model name must be in the format provider/model_name")

        model = self.model_name.split("/")[-1] if self.model_name else None

        # Gemini CLI refuses to honor `--yolo` in an untrusted workspace and
        # overrides approval mode back to "default".
        env = {"GEMINI_CLI_TRUST_WORKSPACE": "true"}

        auth_vars = [
            "GEMINI_API_KEY",
            "GOOGLE_APPLICATION_CREDENTIALS",
            "GOOGLE_CLOUD_PROJECT",
            "GOOGLE_CLOUD_LOCATION",
            "GOOGLE_GEMINI_BASE_URL",
            "GOOGLE_GENAI_USE_VERTEXAI",
            "GOOGLE_VERTEX_BASE_URL",
            "GOOGLE_API_KEY",
        ]
        for var in auth_vars:
            _copy_if_present(self, env, var)

        has_explicit_key = (
            self._resolve_env("GEMINI_API_KEY") is not None
            or self._resolve_env("GOOGLE_API_KEY") is not None
        )
        fallback = self._get_env(_GOOGLE_FALLBACK_ENV)
        if not has_explicit_key and fallback:
            env["GEMINI_API_KEY"] = fallback
        elif not has_explicit_key and not self.route_oauth_credentials:
            env.pop("GOOGLE_GEMINI_BASE_URL", None)
            env.pop("GOOGLE_VERTEX_BASE_URL", None)

        skills_command = self._build_register_skills_command()
        if skills_command:
            await self.exec_as_agent(environment, command=skills_command, env=env)

        settings_command = self._build_settings_command(model)
        if settings_command:
            await self.exec_as_agent(environment, command=settings_command, env=env)

        cli_flags = self.build_cli_flags()
        extra_flags = (cli_flags + " ") if cli_flags else ""
        # The agy Go CLI selects its model from the --model flag; the ~/.agy
        # settings.json written above is read only by the legacy CLI, so pass
        # an explicitly configured model through to the current CLI.
        model_flag = f"--model {shlex.quote(model)} " if model else ""
        marker_created = False
        try:
            await self.exec_as_agent(
                environment,
                command=f"touch {self._RUN_MARKER}",
            )
            marker_created = True
        except Exception:
            self.logger.debug("run marker creation failed; collecting unscoped")
        try:
            await self.exec_as_agent(
                environment,
                command=(
                    'export PATH="$HOME/.local/bin:$PATH"\n'
                    f"agy --new-project --dangerously-skip-permissions {model_flag}{extra_flags}"
                    f"--prompt={escaped_instruction} "
                    "2>&1 </dev/null | stdbuf -oL tee /logs/agent/antigravity-cli.txt"
                ),
                env=env,
            )
        finally:
            try:
                collect_command = self._make_collect_trajectory_command_portable(
                    super()._build_collect_trajectory_command(scoped=marker_created)
                )
                await self.exec_as_agent(
                    environment,
                    command=collect_command,
                )
            except Exception:
                pass

            self.populate_context_post_run(context)

    def _convert_gemini_to_atif(self, gemini_trajectory: dict[str, Any]) -> Trajectory | None:
        trajectory = super()._convert_gemini_to_atif(gemini_trajectory)
        if trajectory is None:
            return None
        return trajectory.model_copy(
            update={"agent": trajectory.agent.model_copy(update={"name": self.name()})}
        )


class RunmeClaudeCode(ClaudeCode):
    """Claude Code-backed Runme agent without container bootstrap.

    Harbor's installed Claude Code agent assumes a disposable container and
    prepares Claude config, credentials, skills, memory, and MCP servers before
    running. Runme Harbor executes through Runme's runtime, so this wrapper
    expects `claude` to already be available and configured.
    """

    @staticmethod
    def name() -> str:
        return "runme-claude-code"

    def __init__(self, *args, route_oauth_credentials: bool | str = False, **kwargs):
        self.route_oauth_credentials = _parse_bool_kwarg(
            "route_oauth_credentials", route_oauth_credentials
        )
        super().__init__(*args, **kwargs)

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

    def _model_arg(self, route_via_router: bool | None = None) -> str:
        if not self.model_name:
            return ""
        if route_via_router is None:
            route_via_router = bool(self._get_env("ANTHROPIC_BASE_URL"))
        if route_via_router:
            model = self.model_name
        elif self.model_connection.provider is None and "/" in self.model_name:
            model = self.model_name.split("/", 1)[-1]
        else:
            model = self.model_name.split("/")[-1]
        return f"--model {shlex.quote(model)} "

    async def _has_native_auth(
        self,
        environment: BaseEnvironment,
        env: dict[str, str],
    ) -> bool | None:
        return await _command_succeeds(
            environment,
            "unset ANTHROPIC_API_KEY ANTHROPIC_BASE_URL; "
            'export PATH="$HOME/.local/bin:$PATH"; '
            "claude auth status",
            env=env,
        )

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

    async def _agent_env(self, environment: BaseEnvironment) -> dict[str, str]:
        env, _ = await self._agent_env_and_auth_source(environment)
        return env

    async def _agent_env_and_auth_source(
        self, environment: BaseEnvironment
    ) -> tuple[dict[str, str], _ProviderAuthSource]:
        env = self._runtime_env()
        _copy_if_present(self, env, "ANTHROPIC_BASE_URL")
        auth_source = await _promote_fallback_auth(
            self,
            environment,
            env,
            "ANTHROPIC_API_KEY",
            _ANTHROPIC_FALLBACK_ENV,
            self._has_native_auth,
            native_auth_first=True,
            detect_native_auth=bool(self._get_env("ANTHROPIC_BASE_URL")),
        )
        if auth_source in {"native", "indeterminate"} and not self.route_oauth_credentials:
            env.pop("ANTHROPIC_BASE_URL", None)
        return env, auth_source

    @with_prompt_template
    async def run(
        self,
        instruction: str,
        environment: BaseEnvironment,
        context: AgentContext,
    ) -> None:
        escaped_instruction = shlex.quote(instruction)
        cli_flags = self.build_cli_flags()
        cli_flags_arg = f"{cli_flags} " if cli_flags else ""
        session_files_before = self._snapshot_session_files()
        env, auth_source = await self._agent_env_and_auth_source(environment)
        native_auth_arg = (
            "unset ANTHROPIC_API_KEY\n" if auth_source in {"native", "indeterminate"} else ""
        )
        model_arg = self._model_arg(bool(env.get("ANTHROPIC_BASE_URL")))

        try:
            await self.exec_as_agent(
                environment,
                command=(
                    "set -o pipefail\n"
                    f"{native_auth_arg}"
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


class RunmeCodex(Codex):
    """Codex-backed Runme agent without container bootstrap.

    Harbor's installed Codex agent assumes a disposable container and mutates
    that environment during setup by installing system packages, Node, and the
    Codex CLI. Runme Harbor executes through Runme's runtime, so this
    wrapper expects `codex` to already be available and skips setup-time
    environment changes.
    """

    @staticmethod
    def name() -> str:
        return "runme-codex"

    def __init__(self, *args, route_oauth_credentials: bool | str = False, **kwargs):
        self.route_oauth_credentials = _parse_bool_kwarg(
            "route_oauth_credentials", route_oauth_credentials
        )
        super().__init__(*args, **kwargs)

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

    def _codex_thread_id_from_output(self) -> str | None:
        output_path = self.logs_dir / self._OUTPUT_FILENAME
        if not output_path.exists():
            return None

        with open(output_path) as handle:
            for line in handle:
                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    continue

                if event.get("type") != "thread.started":
                    continue

                thread_id = event.get("thread_id")
                if isinstance(thread_id, str) and thread_id:
                    return thread_id

        return None

    @staticmethod
    def _codex_session_ids_from_file(session_file: Path) -> set[str]:
        with open(session_file) as handle:
            for line in handle:
                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    continue

                if event.get("type") != "session_meta":
                    continue

                payload = event.get("payload")
                if not isinstance(payload, dict):
                    return set()

                session_ids: set[str] = set()
                for key in ("id", "session_id"):
                    session_id = payload.get(key)
                    if isinstance(session_id, str) and session_id:
                        session_ids.add(session_id)
                return session_ids

        return set()

    def _select_new_session(self, new_sessions: set[Path]) -> Path | None:
        if not new_sessions:
            return None

        thread_id = self._codex_thread_id_from_output()
        if thread_id:
            matches = [
                session_file
                for session_file in new_sessions
                if thread_id in self._codex_session_ids_from_file(session_file)
            ]
            if len(matches) == 1:
                return matches[0]
            self.logger.warning(
                "Could not uniquely match Codex thread %s to new session files: %s",
                thread_id,
                sorted(str(path) for path in new_sessions),
            )
            return None

        if len(new_sessions) == 1:
            return next(iter(new_sessions))

        self.logger.warning(
            "Could not identify Codex session file; multiple new sessions found: %s",
            sorted(str(path) for path in new_sessions),
        )
        return None

    def _collect_new_sessions(self, before: set[Path]) -> bool:
        sessions_dir = self._codex_sessions_dir()
        if not sessions_dir.exists():
            return False

        after = {path for path in sessions_dir.rglob("*.jsonl") if path.is_file()}
        new_sessions = after - before

        target_dir = self.logs_dir / "sessions"
        if target_dir.exists():
            shutil.rmtree(target_dir)

        session_file = self._select_new_session(new_sessions)
        if session_file is None:
            return False

        target = target_dir / session_file.relative_to(sessions_dir)
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(session_file, target)
        return True

    async def _has_native_auth(
        self,
        environment: BaseEnvironment,
        env: dict[str, str],
    ) -> bool | None:
        return await _command_succeeds(
            environment,
            "unset OPENAI_API_KEY OPENAI_BASE_URL; "
            "if [ -s ~/.nvm/nvm.sh ]; then . ~/.nvm/nvm.sh; fi; "
            "codex login status",
            env=env,
        )

    async def _agent_env(self, environment: BaseEnvironment) -> dict[str, str]:
        env, _ = await self._agent_env_and_router_state(environment)
        return env

    async def _agent_env_and_router_state(
        self,
        environment: BaseEnvironment,
    ) -> tuple[dict[str, str], _ProviderAuthSource]:
        env: dict[str, str] = {}
        auth_source = await _promote_fallback_auth(
            self,
            environment,
            env,
            "OPENAI_API_KEY",
            _OPENAI_FALLBACK_ENV,
            self._has_native_auth,
            native_auth_first=True,
            detect_native_auth=bool(self._get_env("OPENAI_BASE_URL")),
        )
        return env, auth_source

    def _router_provider_args(self) -> str:
        base_url = self._get_env("OPENAI_BASE_URL")
        if not base_url:
            return ""
        provider = f"model_providers.{_CODEX_ROUTER_PROVIDER}"
        settings = (
            ("model_provider", _CODEX_ROUTER_PROVIDER),
            (f"{provider}.name", "Runme Router"),
            (f"{provider}.base_url", base_url),
            (f"{provider}.env_key", "OPENAI_API_KEY"),
            (f"{provider}.wire_api", "responses"),
        )
        args = "".join(f"-c {_config_arg(key, value)} " for key, value in settings)
        return f"{args}-c {provider}.supports_websockets=true "

    def _native_router_args(self) -> str:
        base_url = self._get_env("OPENAI_BASE_URL")
        if not base_url:
            return ""
        return f"-c {_config_arg('openai_base_url', base_url)} "

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
        env, auth_source = await self._agent_env_and_router_state(environment)
        native_auth = auth_source in {"native", "indeterminate"}
        provider_args = (
            self._native_router_args()
            if native_auth and self.route_oauth_credentials
            else self._router_provider_args()
            if not native_auth
            else ""
        )
        native_auth_arg = "unset OPENAI_API_KEY OPENAI_BASE_URL\n" if native_auth else ""

        try:
            await self.exec_as_agent(
                environment,
                command=(
                    "set -o pipefail\n"
                    f"{native_auth_arg}"
                    "if [ -s ~/.nvm/nvm.sh ]; then . ~/.nvm/nvm.sh; fi\n"
                    "codex exec "
                    "--dangerously-bypass-approvals-and-sandbox "
                    "--skip-git-repo-check "
                    f"{provider_args}"
                    f"{model_arg}"
                    "--json "
                    "--enable unified_exec "
                    f"{cli_flags_arg}"
                    "-- "
                    f"{escaped_instruction} "
                    f"2>&1 </dev/null | tee "
                    f"{EnvironmentPaths.agent_dir / self._OUTPUT_FILENAME}"
                ),
                env=env,
            )
        finally:
            collected_session = False
            try:
                collected_session = self._collect_new_sessions(session_files_before)
            except Exception:
                self.logger.exception("Failed to collect Codex session file")

            if collected_session:
                self.populate_context_post_run(context)


class RunmeCursorCli(CursorCli):
    """Cursor CLI-backed Runme agent without container bootstrap.

    Harbor's installed Cursor agent assumes a disposable container and requires
    API-key setup before running. Runme Harbor executes through Runme's runtime,
    so this wrapper expects `cursor-agent` to already be available and uses the
    user's ambient Cursor authentication when present.
    """

    @staticmethod
    def name() -> str:
        return "runme-cursor-cli"

    async def install(self, environment: BaseEnvironment) -> None:
        return None

    async def setup(self, environment: BaseEnvironment) -> None:
        return None

    def _runtime_env(self) -> dict[str, str]:
        env: dict[str, str] = {}
        if cursor_api_key := os.environ.get("CURSOR_API_KEY"):
            env["CURSOR_API_KEY"] = cursor_api_key
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

        if self.model_name and "/" not in self.model_name:
            raise ValueError("Model name must be in the format provider/model_name")

        model_arg = (
            f"--model={shlex.quote(self.model_name.split('/')[-1])} " if self.model_name else ""
        )
        env = self._runtime_env()

        mcp_command = self._build_register_mcp_servers_command()
        if mcp_command:
            await self.exec_as_agent(environment, command=mcp_command, env=env)

        cli_flags = self.build_cli_flags()
        cli_flags_arg = f"{cli_flags} " if cli_flags else ""

        await self.exec_as_agent(
            environment,
            command=(
                "set -o pipefail\n"
                'export PATH="$HOME/.local/bin:$PATH"\n'
                "cursor-agent --yolo --print --output-format=stream-json "
                f"{cli_flags_arg}"
                f"{model_arg}"
                f"-- {escaped_instruction} "
                f"2>&1 </dev/null | tee "
                f"{EnvironmentPaths.agent_dir / self._OUTPUT_FILENAME}"
            ),
            env=env,
        )

        self.populate_context_post_run(context)


class RunmeOpenClaw(OpenClaw):
    """OpenClaw-backed Runme agent without container bootstrap.

    Harbor's installed OpenClaw agent installs Node/OpenClaw in a disposable
    container and writes container-local config before execution. Runme Harbor
    executes through Runme's runtime, so this wrapper expects `openclaw` to
    already be available and configured.
    """

    CLI_FLAGS = [
        *OpenClaw.CLI_FLAGS,
        CliFlag("session_id", cli="--session-id", type="str"),
        CliFlag("session_key", cli="--session-key", type="str"),
    ]

    @staticmethod
    def name() -> str:
        return "runme-openclaw"

    async def install(self, environment: BaseEnvironment) -> None:
        return None

    async def setup(self, environment: BaseEnvironment) -> None:
        return None

    @staticmethod
    def _openclaw_config_path() -> Path:
        config_path = os.environ.get("OPENCLAW_CONFIG_PATH")
        if config_path:
            return Path(config_path).expanduser()
        return Path.home() / ".openclaw" / "openclaw.json"

    @staticmethod
    def _workspace_path(environment: BaseEnvironment) -> str | None:
        task_env_config = getattr(environment, "task_env_config", None)
        workdir = getattr(task_env_config, "workdir", None) or "/app"
        map_remote_path = getattr(environment, "_map_remote_path", None)
        if not callable(map_remote_path):
            return None
        return str(map_remote_path(workdir))

    def _create_runtime_config(self, environment: BaseEnvironment) -> Path | None:
        source = self._openclaw_config_path()
        if not source.is_file():
            return None

        workspace_path = self._workspace_path(environment)
        if not workspace_path:
            return None

        try:
            config = json.loads(source.read_text())
        except json.JSONDecodeError as exc:
            raise ValueError(f"OpenClaw config is not valid JSON: {source}") from exc

        agents = config.setdefault("agents", {})
        defaults = agents.setdefault("defaults", {})
        defaults["workspace"] = workspace_path
        defaults["skipBootstrap"] = True

        config_dir = Path(tempfile.mkdtemp(prefix="runme-harbor-openclaw-"))
        target = config_dir / "openclaw.json"
        target.write_text(json.dumps(config, indent=2) + "\n")
        target.chmod(0o600)
        return target

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
        if self._resolved_flags.get("session_id") or self._resolved_flags.get("session_key"):
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
        runtime_config_path = self._create_runtime_config(environment)
        if runtime_config_path:
            env = dict(env)
            env["OPENCLAW_CONFIG_PATH"] = str(runtime_config_path)

        try:
            instruction_path = self.logs_dir / "instruction.txt"
            instruction_path.write_text(instruction)
        except OSError:
            pass

        cli_flags = self.build_cli_flags()
        cli_flags_arg = f"{cli_flags} " if cli_flags else ""
        session_key_arg = self._session_key_arg()
        model_arg = f"--model {shlex.quote(self.model_name)} " if self.model_name else ""

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
            if runtime_config_path:
                shutil.rmtree(runtime_config_path.parent, ignore_errors=True)

            self.populate_context_post_run(context)
