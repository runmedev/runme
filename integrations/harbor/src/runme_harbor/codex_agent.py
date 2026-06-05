import shlex
from pathlib import Path, PurePosixPath

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

    _REMOTE_CODEX_HOME = PurePosixPath("/tmp/runme-codex-home")
    _REMOTE_CODEX_SECRETS_DIR = PurePosixPath("/tmp/runme-codex-secrets")

    @staticmethod
    def name() -> str:
        return "runme-codex"

    async def install(self, environment: BaseEnvironment) -> None:
        return None

    async def setup(self, environment: BaseEnvironment) -> None:
        return None

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

        auth_json_path = self._resolve_preferred_auth_json_path()
        remote_codex_home = self._REMOTE_CODEX_HOME.as_posix()
        remote_secrets_dir = self._REMOTE_CODEX_SECRETS_DIR.as_posix()
        remote_auth_path = (self._REMOTE_CODEX_SECRETS_DIR / "auth.json").as_posix()

        env: dict[str, str] = {
            "CODEX_HOME": remote_codex_home,
        }

        await self.exec_as_agent(
            environment,
            command=(
                f'mkdir -p "$CODEX_HOME" {shlex.quote(remote_secrets_dir)} '
                f"{shlex.quote(EnvironmentPaths.agent_dir.as_posix())}"
            ),
            env=env,
        )

        if auth_json_path:
            self.logger.debug("Codex auth: using auth.json from %s", auth_json_path)
            await environment.upload_file(auth_json_path, remote_auth_path)
            setup_command = f'ln -sf {shlex.quote(remote_auth_path)} "$CODEX_HOME/auth.json"\n'
        elif openai_api_key := self._get_env("OPENAI_API_KEY"):
            self.logger.debug("Codex auth: using OPENAI_API_KEY")
            env["OPENAI_API_KEY"] = openai_api_key
            setup_command = (
                f"cat >{shlex.quote(remote_auth_path)} <<EOF\n"
                '{\n  "OPENAI_API_KEY": "${OPENAI_API_KEY}"\n}\nEOF\n'
                f"ln -sf {shlex.quote(remote_auth_path)} "
                '"$CODEX_HOME/auth.json"\n'
            )
        elif (local_auth_json_path := Path.home() / ".codex" / "auth.json").is_file():
            self.logger.debug(
                "Codex auth: using local auth.json from %s",
                local_auth_json_path,
            )
            await environment.upload_file(local_auth_json_path, remote_auth_path)
            setup_command = f'ln -sf {shlex.quote(remote_auth_path)} "$CODEX_HOME/auth.json"\n'
        else:
            raise ValueError(
                "Codex auth not found. Run `codex login`, set OPENAI_API_KEY, "
                "or set CODEX_AUTH_JSON_PATH."
            )

        if openai_base_url := self._get_env("OPENAI_BASE_URL"):
            env["OPENAI_BASE_URL"] = openai_base_url
            setup_command += (
                '\ncat >>"$CODEX_HOME/config.toml" <<TOML\n'
                'openai_base_url = "${OPENAI_BASE_URL}"\n'
                "TOML"
            )

        skills_command = self._build_register_skills_command()
        if skills_command:
            setup_command += f"\n{skills_command}"

        mcp_command = self._build_register_mcp_servers_command()
        if mcp_command:
            setup_command += f"\n{mcp_command}"

        if setup_command.strip():
            await self.exec_as_agent(environment, command=setup_command, env=env)

        try:
            await self.exec_as_agent(
                environment,
                command=(
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
                env=env,
            )
        finally:
            try:
                await self.exec_as_agent(
                    environment,
                    command=(
                        f"mkdir -p {EnvironmentPaths.agent_dir.as_posix()}\n"
                        'if [ -d "$CODEX_HOME/sessions" ]; then\n'
                        f"  rm -rf "
                        f"{(EnvironmentPaths.agent_dir / 'sessions').as_posix()}\n"
                        f'  cp -R "$CODEX_HOME/sessions" '
                        f"{(EnvironmentPaths.agent_dir / 'sessions').as_posix()}\n"
                        "fi"
                    ),
                    env=env,
                )
            except Exception:
                pass

            try:
                await self.exec_as_agent(
                    environment,
                    command=f'rm -rf {shlex.quote(remote_secrets_dir)} "$CODEX_HOME"',
                    env=env,
                )
            except Exception:
                pass

            self.populate_context_post_run(context)

    def _resolve_preferred_auth_json_path(self) -> Path | None:
        if auth_json_path := self._resolve_auth_json_path():
            return auth_json_path
        local_auth_json_path = Path.home() / ".codex" / "auth.json"
        if local_auth_json_path.is_file():
            return local_auth_json_path
        return None
