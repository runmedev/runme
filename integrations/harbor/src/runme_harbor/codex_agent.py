import shlex

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
            )
        finally:
            try:
                await self.exec_as_agent(
                    environment,
                    command=(
                        f"mkdir -p {EnvironmentPaths.agent_dir.as_posix()}\n"
                        'codex_home="${CODEX_HOME:-$HOME/.codex}"\n'
                        'if [ -d "$codex_home/sessions" ]; then\n'
                        f"  rm -rf "
                        f"{(EnvironmentPaths.agent_dir / 'sessions').as_posix()}\n"
                        f'  cp -R "$codex_home/sessions" '
                        f"{(EnvironmentPaths.agent_dir / 'sessions').as_posix()}\n"
                        "fi"
                    ),
                )
            except Exception:
                pass

            self.populate_context_post_run(context)
